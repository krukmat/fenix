package handlers

// W1-T4 (mobile_wedge_harmonization_plan): governance summary handler tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	usagedomain "github.com/matiasleandrokruk/fenix/internal/domain/usage"
)

func TestGovernanceHandler_GetGovernanceSummary_NoData(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	service := usagedomain.NewService(db)
	h := NewGovernanceHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/governance/summary", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.GetGovernanceSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp struct {
		RecentUsage []interface{} `json:"recentUsage"`
		QuotaStates []interface{} `json:"quotaStates"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.RecentUsage) != 0 {
		t.Errorf("recentUsage want=0 got=%d", len(resp.RecentUsage))
	}
	if len(resp.QuotaStates) != 0 {
		t.Errorf("quotaStates want=0 got=%d", len(resp.QuotaStates))
	}
}

func TestGovernanceHandler_GetGovernanceSummary_WithPolicyAndUsage(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	seedUsageWorkspaceDeps(t, db, wsID)
	service := usagedomain.NewService(db)

	// Create an active policy
	policy, err := service.CreatePolicy(t.Context(), usagedomain.CreatePolicyInput{
		WorkspaceID:     wsID,
		PolicyType:      "token_budget",
		MetricName:      "input_units",
		LimitValue:      10000,
		ResetPeriod:     "monthly",
		EnforcementMode: "soft",
		IsActive:        true,
	})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	// Record a usage event
	runID := "run-usage-test"
	if _, err := service.RecordEvent(t.Context(), usagedomain.RecordEventInput{
		WorkspaceID: wsID,
		ActorID:     "user-1",
		ActorType:   "user",
		RunID:       &runID,
		InputUnits:  100,
		OutputUnits: 20,
	}); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	h := NewGovernanceHandler(service)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/governance/summary", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.GetGovernanceSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp struct {
		RecentUsage []map[string]interface{} `json:"recentUsage"`
		QuotaStates []map[string]interface{} `json:"quotaStates"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.RecentUsage) != 1 {
		t.Errorf("recentUsage want=1 got=%d", len(resp.RecentUsage))
	}
	if len(resp.QuotaStates) != 1 {
		t.Errorf("quotaStates want=1 got=%d", len(resp.QuotaStates))
	}

	// Verify enriched quota state fields are present
	qs := resp.QuotaStates[0]
	if qs["policyId"] != policy.ID {
		t.Errorf("policyId want=%s got=%v", policy.ID, qs["policyId"])
	}
	if qs["policyType"] != "token_budget" {
		t.Errorf("policyType want=token_budget got=%v", qs["policyType"])
	}
	// No state row was upserted — statePresent should be false, currentValue 0
	if qs["statePresent"] != false {
		t.Errorf("statePresent want=false got=%v", qs["statePresent"])
	}
	if qs["currentValue"] != float64(0) {
		t.Errorf("currentValue want=0 got=%v", qs["currentValue"])
	}
}

func TestGovernanceHandler_GetGovernanceSummary_WithUpsertedState(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	service := usagedomain.NewService(db)

	policy, err := service.CreatePolicy(t.Context(), usagedomain.CreatePolicyInput{
		WorkspaceID:     wsID,
		PolicyType:      "cost_budget",
		MetricName:      "estimated_cost",
		LimitValue:      50.0,
		ResetPeriod:     "monthly",
		EnforcementMode: "hard",
		IsActive:        true,
	})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	// Upsert a state row for the current month using the same period bounds the handler computes
	h := NewGovernanceHandler(service)
	start, end := currentPeriodBounds(time.Now().UTC(), "monthly")
	if _, upsertErr := service.UpsertState(t.Context(), usagedomain.UpsertStateInput{
		WorkspaceID:   wsID,
		QuotaPolicyID: policy.ID,
		CurrentValue:  12.5,
		PeriodStart:   start,
		PeriodEnd:     end,
	}); upsertErr != nil {
		t.Fatalf("UpsertState: %v", upsertErr)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/governance/summary", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	h.GetGovernanceSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp struct {
		QuotaStates []map[string]interface{} `json:"quotaStates"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.QuotaStates) != 1 {
		t.Fatalf("quotaStates want=1 got=%d", len(resp.QuotaStates))
	}
	qs := resp.QuotaStates[0]
	if qs["statePresent"] != true {
		t.Errorf("statePresent want=true got=%v", qs["statePresent"])
	}
	if qs["currentValue"] != float64(12.5) {
		t.Errorf("currentValue want=12.5 got=%v", qs["currentValue"])
	}
}
