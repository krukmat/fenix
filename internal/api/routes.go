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
	domainaudit "github.com/matiasleandrokruk/fenix/internal/domain/audit"
	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/infra/config"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

// NewRouter creates and configures a new chi router with all routes.
// Task 1.3.8: Setup go-chi router with middleware + account endpoints
// Task 1.6.13: Public routes (/health, /auth/*) vs protected routes (/api/v1/*)
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
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
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

		// Account endpoints (Task 1.3.8)
		accountHandler := handlers.NewAccountHandler(crm.NewAccountService(db))
		contactHandler := handlers.NewContactHandler(crm.NewContactService(db))
		r.Route("/accounts", func(r chi.Router) {
			r.Post("/", accountHandler.CreateAccount)       // POST /api/v1/accounts
			r.Get("/", accountHandler.ListAccounts)         // GET /api/v1/accounts
			r.Get("/{id}", accountHandler.GetAccount)       // GET /api/v1/accounts/{id}
			r.Put("/{id}", accountHandler.UpdateAccount)    // PUT /api/v1/accounts/{id}
			r.Delete("/{id}", accountHandler.DeleteAccount) // DELETE /api/v1/accounts/{id}
			r.Get("/{account_id}/contacts", contactHandler.ListContactsByAccount)
		})

		r.Route("/contacts", func(r chi.Router) {
			r.Post("/", contactHandler.CreateContact)       // POST /api/v1/contacts
			r.Get("/", contactHandler.ListContacts)         // GET /api/v1/contacts
			r.Get("/{id}", contactHandler.GetContact)       // GET /api/v1/contacts/{id}
			r.Put("/{id}", contactHandler.UpdateContact)    // PUT /api/v1/contacts/{id}
			r.Delete("/{id}", contactHandler.DeleteContact) // DELETE /api/v1/contacts/{id}
		})

		// Lead endpoints (Task 1.5)
		leadHandler := handlers.NewLeadHandler(crm.NewLeadService(db))
		dealHandler := handlers.NewDealHandler(crm.NewDealService(db))
		caseHandler := handlers.NewCaseHandler(crm.NewCaseService(db))
		pipelineHandler := handlers.NewPipelineHandler(crm.NewPipelineService(db))
		activityHandler := handlers.NewActivityHandler(crm.NewActivityService(db))
		noteHandler := handlers.NewNoteHandler(crm.NewNoteService(db))
		attachmentHandler := handlers.NewAttachmentHandler(crm.NewAttachmentService(db))
		timelineHandler := handlers.NewTimelineHandler(crm.NewTimelineService(db))
		r.Route("/leads", func(r chi.Router) {
			r.Post("/", leadHandler.CreateLead)       // POST /api/v1/leads
			r.Get("/", leadHandler.ListLeads)         // GET /api/v1/leads
			r.Get("/{id}", leadHandler.GetLead)       // GET /api/v1/leads/{id}
			r.Put("/{id}", leadHandler.UpdateLead)    // PUT /api/v1/leads/{id}
			r.Delete("/{id}", leadHandler.DeleteLead) // DELETE /api/v1/leads/{id}
		})

		r.Route("/deals", func(r chi.Router) {
			r.Post("/", dealHandler.CreateDeal)
			r.Get("/", dealHandler.ListDeals)
			r.Get("/{id}", dealHandler.GetDeal)
			r.Put("/{id}", dealHandler.UpdateDeal)
			r.Delete("/{id}", dealHandler.DeleteDeal)
		})

		r.Route("/cases", func(r chi.Router) {
			r.Post("/", caseHandler.CreateCase)
			r.Get("/", caseHandler.ListCases)
			r.Get("/{id}", caseHandler.GetCase)
			r.Put("/{id}", caseHandler.UpdateCase)
			r.Delete("/{id}", caseHandler.DeleteCase)
		})

		r.Route("/pipelines", func(r chi.Router) {
			r.Post("/", pipelineHandler.CreatePipeline)
			r.Get("/", pipelineHandler.ListPipelines)
			r.Get("/{id}", pipelineHandler.GetPipeline)
			r.Put("/{id}", pipelineHandler.UpdatePipeline)
			r.Delete("/{id}", pipelineHandler.DeletePipeline)
			r.Post("/{id}/stages", pipelineHandler.CreateStage)
			r.Get("/{id}/stages", pipelineHandler.ListStages)
			r.Put("/stages/{stage_id}", pipelineHandler.UpdateStage)
			r.Delete("/stages/{stage_id}", pipelineHandler.DeleteStage)
		})

		r.Route("/activities", func(r chi.Router) {
			r.Post("/", activityHandler.CreateActivity)
			r.Get("/", activityHandler.ListActivities)
			r.Get("/{id}", activityHandler.GetActivity)
			r.Put("/{id}", activityHandler.UpdateActivity)
			r.Delete("/{id}", activityHandler.DeleteActivity)
		})

		r.Route("/notes", func(r chi.Router) {
			r.Post("/", noteHandler.CreateNote)
			r.Get("/", noteHandler.ListNotes)
			r.Get("/{id}", noteHandler.GetNote)
			r.Put("/{id}", noteHandler.UpdateNote)
			r.Delete("/{id}", noteHandler.DeleteNote)
		})

		r.Route("/attachments", func(r chi.Router) {
			r.Post("/", attachmentHandler.CreateAttachment)
			r.Get("/", attachmentHandler.ListAttachments)
			r.Get("/{id}", attachmentHandler.GetAttachment)
			r.Delete("/{id}", attachmentHandler.DeleteAttachment)
		})

		r.Route("/timeline", func(r chi.Router) {
			r.Get("/", timelineHandler.ListTimeline)
			r.Get("/{entity_type}/{entity_id}", timelineHandler.ListTimelineByEntity)
		})

		// Task 2.2: Knowledge ingestion pipeline
		// Task 2.4: Shared event bus — IngestService publishes, EmbedderService subscribes
		knowledgeBus := eventbus.New()
		ingestSvc := knowledge.NewIngestService(db, knowledgeBus)

		// Task 2.4: EmbedderService — async goroutine processes knowledge.ingested events
		cfg := config.Load()
		llmProvider := llm.NewOllamaProvider(cfg.OllamaBaseURL, cfg.OllamaModel)
		embedder := knowledge.NewEmbedderService(db, llmProvider)
		go embedder.Start(context.Background(), knowledgeBus)

		knowledgeIngestHandler := handlers.NewKnowledgeIngestHandler(ingestSvc)
		r.Route("/knowledge", func(r chi.Router) {
			r.Post("/ingest", knowledgeIngestHandler.Ingest) // POST /api/v1/knowledge/ingest
		})
	})

	return r
}
