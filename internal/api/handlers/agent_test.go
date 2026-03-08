// Task 3.7: Agent Runtime handler tests
package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent/agents"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// mockKnowledgeSearchHandler is a no-op search mock for handler tests.
type mockKnowledgeSearchHandler struct{}

func (m *mockKnowledgeSearchHandler) HybridSearch(_ context.Context, _ knowledge.SearchInput) (*knowledge.SearchResults, error) {
	return &knowledge.SearchResults{}, nil
}

type mockLLMProviderHandler struct{}

func (m *mockLLMProviderHandler) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "Hola, ¿agendamos una llamada breve?", Tokens: 24}, nil
}

func (m *mockLLMProviderHandler) Embed(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{}, nil
}

func (m *mockLLMProviderHandler) ModelInfo() llm.ModelMeta {
	return llm.ModelMeta{ID: "test", Provider: "mock", Version: "v1", MaxTokens: 1024}
}

func (m *mockLLMProviderHandler) HealthCheck(_ context.Context) error { return nil }

type mockKBToolExecutorHandler struct {
	out json.RawMessage
	err error
}

func (m *mockKBToolExecutorHandler) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.out, nil
}

// newTestSupportAgentHandler builds a SupportAgentHandler backed by an in-memory DB.
func newTestSupportAgentHandler(t *testing.T) *SupportAgentHandler {
	t.Helper()
	db := mustOpenDBWithMigrations(t)
	orch := agent.NewOrchestrator(db)
	reg := tool.NewToolRegistry(db)
	sa := agents.NewSupportAgent(orch, reg, &mockKnowledgeSearchHandler{})
	return NewSupportAgentHandler(sa)
}

// newTestProspectingAgentHandler builds a ProspectingAgentHandler backed by an in-memory DB.
func newTestProspectingAgentHandler(t *testing.T, db *sql.DB, wsID, ownerID string) (*ProspectingAgentHandler, string) {
	t.Helper()
	orch := agent.NewOrchestrator(db)
	reg := tool.NewToolRegistry(db)

	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('prospecting-agent', ?, 'Prospecting Agent', 'prospecting', 'active')`, wsID)
	if err != nil {
		t.Fatalf("insert prospecting agent_definition: %v", err)
	}

	accountSvc := crm.NewAccountService(db)
	leadSvc := crm.NewLeadService(db)
	acc, err := accountSvc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Prospect Corp",
		OwnerID:     ownerID,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	lead, err := leadSvc.Create(context.Background(), crm.CreateLeadInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		AccountID:   acc.ID,
		Status:      "new",
	})
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}
	if regErr := reg.Register(tool.BuiltinGetLead, tool.NewGetLeadExecutor(leadSvc)); regErr != nil {
		t.Fatalf("register get_lead executor: %v", regErr)
	}
	if regErr := reg.Register(tool.BuiltinGetAccount, tool.NewGetAccountExecutor(accountSvc)); regErr != nil {
		t.Fatalf("register get_account executor: %v", regErr)
	}
	if regErr := reg.Register(tool.BuiltinCreateTask, tool.NewCreateTaskExecutor(db)); regErr != nil {
		t.Fatalf("register create_task executor: %v", regErr)
	}

	pa := agents.NewProspectingAgent(orch, reg, &mockKnowledgeSearchHandler{}, &mockLLMProviderHandler{}, leadSvc, accountSvc, db)
	return NewProspectingAgentHandler(pa), lead.ID
}

// newTestKBAgentHandler builds a KBAgentHandler backed by an in-memory DB.
func newTestKBAgentHandler(t *testing.T, db *sql.DB, wsID, ownerID string) (*KBAgentHandler, string) {
	t.Helper()
	orch := agent.NewOrchestrator(db)
	reg := tool.NewToolRegistry(db)

	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('kb-agent', ?, 'KB Agent', 'kb', 'active')`, wsID)
	if err != nil {
		t.Fatalf("insert kb agent_definition: %v", err)
	}

	if regErr := reg.Register(tool.BuiltinCreateKnowledgeItem, &mockKBToolExecutorHandler{out: json.RawMessage(`{"knowledge_item_id":"kb-created"}`)}); regErr != nil {
		t.Fatalf("register create_knowledge_item executor: %v", regErr)
	}
	if regErr := reg.Register(tool.BuiltinUpdateKnowledgeItem, &mockKBToolExecutorHandler{out: json.RawMessage(`{"knowledge_item_id":"kb-updated"}`)}); regErr != nil {
		t.Fatalf("register update_knowledge_item executor: %v", regErr)
	}

	caseSvc := crm.NewCaseService(db)
	caseTicket, caseErr := caseSvc.Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		OwnerID:     ownerID,
		Subject:     "Caso resuelto",
		Status:      "resolved",
	})
	if caseErr != nil {
		t.Fatalf("create case: %v", caseErr)
	}

	kbAgent := agents.NewKBAgent(orch, reg, &mockKnowledgeSearchHandler{}, &mockLLMProviderHandler{}, caseSvc, db)
	return NewKBAgentHandler(kbAgent), caseTicket.ID
}

// newTestInsightsAgentHandler builds an InsightsAgentHandler backed by an in-memory DB.
func newTestInsightsAgentHandler(t *testing.T, db *sql.DB, wsID string) *InsightsAgentHandler {
	t.Helper()
	orch := agent.NewOrchestrator(db)
	reg := tool.NewToolRegistry(db)
	if regErr := reg.Register(tool.BuiltinQueryMetrics, tool.NewQueryMetricsExecutor(db)); regErr != nil {
		t.Fatalf("register query_metrics executor: %v", regErr)
	}

	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('insights-agent', ?, 'Insights Agent', 'insights', 'active')`, wsID)
	if err != nil {
		t.Fatalf("insert insights agent_definition: %v", err)
	}

	insightsAgent := agents.NewInsightsAgent(orch, reg, &mockKnowledgeSearchHandler{}, db)
	return NewInsightsAgentHandler(insightsAgent)
}

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

// TestAgentHandler_TriggerAgent_NotActive returns 400 for paused agent.
// Traces: FR-230
func TestAgentHandler_TriggerAgent_NotActive(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)

	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('agent-paused', ?, 'Paused', 'support', 'paused')`, wsID)
	if err != nil {
		t.Fatalf("insert paused agent: %v", err)
	}

	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	body, _ := json.Marshal(map[string]any{"agent_id": "agent-paused", "trigger_type": "manual"})
	req := httptest.NewRequest(http.MethodPost, "/agents/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for paused agent, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAgentHandler_TriggerAgent_InvalidTriggerType returns 400.
// Traces: FR-230
func TestAgentHandler_TriggerAgent_InvalidTriggerType(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	insertTestAgentDef(t, db, "agent-tt", wsID)

	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)

	body, _ := json.Marshal(map[string]any{"agent_id": "agent-tt", "trigger_type": "bad-type"})
	req := httptest.NewRequest(http.MethodPost, "/agents/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid trigger type, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAgentHandler_GetAgentRun_Success returns 200 for existing run.
// Traces: FR-230
func TestAgentHandler_GetAgentRun_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	insertTestAgentDef(t, db, "agent-gr", wsID)

	orch := agent.NewOrchestrator(db)
	run, err := orch.TriggerAgent(context.Background(), agent.TriggerAgentInput{
		AgentID:     "agent-gr",
		WorkspaceID: wsID,
		TriggerType: agent.TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent: %v", err)
	}

	h := NewAgentHandler(orch)
	r := chi.NewRouter()
	r.Get("/agents/runs/{id}", h.GetAgentRun)

	req := httptest.NewRequest(http.MethodGet, "/agents/runs/"+run.ID, nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAgentHandler_GetAgentRun_WithCompletedAt verifies agentRunToResponse covers the non-nil CompletedAt branch.
func TestAgentHandler_GetAgentRun_WithCompletedAt(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	insertTestAgentDef(t, db, "agent-completed", wsID)

	runID := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO agent_run (id, workspace_id, agent_definition_id, trigger_type, status, started_at, completed_at, created_at)
		VALUES (?, ?, 'agent-completed', 'manual', 'success', ?, ?, ?)
	`, runID, wsID, now, now, now)
	if err != nil {
		t.Fatalf("insert agent_run: %v", err)
	}

	orch := agent.NewOrchestrator(db)
	h := NewAgentHandler(orch)
	r := chi.NewRouter()
	r.Get("/agents/runs/{id}", h.GetAgentRun)

	req := httptest.NewRequest(http.MethodGet, "/agents/runs/"+runID, nil)
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T: %v", resp["data"], resp)
	}
	if data["completedAt"] == nil {
		t.Fatalf("expected completedAt in response data, got nil; data=%v", data)
	}
}

// TestSupportAgentHandler_TriggerSupportAgent_MissingWorkspace returns 401.
// Traces: FR-230, FR-231
func TestSupportAgentHandler_TriggerSupportAgent_MissingWorkspace(t *testing.T) {
	t.Parallel()

	h := newTestSupportAgentHandler(t)

	body, _ := json.Marshal(map[string]any{"case_id": "c1", "customer_query": "help"})
	req := httptest.NewRequest(http.MethodPost, "/agents/support/trigger", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerSupportAgent(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestSupportAgentHandler_TriggerSupportAgent_MissingCaseID returns 400.
// Traces: FR-230, FR-231
func TestSupportAgentHandler_TriggerSupportAgent_MissingCaseID(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	_ = db
	h := newTestSupportAgentHandler(t)

	body, _ := json.Marshal(map[string]any{"customer_query": "help"})
	req := httptest.NewRequest(http.MethodPost, "/agents/support/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerSupportAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestSupportAgentHandler_TriggerSupportAgent_MissingQuery returns 400.
// Traces: FR-230, FR-231
func TestSupportAgentHandler_TriggerSupportAgent_MissingQuery(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	_ = db
	h := newTestSupportAgentHandler(t)

	body, _ := json.Marshal(map[string]any{"case_id": "c1"})
	req := httptest.NewRequest(http.MethodPost, "/agents/support/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerSupportAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestSupportAgentHandler_TriggerSupportAgent_InvalidJSON returns 400.
// Traces: FR-230, FR-231
func TestSupportAgentHandler_TriggerSupportAgent_InvalidJSON(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	_ = db
	h := newTestSupportAgentHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/agents/support/trigger", bytes.NewReader([]byte("not-json")))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerSupportAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestProspectingAgentHandler_TriggerProspecting_200 validates 4.5b handler success path.
// Traces: FR-231
func TestProspectingAgentHandler_TriggerProspecting_200(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	ownerID := createUser(t, db, wsID)
	h, leadID := newTestProspectingAgentHandler(t, db, wsID, ownerID)

	body, _ := json.Marshal(map[string]any{"lead_id": leadID, "language": "es"})
	req := httptest.NewRequest(http.MethodPost, "/agents/prospecting/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerProspectingAgent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["run_id"]; !ok {
		t.Fatalf("expected run_id field, got: %v", resp)
	}
	if got, _ := resp["status"].(string); got != "queued" {
		t.Fatalf("expected status=queued, got=%q", got)
	}
	if got, _ := resp["agent"].(string); got != "prospecting" {
		t.Fatalf("expected agent=prospecting, got=%q", got)
	}
}

func TestProspectingAgentHandler_TriggerProspecting_MissingLeadID(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	ownerID := createUser(t, db, wsID)
	h, _ := newTestProspectingAgentHandler(t, db, wsID, ownerID)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/agents/prospecting/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerProspectingAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestProspectingAgentHandler_TriggerProspecting_PropagatesTriggeredByUser(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	ownerID := createUser(t, db, wsID)
	h, leadID := newTestProspectingAgentHandler(t, db, wsID, ownerID)

	body, _ := json.Marshal(map[string]any{"lead_id": leadID, "language": "es"})
	req := httptest.NewRequest(http.MethodPost, "/agents/prospecting/trigger", bytes.NewReader(body))
	ctx := contextWithWorkspaceID(req.Context(), wsID)
	ctx = context.WithValue(ctx, ctxkeys.UserID, ownerID)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerProspectingAgent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.RunID == "" {
		t.Fatal("expected run_id in response")
	}

	var triggeredBy sql.NullString
	err := db.QueryRowContext(context.Background(), `
		SELECT triggered_by_user_id
		FROM agent_run
		WHERE id = ?
	`, resp.RunID).Scan(&triggeredBy)
	if err != nil {
		t.Fatalf("query triggered_by_user_id: %v", err)
	}
	if !triggeredBy.Valid || triggeredBy.String != ownerID {
		t.Fatalf("expected triggered_by_user_id=%s, got valid=%v value=%q", ownerID, triggeredBy.Valid, triggeredBy.String)
	}
}

func TestKBAgentHandler_TriggerKB_200(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	ownerID := createUser(t, db, wsID)
	h, caseID := newTestKBAgentHandler(t, db, wsID, ownerID)

	body, _ := json.Marshal(map[string]any{"case_id": caseID})
	req := httptest.NewRequest(http.MethodPost, "/agents/kb/trigger", bytes.NewReader(body))
	ctx := contextWithWorkspaceID(req.Context(), wsID)
	ctx = context.WithValue(ctx, ctxkeys.UserID, ownerID)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerKBAgent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["run_id"]; !ok {
		t.Fatalf("expected run_id field, got: %v", resp)
	}
	if got, _ := resp["status"].(string); got != "queued" {
		t.Fatalf("expected status=queued, got=%q", got)
	}
	if got, _ := resp["agent"].(string); got != "kb" {
		t.Fatalf("expected agent=kb, got=%q", got)
	}

	runID, _ := resp["run_id"].(string)
	if runID == "" {
		t.Fatal("expected run_id in response")
	}
	var triggeredBy sql.NullString
	err := db.QueryRowContext(context.Background(), `
		SELECT triggered_by_user_id
		FROM agent_run
		WHERE id = ?
	`, runID).Scan(&triggeredBy)
	if err != nil {
		t.Fatalf("query triggered_by_user_id: %v", err)
	}
	if !triggeredBy.Valid || triggeredBy.String != ownerID {
		t.Fatalf("expected triggered_by_user_id=%s, got valid=%v value=%q", ownerID, triggeredBy.Valid, triggeredBy.String)
	}
}

func TestKBAgentHandler_TriggerKB_MissingCaseID(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	ownerID := createUser(t, db, wsID)
	h, _ := newTestKBAgentHandler(t, db, wsID, ownerID)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/agents/kb/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerKBAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestInsightsAgentHandler_TriggerInsights_200(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandler(t, db, wsID)

	body, _ := json.Marshal(map[string]any{"query": "cuántos deals", "language": "es"})
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerInsightsAgent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["run_id"]; !ok {
		t.Fatalf("expected run_id field, got: %v", resp)
	}
	if got, _ := resp["status"].(string); got != "queued" {
		t.Fatalf("expected status=queued, got=%q", got)
	}
	if got, _ := resp["agent"].(string); got != "insights" {
		t.Fatalf("expected agent=insights, got=%q", got)
	}
}

func TestInsightsAgentHandler_TriggerInsights_MissingQuery(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandler(t, db, wsID)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerInsightsAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestHandleProspectingRunError_LeadNotFound verifies handleProspectingRunError maps ErrLeadNotFound → 404.
func TestHandleProspectingRunError_LeadNotFound(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleProspectingRunError(rr, agents.ErrLeadNotFound)
	if !handled {
		t.Fatal("expected handled=true for ErrLeadNotFound")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

// TestHandleProspectingRunError_LeadIDRequired verifies handleProspectingRunError maps ErrLeadIDRequired → 400.
func TestHandleProspectingRunError_LeadIDRequired(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleProspectingRunError(rr, agents.ErrLeadIDRequired)
	if !handled {
		t.Fatal("expected handled=true for ErrLeadIDRequired")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// TestHandleProspectingRunError_DailyLeadLimit verifies handleProspectingRunError maps limit exceeded → 429.
func TestHandleProspectingRunError_DailyLeadLimit(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleProspectingRunError(rr, agents.ErrProspectingDailyLeadLimitExceeded)
	if !handled {
		t.Fatal("expected handled=true for ErrProspectingDailyLeadLimitExceeded")
	}
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
}

// TestHandleProspectingRunError_DailyCostLimit verifies handleProspectingRunError maps cost limit exceeded → 429.
func TestHandleProspectingRunError_DailyCostLimit(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleProspectingRunError(rr, agents.ErrProspectingDailyCostLimitExceeded)
	if !handled {
		t.Fatal("expected handled=true for ErrProspectingDailyCostLimitExceeded")
	}
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
}

// TestHandleProspectingRunError_Unknown verifies handleProspectingRunError returns false for unknown errors.
func TestHandleProspectingRunError_Unknown(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleProspectingRunError(rr, errors.New("unexpected error"))
	if handled {
		t.Fatal("expected handled=false for unknown error")
	}
}

// TestHandleKBRunError_CaseNotFound verifies handleKBRunError maps ErrCaseNotFound → 404.
func TestHandleKBRunError_CaseNotFound(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleKBRunError(rr, agents.ErrCaseNotFound)
	if !handled {
		t.Fatal("expected handled=true for ErrCaseNotFound")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

// TestHandleKBRunError_CaseIDRequired verifies handleKBRunError maps ErrKBCaseIDRequired → 400.
func TestHandleKBRunError_CaseIDRequired(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleKBRunError(rr, agents.ErrKBCaseIDRequired)
	if !handled {
		t.Fatal("expected handled=true for ErrKBCaseIDRequired")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// TestHandleKBRunError_CaseNotResolved verifies handleKBRunError maps ErrCaseNotResolved → 422.
func TestHandleKBRunError_CaseNotResolved(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleKBRunError(rr, agents.ErrCaseNotResolved)
	if !handled {
		t.Fatal("expected handled=true for ErrCaseNotResolved")
	}
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rr.Code)
	}
}

// TestHandleKBRunError_DailyLimit verifies handleKBRunError maps ErrKBDailyLimitExceeded → 429.
func TestHandleKBRunError_DailyLimit(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleKBRunError(rr, agents.ErrKBDailyLimitExceeded)
	if !handled {
		t.Fatal("expected handled=true for ErrKBDailyLimitExceeded")
	}
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
}

// TestHandleKBRunError_Unknown verifies handleKBRunError returns false for unknown errors.
func TestHandleKBRunError_Unknown(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleKBRunError(rr, errors.New("unexpected error"))
	if handled {
		t.Fatal("expected handled=false for unknown error")
	}
}

// TestHandleInsightsRunError_QueryRequired verifies handleInsightsRunError maps ErrInsightsQueryRequired → 400.
func TestHandleInsightsRunError_QueryRequired(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleInsightsRunError(rr, agents.ErrInsightsQueryRequired)
	if !handled {
		t.Fatal("expected handled=true for ErrInsightsQueryRequired")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// TestHandleInsightsRunError_DailyLimit verifies handleInsightsRunError maps ErrInsightsDailyLimitExceeded → 429.
func TestHandleInsightsRunError_DailyLimit(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleInsightsRunError(rr, agents.ErrInsightsDailyLimitExceeded)
	if !handled {
		t.Fatal("expected handled=true for ErrInsightsDailyLimitExceeded")
	}
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
}

// TestInsightsAgentHandler_TriggerInsights_WithUserID verifies withInsightsTriggeredBy is exercised.
func TestInsightsAgentHandler_TriggerInsights_WithUserID(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	ownerID := createUser(t, db, wsID)
	h := newTestInsightsAgentHandler(t, db, wsID)

	body, _ := json.Marshal(map[string]any{"query": "cuántos deals"})
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	ctx := contextWithWorkspaceID(req.Context(), wsID)
	// NO userID set — exercises withInsightsTriggeredBy empty-userID branch
	_ = ownerID
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerInsightsAgent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestInsightsAgentHandler_TriggerInsights_WithNonEmptyUserID exercises the non-empty userID branch of withInsightsTriggeredBy.
func TestInsightsAgentHandler_TriggerInsights_WithNonEmptyUserID(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	ownerID := createUser(t, db, wsID)
	h := newTestInsightsAgentHandler(t, db, wsID)

	body, _ := json.Marshal(map[string]any{"query": "cuántos deals"})
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	ctx := contextWithWorkspaceID(req.Context(), wsID)
	ctx = context.WithValue(ctx, ctxkeys.UserID, ownerID)
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerInsightsAgent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestHandleInsightsRunError_Unknown verifies handleInsightsRunError returns false for unknown errors.
func TestHandleInsightsRunError_Unknown(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	handled := handleInsightsRunError(rr, errors.New("some error"))
	if handled {
		t.Fatal("expected handled=false for unknown error")
	}
}

// TestParseDateTimeValue_Valid verifies parseDateTimeValue parses a valid RFC3339 string.
func TestParseDateTimeValue_Valid(t *testing.T) {
	t.Parallel()

	result, err := parseDateTimeValue("2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Year() != 2026 {
		t.Fatalf("expected year 2026, got %d", result.Year())
	}
}

// TestParseDateTimeValue_Invalid verifies parseDateTimeValue returns error for bad input.
func TestParseDateTimeValue_Invalid(t *testing.T) {
	t.Parallel()

	result, err := parseDateTimeValue("not-a-date")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}
}

// TestInsightsAgentHandler_TriggerInsights_InvalidDateFrom verifies date_from parsing error → 400.
func TestInsightsAgentHandler_TriggerInsights_InvalidDateFrom(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandler(t, db, wsID)

	body, _ := json.Marshal(map[string]any{"query": "cuántos deals", "date_from": "not-a-date"})
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerInsightsAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad date_from, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestInsightsAgentHandler_TriggerInsights_InvalidDateTo verifies date_to parsing error → 400.
func TestInsightsAgentHandler_TriggerInsights_InvalidDateTo(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandler(t, db, wsID)

	body, _ := json.Marshal(map[string]any{"query": "cuántos deals", "date_to": "bad"})
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerInsightsAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad date_to, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestInsightsAgentHandler_TriggerInsights_DailyLimitExceeded(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandler(t, db, wsID)

	ctx := context.Background()
	for i := range 100 {
		_, err := db.ExecContext(ctx, `
			INSERT INTO agent_run (
				id, workspace_id, agent_definition_id, trigger_type, status,
				started_at, created_at
			) VALUES (?, ?, 'insights-agent', 'manual', 'success', datetime('now'), datetime('now'))
		`, uuid.NewV7().String(), wsID)
		if err != nil {
			t.Fatalf("insert agent_run #%d: %v", i, err)
		}
	}

	body, _ := json.Marshal(map[string]any{"query": "cuántos deals"})
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerInsightsAgent(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestSupportAgentHandler_TriggerSupportAgent_RunError verifies 500 when agent.Run fails.
func TestSupportAgentHandler_TriggerSupportAgent_RunError(t *testing.T) {
	t.Parallel()

	h := newTestSupportAgentHandler(t)

	body, _ := json.Marshal(map[string]any{"case_id": "case-1", "customer_query": "how do I reset my password?"})
	req := httptest.NewRequest(http.MethodPost, "/agents/support/trigger", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Use a workspace that doesn't exist so the agent.Run fails with a DB error.
	req = req.WithContext(contextWithWorkspaceID(req.Context(), "ws-nonexistent"))
	rr := httptest.NewRecorder()

	h.TriggerSupportAgent(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
}
