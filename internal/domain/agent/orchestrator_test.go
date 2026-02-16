// Package agent — orchestrator tests.
// Task 3.7: Agent Runtime state machine
package agent

import (
	"context"
	"testing"
)

// insertTestAgentDefinition inserts an agent definition for tests.
func insertTestAgentDefinition(t *testing.T, ctx context.Context, db interface {
	ExecContext(ctx context.Context, query string, args ...any) (interface{ LastInsertId() (int64, error); RowsAffected() (int64, error) }, error)
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
