package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

const (
	dealRiskAgentID       = "deal-risk-agent"
	dealRiskBaseRunCost   = 0.02
	dealRiskDefaultLang   = "es"
	dealRiskLevelNone     = "none"
	dealRiskLevelLow      = "low"
	dealRiskLevelMedium   = "medium"
	dealRiskLevelHigh     = "high"
	dealRiskActionMonitor = "monitored"
	dealRiskActionTask    = "create_task"
	dealRiskTaskTitle     = "Review at-risk deal"
	dealRiskEvidenceLimit = 5
	dealRiskMaxDailyRuns  = 20
	dealRiskMaxDailyCost  = 5.0
)

type DealRiskAgentConfig struct {
	WorkspaceID       string  `json:"workspace_id"`
	DealID            string  `json:"deal_id"`
	Language          string  `json:"language,omitempty"`
	TriggeredByUserID *string `json:"-"`
}

type DealGetter interface {
	Get(ctx context.Context, workspaceID, dealID string) (*crm.Deal, error)
}

type DealRiskAgent struct {
	orchestrator    *agent.Orchestrator
	toolRegistry    *tool.ToolRegistry
	knowledgeSearch KnowledgeSearchInterface
	llmProvider     llm.LLMProvider
	dealService     DealGetter
	accountService  AccountGetter
	db              *sql.DB
}

type DealRiskSignals struct {
	Stale       bool   `json:"stale"`
	StageStuck  bool   `json:"stage_stuck"`
	LowActivity bool   `json:"low_activity"`
	RiskLevel   string `json:"risk_level"`
}

type DealRiskResult struct {
	Status         string
	Output         json.RawMessage
	ToolCalls      json.RawMessage
	ReasoningTrace json.RawMessage
	TotalTokens    *int64
	TotalCost      *float64
	LatencyMs      *int64
}

func NewDealRiskAgent(
	orchestrator *agent.Orchestrator,
	toolRegistry *tool.ToolRegistry,
	knowledgeSearch KnowledgeSearchInterface,
	llmProvider llm.LLMProvider,
	dealService DealGetter,
	accountService AccountGetter,
	db *sql.DB,
) *DealRiskAgent {
	return &DealRiskAgent{
		orchestrator:    orchestrator,
		toolRegistry:    toolRegistry,
		knowledgeSearch: knowledgeSearch,
		llmProvider:     llmProvider,
		dealService:     dealService,
		accountService:  accountService,
		db:              db,
	}
}

func (a *DealRiskAgent) AllowedTools() []string {
	return []string{"search_knowledge", "create_task", "get_deal", "get_account"}
}

func (a *DealRiskAgent) Objective() json.RawMessage {
	objective := map[string]any{
		"role": "deal_analyst",
		"goal": "assess_deal_risk",
	}
	obj, _ := json.Marshal(objective)
	return obj
}

func (a *DealRiskAgent) Run(ctx context.Context, config DealRiskAgentConfig) (*agent.Run, error) {
	normalized, err := a.normalizeConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	triggerContext, _ := json.Marshal(map[string]any{
		"deal_id":      normalized.DealID,
		"language":     normalized.Language,
		"agent_type":   "deal-risk",
		"capabilities": a.AllowedTools(),
	})
	inputs, _ := json.Marshal(normalized)

	run, err := a.orchestrator.TriggerAgent(ctx, agent.TriggerAgentInput{
		AgentID:        dealRiskAgentID,
		WorkspaceID:    normalized.WorkspaceID,
		TriggeredBy:    normalized.TriggeredByUserID,
		TriggerType:    agent.TriggerTypeManual,
		TriggerContext: triggerContext,
		Inputs:         inputs,
	})
	if err != nil {
		return nil, err
	}

	result, err := a.executeDealRiskFlow(ctx, normalized)
	if err != nil {
		if updateErr := a.markRunFailed(ctx, run); updateErr != nil {
			return run, updateErr
		}
		return run, err
	}

	_, err = a.orchestrator.UpdateAgentRun(ctx, run.WorkspaceID, run.ID, agent.RunUpdates{
		Status:         result.Status,
		Output:         result.Output,
		ToolCalls:      result.ToolCalls,
		ReasoningTrace: result.ReasoningTrace,
		TotalTokens:    result.TotalTokens,
		TotalCost:      result.TotalCost,
		LatencyMs:      result.LatencyMs,
		Completed:      true,
	})
	if err != nil {
		return run, err
	}

	return run, nil
}

func (a *DealRiskAgent) normalizeConfig(ctx context.Context, config DealRiskAgentConfig) (DealRiskAgentConfig, error) {
	if config.DealID == "" {
		return DealRiskAgentConfig{}, ErrDealIDRequired
	}
	if config.Language == "" {
		config.Language = dealRiskDefaultLang
	}
	if err := a.checkDailyLimits(ctx, config.WorkspaceID); err != nil {
		return DealRiskAgentConfig{}, err
	}
	return config, nil
}

func (a *DealRiskAgent) markRunFailed(ctx context.Context, run *agent.Run) error {
	_, err := a.orchestrator.UpdateAgentRunStatus(ctx, run.WorkspaceID, run.ID, agent.StatusFailed)
	return err
}

func (a *DealRiskAgent) executeDealRiskFlow(ctx context.Context, config DealRiskAgentConfig) (*DealRiskResult, error) {
	start := time.Now()
	totalTokens := int64(0)
	totalCost := dealRiskBaseRunCost

	toolCtx := context.WithValue(ctx, ctxkeys.WorkspaceID, config.WorkspaceID)
	deal, err := a.fetchDeal(toolCtx, config)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDealNotFound
	}
	if err != nil {
		return nil, err
	}

	toolCtx = context.WithValue(toolCtx, ctxkeys.UserID, deal.OwnerID)

	account, err := a.fetchAccount(toolCtx, deal.AccountID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("deal title=%s account=%s stage=%s status=%s", deal.Title, account.Name, deal.StageID, deal.Status)
	evidence := a.searchSignals(toolCtx, config.WorkspaceID, query)
	signals := evaluateDealRisk(deal, account, evidence)
	status, output, toolCalls, err := a.resolveDealRiskOutcome(toolCtx, config, deal, account, evidence, signals, query)
	if err != nil {
		return nil, err
	}

	outputJSON, _ := json.Marshal(output)
	toolCallsJSON, _ := json.Marshal(toolCalls)
	reasoningTrace, _ := json.Marshal([]map[string]any{{
		"step":       "evaluate_deal_risk",
		"risk_level": signals.RiskLevel,
		"signals":    signals,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}})
	latency := time.Since(start).Milliseconds()

	return &DealRiskResult{
		Status:         status,
		Output:         outputJSON,
		ToolCalls:      toolCallsJSON,
		ReasoningTrace: reasoningTrace,
		TotalTokens:    &totalTokens,
		TotalCost:      &totalCost,
		LatencyMs:      &latency,
	}, nil
}

func (a *DealRiskAgent) resolveDealRiskOutcome(
	ctx context.Context,
	config DealRiskAgentConfig,
	deal *crm.Deal,
	account *crm.Account,
	evidence *knowledge.SearchResults,
	signals DealRiskSignals,
	query string,
) (string, map[string]any, []map[string]any, error) {
	toolCalls := baseDealRiskToolCalls(config.DealID, deal.AccountID, query)
	output := map[string]any{
		"deal_id":          deal.ID,
		"account_id":       account.ID,
		"signals":          signals,
		"evidence_summary": evidenceSnippets(evidence),
		"language":         config.Language,
	}

	if signals.RiskLevel == dealRiskLevelNone {
		output["action"] = dealRiskActionMonitor
		return agent.StatusSuccess, output, toolCalls, nil
	}

	taskID, err := a.createMitigationTask(ctx, deal)
	if err != nil {
		return "", nil, nil, err
	}
	output["action"] = dealRiskActionTask
	output["task_id"] = taskID
	toolCalls = append(toolCalls, map[string]any{
		"tool_name":   "create_task",
		"params":      map[string]any{"entity_type": "deal", "entity_id": deal.ID},
		"executed_at": time.Now().UTC().Format(time.RFC3339),
	})
	return agent.StatusEscalated, output, toolCalls, nil
}

func evaluateDealRisk(deal *crm.Deal, _ *crm.Account, evidence *knowledge.SearchResults) DealRiskSignals {
	now := time.Now().UTC()
	signals := DealRiskSignals{
		Stale:       now.Sub(deal.UpdatedAt) > 14*24*time.Hour,
		StageStuck:  now.Sub(deal.UpdatedAt) > 30*24*time.Hour,
		LowActivity: evidenceItemCount(evidence) < 2,
		RiskLevel:   dealRiskLevelNone,
	}
	switch {
	case signals.Stale:
		signals.RiskLevel = dealRiskLevelHigh
	case signals.StageStuck:
		signals.RiskLevel = dealRiskLevelMedium
	case signals.LowActivity:
		signals.RiskLevel = dealRiskLevelLow
	}
	return signals
}

func (a *DealRiskAgent) fetchDeal(ctx context.Context, config DealRiskAgentConfig) (*crm.Deal, error) {
	raw, err := a.toolRegistry.Execute(ctx, config.WorkspaceID, tool.BuiltinGetDeal, mustJSON(map[string]any{"deal_id": config.DealID}))
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Deal *crm.Deal `json:"deal"`
	}
	unmarshalErr := json.Unmarshal(raw, &parsed)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}
	if parsed.Deal == nil {
		return nil, ErrDealNotFound
	}
	return parsed.Deal, nil
}

func (a *DealRiskAgent) fetchAccount(ctx context.Context, accountID string) (*crm.Account, error) {
	raw, err := a.toolRegistry.Execute(ctx, workspaceFromCtx(ctx), tool.BuiltinGetAccount, mustJSON(map[string]any{"account_id": accountID}))
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Account *crm.Account `json:"account"`
	}
	unmarshalErr := json.Unmarshal(raw, &parsed)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}
	if parsed.Account == nil {
		return nil, ErrAccountNotFound
	}
	return parsed.Account, nil
}

func (a *DealRiskAgent) createMitigationTask(ctx context.Context, deal *crm.Deal) (string, error) {
	raw, err := a.toolRegistry.Execute(ctx, workspaceFromCtx(ctx), tool.BuiltinCreateTask, mustJSON(map[string]any{
		"owner_id":    deal.OwnerID,
		"title":       dealRiskTaskTitle,
		"entity_type": "deal",
		"entity_id":   deal.ID,
	}))
	if err != nil {
		return "", err
	}
	var parsed struct {
		TaskID string `json:"task_id"`
	}
	unmarshalErr := json.Unmarshal(raw, &parsed)
	if unmarshalErr != nil {
		return "", unmarshalErr
	}
	if parsed.TaskID == "" {
		return "", ErrTaskCreationFailed
	}
	return parsed.TaskID, nil
}

func (a *DealRiskAgent) searchSignals(ctx context.Context, workspaceID, query string) *knowledge.SearchResults {
	evidence, err := a.knowledgeSearch.HybridSearch(ctx, knowledge.SearchInput{
		WorkspaceID: workspaceID,
		Query:       query,
		Limit:       dealRiskEvidenceLimit,
	})
	if err != nil || evidence == nil {
		return &knowledge.SearchResults{Items: []knowledge.SearchResult{}}
	}
	return evidence
}

func baseDealRiskToolCalls(dealID, accountID, query string) []map[string]any {
	return []map[string]any{
		{
			"tool_name":   "get_deal",
			"params":      map[string]any{"deal_id": dealID},
			"executed_at": time.Now().UTC().Format(time.RFC3339),
		},
		{
			"tool_name":   "get_account",
			"params":      map[string]any{"account_id": accountID},
			"executed_at": time.Now().UTC().Format(time.RFC3339),
		},
		{
			"tool_name": "search_knowledge",
			"params": map[string]any{
				"query": query,
				"limit": dealRiskEvidenceLimit,
			},
			"executed_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
}

func evidenceItemCount(results *knowledge.SearchResults) int {
	if results == nil {
		return 0
	}
	return len(results.Items)
}

func evidenceSnippets(results *knowledge.SearchResults) []string {
	if results == nil {
		return []string{}
	}
	out := make([]string, 0, len(results.Items))
	for _, item := range results.Items {
		if item.Snippet != "" {
			out = append(out, item.Snippet)
		}
	}
	return out
}

func (a *DealRiskAgent) checkDailyLimits(ctx context.Context, workspaceID string) error {
	return checkDailyRunAndCostLimits(
		ctx,
		a.db,
		workspaceID,
		dealRiskAgentID,
		dealRiskMaxDailyRuns,
		dealRiskMaxDailyCost,
		ErrDealRiskDailyLimitExceeded,
		ErrDealRiskDailyCostLimitExceeded,
	)
}

var (
	ErrDealIDRequired                 = &DealRiskError{message: "deal_id is required"}
	ErrDealNotFound                   = &DealRiskError{message: "deal not found"}
	ErrDealRiskDailyLimitExceeded     = &DealRiskError{message: "deal risk daily run limit exceeded"}
	ErrDealRiskDailyCostLimitExceeded = &DealRiskError{message: "deal risk daily cost limit exceeded"}
)

type DealRiskError struct {
	message string
}

func (e *DealRiskError) Error() string { return e.message }
