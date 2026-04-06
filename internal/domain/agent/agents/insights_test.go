// Package agents provides concrete agent implementations.
// Task 4.5d — FR-231: Insights Agent tests
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
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

func insertInsightsAgentDefinition(t *testing.T, db *sql.DB, workspaceID string) {
	t.Helper()
	ensureAgentTestWorkspace(t, db, workspaceID)
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO agent_definition (id, workspace_id, name, agent_type, status)
		 VALUES ('insights-agent', ?, 'Insights Agent', 'insights', 'active')`, workspaceID)
	if err != nil {
		t.Fatalf("insert insights agent_definition: %v", err)
	}
}

func newTestInsightsAgent(t *testing.T, db *sql.DB, search KnowledgeSearchInterface) *InsightsAgent {
	t.Helper()
	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	if err := registry.Register(tool.BuiltinQueryMetrics, tool.NewQueryMetricsExecutor(db)); err != nil {
		t.Fatalf("register query_metrics: %v", err)
	}
	return NewInsightsAgent(orch, registry, search, db)
}

func TestInsightsAgent_AllowedTools(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()

	a := newTestInsightsAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	tools := a.AllowedTools()
	want := []string{"search_knowledge", "query_metrics"}
	if len(tools) != len(want) {
		t.Fatalf("expected %d tools, got %d", len(want), len(tools))
	}
	for i := range want {
		if tools[i] != want[i] {
			t.Fatalf("tool[%d]=%s want=%s", i, tools[i], want[i])
		}
	}
}

func TestInsightsAgent_Run_SalesFunnelQuery(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	workspaceID := "ws-insights-sales"
	insertInsightsAgentDefinition(t, db, workspaceID)
	ownerID := insertProspectingTestUser(t, db, workspaceID)

	account, err := crm.NewAccountService(db).Create(context.Background(), crm.CreateAccountInput{
		WorkspaceID: workspaceID,
		Name:        "Acme",
		OwnerID:     ownerID,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	pipeline, err := crm.NewPipelineService(db).Create(context.Background(), crm.CreatePipelineInput{
		WorkspaceID: workspaceID,
		Name:        "Ventas",
		EntityType:  "deal",
	})
	if err != nil {
		t.Fatalf("create pipeline: %v", err)
	}
	stage, err := crm.NewPipelineService(db).CreateStage(context.Background(), crm.CreatePipelineStageInput{
		PipelineID: pipeline.ID,
		Name:       "Discovery",
		Position:   0,
	})
	if err != nil {
		t.Fatalf("create stage: %v", err)
	}
	amount := 1200.0
	_, err = crm.NewDealService(db).Create(context.Background(), crm.CreateDealInput{
		WorkspaceID: workspaceID,
		AccountID:   account.ID,
		PipelineID:  pipeline.ID,
		StageID:     stage.ID,
		OwnerID:     ownerID,
		Title:       "Deal 1",
		Amount:      &amount,
	})
	if err != nil {
		t.Fatalf("create deal: %v", err)
	}

	a := newTestInsightsAgent(t, db, &mockKnowledgeSearch{results: &knowledge.SearchResults{Items: []knowledge.SearchResult{{KnowledgeItemID: "kb-1", Score: 0.9}}}})
	run, err := a.Run(context.Background(), InsightsAgentConfig{WorkspaceID: workspaceID, Query: "cuántos deals en pipeline"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), workspaceID, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	var output struct {
		Action     string           `json:"action"`
		Metrics    []map[string]any `json:"metrics"`
		Confidence string           `json:"confidence"`
	}
	if err := json.Unmarshal(stored.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Action == "abstain" {
		t.Fatalf("expected non-abstain action, got %q", output.Action)
	}
	if len(output.Metrics) == 0 {
		t.Fatal("expected metrics in output")
	}
	if output.Confidence == "" {
		t.Fatal("expected confidence in output")
	}
}

func TestInsightsAgent_Run_CaseBacklogQuery(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	workspaceID := "ws-insights-cases"
	insertInsightsAgentDefinition(t, db, workspaceID)
	ownerID := insertProspectingTestUser(t, db, workspaceID)

	caseTicket, err := crm.NewCaseService(db).Create(context.Background(), crm.CreateCaseInput{
		WorkspaceID: workspaceID,
		OwnerID:     ownerID,
		Subject:     "Caso pendiente",
		Status:      "open",
		Priority:    "high",
	})
	if err != nil {
		t.Fatalf("create case: %v", err)
	}
	_, err = db.ExecContext(context.Background(), `
		UPDATE case_ticket
		SET created_at = datetime('now', '-40 day'), updated_at = datetime('now', '-1 day')
		WHERE id = ?
	`, caseTicket.ID)
	if err != nil {
		t.Fatalf("age case ticket: %v", err)
	}

	a := newTestInsightsAgent(t, db, &mockKnowledgeSearch{results: &knowledge.SearchResults{Items: []knowledge.SearchResult{{KnowledgeItemID: "kb-2", Score: 0.7}}}})
	run, err := a.Run(context.Background(), InsightsAgentConfig{WorkspaceID: workspaceID, Query: "casos pendientes"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), workspaceID, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	var output struct {
		Metrics []map[string]any `json:"metrics"`
		Answer  string           `json:"answer"`
	}
	if err := json.Unmarshal(stored.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(output.Metrics) == 0 {
		t.Fatal("expected metrics for case backlog query")
	}
	if output.Answer == "" {
		t.Fatal("expected answer for case backlog query")
	}

	var toolCalls []struct {
		ToolName string `json:"tool_name"`
		Metric   string `json:"metric"`
	}
	if err := json.Unmarshal(stored.ToolCalls, &toolCalls); err != nil {
		t.Fatalf("unmarshal tool_calls: %v", err)
	}
	if len(toolCalls) == 0 {
		t.Fatal("expected at least one tool call")
	}
	if toolCalls[0].ToolName != "query_metrics" {
		t.Fatalf("expected first tool call query_metrics, got %q", toolCalls[0].ToolName)
	}
	if toolCalls[0].Metric != "case_backlog" {
		t.Fatalf("expected metric case_backlog, got %q", toolCalls[0].Metric)
	}
}

func TestInsightsAgent_Run_EmptyData_Abstains(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	workspaceID := "ws-insights-empty"
	insertInsightsAgentDefinition(t, db, workspaceID)
	_ = insertProspectingTestUser(t, db, workspaceID)

	a := newTestInsightsAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	run, err := a.Run(context.Background(), InsightsAgentConfig{WorkspaceID: workspaceID, Query: "deals en pipeline"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, err := agent.NewOrchestrator(db).GetAgentRun(context.Background(), workspaceID, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	var output struct {
		Action     string `json:"action"`
		Confidence string `json:"confidence"`
	}
	if err := json.Unmarshal(stored.Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output.Action != "abstain" {
		t.Fatalf("expected action=abstain, got %q", output.Action)
	}
	if output.Confidence != "low" {
		t.Fatalf("expected confidence=low, got %q", output.Confidence)
	}
}

func TestParseQueryIntent_BacklogPriorityOverCaseVolume(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{name: "casos pendientes", query: "casos pendientes", want: "case_backlog"},
		{name: "casos abiertos", query: "casos abiertos", want: "case_backlog"},
		{name: "backlog de casos", query: "backlog de casos", want: "case_backlog"},
		{name: "volumen de casos", query: "volumen de casos", want: "case_volume"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseQueryIntent(tc.query)
			if got != tc.want {
				t.Fatalf("parseQueryIntent(%q)=%q want=%q", tc.query, got, tc.want)
			}
		})
	}
}

func TestParseQueryIntent_AdditionalBranches(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{query: "aging de deals", want: "deal_aging"},
		{query: "tiempo de resolución mttr", want: "mttr"},
		{query: "ventas del funnel", want: "sales_funnel"},
		{query: "consulta desconocida", want: "sales_funnel"},
	}

	for _, tc := range tests {
		if got := parseQueryIntent(tc.query); got != tc.want {
			t.Fatalf("parseQueryIntent(%q)=%q want=%q", tc.query, got, tc.want)
		}
	}
}

func TestInsightsAgent_QueryMetricsFailures(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()

	orch := agent.NewOrchestrator(db)
	registry := tool.NewToolRegistry(db)
	a := NewInsightsAgent(orch, registry, &mockKnowledgeSearch{results: emptyResults()}, db)

	if _, err := a.queryMetrics(context.Background(), "ws-1", "sales_funnel", nil, nil); err != ErrInsightsQueryMetricsFailed {
		t.Fatalf("missing executor error = %v want %v", err, ErrInsightsQueryMetricsFailed)
	}

	registry = tool.NewToolRegistry(db)
	if err := registry.Register(tool.BuiltinQueryMetrics, &mockToolExecutor{out: json.RawMessage(`{invalid`)}); err != nil {
		t.Fatalf("register query_metrics: %v", err)
	}
	a = NewInsightsAgent(orch, registry, &mockKnowledgeSearch{results: emptyResults()}, db)
	if _, err := a.queryMetrics(context.Background(), "ws-1", "sales_funnel", nil, nil); err != ErrInsightsQueryMetricsFailed {
		t.Fatalf("invalid json shape error = %v want %v", err, ErrInsightsQueryMetricsFailed)
	}
}

func TestInsightsAgent_CheckDailyLimits(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()

	a := newTestInsightsAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	if err := a.checkDailyLimits(context.Background(), "ws-insights-limits"); err != nil {
		t.Fatalf("empty limits check error = %v", err)
	}

	workspaceID := "ws-insights-limits"
	insertInsightsAgentDefinition(t, db, workspaceID)
	now := time.Now().UTC().Format(time.RFC3339)
	for i := 0; i < 100; i++ {
		_, err := db.ExecContext(context.Background(), `
			INSERT INTO agent_run (id, workspace_id, agent_definition_id, trigger_type, status, started_at, created_at)
			VALUES (?, ?, 'insights-agent', 'manual', 'success', ?, ?)
		`, uuid.NewV7().String(), workspaceID, now, now)
		if err != nil {
			t.Fatalf("insert agent_run #%d: %v", i, err)
		}
	}
	if err := a.checkDailyLimits(context.Background(), workspaceID); err != ErrInsightsDailyLimitExceeded {
		t.Fatalf("count limit error = %v want %v", err, ErrInsightsDailyLimitExceeded)
	}

	if _, err := db.ExecContext(context.Background(), `DELETE FROM agent_run WHERE workspace_id = ?`, workspaceID); err != nil {
		t.Fatalf("clear agent_run rows: %v", err)
	}
	workspaceCost := workspaceID
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO agent_run (id, workspace_id, agent_definition_id, trigger_type, status, total_cost, started_at, created_at)
		VALUES (?, ?, 'insights-agent', 'manual', 'success', 20.0, ?, ?)
	`, uuid.NewV7().String(), workspaceCost, now, now)
	if err != nil {
		t.Fatalf("insert cost run: %v", err)
	}
	if err := a.checkDailyLimits(context.Background(), workspaceCost); err != ErrInsightsDailyLimitExceeded {
		t.Fatalf("cost limit error = %v want %v", err, ErrInsightsDailyLimitExceeded)
	}

	nilDBAgent := NewInsightsAgent(agent.NewOrchestrator(db), tool.NewToolRegistry(db), &mockKnowledgeSearch{results: emptyResults()}, nil)
	if err := nilDBAgent.checkDailyLimits(context.Background(), workspaceID); err != nil {
		t.Fatalf("nil db checkDailyLimits() error = %v", err)
	}
}

func TestInsightsAgent_Objective_ReturnsJSON(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	a := newTestInsightsAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})
	obj := a.Objective()
	if len(obj) == 0 {
		t.Fatal("Objective() returned empty")
	}
	var m map[string]any
	if err := json.Unmarshal(obj, &m); err != nil {
		t.Fatalf("Objective() not valid JSON: %v", err)
	}
}

func TestInsightsError_Error_ReturnsMessage(t *testing.T) {
	err := ErrInsightsQueryRequired
	if err.Error() != "query is required" {
		t.Fatalf("unexpected message: %q", err.Error())
	}
}

func TestInsightsAgent_MarkRunFailed(t *testing.T) {
	db := setupProspectingTestDB(t)
	defer db.Close()
	workspaceID := "ws-insights-markfailed"
	insertInsightsAgentDefinition(t, db, workspaceID)
	_ = insertProspectingTestUser(t, db, workspaceID)
	a := newTestInsightsAgent(t, db, &mockKnowledgeSearch{results: emptyResults()})

	orch := agent.NewOrchestrator(db)
	run, err := orch.TriggerAgent(context.Background(), agent.TriggerAgentInput{
		AgentID:     "insights-agent",
		WorkspaceID: workspaceID,
		TriggerType: agent.TriggerTypeManual,
	})
	if err != nil {
		t.Fatalf("TriggerAgent: %v", err)
	}
	if err := a.markRunFailed(context.Background(), run); err != nil {
		t.Fatalf("markRunFailed: %v", err)
	}
}
