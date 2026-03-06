// Package agents provides concrete agent implementations.
// Task 4.5d — FR-231: Insights Agent
package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

// Task 4.5d — FR-231: Insights Agent config.
type InsightsAgentConfig struct {
	WorkspaceID       string     `json:"workspace_id"`
	Query             string     `json:"query"`
	DateFrom          *time.Time `json:"date_from,omitempty"`
	DateTo            *time.Time `json:"date_to,omitempty"`
	Language          string     `json:"language,omitempty"`
	TriggeredByUserID *string    `json:"-"`
}

const insightsBaseRunCostEuros = 0.01 // Task 4.5d — sin LLM en MVP.
const insightsDefaultLanguage = "es"

// InsightsAgent implements FR-231 insights flow.
type InsightsAgent struct {
	orchestrator    *agent.Orchestrator
	toolRegistry    *tool.ToolRegistry
	knowledgeSearch KnowledgeSearchInterface
	db              *sql.DB
}

// NewInsightsAgent creates an insights agent.
func NewInsightsAgent(
	orchestrator *agent.Orchestrator,
	toolRegistry *tool.ToolRegistry,
	knowledgeSearch KnowledgeSearchInterface,
	db *sql.DB,
) *InsightsAgent {
	return &InsightsAgent{
		orchestrator:    orchestrator,
		toolRegistry:    toolRegistry,
		knowledgeSearch: knowledgeSearch,
		db:              db,
	}
}

// AllowedTools returns tools allowed to the insights agent.
func (a *InsightsAgent) AllowedTools() []string {
	return []string{"search_knowledge", "query_metrics"}
}

// Objective returns objective payload used by the runtime.
func (a *InsightsAgent) Objective() json.RawMessage {
	objective := map[string]any{
		"role": "business_analyst",
		"goal": "answer_crm_metrics_questions",
	}
	obj, _ := json.Marshal(objective)
	return obj
}

// Run executes insights flow and persists agent_run updates.
func (a *InsightsAgent) Run(ctx context.Context, config InsightsAgentConfig) (*agent.Run, error) {
	normalized, err := a.normalizeConfig(ctx, config) // Task 4.5d — normalize before TriggerAgent.
	if err != nil {
		return nil, err
	}

	triggerContext, _ := json.Marshal(map[string]any{
		"query":        normalized.Query,
		"language":     normalized.Language,
		"agent_type":   "insights",
		"capabilities": a.AllowedTools(),
	})
	inputs, _ := json.Marshal(normalized)

	run, err := a.orchestrator.TriggerAgent(ctx, agent.TriggerAgentInput{
		AgentID:        "insights-agent",
		WorkspaceID:    normalized.WorkspaceID,
		TriggeredBy:    normalized.TriggeredByUserID,
		TriggerType:    agent.TriggerTypeManual,
		TriggerContext: triggerContext,
		Inputs:         inputs,
	})
	if err != nil {
		return nil, err
	}

	toolCtx := context.WithValue(ctx, ctxkeys.WorkspaceID, normalized.WorkspaceID) // Task 4.5d — toolCtx workspace propagation.
	if normalized.TriggeredByUserID != nil {
		toolCtx = context.WithValue(toolCtx, ctxkeys.UserID, *normalized.TriggeredByUserID) // Task 4.5d — toolCtx user propagation.
	}

	result, err := a.executeInsightsFlow(toolCtx, normalized)
	if err != nil {
		if updateErr := a.markRunFailed(ctx, run); updateErr != nil {
			return run, updateErr
		}
		return run, err
	}

	_, err = a.orchestrator.UpdateAgentRun(ctx, run.WorkspaceID, run.ID, agent.RunUpdates{
		Status:      result.Status,
		Output:      result.Output,
		ToolCalls:   result.ToolCalls,
		TotalCost:   result.TotalCost,
		TotalTokens: result.TotalTokens,
		LatencyMs:   result.LatencyMs,
		Completed:   true,
	})
	if err != nil {
		return run, err
	}

	return run, nil
}

func (a *InsightsAgent) normalizeConfig(ctx context.Context, config InsightsAgentConfig) (InsightsAgentConfig, error) {
	if strings.TrimSpace(config.Query) == "" {
		return InsightsAgentConfig{}, ErrInsightsQueryRequired
	}
	if config.Language == "" {
		config.Language = insightsDefaultLanguage
	}
	if err := a.checkDailyLimits(ctx, config.WorkspaceID); err != nil {
		return InsightsAgentConfig{}, err
	}
	return config, nil
}

func (a *InsightsAgent) markRunFailed(ctx context.Context, run *agent.Run) error {
	_, err := a.orchestrator.UpdateAgentRunStatus(ctx, run.WorkspaceID, run.ID, agent.StatusFailed)
	return err
}

// InsightsResult holds runtime update payload.
type InsightsResult struct {
	Status      string
	Output      json.RawMessage
	ToolCalls   json.RawMessage
	TotalTokens *int64
	TotalCost   *float64
	LatencyMs   *int64
}

func (a *InsightsAgent) executeInsightsFlow(
	toolCtx context.Context,
	config InsightsAgentConfig,
) (*InsightsResult, error) {
	start := time.Now()
	tokens := int64(0)
	cost := insightsBaseRunCostEuros

	metric := parseQueryIntent(config.Query)
	metricsData, err := a.queryMetrics(toolCtx, config.WorkspaceID, metric, config.DateFrom, config.DateTo)
	if err != nil {
		return nil, err
	}
	searchResults := a.searchKnowledge(toolCtx, config.WorkspaceID, config.Query)
	topScore := topKnowledgeScore(searchResults)

	output := a.resolveInsightsOutput(metric, config.Query, metricsData, searchResults, topScore)
	calls := []map[string]any{
		{"tool_name": "query_metrics", "metric": metric, "executed_at": time.Now().UTC().Format(time.RFC3339)},
		{"tool_name": "search_knowledge", "limit": 3, "executed_at": time.Now().UTC().Format(time.RFC3339)},
	}
	outputJSON, _ := json.Marshal(output)
	toolCalls, _ := json.Marshal(calls)
	latency := time.Since(start).Milliseconds()

	return &InsightsResult{
		Status:      agent.StatusSuccess,
		Output:      outputJSON,
		ToolCalls:   toolCalls,
		TotalTokens: &tokens,
		TotalCost:   &cost,
		LatencyMs:   &latency,
	}, nil
}

func parseQueryIntent(query string) string {
	q := strings.ToLower(query)
	switch {
	case strings.Contains(q, "aging"), strings.Contains(q, "días en stage"):
		return "deal_aging"
	case strings.Contains(q, "backlog"), strings.Contains(q, "pendiente"), strings.Contains(q, "abierto"):
		return "case_backlog"
	case strings.Contains(q, "caso"), strings.Contains(q, "ticket"), strings.Contains(q, "volumen"):
		return "case_volume"
	case strings.Contains(q, "mttr"), strings.Contains(q, "resolución"), strings.Contains(q, "tiempo"):
		return "mttr"
	case strings.Contains(q, "deal"), strings.Contains(q, "venta"), strings.Contains(q, "funnel"):
		return "sales_funnel"
	default:
		return "sales_funnel"
	}
}

func (a *InsightsAgent) queryMetrics(
	ctx context.Context,
	workspaceID, metric string,
	dateFrom, dateTo *time.Time,
) ([]map[string]any, error) {
	_, err := a.toolRegistry.Get(tool.BuiltinQueryMetrics) // Task 4.5d — use ToolRegistry, no direct SQL metric queries.
	if err != nil {
		return nil, ErrInsightsQueryMetricsFailed
	}
	payload := map[string]any{"metric": metric, "workspace_id": workspaceID}
	if dateFrom != nil {
		payload["from"] = dateFrom.UTC().Format(time.RFC3339)
	}
	if dateTo != nil {
		payload["to"] = dateTo.UTC().Format(time.RFC3339)
	}
	raw, err := a.toolRegistry.Execute(ctx, workspaceID, tool.BuiltinQueryMetrics, mustJSON(payload))
	if err != nil {
		return nil, ErrInsightsQueryMetricsFailed
	}
	var parsed struct {
		Data []map[string]any `json:"data"`
	}
	if unmarshalErr := json.Unmarshal(raw, &parsed); unmarshalErr != nil {
		return nil, ErrInsightsQueryMetricsFailed
	}
	return parsed.Data, nil
}

func (a *InsightsAgent) searchKnowledge(ctx context.Context, workspaceID, query string) *knowledge.SearchResults {
	results, err := a.knowledgeSearch.HybridSearch(ctx, knowledge.SearchInput{ // Task 4.5d — search_knowledge via SearchService.
		WorkspaceID: workspaceID,
		Query:       query,
		Limit:       3,
	})
	if err != nil || results == nil {
		return &knowledge.SearchResults{Items: []knowledge.SearchResult{}}
	}
	return results
}

func (a *InsightsAgent) resolveInsightsOutput(
	metric, query string,
	metricsData []map[string]any,
	searchResults *knowledge.SearchResults,
	topScore float64,
) map[string]any {
	if len(metricsData) == 0 && (len(searchResults.Items) == 0 || topScore < 0.4) {
		return map[string]any{ // Task 4.5d — FR-210 abstención obligatoria.
			"action":     "abstain",
			"reason":     "insufficient_data",
			"confidence": "low",
		}
	}
	confidence := confidenceFromScore(topScore)
	return map[string]any{
		"action":       "answer",
		"answer":       formatInsightAnswer(metric, query, metricsData),
		"metrics":      metricsData,
		"confidence":   confidence,
		"evidence_ids": extractEvidenceIDs(searchResults),
	}
}

func formatInsightAnswer(metric, query string, metricsData []map[string]any) string {
	if len(metricsData) == 0 {
		return fmt.Sprintf("No encontré métricas para responder %q (metric=%s).", query, metric)
	}
	rows := make([]string, 0, len(metricsData))
	for i, m := range metricsData {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%v", k, m[k]))
		}
		rows = append(rows, fmt.Sprintf("fila_%d{%s}", i+1, strings.Join(parts, ", ")))
	}
	return fmt.Sprintf("Consulta: %s. Métrica %s con %d resultados numéricos: %s.", query, metric, len(metricsData), strings.Join(rows, " | "))
}

func topKnowledgeScore(results *knowledge.SearchResults) float64 {
	if results == nil || len(results.Items) == 0 {
		return 0
	}
	return results.Items[0].Score
}

func extractEvidenceIDs(results *knowledge.SearchResults) []string {
	if results == nil || len(results.Items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(results.Items))
	for _, item := range results.Items {
		if item.KnowledgeItemID != "" {
			out = append(out, item.KnowledgeItemID)
		}
	}
	return out
}

func confidenceFromScore(score float64) string {
	if score > 0.8 {
		return "high"
	}
	if score > 0.5 {
		return "medium"
	}
	return "low"
}

func (a *InsightsAgent) checkDailyLimits(ctx context.Context, workspaceID string) error {
	if a.db == nil {
		return nil
	}
	const maxDailyQueries = 100
	const maxDailyCost = 20.0

	var runsToday int
	if err := a.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM agent_run
		WHERE workspace_id = ?
		  AND agent_definition_id = 'insights-agent'
		  AND date(created_at) = date('now')
	`, workspaceID).Scan(&runsToday); err != nil {
		return err
	}
	if runsToday >= maxDailyQueries {
		return ErrInsightsDailyLimitExceeded
	}

	var dailyCost float64
	if err := a.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_cost), 0)
		FROM agent_run
		WHERE workspace_id = ?
		  AND agent_definition_id = 'insights-agent'
		  AND date(created_at) = date('now')
	`, workspaceID).Scan(&dailyCost); err != nil {
		return err
	}
	if dailyCost >= maxDailyCost {
		return ErrInsightsDailyLimitExceeded
	}
	return nil
}

var (
	ErrInsightsQueryRequired      = &InsightsError{message: "query is required"}
	ErrInsightsDailyLimitExceeded = &InsightsError{message: "daily query limit exceeded (max 100/day)"}
	ErrInsightsQueryMetricsFailed = &InsightsError{message: "failed to query metrics"}
)

// InsightsError is the typed error for the insights agent.
type InsightsError struct{ message string }

func (e *InsightsError) Error() string { return e.message }
