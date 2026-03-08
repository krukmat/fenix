// Package agents provides concrete agent implementations.
// Task 3.7: Support Agent UC-C1 - tests
package agents

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

type mockKnowledgeSearch struct {
	results *knowledge.SearchResults
	err     error
}

func (m *mockKnowledgeSearch) HybridSearch(_ context.Context, _ knowledge.SearchInput) (*knowledge.SearchResults, error) {
	return m.results, m.err
}

func setupAgentTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func insertSupportAgentDefinition(t *testing.T, db *sql.DB, workspaceID string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('support-agent', ?, 'Support Agent', 'support', 'active')`,
		workspaceID,
	)
	if err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
}

func newTestSupportAgent(t *testing.T, db *sql.DB, search KnowledgeSearchInterface) *SupportAgent {
	t.Helper()
	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	if err := tool.RegisterBuiltInExecutors(registry, tool.BuiltinServices{
		DB:   db,
		Case: crm.NewCaseService(db),
	}); err != nil {
		t.Fatalf("register builtins: %v", err)
	}
	if err := registry.EnsureBuiltInToolDefinitionsForAllWorkspaces(context.Background()); err != nil {
		t.Fatalf("ensure builtins: %v", err)
	}
	return NewSupportAgentWithDB(orch, registry, search, db)
}

func TestSupportAgent_AllowedTools(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	tools := sa.AllowedTools()
	required := []string{"update_case", "send_reply", "create_task", "search_knowledge", "get_case", "get_contact"}
	if len(tools) != len(required) {
		t.Fatalf("expected %d tools, got %d", len(required), len(tools))
	}
	seen := make(map[string]bool, len(tools))
	for _, item := range tools {
		seen[item] = true
	}
	for _, item := range required {
		if !seen[item] {
			t.Fatalf("missing tool %s", item)
		}
	}
}

func TestDetermineAction_NoEvidence_Escalates(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	action := sa.determineAction(
		SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help", Priority: "high"},
		&CaseContext{ID: "case-1", WorkspaceID: "ws-1", Priority: "high"},
		emptyResults(),
	)
	if action.Type != supportActionEscalate {
		t.Fatalf("expected escalate, got %s", action.Type)
	}
}

func TestDetermineAction_HighScore_Resolves(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	action := sa.determineAction(
		SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help", Priority: "medium"},
		&CaseContext{ID: "case-1", WorkspaceID: "ws-1", Priority: "medium"},
		&knowledge.SearchResults{Items: []knowledge.SearchResult{{Score: 0.95}}},
	)
	if action.Type != supportActionUpdateCase {
		t.Fatalf("expected update_case, got %s", action.Type)
	}
}

func TestDetermineAction_MediumScore_Abstains(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	action := sa.determineAction(
		SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help", Priority: "medium"},
		&CaseContext{ID: "case-1", WorkspaceID: "ws-1", Priority: "medium"},
		&knowledge.SearchResults{Items: []knowledge.SearchResult{{Score: 0.7}}},
	)
	if action.Type != supportActionAbstain {
		t.Fatalf("expected abstain, got %s", action.Type)
	}
}

func TestSupportAgent_Run_MissingCaseID(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	_, err := sa.Run(context.Background(), SupportAgentConfig{WorkspaceID: "ws-1", CustomerQuery: "help"})
	if err != ErrCaseIDRequired {
		t.Fatalf("expected ErrCaseIDRequired, got %v", err)
	}
}

func TestSupportAgent_Run_MissingWorkspaceID(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	_, err := sa.Run(context.Background(), SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help"})
	if err != ErrWorkspaceIDRequired {
		t.Fatalf("expected ErrWorkspaceIDRequired, got %v", err)
	}
}

func TestSupportAgent_Run_EscalatesWhenNoKnowledge(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "high")
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	run, err := sa.Run(supportRunContext(context.Background(), wsID, ownerID), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        caseID,
		CustomerQuery: "I need help",
		Priority:      "high",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if stored.Status != agent.StatusEscalated {
		t.Fatalf("expected escalated, got %s", stored.Status)
	}

	caseTicket, err := crm.NewCaseService(db).Get(context.Background(), wsID, caseID)
	if err != nil {
		t.Fatalf("get case: %v", err)
	}
	if caseTicket.Status != agent.StatusEscalated {
		t.Fatalf("expected escalated case, got %s", caseTicket.Status)
	}
}

func TestSupportAgent_Run_ResolvesWhenHighConfidence(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "medium")
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{
		results: &knowledge.SearchResults{
			Items: []knowledge.SearchResult{{Score: 0.9, Snippet: "restart the service"}},
		},
	})

	run, err := sa.Run(supportRunContext(context.Background(), wsID, ownerID), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        caseID,
		CustomerQuery: "service is down",
		Priority:      "medium",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if stored.Status != agent.StatusSuccess {
		t.Fatalf("expected success, got %s", stored.Status)
	}

	caseTicket, err := crm.NewCaseService(db).Get(context.Background(), wsID, caseID)
	if err != nil {
		t.Fatalf("get case: %v", err)
	}
	if caseTicket.Status != "resolved" {
		t.Fatalf("expected resolved case, got %s", caseTicket.Status)
	}
}

func TestSupportAgent_Run_AbstainsWhenConfidenceIsMedium(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	wsID, ownerID := seedSupportWorkspace(t, db)
	insertSupportAgentDefinition(t, db, wsID)
	caseID := seedSupportCase(t, db, wsID, ownerID, "medium")
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{
		results: &knowledge.SearchResults{
			Items: []knowledge.SearchResult{{Score: 0.7, Snippet: "possible workaround"}},
		},
	})

	run, err := sa.Run(supportRunContext(context.Background(), wsID, ownerID), SupportAgentConfig{
		WorkspaceID:   wsID,
		CaseID:        caseID,
		CustomerQuery: "service is unstable",
		Priority:      "medium",
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), wsID, run.ID)
	if err != nil {
		t.Fatalf("load run: %v", err)
	}
	if stored.Status != agent.StatusAbstained {
		t.Fatalf("expected abstained, got %s", stored.Status)
	}

	caseTicket, err := crm.NewCaseService(db).Get(context.Background(), wsID, caseID)
	if err != nil {
		t.Fatalf("get case: %v", err)
	}
	if caseTicket.Status != "open" {
		t.Fatalf("expected open case, got %s", caseTicket.Status)
	}
}

func emptyResults() *knowledge.SearchResults {
	return &knowledge.SearchResults{Items: []knowledge.SearchResult{}}
}

func seedSupportWorkspace(t *testing.T, db *sql.DB) (string, string) {
	t.Helper()
	suffix := time.Now().UTC().Format("150405.000000000")
	wsID := "ws-support-" + suffix
	_, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, 'Support Workspace', ?, datetime('now'), datetime('now'))
	`, wsID, "support-"+suffix)
	if err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	ownerID := "user-support-" + suffix
	_, err = db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, 'Support Owner', 'active', datetime('now'), datetime('now'))
	`, ownerID, wsID, ownerID+"@example.com")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return wsID, ownerID
}

func seedSupportCase(t *testing.T, db *sql.DB, wsID, ownerID, priority string) string {
	t.Helper()
	contact, err := crm.NewContactService(db).Create(context.Background(), crm.CreateContactInput{
		WorkspaceID: wsID,
		FirstName:   "Ana",
		LastName:    "Cliente",
		Email:       "ana@example.com",
		Status:      "active",
		OwnerID:     ownerID,
	})
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}
	ticket, err := crm.NewCaseService(db).Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: wsID,
		ContactID:   contact.ID,
		OwnerID:     ownerID,
		Subject:     "Service issue",
		Description: "Customer cannot access the service",
		Priority:    priority,
		Status:      "open",
	})
	if err != nil {
		t.Fatalf("create case: %v", err)
	}
	return ticket.ID
}

func supportRunContext(ctx context.Context, workspaceID, ownerID string) context.Context {
	ctx = context.WithValue(ctx, ctxkeys.WorkspaceID, workspaceID)
	return context.WithValue(ctx, ctxkeys.UserID, ownerID)
}

func TestSupportAgent_NewSupportAgent_Constructor(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()
	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	sa := NewSupportAgent(orch, registry, &mockKnowledgeSearch{results: emptyResults()})
	if sa == nil {
		t.Fatal("NewSupportAgent returned nil")
	}
}

func TestSupportAgent_Objective_ReturnsJSON(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	obj := sa.Objective()
	if len(obj) == 0 {
		t.Fatal("Objective() returned empty")
	}
}

func TestSupportError_Error_ReturnsMessage(t *testing.T) {
	err := ErrSupportDBNotConfigured
	if err.Error() == "" {
		t.Fatal("SupportError.Error() should not be empty")
	}
}
