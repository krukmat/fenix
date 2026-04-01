package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	signaldomain "github.com/matiasleandrokruk/fenix/internal/domain/signal"
	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

func BenchmarkHandler_Critical_KnowledgeSearch(b *testing.B) {
	db := mustOpenDBWithMigrations(b)
	wsID, _ := setupWorkspaceAndOwner(b, db)

	bus := eventbus.New()
	ingestSvc := knowledge.NewIngestService(db, bus)
	for i := 0; i < 5; i++ {
		_, err := ingestSvc.Ingest(contextWithWorkspaceID(context.Background(), wsID), knowledge.CreateKnowledgeItemInput{
			WorkspaceID: wsID,
			SourceType:  knowledge.SourceTypeDocument,
			Title:       "Pricing Strategy",
			RawContent:  "enterprise pricing discount policy and renewal guidance",
		})
		if err != nil {
			b.Fatalf("seed ingest failed: %v", err)
		}
	}

	handler := NewKnowledgeSearchHandler(knowledge.NewSearchService(db, &searchStubLLM{}))
	body := []byte(`{"query":"pricing","limit":10}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/search", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
		rr := httptest.NewRecorder()

		handler.Search(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("Search status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func BenchmarkHandler_Critical_ListDeals(b *testing.B) {
	db := mustOpenDBWithMigrations(b)
	wsID, ownerID := setupWorkspaceAndOwner(b, db)
	handler := NewDealHandler(crm.NewDealService(db))

	accountID := createAccountForTask15(b, db, wsID, ownerID, "Benchmark Deal Account")
	pipelineID, stageID := createPipelineAndStageForTask15(b, db, wsID)
	svc := crm.NewDealService(db)
	for i := 0; i < 20; i++ {
		_, err := svc.Create(context.Background(), crm.CreateDealInput{
			WorkspaceID: wsID,
			AccountID:   accountID,
			PipelineID:  pipelineID,
			StageID:     stageID,
			OwnerID:     ownerID,
			Title:       "Benchmark Deal",
		})
		if err != nil {
			b.Fatalf("seed deal failed: %v", err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/deals?limit=10&offset=0", nil)
		req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
		rr := httptest.NewRecorder()

		handler.ListDeals(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("ListDeals status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func BenchmarkHandler_Critical_ListCases(b *testing.B) {
	db := mustOpenDBWithMigrations(b)
	wsID, ownerID := setupWorkspaceAndOwner(b, db)
	svc := crm.NewCaseService(db)
	handler := NewCaseHandler(svc)

	for i := 0; i < 20; i++ {
		_, err := svc.Create(context.Background(), crm.CreateCaseInput{
			WorkspaceID: wsID,
			OwnerID:     ownerID,
			Subject:     "benchmark-case",
		})
		if err != nil {
			b.Fatalf("seed case failed: %v", err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?limit=10&offset=0", nil)
		req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
		rr := httptest.NewRecorder()

		handler.ListCases(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("ListCases status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func BenchmarkHandler_Critical_ListTimelineByEntity(b *testing.B) {
	db := mustOpenDBWithMigrations(b)
	wsID, ownerID := setupWorkspaceAndOwner(b, db)
	accountSvc := crm.NewAccountService(db)
	account, err := accountSvc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Benchmark Timeline Account",
		OwnerID:     ownerID,
	})
	if err != nil {
		b.Fatalf("seed account failed: %v", err)
	}

	svc := crm.NewTimelineService(db)
	for i := 0; i < 20; i++ {
		_, err := svc.Create(context.Background(), crm.CreateTimelineEventInput{
			WorkspaceID: wsID,
			EntityType:  "account",
			EntityID:    account.ID,
			ActorID:     ownerID,
			EventType:   "created",
		})
		if err != nil {
			b.Fatalf("seed timeline failed: %v", err)
		}
	}
	handler := NewTimelineHandler(svc)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/timeline/account/"+account.ID, nil)
		req = withRouteParams(req, map[string]string{
			"entity_type": "account",
			"entity_id":   account.ID,
		})
		req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
		rr := httptest.NewRecorder()

		handler.ListTimelineByEntity(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("ListTimelineByEntity status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func BenchmarkHandler_Critical_WorkflowExecute(b *testing.B) {
	db := mustOpenDBWithMigrations(b)
	wsID := createWorkspace(b, db)
	if _, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES ('user_test', ?, 'user_test@example.com', 'User Test', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, wsID); err != nil {
		b.Fatalf("insert user_account: %v", err)
	}
	insertDSLWorkflowAgent(b, db, wsID, "dsl-agent-bench")
	insertExecutableWorkflow(b, db, wsID, "wf_exec_bench", "dsl-agent-bench")

	toolRegistry := setupWorkflowToolRegistry(b, db, wsID)
	orch := agent.NewOrchestratorWithRegistry(db, agent.NewRunnerRegistry())
	handler := NewWorkflowHandlerWithRuntime(workflowdomain.NewService(db), nil, db, orch, toolRegistry, nil, nil, nil, nil)
	router := chi.NewRouter()
	router.Post("/workflows/{id}/execute", handler.Execute)
	body := []byte(`{"trigger_context":{"case":{"id":"case-1"}}}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/workflows/wf_exec_bench/execute", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithWorkspaceID(context.WithValue(req.Context(), ctxkeys.UserID, "user_test"), wsID))
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("Execute status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func BenchmarkHandler_Critical_AgentTrigger(b *testing.B) {
	db := mustOpenDBWithMigrations(b)
	wsID := createWorkspace(b, db)
	insertTestAgentDef(b, db, "agent-bench", wsID)
	handler := NewAgentHandler(agent.NewOrchestrator(db))
	body := []byte(`{"agent_id":"agent-bench","trigger_type":"manual"}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/agents/trigger", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
		rr := httptest.NewRecorder()

		handler.TriggerAgent(rr, req)

		if rr.Code != http.StatusCreated {
			b.Fatalf("TriggerAgent status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func BenchmarkHandler_Critical_SignalList(b *testing.B) {
	mock := newMockSignalService()
	mock.items["sig_1"] = benchmarkSignal("sig_1", "lead", "lead_1")
	mock.items["sig_2"] = benchmarkSignal("sig_2", "lead", "lead_2")
	handler := NewSignalHandler(signalServiceAdapter{mock})
	router := chi.NewRouter()
	router.Get("/signals", handler.List)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/signals", nil)
		req = req.WithContext(withSignalContext(req.Context()))
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("SignalList status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func BenchmarkHandler_ListAccounts(b *testing.B) {
	db := mustOpenDBWithMigrations(b)
	wsID, ownerID := setupWorkspaceAndOwner(b, db)
	svc := crm.NewAccountService(db)
	handler := NewAccountHandler(svc)

	for i := 0; i < 20; i++ {
		_, err := svc.Create(context.Background(), crm.CreateAccountInput{
			WorkspaceID: wsID,
			Name:        "Benchmark Account " + randID(),
			OwnerID:     ownerID,
		})
		if err != nil {
			b.Fatalf("seed account failed: %v", err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts?limit=10&offset=0", nil)
		req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
		rr := httptest.NewRecorder()

		handler.ListAccounts(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("ListAccounts status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func BenchmarkHandler_ListTimeline(b *testing.B) {
	db := mustOpenDBWithMigrations(b)
	wsID, ownerID := setupWorkspaceAndOwner(b, db)
	accountSvc := crm.NewAccountService(db)
	account, err := accountSvc.Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: wsID,
		Name:        "Timeline Bench Account",
		OwnerID:     ownerID,
	})
	if err != nil {
		b.Fatalf("seed account failed: %v", err)
	}

	svc := crm.NewTimelineService(db)
	for i := 0; i < 20; i++ {
		_, err := svc.Create(context.Background(), crm.CreateTimelineEventInput{
			WorkspaceID: wsID,
			EntityType:  "account",
			EntityID:    account.ID,
			ActorID:     ownerID,
			EventType:   "created",
		})
		if err != nil {
			b.Fatalf("seed timeline failed: %v", err)
		}
	}
	handler := NewTimelineHandler(svc)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/timeline?limit=10&offset=0", nil)
		req = req.WithContext(contextWithWorkspaceID(req.Context(), wsID))
		rr := httptest.NewRecorder()

		handler.ListTimeline(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("ListTimeline status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func BenchmarkHandler_WorkflowList(b *testing.B) {
	mock := newMockWorkflowService()
	mock.items["wf_1"] = &workflowdomain.Workflow{ID: "wf_1", WorkspaceID: "ws_test", Name: "workflow_1"}
	mock.items["wf_2"] = &workflowdomain.Workflow{ID: "wf_2", WorkspaceID: "ws_test", Name: "workflow_2"}
	handler := NewWorkflowHandler(workflowServiceAdapter{mock})
	router := chi.NewRouter()
	router.Get("/workflows", handler.List)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/workflows", nil)
		req = req.WithContext(withWorkflowContext(req.Context()))
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("WorkflowList status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func BenchmarkHandler_SignalDismiss(b *testing.B) {
	router := chi.NewRouter()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock := newMockSignalService()
		mock.items["sig_1"] = benchmarkSignal("sig_1", "lead", "lead_1")
		handler := NewSignalHandler(signalServiceAdapter{mock})
		router = chi.NewRouter()
		router.Put("/signals/{id}/dismiss", handler.Dismiss)

		req := httptest.NewRequest(http.MethodPut, "/signals/sig_1/dismiss", nil)
		req = req.WithContext(withSignalContext(req.Context()))
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			b.Fatalf("SignalDismiss status=%d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func withRouteParams(req *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for key, value := range params {
		rctx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func benchmarkSignal(id, entityType, entityID string) *signaldomain.Signal {
	return &signaldomain.Signal{
		ID:          id,
		WorkspaceID: "ws_test",
		EntityType:  entityType,
		EntityID:    entityID,
		SignalType:  "intent_high",
		Confidence:  0.9,
		EvidenceIDs: []string{"ev-1"},
		SourceType:  "workflow",
		SourceID:    "wf_1",
		Status:      signaldomain.StatusActive,
	}
}
