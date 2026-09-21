package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/usage"
)

const auditActionToolExecuted = "tool.executed"

type AuditLogger interface {
	LogWithDetails(
		ctx context.Context,
		workspaceID string,
		actorID string,
		actorType audit.ActorType,
		action string,
		entityType *string,
		entityID *string,
		details *audit.EventDetails,
		outcome audit.Outcome,
	) error
}

type ExecutionErrorCode string

const (
	ToolErrorInvalidInput        ExecutionErrorCode = "invalid_input"
	ToolErrorPermissionDenied    ExecutionErrorCode = "permission_denied"
	ToolErrorToolInactive        ExecutionErrorCode = "tool_inactive"
	ToolErrorCapabilityContext       ExecutionErrorCode = "capability_context_missing"
	ToolErrorGovernanceDenied        ExecutionErrorCode = "governance_denied"
	ToolErrorCapabilityFailed        ExecutionErrorCode = "capability_failed"
	ToolErrorCapabilityIndeterminate ExecutionErrorCode = "capability_indeterminate"
	ToolErrorInternal                ExecutionErrorCode = "internal_error"
)

type ExecutionError struct {
	ToolName string
	Code     ExecutionErrorCode
	Err      error
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("tool %s %s: %v", e.ToolName, e.Code, e.Err)
}

func (e *ExecutionError) Unwrap() error {
	return e.Err
}

func (r *ToolRegistry) Execute(ctx context.Context, workspaceID, toolName string, params json.RawMessage) (json.RawMessage, error) {
	def, err := r.getDefinitionForExecution(ctx, workspaceID, toolName)
	if err != nil {
		return nil, err
	}
	if _, ok := r.capability(toolName); ok {
		ctx = prepareCapabilityExecutionContext(ctx, workspaceID)
	}
	return r.executeDefinition(ctx, workspaceID, def, normalizeToolParams(params))
}

func (r *ToolRegistry) executeDefinition(
	ctx context.Context,
	workspaceID string,
	def *ToolDefinition,
	params json.RawMessage,
) (json.RawMessage, error) {
	startedAt := time.Now()
	if err := r.ensureExecutable(ctx, workspaceID, def, params, startedAt); err != nil {
		return nil, err
	}

	executor, err := r.Get(def.Name)
	if err != nil {
		return nil, r.handleExecutionError(ctx, workspaceID, def.Name, params, ToolErrorInternal, err, startedAt, nil)
	}
	if descriptor, ok := r.capability(def.Name); ok {
		return r.executeCapability(ctx, workspaceID, descriptor, executor, params, startedAt)
	}

	out, err := executor.Execute(ctx, params)
	if err != nil {
		return nil, r.handleExecutionError(ctx, workspaceID, def.Name, params, ToolErrorInternal, err, startedAt, nil)
	}

	r.auditToolExecution(ctx, workspaceID, def.Name, params, audit.OutcomeSuccess, "", nil)
	r.recordToolUsage(ctx, workspaceID, def.Name, startedAt)
	return out, nil
}

func (r *ToolRegistry) executeCapability(
	ctx context.Context,
	workspaceID string,
	descriptor CapabilityDescriptor,
	executor ToolExecutor,
	params json.RawMessage,
	startedAt time.Time,
) (json.RawMessage, error) {
	attempts := descriptor.maxAttempts()
	for attempt := 1; attempt <= attempts; attempt++ {
		out, err := executor.Execute(ctx, params)
		if err == nil {
			r.auditCapabilitySuccess(ctx, workspaceID, descriptor, params, startedAt, attempt)
			return out, nil
		}

		decision := decideCapabilityFailure(descriptor, err, attempt, attempts)
		if decision.retry {
			continue
		}
		return nil, r.handleCapabilityError(
			ctx,
			workspaceID,
			descriptor,
			params,
			decision.code,
			err,
			startedAt,
			attempt,
			decision.status,
		)
	}
	return nil, r.handleCapabilityError(
		ctx,
		workspaceID,
		descriptor,
		params,
		ToolErrorCapabilityFailed,
		errors.New("capability retry budget exhausted"),
		startedAt,
		attempts,
		CapabilityStatusFailed,
	)
}

type capabilityFailureDecision struct {
	retry  bool
	code   ExecutionErrorCode
	status CapabilityExecutionStatus
}

func decideCapabilityFailure(
	descriptor CapabilityDescriptor,
	err error,
	attempt, maxAttempts int,
) capabilityFailureDecision {
	switch capabilityRetryDisposition(err) {
	case CapabilityRetryIndeterminate:
		return capabilityFailureDecision{
			code:   ToolErrorCapabilityIndeterminate,
			status: CapabilityStatusIndeterminate,
		}
	case CapabilityRetryable:
		if attempt < maxAttempts && descriptor.canRetryAutomatically() {
			return capabilityFailureDecision{retry: true}
		}
		return capabilityFailureDecision{
			code:   ToolErrorCapabilityFailed,
			status: CapabilityStatusFailed,
		}
	default:
		if !descriptor.canRetryAutomatically() {
			return capabilityFailureDecision{
				code:   ToolErrorCapabilityIndeterminate,
				status: CapabilityStatusIndeterminate,
			}
		}
		return capabilityFailureDecision{
			code:   ToolErrorCapabilityFailed,
			status: CapabilityStatusFailed,
		}
	}
}

func (r *ToolRegistry) auditCapabilitySuccess(
	ctx context.Context,
	workspaceID string,
	descriptor CapabilityDescriptor,
	params json.RawMessage,
	startedAt time.Time,
	attempt int,
) {
	r.auditToolExecution(ctx, workspaceID, descriptor.Name, params, audit.OutcomeSuccess, "", map[string]any{
		"capability_execution_status": string(CapabilityStatusSucceeded),
		"attempt_count":              attempt,
	})
	r.recordToolUsage(ctx, workspaceID, descriptor.Name, startedAt)
}

func (r *ToolRegistry) handleCapabilityError(
	ctx context.Context,
	workspaceID string,
	descriptor CapabilityDescriptor,
	params json.RawMessage,
	code ExecutionErrorCode,
	err error,
	startedAt time.Time,
	attempt int,
	status CapabilityExecutionStatus,
) error {
	return r.handleExecutionError(ctx, workspaceID, descriptor.Name, params, code, err, startedAt, map[string]any{
		"capability_execution_status": string(status),
		"attempt_count":              attempt,
	})
}

func (r *ToolRegistry) ensureExecutable(
	ctx context.Context,
	workspaceID string,
	def *ToolDefinition,
	params json.RawMessage,
	startedAt time.Time,
) error {
	if !def.IsActive {
		return r.handleExecutionError(ctx, workspaceID, def.Name, params, ToolErrorToolInactive, ErrToolInactive, startedAt, nil)
	}
	if err := r.ValidateParams(ctx, workspaceID, def.Name, params); err != nil {
		return r.handleExecutionError(ctx, workspaceID, def.Name, params, ToolErrorInvalidInput, err, startedAt, nil)
	}
	if err := r.enforceToolPermission(ctx, def.Name); err != nil {
		return r.handleExecutionError(ctx, workspaceID, def.Name, params, ToolErrorPermissionDenied, err, startedAt, nil)
	}
	if err := r.enforceCapabilityBoundary(ctx, def.Name); err != nil {
		code := ToolErrorGovernanceDenied
		if errors.Is(err, ErrCapabilityContextMissing) {
			code = ToolErrorCapabilityContext
		}
		return r.handleExecutionError(ctx, workspaceID, def.Name, params, code, err, startedAt, nil)
	}
	return nil
}

func (r *ToolRegistry) enforceToolPermission(ctx context.Context, toolName string) error {
	if r.authz == nil {
		return nil
	}

	userID, ok := ctx.Value(ctxkeys.UserID).(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return ErrToolUserContextMissing
	}

	allowed, err := r.authz.CheckToolPermission(ctx, userID, toolName)
	if err != nil {
		return fmt.Errorf("check tool permission: %w", err)
	}
	if !allowed {
		return ErrToolPermissionDenied
	}
	return nil
}

func (r *ToolRegistry) enforceCapabilityBoundary(ctx context.Context, toolName string) error {
	descriptor, ok := r.capability(toolName)
	if !ok {
		return nil
	}
	if err := validateCapabilityExecutionContext(ctx); err != nil {
		return err
	}
	return r.enforceCapabilityGovernance(ctx, descriptor)
}

func validateCapabilityExecutionContext(ctx context.Context) error {
	if contextValue(ctx, ctxkeys.TraceID) == "" || contextValue(ctx, ctxkeys.ExecutionID) == "" {
		return ErrCapabilityContextMissing
	}
	return nil
}

func (r *ToolRegistry) enforceCapabilityGovernance(ctx context.Context, descriptor CapabilityDescriptor) error {
	if !requiresCapabilityGovernor(descriptor.SideEffectClass) {
		return nil
	}
	if r.governor == nil {
		return ErrCapabilityGovernanceRequired
	}
	if err := r.governor.CheckCapabilityExecution(ctx, descriptor); err != nil {
		return fmt.Errorf("check capability governance: %w", err)
	}
	return nil
}

func (r *ToolRegistry) handleExecutionError(
	ctx context.Context,
	workspaceID, toolName string,
	params json.RawMessage,
	code ExecutionErrorCode,
	err error,
	startedAt time.Time,
	extra map[string]any,
) error {
	wrapped := &ExecutionError{ToolName: toolName, Code: code, Err: err}
	r.auditToolExecution(ctx, workspaceID, toolName, params, resolveAuditOutcome(code), string(code), extra)
	r.recordToolUsage(ctx, workspaceID, toolName, startedAt)
	return wrapped
}

func (r *ToolRegistry) auditToolExecution(
	ctx context.Context,
	workspaceID, toolName string,
	params json.RawMessage,
	outcome audit.Outcome,
	errorCode string,
	extra map[string]any,
) {
	if r.audit == nil {
		return
	}

	action := auditActionToolExecuted
	if outcome == audit.OutcomeDenied {
		action = "tool.denied"
	}

	actorID, actorType := auditActorFromContext(ctx)
	entityType := "tool"
	entityID := toolName
	_ = r.audit.LogWithDetails(
		ctx,
		workspaceID,
		actorID,
		actorType,
		action,
		&entityType,
		&entityID,
		&audit.EventDetails{Metadata: r.buildToolAuditMetadata(ctx, toolName, params, errorCode, extra)},
		outcome,
	)
}

func resolveAuditOutcome(code ExecutionErrorCode) audit.Outcome {
	if code == ToolErrorPermissionDenied || code == ToolErrorGovernanceDenied {
		return audit.OutcomeDenied
	}
	return audit.OutcomeError
}

func auditActorFromContext(ctx context.Context) (string, audit.ActorType) {
	if userID, ok := ctx.Value(ctxkeys.UserID).(string); ok && strings.TrimSpace(userID) != "" {
		return userID, audit.ActorTypeUser
	}
	return "system", audit.ActorTypeSystem
}

func (r *ToolRegistry) buildToolAuditMetadata(ctx context.Context, toolName string, params json.RawMessage, errorCode string, extra map[string]any) map[string]any {
	meta := map[string]any{
		"tool_name":  toolName,
		"param_keys": extractParamKeys(params),
	}
	addExecutionAuditMetadata(meta, ctx)
	r.addCapabilityAuditMetadata(meta, toolName)
	if errorCode != "" {
		meta["error_code"] = errorCode
	}
	for key, value := range extra {
		meta[key] = value
	}
	return meta
}

func addExecutionAuditMetadata(meta map[string]any, ctx context.Context) {
	addContextMetadata(meta, "execution_id", ctx, ctxkeys.ExecutionID)
	addContextMetadata(meta, "run_id", ctx, ctxkeys.RunID)
	addContextMetadata(meta, "approval_id", ctx, ctxkeys.ApprovalID)
}

func addContextMetadata(meta map[string]any, field string, ctx context.Context, key ctxkeys.Key) {
	if value := contextValue(ctx, key); value != "" {
		meta[field] = value
	}
}

func (r *ToolRegistry) addCapabilityAuditMetadata(meta map[string]any, toolName string) {
	descriptor, ok := r.capability(toolName)
	if !ok {
		return
	}
	meta["capability_name"] = descriptor.Name
	meta["capability_version"] = descriptor.Version
	meta["capability_operation"] = descriptor.Operation
	meta["side_effect_class"] = string(descriptor.SideEffectClass)
	meta["approval_required"] = descriptor.RequiresApproval()
	meta["max_attempts"] = descriptor.maxAttempts()
	if descriptor.IdempotencyMode != CapabilityIdempotencyNone {
		meta["idempotency_mode"] = string(descriptor.IdempotencyMode)
	}
}

func (r *ToolRegistry) recordToolUsage(ctx context.Context, workspaceID, toolName string, startedAt time.Time) {
	if r.usage == nil {
		return
	}

	actorID, actorType := auditActorFromContext(ctx)
	runID := runIDFromContext(ctx)
	toolNameCopy := toolName
	latencyMs := time.Since(startedAt).Milliseconds()
	_, _ = r.usage.RecordEvent(ctx, usage.RecordEventInput{
		WorkspaceID: workspaceID,
		ActorID:     actorID,
		ActorType:   string(actorType),
		RunID:       runID,
		ToolName:    &toolNameCopy,
		LatencyMs:   &latencyMs,
	})
}

func runIDFromContext(ctx context.Context) *string {
	runID, ok := ctx.Value(ctxkeys.RunID).(string)
	if !ok || strings.TrimSpace(runID) == "" {
		return nil
	}
	return &runID
}

func extractParamKeys(params json.RawMessage) []string {
	var payload map[string]any
	if len(params) == 0 || json.Unmarshal(params, &payload) != nil {
		return nil
	}

	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	return keys
}

func normalizeToolParams(params json.RawMessage) json.RawMessage {
	if len(params) == 0 {
		return json.RawMessage(`{}`)
	}
	return params
}

func IsToolExecutionErrorCode(err error, code ExecutionErrorCode) bool {
	var execErr *ExecutionError
	return errors.As(err, &execErr) && execErr.Code == code
}
