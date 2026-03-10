// Package agent — orchestrator tests.
// Task 3.7: Agent Runtime state machine
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// insertTestAgentDefinition inserts an agent definition for tests.
func insertTestAgentDefinition(t *testing.T, ctx context.Context, db interface {
	ExecContext(ctx context.Context, query string, args ...any) (interface {
		LastInsertId() (int64, error)
		RowsAffected() (int64, error)
	}, error)
}, id, workspaceID, name, status string) {
	t.Helper()
}

// TestTriggerAgent_Success verifies a valid trigger creates a run with status=running.
// Traces: FR-230
func TestTriggerAgent_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	orch := NewOrchestrator(db)

	// Insert agent_definition
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-1', 'ws-1', 'Test Agent', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}

	run, err := orch.TriggerAgent(ctx, TriggerAgentInput{
		AgentID:     "agent-1",
		WorkspaceID: "ws-1",
		TriggerType: TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent: %v", err)
	}
	if run.Status != StatusRunning {
		t.Errorf("expected status=running, got %s", run.Status)
	}
	if run.DefinitionID != "agent-1" {
		t.Errorf("expected definition_id=agent-1, got %s", run.DefinitionID)
	}
	if run.WorkspaceID != "ws-1" {
		t.Errorf("expected workspace_id=ws-1, got %s", run.WorkspaceID)
	}
}

// TestTriggerAgent_AgentNotFound returns ErrAgentNotFound for unknown agent.
// Traces: FR-230
func TestTriggerAgent_AgentNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orch := NewOrchestrator(db)
	_, err := orch.TriggerAgent(context.Background(), TriggerAgentInput{
		AgentID:     "nonexistent",
		WorkspaceID: "ws-1",
		TriggerType: TriggerTypeManual,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrAgentNotFound {
		t.Errorf("expected ErrAgentNotFound, got: %v", err)
	}
}

// TestTriggerAgent_AgentNotActive returns ErrAgentNotActive for paused agent.
// Traces: FR-230
func TestTriggerAgent_AgentNotActive(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-paused', 'ws-1', 'Paused', 'support', 'paused')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	orch := NewOrchestrator(db)
	_, err = orch.TriggerAgent(ctx, TriggerAgentInput{
		AgentID:     "agent-paused",
		WorkspaceID: "ws-1",
		TriggerType: TriggerTypeManual,
	})
	if err != ErrAgentNotActive {
		t.Errorf("expected ErrAgentNotActive, got: %v", err)
	}
}

// TestTriggerAgent_InvalidTriggerType returns ErrInvalidTriggerType.
// Traces: FR-230
func TestTriggerAgent_InvalidTriggerType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orch := NewOrchestrator(db)
	_, err := orch.TriggerAgent(context.Background(), TriggerAgentInput{
		AgentID:     "agent-1",
		WorkspaceID: "ws-1",
		TriggerType: "invalid-type",
	})
	if err != ErrInvalidTriggerType {
		t.Errorf("expected ErrInvalidTriggerType, got: %v", err)
	}
}

// TestGetAgentRun_NotFound returns ErrAgentRunNotFound for unknown run.
// Traces: FR-230
func TestGetAgentRun_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orch := NewOrchestrator(db)
	_, err := orch.GetAgentRun(context.Background(), "ws-1", "nonexistent-run")
	if err != ErrAgentRunNotFound {
		t.Errorf("expected ErrAgentRunNotFound, got: %v", err)
	}
}

// TestListAgentRuns_Empty returns empty slice when no runs.
// Traces: FR-230
func TestListAgentRuns_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orch := NewOrchestrator(db)
	runs, total, err := orch.ListAgentRuns(context.Background(), "ws-1", 25, 0)
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}
	if total != 0 {
		t.Errorf("expected total=0, got %d", total)
	}
}

// TestListAgentRuns_Pagination verifies limit is respected.
// Traces: FR-230
func TestListAgentRuns_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-pg', 'ws-pg', 'Paginate', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert definition: %v", err)
	}

	orch := NewOrchestrator(db)
	for i := 0; i < 3; i++ {
		_, err := orch.TriggerAgent(ctx, TriggerAgentInput{
			AgentID:     "agent-pg",
			WorkspaceID: "ws-pg",
			TriggerType: TriggerTypeManual,
		})
		if err != nil {
			t.Fatalf("TriggerAgent[%d]: %v", i, err)
		}
	}

	runs, total, err := orch.ListAgentRuns(ctx, "ws-pg", 2, 0)
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 runs (limit), got %d", len(runs))
	}
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
}

// TestUpdateAgentRunStatus_Success updates status and sets completed_at.
// Traces: FR-230
func TestUpdateAgentRunStatus_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-upd', 'ws-upd', 'Update', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert definition: %v", err)
	}

	orch := NewOrchestrator(db)
	run, err := orch.TriggerAgent(ctx, TriggerAgentInput{
		AgentID:     "agent-upd",
		WorkspaceID: "ws-upd",
		TriggerType: TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent: %v", err)
	}

	updated, err := orch.UpdateAgentRunStatus(ctx, "ws-upd", run.ID, StatusSuccess)
	if err != nil {
		t.Fatalf("UpdateAgentRunStatus: %v", err)
	}
	if updated.Status != StatusSuccess {
		t.Errorf("expected status=success, got %s", updated.Status)
	}
	if updated.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

// TestTriggerAgent_CreatesInitialPendingStep verifies the runtime creates the first pending step.
// Traces: FR-230
func TestTriggerAgent_CreatesInitialPendingStep(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-steps', 'ws-steps', 'Steps', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert definition: %v", err)
	}

	orch := NewOrchestrator(db)
	run, err := orch.TriggerAgent(ctx, TriggerAgentInput{
		AgentID:     "agent-steps",
		WorkspaceID: "ws-steps",
		TriggerType: TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent: %v", err)
	}

	steps, err := orch.ListRunSteps(ctx, "ws-steps", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].StepType != StepTypeRetrieveEvidence {
		t.Fatalf("expected retrieve_evidence, got %s", steps[0].StepType)
	}
	if steps[0].Status != StepStatusPending {
		t.Fatalf("expected pending, got %s", steps[0].Status)
	}
}

// TestUpdateAgentRunStatus_InvalidTerminalTransition rejects changes after completion.
// Traces: FR-230
func TestUpdateAgentRunStatus_InvalidTerminalTransition(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-term', 'ws-term', 'Terminal', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert definition: %v", err)
	}

	orch := NewOrchestrator(db)
	run, err := orch.TriggerAgent(ctx, TriggerAgentInput{
		AgentID:     "agent-term",
		WorkspaceID: "ws-term",
		TriggerType: TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent: %v", err)
	}

	if _, err := orch.UpdateAgentRunStatus(ctx, "ws-term", run.ID, StatusSuccess); err != nil {
		t.Fatalf("UpdateAgentRunStatus(success): %v", err)
	}
	_, err = orch.UpdateAgentRunStatus(ctx, "ws-term", run.ID, StatusFailed)
	if !errors.Is(err, ErrInvalidRunTransition) {
		t.Fatalf("expected ErrInvalidRunTransition, got %v", err)
	}
}

func TestUpdateAgentRunStatus_AcceptsNewTerminalStates(t *testing.T) {
	ctx := context.Background()

	cases := []string{StatusRejected, StatusDelegated}
	for _, nextStatus := range cases {
		runDB := setupTestDB(t)
		defer runDB.Close()

		_, err := runDB.ExecContext(ctx,
			`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
			 VALUES ('agent-f17', 'ws-f17', 'F17', 'support', 'active')`)
		if err != nil {
			t.Fatalf("insert definition: %v", err)
		}

		orch := NewOrchestrator(runDB)
		run, err := orch.TriggerAgent(ctx, TriggerAgentInput{
			AgentID:     "agent-f17",
			WorkspaceID: "ws-f17",
			TriggerType: TriggerTypeManual,
		})
		if err != nil {
			t.Fatalf("TriggerAgent: %v", err)
		}

		updated, err := orch.UpdateAgentRunStatus(ctx, "ws-f17", run.ID, nextStatus)
		if err != nil {
			t.Fatalf("UpdateAgentRunStatus(%s): %v", nextStatus, err)
		}
		if updated.Status != nextStatus {
			t.Fatalf("Status = %q, want %q", updated.Status, nextStatus)
		}
	}
}

func TestUpdateAgentRun_AcceptedThenSuccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-accepted', 'ws-accepted', 'Accepted', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert definition: %v", err)
	}

	orch := NewOrchestrator(db)
	run, err := orch.TriggerAgent(ctx, TriggerAgentInput{
		AgentID:     "agent-accepted",
		WorkspaceID: "ws-accepted",
		TriggerType: TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent: %v", err)
	}

	if _, err := orch.UpdateAgentRun(ctx, "ws-accepted", run.ID, RunUpdates{
		Status:    StatusAccepted,
		Completed: false,
	}); err != nil {
		t.Fatalf("UpdateAgentRun(accepted): %v", err)
	}

	updated, err := orch.UpdateAgentRun(ctx, "ws-accepted", run.ID, RunUpdates{
		Status:    StatusSuccess,
		Output:    json.RawMessage(`{"ok":true}`),
		Completed: true,
	})
	if err != nil {
		t.Fatalf("UpdateAgentRun(success): %v", err)
	}
	if updated.Status != StatusSuccess {
		t.Fatalf("Status = %q, want %q", updated.Status, StatusSuccess)
	}
}

// TestUpdateAgentRun_SynthesizesStepsForCompletedRun verifies compatibility path materializes runtime steps.
// Traces: FR-230
func TestUpdateAgentRun_SynthesizesStepsForCompletedRun(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-sync', 'ws-sync', 'Synthesize', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert definition: %v", err)
	}

	orch := NewOrchestrator(db)
	run, err := orch.TriggerAgent(ctx, TriggerAgentInput{
		AgentID:     "agent-sync",
		WorkspaceID: "ws-sync",
		TriggerType: TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent: %v", err)
	}

	updated, err := orch.UpdateAgentRun(ctx, "ws-sync", run.ID, RunUpdates{
		Status:               StatusSuccess,
		Inputs:               json.RawMessage(`{"case_id":"case-1"}`),
		RetrievalQueries:     json.RawMessage(`["case status"]`),
		RetrievedEvidenceIDs: json.RawMessage(`["evidence-1"]`),
		ReasoningTrace:       json.RawMessage(`["reasoned over evidence"]`),
		ToolCalls:            json.RawMessage(`[{"tool_name":"create_task"}]`),
		Output:               json.RawMessage(`{"summary":"done"}`),
		Completed:            true,
	})
	if err != nil {
		t.Fatalf("UpdateAgentRun: %v", err)
	}
	if updated.CompletedAt == nil {
		t.Fatal("expected completed_at")
	}
	if updated.LatencyMs == nil {
		t.Fatal("expected latency_ms to be synthesized")
	}

	steps, err := orch.ListRunSteps(ctx, "ws-sync", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}
	if len(steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(steps))
	}
	if steps[0].Status != StepStatusSuccess {
		t.Fatalf("expected retrieval success, got %s", steps[0].Status)
	}
	if steps[1].StepType != StepTypeReason || steps[1].Status != StepStatusSuccess {
		t.Fatalf("expected reason success, got %s/%s", steps[1].StepType, steps[1].Status)
	}
	if steps[2].StepType != StepTypeToolCall || steps[2].Status != StepStatusSuccess {
		t.Fatalf("expected tool_call success, got %s/%s", steps[2].StepType, steps[2].Status)
	}
	if steps[3].StepType != StepTypeFinalize || steps[3].Status != StepStatusSuccess {
		t.Fatalf("expected finalize success, got %s/%s", steps[3].StepType, steps[3].Status)
	}
}

// TestRecoverRun_RetryableRunningStepCreatesRetryAttempt verifies retry orchestration for retryable steps.
// Traces: FR-230
func TestRecoverRun_RetryableRunningStepCreatesRetryAttempt(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-retry', 'ws-retry', 'Retry', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert definition: %v", err)
	}

	orch := NewOrchestrator(db)
	run, err := orch.TriggerAgent(ctx, TriggerAgentInput{
		AgentID:     "agent-retry",
		WorkspaceID: "ws-retry",
		TriggerType: TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO agent_run_step (
			id, workspace_id, agent_run_id, step_index, step_type, status, attempt, created_at, updated_at
		) VALUES ('step-tool-1', 'ws-retry', ?, 1, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, run.ID, StepTypeToolCall, StepStatusRunning)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	if _, err := orch.RecoverRun(ctx, "ws-retry", run.ID); err != nil {
		t.Fatalf("RecoverRun: %v", err)
	}

	steps, err := orch.ListRunSteps(ctx, "ws-retry", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps after retry, got %d", len(steps))
	}
	if steps[1].Status != StepStatusRetrying {
		t.Fatalf("expected original step retrying, got %s", steps[1].Status)
	}
	if steps[2].Status != StepStatusPending || steps[2].Attempt != 2 {
		t.Fatalf("expected new pending retry attempt, got %s/%d", steps[2].Status, steps[2].Attempt)
	}
}

// TestRecoverRun_NonRetryableRunningStepFailsRun verifies exhausted or non-retryable steps fail the run.
// Traces: FR-230
func TestRecoverRun_NonRetryableRunningStepFailsRun(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-fail', 'ws-fail', 'Fail', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert definition: %v", err)
	}

	orch := NewOrchestrator(db)
	run, err := orch.TriggerAgent(ctx, TriggerAgentInput{
		AgentID:     "agent-fail",
		WorkspaceID: "ws-fail",
		TriggerType: TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO agent_run_step (
			id, workspace_id, agent_run_id, step_index, step_type, status, attempt, created_at, updated_at
		) VALUES ('step-reason-1', 'ws-fail', ?, 1, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, run.ID, StepTypeReason, StepStatusRunning)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	recovered, err := orch.RecoverRun(ctx, "ws-fail", run.ID)
	if err != nil {
		t.Fatalf("RecoverRun: %v", err)
	}
	if recovered.Status != StatusFailed {
		t.Fatalf("expected failed run, got %s", recovered.Status)
	}

	steps, err := orch.ListRunSteps(ctx, "ws-fail", run.ID)
	if err != nil {
		t.Fatalf("ListRunSteps: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	if steps[1].Status != StepStatusFailed {
		t.Fatalf("expected failing step to be failed, got %s", steps[1].Status)
	}
	if steps[2].StepType != StepTypeFinalize || steps[2].Status != StepStatusFailed {
		t.Fatalf("expected failed finalize step, got %s/%s", steps[2].StepType, steps[2].Status)
	}
}

// TestListAgentDefinitions_Success lists all definitions for a workspace.
// Traces: FR-230
func TestListAgentDefinitions_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	for _, row := range []struct{ id, name string }{
		{"def-1", "Agent One"},
		{"def-2", "Agent Two"},
	} {
		_, err := db.ExecContext(ctx,
			`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
			 VALUES (?, 'ws-list', ?, 'support', 'active')`, row.id, row.name)
		if err != nil {
			t.Fatalf("insert %s: %v", row.id, err)
		}
	}

	orch := NewOrchestrator(db)
	defs, err := orch.ListAgentDefinitions(ctx, "ws-list")
	if err != nil {
		t.Fatalf("ListAgentDefinitions: %v", err)
	}
	if len(defs) != 2 {
		t.Errorf("expected 2 definitions, got %d", len(defs))
	}
}

// TestGetAgentDefinition_Success retrieves a specific definition.
// Traces: FR-230
func TestGetAgentDefinition_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('def-get', 'ws-get', 'Get Agent', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	orch := NewOrchestrator(db)
	def, err := orch.GetAgentDefinition(ctx, "ws-get", "def-get")
	if err != nil {
		t.Fatalf("GetAgentDefinition: %v", err)
	}
	if def.Name != "Get Agent" {
		t.Errorf("expected name='Get Agent', got %s", def.Name)
	}
}

func TestResolveRunner_RegistryNotConfigured(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('def-runner', 'ws-runner', 'Runner Agent', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	orch := NewOrchestrator(db)
	_, err = orch.ResolveRunner(ctx, "ws-runner", "def-runner")
	if err != ErrRunnerRegistryUnset {
		t.Fatalf("ResolveRunner() error = %v, want %v", err, ErrRunnerRegistryUnset)
	}
}

func TestResolveRunner_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('def-runner', 'ws-runner', 'Runner Agent', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	registry := NewRunnerRegistry()
	if err := registry.Register("support", stubRunner{}); err != nil {
		t.Fatalf("Register(): %v", err)
	}

	orch := NewOrchestratorWithRegistry(db, registry)
	runner, err := orch.ResolveRunner(ctx, "ws-runner", "def-runner")
	if err != nil {
		t.Fatalf("ResolveRunner(): %v", err)
	}
	if runner == nil {
		t.Fatal("ResolveRunner() returned nil runner")
	}
}

func TestExecuteAgent_DelegatesToRunner(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	_, err := db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('def-exec', 'ws-exec', 'Exec Agent', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	registry := NewRunnerRegistry()
	if err := registry.Register("support", stubRunner{}); err != nil {
		t.Fatalf("Register(): %v", err)
	}

	orch := NewOrchestratorWithRegistry(db, registry)
	run, err := orch.ExecuteAgent(ctx, &RunContext{}, TriggerAgentInput{
		AgentID:     "def-exec",
		WorkspaceID: "ws-exec",
		TriggerType: TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("ExecuteAgent(): %v", err)
	}
	if run.DefinitionID != "def-exec" {
		t.Fatalf("DefinitionID = %q, want %q", run.DefinitionID, "def-exec")
	}
}
