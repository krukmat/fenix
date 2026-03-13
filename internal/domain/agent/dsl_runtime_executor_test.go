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
