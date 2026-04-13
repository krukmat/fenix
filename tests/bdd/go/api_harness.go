package gobdd

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/api/handlers"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent/agents"
	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/copilot"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/domain/usage"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
	isqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

const (
	bddAuthUserHeader = "X-BDD-User"
	bddModelID        = "bdd-llm"
)

type bddAPIRuntime struct {
	db          *sql.DB
	router      http.Handler
	workspaceID string
	userID      string
	evidence    *bddEvidenceBuilder
	llm         *bddLLMProvider
	usage       *usage.Service
}

type bddEvidenceBuilder struct {
	packs   map[string]*knowledge.EvidencePack
	results *knowledge.SearchResults
}

type bddLLMProvider struct {
	responses []string
}

func newBDDAPIRuntime() (*bddAPIRuntime, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := isqlite.MigrateUp(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	runtime := &bddAPIRuntime{
		db:          db,
		workspaceID: "ws_bdd",
		userID:      "user_bdd",
		evidence:    &bddEvidenceBuilder{packs: map[string]*knowledge.EvidencePack{}},
		llm:         &bddLLMProvider{},
		usage:       usage.NewService(db),
	}
	if err := runtime.seedActors(); err != nil {
		_ = db.Close()
		return nil, err
	}
	runtime.router = runtime.buildRouter()
	return runtime, nil
}

func ensureBDDRuntime(state *scenarioState) (*bddAPIRuntime, error) {
	if state.apiRuntime != nil {
		return state.apiRuntime, nil
	}
	runtime, err := newBDDAPIRuntime()
	if err != nil {
		return nil, err
	}
	state.apiRuntime = runtime
	return runtime, nil
}

func (r *bddAPIRuntime) close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *bddAPIRuntime) buildRouter() http.Handler {
	auditService := audit.NewAuditService(r.db)
	policyEngine := policy.NewPolicyEngine(r.db, nil, auditService)
	toolRegistry := tool.NewToolRegistryWithRuntimeAndUsage(r.db, policyEngine, auditService, r.usage)
	caseService := crm.NewCaseService(r.db)
	dealService := crm.NewDealService(r.db)
	ingestService := knowledge.NewIngestService(r.db, eventbus.New())
	_ = tool.RegisterBuiltInExecutors(toolRegistry, tool.BuiltinServices{
		DB:      r.db,
		Case:    caseService,
		Account: crm.NewAccountService(r.db),
		Deal:    dealService,
		Ingest:  ingestService,
	})
	_ = toolRegistry.EnsureBuiltInToolDefinitionsForAllWorkspaces(context.Background())

	orchestrator := agent.NewOrchestrator(r.db)
	approvalService := policy.NewApprovalService(r.db, auditService)
	supportAgent := agents.NewSupportAgentWithDBAndUsage(orchestrator, toolRegistry, r.evidence, r.db, r.usage)
	dealRiskAgent := agents.NewDealRiskAgent(
		orchestrator,
		toolRegistry,
		r.evidence,
		nil,
		dealService,
		crm.NewAccountService(r.db),
		r.db,
	)
	handoffService := agent.NewHandoffService(r.db, caseService, eventbus.New())
	copilotService := copilot.NewActionServiceWithUsage(r.evidence, r.llm, policyEngine, auditService, r.usage)

	rx := chi.NewRouter()
	rx.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			userID := req.Header.Get(bddAuthUserHeader)
			if userID == "" {
				userID = r.userID
			}
			ctx := context.WithValue(req.Context(), ctxkeys.WorkspaceID, r.workspaceID)
			ctx = context.WithValue(ctx, ctxkeys.UserID, userID)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})

	agentHandler := handlers.NewAgentHandler(orchestrator)
	supportHandler := handlers.NewSupportAgentHandler(supportAgent)
	dealRiskHandler := handlers.NewDealRiskAgentHandler(dealRiskAgent)
	handoffHandler := handlers.NewHandoffHandler(handoffService)
	approvalHandler := handlers.NewApprovalHandler(approvalService)
	auditHandler := handlers.NewAuditHandler(auditService)
	usageHandler := handlers.NewUsageHandler(r.usage)
	copilotHandler := handlers.NewCopilotActionsHandler(copilotService)

	rx.Route("/api/v1", func(api chi.Router) {
		api.Route("/copilot", func(c chi.Router) {
			c.Post("/sales-brief", copilotHandler.SalesBrief)
		})
		api.Route("/agents", func(a chi.Router) {
			a.Get("/runs/{id}", agentHandler.GetAgentRun)
			a.Post("/support/trigger", supportHandler.TriggerSupportAgent)
			a.Post("/deal-risk/trigger", dealRiskHandler.TriggerDealRiskAgent)
			a.Get("/runs/{id}/handoff", handoffHandler.GetHandoffPackage)
		})
		api.Route("/approvals", func(a chi.Router) {
			a.Get("/", approvalHandler.ListPendingApprovals)
			a.Put("/{id}", approvalHandler.DecideApproval)
		})
		api.Route("/audit", func(a chi.Router) {
			a.Get("/events", auditHandler.Query)
		})
		api.Get("/usage", usageHandler.ListUsage)
		api.Get("/quota-state", usageHandler.GetQuotaState)
	})

	return rx
}

func (r *bddAPIRuntime) seedActors() error {
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := r.db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, 'BDD Workspace', 'bdd-workspace', ?, ?)
	`, r.workspaceID, now, now); err != nil {
		return err
	}
	for _, userID := range []string{r.userID, "governance_bdd"} {
		if _, err := r.db.Exec(`
			INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'active', ?, ?)
		`, userID, r.workspaceID, userID+"@example.com", strings.ReplaceAll(userID, "_", " "), now, now); err != nil {
			return err
		}
	}
	return nil
}

func (r *bddAPIRuntime) ensureSupportAgentDefinition() error {
	_, err := r.db.Exec(`
		INSERT OR IGNORE INTO agent_definition (id, workspace_id, name, agent_type, status)
		VALUES ('support-agent', ?, 'Support Agent', 'support', 'active')
	`, r.workspaceID)
	return err
}

func (r *bddAPIRuntime) ensureDealRiskAgentDefinition() error {
	_, err := r.db.Exec(`
		INSERT OR IGNORE INTO agent_definition (id, workspace_id, name, agent_type, status)
		VALUES ('deal-risk-agent', ?, 'Deal Risk Agent', 'deal-risk', 'active')
	`, r.workspaceID)
	return err
}

func (r *bddAPIRuntime) createSupportCase(priority string) (string, error) {
	contact, err := crm.NewContactService(r.db).Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: r.workspaceID,
		FirstName:   "Ana",
		LastName:    "Cliente",
		Email:       "ana@example.com",
		Status:      "active",
		OwnerID:     r.userID,
	})
	if err != nil {
		return "", err
	}
	ticket, err := crm.NewCaseService(r.db).Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: r.workspaceID,
		ContactID:   contact.ID,
		OwnerID:     r.userID,
		Subject:     "Service issue",
		Description: "Customer cannot access the service",
		Priority:    priority,
		Status:      "open",
	})
	if err != nil {
		return "", err
	}
	return ticket.ID, nil
}

func (r *bddAPIRuntime) createSalesAccount() (string, error) {
	account, err := crm.NewAccountService(r.db).Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: r.workspaceID,
		Name:        "Acme Corp",
		Domain:      "acme.example.com",
		Industry:    "manufacturing",
		SizeSegment: "mid",
		OwnerID:     r.userID,
	})
	if err != nil {
		return "", err
	}
	return account.ID, nil
}

func (r *bddAPIRuntime) createSalesDeal() (string, error) {
	accountID, err := r.createSalesAccount()
	if err != nil {
		return "", err
	}
	pipelineService := crm.NewPipelineService(r.db)
	pipeline, err := pipelineService.Create(context.Background(), crm.CreatePipelineInput{
		WorkspaceID: r.workspaceID,
		Name:        "Sales",
		EntityType:  "deal",
	})
	if err != nil {
		return "", err
	}
	stage, err := pipelineService.CreateStage(context.Background(), crm.CreatePipelineStageInput{
		PipelineID: pipeline.ID,
		Name:       "Qualification",
		Position:   1,
	})
	if err != nil {
		return "", err
	}
	deal, err := crm.NewDealService(r.db).Create(context.Background(), crm.CreateDealInput{
		WorkspaceID: r.workspaceID,
		AccountID:   accountID,
		PipelineID:  pipeline.ID,
		StageID:     stage.ID,
		OwnerID:     r.userID,
		Title:       "Acme renewal",
		Status:      "open",
	})
	if err != nil {
		return "", err
	}
	return deal.ID, nil
}

func (r *bddAPIRuntime) markDealStale(dealID string, createdAt, updatedAt time.Time) error {
	_, err := r.db.Exec(`
		UPDATE deal
		SET created_at = ?, updated_at = ?
		WHERE id = ? AND workspace_id = ?
	`, createdAt.Format(time.RFC3339), updatedAt.Format(time.RFC3339), dealID, r.workspaceID)
	return err
}

func (r *bddAPIRuntime) recordQuotaState() (string, error) {
	policyRecord, err := r.usage.CreatePolicy(context.Background(), usage.CreatePolicyInput{
		WorkspaceID:     r.workspaceID,
		PolicyType:      "quota",
		ScopeType:       "workspace",
		MetricName:      "requests",
		LimitValue:      1000,
		ResetPeriod:     "monthly",
		EnforcementMode: "soft",
		IsActive:        true,
	})
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	_, err = r.usage.UpsertState(context.Background(), usage.UpsertStateInput{
		WorkspaceID:   r.workspaceID,
		QuotaPolicyID: policyRecord.ID,
		CurrentValue:  3,
		PeriodStart:   start,
		PeriodEnd:     end,
		LastEventAt:   &now,
	})
	if err != nil {
		return "", err
	}
	return policyRecord.ID, nil
}

func (r *bddAPIRuntime) request(method, path, userID string, body any) (int, []byte, error) {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(payload)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set(bddAuthUserHeader, userID)
	}
	rr := httptest.NewRecorder()
	r.router.ServeHTTP(rr, req)
	return rr.Code, rr.Body.Bytes(), nil
}

func (r *bddEvidenceBuilder) BuildEvidencePack(_ context.Context, input knowledge.BuildEvidencePackInput) (*knowledge.EvidencePack, error) {
	if pack, ok := r.packs[input.Query]; ok {
		return cloneEvidencePack(pack), nil
	}
	return &knowledge.EvidencePack{
		SchemaVersion:        knowledge.EvidencePackSchemaVersion,
		Query:                input.Query,
		Sources:              []knowledge.Evidence{},
		SourceCount:          0,
		DedupCount:           0,
		FilteredCount:        0,
		Confidence:           knowledge.ConfidenceLow,
		Warnings:             []string{"no evidence configured for query"},
		RetrievalMethodsUsed: []knowledge.EvidenceMethod{},
		BuiltAt:              time.Now().UTC(),
	}, nil
}

func (r *bddEvidenceBuilder) HybridSearch(_ context.Context, _ knowledge.SearchInput) (*knowledge.SearchResults, error) {
	if r.results == nil {
		return &knowledge.SearchResults{Items: []knowledge.SearchResult{}}, nil
	}
	cloned := &knowledge.SearchResults{Items: append([]knowledge.SearchResult(nil), r.results.Items...)}
	return cloned, nil
}

func (r *bddEvidenceBuilder) set(query string, pack *knowledge.EvidencePack) {
	r.packs[query] = cloneEvidencePack(pack)
}

func (r *bddEvidenceBuilder) setResults(results *knowledge.SearchResults) {
	if results == nil {
		r.results = &knowledge.SearchResults{Items: []knowledge.SearchResult{}}
		return
	}
	r.results = &knowledge.SearchResults{Items: append([]knowledge.SearchResult(nil), results.Items...)}
}

func (p *bddLLMProvider) queue(responses ...string) {
	p.responses = append(p.responses, responses...)
}

func (p *bddLLMProvider) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	if len(p.responses) == 0 {
		return nil, fmt.Errorf("bdd llm: no response queued")
	}
	content := p.responses[0]
	p.responses = p.responses[1:]
	return &llm.ChatResponse{Content: content, Tokens: 42}, nil
}

func (p *bddLLMProvider) Embed(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{Embeddings: [][]float32{{0.1, 0.2, 0.3}}}, nil
}

func (p *bddLLMProvider) ModelInfo() llm.ModelMeta {
	return llm.ModelMeta{ID: bddModelID}
}

func (p *bddLLMProvider) HealthCheck(_ context.Context) error {
	return nil
}

func cloneEvidencePack(pack *knowledge.EvidencePack) *knowledge.EvidencePack {
	if pack == nil {
		return nil
	}
	cloned := *pack
	cloned.Sources = append([]knowledge.Evidence(nil), pack.Sources...)
	cloned.Warnings = append([]string(nil), pack.Warnings...)
	cloned.RetrievalMethodsUsed = append([]knowledge.EvidenceMethod(nil), pack.RetrievalMethodsUsed...)
	return &cloned
}

func newBDDEvidencePack(query string, confidence knowledge.ConfidenceLevel, scores ...float64) *knowledge.EvidencePack {
	sources := make([]knowledge.Evidence, 0, len(scores))
	methods := make([]knowledge.EvidenceMethod, 0, len(scores))
	for i, score := range scores {
		snippet := fmt.Sprintf("evidence snippet %d", i+1)
		sources = append(sources, knowledge.Evidence{
			ID:              fmt.Sprintf("ev_%d", i+1),
			KnowledgeItemID: fmt.Sprintf("ki_%d", i+1),
			Method:          knowledge.EvidenceMethodHybrid,
			Score:           score,
			Snippet:         &snippet,
			CreatedAt:       time.Now().UTC(),
		})
		methods = append(methods, knowledge.EvidenceMethodHybrid)
	}
	return &knowledge.EvidencePack{
		SchemaVersion:        knowledge.EvidencePackSchemaVersion,
		Query:                query,
		Sources:              sources,
		SourceCount:          len(sources),
		DedupCount:           0,
		FilteredCount:        0,
		Confidence:           confidence,
		Warnings:             []string{},
		RetrievalMethodsUsed: methods,
		BuiltAt:              time.Now().UTC(),
	}
}

func decodeBDDEnvelope(body []byte) (map[string]any, error) {
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func salesEvidenceQuery(entityType, entityID string) string {
	return fmt.Sprintf("entity_type:%s entity_id:%s latest updates timeline next steps", entityType, entityID)
}
