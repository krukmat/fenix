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
	ToolErrorInvalidInput     ExecutionErrorCode = "invalid_input"
	ToolErrorPermissionDenied ExecutionErrorCode = "permission_denied"
	ToolErrorToolInactive     ExecutionErrorCode = "tool_inactive"
	ToolErrorInternal         ExecutionErrorCode = "internal_error"
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
		return nil, r.handleExecutionError(ctx, workspaceID, def.Name, params, ToolErrorInternal, err, startedAt)
	}

	out, err := executor.Execute(ctx, params)
	if err != nil {
		return nil, r.handleExecutionError(ctx, workspaceID, def.Name, params, ToolErrorInternal, err, startedAt)
	}

	r.auditToolExecution(ctx, workspaceID, def.Name, params, audit.OutcomeSuccess, "")
	r.recordToolUsage(ctx, workspaceID, def.Name, startedAt)
	return out, nil
}

func (r *ToolRegistry) ensureExecutable(
	ctx context.Context,
	workspaceID string,
	def *ToolDefinition,
	params json.RawMessage,
	startedAt time.Time,
) error {
	if !def.IsActive {
		return r.handleExecutionError(ctx, workspaceID, def.Name, params, ToolErrorToolInactive, ErrToolInactive, startedAt)
	}
	if err := r.ValidateParams(ctx, workspaceID, def.Name, params); err != nil {
		return r.handleExecutionError(ctx, workspaceID, def.Name, params, ToolErrorInvalidInput, err, startedAt)
	}
	if err := r.enforceToolPermission(ctx, def.Name); err != nil {
		return r.handleExecutionError(ctx, workspaceID, def.Name, params, ToolErrorPermissionDenied, err, startedAt)
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
		return err
	}
	if !allowed {
		return ErrToolPermissionDenied
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
) error {
	wrapped := &ExecutionError{ToolName: toolName, Code: code, Err: err}
	r.auditToolExecution(ctx, workspaceID, toolName, params, resolveAuditOutcome(code), string(code))
	r.recordToolUsage(ctx, workspaceID, toolName, startedAt)
	return wrapped
}

func (r *ToolRegistry) auditToolExecution(
	ctx context.Context,
	workspaceID, toolName string,
	params json.RawMessage,
	outcome audit.Outcome,
	errorCode string,
) {
	if r.audit == nil || !isBuiltinTool(toolName) {
		return
	}

	action := "tool.executed"
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
		&audit.EventDetails{Metadata: buildToolAuditMetadata(toolName, params, errorCode)},
		outcome,
	)
}

func resolveAuditOutcome(code ExecutionErrorCode) audit.Outcome {
	if code == ToolErrorPermissionDenied {
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

func buildToolAuditMetadata(toolName string, params json.RawMessage, errorCode string) map[string]any {
	meta := map[string]any{
		"tool_name":  toolName,
		"param_keys": extractParamKeys(params),
	}
	if errorCode != "" {
		meta["error_code"] = errorCode
	}
	return meta
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

func isBuiltinTool(toolName string) bool {
	switch toolName {
	case BuiltinCreateTask, BuiltinUpdateCase, BuiltinSendReply,
		BuiltinGetLead, BuiltinGetAccount, BuiltinCreateKnowledgeItem,
		BuiltinUpdateKnowledgeItem, BuiltinQueryMetrics:
		return true
	default:
		return false
	}
}

func IsToolExecutionErrorCode(err error, code ExecutionErrorCode) bool {
	var execErr *ExecutionError
	return errors.As(err, &execErr) && execErr.Code == code
}
