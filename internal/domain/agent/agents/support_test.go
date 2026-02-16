// Package agents provides concrete agent implementations.
// Task 3.7: Support Agent UC-C1 — tests
package agents

import (
	"context"
	"database/sql"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	_ "modernc.org/sqlite"
)

// mockKnowledgeSearch implements KnowledgeSearchInterface for tests.
type mockKnowledgeSearch struct {
	results *knowledge.SearchResults
	err     error
}

func (m *mockKnowledgeSearch) HybridSearch(_ context.Context, _ knowledge.SearchInput) (*knowledge.SearchResults, error) {
	return m.results, m.err
}

// setupAgentTestDB creates an in-memory SQLite DB with all migrations applied.
func setupAgentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// insertSupportAgentDefinition inserts the "support-agent" definition required by SupportAgent.Run.
func insertSupportAgentDefinition(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('support-agent', '', 'Support Agent', 'support', 'active')`)
	if err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
}

// newTestSupportAgent creates a SupportAgent wired to a real in-memory DB.
func newTestSupportAgent(t *testing.T, db *sql.DB, search KnowledgeSearchInterface) *SupportAgent {
	t.Helper()
	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	return NewSupportAgent(orch, registry, search)
}

// TestSupportAgent_AllowedTools verifies the tool list matches UC-C1 spec.
// Traces: FR-230
func TestSupportAgent_AllowedTools(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	tools := sa.AllowedTools()

	required := []string{"update_case", "send_reply", "create_task", "search_knowledge", "get_case", "get_contact"}
	if len(tools) != len(required) {
		t.Fatalf("expected %d tools, got %d: %v", len(required), len(tools), tools)
	}
	toolSet := make(map[string]bool, len(tools))
	for _, tl := range tools {
		toolSet[tl] = true
	}
	for _, req := range required {
		if !toolSet[req] {
			t.Errorf("missing required tool: %s", req)
		}
	}
}

// TestSupportAgent_Objective returns valid JSON with role and goal.
// Traces: FR-230
func TestSupportAgent_Objective(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	obj := sa.Objective()

	if len(obj) == 0 {
		t.Fatal("Objective() returned empty JSON")
	}
}

// TestDetermineAction_NoEvidence_Escalates verifies escalation when KB returns nothing.
// Traces: FR-230, FR-231
func TestDetermineAction_NoEvidence_Escalates(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	config := SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help"}
	ctx := &CaseContext{ID: "case-1", WorkspaceID: "ws-1"}
	evidence := emptyResults()

	action := sa.determineAction(config, ctx, evidence)
	if action.Type != "escalate" {
		t.Errorf("expected escalate, got %s", action.Type)
	}
}

// TestDetermineAction_HighScore_Resolves verifies resolution when KB has high-confidence result.
// Traces: FR-230, FR-231
func TestDetermineAction_HighScore_Resolves(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	config := SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help"}
	ctx := &CaseContext{ID: "case-1", WorkspaceID: "ws-1"}
	evidence := &knowledge.SearchResults{
		Items: []knowledge.SearchResult{{Score: 0.95}},
	}

	action := sa.determineAction(config, ctx, evidence)
	if action.Type != "update_case" {
		t.Errorf("expected update_case, got %s", action.Type)
	}
	if action.Status != "resolved" {
		t.Errorf("expected resolved, got %s", action.Status)
	}
}

// TestDetermineAction_MediumScore_CreateTask verifies task creation for medium confidence.
// Traces: FR-230, FR-231
func TestDetermineAction_MediumScore_CreateTask(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	config := SupportAgentConfig{CaseID: "case-1", CustomerQuery: "help"}
	ctx := &CaseContext{ID: "case-1", WorkspaceID: "ws-1"}
	evidence := &knowledge.SearchResults{
		Items: []knowledge.SearchResult{{Score: 0.5}},
	}

	action := sa.determineAction(config, ctx, evidence)
	if action.Type != "create_task" {
		t.Errorf("expected create_task, got %s", action.Type)
	}
}

// TestSupportAgent_Run_MissingCaseID returns ErrCaseIDRequired.
// Traces: FR-230
func TestSupportAgent_Run_MissingCaseID(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	_, err := sa.Run(context.Background(), SupportAgentConfig{CustomerQuery: "help"})
	if err == nil {
		t.Fatal("expected error for missing case_id, got nil")
	}
	if err != ErrCaseIDRequired {
		t.Errorf("expected ErrCaseIDRequired, got: %v", err)
	}
}

// TestSupportAgent_Run_EscalatesWhenNoKnowledge verifies full run with empty KB.
// Traces: FR-230, FR-231
func TestSupportAgent_Run_EscalatesWhenNoKnowledge(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()
	insertSupportAgentDefinition(t, db)

	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	run, err := sa.Run(context.Background(), SupportAgentConfig{
		CaseID:        "case-123",
		CustomerQuery: "I need help",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run == nil {
		t.Fatal("expected non-nil run")
	}
}

// TestSupportAgent_Run_ResolvesWhenHighConfidence verifies full run with high-score KB result.
// Traces: FR-230, FR-231
func TestSupportAgent_Run_ResolvesWhenHighConfidence(t *testing.T) {
	db := setupAgentTestDB(t)
	defer db.Close()
	insertSupportAgentDefinition(t, db)

	highConfidence := &knowledge.SearchResults{
		Items: []knowledge.SearchResult{{Score: 0.9, Snippet: "Solution: restart the service."}},
	}
	sa := newTestSupportAgent(t, db, &mockKnowledgeSearch{results: highConfidence})

	run, err := sa.Run(context.Background(), SupportAgentConfig{
		CaseID:        "case-456",
		CustomerQuery: "service is down",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run == nil {
		t.Fatal("expected non-nil run")
	}
}

// emptyResults returns an empty SearchResults for test use.
func emptyResults() *knowledge.SearchResults {
	return &knowledge.SearchResults{Items: []knowledge.SearchResult{}}
}
