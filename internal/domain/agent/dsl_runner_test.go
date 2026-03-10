package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

type stubPendingNestedRunner struct{}

func (s stubPendingNestedRunner) Run(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*Run, error) {
	run, err := rc.Orchestrator.TriggerAgent(ctx, input)
	if err != nil {
		return nil, err
	}
	output, _ := json.Marshal(map[string]any{
		"action":      "pending_approval",
		"approval_id": "apr_nested_1",
	})
	return rc.Orchestrator.UpdateAgentRun(ctx, input.WorkspaceID, run.ID, emptyTracesUpdate(StatusAccepted, output, json.RawMessage(emptyJSONArray), false))
}

func TestDSLRunnerRunLoadsActiveWorkflowAndSucceeds(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_1', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_1', 'ws_dsl', 'agent_dsl_1', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
SET case.status = "resolved"
NOTIFY contact WITH "done"', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	registry := NewRunnerRegistry()
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewDSLRunner(db)
	toolRegistry := setupDSLToolRegistry(t, db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		ToolRegistry: toolRegistry,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_1",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusSuccess {
		t.Fatalf("status = %s, want %s", run.Status, StatusSuccess)
	}

	var output DSLRunOutput
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.WorkflowID != "wf_dsl_1" {
		t.Fatalf("workflow_id = %s, want wf_dsl_1", output.WorkflowID)
	}
	if len(output.Statements) != 2 {
		t.Fatalf("len(statements) = %d, want 2", len(output.Statements))
	}
	if output.Statements[0].Status != StepStatusSuccess || output.Statements[1].Status != StepStatusSuccess {
		t.Fatalf("unexpected statement statuses = %#v", output.Statements)
	}

	var toolCalls []ToolCall
	if err := json.Unmarshal(run.ToolCalls, &toolCalls); err != nil {
		t.Fatalf("unmarshal tool calls: %v", err)
	}
	if len(toolCalls) != 2 {
		t.Fatalf("len(toolCalls) = %d, want 2", len(toolCalls))
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps() error = %v", err)
	}
	dslSteps := filterRunStepsByType(steps, StepTypeDSLStatement)
	if len(dslSteps) != 2 {
		t.Fatalf("len(dslSteps) = %d, want 2", len(dslSteps))
	}
	if dslSteps[0].Status != StepStatusSuccess || dslSteps[1].Status != StepStatusSuccess {
		t.Fatalf("unexpected dsl step statuses = %#v", dslSteps)
	}
}

func TestDSLRunnerRunRequiresOrchestrator(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	runner := NewDSLRunner(db)

	_, err := runner.Run(context.Background(), &RunContext{}, TriggerAgentInput{
		AgentID:     "agent_dsl_1",
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeManual,
	})
	if !errors.Is(err, ErrDSLRunnerMissingOrchestrator) {
		t.Fatalf("expected ErrDSLRunnerMissingOrchestrator, got %v", err)
	}
}

func TestDSLRunnerRunFailsWithoutActiveWorkflow(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_missing', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	registry := NewRunnerRegistry()
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewDSLRunner(db)

	_, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:     "agent_dsl_missing",
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeManual,
	})
	if !errors.Is(err, ErrDSLWorkflowNotFound) {
		t.Fatalf("expected ErrDSLWorkflowNotFound, got %v", err)
	}
}

func TestDSLRunnerRunMarksFailedWhenExecutorFails(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_fail', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_fail', 'ws_dsl', 'agent_dsl_fail', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
SET case.status = "resolved"', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	registry := NewRunnerRegistry()
	orch := NewOrchestratorWithRegistry(db, registry)
	executor := &stubRuntimeExecutor{err: errors.New("boom")}
	runner := NewDSLRunnerWithDependencies(workflowdomain.NewService(db), NewDSLRuntime(), executor)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_fail",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
	})
	if err != nil {
		t.Fatalf("Run() transport error = %v", err)
	}
	if run.Status != StatusFailed {
		t.Fatalf("status = %s, want %s", run.Status, StatusFailed)
	}
}

func TestDSLRunnerRunTracesSkippedIfStatement(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_if', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_if', 'ws_dsl', 'agent_dsl_if', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
IF case.priority == "high":
  NOTIFY contact WITH "done"', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	registry := NewRunnerRegistry()
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewDSLRunner(db)
	toolRegistry := setupDSLToolRegistry(t, db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		ToolRegistry: toolRegistry,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_if",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1","priority":"low"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusSuccess {
		t.Fatalf("status = %s, want %s", run.Status, StatusSuccess)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps() error = %v", err)
	}
	dslSteps := filterRunStepsByType(steps, StepTypeDSLStatement)
	if len(dslSteps) != 1 {
		t.Fatalf("len(dslSteps) = %d, want 1", len(dslSteps))
	}
	if dslSteps[0].Status != StepStatusSkipped {
		t.Fatalf("status = %s, want %s", dslSteps[0].Status, StepStatusSkipped)
	}
}

func TestDSLRunnerRunExecutesIfBodyInTracedMode(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_if_true', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_if_true', 'ws_dsl', 'agent_dsl_if_true', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
IF case.priority == "high":
  SET case.status = "resolved"', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
	runner := NewDSLRunner(db)
	toolRegistry := setupDSLToolRegistry(t, db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		ToolRegistry: toolRegistry,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_if_true",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1","priority":"high"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusSuccess {
		t.Fatalf("status = %s, want %s", run.Status, StatusSuccess)
	}

	var output DSLRunOutput
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Statements) != 2 {
		t.Fatalf("len(statements) = %d, want 2", len(output.Statements))
	}
	if output.Statements[0].Type != "IF" || output.Statements[1].Type != "SET" {
		t.Fatalf("unexpected statement order = %#v", output.Statements)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps() error = %v", err)
	}
	dslSteps := filterRunStepsByType(steps, StepTypeDSLStatement)
	if len(dslSteps) != 2 {
		t.Fatalf("len(dslSteps) = %d, want 2", len(dslSteps))
	}
	if dslSteps[0].Status != StepStatusSuccess || dslSteps[1].Status != StepStatusSuccess {
		t.Fatalf("unexpected traced statuses = %#v", dslSteps)
	}
}

func TestDSLRunnerRunTracesPendingApprovalInStatementOutput(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
	VALUES ('user_dsl', 'ws_dsl', 'dsl@example.com', 'DSL User', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_pending', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('nested_pending_agent', 'ws_dsl', 'pending_agent', 'nested_pending', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_pending', 'ws_dsl', 'agent_dsl_pending', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
AGENT pending_agent WITH {"case_id":"case-1"}', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	registry := NewRunnerRegistry()
	if err := registry.Register("nested_pending", stubPendingNestedRunner{}); err != nil {
		t.Fatalf("registry.Register() error = %v", err)
	}
	toolRegistry := setupDSLToolRegistry(t, db)
	approvalService := policy.NewApprovalService(db, nil)
	orch := NewOrchestratorWithRegistry(db, registry)
	triggeredBy := "user_dsl"
	runner := NewDSLRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator:    orch,
		RunnerRegistry:  registry,
		ToolRegistry:    toolRegistry,
		ApprovalService: approvalService,
		DB:              db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_pending",
		WorkspaceID:    "ws_dsl",
		TriggeredBy:    &triggeredBy,
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusAccepted {
		t.Fatalf("status = %s, want %s", run.Status, StatusAccepted)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps() error = %v", err)
	}
	dslSteps := filterRunStepsByType(steps, StepTypeDSLStatement)
	if len(dslSteps) != 1 {
		t.Fatalf("len(dslSteps) = %d, want 1", len(dslSteps))
	}
	if dslSteps[0].Status != StepStatusRunning {
		t.Fatalf("status = %s, want %s", dslSteps[0].Status, StepStatusRunning)
	}
	var traceOutput map[string]any
	if err := json.Unmarshal(dslSteps[0].Output, &traceOutput); err != nil {
		t.Fatalf("unmarshal trace output: %v", err)
	}
	output, ok := traceOutput["output"].(map[string]any)
	if !ok || output["action"] != "pending_approval" {
		t.Fatalf("expected pending_approval marker in trace output, got %#v", traceOutput)
	}
}

func TestDSLRunnerInvalidateCacheReloadsUpdatedDSL(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_cache', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_cache', 'ws_dsl', 'agent_dsl_cache', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
SET case.status = "resolved"', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	runner := NewDSLRunner(db)
	repo := workflowdomain.NewRepository(db)
	firstProgram, err := runner.loadProgram(&workflowdomain.Workflow{
		ID:        "wf_dsl_cache",
		Name:      "resolve_support_case",
		DSLSource: "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
	})
	if err != nil {
		t.Fatalf("loadProgram() error = %v", err)
	}
	if _, err := repo.Update(context.Background(), "ws_dsl", "wf_dsl_cache", workflowdomain.UpdateInput{
		DSLSource: "WORKFLOW resolve_support_case\nON case.created\nNOTIFY contact WITH \"updated\"",
		Status:    workflowdomain.StatusActive,
	}); err != nil {
		t.Fatalf("repo.Update() error = %v", err)
	}
	runner.InvalidateCache("wf_dsl_cache")

	item, err := repo.GetByID(context.Background(), "ws_dsl", "wf_dsl_cache")
	if err != nil {
		t.Fatalf("repo.GetByID() error = %v", err)
	}
	secondProgram, err := runner.loadProgram(item)
	if err != nil {
		t.Fatalf("loadProgram(updated) error = %v", err)
	}
	if _, ok := firstProgram.Workflow.Body[0].(*SetStatement); !ok {
		t.Fatalf("expected first statement to be SET")
	}
	if _, ok := secondProgram.Workflow.Body[0].(*NotifyStatement); !ok {
		t.Fatalf("expected invalidated cache to reload NOTIFY statement, got %#v", secondProgram.Workflow.Body[0])
	}
}

func setupDSLRunnerDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := isqlite.MigrateUp(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	mustExecDSLRunner(t, db, `INSERT INTO workspace (id, name, slug, created_at, updated_at)
	VALUES ('ws_dsl', 'DSL Workspace', 'dsl-workspace', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func mustExecDSLRunner(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("exec failed: %v", err)
	}
}
