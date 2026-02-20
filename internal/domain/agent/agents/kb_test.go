// Package agents provides concrete agent implementations.
// Task 4.5c — FR-231: KB Agent tests
package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

type mockKBCaseGetter struct {
	caseTicket *crm.CaseTicket
	err        error
}

func (m *mockKBCaseGetter) Get(_ context.Context, _, _ string) (*crm.CaseTicket, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.caseTicket, nil
}

type mockToolExecutor struct {
	out json.RawMessage
	err error
}

func (m *mockToolExecutor) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.out, nil
}

func insertKBAgentDefinition(t *testing.T, db *sql.DB, workspaceID string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('kb-agent', ?, 'KB Agent', 'kb', 'active')`, workspaceID)
	if err != nil {
		t.Fatalf("insert kb agent_definition: %v", err)
	}
}

func newTestKBAgent(
	t *testing.T,
	db *sql.DB,
	search KnowledgeSearchInterface,
	caseGetter KBCaseGetter,
	createOut json.RawMessage,
	updateOut json.RawMessage,
) *KBAgent {
	t.Helper()
	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	if err := registry.Register(tool.BuiltinCreateKnowledgeItem, &mockToolExecutor{out: createOut}); err != nil {
		t.Fatalf("register create_knowledge_item: %v", err)
	}
	if err := registry.Register(tool.BuiltinUpdateKnowledgeItem, &mockToolExecutor{out: updateOut}); err != nil {
		t.Fatalf("register update_knowledge_item: %v", err)
	}
	return NewKBAgent(orch, registry, search, nil, caseGetter, db)
}

// Task 4.5c — TDD 1/4.
func TestKBAgent_AllowedTools(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()

	a := newTestKBAgent(t, db, &mockKnowledgeSearch{results: emptyResults()}, &mockKBCaseGetter{}, mustJSON(map[string]any{"knowledge_item_id": "k1"}), mustJSON(map[string]any{"knowledge_item_id": "k1"}))
	tools := a.AllowedTools()
	want := []string{"search_knowledge", "create_knowledge_item", "update_knowledge_item"}
	if len(tools) != len(want) {
		t.Fatalf("expected %d tools, got %d", len(want), len(tools))
	}
	for i := range want {
		if tools[i] != want[i] {
			t.Fatalf("tool[%d]=%s want=%s", i, tools[i], want[i])
		}
	}
}

// Task 4.5c — TDD 2/4.
func TestKBAgent_Run_ResolvedCase_CreatesArticle(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertKBAgentDefinition(t, db, "ws-1")

	a := newTestKBAgent(
		t,
		db,
		&mockKnowledgeSearch{results: &knowledge.SearchResults{Items: []knowledge.SearchResult{{Score: 0.4}}}},
		&mockKBCaseGetter{caseTicket: &crm.CaseTicket{ID: "case-1", WorkspaceID: "ws-1", Subject: "Reset", Status: "resolved", OwnerID: "owner-1"}},
		mustJSON(map[string]any{"knowledge_item_id": "kb-new"}),
		mustJSON(map[string]any{"knowledge_item_id": "kb-upd"}),
	)

	run, err := a.Run(context.Background(), KBAgentConfig{WorkspaceID: "ws-1", CaseID: "case-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, getErr := agent.NewOrchestrator(db).GetAgentRun(context.Background(), "ws-1", run.ID)
	if getErr != nil {
		t.Fatalf("GetAgentRun: %v", getErr)
	}
	var output struct {
		Action    string `json:"action"`
		ArticleID string `json:"article_id"`
	}
	if unmarshalErr := json.Unmarshal(stored.Output, &output); unmarshalErr != nil {
		t.Fatalf("unmarshal output: %v", unmarshalErr)
	}
	if output.Action != "created" {
		t.Fatalf("action=%s want=created", output.Action)
	}
	if output.ArticleID == "" {
		t.Fatal("expected non-empty article_id")
	}
}

// Task 4.5c — TDD 3/4.
func TestKBAgent_Run_DuplicateFound_Updates(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertKBAgentDefinition(t, db, "ws-1")

	a := newTestKBAgent(
		t,
		db,
		&mockKnowledgeSearch{results: &knowledge.SearchResults{Items: []knowledge.SearchResult{{KnowledgeItemID: "kb-existing", Score: 0.95}}}},
		&mockKBCaseGetter{caseTicket: &crm.CaseTicket{ID: "case-2", WorkspaceID: "ws-1", Subject: "VPN", Status: "resolved", OwnerID: "owner-1"}},
		mustJSON(map[string]any{"knowledge_item_id": "kb-new"}),
		mustJSON(map[string]any{"knowledge_item_id": "kb-existing"}),
	)

	run, err := a.Run(context.Background(), KBAgentConfig{WorkspaceID: "ws-1", CaseID: "case-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, getErr := agent.NewOrchestrator(db).GetAgentRun(context.Background(), "ws-1", run.ID)
	if getErr != nil {
		t.Fatalf("GetAgentRun: %v", getErr)
	}
	var output struct {
		Action    string `json:"action"`
		ArticleID string `json:"article_id"`
	}
	if unmarshalErr := json.Unmarshal(stored.Output, &output); unmarshalErr != nil {
		t.Fatalf("unmarshal output: %v", unmarshalErr)
	}
	if output.Action != "updated" {
		t.Fatalf("action=%s want=updated", output.Action)
	}
	if output.ArticleID != "kb-existing" {
		t.Fatalf("article_id=%s want=kb-existing", output.ArticleID)
	}
}

// Task 4.5c — TDD 4/4.
func TestKBAgent_Run_UnresolvedCase_Error(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertKBAgentDefinition(t, db, "ws-1")

	a := newTestKBAgent(
		t,
		db,
		&mockKnowledgeSearch{results: emptyResults()},
		&mockKBCaseGetter{caseTicket: &crm.CaseTicket{ID: "case-3", WorkspaceID: "ws-1", Subject: "Issue", Status: "open", OwnerID: "owner-1"}},
		mustJSON(map[string]any{"knowledge_item_id": "kb-new"}),
		mustJSON(map[string]any{"knowledge_item_id": "kb-existing"}),
	)

	_, err := a.Run(context.Background(), KBAgentConfig{WorkspaceID: "ws-1", CaseID: "case-3"})
	if !errors.Is(err, ErrCaseNotResolved) {
		t.Fatalf("expected ErrCaseNotResolved, got %v", err)
	}
}
