// Package agents provides concrete agent implementations.
// Task 4.5b — FR-231: Prospecting Agent tests
package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
	_ "modernc.org/sqlite"
)

type mockLLMProvider struct {
	content string
	tokens  int
	err     error
}

func (m *mockLLMProvider) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.ChatResponse{Content: m.content, Tokens: m.tokens}, nil
}

func (m *mockLLMProvider) Embed(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
	return &llm.EmbedResponse{}, nil
}

func (m *mockLLMProvider) ModelInfo() llm.ModelMeta {
	return llm.ModelMeta{ID: "test", Provider: "mock", Version: "v1", MaxTokens: 1024}
}

func (m *mockLLMProvider) HealthCheck(_ context.Context) error { return nil }

type mockLeadGetter struct {
	lead *crm.Lead
	err  error
}

func (m *mockLeadGetter) Get(_ context.Context, _, _ string) (*crm.Lead, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.lead, nil
}

type mockAccountGetter struct {
	account *crm.Account
	err     error
}

func (m *mockAccountGetter) Get(_ context.Context, _, _ string) (*crm.Account, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.account, nil
}

func setupProspectingTestDB(t *testing.T) *sql.DB {
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

func insertProspectingAgentDefinition(t *testing.T, db *sql.DB, workspaceID string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('prospecting-agent', ?, 'Prospecting Agent', 'prospecting', 'active')`, workspaceID)
	if err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
}

func insertProspectingTestUser(t *testing.T, db *sql.DB, workspaceID string) string {
	t.Helper()
	userID := uuid.NewV7().String()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO user_account (id, workspace_id, email, password_hash, display_name, status, created_at, updated_at)
		 VALUES (?, ?, ?, 'x', 'owner', 'active', ?, ?)`,
		userID,
		workspaceID,
		userID+"@example.com",
		now,
		now,
	)
	if err != nil {
		t.Fatalf("insert user_account: %v", err)
	}
	return userID
}

func newTestProspectingAgent(
	t *testing.T,
	db *sql.DB,
	search KnowledgeSearchInterface,
	provider llm.LLMProvider,
	lead LeadGetter,
	account AccountGetter,
) *ProspectingAgent {
	t.Helper()
	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	if err := registry.Register(tool.BuiltinCreateTask, tool.NewCreateTaskExecutor(db)); err != nil {
		t.Fatalf("register create_task: %v", err)
	}
	return NewProspectingAgent(orch, registry, search, provider, lead, account, db)
}

// Task 4.5b — TDD 1/5.
func TestProspectingAgent_AllowedTools(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()

	a := newTestProspectingAgent(t, db, &mockKnowledgeSearch{results: emptyResults()}, &mockLLMProvider{}, &mockLeadGetter{}, &mockAccountGetter{})
	tools := a.AllowedTools()
	want := []string{"search_knowledge", "create_task", "get_lead", "get_account"}
	if len(tools) != len(want) {
		t.Fatalf("expected %d tools, got %d", len(want), len(tools))
	}
	for i := range want {
		if tools[i] != want[i] {
			t.Fatalf("tool[%d]=%s want=%s", i, tools[i], want[i])
		}
	}
}

// Task 4.5b — TDD 2/5.
func TestProspectingAgent_Run_ConfigValidation(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()

	a := newTestProspectingAgent(t, db, &mockKnowledgeSearch{results: emptyResults()}, &mockLLMProvider{}, &mockLeadGetter{}, &mockAccountGetter{})
	_, err := a.Run(context.Background(), ProspectingAgentConfig{WorkspaceID: "ws-1"})
	if err != ErrLeadIDRequired {
		t.Fatalf("expected ErrLeadIDRequired, got %v", err)
	}
}

// Task 4.5b — TDD 3/5.
func TestProspectingAgent_Run_HighConfidence_DraftsOutreach(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertProspectingAgentDefinition(t, db, "ws-1")
	ownerID := insertProspectingTestUser(t, db, "ws-1")

	leadID := "lead-1"
	accountID := "acc-1"
	a := newTestProspectingAgent(t, db,
		&mockKnowledgeSearch{results: &knowledge.SearchResults{Items: []knowledge.SearchResult{{Score: 0.9}}}},
		&mockLLMProvider{content: "Hola, ¿agendamos una llamada breve esta semana?", tokens: 32},
		&mockLeadGetter{lead: &crm.Lead{ID: leadID, AccountID: &accountID, Status: "new", OwnerID: ownerID}},
		&mockAccountGetter{account: &crm.Account{ID: accountID, Name: "Acme"}},
	)

	run, err := a.Run(context.Background(), ProspectingAgentConfig{WorkspaceID: "ws-1", LeadID: leadID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run == nil {
		t.Fatal("expected run")
	}
	stored, getErr := agent.NewOrchestrator(db).GetAgentRun(context.Background(), "ws-1", run.ID)
	if getErr != nil {
		t.Fatalf("GetAgentRun: %v", getErr)
	}
	if stored.Status != agent.StatusSuccess {
		t.Fatalf("status=%s want=%s", stored.Status, agent.StatusSuccess)
	}
	var output struct {
		Action     string  `json:"action"`
		LeadID     string  `json:"lead_id"`
		Confidence float64 `json:"confidence"`
		Details    struct {
			Draft  string `json:"draft"`
			TaskID string `json:"task_id"`
		} `json:"details"`
	}
	if err := json.Unmarshal(stored.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Action != "draft_outreach" {
		t.Fatalf("action=%s want=draft_outreach", output.Action)
	}
	if output.Details.Draft == "" || output.Details.TaskID == "" {
		t.Fatalf("expected draft and task_id in details, got %+v", output.Details)
	}
	if output.Confidence <= 0.6 {
		t.Fatalf("confidence=%f want > 0.6", output.Confidence)
	}
}

// Task 4.5b — TDD 4/5.
func TestProspectingAgent_Run_LowConfidence_Skips(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertProspectingAgentDefinition(t, db, "ws-1")

	leadID := "lead-2"
	a := newTestProspectingAgent(t, db,
		&mockKnowledgeSearch{results: &knowledge.SearchResults{Items: []knowledge.SearchResult{{Score: 0.4}}}},
		&mockLLMProvider{},
		&mockLeadGetter{lead: &crm.Lead{ID: leadID, Status: "new"}},
		&mockAccountGetter{},
	)

	run, err := a.Run(context.Background(), ProspectingAgentConfig{WorkspaceID: "ws-1", LeadID: leadID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, getErr := agent.NewOrchestrator(db).GetAgentRun(context.Background(), "ws-1", run.ID)
	if getErr != nil {
		t.Fatalf("GetAgentRun: %v", getErr)
	}
	if !contains(string(stored.Output), "\"skip\"") {
		t.Fatalf("output=%s expected skip", string(stored.Output))
	}
}

// Task 4.5b — TDD 5/5.
func TestProspectingAgent_Run_MissingLead_Error(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertProspectingAgentDefinition(t, db, "ws-1")

	a := newTestProspectingAgent(t, db,
		&mockKnowledgeSearch{results: emptyResults()},
		&mockLLMProvider{},
		&mockLeadGetter{err: sql.ErrNoRows},
		&mockAccountGetter{},
	)

	_, err := a.Run(context.Background(), ProspectingAgentConfig{WorkspaceID: "ws-1", LeadID: "missing"})
	if err != ErrLeadNotFound {
		t.Fatalf("expected ErrLeadNotFound, got %v", err)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
