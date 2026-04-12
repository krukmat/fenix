package usage

import (
	"context"
	"database/sql"
	"errors"
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

func TestService_ListActivePolicies(t *testing.T) {
	db := setupUsageTestDB(t)
	defer db.Close()

	service := NewService(db)

	t.Run("empty workspace", func(t *testing.T) {
		policies, err := service.ListActivePolicies(context.Background(), "ws-usage")
		if err != nil {
			t.Fatalf("ListActivePolicies: %v", err)
		}
		if len(policies) != 0 {
			t.Fatalf("policies len = %d, want 0", len(policies))
		}
	})

	t.Run("returns only active policies", func(t *testing.T) {
		active, err := service.CreatePolicy(context.Background(), CreatePolicyInput{
			WorkspaceID:     "ws-usage",
			PolicyType:      "workspace_budget",
			MetricName:      "estimated_cost",
			LimitValue:      100,
			ResetPeriod:     "monthly",
			EnforcementMode: "soft",
			IsActive:        true,
		})
		if err != nil {
			t.Fatalf("CreatePolicy active: %v", err)
		}

		_, err = service.CreatePolicy(context.Background(), CreatePolicyInput{
			WorkspaceID:     "ws-usage",
			PolicyType:      "agent_budget",
			MetricName:      "estimated_cost",
			LimitValue:      10,
			ResetPeriod:     "daily",
			EnforcementMode: "hard",
			IsActive:        false,
		})
		if err != nil {
			t.Fatalf("CreatePolicy inactive: %v", err)
		}

		policies, err := service.ListActivePolicies(context.Background(), "ws-usage")
		if err != nil {
			t.Fatalf("ListActivePolicies: %v", err)
		}
		if len(policies) != 1 {
			t.Fatalf("policies len = %d, want 1", len(policies))
		}
		if policies[0].ID != active.ID {
			t.Fatalf("policy id = %q, want %q", policies[0].ID, active.ID)
		}
		if !policies[0].IsActive {
			t.Fatal("expected IsActive=true")
		}
	})

	t.Run("policy with scope id", func(t *testing.T) {
		scopeID := "agent-usage"
		_, err := service.CreatePolicy(context.Background(), CreatePolicyInput{
			WorkspaceID:     "ws-usage",
			PolicyType:      "agent_budget",
			ScopeID:         &scopeID,
			MetricName:      "tokens",
			LimitValue:      50000,
			ResetPeriod:     "daily",
			EnforcementMode: "soft",
			IsActive:        true,
		})
		if err != nil {
			t.Fatalf("CreatePolicy with scope: %v", err)
		}

		policies, err := service.ListActivePolicies(context.Background(), "ws-usage")
		if err != nil {
			t.Fatalf("ListActivePolicies: %v", err)
		}

		found := false
		for _, policy := range policies {
			if policy.ScopeID != nil && *policy.ScopeID == scopeID {
				found = true
			}
		}
		if !found {
			t.Fatal("expected policy with ScopeID set")
		}
	})
}

func TestService_GetState_NotFound(t *testing.T) {
	db := setupUsageTestDB(t)
	defer db.Close()

	service := NewService(db)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	_, err := service.GetState(context.Background(), "ws-usage", "nonexistent-policy-id", start, end)
	if !errors.Is(err, ErrQuotaStateNotFound) {
		t.Fatalf("want ErrQuotaStateNotFound, got %v", err)
	}
}

func TestService_ListEvents_DefaultLimitAndNilRunID(t *testing.T) {
	db := setupUsageTestDB(t)
	defer db.Close()

	service := NewService(db)

	for i := 0; i < 2; i++ {
		_, err := service.RecordEvent(context.Background(), RecordEventInput{
			WorkspaceID: "ws-usage",
			ActorID:     "user-1",
			ActorType:   "user",
			InputUnits:  10,
			OutputUnits: 5,
		})
		if err != nil {
			t.Fatalf("RecordEvent %d: %v", i, err)
		}
	}

	t.Run("limit defaults to 50", func(t *testing.T) {
		events, err := service.ListEvents(context.Background(), "ws-usage", nil, 0)
		if err != nil {
			t.Fatalf("ListEvents: %v", err)
		}
		if len(events) < 2 {
			t.Fatalf("events len = %d, want at least 2", len(events))
		}
	})

	t.Run("nil run id returns all workspace events", func(t *testing.T) {
		events, err := service.ListEvents(context.Background(), "ws-usage", nil, 10)
		if err != nil {
			t.Fatalf("ListEvents nil runID: %v", err)
		}
		if len(events) < 2 {
			t.Fatalf("events len = %d, want at least 2", len(events))
		}
	})
}

func TestService_CreatePolicy_Defaults(t *testing.T) {
	db := setupUsageTestDB(t)
	defer db.Close()

	service := NewService(db)

	policy, err := service.CreatePolicy(context.Background(), CreatePolicyInput{
		WorkspaceID: "ws-usage",
		PolicyType:  "workspace_budget",
		MetricName:  "estimated_cost",
		LimitValue:  200,
		ResetPeriod: "monthly",
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}
	if policy.ScopeType != "workspace" {
		t.Fatalf("ScopeType = %q, want workspace", policy.ScopeType)
	}
	if policy.EnforcementMode != "soft" {
		t.Fatalf("EnforcementMode = %q, want soft", policy.EnforcementMode)
	}
}

func TestService_UpsertState_NilLastEventAt(t *testing.T) {
	db := setupUsageTestDB(t)
	defer db.Close()

	service := NewService(db)

	policy, err := service.CreatePolicy(context.Background(), CreatePolicyInput{
		WorkspaceID:     "ws-usage",
		PolicyType:      "workspace_budget",
		MetricName:      "estimated_cost",
		LimitValue:      100,
		ResetPeriod:     "monthly",
		EnforcementMode: "soft",
		IsActive:        true,
	})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	state, err := service.UpsertState(context.Background(), UpsertStateInput{
		WorkspaceID:   "ws-usage",
		QuotaPolicyID: policy.ID,
		CurrentValue:  0,
		PeriodStart:   start,
		PeriodEnd:     end,
		LastEventAt:   nil,
	})
	if err != nil {
		t.Fatalf("UpsertState: %v", err)
	}
	if state.LastEventAt != nil {
		t.Fatalf("LastEventAt = %v, want nil", state.LastEventAt)
	}
}

func TestService_RecordEvent_DBError(t *testing.T) {
	db := setupUsageTestDB(t)
	service := NewService(db)
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := service.RecordEvent(context.Background(), RecordEventInput{
		WorkspaceID: "ws-usage",
		ActorID:     "user-1",
		ActorType:   "user",
	})
	if err == nil {
		t.Fatal("expected RecordEvent error with closed DB")
	}
}

func TestService_ListEvents_DBError(t *testing.T) {
	db := setupUsageTestDB(t)
	service := NewService(db)
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := service.ListEvents(context.Background(), "ws-usage", nil, 10)
	if err == nil {
		t.Fatal("expected ListEvents error with closed DB")
	}
}

func TestService_CreatePolicy_DBError(t *testing.T) {
	db := setupUsageTestDB(t)
	service := NewService(db)
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := service.CreatePolicy(context.Background(), CreatePolicyInput{
		WorkspaceID: "ws-usage",
		PolicyType:  "workspace_budget",
		MetricName:  "estimated_cost",
		LimitValue:  100,
		ResetPeriod: "monthly",
		IsActive:    true,
	})
	if err == nil {
		t.Fatal("expected CreatePolicy error with closed DB")
	}
}

func TestService_UpsertState_DBError(t *testing.T) {
	db := setupUsageTestDB(t)
	service := NewService(db)
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	_, err := service.UpsertState(context.Background(), UpsertStateInput{
		WorkspaceID:   "ws-usage",
		QuotaPolicyID: "policy-id",
		CurrentValue:  1,
		PeriodStart:   start,
		PeriodEnd:     end,
	})
	if err == nil {
		t.Fatal("expected UpsertState error with closed DB")
	}
}
