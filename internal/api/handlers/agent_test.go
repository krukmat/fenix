// Task 3.7: Agent Runtime handler tests
package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

// insertTestAgentDef inserts an agent_definition for handler tests.
func insertTestAgentDef(t *testing.T, db *sql.DB, id, wsID string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES (?, ?, 'Test Agent', 'support', 'active')`, id, wsID)
	if err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
}

// TestAgentHandler_TriggerAgent_MissingWorkspace returns 401 without workspace context.
// Traces: FR-230
func TestAgentHandler_TriggerAgent_MissingWorkspace(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	body, _ := json.Marshal(map[string]any{"agent_id": "test", "trigger_type": "manual"})
	req := httptest.NewRequest(http.MethodPost, "/agents/trigger", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerAgent(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAgentHandler_TriggerAgent_MissingAgentID returns 400 without agent_id.
// Traces: FR-230
func TestAgentHandler_TriggerAgent_MissingAgentID(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	body, _ := json.Marshal(map[string]any{"trigger_type": "manual"})
	req := httptest.NewRequest(http.MethodPost, "/agents/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAgentHandler_TriggerAgent_AgentNotFound returns 404 for nonexistent agent.
// Traces: FR-230
func TestAgentHandler_TriggerAgent_AgentNotFound(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	body, _ := json.Marshal(map[string]any{"agent_id": "nonexistent", "trigger_type": "manual"})
	req := httptest.NewRequest(http.MethodPost, "/agents/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerAgent(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAgentHandler_TriggerAgent_Success returns 201 for valid trigger.
// Traces: FR-230, FR-231
func TestAgentHandler_TriggerAgent_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	insertTestAgentDef(t, db, "agent-ok", wsID)

	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	body, _ := json.Marshal(map[string]any{"agent_id": "agent-ok", "trigger_type": "manual"})
	req := httptest.NewRequest(http.MethodPost, "/agents/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerAgent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAgentHandler_GetAgentRun_MissingWorkspace returns 401.
// Traces: FR-230
func TestAgentHandler_GetAgentRun_MissingWorkspace(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	r := chi.NewRouter()
	r.Get("/agents/runs/{id}", h.GetAgentRun)

	req := httptest.NewRequest(http.MethodGet, "/agents/runs/nonexistent", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// TestAgentHandler_GetAgentRun_NotFound returns 404 for unknown run ID.
// Traces: FR-230
func TestAgentHandler_GetAgentRun_NotFound(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	r := chi.NewRouter()
	r.Get("/agents/runs/{id}", h.GetAgentRun)

	req := httptest.NewRequest(http.MethodGet, "/agents/runs/no-such-run", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAgentHandler_ListAgentRuns_MissingWorkspace returns 401.
// Traces: FR-230
func TestAgentHandler_ListAgentRuns_MissingWorkspace(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	req := httptest.NewRequest(http.MethodGet, "/agents/runs", nil)
	rr := httptest.NewRecorder()

	h.ListAgentRuns(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// TestAgentHandler_ListAgentRuns_Success returns 200 with meta.
// Traces: FR-230
func TestAgentHandler_ListAgentRuns_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	req := httptest.NewRequest(http.MethodGet, "/agents/runs", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ListAgentRuns(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["meta"]; !ok {
		t.Error("expected 'meta' key in response")
	}
}

// TestAgentHandler_ListAgentDefinitions_MissingWorkspace returns 401.
// Traces: FR-230
func TestAgentHandler_ListAgentDefinitions_MissingWorkspace(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	req := httptest.NewRequest(http.MethodGet, "/agents/definitions", nil)
	rr := httptest.NewRecorder()

	h.ListAgentDefinitions(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// TestAgentHandler_ListAgentDefinitions_Success returns 200.
// Traces: FR-230
func TestAgentHandler_ListAgentDefinitions_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	req := httptest.NewRequest(http.MethodGet, "/agents/definitions", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()

	h.ListAgentDefinitions(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAgentHandler_CancelAgentRun_MissingWorkspace returns 401.
// Traces: FR-230
func TestAgentHandler_CancelAgentRun_MissingWorkspace(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	r := chi.NewRouter()
	r.Post("/agents/runs/{id}/cancel", h.CancelAgentRun)

	req := httptest.NewRequest(http.MethodPost, "/agents/runs/run-1/cancel", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// TestAgentHandler_CancelAgentRun_NotFound returns 404.
// Traces: FR-230
func TestAgentHandler_CancelAgentRun_NotFound(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	r := chi.NewRouter()
	r.Post("/agents/runs/{id}/cancel", h.CancelAgentRun)

	req := httptest.NewRequest(http.MethodPost, "/agents/runs/no-run/cancel", nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAgentHandler_TriggerAgent_InvalidJSON returns 400.
// Traces: FR-230
func TestAgentHandler_TriggerAgent_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	req := httptest.NewRequest(http.MethodPost, "/agents/trigger", bytes.NewReader([]byte("not-json")))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}
