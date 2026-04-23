package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

type stubPendingNestedRunner struct{}

type stubGroundsEvidenceBuilder struct {
	pack *knowledge.EvidencePack
	err  error
}

func (s stubGroundsEvidenceBuilder) BuildEvidencePack(_ context.Context, _ knowledge.BuildEvidencePackInput) (*knowledge.EvidencePack, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.pack, nil
}

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

func TestDSLRunnerRunDelegatesBeforeDSLExecutionWhenCartaDelegateMatches(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
	VALUES ('user_delegate', 'ws_dsl', 'delegate@example.com', 'Delegate Owner', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at)
	VALUES ('case-1', 'ws_dsl', 'user_delegate', 'Delegate Case', 'high', 'open', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_delegate', 'ws_dsl', 'dsl delegate', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, spec_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_delegate', 'ws_dsl', 'agent_dsl_delegate', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
SET case.status = "resolved"', 'CARTA resolve_support_case
AGENT search_knowledge
  DELEGATE TO HUMAN
    when: case.tier == "enterprise"
    reason: "Enterprise review required"
    package: [evidence_ids, case_summary]', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
	runner := NewDSLRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		DB:           db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_delegate",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1","tier":"enterprise"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusDelegated {
		t.Fatalf("status = %s, want %s", run.Status, StatusDelegated)
	}
	if run.AbstentionReason == nil || *run.AbstentionReason != "Enterprise review required" {
		t.Fatalf("AbstentionReason = %v, want Enterprise review required", run.AbstentionReason)
	}
}

func TestDSLRunnerRunDelegatesBeforeGroundsWhenBothCartaPreflightsExist(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
	VALUES ('user_delegate_order', 'ws_dsl', 'delegate-order@example.com', 'Delegate Order Owner', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at)
	VALUES ('case-order', 'ws_dsl', 'user_delegate_order', 'Delegate Order Case', 'high', 'open', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_preflight_order', 'ws_dsl', 'dsl preflight order', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, spec_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_preflight_order', 'ws_dsl', 'agent_dsl_preflight_order', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
SET case.status = "resolved"', 'CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  DELEGATE TO HUMAN
    when: case.tier == "enterprise"
    reason: "Enterprise review required"
    package: [evidence_ids, case_summary]', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
	runner := NewDSLRunner(db)
	groundsValidator := NewGroundsValidator(stubGroundsEvidenceBuilder{
		err: errors.New("grounds should not run before matching delegate"),
	})

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator:     orch,
		GroundsValidator: groundsValidator,
		DB:               db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_preflight_order",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-order","tier":"enterprise"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusDelegated {
		t.Fatalf("status = %s, want %s", run.Status, StatusDelegated)
	}
	if run.AbstentionReason == nil || *run.AbstentionReason != "Enterprise review required" {
		t.Fatalf("AbstentionReason = %v, want Enterprise review required", run.AbstentionReason)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps() error = %v", err)
	}
	if dslSteps := filterRunStepsByType(steps, StepTypeDSLStatement); len(dslSteps) != 0 {
		t.Fatalf("DSL steps = %#v, want none before delegated preflight", dslSteps)
	}
}

func TestDSLRunnerRunAbstainsBeforeDSLExecutionWhenGroundsFail(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_grounds', 'ws_dsl', 'dsl grounds', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, spec_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_grounds', 'ws_dsl', 'agent_dsl_grounds', 'resolve_support_case', 'WORKFLOW resolve_support_case
ON case.created
SET case.status = "resolved"', 'CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
    min_confidence: medium', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
	runner := NewDSLRunner(db)
	groundsValidator := NewGroundsValidator(stubGroundsEvidenceBuilder{
		pack: &knowledge.EvidencePack{
			Sources:    []knowledge.Evidence{{ID: "ev-1", CreatedAt: time.Now()}},
			Confidence: knowledge.ConfidenceLow,
		},
	})

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator:     orch,
		GroundsValidator: groundsValidator,
		DB:               db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_grounds",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1","priority":"high","summary":"Cannot resolve billing issue"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusAbstained {
		t.Fatalf("status = %s, want %s", run.Status, StatusAbstained)
	}
	if run.AbstentionReason == nil || *run.AbstentionReason == "" {
		t.Fatalf("AbstentionReason = %v, want non-empty", run.AbstentionReason)
	}
	if len(run.RetrievalQueries) == 0 {
		t.Fatal("RetrievalQueries = empty, want grounds query")
	}
	var queries []string
	if err := json.Unmarshal(run.RetrievalQueries, &queries); err != nil {
		t.Fatalf("unmarshal RetrievalQueries: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("RetrievalQueries = %#v, want one query", queries)
	}
	for _, want := range []string{"case-1", "high", "Cannot resolve billing issue"} {
		if !strings.Contains(queries[0], want) {
			t.Fatalf("RetrievalQueries = %#v, want value %q", queries, want)
		}
	}
	if string(run.RetrievedEvidenceIDs) != `["ev-1"]` {
		t.Fatalf("RetrievedEvidenceIDs = %s, want ev-1", run.RetrievedEvidenceIDs)
	}
	var output map[string]any
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output["status"] != "abstained" || output["reason"] == "" || output["query"] == "" {
		t.Fatalf("unexpected abstention output = %#v", output)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps() error = %v", err)
	}
	if dslSteps := filterRunStepsByType(steps, StepTypeDSLStatement); len(dslSteps) != 0 {
		t.Fatalf("DSL steps = %#v, want none before grounds abstention", dslSteps)
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

func TestDSLRunnerRunSchedulesWaitAndLeavesRunAccepted(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_wait', 'ws_dsl', 'dsl support', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_wait', 'ws_dsl', 'agent_dsl_wait', 'wait_support_case', 'WORKFLOW wait_support_case
ON case.created
WAIT 0
NOTIFY contact WITH "done"', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	repo := schedulerdomain.NewRepository(db)
	scheduler := schedulerdomain.NewService(repo)
	orch := NewOrchestratorWithRegistry(db, NewRunnerRegistry())
	runner := NewDSLRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator: orch,
		Scheduler:    scheduler,
		ToolRegistry: setupDSLToolRegistry(t, db),
		DB:           db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_wait",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusAccepted {
		t.Fatalf("status = %s, want %s", run.Status, StatusAccepted)
	}

	var output map[string]any
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output["action"] != pendingWaitAction {
		t.Fatalf("action = %v, want %s", output["action"], pendingWaitAction)
	}

	jobs, err := repo.ListDue(context.Background(), time.Now().UTC().Add(time.Second), 10)
	if err != nil {
		t.Fatalf("ListDue() error = %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("len(jobs) = %d, want 1", len(jobs))
	}
	payload, err := schedulerdomain.DecodeWorkflowResumePayload(jobs[0].Payload)
	if err != nil {
		t.Fatalf("DecodeWorkflowResumePayload() error = %v", err)
	}
	if payload.ResumeStepIndex != 1 {
		t.Fatalf("resume_step_index = %d, want 1", payload.ResumeStepIndex)
	}
}

func TestDSLRunnerRunDispatchesInternallyAndLeavesRunDelegated(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_dispatch', 'ws_dsl', 'dsl dispatch', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('dispatch_target_id', 'ws_dsl', 'dispatch_target', 'nested', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_dispatch', 'ws_dsl', 'agent_dsl_dispatch', 'delegate_case', 'WORKFLOW delegate_case
ON case.created
DISPATCH TO dispatch_target WITH {"case_id":"case-1"}', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	registry := NewRunnerRegistry()
	if err := registry.Register("nested", stubNestedRunner{}); err != nil {
		t.Fatalf("registry.Register() error = %v", err)
	}
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewDSLRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		DB:             db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_dispatch",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusDelegated {
		t.Fatalf("status = %s, want %s", run.Status, StatusDelegated)
	}

	var output DSLRunOutput
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Statements) != 1 {
		t.Fatalf("len(statements) = %d, want 1", len(output.Statements))
	}
	stmtOutput, ok := output.Statements[0].Output.(map[string]any)
	if !ok || stmtOutput["action"] != pendingDispatchAction || stmtOutput["dispatch_result"] != dispatchResultDelegated {
		t.Fatalf("unexpected output = %#v", output)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps() error = %v", err)
	}
	dslSteps := filterRunStepsByType(steps, StepTypeDSLStatement)
	if len(dslSteps) != 1 {
		t.Fatalf("len(dslSteps) = %d, want 1", len(dslSteps))
	}
	if dslSteps[0].Status != StepStatusSuccess {
		t.Fatalf("step status = %s, want %s", dslSteps[0].Status, StepStatusSuccess)
	}
}

func TestDSLRunnerRunDispatchAcceptedLeavesRunAccepted(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_dispatch_pending', 'ws_dsl', 'dsl dispatch', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('dispatch_target_pending_id', 'ws_dsl', 'dispatch_target_pending', 'nested_pending', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_dispatch_pending', 'ws_dsl', 'agent_dsl_dispatch_pending', 'delegate_case_pending', 'WORKFLOW delegate_case_pending
ON case.created
DISPATCH TO dispatch_target_pending WITH {"case_id":"case-1"}', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	registry := NewRunnerRegistry()
	if err := registry.Register("nested_pending", stubPendingNestedRunner{}); err != nil {
		t.Fatalf("registry.Register() error = %v", err)
	}
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewDSLRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		DB:             db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_dispatch_pending",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusAccepted {
		t.Fatalf("status = %s, want %s", run.Status, StatusAccepted)
	}

	var output map[string]any
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output["dispatch_result"] != dispatchResultAccepted {
		t.Fatalf("unexpected output = %#v", output)
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
		t.Fatalf("step status = %s, want %s", dslSteps[0].Status, StepStatusRunning)
	}
}

func TestDSLRunnerRunDispatchRejectedLeavesRunRejectedWithReason(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_dsl_dispatch_rejected', 'ws_dsl', 'dsl dispatch', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('dispatch_target_rejected_id', 'ws_dsl', 'dispatch_target_rejected', 'nested_rejected', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	mustExecDSLRunner(t, db, `INSERT INTO workflow (id, workspace_id, agent_definition_id, name, dsl_source, version, status, created_at, updated_at)
	VALUES ('wf_dsl_dispatch_rejected', 'ws_dsl', 'agent_dsl_dispatch_rejected', 'delegate_case_rejected', 'WORKFLOW delegate_case_rejected
ON case.created
DISPATCH TO dispatch_target_rejected WITH {"case_id":"case-1"}', 1, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)

	registry := NewRunnerRegistry()
	if err := registry.Register("nested_rejected", stubRejectedRunner{}); err != nil {
		t.Fatalf("registry.Register() error = %v", err)
	}
	orch := NewOrchestratorWithRegistry(db, registry)
	runner := NewDSLRunner(db)

	run, err := runner.Run(context.Background(), &RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		DB:             db,
	}, TriggerAgentInput{
		AgentID:        "agent_dsl_dispatch_rejected",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeEvent,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if run.Status != StatusRejected {
		t.Fatalf("status = %s, want %s", run.Status, StatusRejected)
	}

	var output DSLRunOutput
	if err := json.Unmarshal(run.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Statements) != 1 {
		t.Fatalf("len(statements) = %d, want 1", len(output.Statements))
	}
	stmtOutput, ok := output.Statements[0].Output.(map[string]any)
	if !ok || stmtOutput["dispatch_result"] != dispatchResultRejected || stmtOutput["reason"] == "" {
		t.Fatalf("unexpected output = %#v", output)
	}

	steps, err := orch.ListRunSteps(context.Background(), "ws_dsl", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps() error = %v", err)
	}
	dslSteps := filterRunStepsByType(steps, StepTypeDSLStatement)
	if len(dslSteps) != 1 {
		t.Fatalf("len(dslSteps) = %d, want 1", len(dslSteps))
	}
	if dslSteps[0].Status != StepStatusFailed {
		t.Fatalf("step status = %s, want %s", dslSteps[0].Status, StepStatusFailed)
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

func TestDSLRunnerHelperCoverage(t *testing.T) {
	t.Parallel()

	if got := dslToolCallsJSON(nil); string(got) != emptyJSONArray {
		t.Fatalf("dslToolCallsJSON(nil) = %s", string(got))
	}

	dispatchResult := &DSLRuntimeResult{
		Statements: []DSLStatementResult{{Type: dslStatementTypeDispatch, Status: StatusDelegated}},
	}
	if status, ok := terminalDispatchStatus(dispatchResult); !ok || status != StatusDelegated {
		t.Fatalf("terminalDispatchStatus(dispatch) = %q, %v", status, ok)
	}
	if _, ok := terminalDispatchStatus(&DSLRuntimeResult{
		Statements: []DSLStatementResult{{Type: "SET", Status: StatusSuccess}},
	}); ok {
		t.Fatal("terminalDispatchStatus(non-dispatch) expected false")
	}

	ctx := mergeDSLContexts(json.RawMessage(`{"a":1}`), json.RawMessage(`{"b":2}`))
	if ctx["a"] != float64(1) || ctx["b"] != float64(2) {
		t.Fatalf("mergeDSLContexts() = %#v", ctx)
	}

	dst := map[string]any{"left": true}
	mergeRawObjectInto(dst, json.RawMessage(`{"right":42}`))
	if dst["right"] != float64(42) {
		t.Fatalf("mergeRawObjectInto(valid) = %#v", dst)
	}
	mergeRawObjectInto(dst, json.RawMessage(`not-json`))

	leftCalls, _ := json.Marshal([]ToolCall{{ToolName: "left"}})
	rightCalls, _ := json.Marshal([]ToolCall{{ToolName: "right"}})
	merged := mergeRunToolCalls(leftCalls, rightCalls)
	decoded := decodeToolCallArray(merged)
	if len(decoded) != 2 || decoded[0].ToolName != "left" || decoded[1].ToolName != "right" {
		t.Fatalf("mergeRunToolCalls() = %#v", decoded)
	}
	if got := decodeToolCallArray(json.RawMessage(`not-json`)); got != nil {
		t.Fatalf("decodeToolCallArray(invalid) = %#v", got)
	}
}

func TestDSLRunnerFinalizeResumeHelpers(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	mustExecDSLRunner(t, db, `INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
	VALUES ('agent_resume', 'ws_dsl', 'resume agent', 'dsl', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	orch := NewOrchestrator(db)
	runner := NewDSLRunner(db)
	rc := &RunContext{Orchestrator: orch, DB: db}

	baseRun, err := orch.TriggerAgent(context.Background(), TriggerAgentInput{
		AgentID:        "agent_resume",
		WorkspaceID:    "ws_dsl",
		TriggerType:    TriggerTypeManual,
		TriggerContext: json.RawMessage(`{"case":{"id":"case-1"}}`),
	})
	if err != nil {
		t.Fatalf("TriggerAgent() error = %v", err)
	}
	existing, err := orch.GetAgentRun(context.Background(), "ws_dsl", baseRun.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	existing.ToolCalls = json.RawMessage(`[{"tool_name":"left"}]`)

	workflow := &workflowdomain.Workflow{
		ID:          "wf-1",
		WorkspaceID: "ws_dsl",
		Name:        "resume_workflow",
		Version:     2,
		Status:      workflowdomain.StatusActive,
	}
	input := schedulerdomain.WorkflowResumePayload{
		WorkflowID:      "wf-1",
		RunID:           baseRun.ID,
		ResumeStepIndex: 3,
	}

	executor := &dslRuntimeExecutor{
		pending:         true,
		pendingApproval: &skillApprovalResult{ApprovalID: "appr-1"},
		pendingOutput:   map[string]any{"action": pendingWaitAction, "resume_step_index": 3},
		toolCalls:       []ToolCall{{ToolName: "right"}},
	}
	result := &DSLRuntimeResult{
		Statements: []DSLStatementResult{{Type: "WAIT", Status: StatusAccepted}},
	}

	updated, err := runner.finalizeResumePending(context.Background(), rc, "ws_dsl", input, workflow, existing, result, executor)
	if err != nil {
		t.Fatalf("finalizeResumePending() error = %v", err)
	}
	if updated.Status != StatusAccepted {
		t.Fatalf("status = %s, want %s", updated.Status, StatusAccepted)
	}

	var pendingOutput map[string]any
	if err := json.Unmarshal(updated.Output, &pendingOutput); err != nil {
		t.Fatalf("unmarshal pending output: %v", err)
	}
	if pendingOutput["approval_id"] != "appr-1" || pendingOutput["resumed"] != true {
		t.Fatalf("unexpected pending output = %#v", pendingOutput)
	}

	result = &DSLRuntimeResult{
		Statements: []DSLStatementResult{{Type: dslStatementTypeDispatch, Status: StatusDelegated}},
	}
	updated, err = runner.finalizeResumeDispatchTerminal(context.Background(), rc, "ws_dsl", input, workflow, existing, result, StatusDelegated)
	if err != nil {
		t.Fatalf("finalizeResumeDispatchTerminal() error = %v", err)
	}
	if updated.Status != StatusDelegated {
		t.Fatalf("status = %s, want %s", updated.Status, StatusDelegated)
	}
}
