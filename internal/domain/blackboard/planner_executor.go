package blackboard

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

const (
	DefaultPlannerExecutionResultMemoryKey  = "planning/last_execution_result"
	DefaultPlannerPendingExecutionMemoryKey = "planning/pending_execution"

	defaultPlannerApprovalTTL = 24 * time.Hour

	deferralReasonPolicyDeny      = "policy_deny"
	deferralReasonToolFailure     = "tool_failure"
	deferralReasonApprovalPending = "awaiting_approval"
	deferralReasonApprovalDenied  = "approval_rejected"
)

var (
	ErrPlannerExecutionPlanRequired     = errors.New("planner executor requires a collaborative plan")
	ErrPlannerExecutionWorkspaceID      = errors.New("planner executor requires workspace identifiers")
	ErrPlannerExecutionPendingNotFound  = errors.New("planner executor pending approval state not found")
	ErrPlannerExecutionApprovalMismatch = errors.New("planner executor approval decision does not match pending state")
)

type PlannerPolicyEffect string

const (
	PlannerPolicyEffectAllow         PlannerPolicyEffect = "allow"
	PlannerPolicyEffectDeny          PlannerPolicyEffect = "deny"
	PlannerPolicyEffectNeedsApproval PlannerPolicyEffect = "needs_approval"
)

type PlannerPolicyDecision struct {
	Effect PlannerPolicyEffect
	Reason string
}

type ExecutedStep struct {
	Step           ToolSequenceStep `json:"step"`
	IdempotencyKey string           `json:"idempotency_key"`
	Result         json.RawMessage  `json:"result,omitempty"`
	Error          string           `json:"error,omitempty"`
	ExecutedAt     time.Time        `json:"executed_at"`
}

type PendingStep struct {
	ApprovalID string           `json:"approval_id"`
	ApproverID string           `json:"approver_id"`
	Step       ToolSequenceStep `json:"step"`
}

type ExecutionOutcome struct {
	CognitiveWorkspaceID string         `json:"cognitive_workspace_id"`
	WorkspaceID          string         `json:"workspace_id"`
	ProposalID           string         `json:"proposal_id,omitempty"`
	Executed             []ExecutedStep `json:"executed,omitempty"`
	Pending              *PendingStep   `json:"pending,omitempty"`
	DeferralReason       string         `json:"deferral_reason,omitempty"`
}

type ApprovalDecision struct {
	CognitiveWorkspaceID string    `json:"cognitive_workspace_id"`
	ApprovalID           string    `json:"approval_id"`
	Approved             bool      `json:"approved"`
	DecidedBy            string    `json:"decided_by,omitempty"`
	DecidedAt            time.Time `json:"decided_at"`
}

type plannerPolicyEvaluator interface {
	DecideStep(ctx context.Context, workspaceID, actorID string, step ToolSequenceStep) (PlannerPolicyDecision, error)
}

type plannerApprovalRequester interface {
	CreateApprovalRequest(ctx context.Context, input policy.CreateApprovalRequestInput) (*policy.ApprovalRequest, error)
}

type plannerToolExecutor interface {
	Execute(ctx context.Context, workspaceID, toolName string, params json.RawMessage) (json.RawMessage, error)
}

type PlannerExecutor struct {
	memory            MemoryStore
	timeline          ReasoningTimeline
	policy            plannerPolicyEvaluator
	approval          plannerApprovalRequester
	tools             plannerToolExecutor
	audit             *audit.AuditService
	now               func() time.Time
	resultMemoryKey   string
	pendingMemoryKey  string
	defaultApproverID string
	approvalTTL       time.Duration
}

type plannerPendingExecutionState struct {
	CognitiveWorkspaceID string                    `json:"cognitive_workspace_id"`
	WorkspaceID          string                    `json:"workspace_id"`
	Proposal             CollaborativePlanProposal `json:"proposal"`
	PendingIndex         int                       `json:"pending_index"`
	ApprovalID           string                    `json:"approval_id"`
	ApproverID           string                    `json:"approver_id"`
	RequestedBy          string                    `json:"requested_by"`
	Executed             []ExecutedStep            `json:"executed,omitempty"`
	UpdatedAt            time.Time                 `json:"updated_at"`
}

type policyEngineAdapter struct {
	engine *policy.PolicyEngine
}

func NewPlannerExecutor(
	db *sql.DB,
	policyEngine *policy.PolicyEngine,
	approvalService *policy.ApprovalService,
	tools *tool.ToolRegistry,
	auditService *audit.AuditService,
) *PlannerExecutor {
	return &PlannerExecutor{
		memory:            NewMemoryStore(db),
		timeline:          NewReasoningTimeline(db),
		policy:            &policyEngineAdapter{engine: policyEngine},
		approval:          approvalService,
		tools:             tools,
		audit:             auditService,
		now:               func() time.Time { return time.Now().UTC() },
		resultMemoryKey:   DefaultPlannerExecutionResultMemoryKey,
		pendingMemoryKey:  DefaultPlannerPendingExecutionMemoryKey,
		defaultApproverID: "system",
		approvalTTL:       defaultPlannerApprovalTTL,
	}
}

func (e *PlannerExecutor) Execute(
	ctx context.Context,
	workspace CognitiveWorkspace,
	plan *CollaborativePlanningResult,
) (*ExecutionOutcome, error) {
	if err := validateExecutionInputs(workspace, plan); err != nil {
		return nil, err
	}

	outcome := newExecutionOutcome(workspace, plan.SelectedProposal)
	if plan.State != PlanningStateReady || plan.SelectedProposal == nil {
		outcome.DeferralReason = deferralReasonForPlan(plan)
		return outcome, e.persistExecutionOutcome(ctx, workspace.ID, outcome, true)
	}

	return e.runProposal(ctx, workspace, *plan.SelectedProposal, 0, nil, nil)
}

func (e *PlannerExecutor) ResumeFromApproval(ctx context.Context, decision ApprovalDecision) (*ExecutionOutcome, error) {
	if strings.TrimSpace(decision.CognitiveWorkspaceID) == "" || strings.TrimSpace(decision.ApprovalID) == "" {
		return nil, ErrPlannerExecutionPendingNotFound
	}

	state, workspace, outcome, prepErr := e.prepareResume(ctx, decision)
	if prepErr != nil {
		return nil, prepErr
	}

	if !decision.Approved {
		return e.finishRejectedApproval(ctx, workspace, state, outcome)
	}
	return e.resumeApproved(ctx, workspace, state)
}

func (e *PlannerExecutor) prepareResume(
	ctx context.Context,
	decision ApprovalDecision,
) (*plannerPendingExecutionState, CognitiveWorkspace, *ExecutionOutcome, error) {
	state, loadErr := e.loadPendingState(ctx, decision.CognitiveWorkspaceID)
	if loadErr != nil {
		return nil, CognitiveWorkspace{}, nil, loadErr
	}
	if state.ApprovalID != decision.ApprovalID {
		return nil, CognitiveWorkspace{}, nil, ErrPlannerExecutionApprovalMismatch
	}

	workspace := CognitiveWorkspace{ID: state.CognitiveWorkspaceID, WorkspaceID: state.WorkspaceID}
	outcome := newExecutionOutcome(workspace, &state.Proposal)
	outcome.Executed = append(outcome.Executed, cloneExecuted(state.Executed)...)
	return state, workspace, outcome, nil
}

func (e *PlannerExecutor) finishRejectedApproval(
	ctx context.Context,
	workspace CognitiveWorkspace,
	state *plannerPendingExecutionState,
	outcome *ExecutionOutcome,
) (*ExecutionOutcome, error) {
	outcome.Pending = nil
	outcome.DeferralReason = deferralReasonApprovalDenied
	step := state.Proposal.Steps[state.PendingIndex]
	finalOutcome, finishErr := e.finishStoppedExecution(
		ctx,
		workspace,
		state.Proposal,
		step,
		outcome,
		deferralReasonApprovalDenied,
		EventTypeRisk,
		"approval rejected; execution stopped",
		"planner.execution.approval_rejected",
		audit.OutcomeDenied,
		map[string]any{"approval_id": state.ApprovalID},
	)
	if finishErr != nil {
		return nil, finishErr
	}
	return finalOutcome, nil
}

func (e *PlannerExecutor) resumeApproved(
	ctx context.Context,
	workspace CognitiveWorkspace,
	state *plannerPendingExecutionState,
) (*ExecutionOutcome, error) {
	step := state.Proposal.Steps[state.PendingIndex]
	timelineErr := e.appendTimelineEvent(ctx, workspace.ID, EventTypeIntent, map[string]any{
		"message":       "approval approved; resuming planner execution",
		"approval_id":   state.ApprovalID,
		"proposal_id":   state.Proposal.ProposalID,
		"step_sequence": step.Sequence,
	})
	if timelineErr != nil {
		return nil, timelineErr
	}

	skipApproval := map[int]string{step.Sequence: state.ApprovalID}
	return e.runProposal(ctx, workspace, state.Proposal, state.PendingIndex, state.Executed, skipApproval)
}

func (e *PlannerExecutor) runProposal(
	ctx context.Context,
	workspace CognitiveWorkspace,
	proposal CollaborativePlanProposal,
	startIndex int,
	executed []ExecutedStep,
	skipApproval map[int]string,
) (*ExecutionOutcome, error) {
	outcome := newExecutionOutcome(workspace, &proposal)
	outcome.Executed = append(outcome.Executed, cloneExecuted(executed)...)

	for idx := startIndex; idx < len(proposal.Steps); idx++ {
		finished, processErr := e.processStep(ctx, workspace, proposal, idx, outcome, skipApproval)
		if processErr != nil {
			return nil, processErr
		}
		if finished != nil {
			return finished, nil
		}
	}

	auditErr := e.logAudit(ctx, workspace.WorkspaceID, proposal.ProposalID, "planner.execution.completed", audit.OutcomeSuccess, map[string]any{
		"executed_steps": len(outcome.Executed),
	})
	if auditErr != nil {
		return nil, auditErr
	}
	return outcome, e.persistExecutionOutcome(ctx, workspace.ID, outcome, true)
}

func (e *PlannerExecutor) processStep(
	ctx context.Context,
	workspace CognitiveWorkspace,
	proposal CollaborativePlanProposal,
	index int,
	outcome *ExecutionOutcome,
	skipApproval map[int]string,
) (*ExecutionOutcome, error) {
	step := proposal.Steps[index]
	decision, decisionErr := e.decideStep(ctx, workspace.WorkspaceID, step)
	if decisionErr != nil {
		return nil, decisionErr
	}

	if shouldDeny(decision) {
		return e.finishPolicyDeny(ctx, workspace, proposal, step, decision, outcome)
	}
	if needsApproval(step, decision, skipApproval) {
		return e.finishPendingApproval(ctx, workspace, proposal, index, step, outcome)
	}
	if bypassedApproval(step, skipApproval) {
		delete(skipApproval, step.Sequence)
	}

	executedStep, execErr := e.executeStep(ctx, workspace, proposal, step)
	outcome.Executed = append(outcome.Executed, executedStep)
	if execErr != nil {
		return e.finishToolFailure(ctx, workspace, proposal, step, executedStep, outcome)
	}
	return nil, nil
}

func (e *PlannerExecutor) decideStep(
	ctx context.Context,
	workspaceID string,
	step ToolSequenceStep,
) (PlannerPolicyDecision, error) {
	if e.policy == nil {
		return PlannerPolicyDecision{Effect: PlannerPolicyEffectAllow}, nil
	}
	decision, decideErr := e.policy.DecideStep(ctx, workspaceID, actorIDFromContext(ctx), step)
	if decideErr != nil {
		return PlannerPolicyDecision{}, fmt.Errorf("planner policy decide step: %w", decideErr)
	}
	return decision, nil
}

func (e *PlannerExecutor) executeStep(
	ctx context.Context,
	workspace CognitiveWorkspace,
	proposal CollaborativePlanProposal,
	step ToolSequenceStep,
) (ExecutedStep, error) {
	key := idempotencyKey(workspace.WorkspaceID, proposal.ProposalID, step.Sequence)
	now := e.currentTime()
	params := normalizeStepParams(step.Params)
	result := ExecutedStep{
		Step:           step,
		IdempotencyKey: key,
		ExecutedAt:     now,
	}

	if e.tools == nil {
		result.Error = "planner executor tool registry is not configured"
		if err := e.appendObservation(ctx, workspace.ID, proposal.ProposalID, result); err != nil {
			return result, err
		}
		return result, errors.New(result.Error)
	}

	raw, toolErr := e.tools.Execute(ctx, workspace.WorkspaceID, step.ToolName, params)
	if raw != nil {
		result.Result = raw
	}
	var execErr error
	if toolErr != nil {
		result.Error = toolErr.Error()
		execErr = fmt.Errorf("planner tool execute %s: %w", step.ToolName, toolErr)
	}
	if appendErr := e.appendObservation(ctx, workspace.ID, proposal.ProposalID, result); appendErr != nil {
		return result, appendErr
	}
	return result, execErr
}

func (e *PlannerExecutor) finishPolicyDeny(
	ctx context.Context,
	workspace CognitiveWorkspace,
	proposal CollaborativePlanProposal,
	step ToolSequenceStep,
	decision PlannerPolicyDecision,
	outcome *ExecutionOutcome,
) (*ExecutionOutcome, error) {
	return e.finishStoppedExecution(
		ctx,
		workspace,
		proposal,
		step,
		outcome,
		deferralReasonPolicyDeny,
		EventTypeRisk,
		"planner execution denied by policy",
		"planner.execution.denied",
		audit.OutcomeDenied,
		map[string]any{"reason": decision.Reason},
	)
}

func (e *PlannerExecutor) finishPendingApproval(
	ctx context.Context,
	workspace CognitiveWorkspace,
	proposal CollaborativePlanProposal,
	index int,
	step ToolSequenceStep,
	outcome *ExecutionOutcome,
) (*ExecutionOutcome, error) {
	request, err := e.createApprovalRequest(ctx, workspace, proposal, step)
	if err != nil {
		return nil, err
	}
	outcome.Pending = &PendingStep{
		ApprovalID: request.ID,
		ApproverID: request.ApproverID,
		Step:       step,
	}
	outcome.DeferralReason = deferralReasonApprovalPending

	state := plannerPendingExecutionState{
		CognitiveWorkspaceID: workspace.ID,
		WorkspaceID:          workspace.WorkspaceID,
		Proposal:             proposal,
		PendingIndex:         index,
		ApprovalID:           request.ID,
		ApproverID:           request.ApproverID,
		RequestedBy:          actorIDFromContext(ctx),
		Executed:             cloneExecuted(outcome.Executed),
		UpdatedAt:            e.currentTime(),
	}
	persistErr := e.persistPendingState(ctx, workspace.ID, state)
	if persistErr != nil {
		return nil, persistErr
	}
	timelineErr := e.appendTimelineEvent(ctx, workspace.ID, EventTypeIntent, map[string]any{
		"message":       "planner execution awaiting approval",
		"approval_id":   request.ID,
		"proposal_id":   proposal.ProposalID,
		"step_sequence": step.Sequence,
		"tool_name":     step.ToolName,
	})
	if timelineErr != nil {
		return nil, timelineErr
	}
	auditErr := e.logAudit(ctx, workspace.WorkspaceID, proposal.ProposalID, "planner.execution.awaiting_approval", audit.OutcomeSuccess, map[string]any{
		"approval_id":   request.ID,
		"step_sequence": step.Sequence,
		"tool_name":     step.ToolName,
	})
	if auditErr != nil {
		return nil, auditErr
	}
	return outcome, e.persistExecutionOutcome(ctx, workspace.ID, outcome, false)
}

func (e *PlannerExecutor) finishToolFailure(
	ctx context.Context,
	workspace CognitiveWorkspace,
	proposal CollaborativePlanProposal,
	step ToolSequenceStep,
	failed ExecutedStep,
	outcome *ExecutionOutcome,
) (*ExecutionOutcome, error) {
	return e.finishStoppedExecution(
		ctx,
		workspace,
		proposal,
		step,
		outcome,
		deferralReasonToolFailure,
		EventTypeRecommendation,
		"planner execution stopped after tool failure; compensating action recommended",
		"planner.execution.failed",
		audit.OutcomeError,
		map[string]any{"error": failed.Error},
	)
}

func (e *PlannerExecutor) createApprovalRequest(
	ctx context.Context,
	workspace CognitiveWorkspace,
	proposal CollaborativePlanProposal,
	step ToolSequenceStep,
) (*policy.ApprovalRequest, error) {
	if e.approval == nil {
		return nil, errors.New("planner executor approval service is not configured")
	}
	requestedBy := actorIDFromContext(ctx)
	approverID := e.defaultApproverID
	if strings.TrimSpace(approverID) == "" {
		approverID = requestedBy
	}
	reason := step.Reason
	action := fmt.Sprintf("planner.%s", step.ToolName)
	resourceType := "tool"
	resourceID := step.ToolName
	payload, err := json.Marshal(map[string]any{
		"proposal_id":            proposal.ProposalID,
		"cognitive_workspace_id": workspace.ID,
		"workspace_id":           workspace.WorkspaceID,
		"step":                   step,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal planner approval payload: %w", err)
	}
	req, approvalErr := e.approval.CreateApprovalRequest(ctx, policy.CreateApprovalRequestInput{
		WorkspaceID:  workspace.WorkspaceID,
		RequestedBy:  requestedBy,
		ApproverID:   approverID,
		Action:       action,
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		Payload:      payload,
		Reason:       &reason,
		ExpiresAt:    e.currentTime().Add(e.approvalDuration()),
	})
	if approvalErr != nil {
		return nil, fmt.Errorf("planner create approval request: %w", approvalErr)
	}
	return req, nil
}

func (e *PlannerExecutor) persistExecutionOutcome(ctx context.Context, cognitiveWorkspaceID string, outcome *ExecutionOutcome, clearPending bool) error {
	if clearPending {
		if deleteErr := e.memory.Delete(ctx, cognitiveWorkspaceID, e.pendingKey()); deleteErr != nil {
			return fmt.Errorf("planner delete pending state: %w", deleteErr)
		}
	}
	return e.persistMemoryJSON(ctx, cognitiveWorkspaceID, e.resultKey(), outcome)
}

func (e *PlannerExecutor) persistPendingState(ctx context.Context, cognitiveWorkspaceID string, state plannerPendingExecutionState) error {
	return e.persistMemoryJSON(ctx, cognitiveWorkspaceID, e.pendingKey(), state)
}

func (e *PlannerExecutor) loadPendingState(ctx context.Context, cognitiveWorkspaceID string) (*plannerPendingExecutionState, error) {
	entry, err := e.memory.Get(ctx, cognitiveWorkspaceID, e.pendingKey())
	if errors.Is(err, ErrMemoryNotFound) || errors.Is(err, ErrMemoryExpired) {
		return nil, ErrPlannerExecutionPendingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("planner load pending state: %w", err)
	}

	var state plannerPendingExecutionState
	unmarshalErr := json.Unmarshal(entry.Value, &state)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal pending planner execution state: %w", unmarshalErr)
	}
	return &state, nil
}

func (e *PlannerExecutor) persistMemoryJSON(ctx context.Context, cognitiveWorkspaceID, key string, value any) error {
	if e.memory == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal planner execution memory %s: %w", key, err)
	}
	now := e.currentTime()
	setErr := e.memory.Set(ctx, AgentMemory{
		ID:                   uuid.NewV7().String(),
		CognitiveWorkspaceID: cognitiveWorkspaceID,
		Key:                  key,
		Value:                raw,
		Scope:                MemoryScopeSession,
		CreatedAt:            now,
		UpdatedAt:            now,
	})
	if setErr != nil {
		return fmt.Errorf("planner persist memory %s: %w", key, setErr)
	}
	return nil
}

func (e *PlannerExecutor) appendObservation(
	ctx context.Context,
	cognitiveWorkspaceID, proposalID string,
	step ExecutedStep,
) error {
	payload := map[string]any{
		"proposal_id":     proposalID,
		"step_sequence":   step.Step.Sequence,
		"tool_name":       step.Step.ToolName,
		"idempotency_key": step.IdempotencyKey,
		"error":           step.Error,
	}
	if len(step.Result) > 0 {
		payload["result"] = json.RawMessage(step.Result)
	}
	return e.appendTimelineEvent(ctx, cognitiveWorkspaceID, EventTypeObservation, payload)
}

func (e *PlannerExecutor) appendTimelineEvent(
	ctx context.Context,
	cognitiveWorkspaceID string,
	eventType EventType,
	payload any,
) error {
	if e.timeline == nil {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal planner timeline payload: %w", err)
	}
	appendErr := e.timeline.Append(ctx, ReasoningEvent{
		ID:                   uuid.NewV7().String(),
		CognitiveWorkspaceID: cognitiveWorkspaceID,
		EventType:            eventType,
		Payload:              raw,
		CreatedAt:            e.currentTime(),
	})
	if appendErr != nil {
		return fmt.Errorf("planner append timeline: %w", appendErr)
	}
	return nil
}

func (e *PlannerExecutor) logAudit(
	ctx context.Context,
	workspaceID, proposalID, action string,
	outcome audit.Outcome,
	metadata map[string]any,
) error {
	if e.audit == nil {
		return nil
	}
	entityType := "planner_execution"
	entityID := proposalID
	auditErr := e.audit.LogWithDetails(
		ctx,
		workspaceID,
		actorIDFromContext(ctx),
		actorTypeFromContext(ctx),
		action,
		&entityType,
		&entityID,
		&audit.EventDetails{Metadata: metadata},
		outcome,
	)
	if auditErr != nil {
		return fmt.Errorf("planner log audit: %w", auditErr)
	}
	return nil
}

func (a *policyEngineAdapter) DecideStep(
	ctx context.Context,
	workspaceID, actorID string,
	step ToolSequenceStep,
) (PlannerPolicyDecision, error) {
	if a == nil || a.engine == nil {
		return PlannerPolicyDecision{Effect: PlannerPolicyEffectAllow}, nil
	}
	decision, err := a.engine.EvaluatePolicyDecision(ctx, workspaceID, actorID, "tools", step.ToolName, nil)
	if err != nil {
		return PlannerPolicyDecision{}, fmt.Errorf("policy engine evaluate decision: %w", err)
	}
	if !decision.Allow {
		return PlannerPolicyDecision{
			Effect: PlannerPolicyEffectDeny,
			Reason: policyDecisionReason(decision),
		}, nil
	}
	return PlannerPolicyDecision{Effect: PlannerPolicyEffectAllow}, nil
}

func validateExecutionInputs(workspace CognitiveWorkspace, plan *CollaborativePlanningResult) error {
	if plan == nil {
		return ErrPlannerExecutionPlanRequired
	}
	if strings.TrimSpace(workspace.ID) == "" ||
		strings.TrimSpace(workspace.WorkspaceID) == "" ||
		(plan.CognitiveWorkspaceID != "" && plan.CognitiveWorkspaceID != workspace.ID) {
		return ErrPlannerExecutionWorkspaceID
	}
	return nil
}

func (e *PlannerExecutor) finishStoppedExecution(
	ctx context.Context,
	workspace CognitiveWorkspace,
	proposal CollaborativePlanProposal,
	step ToolSequenceStep,
	outcome *ExecutionOutcome,
	deferralReason string,
	eventType EventType,
	message string,
	auditAction string,
	auditOutcome audit.Outcome,
	metadata map[string]any,
) (*ExecutionOutcome, error) {
	outcome.DeferralReason = deferralReason
	payload := mergeExecutionMetadata(step, proposal.ProposalID, metadata)
	payload["message"] = message

	timelineErr := e.appendTimelineEvent(ctx, workspace.ID, eventType, payload)
	if timelineErr != nil {
		return nil, timelineErr
	}
	auditErr := e.logAudit(ctx, workspace.WorkspaceID, proposal.ProposalID, auditAction, auditOutcome, payload)
	if auditErr != nil {
		return nil, auditErr
	}
	return outcome, e.persistExecutionOutcome(ctx, workspace.ID, outcome, true)
}

func mergeExecutionMetadata(step ToolSequenceStep, proposalID string, metadata map[string]any) map[string]any {
	payload := map[string]any{
		"proposal_id":   proposalID,
		"step_sequence": step.Sequence,
		"tool_name":     step.ToolName,
	}
	for key, value := range metadata {
		payload[key] = value
	}
	return payload
}

func newExecutionOutcome(workspace CognitiveWorkspace, proposal *CollaborativePlanProposal) *ExecutionOutcome {
	outcome := &ExecutionOutcome{
		CognitiveWorkspaceID: workspace.ID,
		WorkspaceID:          workspace.WorkspaceID,
		Executed:             []ExecutedStep{},
	}
	if proposal != nil {
		outcome.ProposalID = proposal.ProposalID
	}
	return outcome
}

func deferralReasonForPlan(plan *CollaborativePlanningResult) string {
	if plan == nil {
		return string(PlanningStateNoAction)
	}
	if plan.SelectedProposal == nil && plan.State == PlanningStateReady {
		return "missing_selected_proposal"
	}
	if plan.State == "" {
		return string(PlanningStateNoAction)
	}
	return string(plan.State)
}

func idempotencyKey(workspaceID, proposalID string, sequence int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d", workspaceID, proposalID, sequence)))
	return hex.EncodeToString(sum[:])
}

func shouldDeny(decision PlannerPolicyDecision) bool {
	return decision.Effect == PlannerPolicyEffectDeny
}

func needsApproval(step ToolSequenceStep, decision PlannerPolicyDecision, skipApproval map[int]string) bool {
	if bypassedApproval(step, skipApproval) {
		return false
	}
	return step.RequiresApproval || decision.Effect == PlannerPolicyEffectNeedsApproval
}

func bypassedApproval(step ToolSequenceStep, skipApproval map[int]string) bool {
	if len(skipApproval) == 0 {
		return false
	}
	_, ok := skipApproval[step.Sequence]
	return ok
}

func normalizeStepParams(params json.RawMessage) json.RawMessage {
	if len(params) == 0 {
		return json.RawMessage(`{}`)
	}
	return params
}

func policyDecisionReason(decision policy.PolicyDecision) string {
	if decision.Trace != nil && strings.TrimSpace(decision.Trace.MatchedEffect) != "" {
		return decision.Trace.MatchedEffect
	}
	return "policy_denied"
}

func cloneExecuted(in []ExecutedStep) []ExecutedStep {
	if len(in) == 0 {
		return []ExecutedStep{}
	}
	out := make([]ExecutedStep, 0, len(in))
	for _, item := range in {
		cloned := item
		if len(item.Result) > 0 {
			cloned.Result = append(json.RawMessage(nil), item.Result...)
		}
		out = append(out, cloned)
	}
	return out
}

func actorIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(ctxkeys.UserID).(string); ok && strings.TrimSpace(userID) != "" {
		return userID
	}
	return "system"
}

func actorTypeFromContext(ctx context.Context) audit.ActorType {
	if userID, ok := ctx.Value(ctxkeys.UserID).(string); ok && strings.TrimSpace(userID) != "" {
		return audit.ActorTypeUser
	}
	return audit.ActorTypeSystem
}

func (e *PlannerExecutor) currentTime() time.Time {
	if e != nil && e.now != nil {
		return e.now().UTC()
	}
	return time.Now().UTC()
}

func (e *PlannerExecutor) resultKey() string {
	if e != nil && strings.TrimSpace(e.resultMemoryKey) != "" {
		return e.resultMemoryKey
	}
	return DefaultPlannerExecutionResultMemoryKey
}

func (e *PlannerExecutor) pendingKey() string {
	if e != nil && strings.TrimSpace(e.pendingMemoryKey) != "" {
		return e.pendingMemoryKey
	}
	return DefaultPlannerPendingExecutionMemoryKey
}

func (e *PlannerExecutor) approvalDuration() time.Duration {
	if e != nil && e.approvalTTL > 0 {
		return e.approvalTTL
	}
	return defaultPlannerApprovalTTL
}
