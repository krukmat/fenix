// Task 1.3.8: Route registration and go-chi router setup
// Task 1.6.13: Restructured into public routes (/auth/*) and JWT-protected routes (/api/v1/*)
package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/matiasleandrokruk/fenix/internal/api/handlers"
	apmiddleware "github.com/matiasleandrokruk/fenix/internal/api/middleware"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent/agents"
	domainaudit "github.com/matiasleandrokruk/fenix/internal/domain/audit"
	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	copilotdomain "github.com/matiasleandrokruk/fenix/internal/domain/copilot"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
	tooldomain "github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/config"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

// routeByID is the chi route pattern for resource-by-ID endpoints (used 27 times).
const routeByID = "/{id}"

// NewRouter creates and configures a new chi router with all routes.
// Task 1.3.8: Setup go-chi router with middleware + account endpoints
// Task 1.6.13: Public routes (/health, /auth/*) vs protected routes (/api/v1/*)
//
//nolint:funlen // router principal mantiene registro centralizado de rutas por diseño
func NewRouter(db *sql.DB) *chi.Mux {
	r := chi.NewRouter()
	auditService := domainaudit.NewAuditService(db)

	// Global middleware (runs on all routes)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// ===== PUBLIC ROUTES (no auth required) =====

	// Health check — unauthenticated, used by load balancers and health probes
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	// Auth endpoints — public, no JWT required (Task 1.6.13)
	authHandler := handlers.NewAuthHandler(domainauth.NewAuthServiceWithAudit(db, auditService))
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register) // POST /auth/register
		r.Post("/login", authHandler.Login)       // POST /auth/login
	})

	// ===== PROTECTED ROUTES (JWT required via AuthMiddleware) =====

	// All /api/v1/* routes require a valid Bearer JWT token (Task 1.6.13)
	// AuthMiddleware validates the token and injects UserID + WorkspaceID into context.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(apmiddleware.AuthMiddleware)
		r.Use(apmiddleware.AuditMiddleware(auditService))

		// Shared app services for protected APIs
		knowledgeBus := eventbus.New()
		cfg := config.Load()
		llmProvider := llm.NewOllamaProvider(cfg.OllamaBaseURL, cfg.OllamaModel)
		ingestSvc := knowledge.NewIngestService(db, knowledgeBus)
		embedder := knowledge.NewEmbedderService(db, llmProvider)
		reindexSvc := knowledge.NewReindexService(db, knowledgeBus, ingestSvc, auditService)
		go embedder.Start(context.Background(), knowledgeBus)
		go reindexSvc.Start(context.Background())
		toolRegistry := tooldomain.NewToolRegistry(db)

		// Account endpoints (Task 1.3.8)
		accountHandler := handlers.NewAccountHandler(crm.NewAccountServiceWithBus(db, knowledgeBus))
		contactHandler := handlers.NewContactHandler(crm.NewContactService(db))
		r.Route("/accounts", func(r chi.Router) {
			r.Post("/", accountHandler.CreateAccount)         // POST /api/v1/accounts
			r.Get("/", accountHandler.ListAccounts)           // GET /api/v1/accounts
			r.Get(routeByID, accountHandler.GetAccount)       // GET /api/v1/accounts/{id}
			r.Put(routeByID, accountHandler.UpdateAccount)    // PUT /api/v1/accounts/{id}
			r.Delete(routeByID, accountHandler.DeleteAccount) // DELETE /api/v1/accounts/{id}
			r.Get("/{account_id}/contacts", contactHandler.ListContactsByAccount)
		})

		r.Route("/contacts", func(r chi.Router) {
			r.Post("/", contactHandler.CreateContact)         // POST /api/v1/contacts
			r.Get("/", contactHandler.ListContacts)           // GET /api/v1/contacts
			r.Get(routeByID, contactHandler.GetContact)       // GET /api/v1/contacts/{id}
			r.Put(routeByID, contactHandler.UpdateContact)    // PUT /api/v1/contacts/{id}
			r.Delete(routeByID, contactHandler.DeleteContact) // DELETE /api/v1/contacts/{id}
		})

		// Lead endpoints (Task 1.5)
		leadHandler := handlers.NewLeadHandler(crm.NewLeadService(db))
		dealHandler := handlers.NewDealHandler(crm.NewDealService(db))
		caseService := crm.NewCaseServiceWithBus(db, knowledgeBus)
		caseHandler := handlers.NewCaseHandler(caseService)
		pipelineHandler := handlers.NewPipelineHandler(crm.NewPipelineService(db))
		activityHandler := handlers.NewActivityHandler(crm.NewActivityService(db))
		noteHandler := handlers.NewNoteHandler(crm.NewNoteService(db))
		attachmentHandler := handlers.NewAttachmentHandler(crm.NewAttachmentService(db))
		timelineHandler := handlers.NewTimelineHandler(crm.NewTimelineService(db))
		r.Route("/leads", func(r chi.Router) {
			r.Post("/", leadHandler.CreateLead)         // POST /api/v1/leads
			r.Get("/", leadHandler.ListLeads)           // GET /api/v1/leads
			r.Get(routeByID, leadHandler.GetLead)       // GET /api/v1/leads/{id}
			r.Put(routeByID, leadHandler.UpdateLead)    // PUT /api/v1/leads/{id}
			r.Delete(routeByID, leadHandler.DeleteLead) // DELETE /api/v1/leads/{id}
		})

		r.Route("/deals", func(r chi.Router) {
			r.Post("/", dealHandler.CreateDeal)
			r.Get("/", dealHandler.ListDeals)
			r.Get(routeByID, dealHandler.GetDeal)
			r.Put(routeByID, dealHandler.UpdateDeal)
			r.Delete(routeByID, dealHandler.DeleteDeal)
		})

		r.Route("/cases", func(r chi.Router) {
			r.Post("/", caseHandler.CreateCase)
			r.Get("/", caseHandler.ListCases)
			r.Get(routeByID, caseHandler.GetCase)
			r.Put(routeByID, caseHandler.UpdateCase)
			r.Delete(routeByID, caseHandler.DeleteCase)
		})

		r.Route("/pipelines", func(r chi.Router) {
			r.Post("/", pipelineHandler.CreatePipeline)
			r.Get("/", pipelineHandler.ListPipelines)
			r.Get(routeByID, pipelineHandler.GetPipeline)
			r.Put(routeByID, pipelineHandler.UpdatePipeline)
			r.Delete(routeByID, pipelineHandler.DeletePipeline)
			r.Post("/{id}/stages", pipelineHandler.CreateStage)
			r.Get("/{id}/stages", pipelineHandler.ListStages)
			r.Put("/stages/{stage_id}", pipelineHandler.UpdateStage)
			r.Delete("/stages/{stage_id}", pipelineHandler.DeleteStage)
		})

		r.Route("/activities", func(r chi.Router) {
			r.Post("/", activityHandler.CreateActivity)
			r.Get("/", activityHandler.ListActivities)
			r.Get(routeByID, activityHandler.GetActivity)
			r.Put(routeByID, activityHandler.UpdateActivity)
			r.Delete(routeByID, activityHandler.DeleteActivity)
		})

		r.Route("/notes", func(r chi.Router) {
			r.Post("/", noteHandler.CreateNote)
			r.Get("/", noteHandler.ListNotes)
			r.Get(routeByID, noteHandler.GetNote)
			r.Put(routeByID, noteHandler.UpdateNote)
			r.Delete(routeByID, noteHandler.DeleteNote)
		})

		r.Route("/attachments", func(r chi.Router) {
			r.Post("/", attachmentHandler.CreateAttachment)
			r.Get("/", attachmentHandler.ListAttachments)
			r.Get(routeByID, attachmentHandler.GetAttachment)
			r.Delete(routeByID, attachmentHandler.DeleteAttachment)
		})

		r.Route("/timeline", func(r chi.Router) {
			r.Get("/", timelineHandler.ListTimeline)
			r.Get("/{entity_type}/{entity_id}", timelineHandler.ListTimelineByEntity)
		})

		// Task 2.5: SearchService — hybrid BM25 + vector search with RRF ranking
		searchSvc := knowledge.NewSearchService(db, llmProvider)
		// Task 2.6: EvidencePackService — curated evidence packs for AI layer
		evidenceSvc := knowledge.NewEvidencePackService(db, searchSvc, knowledge.DefaultEvidenceConfig())

		knowledgeIngestHandler := handlers.NewKnowledgeIngestHandler(ingestSvc)
		knowledgeSearchHandler := handlers.NewKnowledgeSearchHandler(searchSvc)
		knowledgeEvidenceHandler := handlers.NewKnowledgeEvidenceHandler(evidenceSvc)
		knowledgeReindexHandler := handlers.NewKnowledgeReindexHandler(reindexSvc)
		approvalHandler := handlers.NewApprovalHandler(policy.NewApprovalService(db, auditService))
		toolHandler := handlers.NewToolHandler(toolRegistry)
		// Task 3.9: Prompt Versioning
		promptHandler := handlers.NewPromptHandler(agent.NewPromptService(db, auditService))
		policyEngine := policy.NewPolicyEngine(db, nil, auditService)
		copilotChatSvc := copilotdomain.NewChatService(evidenceSvc, llmProvider, policyEngine, auditService)
		copilotChatHandler := handlers.NewCopilotChatHandler(copilotChatSvc)
		copilotActionsSvc := copilotdomain.NewActionService(evidenceSvc, llmProvider, policyEngine, auditService)
		copilotActionsHandler := handlers.NewCopilotActionsHandler(copilotActionsSvc)

		_ = tooldomain.RegisterBuiltInExecutors(toolRegistry, tooldomain.BuiltinServices{
			DB:      db,
			Case:    caseService,
			Lead:    crm.NewLeadService(db),
			Account: crm.NewAccountService(db),
			Ingest:  ingestSvc,
		})
		_ = toolRegistry.EnsureBuiltInToolDefinitionsForAllWorkspaces(context.Background())
		r.Route("/knowledge", func(r chi.Router) {
			r.Post("/ingest", knowledgeIngestHandler.Ingest)    // POST /api/v1/knowledge/ingest
			r.Post("/search", knowledgeSearchHandler.Search)    // POST /api/v1/knowledge/search
			r.Post("/evidence", knowledgeEvidenceHandler.Build) // POST /api/v1/knowledge/evidence
			r.Post("/reindex", knowledgeReindexHandler.Reindex) // POST /api/v1/knowledge/reindex
		})

		r.Route("/approvals", func(r chi.Router) {
			r.Get("/", approvalHandler.ListPendingApprovals) // GET /api/v1/approvals
			r.Put(routeByID, approvalHandler.DecideApproval) // PUT /api/v1/approvals/{id}
		})

		r.Route("/admin/tools", func(r chi.Router) {
			r.Get("/", toolHandler.ListTools)   // GET /api/v1/admin/tools
			r.Post("/", toolHandler.CreateTool) // POST /api/v1/admin/tools
		})

		// Task 3.9: Prompt Versioning routes
		r.Route("/admin/prompts", func(r chi.Router) {
			r.Get("/", promptHandler.List)                  // GET /api/v1/admin/prompts?agent_id={id}
			r.Post("/", promptHandler.Create)               // POST /api/v1/admin/prompts
			r.Put("/{id}/promote", promptHandler.Promote)   // PUT /api/v1/admin/prompts/{id}/promote
			r.Put("/{id}/rollback", promptHandler.Rollback) // PUT /api/v1/admin/prompts/{id}/rollback
		})

		r.Route("/copilot", func(r chi.Router) {
			r.Post("/chat", copilotChatHandler.Chat)                         // POST /api/v1/copilot/chat
			r.Post("/suggest-actions", copilotActionsHandler.SuggestActions) // POST /api/v1/copilot/suggest-actions
			r.Post("/summarize", copilotActionsHandler.Summarize)            // POST /api/v1/copilot/summarize
		})

		// Task 3.7: Agent Runtime routes
		agentOrchestrator := agent.NewOrchestrator(db)
		agentHandler := handlers.NewAgentHandler(agentOrchestrator)
		supportAgent := agents.NewSupportAgent(agentOrchestrator, toolRegistry, searchSvc)
		supportAgentHandler := handlers.NewSupportAgentHandler(supportAgent)
		// Task 4.5b — FR-231: Prospecting Agent wiring.
		prospectingAgent := agents.NewProspectingAgent(
			agentOrchestrator,
			toolRegistry,
			searchSvc,
			llmProvider,
			crm.NewLeadService(db),
			crm.NewAccountService(db),
			db,
		)
		prospectingAgentHandler := handlers.NewProspectingAgentHandler(prospectingAgent)

		// Task 3.8: Handoff Manager (reuses caseService + knowledgeBus from above)
		handoffService := agent.NewHandoffService(db, caseService, knowledgeBus)
		handoffHandler := handlers.NewHandoffHandler(handoffService)

		r.Route("/agents", func(r chi.Router) {
			r.Post("/trigger", agentHandler.TriggerAgent)                       // POST /api/v1/agents/trigger
			r.Get("/runs", agentHandler.ListAgentRuns)                          // GET  /api/v1/agents/runs
			r.Get("/runs/{id}", agentHandler.GetAgentRun)                       // GET  /api/v1/agents/runs/{id}
			r.Post("/runs/{id}/cancel", agentHandler.CancelAgentRun)            // POST /api/v1/agents/runs/{id}/cancel
			r.Get("/runs/{id}/handoff", handoffHandler.GetHandoffPackage)       // GET  /api/v1/agents/runs/{id}/handoff
			r.Post("/runs/{id}/handoff", handoffHandler.InitiateHandoff)        // POST /api/v1/agents/runs/{id}/handoff
			r.Get("/definitions", agentHandler.ListAgentDefinitions)            // GET  /api/v1/agents/definitions
			r.Post("/support/trigger", supportAgentHandler.TriggerSupportAgent) // POST /api/v1/agents/support/trigger
			r.Post("/prospecting/trigger", prospectingAgentHandler.TriggerProspectingAgent)
		})
	})

	return r
}
