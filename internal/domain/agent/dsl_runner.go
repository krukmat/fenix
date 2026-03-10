package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

var (
	ErrDSLRunnerMissingOrchestrator = errors.New("dsl runner requires orchestrator")
	ErrDSLWorkflowNotFound          = errors.New("dsl workflow not found")
)

type DSLRunner struct {
	workflowService workflowResolver
	runtime         *DSLRuntime
	executor        RuntimeOperationExecutor

	cacheMu  sync.RWMutex
	astCache map[string]*Program
}

type workflowResolver interface {
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

	execCtx := rc.WithCall(input.AgentID)
	evalCtx := mergeDSLContexts(input.TriggerContext, input.Inputs)
	executor := r.executor
	var defaultExecutor *dslRuntimeExecutor
	useDefaultExecutor := false
	if executor == nil {
		defaultExecutor = newDSLRuntimeExecutor(execCtx, input, evalCtx)
		executor = defaultExecutor
		useDefaultExecutor = true
	}

	run, err := rc.Orchestrator.TriggerAgent(ctx, input)
	if err != nil {
		return nil, err
	}
	run, err = rc.Orchestrator.UpdateAgentRunStatus(ctx, input.WorkspaceID, run.ID, StatusAccepted)
	if err != nil {
		return nil, err
	}

	if useDefaultExecutor {
		executor = newTracedDSLExecutor(input.WorkspaceID, run.ID, execCtx, r.runtime, defaultExecutor)
	}

	result, execErr := r.runtime.ExecuteProgram(ctx, program, evalCtx, executor)
	if execErr != nil {
		toolCalls := json.RawMessage(emptyJSONArray)
		if useDefaultExecutor {
			toolCalls = defaultExecutor.ToolCallsJSON()
		}
		return r.finalizeFailure(ctx, rc, input.WorkspaceID, run.ID, workflow, result, toolCalls, execErr)
	}
	if useDefaultExecutor && defaultExecutor.IsPending() {
		return r.finalizePending(ctx, rc, input.WorkspaceID, run.ID, workflow, result, defaultExecutor)
	}
	toolCalls := json.RawMessage(emptyJSONArray)
	if useDefaultExecutor {
		toolCalls = defaultExecutor.ToolCallsJSON()
	}
	return r.finalizeSuccess(ctx, rc, input.WorkspaceID, run.ID, workflow, result, toolCalls)
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

func (r *DSLRunner) finalizePending(ctx context.Context, rc *RunContext, workspaceID, runID string, workflow *workflowdomain.Workflow, result *DSLRuntimeResult, executor *dslRuntimeExecutor) (*Run, error) {
	payload := map[string]any{
		"workflow_id":      workflow.ID,
		"workflow_name":    workflow.Name,
		"workflow_version": workflow.Version,
		"action":           "pending_approval",
		"statements":       result.Statements,
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
