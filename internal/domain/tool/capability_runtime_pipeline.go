package tool

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
)

const evidenceStateNotPlanned = "not_planned"
const evidenceStateIndeterminate = "indeterminate"

func (r *ToolRegistry) executeGovernedCapability(
	ctx context.Context,
	workspaceID string,
	descriptor CapabilityDescriptor,
	executor ToolExecutor,
	params json.RawMessage,
	startedAt time.Time,
) (json.RawMessage, error) {
	decision, err := r.resolveRuntimeGovernance(ctx, descriptor)
	if err != nil {
		code := ToolErrorGovernanceDenied
		if errors.Is(err, ErrCapabilityContextMissing) {
			code = ToolErrorCapabilityContext
		}
		return nil, r.handleExecutionError(
			ctx, workspaceID, descriptor.Name, params,
			code, err, startedAt, nil,
		)
	}
	if !decision.Allowed {
		return r.handleGovernanceDenial(ctx, workspaceID, descriptor, params, startedAt, decision)
	}
	return r.executeGovernedProvider(ctx, workspaceID, descriptor, executor, params, startedAt, decision)
}

func (r *ToolRegistry) resolveRuntimeGovernance(
	ctx context.Context,
	descriptor CapabilityDescriptor,
) (CapabilityRuntimeDecision, error) {
	if err := validateCapabilityExecutionContext(ctx); err != nil {
		return CapabilityRuntimeDecision{}, err
	}
	facts := r.collectGovernanceFacts(ctx, descriptor)
	decision := defaultRuntimeDecision(descriptor, facts)
	var err error
	if r.planner != nil {
		decision, err = r.planner.PlanCapability(ctx, descriptor, facts)
		if err != nil {
			return CapabilityRuntimeDecision{}, err
		}
	}
	if err := validateRuntimeDecision(decision, facts); err != nil {
		return CapabilityRuntimeDecision{}, err
	}
	return decision, nil
}

func (r *ToolRegistry) collectGovernanceFacts(
	ctx context.Context,
	descriptor CapabilityDescriptor,
) CapabilityGovernanceFacts {
	facts := CapabilityGovernanceFacts{
		GovernorRequired: requiresCapabilityGovernor(descriptor.SideEffectClass) ||
			descriptor.RequiresApproval(),
		ApprovalRequired: descriptor.RequiresApproval(),
	}
	if !facts.GovernorRequired {
		facts.GovernorPassed = true
		facts.ApprovalGranted = true
		return facts
	}
	if r.governor == nil {
		facts.GovernanceError = "governor_unavailable"
		return facts
	}
	if err := r.governor.CheckCapabilityExecution(ctx, descriptor); err != nil {
		facts.GovernanceError = "governor_denied"
		return facts
	}
	facts.GovernorPassed = true
	facts.ApprovalGranted = true
	return facts
}

func (r *ToolRegistry) handleGovernanceDenial(
	ctx context.Context,
	workspaceID string,
	descriptor CapabilityDescriptor,
	params json.RawMessage,
	startedAt time.Time,
	decision CapabilityRuntimeDecision,
) (json.RawMessage, error) {
	evidenceResult, evidenceErr := r.recordRuntimeEvidence(
		ctx, workspaceID, descriptor, decision,
		CapabilityStatusDenied, string(ToolErrorGovernanceDenied), params, nil,
	)
	extra := capabilityTerminalMetadata(decision, CapabilityStatusDenied, 0, evidenceResult)
	if evidenceErr != nil {
		extra["evidence_error_code"] = string(ToolErrorEvidenceIndeterminate)
	}
	return nil, r.handleExecutionError(
		ctx, workspaceID, descriptor.Name, params,
		ToolErrorGovernanceDenied, ErrCapabilityGovernanceDenied, startedAt, extra,
	)
}

func (r *ToolRegistry) executeGovernedProvider(
	ctx context.Context,
	workspaceID string,
	descriptor CapabilityDescriptor,
	executor ToolExecutor,
	params json.RawMessage,
	startedAt time.Time,
	decision CapabilityRuntimeDecision,
) (json.RawMessage, error) {
	attempts := descriptor.maxAttempts()
	for attempt := 1; attempt <= attempts; attempt++ {
		out, err := executor.Execute(ctx, params)
		if err == nil {
			return r.finalizeCapabilitySuccess(
				ctx, workspaceID, descriptor, params, out, startedAt, attempt, decision,
			)
		}
		failure := decideCapabilityFailure(descriptor, err, attempt, attempts)
		if failure.retry {
			continue
		}
		return nil, r.finalizeCapabilityFailure(
			ctx, workspaceID, descriptor, params, startedAt, attempt,
			decision, failure.code, failure.status, err,
		)
	}
	exhausted := errors.New("capability retry budget exhausted")
	return nil, r.finalizeCapabilityFailure(
		ctx, workspaceID, descriptor, params, startedAt, attempts,
		decision, ToolErrorCapabilityFailed, CapabilityStatusFailed, exhausted,
	)
}

func (r *ToolRegistry) finalizeCapabilitySuccess(
	ctx context.Context,
	workspaceID string,
	descriptor CapabilityDescriptor,
	params, output json.RawMessage,
	startedAt time.Time,
	attempt int,
	decision CapabilityRuntimeDecision,
) (json.RawMessage, error) {
	evidenceResult, evidenceErr := r.recordRuntimeEvidence(
		ctx, workspaceID, descriptor, decision,
		CapabilityStatusSucceeded, "", params, output,
	)
	extra := capabilityTerminalMetadata(decision, CapabilityStatusSucceeded, attempt, evidenceResult)
	if evidenceErr != nil {
		extra["evidence_error_code"] = string(ToolErrorEvidenceIndeterminate)
		err := r.handleExecutionError(
			ctx, workspaceID, descriptor.Name, params,
			ToolErrorEvidenceIndeterminate, evidenceErr, startedAt, extra,
		)
		return output, err
	}

	r.auditToolExecution(ctx, workspaceID, descriptor.Name, params, audit.OutcomeSuccess, "", extra)
	r.recordToolUsage(ctx, workspaceID, descriptor.Name, startedAt)
	return output, nil
}

func (r *ToolRegistry) finalizeCapabilityFailure(
	ctx context.Context,
	workspaceID string,
	descriptor CapabilityDescriptor,
	params json.RawMessage,
	startedAt time.Time,
	attempt int,
	decision CapabilityRuntimeDecision,
	code ExecutionErrorCode,
	status CapabilityExecutionStatus,
	executionErr error,
) error {
	evidenceResult, evidenceErr := r.recordRuntimeEvidence(
		ctx, workspaceID, descriptor, decision,
		status, string(code), params, nil,
	)
	extra := capabilityTerminalMetadata(decision, status, attempt, evidenceResult)
	if evidenceErr != nil {
		extra["evidence_error_code"] = string(ToolErrorEvidenceIndeterminate)
	}
	return r.handleExecutionError(
		ctx, workspaceID, descriptor.Name, params,
		code, executionErr, startedAt, extra,
	)
}

func (r *ToolRegistry) recordRuntimeEvidence(
	ctx context.Context,
	workspaceID string,
	descriptor CapabilityDescriptor,
	decision CapabilityRuntimeDecision,
	status CapabilityExecutionStatus,
	errorCode string,
	input, output json.RawMessage,
) (CapabilityEvidenceResult, error) {
	if !decision.EvidencePlanned {
		return CapabilityEvidenceResult{State: evidenceStateNotPlanned}, nil
	}
	if r.evidence == nil {
		return CapabilityEvidenceResult{State: evidenceStateIndeterminate}, ErrCapabilityEvidenceUnavailable
	}
	actorID, actorType := auditActorFromContext(ctx)
	result, err := r.evidence.RecordCapabilityEvidence(ctx, CapabilityEvidenceRequest{
		WorkspaceID: workspaceID,
		TraceID:     contextValue(ctx, ctxkeys.TraceID),
		ExecutionID: contextValue(ctx, ctxkeys.ExecutionID),
		RunID:       contextValue(ctx, ctxkeys.RunID),
		ApprovalID:  contextValue(ctx, ctxkeys.ApprovalID),
		ActorID:     actorID,
		ActorType:   string(actorType),
		Descriptor:  descriptor,
		Governance:  decision,
		Status:      status,
		ErrorCode:   errorCode,
		Input:       input,
		Output:      output,
	})
	if err != nil {
		if strings.TrimSpace(result.State) == "" {
			result.State = evidenceStateIndeterminate
		}
		return result, err
	}
	return result, nil
}

func capabilityTerminalMetadata(
	decision CapabilityRuntimeDecision,
	status CapabilityExecutionStatus,
	attempt int,
	evidenceResult CapabilityEvidenceResult,
) map[string]any {
	meta := map[string]any{
		"capability_execution_status": string(status),
		"attempt_count":               attempt,
		"governance": map[string]any{
			"allowed":              decision.Allowed,
			"denial_reason":        decision.DenialReason,
			"approval_required":    decision.ApprovalRequired,
			"evidence_requirement": string(decision.EvidenceRequirement),
			"evidence_planned":     decision.EvidencePlanned,
			"evidence_reason":      decision.EvidenceReason,
			"policy_reference":     decision.PolicyReference,
		},
		"evidence": evidenceAuditMetadata(evidenceResult),
	}
	return meta
}

func evidenceAuditMetadata(result CapabilityEvidenceResult) map[string]any {
	meta := map[string]any{"state": result.State}
	for key, value := range result.AuditMetadata {
		meta[key] = value
	}
	return meta
}
