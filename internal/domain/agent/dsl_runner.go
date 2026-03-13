package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

var (
	ErrDSLRunnerMissingOrchestrator = errors.New("dsl runner requires orchestrator")
	ErrDSLWorkflowNotFound          = errors.New("dsl workflow not found")
	ErrDSLResumeInvalidInput        = errors.New("dsl resume input is invalid")
	ErrDSLWorkflowNotActive         = errors.New("dsl workflow is not active")
)

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
	workflow, err := r.loadActiveWorkflow(ctx, input.WorkspaceID, input.AgentID)
	if err != nil {
		return nil, err
	}
	program, err := r.loadProgram(workflow)
	if err != nil {
		return nil, err
	}
	run, err := rc.Orchestrator.TriggerAgent(ctx, input)
	if err != nil {
		return nil, err
	}
	run, err = rc.Orchestrator.UpdateAgentRunStatus(ctx, input.WorkspaceID, run.ID, StatusAccepted)
	if err != nil {
		return nil, err
	}
	execCtx := rc.WithCall(input.AgentID)
	evalCtx := mergeDSLContexts(input.TriggerContext, input.Inputs)
	baseExecutor, defaultExecutor := r.buildBaseExecutor(execCtx, input, evalCtx, workflow.ID, run.ID)
	executor := wrapWithTrace(input.WorkspaceID, run.ID, execCtx, r.runtime, baseExecutor, defaultExecutor)
	result, execErr := r.runtime.ExecuteProgram(ctx, program, evalCtx, executor)
	return r.finalizeRun(ctx, rc, input.WorkspaceID, run.ID, workflow, result, defaultExecutor, execErr)
}

func (r *DSLRunner) Resume(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload) (*Run, error) {
	run, workflow, err := r.prepareResume(ctx, rc, workspaceID, input)
	if err != nil {
		if run != nil {
			return r.failResumeAndReturnErr(ctx, rc, workspaceID, input, workflow, run, err)
		}
		return nil, err
	}

	program, err := r.loadProgram(workflow)
	if err != nil {
		return r.failResumeAndReturnErr(ctx, rc, workspaceID, input, workflow, run, err)
	}
	if _, err = rc.Orchestrator.UpdateAgentRunStatus(ctx, workspaceID, input.RunID, StatusAccepted); err != nil {
		return r.failResumeAndReturnErr(ctx, rc, workspaceID, input, workflow, run, err)
	}

	execCtx := rc.WithCall(run.DefinitionID)
	triggerInput := TriggerAgentInput{
		AgentID:        run.DefinitionID,
		WorkspaceID:    workspaceID,
		TriggeredBy:    run.TriggeredByUserID,
		TriggerType:    run.TriggerType,
		TriggerContext: run.TriggerContext,
		Inputs:         run.Inputs,
	}
	evalCtx := mergeDSLContexts(run.TriggerContext, run.Inputs)
	baseExecutor, defaultExecutor := r.buildBaseExecutor(execCtx, triggerInput, evalCtx, workflow.ID, input.RunID)
	executor := wrapWithTrace(workspaceID, input.RunID, execCtx, r.runtime, baseExecutor, defaultExecutor)
	result, execErr := r.runtime.ExecuteProgramFromIndex(ctx, program, input.ResumeStepIndex, evalCtx, executor)
	return r.finalizeResumedRun(ctx, rc, workspaceID, input.RunID, input, workflow, run, result, defaultExecutor, execErr)
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
	if strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(input.WorkflowID) == "" || strings.TrimSpace(input.RunID) == "" {
		return ErrDSLResumeInvalidInput
	}
	if input.ResumeStepIndex < 0 {
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
	output, err := json.Marshal(DSLRunOutput{
		WorkflowID:      workflow.ID,
		WorkflowName:    workflow.Name,
		WorkflowVersion: workflow.Version,
		Statements:      result.Statements,
	})
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRun(ctx, workspaceID, runID, emptyTracesUpdate(StatusSuccess, output, toolCalls, true))
}

func (r *DSLRunner) finalizeResumeSuccess(ctx context.Context, rc *RunContext, workspaceID, runID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, result *DSLRuntimeResult, toolCalls json.RawMessage) (*Run, error) {
	output, err := json.Marshal(map[string]any{
		"workflow_id":       workflow.ID,
		"workflow_name":     workflow.Name,
		"workflow_version":  workflow.Version,
		"resume_step_index": input.ResumeStepIndex,
		"statements":        result.Statements,
		"resumed":           true,
	})
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRun(ctx, workspaceID, runID, emptyTracesUpdate(StatusSuccess, output, mergeRunToolCalls(existing.ToolCalls, toolCalls), true))
}

func (r *DSLRunner) finalizeFailure(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *workflowdomain.Workflow, result *DSLRuntimeResult, toolCalls json.RawMessage, execErr error) (*Run, error) {
	payload := map[string]any{
		"workflow_id":      workflow.ID,
		"workflow_name":    workflow.Name,
		"workflow_version": workflow.Version,
		"error":            execErr.Error(),
	}
	if result != nil {
		payload["statements"] = result.Statements
	}
	output, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRun(ctx, workspaceID, runID, emptyTracesUpdate(StatusFailed, output, toolCalls, true))
}

func (r *DSLRunner) finalizeResumeFailure(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, execErr error) (*Run, error) {
	if rc == nil || rc.Orchestrator == nil || existing == nil {
		return nil, execErr
	}
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
	output, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRun(ctx, workspaceID, input.RunID, emptyTracesUpdate(StatusFailed, output, existing.ToolCalls, true))
}

func (r *DSLRunner) failResumeAndReturnErr(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, resumeErr error) (*Run, error) {
	run, err := r.finalizeResumeFailure(ctx, rc, workspaceID, input, workflow, existing, resumeErr)
	if err != nil {
		return nil, err
	}
	return run, resumeErr
}

func (r *DSLRunner) finalizePending(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *workflowdomain.Workflow, result *DSLRuntimeResult, executor *dslRuntimeExecutor) (*Run, error) {
	payload := map[string]any{
		"workflow_id":      workflow.ID,
		"workflow_name":    workflow.Name,
		"workflow_version": workflow.Version,
		"statements":       result.Statements,
	}
	for key, value := range executor.PendingOutput() {
		payload[key] = value
	}
	if _, ok := payload["action"]; !ok {
		payload["action"] = pendingApprovalAction
	}
	if executor.PendingApproval() != nil {
		payload["approval_id"] = executor.PendingApproval().ApprovalID
	}
	output, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRun(ctx, workspaceID, runID, emptyTracesUpdate(StatusAccepted, output, executor.ToolCallsJSON(), false))
}

func (r *DSLRunner) finalizeResumePending(ctx context.Context, rc *RunContext, workspaceID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, result *DSLRuntimeResult, executor *dslRuntimeExecutor) (*Run, error) {
	payload := map[string]any{
		"workflow_id":       workflow.ID,
		"workflow_name":     workflow.Name,
		"workflow_version":  workflow.Version,
		"resume_step_index": input.ResumeStepIndex,
		"statements":        result.Statements,
		"resumed":           true,
	}
	for key, value := range executor.PendingOutput() {
		payload[key] = value
	}
	if _, ok := payload["action"]; !ok {
		payload["action"] = pendingApprovalAction
	}
	if executor.PendingApproval() != nil {
		payload["approval_id"] = executor.PendingApproval().ApprovalID
	}
	output, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return rc.Orchestrator.UpdateAgentRun(ctx, workspaceID, input.RunID, emptyTracesUpdate(StatusAccepted, output, mergeRunToolCalls(existing.ToolCalls, executor.ToolCallsJSON()), false))
}

func (r *DSLRunner) finalizeResumedRun(ctx context.Context, rc *RunContext, workspaceID, runID string, input schedulerdomain.WorkflowResumePayload, workflow *workflowdomain.Workflow, existing *Run, result *DSLRuntimeResult, defaultExecutor *dslRuntimeExecutor, execErr error) (*Run, error) {
	toolCalls := dslToolCallsJSON(defaultExecutor)
	if execErr != nil {
		return r.finalizeResumeFailure(ctx, rc, workspaceID, input, workflow, existing, execErr)
	}
	if defaultExecutor != nil && defaultExecutor.IsPending() {
		return r.finalizeResumePending(ctx, rc, workspaceID, input, workflow, existing, result, defaultExecutor)
	}
	return r.finalizeResumeSuccess(ctx, rc, workspaceID, runID, input, workflow, existing, result, toolCalls)
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
