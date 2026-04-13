package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

type mockDealGetter struct {
	deal *crm.Deal
	err  error
}

func (m *mockDealGetter) Get(_ context.Context, _, _ string) (*crm.Deal, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.deal, nil
}

type mockDealToolExecutor struct{ getter DealGetter }

func (m *mockDealToolExecutor) Execute(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
	var in struct {
		DealID string `json:"deal_id"`
	}
	if err := json.Unmarshal(params, &in); err != nil {
		return nil, err
	}
	deal, err := m.getter.Get(ctx, "", in.DealID)
	if err != nil {
		return nil, err
	}
	return mustJSON(map[string]any{"deal": deal}), nil
}

func insertDealRiskAgentDefinition(t *testing.T, db *sql.DB, workspaceID string) {
	t.Helper()
	ensureAgentTestWorkspace(t, db, workspaceID)
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('deal-risk-agent', ?, 'Deal Risk Agent', 'deal-risk', 'active')`, workspaceID)
	if err != nil {
		t.Fatalf("insert agent_definition: %v", err)
	}
}

func newTestDealRiskAgent(
	t *testing.T,
	db *sql.DB,
	search KnowledgeSearchInterface,
	dealGetter DealGetter,
	accountGetter AccountGetter,
) *DealRiskAgent {
	t.Helper()
	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	if err := registry.Register(tool.BuiltinCreateTask, tool.NewCreateTaskExecutor(db)); err != nil {
		t.Fatalf("register create_task: %v", err)
	}
	if err := registry.Register(tool.BuiltinGetDeal, &mockDealToolExecutor{getter: dealGetter}); err != nil {
		t.Fatalf("register get_deal: %v", err)
	}
	if err := registry.Register(tool.BuiltinGetAccount, &mockAccountToolExecutor{getter: accountGetter}); err != nil {
		t.Fatalf("register get_account: %v", err)
	}
	return NewDealRiskAgent(orch, registry, search, nil, dealGetter, accountGetter, db)
}

func TestDealRiskAgent_Run_MissingDealID(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()

	a := newTestDealRiskAgent(t, db, &mockKnowledgeSearch{results: emptyResults()}, &mockDealGetter{}, &mockAccountGetter{})
	_, err := a.Run(context.Background(), DealRiskAgentConfig{WorkspaceID: "ws-1"})
	if err != ErrDealIDRequired {
		t.Fatalf("expected ErrDealIDRequired, got %v", err)
	}
}

func TestDealRiskAgent_Run_DealNotFound(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertDealRiskAgentDefinition(t, db, "ws-1")

	a := newTestDealRiskAgent(t, db, &mockKnowledgeSearch{results: emptyResults()}, &mockDealGetter{err: sql.ErrNoRows}, &mockAccountGetter{})
	_, err := a.Run(context.Background(), DealRiskAgentConfig{WorkspaceID: "ws-1", DealID: "missing"})
	if err != ErrDealNotFound {
		t.Fatalf("expected ErrDealNotFound, got %v", err)
	}
}

func TestDealRiskAgent_Run_RiskDetected(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertDealRiskAgentDefinition(t, db, "ws-1")
	ownerID := insertProspectingTestUser(t, db, "ws-1")

	dealID := "deal-1"
	accountID := "acc-1"
	now := time.Now().UTC()
	a := newTestDealRiskAgent(
		t,
		db,
		&mockKnowledgeSearch{results: &knowledge.SearchResults{Items: []knowledge.SearchResult{{KnowledgeItemID: "ev-1", Score: 0.91, Snippet: "No meetings in 21 days"}}}},
		&mockDealGetter{deal: &crm.Deal{
			ID:        dealID,
			AccountID: accountID,
			OwnerID:   ownerID,
			Title:     "Expansion ACME",
			Status:    "open",
			CreatedAt: now.Add(-45 * 24 * time.Hour),
			UpdatedAt: now.Add(-31 * 24 * time.Hour),
		}},
		&mockAccountGetter{account: &crm.Account{ID: accountID, Name: "Acme"}},
	)

	run, err := a.Run(context.Background(), DealRiskAgentConfig{WorkspaceID: "ws-1", DealID: dealID})
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
		Action  string          `json:"action"`
		DealID  string          `json:"deal_id"`
		TaskID  string          `json:"task_id"`
		Signals DealRiskSignals `json:"signals"`
	}
	if err := json.Unmarshal(stored.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Action != "create_task" {
		t.Fatalf("action=%s want=create_task", output.Action)
	}
	if output.TaskID == "" {
		t.Fatal("expected task_id in output")
	}
	if !output.Signals.Stale || output.Signals.RiskLevel != dealRiskLevelHigh {
		t.Fatalf("signals=%+v want stale high risk", output.Signals)
	}
	if string(stored.ToolCalls) == "" || !contains(string(stored.ToolCalls), `"create_task"`) {
		t.Fatalf("tool_calls=%s expected create_task", string(stored.ToolCalls))
	}
}

func TestDealRiskAgent_Run_NoRisk(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertDealRiskAgentDefinition(t, db, "ws-1")

	dealID := "deal-2"
	accountID := "acc-2"
	now := time.Now().UTC()
	a := newTestDealRiskAgent(
		t,
		db,
		&mockKnowledgeSearch{results: &knowledge.SearchResults{Items: []knowledge.SearchResult{
			{KnowledgeItemID: "ev-1", Score: 0.82, Snippet: "Customer confirmed next step"},
			{KnowledgeItemID: "ev-2", Score: 0.77, Snippet: "Recent meeting logged"},
		}}},
		&mockDealGetter{deal: &crm.Deal{
			ID:        dealID,
			AccountID: accountID,
			Title:     "Renewal ACME",
			Status:    "open",
			CreatedAt: now.Add(-7 * 24 * time.Hour),
			UpdatedAt: now.Add(-2 * 24 * time.Hour),
		}},
		&mockAccountGetter{account: &crm.Account{ID: accountID, Name: "Acme"}},
	)

	run, err := a.Run(context.Background(), DealRiskAgentConfig{WorkspaceID: "ws-1", DealID: dealID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, getErr := agent.NewOrchestrator(db).GetAgentRun(context.Background(), "ws-1", run.ID)
	if getErr != nil {
		t.Fatalf("GetAgentRun: %v", getErr)
	}
	if stored.Status != agent.StatusSuccess {
		t.Fatalf("status=%s want=%s", stored.Status, agent.StatusSuccess)
	}
	if !contains(string(stored.Output), `"monitored"`) {
		t.Fatalf("output=%s expected monitored action", string(stored.Output))
	}
	if contains(string(stored.ToolCalls), `"create_task"`) {
		t.Fatalf("tool_calls=%s did not expect create_task", string(stored.ToolCalls))
	}
}

func TestDealRiskAgent_DailyLimitExceeded(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	insertDealRiskAgentDefinition(t, db, "ws-limit")

	for i := 0; i < 20; i++ {
		_, err := db.ExecContext(context.Background(), `
			INSERT INTO agent_run (id, workspace_id, agent_definition_id, trigger_type, status, started_at, created_at, updated_at)
			VALUES (?, ?, 'deal-risk-agent', 'manual', 'success', datetime('now'), datetime('now'), datetime('now'))
		`, mustID(t), "ws-limit")
		if err != nil {
			t.Fatalf("insert agent_run[%d]: %v", i, err)
		}
	}

	a := newTestDealRiskAgent(t, db, &mockKnowledgeSearch{results: emptyResults()}, &mockDealGetter{}, &mockAccountGetter{})
	_, err := a.Run(context.Background(), DealRiskAgentConfig{WorkspaceID: "ws-limit", DealID: "deal-x"})
	if err != ErrDealRiskDailyLimitExceeded {
		t.Fatalf("expected ErrDealRiskDailyLimitExceeded, got %v", err)
	}
}

func TestEvaluateDealRisk_StaleSignal(t *testing.T) {
	now := time.Now().UTC()
	signals := evaluateDealRisk(
		&crm.Deal{CreatedAt: now.Add(-35 * 24 * time.Hour), UpdatedAt: now.Add(-20 * 24 * time.Hour)},
		&crm.Account{Name: "Acme"},
		&knowledge.SearchResults{Items: []knowledge.SearchResult{{KnowledgeItemID: "ev-1", Score: 0.8}}},
	)
	if !signals.Stale {
		t.Fatal("expected stale signal")
	}
	if signals.RiskLevel != dealRiskLevelHigh {
		t.Fatalf("risk_level=%s want=%s", signals.RiskLevel, dealRiskLevelHigh)
	}
}

func TestEvaluateDealRisk_NoSignals(t *testing.T) {
	now := time.Now().UTC()
	signals := evaluateDealRisk(
		&crm.Deal{CreatedAt: now.Add(-5 * 24 * time.Hour), UpdatedAt: now.Add(-2 * 24 * time.Hour)},
		&crm.Account{Name: "Acme"},
		&knowledge.SearchResults{Items: []knowledge.SearchResult{
			{KnowledgeItemID: "ev-1", Score: 0.8},
			{KnowledgeItemID: "ev-2", Score: 0.7},
		}},
	)
	if signals.Stale || signals.StageStuck || signals.LowActivity {
		t.Fatalf("expected no signals, got %+v", signals)
	}
	if signals.RiskLevel != dealRiskLevelNone {
		t.Fatalf("risk_level=%s want=%s", signals.RiskLevel, dealRiskLevelNone)
	}
}

func mustID(t *testing.T) string {
	t.Helper()
	return time.Now().UTC().Format("20060102150405.000000000")
}
