package usage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func setupUsageTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-usage', 'Usage Workspace', 'usage-workspace', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('agent-usage', 'ws-usage', 'Usage Agent', 'support', 'active', ?, ?)
	`, now, now); err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO agent_run (id, workspace_id, agent_definition_id, trigger_type, status, started_at, created_at, updated_at)
		VALUES ('run-usage', 'ws-usage', 'agent-usage', 'manual', 'success', ?, ?, ?)
	`, now, now, now); err != nil {
		t.Fatalf("insert agent_run: %v", err)
	}

	return db
}

func TestService_RecordEventAndListEvents(t *testing.T) {
	db := setupUsageTestDB(t)
	defer db.Close()

	service := NewService(db)
	runID := "run-usage"
	toolName := "update_case"
	modelName := "gpt-5.4-mini"
	latencyMs := int64(1430)

	event, err := service.RecordEvent(context.Background(), RecordEventInput{
		WorkspaceID:   "ws-usage",
		ActorID:       "user-1",
		ActorType:     "user",
		RunID:         &runID,
		ToolName:      &toolName,
		ModelName:     &modelName,
		InputUnits:    120,
		OutputUnits:   34,
		EstimatedCost: 0.021,
		LatencyMs:     &latencyMs,
	})
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if event.ID == "" {
		t.Fatal("expected event ID")
	}

	events, err := service.ListEvents(context.Background(), "ws-usage", &runID, 10)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if events[0].ActorID != "user-1" {
		t.Fatalf("actor_id = %q, want user-1", events[0].ActorID)
	}
	if events[0].RunID == nil || *events[0].RunID != runID {
		t.Fatalf("run_id = %v, want %q", events[0].RunID, runID)
	}
}

func TestService_CreatePolicyAndUpsertState(t *testing.T) {
	db := setupUsageTestDB(t)
	defer db.Close()

	service := NewService(db)

	policy, err := service.CreatePolicy(context.Background(), CreatePolicyInput{
		WorkspaceID:     "ws-usage",
		PolicyType:      "workspace_budget",
		ScopeType:       "workspace",
		MetricName:      "estimated_cost",
		LimitValue:      100,
		ResetPeriod:     "monthly",
		EnforcementMode: "soft",
		IsActive:        true,
	})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if policy.ID == "" {
		t.Fatal("expected policy ID")
	}

	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	lastEvent := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)

	state, err := service.UpsertState(context.Background(), UpsertStateInput{
		WorkspaceID:   "ws-usage",
		QuotaPolicyID: policy.ID,
		CurrentValue:  42.5,
		PeriodStart:   start,
		PeriodEnd:     end,
		LastEventAt:   &lastEvent,
	})
	if err != nil {
		t.Fatalf("UpsertState: %v", err)
	}
	if state.CurrentValue != 42.5 {
		t.Fatalf("current_value = %f, want 42.5", state.CurrentValue)
	}
	if state.LastEventAt == nil || !state.LastEventAt.Equal(lastEvent) {
		t.Fatalf("last_event_at = %v, want %v", state.LastEventAt, lastEvent)
	}

	stored, err := service.GetState(context.Background(), "ws-usage", policy.ID, start, end)
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if stored.ID != state.ID {
		t.Fatalf("state id = %q, want %q", stored.ID, state.ID)
	}
}
