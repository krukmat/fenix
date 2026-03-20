// Package agents provides concrete agent implementations.
// Task 4.5c — FR-231: KB Agent tests
package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
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
	ensureAgentTestWorkspace(t, db, workspaceID)
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

func TestKBAgent_Run_HighSensitivity_CreatesApprovalAndBlocksMutation(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertKBAgentDefinition(t, db, "ws-1")
	ownerID := insertProspectingTestUser(t, db, "ws-1")

	highSensitivityMeta := `{"sensitivity":"high"}`
	a := newTestKBAgent(
		t,
		db,
		&mockKnowledgeSearch{results: &knowledge.SearchResults{Items: []knowledge.SearchResult{{Score: 0.4}}}},
		&mockKBCaseGetter{caseTicket: &crm.CaseTicket{ID: "case-hs-1", WorkspaceID: "ws-1", Subject: "Reset", Status: "resolved", OwnerID: ownerID, Metadata: &highSensitivityMeta}},
		mustJSON(map[string]any{"knowledge_item_id": "kb-new"}),
		mustJSON(map[string]any{"knowledge_item_id": "kb-upd"}),
	)

	run, err := a.Run(context.Background(), KBAgentConfig{WorkspaceID: "ws-1", CaseID: "case-hs-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, getErr := agent.NewOrchestrator(db).GetAgentRun(context.Background(), "ws-1", run.ID)
	if getErr != nil {
		t.Fatalf("GetAgentRun: %v", getErr)
	}
	if stored.Status != agent.StatusEscalated {
		t.Fatalf("status=%s want=%s", stored.Status, agent.StatusEscalated)
	}

	var output struct {
		Action     string `json:"action"`
		Reason     string `json:"reason"`
		ApprovalID string `json:"approval_id"`
		ArticleID  string `json:"article_id"`
	}
	if unmarshalErr := json.Unmarshal(stored.Output, &output); unmarshalErr != nil {
		t.Fatalf("unmarshal output: %v", unmarshalErr)
	}
	if output.Action != "pending_approval" {
		t.Fatalf("action=%s want=pending_approval", output.Action)
	}
	if output.Reason != "high_sensitivity" {
		t.Fatalf("reason=%s want=high_sensitivity", output.Reason)
	}
	if output.ApprovalID == "" {
		t.Fatalf("expected non-empty approval_id")
	}
	if output.ArticleID != "" {
		t.Fatalf("article_id=%q want empty", output.ArticleID)
	}

	var (
		status string
		action string
	)
	err = db.QueryRowContext(context.Background(), `
		SELECT status, action
		FROM approval_request
		WHERE id = ?
	`, output.ApprovalID).Scan(&status, &action)
	if err != nil {
		t.Fatalf("query approval_request: %v", err)
	}
	if status != "pending" {
		t.Fatalf("approval status=%s want=pending", status)
	}
	if action != "kb.article.mutation" {
		t.Fatalf("approval action=%s want=kb.article.mutation", action)
	}
}

func TestKBAgent_Objective_ReturnsJSON(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	a := newTestKBAgent(t, db, &mockKnowledgeSearch{results: emptyResults()}, &mockKBCaseGetter{}, nil, nil)
	obj := a.Objective()
	if len(obj) == 0 {
		t.Fatal("Objective() returned empty")
	}
	var m map[string]any
	if err := json.Unmarshal(obj, &m); err != nil {
		t.Fatalf("Objective() not valid JSON: %v", err)
	}
}

func TestKBError_Error_ReturnsMessage(t *testing.T) {
	err := ErrKBCaseIDRequired
	if err.Error() == "" {
		t.Fatal("KBError.Error() should not be empty")
	}
}

func TestKBHelperFunctionsAndMutationFallbacks(t *testing.T) {
	if got := plannedKBTool(0.9); got != tool.BuiltinUpdateKnowledgeItem {
		t.Fatalf("plannedKBTool(high)=%q want %q", got, tool.BuiltinUpdateKnowledgeItem)
	}
	if got := plannedKBTool(0.2); got != tool.BuiltinCreateKnowledgeItem {
		t.Fatalf("plannedKBTool(low)=%q want %q", got, tool.BuiltinCreateKnowledgeItem)
	}

	ownerID := "owner-1"
	if got := resolveKBAgentUserID(ownerID, nil); got != ownerID {
		t.Fatalf("resolveKBAgentUserID(nil)=%q want %q", got, ownerID)
	}
	empty := ""
	if got := resolveKBAgentUserID(ownerID, &empty); got != ownerID {
		t.Fatalf("resolveKBAgentUserID(empty)=%q want %q", got, ownerID)
	}
	triggered := "triggered-user"
	if got := resolveKBAgentUserID(ownerID, &triggered); got != triggered {
		t.Fatalf("resolveKBAgentUserID(triggered)=%q want %q", got, triggered)
	}
}

func TestKBAgent_CreateAndUpdateKnowledgeArticleFallbacks(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()

	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	if err := registry.Register(tool.BuiltinUpdateKnowledgeItem, &mockToolExecutor{out: mustJSON(map[string]any{})}); err != nil {
		t.Fatalf("register update_knowledge_item: %v", err)
	}
	if err := registry.Register(tool.BuiltinCreateKnowledgeItem, &mockToolExecutor{out: mustJSON(map[string]any{})}); err != nil {
		t.Fatalf("register create_knowledge_item: %v", err)
	}
	a := NewKBAgent(orch, registry, &mockKnowledgeSearch{results: emptyResults()}, nil, &mockKBCaseGetter{}, db)

	if got, err := a.updateKnowledgeArticle(context.Background(), "kb-existing", "Subject", "Body"); err != nil || got != "kb-existing" {
		t.Fatalf("updateKnowledgeArticle() got (%q, %v) want (kb-existing, nil)", got, err)
	}
	if _, err := a.createKnowledgeArticle(context.Background(), "ws-1", "Subject", "Body"); !errors.Is(err, ErrKBArticleCreationFailed) {
		t.Fatalf("createKnowledgeArticle() error = %v want %v", err, ErrKBArticleCreationFailed)
	}
}

func TestKBAgent_CheckDailyLimits(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()

	a := newTestKBAgent(t, db, &mockKnowledgeSearch{results: emptyResults()}, &mockKBCaseGetter{}, nil, nil)
	if err := a.checkDailyLimits(context.Background(), "ws-kb-ok"); err != nil {
		t.Fatalf("empty checkDailyLimits() error = %v", err)
	}

	workspaceID := "ws-kb-limit"
	insertKBAgentDefinition(t, db, workspaceID)
	now := time.Now().UTC().Format(time.RFC3339)
	for i := 0; i < 10; i++ {
		_, err := db.ExecContext(context.Background(), `
			INSERT INTO agent_run (id, workspace_id, agent_definition_id, trigger_type, status, started_at, created_at)
			VALUES (?, ?, 'kb-agent', 'manual', 'success', ?, ?)
		`, uuid.NewV7().String(), workspaceID, now, now)
		if err != nil {
			t.Fatalf("insert agent_run #%d: %v", i, err)
		}
	}
	if err := a.checkDailyLimits(context.Background(), workspaceID); err != ErrKBDailyLimitExceeded {
		t.Fatalf("checkDailyLimits() error = %v want %v", err, ErrKBDailyLimitExceeded)
	}
}
