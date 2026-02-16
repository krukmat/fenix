// Package handlers — Handoff Manager handler tests.
// Task 3.8: Human handoff with evidence context (FR-232).
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

// ── GET /agents/runs/{id}/handoff ────────────────────────────────────────────

// TestHandoffHandler_GetHandoffPackage_MissingWorkspace returns 401 without workspace context.
// Traces: FR-232
func TestHandoffHandler_GetHandoffPackage_MissingWorkspace(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	cs := crm.NewCaseService(db)
	bus := eventbus.New()
	svc := agent.NewHandoffService(db, cs, bus)
	h := NewHandoffHandler(svc)

	r := chi.NewRouter()
	r.Get("/agents/runs/{id}/handoff", h.GetHandoffPackage)

	req := httptest.NewRequest(http.MethodGet, "/agents/runs/run-1/handoff?case_id=case-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestHandoffHandler_GetHandoffPackage_NotFound returns 404 for unknown run.
// Traces: FR-232
func TestHandoffHandler_GetHandoffPackage_NotFound(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	cs := crm.NewCaseService(db)
	bus := eventbus.New()
	svc := agent.NewHandoffService(db, cs, bus)
	h := NewHandoffHandler(svc)

	r := chi.NewRouter()
	r.Get("/agents/runs/{id}/handoff", h.GetHandoffPackage)

	req := httptest.NewRequest(http.MethodGet, "/agents/runs/nonexistent/handoff?case_id=case-1", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestHandoffHandler_GetHandoffPackage_Success returns 200 with HandoffPackage JSON.
// Traces: FR-232
func TestHandoffHandler_GetHandoffPackage_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	cs := crm.NewCaseService(db)
	bus := eventbus.New()
	svc := agent.NewHandoffService(db, cs, bus)
	h := NewHandoffHandler(svc)

	// Insert prerequisites
	ctx := context.Background()
	const agentDefID = "agent-h-get-1"
	const runID = "run-h-get-1"
	const caseID = "case-h-get-1"
	_, _ = db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES (?, ?, 'Test Agent', 'support', 'active')`, agentDefID, wsID)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO agent_run (
			id, workspace_id, agent_definition_id, trigger_type, status,
			retrieval_queries, retrieved_evidence_ids, reasoning_trace, tool_calls,
			output, started_at, created_at
		) VALUES (?, ?, ?, 'manual', 'escalated', '[]', '[]', '[]', '[]', '{}', datetime('now'), datetime('now'))
	`, runID, wsID, agentDefID)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at)
		VALUES (?, ?, ?, 'My Test Case', 'medium', 'open', datetime('now'), datetime('now'))
	`, caseID, wsID, ownerID)

	r := chi.NewRouter()
	r.Get("/agents/runs/{id}/handoff", h.GetHandoffPackage)

	req := httptest.NewRequest(http.MethodGet, "/agents/runs/"+runID+"/handoff?case_id="+caseID, nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'data' object in response, got: %v", resp)
	}
	if data["runId"] != runID {
		t.Errorf("runId: got %v, want %s", data["runId"], runID)
	}
	if data["caseSubject"] != "My Test Case" {
		t.Errorf("caseSubject: got %v, want %s", data["caseSubject"], "My Test Case")
	}
}

// ── POST /agents/runs/{id}/handoff ───────────────────────────────────────────

// TestHandoffHandler_InitiateHandoff_MissingWorkspace returns 401.
// Traces: FR-232
func TestHandoffHandler_InitiateHandoff_MissingWorkspace(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	cs := crm.NewCaseService(db)
	bus := eventbus.New()
	svc := agent.NewHandoffService(db, cs, bus)
	h := NewHandoffHandler(svc)

	r := chi.NewRouter()
	r.Post("/agents/runs/{id}/handoff", h.InitiateHandoff)

	body, _ := json.Marshal(map[string]string{"case_id": "case-1", "reason": "no solution"})
	req := httptest.NewRequest(http.MethodPost, "/agents/runs/run-1/handoff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestHandoffHandler_InitiateHandoff_RunNotFound returns 404 for unknown run.
// Traces: FR-232
func TestHandoffHandler_InitiateHandoff_RunNotFound(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	cs := crm.NewCaseService(db)
	bus := eventbus.New()
	svc := agent.NewHandoffService(db, cs, bus)
	h := NewHandoffHandler(svc)

	r := chi.NewRouter()
	r.Post("/agents/runs/{id}/handoff", h.InitiateHandoff)

	body, _ := json.Marshal(map[string]string{"case_id": "case-1", "reason": "no solution"})
	req := httptest.NewRequest(http.MethodPost, "/agents/runs/nonexistent/handoff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestHandoffHandler_InitiateHandoff_Success returns 200 with HandoffPackage JSON.
// Traces: FR-232
func TestHandoffHandler_InitiateHandoff_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID, ownerID := setupWorkspaceAndOwner(t, db)
	cs := crm.NewCaseService(db)
	bus := eventbus.New()
	svc := agent.NewHandoffService(db, cs, bus)
	h := NewHandoffHandler(svc)

	// Insert prerequisites
	ctx := context.Background()
	const agentDefID = "agent-h-post-1"
	const runID = "run-h-post-1"
	const caseID = "case-h-post-1"
	_, _ = db.ExecContext(ctx,
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES (?, ?, 'Test Agent', 'support', 'active')`, agentDefID, wsID)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO agent_run (
			id, workspace_id, agent_definition_id, trigger_type, status,
			retrieval_queries, retrieved_evidence_ids, reasoning_trace, tool_calls,
			output, started_at, created_at
		) VALUES (?, ?, ?, 'manual', 'escalated', '[]', '[]', '[]', '[]', '{}', datetime('now'), datetime('now'))
	`, runID, wsID, agentDefID)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO case_ticket (id, workspace_id, owner_id, subject, priority, status, created_at, updated_at)
		VALUES (?, ?, ?, 'Escalation Subject', 'high', 'open', datetime('now'), datetime('now'))
	`, caseID, wsID, ownerID)

	r := chi.NewRouter()
	r.Post("/agents/runs/{id}/handoff", h.InitiateHandoff)

	body, _ := json.Marshal(map[string]string{
		"case_id": caseID,
		"reason":  "AI could not find a solution",
	})
	req := httptest.NewRequest(http.MethodPost, "/agents/runs/"+runID+"/handoff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'data' object, got: %v", resp)
	}
	if data["caseStatus"] != "escalated" {
		t.Errorf("caseStatus: got %v, want escalated", data["caseStatus"])
	}
	if data["reason"] != "AI could not find a solution" {
		t.Errorf("reason: got %v", data["reason"])
	}
}
