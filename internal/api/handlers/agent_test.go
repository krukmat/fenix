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
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
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
		 VALUES ('insights-agent', ?, 'insights_agent', 'insights', 'active')`, wsID)
	if err != nil {
		t.Fatalf("insert insights agent_definition: %v", err)
	}

	insightsAgent := agents.NewInsightsAgent(orch, reg, &mockKnowledgeSearchHandler{}, db)
	return NewInsightsAgentHandler(insightsAgent)
}

func newTestInsightsAgentHandlerWithShadow(t *testing.T, db *sql.DB, wsID string) *InsightsAgentHandler {
	t.Helper()
	runnerRegistry := agent.NewRunnerRegistry()
	orch := agent.NewOrchestratorWithRegistry(db, runnerRegistry)
	reg := tool.NewToolRegistry(db)
	if regErr := reg.Register(tool.BuiltinQueryMetrics, tool.NewQueryMetricsExecutor(db)); regErr != nil {
		t.Fatalf("register query_metrics executor: %v", regErr)
	}

	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('insights-agent', ?, 'insights_agent', 'insights', 'active')`, wsID)
	if err != nil {
		t.Fatalf("insert insights agent_definition: %v", err)
	}
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES (?, ?, 'Insights Shadow Agent', 'dsl', 'active')`, defaultInsightsShadowAgentID, wsID)
	if err != nil {
		t.Fatalf("insert insights shadow agent_definition: %v", err)
	}

	workflowSvc := workflowdomain.NewService(db)
	shadowWorkflow, err := workflowSvc.Create(context.Background(), workflowdomain.CreateWorkflowInput{
		WorkspaceID:       wsID,
		AgentDefinitionID: testStringPtr(defaultInsightsShadowAgentID),
		Name:              "insights_shadow_pilot",
		DSLSource: `WORKFLOW insights_shadow_pilot
ON insights.query_received
AGENT insights_agent WITH {"workspace_id": workspace_id, "query": query, "language": language}`,
	})
	if err != nil {
		t.Fatalf("create shadow workflow: %v", err)
	}
	if _, err = workflowSvc.MarkTesting(context.Background(), wsID, shadowWorkflow.ID); err != nil {
		t.Fatalf("mark shadow workflow testing: %v", err)
	}
	if _, err = workflowSvc.Activate(context.Background(), wsID, shadowWorkflow.ID); err != nil {
		t.Fatalf("activate shadow workflow: %v", err)
	}

	insightsAgent := agents.NewInsightsAgent(orch, reg, &mockKnowledgeSearchHandler{}, db)
	dslRunner := agent.NewDSLRunner(db)
	if err := runnerRegistry.Register(agents.AgentTypeInsights, &agents.InsightsRunner{Agent: insightsAgent}); err != nil {
		t.Fatalf("register insights runner: %v", err)
	}
	if err := agents.RegisterDSLRunner(runnerRegistry, dslRunner); err != nil {
		t.Fatalf("register dsl runner: %v", err)
	}
	return NewInsightsAgentHandlerWithShadow(insightsAgent, dslRunner, orch, reg, db)
}

func testStringPtr(v string) *string { return &v }

func setWorkspaceSettings(t *testing.T, db *sql.DB, workspaceID string, settings string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		UPDATE workspace
		SET settings = ?, updated_at = datetime('now')
		WHERE id = ?
	`, settings, workspaceID)
	if err != nil {
		t.Fatalf("set workspace settings: %v", err)
	}
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

func TestInsightsAgentHandler_TriggerInsights_DeclarativeRolloutSegment(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	setWorkspaceSettings(t, db, wsID, `{
		"agent_spec": {
			"pilots": {
				"insights": {
					"enabled": true,
					"mode": "declarative",
					"shadow_agent_id": "insights-shadow-agent"
				}
			}
		}
	}`)
	h := newTestInsightsAgentHandlerWithShadow(t, db, wsID)

	body, _ := json.Marshal(map[string]any{"query": "cuántos deals", "language": "es"})
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerInsightsAgent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		RunID   string `json:"run_id"`
		Status  string `json:"status"`
		Agent   string `json:"agent"`
		Rollout struct {
			Enabled           bool   `json:"enabled"`
			Selected          bool   `json:"selected"`
			Mode              string `json:"mode"`
			Source            string `json:"source"`
			AgentDefinitionID string `json:"agent_definition_id"`
			EffectiveRunID    string `json:"effective_run_id"`
			EffectiveStatus   string `json:"effective_status"`
		} `json:"rollout"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.RunID == "" || resp.Status != "queued" || resp.Agent != "insights" {
		t.Fatalf("unexpected rollout response: %+v", resp)
	}
	if !resp.Rollout.Enabled || !resp.Rollout.Selected {
		t.Fatalf("expected active rollout, got %+v", resp.Rollout)
	}
	if resp.Rollout.Mode != "declarative_primary" || resp.Rollout.Source != "workspace.settings" {
		t.Fatalf("unexpected rollout metadata: %+v", resp.Rollout)
	}
	if resp.Rollout.EffectiveRunID == "" {
		t.Fatalf("expected effective run id, got %+v", resp.Rollout)
	}

	var definitionID string
	err := db.QueryRowContext(context.Background(), `
		SELECT agent_definition_id
		FROM agent_run
		WHERE id = ?
	`, resp.RunID).Scan(&definitionID)
	if err != nil {
		t.Fatalf("load primary rollout run: %v", err)
	}
	if definitionID != defaultInsightsShadowAgentID {
		t.Fatalf("expected declarative primary run on %q, got %q", defaultInsightsShadowAgentID, definitionID)
	}
}

func TestInsightsAgentHandler_TriggerInsights_RollbackToGoSegment(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandlerWithShadow(t, db, wsID)

	setWorkspaceSettings(t, db, wsID, `{
		"agent_spec": {
			"pilots": {
				"insights": {
					"enabled": true,
					"mode": "declarative",
					"shadow_agent_id": "insights-shadow-agent"
				}
			}
		}
	}`)

	body, _ := json.Marshal(map[string]any{"query": "cuántos deals", "language": "es"})
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.TriggerInsightsAgent(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("declarative leg expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var firstResp struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &firstResp); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	var firstDefinitionID string
	if err := db.QueryRowContext(context.Background(), `
		SELECT agent_definition_id
		FROM agent_run
		WHERE id = ?
	`, firstResp.RunID).Scan(&firstDefinitionID); err != nil {
		t.Fatalf("load first run definition: %v", err)
	}
	if firstDefinitionID != defaultInsightsShadowAgentID {
		t.Fatalf("expected declarative first leg on %q, got %q", defaultInsightsShadowAgentID, firstDefinitionID)
	}

	setWorkspaceSettings(t, db, wsID, `{
		"agent_spec": {
			"pilots": {
				"insights": {
					"enabled": true,
					"mode": "go",
					"shadow_agent_id": "insights-shadow-agent"
				}
			}
		}
	}`)

	req = httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.TriggerInsightsAgent(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("rollback leg expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var secondResp struct {
		RunID   string `json:"run_id"`
		Status  string `json:"status"`
		Agent   string `json:"agent"`
		Rollout struct {
			Enabled           bool   `json:"enabled"`
			Selected          bool   `json:"selected"`
			Mode              string `json:"mode"`
			Source            string `json:"source"`
			AgentDefinitionID string `json:"agent_definition_id"`
			EffectiveRunID    string `json:"effective_run_id"`
			EffectiveStatus   string `json:"effective_status"`
		} `json:"rollout"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &secondResp); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if secondResp.RunID == "" || secondResp.Status != "queued" || secondResp.Agent != "insights" {
		t.Fatalf("unexpected rollback response: %+v", secondResp)
	}
	if !secondResp.Rollout.Enabled || secondResp.Rollout.Selected {
		t.Fatalf("expected go rollback rollout metadata, got %+v", secondResp.Rollout)
	}
	if secondResp.Rollout.Mode != "go_primary" || secondResp.Rollout.Source != "workspace.settings" {
		t.Fatalf("unexpected rollback rollout metadata: %+v", secondResp.Rollout)
	}

	var secondDefinitionID string
	if err := db.QueryRowContext(context.Background(), `
		SELECT agent_definition_id
		FROM agent_run
		WHERE id = ?
	`, secondResp.RunID).Scan(&secondDefinitionID); err != nil {
		t.Fatalf("load second run definition: %v", err)
	}
	if secondDefinitionID != "insights-agent" {
		t.Fatalf("expected rollback to Go agent definition %q, got %q", "insights-agent", secondDefinitionID)
	}
}

func TestInsightsAgentHandler_TriggerInsights_ShadowMode(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandlerWithShadow(t, db, wsID)

	body, _ := json.Marshal(map[string]any{
		"query":       "cuántos deals",
		"language":    "es",
		"shadow_mode": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", bytes.NewReader(body))
	req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TriggerInsightsAgent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		RunID  string `json:"run_id"`
		Status string `json:"status"`
		Agent  string `json:"agent"`
		Shadow struct {
			Enabled           bool   `json:"enabled"`
			RunID             string `json:"run_id"`
			EffectiveRunID    string `json:"effective_run_id"`
			Status            string `json:"status"`
			AgentDefinitionID string `json:"agent_definition_id"`
			Error             string `json:"error"`
			Comparison        struct {
				PrimaryRunID         string `json:"primary_run_id"`
				ShadowRunID          string `json:"shadow_run_id"`
				EffectiveShadowRunID string `json:"effective_shadow_run_id"`
				Matched              bool   `json:"matched"`
				Differences          []struct {
					Check    string `json:"check"`
					Severity string `json:"severity"`
				} `json:"differences"`
			} `json:"comparison"`
		} `json:"shadow"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.RunID == "" || resp.Agent != "insights" || resp.Status != "queued" {
		t.Fatalf("unexpected primary response: %+v", resp)
	}
	if !resp.Shadow.Enabled {
		t.Fatal("expected shadow enabled=true")
	}
	if resp.Shadow.RunID == "" {
		t.Fatalf("expected shadow run_id, got response=%s", rr.Body.String())
	}
	if resp.Shadow.Error != "" {
		t.Fatalf("unexpected shadow error: %s", resp.Shadow.Error)
	}
	if resp.Shadow.Comparison.PrimaryRunID != resp.RunID {
		t.Fatalf("comparison.primary_run_id = %q, want %q", resp.Shadow.Comparison.PrimaryRunID, resp.RunID)
	}
	if resp.Shadow.Comparison.ShadowRunID != resp.Shadow.RunID {
		t.Fatalf("comparison.shadow_run_id = %q, want %q", resp.Shadow.Comparison.ShadowRunID, resp.Shadow.RunID)
	}
	if resp.Shadow.EffectiveRunID == "" {
		t.Fatal("expected shadow effective_run_id")
	}
	if resp.Shadow.Comparison.EffectiveShadowRunID == "" {
		t.Fatal("expected effective shadow run id")
	}
	if !resp.Shadow.Comparison.Matched {
		t.Fatalf("expected matched comparison, got %+v", resp.Shadow.Comparison.Differences)
	}

	var triggerContextRaw string
	err := db.QueryRowContext(context.Background(), `
		SELECT trigger_context
		FROM agent_run
		WHERE id = ?
	`, resp.Shadow.RunID).Scan(&triggerContextRaw)
	if err != nil {
		t.Fatalf("load shadow run trigger_context: %v", err)
	}
	var triggerContext map[string]any
	if err := json.Unmarshal([]byte(triggerContextRaw), &triggerContext); err != nil {
		t.Fatalf("decode shadow trigger_context: %v", err)
	}
	if got, _ := triggerContext["shadow_of_run_id"].(string); got != resp.RunID {
		t.Fatalf("shadow_of_run_id = %q, want %q", got, resp.RunID)
	}
	if got, _ := triggerContext["pilot"].(string); got != "insights" {
		t.Fatalf("pilot = %q, want insights", got)
	}
}

func TestInsightsAgentHandler_TriggerInsights_ShadowMode_NotConfigured(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandler(t, db, wsID)

	body, _ := json.Marshal(map[string]any{
		"query":       "cuántos deals",
		"shadow_mode": true,
	})
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
	shadow, ok := resp["shadow"].(map[string]any)
	if !ok {
		t.Fatalf("expected shadow object, got %T", resp["shadow"])
	}
	if got, _ := shadow["error"].(string); got == "" {
		t.Fatalf("expected shadow error in response, got %v", shadow)
	}
}

func TestHandleInsightsRunError(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	if !handleInsightsRunError(rr, agents.ErrInsightsQueryRequired) {
		t.Fatal("expected query-required error to be handled")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	if !handleInsightsRunError(rr, agents.ErrInsightsDailyLimitExceeded) {
		t.Fatal("expected daily-limit error to be handled")
	}
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	if handleInsightsRunError(rr, errors.New("other")) {
		t.Fatal("expected unrelated error to be unhandled")
	}
}

func TestBuildInsightsPrimaryResponseAndEnrich(t *testing.T) {
	t.Parallel()

	run := &agent.Run{ID: "run-1", DefinitionID: "insights-agent", Status: agent.StatusSuccess}
	resp := buildInsightsPrimaryResponse(run)
	if got := resp["run_id"]; got != "run-1" {
		t.Fatalf("run_id = %v, want run-1", got)
	}

	rollout := insightsRolloutConfig{
		Enabled:            true,
		DeclarativePrimary: false,
		Source:             "workspace.settings",
	}
	shadow := map[string]any{"enabled": true, "run_id": "shadow-1"}
	enrichInsightsPrimaryResponse(resp, rollout, run, true, shadow)

	rolloutResp, ok := resp["rollout"].(map[string]any)
	if !ok {
		t.Fatalf("expected rollout map, got %T", resp["rollout"])
	}
	if rolloutResp["mode"] != "go_primary" {
		t.Fatalf("mode = %v, want go_primary", rolloutResp["mode"])
	}
	if _, ok := resp["shadow"].(map[string]any); !ok {
		t.Fatalf("expected shadow map, got %T", resp["shadow"])
	}
}

func TestBuildInsightsShadowPayload_Disabled(t *testing.T) {
	t.Parallel()

	got := buildInsightsShadowPayload(nil, httptest.NewRequest(http.MethodPost, "/", nil), agents.InsightsAgentConfig{}, insightsAgentRequest{}, &agent.Run{ID: "run"})
	if got != nil {
		t.Fatalf("expected nil shadow payload, got %v", got)
	}
}

func TestBuildInsightsShadowPayload_Enabled(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandlerWithShadow(t, db, wsID)
	primary := &agent.Run{ID: "primary-run"}

	got := buildInsightsShadowPayload(
		h,
		httptest.NewRequest(http.MethodPost, "/", nil),
		agents.InsightsAgentConfig{WorkspaceID: wsID, Query: "cuantos deals", Language: "es"},
		insightsAgentRequest{ShadowMode: true},
		primary,
	)
	if got == nil {
		t.Fatal("expected shadow payload")
	}
	if got["run_id"] == "" {
		t.Fatalf("expected shadow run_id, got %#v", got)
	}
}

func TestInsightsAgentHandler_RunInsightsPrimary_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandler(t, db, wsID)

	run, ok := h.runInsightsPrimary(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil), agents.InsightsAgentConfig{
		WorkspaceID: wsID,
		Query:       "cuantos deals",
		Language:    "es",
	})
	if !ok || run == nil {
		t.Fatalf("expected primary run success, got ok=%v run=%v", ok, run)
	}
}

func TestInsightsAgentHandler_TriggerInsightsDeclarativePrimary_NotConfigured(t *testing.T) {
	t.Parallel()

	h := &InsightsAgentHandler{}
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", nil)
	rr := httptest.NewRecorder()

	h.triggerInsightsDeclarativePrimary(rr, req, agents.InsightsAgentConfig{
		WorkspaceID: "ws",
		Query:       "cuantos deals",
		Language:    "es",
	}, insightsRolloutConfig{Enabled: true, DeclarativePrimary: true, AgentID: defaultInsightsShadowAgentID})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestInsightsAgentHandler_TriggerInsightsDeclarativePrimary_Success(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	h := newTestInsightsAgentHandlerWithShadow(t, db, wsID)
	req := httptest.NewRequest(http.MethodPost, "/agents/insights/trigger", nil)
	rr := httptest.NewRecorder()

	h.triggerInsightsDeclarativePrimary(rr, req, agents.InsightsAgentConfig{
		WorkspaceID: wsID,
		Query:       "cuantos deals",
		Language:    "es",
	}, insightsRolloutConfig{
		Enabled:            true,
		DeclarativePrimary: true,
		AgentID:            defaultInsightsShadowAgentID,
		Source:             "workspace.settings",
	})

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if rollout, ok := resp["rollout"].(map[string]any); !ok || rollout["mode"] != insightsPrimaryModeDeclarative {
		t.Fatalf("unexpected rollout payload: %#v", resp["rollout"])
	}
}

func TestBuildInsightsShadowSuccessResponse_UsesFallbackRun(t *testing.T) {
	t.Parallel()

	primary := &agent.Run{ID: "go-run"}
	wrapper := &agent.Run{ID: "dsl-run", DefinitionID: "insights-shadow-agent", Status: agent.StatusSuccess}
	resp := buildInsightsShadowSuccessResponse(primary, &insightsShadowExecution{WrapperRun: wrapper})

	if got := resp["status"]; got != agent.StatusSuccess {
		t.Fatalf("status = %v, want %s", got, agent.StatusSuccess)
	}
	if got := resp["effective_run_id"]; got != "dsl-run" {
		t.Fatalf("effective_run_id = %v, want dsl-run", got)
	}
}

func TestBuildInsightsShadowComparison_ClassifiesMismatch(t *testing.T) {
	t.Parallel()

	primaryCost := 0.01
	shadowCost := 0.02
	report := buildInsightsShadowComparison(context.Background(), nil, "ws", &agent.Run{
		ID:        "go-run",
		Status:    agent.StatusSuccess,
		Output:    json.RawMessage(`{"action":"answer","confidence":"high","evidence_ids":["kb-1"]}`),
		ToolCalls: json.RawMessage(`[{"tool_name":"query_metrics"},{"tool_name":"search_knowledge"}]`),
		TotalCost: &primaryCost,
	}, &agent.Run{
		ID:        "shadow-run",
		Status:    agent.StatusFailed,
		Output:    json.RawMessage(`{"action":"abstain","confidence":"low","evidence_ids":[]}`),
		ToolCalls: json.RawMessage(`[{"tool_name":"query_metrics"}]`),
		TotalCost: &shadowCost,
	})

	if report.Matched {
		t.Fatal("expected mismatched report")
	}
	if len(report.Differences) == 0 {
		t.Fatal("expected differences")
	}
	var hasHigh bool
	for _, diff := range report.Differences {
		if diff.Severity == "high" {
			hasHigh = true
			break
		}
	}
	if !hasHigh {
		t.Fatalf("expected at least one high severity difference, got %+v", report.Differences)
	}
}

func TestBuildInsightsShadowComparison_MatchedRuns(t *testing.T) {
	t.Parallel()

	cost := 0.01
	rawOutput := json.RawMessage(`{"action":"answer","confidence":"high","evidence_ids":["kb-1"]}`)
	rawToolCalls := json.RawMessage(`[{"tool_name":"query_metrics"},{"tool_name":"search_knowledge"}]`)
	report := buildInsightsShadowComparison(context.Background(), nil, "ws", &agent.Run{
		ID:        "go-run",
		Status:    agent.StatusSuccess,
		Output:    rawOutput,
		ToolCalls: rawToolCalls,
		TotalCost: &cost,
	}, &agent.Run{
		ID:        "shadow-run",
		Status:    agent.StatusSuccess,
		Output:    rawOutput,
		ToolCalls: rawToolCalls,
		TotalCost: &cost,
	})

	if !report.Matched {
		t.Fatalf("expected matched report, got %+v", report.Differences)
	}
	if len(report.Differences) != 0 {
		t.Fatalf("expected no differences, got %+v", report.Differences)
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

func TestProspectingAndKBHandlerSuccessPaths(t *testing.T) {
	t.Parallel()

	db := mustOpenDBWithMigrations(t)
	wsID := createWorkspace(t, db)
	ownerID := createUser(t, db, wsID)

	prospectingHandler, leadID := newTestProspectingAgentHandler(t, db, wsID, ownerID)
	prospectingBody, _ := json.Marshal(map[string]any{"lead_id": leadID})
	prospectingReq := httptest.NewRequest(http.MethodPost, "/agents/prospecting/trigger", bytes.NewReader(prospectingBody))
	prospectingReq = prospectingReq.WithContext(contextWithWorkspaceID(prospectingReq.Context(), wsID))
	prospectingReq.Header.Set("Content-Type", "application/json")
	prospectingRR := httptest.NewRecorder()
	prospectingHandler.TriggerProspectingAgent(prospectingRR, prospectingReq)
	if prospectingRR.Code != http.StatusCreated {
		t.Fatalf("expected 201 for prospecting success, got %d: %s", prospectingRR.Code, prospectingRR.Body.String())
	}

	kbHandler, kbCaseID := newTestKBAgentHandler(t, db, wsID, ownerID)
	kbBody, _ := json.Marshal(map[string]any{"case_id": kbCaseID})
	kbReq := httptest.NewRequest(http.MethodPost, "/agents/kb/trigger", bytes.NewReader(kbBody))
	kbReq = kbReq.WithContext(contextWithWorkspaceID(kbReq.Context(), wsID))
	kbReq.Header.Set("Content-Type", "application/json")
	kbRR := httptest.NewRecorder()
	kbHandler.TriggerKBAgent(kbRR, kbReq)
	if kbRR.Code != http.StatusCreated {
		t.Fatalf("expected 201 for kb success, got %d: %s", kbRR.Code, kbRR.Body.String())
	}
}

func TestAgentHandlerConfigBuildersAndTriggeredByHelpers(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	supportCfg, ok := buildSupportConfig(rr, supportAgentRequest{
		CaseID:        "case-1",
		CustomerQuery: "help",
	}, "ws-1")
	if !ok || supportCfg.WorkspaceID != "ws-1" || supportCfg.CaseID != "case-1" {
		t.Fatalf("unexpected support config = %#v, ok=%v", supportCfg, ok)
	}

	rr = httptest.NewRecorder()
	prospectingCfg, ok := buildProspectingConfig(rr, prospectingAgentRequest{LeadID: "lead-1"}, "ws-1")
	if !ok || prospectingCfg.Language != defaultAgentLanguage {
		t.Fatalf("unexpected prospecting config = %#v, ok=%v", prospectingCfg, ok)
	}
	if withProspectingTriggeredBy(prospectingCfg, "").TriggeredByUserID != nil {
		t.Fatal("expected empty triggered by for blank user")
	}
	withProspecting := withProspectingTriggeredBy(prospectingCfg, "user-1")
	if withProspecting.TriggeredByUserID == nil || *withProspecting.TriggeredByUserID != "user-1" {
		t.Fatalf("unexpected prospecting triggered by = %#v", withProspecting.TriggeredByUserID)
	}

	rr = httptest.NewRecorder()
	kbCfg, ok := buildKBConfig(rr, kbAgentRequest{CaseID: "case-1"}, "ws-1")
	if !ok || kbCfg.Language != defaultAgentLanguage {
		t.Fatalf("unexpected kb config = %#v, ok=%v", kbCfg, ok)
	}
	if withKBTriggeredBy(kbCfg, "").TriggeredByUserID != nil {
		t.Fatal("expected empty KB triggered by for blank user")
	}
	withKB := withKBTriggeredBy(kbCfg, "user-2")
	if withKB.TriggeredByUserID == nil || *withKB.TriggeredByUserID != "user-2" {
		t.Fatalf("unexpected kb triggered by = %#v", withKB.TriggeredByUserID)
	}
}

func TestAgentHandlerHelperCoverage(t *testing.T) {
	t.Parallel()

	input := buildTriggerInput(triggerAgentRequest{
		AgentID:        "agent-1",
		TriggerContext: json.RawMessage(`{"x":1}`),
		Inputs:         json.RawMessage(`{"y":2}`),
	}, "ws-1", "user-1")
	if input.TriggerType != agent.TriggerTypeManual {
		t.Fatalf("unexpected trigger type = %q", input.TriggerType)
	}
	if input.TriggeredBy == nil || *input.TriggeredBy != "user-1" {
		t.Fatalf("unexpected triggered by = %#v", input.TriggeredBy)
	}

	req := httptest.NewRequest(http.MethodGet, "/agents/runs?limit=10&offset=5", nil)
	limit, offset := parsePageParams(req)
	if limit != 10 || offset != 5 {
		t.Fatalf("parsePageParams() = (%d,%d)", limit, offset)
	}

	defaultReq := httptest.NewRequest(http.MethodGet, "/agents/runs?limit=-1&offset=bad", nil)
	limit, offset = parsePageParams(defaultReq)
	if limit != 25 || offset != 0 {
		t.Fatalf("parsePageParams(default) = (%d,%d)", limit, offset)
	}

	startedAt := time.Now().UTC()
	createdAt := startedAt.Add(-time.Minute)
	resp := agentRunToResponse(&agent.Run{
		ID:           "run-1",
		WorkspaceID:  "ws-1",
		DefinitionID: "agent-1",
		TriggerType:  agent.TriggerTypeManual,
		Status:       agent.StatusSuccess,
		StartedAt:    startedAt,
		CreatedAt:    createdAt,
	})
	if resp.CompletedAt != nil {
		t.Fatalf("unexpected completedAt = %#v", resp.CompletedAt)
	}
}
