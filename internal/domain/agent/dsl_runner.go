package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

var (
	ErrDSLRunnerMissingOrchestrator = errors.New("dsl runner requires orchestrator")
	ErrDSLWorkflowNotFound          = errors.New("dsl workflow not found")
	ErrDSLResumeInvalidInput        = errors.New("dsl resume input is invalid")
	ErrDSLWorkflowNotActive         = errors.New("dsl workflow is not active")
)

const dslStatementTypeDispatch = "DISPATCH"

type DSLRunner struct {
	workflowService workflowResolver
	runtime         *DSLRuntime
	executor        RuntimeOperationExecutor

	cacheMu  sync.RWMutex
	astCache map[string]*Program
}

type workflowResolver interface {
	Get(ctx context.Context, workspaceID, workflowID string) (*workflowdomain.Workflow, error)
	GetActiveByAgentDefinition(ctx context.Context, workspaceID, agentDefinitionID string) (*workflowdomain.Workflow, error)
}

type DSLRunOutput struct {
	WorkflowID      string               `json:"workflow_id"`
	WorkflowName    string               `json:"workflow_name"`
	WorkflowVersion int                  `json:"workflow_version"`
	Statements      []DSLStatementResult `json:"statements"`
}

func NewDSLRunner(db *sql.DB) *DSLRunner {
	return NewDSLRunnerWithDependencies(workflowdomain.NewService(db), NewDSLRuntime(), nil)
}

func NewDSLRunnerWithDependencies(service workflowResolver, runtime *DSLRuntime, executor RuntimeOperationExecutor) *DSLRunner {
	if runtime == nil {
		runtime = NewDSLRuntime()
	}
	return &DSLRunner{
		workflowService: service,
		runtime:         runtime,
		executor:        executor,
		astCache:        make(map[string]*Program),
	}
}

func (r *DSLRunner) InvalidateCache(workflowID string) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	delete(r.astCache, workflowID)
}

func (r *DSLRunner) Run(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*Run, error) {
	if rc == nil || rc.Orchestrator == nil {
		return nil, ErrDSLRunnerMissingOrchestrator
	}
	workflow, program, run, err := r.initializeRun(ctx, rc, input)
	if err != nil {
		return nil, err
	}
	evalCtx := mergeDSLContexts(input.TriggerContext, input.Inputs)
	carta := parseCartaWorkflowSpec(workflow)
	if early, earlyErr := r.runPreflights(ctx, rc, workflow, carta, input, run, evalCtx); earlyErr != nil || early != nil {
		return early, earlyErr
	}
	execCtx := rc.WithCall(input.AgentID)
	baseExecutor, defaultExecutor := r.buildBaseExecutor(execCtx, input, evalCtx, workflow.ID, run.ID)
	executor := wrapWithTrace(input.WorkspaceID, run.ID, execCtx, r.runtime, baseExecutor, defaultExecutor)
	result, execErr := r.runtime.ExecuteProgram(ctx, program, evalCtx, executor)
	return r.finalizeRun(ctx, rc, input.WorkspaceID, run.ID, workflow, result, defaultExecutor, execErr)
}

func (r *DSLRunner) initializeRun(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*workflowdomain.Workflow, *Program, *Run, error) {
	workflow, err := r.loadActiveWorkflow(ctx, input.WorkspaceID, input.AgentID)
	if err != nil {
		return nil, nil, nil, err
	}
	program, err := r.loadProgram(workflow)
	if err != nil {
		return nil, nil, nil, err
	}
	run, err := rc.Orchestrator.TriggerAgent(ctx, input)
	if err != nil {
		return nil, nil, nil, err
	}
	run, err = rc.Orchestrator.UpdateAgentRunStatus(ctx, input.WorkspaceID, run.ID, StatusAccepted)
	if err != nil {
		return nil, nil, nil, err
	}
	return workflow, program, run, nil
}

func (r *DSLRunner) runPreflights(ctx context.Context, rc *RunContext, workflow *workflowdomain.Workflow, carta *CartaSummary, input TriggerAgentInput, run *Run, evalCtx map[string]any) (*Run, error) {
	if delegated, handoffErr := r.preflightDelegate(ctx, rc, workflow, carta, input, run, evalCtx); handoffErr != nil || delegated != nil {
		return delegated, handoffErr
	}
	return r.preflightGrounds(ctx, rc, carta, input, run)
}

func (r *DSLRunner) Resume(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload) (*Run, error) {
	run, workflow, program, err := r.prepareResumeWithProgram(ctx, rc, workspaceID, input)
	if err != nil {
		return run, err
	}

	execCtx := rc.WithCall(run.DefinitionID)
	triggerInput := buildResumeTriggerInput(run, workspaceID)
	evalCtx := mergeDSLContexts(run.TriggerContext, run.Inputs)
	carta := parseCartaWorkflowSpec(workflow)
	if early, earlyErr := r.runPreflights(ctx, rc, workflow, carta, triggerInput, run, evalCtx); earlyErr != nil || early != nil {
		return early, earlyErr
	}
	baseExecutor, defaultExecutor := r.buildBaseExecutor(execCtx, triggerInput, evalCtx, workflow.ID, input.RunID)
	executor := wrapWithTrace(workspaceID, input.RunID, execCtx, r.runtime, baseExecutor, defaultExecutor)
	result, execErr := r.runtime.ExecuteProgramFromIndex(ctx, program, input.ResumeStepIndex, evalCtx, executor)
	return r.finalizeResumedRun(ctx, rc, workspaceID, input.RunID, input, workflow, run, result, defaultExecutor, execErr)
}

func (r *DSLRunner) prepareResumeWithProgram(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload) (*Run, *workflowdomain.Workflow, *Program, error) {
	run, workflow, err := r.prepareResume(ctx, rc, workspaceID, input)
	if err != nil {
		if run != nil {
			failed, failErr := r.failResumeAndReturnErr(ctx, rc, workspaceID, input, workflow, run, err)
			return failed, workflow, nil, failErr
		}
		return nil, nil, nil, err
	}
	program, err := r.loadProgram(workflow)
	if err != nil {
		failed, failErr := r.failResumeAndReturnErr(ctx, rc, workspaceID, input, workflow, run, err)
		return failed, workflow, nil, failErr
	}
	if _, err = rc.Orchestrator.UpdateAgentRunStatus(ctx, workspaceID, input.RunID, StatusAccepted); err != nil {
		failed, failErr := r.failResumeAndReturnErr(ctx, rc, workspaceID, input, workflow, run, err)
		return failed, workflow, nil, failErr
	}
	return run, workflow, program, nil
}

func buildResumeTriggerInput(run *Run, workspaceID string) TriggerAgentInput {
	return TriggerAgentInput{
		AgentID:        run.DefinitionID,
		WorkspaceID:    workspaceID,
		TriggeredBy:    run.TriggeredByUserID,
		TriggerType:    run.TriggerType,
		TriggerContext: run.TriggerContext,
		Inputs:         run.Inputs,
	}
}

func (r *DSLRunner) prepareResume(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload) (*Run, *workflowdomain.Workflow, error) {
	if err := validateResumeInput(rc, workspaceID, input); err != nil {
		return nil, nil, err
	}

	run, err := rc.Orchestrator.GetAgentRun(ctx, workspaceID, input.RunID)
	if err != nil {
		return nil, nil, err
	}
	workflow, err := r.loadWorkflowForResume(ctx, workspaceID, input.WorkflowID)
	if err != nil {
		return run, workflow, err
	}
	if validateErr := validateResumeDefinition(run, workflow); validateErr != nil {
		return run, workflow, validateErr
	}
	if _, err = rc.Orchestrator.UpdateAgentRunStatus(ctx, workspaceID, input.RunID, StatusAccepted); err != nil {
		return run, workflow, err
	}
	return run, workflow, nil
}

func validateResumeInput(rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload) error {
	if rc == nil || rc.Orchestrator == nil {
		return ErrDSLRunnerMissingOrchestrator
	}
	if strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(input.WorkflowID) == "" || strings.TrimSpace(input.RunID) == "" || input.ResumeStepIndex < 0 {
		return ErrDSLResumeInvalidInput
	}
	return nil
}

func validateResumeDefinition(run *Run, workflow *workflowdomain.Workflow) error {
	if workflow.AgentDefinitionID != nil && *workflow.AgentDefinitionID == run.DefinitionID {
		return nil
	}
	return fmt.Errorf("%w: run/workflow definition mismatch", ErrDSLResumeInvalidInput)
}

func (r *DSLRunner) buildBaseExecutor(rc *RunContext, input TriggerAgentInput, evalCtx map[string]any, workflowID, runID string) (RuntimeOperationExecutor, *dslRuntimeExecutor) {
	if r.executor != nil {
		return r.executor, nil
	}
	de := newDSLRuntimeExecutor(rc, input, evalCtx, workflowID, runID)
	return de, de
}

func wrapWithTrace(workspaceID, runID string, rc *RunContext, runtime *DSLRuntime, base RuntimeOperationExecutor, defaultExecutor *dslRuntimeExecutor) RuntimeOperationExecutor {
	if defaultExecutor == nil {
		return base
	}
	return newTracedDSLExecutor(workspaceID, runID, rc, runtime, defaultExecutor)
}

func (r *DSLRunner) finalizeRun(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *workflowdomain.Workflow, result *DSLRuntimeResult, defaultExecutor *dslRuntimeExecutor, execErr error) (*Run, error) {
	toolCalls := dslToolCallsJSON(defaultExecutor)
	if execErr != nil {
		return r.finalizeFailure(ctx, rc, workspaceID, runID, workflow, result, toolCalls, execErr)
	}
	if defaultExecutor != nil && defaultExecutor.IsPending() {
		return r.finalizePending(ctx, rc, workspaceID, runID, workflow, result, defaultExecutor)
	}
	if terminalStatus, ok := terminalDispatchStatus(result); ok {
		return r.finalizeDispatchTerminal(ctx, rc, workspaceID, runID, workflow, result, toolCalls, terminalStatus)
	}
	return r.finalizeSuccess(ctx, rc, workspaceID, runID, workflow, result, toolCalls)
}

func dslToolCallsJSON(defaultExecutor *dslRuntimeExecutor) json.RawMessage {
	if defaultExecutor != nil {
		return defaultExecutor.ToolCallsJSON()
	}
	return json.RawMessage(emptyJSONArray)
}

func (r *DSLRunner) loadActiveWorkflow(ctx context.Context, workspaceID, agentDefinitionID string) (*workflowdomain.Workflow, error) {
	if r.workflowService == nil {
		return nil, ErrDSLWorkflowNotFound
	}
	item, err := r.workflowService.GetActiveByAgentDefinition(ctx, workspaceID, agentDefinitionID)
	if err != nil {
		if errors.Is(err, workflowdomain.ErrWorkflowNotFound) {
			return nil, ErrDSLWorkflowNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *DSLRunner) loadWorkflowForResume(ctx context.Context, workspaceID, workflowID string) (*workflowdomain.Workflow, error) {
	if r.workflowService == nil {
		return nil, ErrDSLWorkflowNotFound
	}
	item, err := r.workflowService.Get(ctx, workspaceID, workflowID)
	if err != nil {
		if errors.Is(err, workflowdomain.ErrWorkflowNotFound) {
			return nil, ErrDSLWorkflowNotFound
		}
		return nil, err
	}
	if item.Status != workflowdomain.StatusActive {
		return nil, ErrDSLWorkflowNotActive
	}
	return item, nil
}

func (r *DSLRunner) loadProgram(workflow *workflowdomain.Workflow) (*Program, error) {
	r.cacheMu.RLock()
	cached := r.astCache[workflow.ID]
	r.cacheMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	program, err := ParseAndValidateDSL(workflow.DSLSource)
	if err != nil {
		return nil, err
	}

	r.cacheMu.Lock()
	r.astCache[workflow.ID] = program
	r.cacheMu.Unlock()
	return program, nil
}

func (r *DSLRunner) finalizeSuccess(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *workflowdomain.Workflow, result *DSLRuntimeResult, toolCalls json.RawMessage) (*Run, error) {
	return r.updateRunWithPayload(ctx, rc, workspaceID, runID, StatusSuccess, buildDSLRunOutputPayload(workflow, result), toolCalls, true)
}

func (r *DSLRunner) finalizeResumeSuccess(ctx context.Context, rc *RunContext, workspaceID, runID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, result *DSLRuntimeResult, toolCalls json.RawMessage) (*Run, error) {
	return r.updateRunWithPayload(ctx, rc, workspaceID, runID, StatusSuccess, buildDSLResumePayload(workflow, input.ResumeStepIndex, result), mergeRunToolCalls(existing.ToolCalls, toolCalls), true)
}

func (r *DSLRunner) finalizeFailure(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *workflowdomain.Workflow, result *DSLRuntimeResult, toolCalls json.RawMessage, execErr error) (*Run, error) {
	return r.updateRunWithPayload(ctx, rc, workspaceID, runID, StatusFailed, buildDSLFailurePayload(workflow, result, execErr), toolCalls, true)
}

func (r *DSLRunner) finalizeResumeFailure(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, execErr error) (*Run, error) {
	if rc == nil || rc.Orchestrator == nil || existing == nil {
		return nil, execErr
	}
	return r.updateRunWithPayload(ctx, rc, workspaceID, input.RunID, StatusFailed, buildDSLResumeFailurePayload(input, workflow, execErr), existing.ToolCalls, true)
}

func (r *DSLRunner) failResumeAndReturnErr(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, resumeErr error) (*Run, error) {
	run, err := r.finalizeResumeFailure(ctx, rc, workspaceID, input, workflow, existing, resumeErr)
	if err != nil {
		return nil, err
	}
	return run, resumeErr
}

func (r *DSLRunner) finalizePending(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *workflowdomain.Workflow, result *DSLRuntimeResult, executor *dslRuntimeExecutor) (*Run, error) {
	return r.updateRunWithPayload(ctx, rc, workspaceID, runID, StatusAccepted, buildPendingPayload(buildDSLWorkflowPayload(workflow, result), executor), executor.ToolCallsJSON(), false)
}

func (r *DSLRunner) finalizeResumePending(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, result *DSLRuntimeResult, executor *dslRuntimeExecutor) (*Run, error) {
	return r.updateRunWithPayload(ctx, rc, workspaceID, input.RunID, StatusAccepted, buildPendingPayload(buildDSLResumePayload(workflow, input.ResumeStepIndex, result), executor), mergeRunToolCalls(existing.ToolCalls, executor.ToolCallsJSON()), false)
}

func (r *DSLRunner) finalizeResumedRun(ctx context.Context, rc *RunContext, workspaceID, runID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, result *DSLRuntimeResult, defaultExecutor *dslRuntimeExecutor, execErr error) (*Run, error) {
	toolCalls := dslToolCallsJSON(defaultExecutor)
	if execErr != nil {
		return r.finalizeResumeFailure(ctx, rc, workspaceID, input, workflow, existing, execErr)
	}
	if defaultExecutor != nil && defaultExecutor.IsPending() {
		return r.finalizeResumePending(ctx, rc, workspaceID, input, workflow, existing, result, defaultExecutor)
	}
	if terminalStatus, ok := terminalDispatchStatus(result); ok {
		return r.finalizeResumeDispatchTerminal(ctx, rc, workspaceID, input, workflow, existing, result, terminalStatus)
	}
	return r.finalizeResumeSuccess(ctx, rc, workspaceID, runID, input, workflow, existing, result, toolCalls)
}

func terminalDispatchStatus(result *DSLRuntimeResult) (string, bool) {
	if result == nil || len(result.Statements) == 0 {
		return "", false
	}
	last := result.Statements[len(result.Statements)-1]
	if last.Type != dslStatementTypeDispatch {
		return "", false
	}
	switch last.Status {
	case StatusDelegated, StatusRejected:
		return last.Status, true
	default:
		return "", false
	}
}

func (r *DSLRunner) finalizeDispatchTerminal(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *workflowdomain.Workflow, result *DSLRuntimeResult, toolCalls json.RawMessage, status string) (*Run, error) {
	return r.updateRunWithPayload(ctx, rc, workspaceID, runID, status, buildDSLRunOutputPayload(workflow, result), toolCalls, true)
}

func (r *DSLRunner) finalizeResumeDispatchTerminal(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, result *DSLRuntimeResult, status string) (*Run, error) {
	return r.updateRunWithPayload(ctx, rc, workspaceID, input.RunID, status, buildDSLResumePayload(workflow, input.ResumeStepIndex, result), mergeRunToolCalls(existing.ToolCalls, json.RawMessage(emptyJSONArray)), true)
}

func (r *DSLRunner) updateRunWithPayload(ctx context.Context, rc *RunContext, workspaceID, runID, status string, payload any, toolCalls json.RawMessage, completed bool) (*Run, error) {
	output, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRun(ctx, workspaceID, runID, emptyTracesUpdate(status, output, toolCalls, completed))
}

func mergeDSLContexts(trigger json.RawMessage, inputs json.RawMessage) map[string]any {
	ctx := make(map[string]any)
	mergeRawObjectInto(ctx, trigger)
	mergeRawObjectInto(ctx, inputs)
	return ctx
}

func mergeRawObjectInto(dst map[string]any, raw json.RawMessage) {
	if len(raw) == 0 || !json.Valid(raw) {
		return
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return
	}
	for k, v := range decoded {
		dst[k] = v
	}
}

func mergeRunToolCalls(existing, next json.RawMessage) json.RawMessage {
	left := decodeToolCallArray(existing)
	right := decodeToolCallArray(next)
	if len(left) == 0 && len(right) == 0 {
		return json.RawMessage(emptyJSONArray)
	}
	merged := append(append([]ToolCall{}, left...), right...)
	raw, err := json.Marshal(merged)
	if err != nil {
		return json.RawMessage(emptyJSONArray)
	}
	return raw
}

func decodeToolCallArray(raw json.RawMessage) []ToolCall {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var calls []ToolCall
	if err := json.Unmarshal(raw, &calls); err != nil {
		return nil
	}
	return calls
}

func (r *DSLRunner) preflightDelegate(ctx context.Context, rc *RunContext, workflow *workflowdomain.Workflow, carta *CartaSummary, input TriggerAgentInput, run *Run, evalCtx map[string]any) (*Run, error) {
	if !delegatePolicyApplies(run, workflow, carta) {
		return nil, nil
	}

	decision, err := NewDelegateEvaluator().EvaluateDelegate(carta.Delegates, evalCtx)
	if err != nil {
		return nil, err
	}
	if !delegateDecisionMatched(decision) {
		return nil, nil
	}

	return r.applyDelegateDecision(ctx, rc, input, run, decision.Delegate.Reason, evalCtx)
}

func delegatePolicyApplies(run *Run, workflow *workflowdomain.Workflow, carta *CartaSummary) bool {
	return run != nil && workflow != nil && carta != nil && len(carta.Delegates) > 0
}

func delegateDecisionMatched(decision *DelegateDecision) bool {
	return decision != nil && decision.Matched && decision.Delegate != nil
}

func (r *DSLRunner) applyDelegateDecision(ctx context.Context, rc *RunContext, input TriggerAgentInput, run *Run, reason string, evalCtx map[string]any) (*Run, error) {
	reason = strings.TrimSpace(reason)
	output, marshalErr := buildDelegatePayload(reason)
	if marshalErr != nil {
		return nil, marshalErr
	}

	updated, err := rc.Orchestrator.UpdateAgentRun(ctx, input.WorkspaceID, run.ID, RunUpdates{
		Status:           StatusDelegated,
		Output:           output,
		AbstentionReason: stringPtr(reason),
		Completed:        true,
	})
	if err != nil {
		return nil, err
	}

	initiateDelegateHandoff(ctx, rc, input, updated.ID, reason, evalCtx)
	return updated, nil
}

func buildDelegatePayload(reason string) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"status": "delegated",
		"reason": reason,
	})
}

func initiateDelegateHandoff(ctx context.Context, rc *RunContext, input TriggerAgentInput, runID, reason string, evalCtx map[string]any) {
	if caseID := extractCaseID(evalCtx); caseID != "" && rc.DB != nil {
		handoffSvc := NewHandoffService(rc.DB, crm.NewCaseService(rc.DB), rc.EventBus)
		_, _ = handoffSvc.InitiateHandoff(ctx, input.WorkspaceID, runID, caseID, reason)
	}
}

func (r *DSLRunner) preflightGrounds(ctx context.Context, rc *RunContext, carta *CartaSummary, input TriggerAgentInput, run *Run) (*Run, error) {
	if !groundsPolicyApplies(run, rc, carta) {
		return nil, nil
	}

	result, err := rc.GroundsValidator.Validate(ctx, carta.Grounds, input)
	if err != nil {
		return nil, err
	}
	if result == nil || result.Met {
		return nil, nil
	}

	return r.applyGroundsAbstention(ctx, rc, input, run, result)
}

func groundsPolicyApplies(run *Run, rc *RunContext, carta *CartaSummary) bool {
	return run != nil && rc != nil && rc.GroundsValidator != nil && carta != nil && carta.Grounds != nil
}

func (r *DSLRunner) applyGroundsAbstention(ctx context.Context, rc *RunContext, input TriggerAgentInput, run *Run, result *GroundsResult) (*Run, error) {
	output, marshalErr := buildGroundsAbstainPayload(result)
	if marshalErr != nil {
		return nil, marshalErr
	}

	updated, err := rc.Orchestrator.UpdateAgentRun(ctx, input.WorkspaceID, run.ID, RunUpdates{
		Status:               StatusAbstained,
		Output:               output,
		AbstentionReason:     stringPtr(result.Reason),
		RetrievalQueries:     marshalStringArray(result.Query),
		RetrievedEvidenceIDs: marshalEvidenceIDs(result.EvidencePack),
		Completed:            true,
	})
	if err != nil {
		return nil, err
	}

	initiateGroundsHandoff(ctx, rc, input, updated.ID, result.Reason)
	return updated, nil
}

func buildGroundsAbstainPayload(result *GroundsResult) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"status": "abstained",
		"reason": result.Reason,
		"query":  result.Query,
	})
}

func initiateGroundsHandoff(ctx context.Context, rc *RunContext, input TriggerAgentInput, runID, reason string) {
	evalCtx := mergeDSLContexts(input.TriggerContext, input.Inputs)
	if caseID := extractCaseID(evalCtx); caseID != "" && rc.DB != nil {
		handoffSvc := NewHandoffService(rc.DB, crm.NewCaseService(rc.DB), rc.EventBus)
		_, _ = handoffSvc.InitiateHandoff(ctx, input.WorkspaceID, runID, caseID, reason)
	}
}

func parseCartaWorkflowSpec(workflow *workflowdomain.Workflow) *CartaSummary {
	if workflow == nil || workflow.SpecSource == nil {
		return nil
	}
	carta, err := ParseCarta(*workflow.SpecSource)
	if err != nil {
		return nil
	}
	return carta
}

func extractCaseID(evalCtx map[string]any) string {
	caseMap, ok := evalCtx["case"].(map[string]any)
	if !ok {
		return ""
	}
	if id, ok := caseMap["id"].(string); ok {
		return strings.TrimSpace(id)
	}
	return ""
}


func marshalStringArray(values ...string) json.RawMessage {
	items := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	if len(items) == 0 {
		return json.RawMessage(emptyJSONArray)
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return json.RawMessage(emptyJSONArray)
	}
	return raw
}

func marshalEvidenceIDs(pack *knowledge.EvidencePack) json.RawMessage {
	if pack == nil || len(pack.Sources) == 0 {
		return json.RawMessage(emptyJSONArray)
	}
	ids := make([]string, 0, len(pack.Sources))
	for _, source := range pack.Sources {
		if trimmed := strings.TrimSpace(source.ID); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	return marshalStringArray(ids...)
}

func buildDSLRunOutputPayload(workflow *workflowdomain.Workflow, result *DSLRuntimeResult) DSLRunOutput {
	return DSLRunOutput{
		WorkflowID:      workflow.ID,
		WorkflowName:    workflow.Name,
		WorkflowVersion: workflow.Version,
		Statements:      runtimeStatements(result),
	}
}

func buildDSLWorkflowPayload(workflow *workflowdomain.Workflow, result *DSLRuntimeResult) map[string]any {
	return map[string]any{
		"workflow_id":      workflow.ID,
		"workflow_name":    workflow.Name,
		"workflow_version": workflow.Version,
		"statements":       runtimeStatements(result),
	}
}

func buildDSLResumePayload(workflow *workflowdomain.Workflow, resumeStepIndex int, result *DSLRuntimeResult) map[string]any {
	payload := buildDSLWorkflowPayload(workflow, result)
	payload["resume_step_index"] = resumeStepIndex
	payload["resumed"] = true
	return payload
}

func buildDSLFailurePayload(workflow *workflowdomain.Workflow, result *DSLRuntimeResult, execErr error) map[string]any {
	payload := buildDSLWorkflowPayload(workflow, result)
	payload["error"] = execErr.Error()
	return payload
}

func buildDSLResumeFailurePayload(input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, execErr error) map[string]any {
	payload := map[string]any{
		"workflow_id":       input.WorkflowID,
		"run_id":            input.RunID,
		"resume_step_index": input.ResumeStepIndex,
		"error":             execErr.Error(),
		"resumed":           true,
	}
	if workflow != nil {
		payload["workflow_name"] = workflow.Name
		payload["workflow_version"] = workflow.Version
	}
	return payload
}

func buildPendingPayload(payload map[string]any, executor *dslRuntimeExecutor) map[string]any {
	for key, value := range executor.PendingOutput() {
		payload[key] = value
	}
	if _, ok := payload["action"]; !ok {
		payload["action"] = pendingApprovalAction
	}
	if approval := executor.PendingApproval(); approval != nil {
		payload["approval_id"] = approval.ApprovalID
	}
	return payload
}

func runtimeStatements(result *DSLRuntimeResult) []DSLStatementResult {
	if result == nil {
		return nil
	}
	return result.Statements
}
