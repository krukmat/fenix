package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	usagedomain "github.com/matiasleandrokruk/fenix/internal/domain/usage"
)

func TestUsageHandler_ListUsage_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	seedUsageWorkspaceDeps(t, db, wsID)
	service := usagedomain.NewService(db)
	runID := "run-usage-test"
	toolName := "update_case"

	_, err := service.RecordEvent(context.Background(), usagedomain.RecordEventInput{
		WorkspaceID:   wsID,
		ActorID:       "user-1",
		ActorType:     "user",
		RunID:         &runID,
		ToolName:      &toolName,
		InputUnits:    144,
		OutputUnits:   32,
		EstimatedCost: 0.031,
	})
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	h := NewUsageHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/usage?run_id="+runID, nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.ListUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp struct {
		Data []map[string]any `json:"data"`
		Meta map[string]any   `json:"meta"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data len=%d want=1", len(resp.Data))
	}
	if got := resp.Data[0]["runId"]; got != runID {
		t.Fatalf("runId=%v want=%s", got, runID)
	}
}

func TestUsageHandler_ListUsage_MissingWorkspace_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	h := NewUsageHandler(usagedomain.NewService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/usage", nil)
	rr := httptest.NewRecorder()
	h.ListUsage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusBadRequest)
	}
}

func TestUsageHandler_GetQuotaState_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	service := usagedomain.NewService(db)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	lastEventAt := time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)

	policy, err := service.CreatePolicy(context.Background(), usagedomain.CreatePolicyInput{
		WorkspaceID:     wsID,
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

	_, err = service.UpsertState(context.Background(), usagedomain.UpsertStateInput{
		WorkspaceID:   wsID,
		QuotaPolicyID: policy.ID,
		CurrentValue:  42.5,
		PeriodStart:   start,
		PeriodEnd:     end,
		LastEventAt:   &lastEventAt,
	})
	if err != nil {
		t.Fatalf("UpsertState: %v", err)
	}

	h := NewUsageHandler(service)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/quota-state?quota_policy_id="+policy.ID+"&period_start="+start.Format(time.RFC3339)+"&period_end="+end.Format(time.RFC3339),
		nil,
	)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))

	rr := httptest.NewRecorder()
	h.GetQuotaState(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := resp.Data["quotaPolicyId"]; got != policy.ID {
		t.Fatalf("quotaPolicyId=%v want=%s", got, policy.ID)
	}
	if got := resp.Data["currentValue"]; got != 42.5 {
		t.Fatalf("currentValue=%v want=42.5", got)
	}
}

func TestUsageHandler_GetQuotaState_MissingPolicyID_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := NewUsageHandler(usagedomain.NewService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quota-state", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	h.GetQuotaState(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusBadRequest)
	}
}

func TestUsageHandler_GetQuotaState_NotFound_Returns404(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := NewUsageHandler(usagedomain.NewService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quota-state?quota_policy_id=missing-policy", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	h.GetQuotaState(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
}

func TestUsageHandler_GetQuotaState_InvalidPeriodStart_Returns400(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := NewUsageHandler(usagedomain.NewService(db))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quota-state?quota_policy_id=policy-1&period_start=bad-date", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	h.GetQuotaState(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
}

func seedUsageWorkspaceDeps(t *testing.T, db *sql.DB, wsID string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO agent_definition (id, workspace_id, name, agent_type, status, created_at, updated_at)
		VALUES ('agent-usage-test', ?, 'Usage Agent', 'support', 'active', datetime('now'), datetime('now'))
	`, wsID); err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO agent_run (id, workspace_id, agent_definition_id, trigger_type, status, started_at, created_at, updated_at)
		VALUES ('run-usage-test', ?, 'agent-usage-test', 'manual', 'success', datetime('now'), datetime('now'), datetime('now'))
	`, wsID); err != nil {
		t.Fatalf("insert agent_run: %v", err)
	}
}
