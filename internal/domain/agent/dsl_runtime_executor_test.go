package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	schedulerdomain "github.com/matiasleandrokruk/fenix/internal/domain/scheduler"
	signaldomain "github.com/matiasleandrokruk/fenix/internal/domain/signal"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

type stubNestedRunner struct{}

func (s stubNestedRunner) Run(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*Run, error) {
	run, err := rc.Orchestrator.TriggerAgent(ctx, input)
	if err != nil {
		return nil, err
	}
	output, _ := json.Marshal(map[string]any{"result": "nested_success"})
	return rc.Orchestrator.UpdateAgentRun(ctx, input.WorkspaceID, run.ID, emptyTracesUpdate(StatusSuccess, output, json.RawMessage(emptyJSONArray), true))
}

type stubRejectedRunner struct{}

func (s stubRejectedRunner) Run(ctx context.Context, rc *RunContext, input TriggerAgentInput) (*Run, error) {
	run, err := rc.Orchestrator.TriggerAgent(ctx, input)
	if err != nil {
		return nil, err
	}
	reason := "policy rejected delegated call"
	output, _ := json.Marshal(map[string]any{"result": "rejected"})
	return rc.Orchestrator.UpdateAgentRun(ctx, input.WorkspaceID, run.ID, RunUpdates{
		Status:               StatusRejected,
		Output:               output,
		AbstentionReason:     &reason,
		ReasoningTrace:       json.RawMessage(emptyJSONArray),
		RetrievalQueries:     json.RawMessage(emptyJSONArray),
		RetrievedEvidenceIDs: json.RawMessage(emptyJSONArray),
		ToolCalls:            json.RawMessage(emptyJSONArray),
		Completed:            true,
	})
}

func TestDSLRuntimeExecutorExecutesToolOperation(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	toolRegistry := setupDSLToolRegistry(t, db)
	executor := newDSLRuntimeExecutor(&RunContext{
		ToolRegistry: toolRegistry,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeEvent,
	}, map[string]any{
		"case": map[string]any{"id": "case-1"},
	}, "", "")

	result, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:     RuntimeOperationTool,
		ToolName: tool.BuiltinUpdateCase,
		Params: map[string]any{
			"case_id": "case-1",
			"status":  "resolved",
		},
	}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if executor.IsPending() {
		t.Fatal("executor pending = true, want false")
	}
	if len(executor.toolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(executor.toolCalls))
	}
	if result.Output == nil {
		t.Fatal("expected output")
	}
}

func TestDSLRuntimeExecutorHonorsPolicyDeny(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	toolRegistry := setupDSLToolRegistry(t, db)
	engine := policy.NewPolicyEngine(db, nil, nil)
	userID := "user_dsl"
	executor := newDSLRuntimeExecutor(&RunContext{
		ToolRegistry: toolRegistry,
		PolicyEngine: engine,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggeredBy: &userID,
		TriggerType: TriggerTypeEvent,
	}, map[string]any{
		"case": map[string]any{"id": "case-1"},
	}, "", "")

	_, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:     RuntimeOperationTool,
		ToolName: tool.BuiltinUpdateCase,
		Params: map[string]any{
			"case_id": "case-1",
			"status":  "resolved",
		},
	}, nil)
	if err == nil {
		t.Fatal("expected policy deny error")
	}
}

func TestDSLRuntimeExecutorCreatesApprovalRequestWhenRequired(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	toolRegistry := setupDSLToolRegistry(t, db)
	approvalService := policy.NewApprovalService(db, nil)
	userID := "user_dsl"
	executor := newDSLRuntimeExecutor(&RunContext{
		ToolRegistry:    toolRegistry,
		ApprovalService: approvalService,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggeredBy: &userID,
		TriggerType: TriggerTypeEvent,
	}, map[string]any{
		"case": map[string]any{"id": "case-1"},
	}, "", "")

	result, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:     RuntimeOperationTool,
		ToolName: tool.BuiltinUpdateCase,
		Params: map[string]any{
			"case_id": "case-1",
			"status":  "resolved",
			"approval": map[string]any{
				"required":    true,
				"approver_id": userID,
				"reason":      "sensitive case mutation",
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !executor.IsPending() {
		t.Fatal("executor pending = false, want true")
	}
	if result.Status != StatusAccepted || !result.Stop {
		t.Fatalf("unexpected result = %#v", result)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM approval_request WHERE workspace_id = 'ws_dsl'`).Scan(&count); err != nil {
		t.Fatalf("count approval_request: %v", err)
	}
	if count != 1 {
		t.Fatalf("approval count = %d, want 1", count)
	}
}

func TestDSLRuntimeExecutorExecutesSubAgentCall(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('nested-agent-id', 'ws_dsl', 'nested-agent', 'nested', 'active', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("insert nested agent: %v", err)
	}

	registry := NewRunnerRegistry()
	if err := registry.Register("nested", stubNestedRunner{}); err != nil {
		t.Fatalf("registry.Register() error = %v", err)
	}
	orch := NewOrchestratorWithRegistry(db, registry)
	executor := newDSLRuntimeExecutor(&RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		DB:             db,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeManual,
	}, map[string]any{"case": map[string]any{"id": "case-1"}}, "", "")

	result, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:      RuntimeOperationAgent,
		AgentName: "nested-agent",
		Params:    map[string]any{"case_id": "case-1"},
	}, map[string]any{"case": map[string]any{"id": "case-1"}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != "" && result.Status != StepStatusSuccess {
		t.Fatalf("unexpected result status = %s", result.Status)
	}
}

func TestDSLRuntimeExecutorExecutesInternalDispatch(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('nested-dispatch-id', 'ws_dsl', 'dispatch-agent', 'nested', 'active', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("insert dispatch agent: %v", err)
	}

	registry := NewRunnerRegistry()
	if err := registry.Register("nested", stubNestedRunner{}); err != nil {
		t.Fatalf("registry.Register() error = %v", err)
	}
	orch := NewOrchestratorWithRegistry(db, registry)
	executor := newDSLRuntimeExecutor(&RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		DB:             db,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeManual,
	}, map[string]any{"case": map[string]any{"id": "case-1"}}, "", "")

	result, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:      RuntimeOperationDispatch,
		AgentName: "dispatch-agent",
		Params:    map[string]any{"case_id": "case-1"},
	}, map[string]any{"case": map[string]any{"id": "case-1"}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Stop != true || result.Status != StatusDelegated {
		t.Fatalf("unexpected dispatch result = %#v", result)
	}
	if executor.IsPending() {
		t.Fatal("executor pending = true, want false")
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["dispatch_result"] != dispatchResultDelegated {
		t.Fatalf("unexpected dispatch output = %#v", result.Output)
	}
}

func TestDSLRuntimeExecutorMarksDispatchAcceptedWhenTargetAccepted(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('nested-dispatch-accepted-id', 'ws_dsl', 'dispatch-pending-agent', 'nested_pending', 'active', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("insert dispatch agent: %v", err)
	}

	registry := NewRunnerRegistry()
	if err := registry.Register("nested_pending", stubPendingNestedRunner{}); err != nil {
		t.Fatalf("registry.Register() error = %v", err)
	}
	orch := NewOrchestratorWithRegistry(db, registry)
	executor := newDSLRuntimeExecutor(&RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		DB:             db,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeManual,
	}, map[string]any{"case": map[string]any{"id": "case-1"}}, "", "")

	result, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:      RuntimeOperationDispatch,
		AgentName: "dispatch-pending-agent",
		Params:    map[string]any{"case_id": "case-1"},
	}, map[string]any{"case": map[string]any{"id": "case-1"}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !executor.IsPending() || !result.Stop || result.Status != StatusAccepted {
		t.Fatalf("unexpected dispatch result = %#v", result)
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["dispatch_result"] != dispatchResultAccepted {
		t.Fatalf("unexpected dispatch output = %#v", result.Output)
	}
}

func TestDSLRuntimeExecutorMarksDispatchRejectedWithReason(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('nested-dispatch-rejected-id', 'ws_dsl', 'dispatch-rejected-agent', 'nested_rejected', 'active', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("insert dispatch agent: %v", err)
	}

	registry := NewRunnerRegistry()
	if err := registry.Register("nested_rejected", stubRejectedRunner{}); err != nil {
		t.Fatalf("registry.Register() error = %v", err)
	}
	orch := NewOrchestratorWithRegistry(db, registry)
	executor := newDSLRuntimeExecutor(&RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		DB:             db,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeManual,
	}, map[string]any{"case": map[string]any{"id": "case-1"}}, "", "")

	result, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:      RuntimeOperationDispatch,
		AgentName: "dispatch-rejected-agent",
		Params:    map[string]any{"case_id": "case-1"},
	}, map[string]any{"case": map[string]any{"id": "case-1"}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != StatusRejected || !result.Stop {
		t.Fatalf("unexpected dispatch result = %#v", result)
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["dispatch_result"] != dispatchResultRejected {
		t.Fatalf("unexpected dispatch output = %#v", result.Output)
	}
	if output["reason"] == "" {
		t.Fatalf("expected rejection reason, got %#v", output)
	}
}

func TestDSLRuntimeExecutorRejectsDispatchOnCircularDelegation(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('nested-loop-id', 'ws_dsl', 'dispatch-loop-agent', 'nested', 'active', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("insert dispatch agent: %v", err)
	}

	executor := newDSLRuntimeExecutor(&RunContext{
		Orchestrator: &Orchestrator{},
		DB:           db,
		CallChain:    []string{"nested-loop-id"},
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeManual,
	}, map[string]any{"case": map[string]any{"id": "case-1"}}, "", "")

	result, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:      RuntimeOperationDispatch,
		AgentName: "dispatch-loop-agent",
		Params:    map[string]any{"case_id": "case-1"},
	}, map[string]any{"case": map[string]any{"id": "case-1"}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != StatusRejected || !result.Stop {
		t.Fatalf("unexpected dispatch result = %#v", result)
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["reason"] != dispatchRejectLoop {
		t.Fatalf("unexpected dispatch output = %#v", result.Output)
	}
}

func TestDSLRuntimeExecutorRejectsDispatchOnDepthLimit(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('nested-depth-id', 'ws_dsl', 'dispatch-depth-agent', 'nested', 'active', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("insert dispatch agent: %v", err)
	}

	executor := newDSLRuntimeExecutor(&RunContext{
		Orchestrator: &Orchestrator{},
		DB:           db,
		CallDepth:    dslAgentCallDepthLimit,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeManual,
	}, map[string]any{"case": map[string]any{"id": "case-1"}}, "", "")

	result, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:      RuntimeOperationDispatch,
		AgentName: "dispatch-depth-agent",
		Params:    map[string]any{"case_id": "case-1"},
	}, map[string]any{"case": map[string]any{"id": "case-1"}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["reason"] != dispatchRejectDepth {
		t.Fatalf("unexpected dispatch output = %#v", result.Output)
	}
}

func TestDSLRuntimeExecutorDeniesSubAgentCallWhenPolicyRejects(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	now := time.Now().UTC()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('nested-agent-policy', 'ws_dsl', 'nested-agent', 'nested', 'active', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("insert nested agent: %v", err)
	}

	registry := NewRunnerRegistry()
	if err := registry.Register("nested", stubNestedRunner{}); err != nil {
		t.Fatalf("registry.Register() error = %v", err)
	}
	seedDSLRunnerRole(t, db, "user_dsl_agent", `{"tools":["update_case"]}`)
	engine := policy.NewPolicyEngine(db, nil, nil)
	triggeredBy := "user_dsl_agent"
	orch := NewOrchestratorWithRegistry(db, registry)
	executor := newDSLRuntimeExecutor(&RunContext{
		Orchestrator:   orch,
		RunnerRegistry: registry,
		PolicyEngine:   engine,
		DB:             db,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggeredBy: &triggeredBy,
		TriggerType: TriggerTypeManual,
	}, map[string]any{"case": map[string]any{"id": "case-1"}}, "", "")

	_, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:      RuntimeOperationAgent,
		AgentName: "nested-agent",
		Params:    map[string]any{"case_id": "case-1"},
	}, map[string]any{"case": map[string]any{"id": "case-1"}})
	if !errors.Is(err, ErrDSLAgentPolicyDenied) {
		t.Fatalf("expected ErrDSLAgentPolicyDenied, got %v", err)
	}
}

func TestDSLRuntimeExecutorCreatesSurfaceSignal(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	seedDSLRunnerCase(t, db, "case-1")
	signalService := signaldomain.NewService(db)
	executor := newDSLRuntimeExecutor(&RunContext{
		SignalService: signalService,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeManual,
	}, map[string]any{"case": map[string]any{"id": "case-1"}}, "workflow-1", "run-1")

	result, err := executor.Execute(context.Background(), &RuntimeOperation{
		Kind:   RuntimeOperationSurface,
		Target: "case",
		Params: map[string]any{
			"entity": "case",
			"view":   "salesperson.view",
			"value":  "review",
		},
	}, map[string]any{"case": map[string]any{"id": "case-1"}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["signal_id"] == "" {
		t.Fatalf("unexpected surface output = %#v", result.Output)
	}

	signals, err := signalService.GetByEntity(context.Background(), "ws_dsl", "case", "case-1")
	if err != nil {
		t.Fatalf("GetByEntity() error = %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("len(signals) = %d, want 1", len(signals))
	}
	if got := signalStringField(signals[0].Metadata, "view"); got != "salesperson.view" {
		t.Fatalf("metadata.view = %q, want %q", got, "salesperson.view")
	}
}

func setupDSLToolRegistry(t *testing.T, db *sql.DB) *tool.ToolRegistry {
	t.Helper()

	registry := tool.NewToolRegistry(db)
	createToolDefinition := func(name string, schema string) {
		if _, err := registry.CreateToolDefinition(context.Background(), tool.CreateToolDefinitionInput{
			WorkspaceID: "ws_dsl",
			Name:        name,
			InputSchema: json.RawMessage(schema),
		}); err != nil {
			t.Fatalf("CreateToolDefinition(%s) error = %v", name, err)
		}
	}

	createToolDefinition(tool.BuiltinUpdateCase, `{"type":"object","required":["case_id"],"properties":{"case_id":{"type":"string"},"status":{"type":"string"},"priority":{"type":"string"}},"additionalProperties":false}`)
	createToolDefinition(tool.BuiltinCreateTask, `{"type":"object","required":["owner_id","title","entity_type","entity_id"],"properties":{"owner_id":{"type":"string"},"title":{"type":"string"},"entity_type":{"type":"string"},"entity_id":{"type":"string"}},"additionalProperties":false}`)
	createToolDefinition(tool.BuiltinSendReply, `{"type":"object","required":["case_id","body"],"properties":{"case_id":{"type":"string"},"body":{"type":"string"}},"additionalProperties":false}`)

	if err := registry.Register(tool.BuiltinUpdateCase, stubToolExecutor{result: json.RawMessage(`{"status":"updated"}`)}); err != nil {
		t.Fatalf("Register(update_case) error = %v", err)
	}
	if err := registry.Register(tool.BuiltinCreateTask, stubToolExecutor{result: json.RawMessage(`{"status":"created"}`)}); err != nil {
		t.Fatalf("Register(create_task) error = %v", err)
	}
	if err := registry.Register(tool.BuiltinSendReply, stubToolExecutor{result: json.RawMessage(`{"status":"sent"}`)}); err != nil {
		t.Fatalf("Register(send_reply) error = %v", err)
	}

	return registry
}

func seedDSLRunnerCase(t *testing.T, db *sql.DB, caseID string) {
	t.Helper()
	mustExecDSLRunner(t, db, `
		INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at)
		VALUES ('`+caseID+`', 'ws_dsl', 'owner-1', 'Surface case', 'medium', 'open', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
}

func signalStringField(metadata json.RawMessage, key string) string {
	var payload map[string]any
	if err := json.Unmarshal(metadata, &payload); err != nil {
		return ""
	}
	value, _ := payload[key].(string)
	return value
}

func seedDSLRunnerRole(t *testing.T, db *sql.DB, userID string, permissions string) {
	t.Helper()

	now := time.Now().UTC().Format(time.RFC3339)
	roleID := uuid.NewV7().String()
	userRoleID := uuid.NewV7().String()

	if _, err := db.Exec(`
		INSERT OR IGNORE INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', ?, ?)
	`, userID, "ws_dsl", userID+"@example.com", "DSL Runner User", now, now); err != nil {
		t.Fatalf("insert user_account: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO role (id, workspace_id, name, permissions, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, roleID, "ws_dsl", "dsl-runner-role", permissions, now, now); err != nil {
		t.Fatalf("insert role: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO user_role (id, user_id, role_id, created_at)
		VALUES (?, ?, ?, ?)
	`, userRoleID, userID, roleID, now); err != nil {
		t.Fatalf("insert user_role: %v", err)
	}
}

func TestValidateAgentExecutionContextErrors(t *testing.T) {
	t.Parallel()

	// nil RunContext
	if err := validateAgentExecutionContext(nil); !errors.Is(err, ErrDSLRunnerMissingOrchestrator) {
		t.Fatalf("nil rc: err = %v, want ErrDSLRunnerMissingOrchestrator", err)
	}
	// nil Orchestrator
	if err := validateAgentExecutionContext(&RunContext{}); !errors.Is(err, ErrDSLRunnerMissingOrchestrator) {
		t.Fatalf("nil orch: err = %v, want ErrDSLRunnerMissingOrchestrator", err)
	}
	// nil DB
	if err := validateAgentExecutionContext(&RunContext{Orchestrator: &Orchestrator{}}); !errors.Is(err, ErrDSLExecutorMissingDB) {
		t.Fatalf("nil db: err = %v, want ErrDSLExecutorMissingDB", err)
	}
	// valid
	if err := validateAgentExecutionContext(&RunContext{Orchestrator: &Orchestrator{}, DB: &sql.DB{}}); err != nil {
		t.Fatalf("valid: unexpected error %v", err)
	}
}

func TestDSLRuntimeExecutorSchedulesWait(t *testing.T) {
	t.Parallel()

	db := setupDSLRunnerDB(t)
	repo := schedulerdomain.NewRepository(db)
	scheduler := schedulerdomain.NewService(repo)
	executor := newDSLRuntimeExecutor(&RunContext{
		Scheduler: scheduler,
	}, TriggerAgentInput{
		WorkspaceID: "ws_dsl",
		TriggerType: TriggerTypeManual,
	}, map[string]any{}, "wf_wait", "run_wait")

	result, err := executor.ExecuteWait(context.Background(), &WaitStatement{
		Amount: 48,
		Unit:   "hours",
	}, 3, nil)
	if err != nil {
		t.Fatalf("ExecuteWait() error = %v", err)
	}
	if !executor.IsPending() || !result.Stop || result.Status != StatusAccepted {
		t.Fatalf("unexpected wait result = %#v", result)
	}

	jobs, err := repo.ListDue(context.Background(), time.Now().UTC().Add(49*time.Hour), 10)
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
	if payload.ResumeStepIndex != 3 || payload.WorkflowID != "wf_wait" || payload.RunID != "run_wait" {
		t.Fatalf("unexpected payload = %#v", payload)
	}
}

func TestValidateAgentCallAllowedErrors(t *testing.T) {
	t.Parallel()

	target := &Definition{ID: "agent-1"}

	// call depth exceeded
	rc := &RunContext{CallDepth: dslAgentCallDepthLimit, CallChain: []string{}}
	if err := validateAgentCallAllowed(rc, "", target); !errors.Is(err, ErrDSLAgentDepthExceeded) {
		t.Fatalf("depth: err = %v, want ErrDSLAgentDepthExceeded", err)
	}
	// loop detected
	rc = &RunContext{CallDepth: 0, CallChain: []string{"agent-1", "agent-2"}}
	if err := validateAgentCallAllowed(rc, "", target); !errors.Is(err, ErrDSLAgentLoopDetected) {
		t.Fatalf("loop: err = %v, want ErrDSLAgentLoopDetected", err)
	}
	// valid
	rc = &RunContext{CallDepth: 1, CallChain: []string{"agent-2"}}
	if err := validateAgentCallAllowed(rc, "", target); err != nil {
		t.Fatalf("valid: unexpected error %v", err)
	}
}

func TestDecodeRuntimeOutput(t *testing.T) {
	t.Parallel()

	if got := decodeRuntimeOutput(json.RawMessage("")); got != nil {
		t.Fatalf("empty: got %v, want nil", got)
	}
	if got := decodeRuntimeOutput(json.RawMessage("{invalid")); got != nil {
		t.Fatalf("invalid: got %v, want nil", got)
	}
	got := decodeRuntimeOutput(json.RawMessage(`{"key":"value"}`))
	if got == nil {
		t.Fatal("valid: expected non-nil")
	}
}

func TestMergePendingApprovalMetadata(t *testing.T) {
	t.Parallel()

	// empty raw → sets default action
	out := map[string]any{}
	mergePendingApprovalMetadata(out, json.RawMessage(""))
	if out["action"] != pendingApprovalAction {
		t.Fatalf("empty: action = %v, want %v", out["action"], pendingApprovalAction)
	}

	// valid JSON with action and approval_id
	out = map[string]any{}
	mergePendingApprovalMetadata(out, json.RawMessage(`{"action":"approve","approval_id":"app-1"}`))
	if out["action"] != "approve" {
		t.Fatalf("action = %v, want approve", out["action"])
	}
	if out["approval_id"] != "app-1" {
		t.Fatalf("approval_id = %v, want app-1", out["approval_id"])
	}

	// valid JSON without action → default
	out = map[string]any{}
	mergePendingApprovalMetadata(out, json.RawMessage(`{"approval_id":"app-2"}`))
	if out["action"] != pendingApprovalAction {
		t.Fatalf("no action: action = %v, want %v", out["action"], pendingApprovalAction)
	}
}

func TestDSLRuntimeExecutorSurfaceHelpers(t *testing.T) {
	t.Parallel()

	op := &RuntimeOperation{
		Kind:   RuntimeOperationSurface,
		Target: "case",
		Params: map[string]any{
			"entity":      "case",
			"view":        "salesperson.view",
			"value":       "review",
			"signal_type": "surface.review",
			"confidence":  "0.75",
			"evidence_ids": []any{
				"ev-1", " ", "ev-2",
			},
			"metadata": map[string]any{
				"channel": "crm",
			},
		},
	}
	evalCtx := map[string]any{"case": map[string]any{"id": "case-7"}}

	input, err := buildSurfaceSignalInput(op, evalCtx, "ws_dsl", "wf-1", "run-1")
	if err != nil {
		t.Fatalf("buildSurfaceSignalInput() error = %v", err)
	}
	if input.EntityType != "case" || input.EntityID != "case-7" {
		t.Fatalf("unexpected entity = %s/%s", input.EntityType, input.EntityID)
	}
	if input.SignalType != "surface.review" {
		t.Fatalf("SignalType = %q, want surface.review", input.SignalType)
	}
	if input.Confidence != 0.75 {
		t.Fatalf("Confidence = %v, want 0.75", input.Confidence)
	}
	if len(input.EvidenceIDs) != 2 || input.EvidenceIDs[0] != "ev-1" || input.EvidenceIDs[1] != "ev-2" {
		t.Fatalf("unexpected evidence ids = %#v", input.EvidenceIDs)
	}
	if got, _ := input.Metadata["view"].(string); got != "salesperson.view" {
		t.Fatalf("metadata.view = %q, want salesperson.view", got)
	}
	if got, _ := input.Metadata["channel"].(string); got != "crm" {
		t.Fatalf("metadata.channel = %q, want crm", got)
	}

	if surfaceView(op) != "salesperson.view" {
		t.Fatalf("surfaceView() = %q", surfaceView(op))
	}
	if runtimeParamString(op, "view") != "salesperson.view" {
		t.Fatalf("runtimeParamString(view) mismatch")
	}
	if signalType := surfaceSignalType("sales.person", map[string]any{}); signalType != "surface.sales_person" {
		t.Fatalf("surfaceSignalType() = %q", signalType)
	}
	if confidence := surfaceConfidence(map[string]any{}); confidence != 1.0 {
		t.Fatalf("surfaceConfidence default = %v, want 1.0", confidence)
	}
	if ids := surfaceEvidenceIDs("", map[string]any{}); len(ids) != 1 || ids[0] != "surface" {
		t.Fatalf("surfaceEvidenceIDs default = %#v", ids)
	}
	if got, ok := floatValue(json.Number("1.5")); !ok || got != 1.5 {
		t.Fatalf("floatValue(json.Number) = %v, %v", got, ok)
	}
	if got, ok := stringSliceValue([]string{" a ", "", "b"}); !ok || len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("stringSliceValue([]string) = %#v, %v", got, ok)
	}
	if trimmed := trimNonEmptyStrings([]string{" a ", "", "b "}); len(trimmed) != 2 || trimmed[0] != "a" || trimmed[1] != "b" {
		t.Fatalf("trimNonEmptyStrings() = %#v", trimmed)
	}
	if got := firstNonEmpty("", "  ", "x", "y"); got != "x" {
		t.Fatalf("firstNonEmpty() = %q, want x", got)
	}
}

func TestDSLRuntimeExecutorDispatchAndWaitHelpers(t *testing.T) {
	t.Parallel()

	target := &Definition{ID: "agent-1", Name: "agent-one"}
	stored := &Run{ID: "run-1", Status: StatusDelegated}
	output := baseDispatchOutput(target, stored)
	if output["agent_id"] != "agent-1" || output["delegated_run_id"] != "run-1" {
		t.Fatalf("baseDispatchOutput() = %#v", output)
	}

	reason := "explicit reason"
	if got := dispatchRejectReason(&Run{Status: StatusRejected, AbstentionReason: &reason}); got != reason {
		t.Fatalf("dispatchRejectReason() = %q, want explicit reason", got)
	}
	if got := dispatchRejectReason(&Run{Status: StatusFailed}); got != "target run returned failed" {
		t.Fatalf("dispatchRejectReason fallback = %q", got)
	}
	if got := dispatchRejectionCode(ErrDSLAgentLoopDetected); got != dispatchRejectLoop {
		t.Fatalf("dispatchRejectionCode(loop) = %q", got)
	}
	if got := dispatchRejectionCode(ErrDSLAgentDepthExceeded); got != dispatchRejectDepth {
		t.Fatalf("dispatchRejectionCode(depth) = %q", got)
	}
	if got := dispatchRejectionCode(errors.New("other")); got != "dispatch_rejected" {
		t.Fatalf("dispatchRejectionCode(default) = %q", got)
	}

	if got, err := waitStatementDuration(&WaitStatement{Amount: 2, Unit: "hours"}); err != nil || got != 2*time.Hour {
		t.Fatalf("waitStatementDuration(hours) = %v, %v", got, err)
	}
	if got, err := waitStatementDuration(&WaitStatement{Amount: 0, Unit: "hours"}); err != nil || got != 0 {
		t.Fatalf("waitStatementDuration(zero) = %v, %v", got, err)
	}
	if _, err := waitStatementDuration(&WaitStatement{Amount: 1, Unit: "fortnights"}); err == nil {
		t.Fatal("expected unsupported WAIT unit error")
	}
	if got, err := waitDurationMultiplier("day"); err != nil || got != 24*time.Hour {
		t.Fatalf("waitDurationMultiplier(day) = %v, %v", got, err)
	}

	m := cloneRuntimeMap(map[string]any{"a": 1})
	m["a"] = 2
	if cloneRuntimeMap(nil) != nil {
		t.Fatal("cloneRuntimeMap(nil) should be nil")
	}
	if containsCall([]string{"a", "b"}, "b") != true || containsCall([]string{"a"}, "c") != false {
		t.Fatal("containsCall() mismatch")
	}

	rejected := rejectedDispatchExecutionResult(target, ErrDSLAgentLoopDetected)
	if rejected.Status != StatusRejected || rejected.Output.(map[string]any)["reason"] != dispatchRejectLoop {
		t.Fatalf("rejectedDispatchExecutionResult() = %#v", rejected)
	}
}

func TestDSLRuntimeExecutorHelperCoverage(t *testing.T) {
	t.Parallel()

	if got := dispatchRejectReason(nil); got != "dispatch rejected" {
		t.Fatalf("dispatchRejectReason(nil) = %q", got)
	}
	if got := dispatchRejectReason(&Run{Status: StatusRejected}); got != "target run returned rejected" {
		t.Fatalf("dispatchRejectReason(status) = %q", got)
	}
	if got := stringValue(nil); got != "" {
		t.Fatalf("stringValue(nil) = %q", got)
	}
	if got := stringValue(42); got != "42" {
		t.Fatalf("stringValue(42) = %q", got)
	}
	if _, ok := floatValue(struct{}{}); ok {
		t.Fatal("floatValue(struct{}) expected false")
	}
	if _, ok := stringSliceValue(struct{}{}); ok {
		t.Fatal("stringSliceValue(struct{}) expected false")
	}
	if got := firstNonEmpty("", " ", "winner"); got != "winner" {
		t.Fatalf("firstNonEmpty() = %q", got)
	}
	if got := surfaceSignalType("", map[string]any{}); got != "surface.generic" {
		t.Fatalf("surfaceSignalType(empty) = %q", got)
	}
	if got := surfaceSignalType("ignored", map[string]any{"signal_type": "custom.signal"}); got != "custom.signal" {
		t.Fatalf("surfaceSignalType(override) = %q", got)
	}
	if got := surfaceEvidenceIDs("", map[string]any{}); len(got) != 1 || got[0] != "surface" {
		t.Fatalf("surfaceEvidenceIDs(default) = %#v", got)
	}
	if got := cloneRuntimeParams(nil); len(got) != 0 {
		t.Fatalf("cloneRuntimeParams(nil) = %#v", got)
	}
	if got := decodeRuntimeOutput(json.RawMessage(`"ok"`)); got != "ok" {
		t.Fatalf("decodeRuntimeOutput(string) = %#v", got)
	}
	if got := marshalRuntimeContext(nil); string(got) != emptyJSONObject {
		t.Fatalf("marshalRuntimeContext(nil) = %s", string(got))
	}
	if err := checkMappedAgentPolicy(context.Background(), nil, "", nil); err != nil {
		t.Fatalf("checkMappedAgentPolicy(nil) error = %v", err)
	}
	if got := cloneRuntimeMap(nil); got != nil {
		t.Fatalf("cloneRuntimeMap(nil) = %#v", got)
	}
}
