// Package agents provides concrete agent implementations.
// Task 4.5b — FR-231: Prospecting Agent
package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

// Task 4.5b — prospecting configuration.
type ProspectingAgentConfig struct {
	WorkspaceID       string  `json:"workspace_id"`
	LeadID            string  `json:"lead_id"`
	Language          string  `json:"language,omitempty"`
	TriggeredByUserID *string `json:"-"`
}

const baseRunCostEuros = 0.05

// LeadGetter abstracts lead retrieval for testability.
type LeadGetter interface {
	Get(ctx context.Context, workspaceID, leadID string) (*crm.Lead, error)
}

// AccountGetter abstracts account retrieval for testability.
type AccountGetter interface {
	Get(ctx context.Context, workspaceID, accountID string) (*crm.Account, error)
}

// ProspectingAgent implements FR-231 prospecting flow.
type ProspectingAgent struct {
	orchestrator    *agent.Orchestrator
	toolRegistry    *tool.ToolRegistry
	knowledgeSearch KnowledgeSearchInterface
	llmProvider     llm.LLMProvider
	leadService     LeadGetter
	accountService  AccountGetter
	db              *sql.DB
}

// NewProspectingAgent creates a prospecting agent.
func NewProspectingAgent(
	orchestrator *agent.Orchestrator,
	toolRegistry *tool.ToolRegistry,
	knowledgeSearch KnowledgeSearchInterface,
	llmProvider llm.LLMProvider,
	leadService LeadGetter,
	accountService AccountGetter,
	db *sql.DB,
) *ProspectingAgent {
	return &ProspectingAgent{
		orchestrator:    orchestrator,
		toolRegistry:    toolRegistry,
		knowledgeSearch: knowledgeSearch,
		llmProvider:     llmProvider,
		leadService:     leadService,
		accountService:  accountService,
		db:              db,
	}
}

// AllowedTools returns tools allowed to the prospecting agent.
func (a *ProspectingAgent) AllowedTools() []string {
	return []string{"search_knowledge", "create_task", "get_lead", "get_account"}
}

// Objective returns objective payload used by the runtime.
func (a *ProspectingAgent) Objective() json.RawMessage {
	objective := map[string]any{
		"role": "sales_dev",
		"goal": "draft_outreach",
		"instructions": []string{
			"1. Retrieve lead and account context",
			"2. Search prior signals in knowledge",
			"3. If confidence > 0.6 draft personalized outreach and create follow-up task",
			"4. If confidence <= 0.6 skip with reason insufficient_signals",
		},
		"response_format": map[string]string{
			"action":     "draft_outreach|skip",
			"details":    "action details",
			"lead_id":    "lead identifier",
			"confidence": "0-1",
		},
	}
	obj, _ := json.Marshal(objective)
	return obj
}

// Run executes prospecting flow and persists agent_run updates.
func (a *ProspectingAgent) Run(ctx context.Context, config ProspectingAgentConfig) (*agent.Run, error) {
	normalized, err := a.normalizeConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	triggerContext, _ := json.Marshal(map[string]any{
		"lead_id":      normalized.LeadID,
		"language":     normalized.Language,
		"agent_type":   "prospecting",
		"capabilities": a.AllowedTools(),
	})
	inputs, _ := json.Marshal(normalized)

	run, err := a.orchestrator.TriggerAgent(ctx, agent.TriggerAgentInput{
		AgentID:        "prospecting-agent",
		WorkspaceID:    normalized.WorkspaceID,
		TriggeredBy:    normalized.TriggeredByUserID,
		TriggerType:    agent.TriggerTypeManual,
		TriggerContext: triggerContext,
		Inputs:         inputs,
	})
	if err != nil {
		return nil, err
	}

	result, err := a.executeProspectingFlow(ctx, normalized)
	if err != nil {
		updateErr := a.markRunFailed(ctx, run)
		if updateErr != nil {
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

func (a *ProspectingAgent) normalizeConfig(ctx context.Context, config ProspectingAgentConfig) (ProspectingAgentConfig, error) {
	if config.LeadID == "" {
		return ProspectingAgentConfig{}, ErrLeadIDRequired
	}
	if config.Language == "" {
		config.Language = "es"
	}
	if err := a.checkDailyLimits(ctx, config.WorkspaceID); err != nil {
		return ProspectingAgentConfig{}, err
	}
	return config, nil
}

func (a *ProspectingAgent) markRunFailed(ctx context.Context, run *agent.Run) error {
	_, err := a.orchestrator.UpdateAgentRunStatus(ctx, run.WorkspaceID, run.ID, agent.StatusFailed)
	return err
}

// ProspectingResult holds runtime update payload.
type ProspectingResult struct {
	Status         string
	Output         json.RawMessage
	ToolCalls      json.RawMessage
	ReasoningTrace json.RawMessage
	TotalTokens    *int64
	TotalCost      *float64
	LatencyMs      *int64
}

func (a *ProspectingAgent) executeProspectingFlow(ctx context.Context, config ProspectingAgentConfig) (*ProspectingResult, error) {
	startTime := time.Now()
	totalTokens := int64(0)
	totalCost := baseRunCostEuros // Task 4.5b — baseline non-LLM run cost tracking.

	toolCtx := context.WithValue(ctx, ctxkeys.WorkspaceID, config.WorkspaceID)

	lead, err := a.fetchLead(toolCtx, config)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrLeadNotFound
	}
	if err != nil {
		return nil, err
	}

	toolCtx = context.WithValue(toolCtx, ctxkeys.UserID, lead.OwnerID)

	accountName := a.resolveAccountName(toolCtx, lead)

	query := fmt.Sprintf("lead source=%s account=%s status=%s", safePtr(lead.Source), accountName, lead.Status)
	evidence := a.searchSignals(toolCtx, config.WorkspaceID, query)

	confidence := 0.0
	if len(evidence.Items) > 0 {
		confidence = evidence.Items[0].Score
	}

	toolCalls := baseProspectingToolCalls(config.LeadID, lead.AccountID, query)
	out, nextToolCalls, tokens, cost, flowErr := a.resolveAction(ctx, toolCtx, config, lead, accountName, confidence)
	if flowErr != nil {
		return nil, flowErr
	}
	toolCalls = append(toolCalls, nextToolCalls...)
	totalTokens += tokens
	totalCost += cost

	outputJSON, _ := json.Marshal(out)
	toolCallsJSON, _ := json.Marshal(toolCalls)
	reasoningTrace, _ := json.Marshal([]map[string]any{{
		"step":       "evaluate_signals",
		"results":    len(evidence.Items),
		"confidence": confidence,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}})
	latency := time.Since(startTime).Milliseconds()

	return &ProspectingResult{
		Status:         agent.StatusSuccess,
		Output:         outputJSON,
		ToolCalls:      toolCallsJSON,
		ReasoningTrace: reasoningTrace,
		TotalTokens:    &totalTokens,
		TotalCost:      &totalCost,
		LatencyMs:      &latency,
	}, nil
}

func (a *ProspectingAgent) resolveAccountName(ctx context.Context, lead *crm.Lead) string {
	if lead.AccountID == nil || *lead.AccountID == "" {
		return ""
	}
	acc, err := a.fetchAccount(ctx, *lead.AccountID)
	if err != nil {
		return ""
	}
	return acc.Name
}

func (a *ProspectingAgent) searchSignals(ctx context.Context, workspaceID, query string) *knowledge.SearchResults {
	evidence, err := a.knowledgeSearch.HybridSearch(ctx, knowledge.SearchInput{
		WorkspaceID: workspaceID,
		Query:       query,
		Limit:       5,
	})
	if err != nil {
		return &knowledge.SearchResults{Items: []knowledge.SearchResult{}}
	}
	return evidence
}

func baseProspectingToolCalls(leadID string, accountID *string, query string) []map[string]any {
	toolCalls := []map[string]any{{
		"tool_name":   "get_lead",
		"params":      map[string]any{"lead_id": leadID},
		"executed_at": time.Now().UTC().Format(time.RFC3339),
	}}
	if accountID != nil && *accountID != "" {
		toolCalls = append(toolCalls, map[string]any{
			"tool_name":   "get_account",
			"params":      map[string]any{"account_id": *accountID},
			"executed_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
	toolCalls = append(toolCalls, map[string]any{
		"tool_name": "search_knowledge",
		"params": map[string]any{
			"query": query,
			"limit": 5,
		},
		"executed_at": time.Now().UTC().Format(time.RFC3339),
	})
	return toolCalls
}

func (a *ProspectingAgent) resolveAction(
	ctx context.Context,
	toolCtx context.Context,
	config ProspectingAgentConfig,
	lead *crm.Lead,
	accountName string,
	confidence float64,
) (map[string]any, []map[string]any, int64, float64, error) {
	if confidence <= 0.6 {
		return map[string]any{
			"action":     "skip",
			"reason":     "insufficient_signals",
			"lead_id":    lead.ID,
			"confidence": confidence,
		}, nil, 0, 0, nil
	}

	draft, usedTokens, draftCost, draftErr := a.generateDraft(ctx, config.Language, lead, accountName)
	if draftErr != nil {
		return nil, nil, 0, 0, draftErr
	}

	taskID, createTaskErr := a.createFollowUpTask(toolCtx, lead)
	if createTaskErr != nil {
		return nil, nil, 0, 0, createTaskErr
	}

	createTaskCall := map[string]any{
		"tool_name": "create_task",
		"params": map[string]any{
			"title":       "Follow-up prospecting",
			"description": "Review outreach draft",
			"entity_type": "account",
			"entity_id":   safePtr(lead.AccountID),
		},
		"executed_at": time.Now().UTC().Format(time.RFC3339),
	}

	out := map[string]any{
		"action":     "draft_outreach",
		"details":    map[string]any{"draft": draft, "task_id": taskID},
		"lead_id":    lead.ID,
		"confidence": confidence,
	}
	return out, []map[string]any{createTaskCall}, usedTokens, draftCost + 0.15, nil
}

func (a *ProspectingAgent) fetchLead(ctx context.Context, config ProspectingAgentConfig) (*crm.Lead, error) {
	exec, err := a.toolRegistry.Get(tool.BuiltinGetLead)
	if err != nil {
		return a.leadService.Get(ctx, config.WorkspaceID, config.LeadID)
	}
	raw, err := exec.Execute(ctx, mustJSON(map[string]any{"lead_id": config.LeadID}))
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Lead *crm.Lead `json:"lead"`
	}
	unmarshalErr := json.Unmarshal(raw, &parsed)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}
	if parsed.Lead == nil {
		return nil, ErrLeadNotFound
	}
	return parsed.Lead, nil
}

func (a *ProspectingAgent) fetchAccount(ctx context.Context, accountID string) (*crm.Account, error) {
	exec, err := a.toolRegistry.Get(tool.BuiltinGetAccount)
	if err != nil {
		return a.accountService.Get(ctx, workspaceFromCtx(ctx), accountID)
	}
	raw, err := exec.Execute(ctx, mustJSON(map[string]any{"account_id": accountID}))
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

func (a *ProspectingAgent) createFollowUpTask(ctx context.Context, lead *crm.Lead) (string, error) {
	if lead.AccountID == nil || *lead.AccountID == "" {
		return "", ErrAccountRequired
	}
	exec, err := a.toolRegistry.Get(tool.BuiltinCreateTask)
	if err != nil {
		return "", ErrTaskCreationFailed
	}
	raw, err := exec.Execute(ctx, mustJSON(map[string]any{
		"owner_id":    lead.OwnerID,
		"title":       "Follow-up prospecting",
		"entity_type": "account",
		"entity_id":   *lead.AccountID,
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

func (a *ProspectingAgent) generateDraft(
	ctx context.Context,
	language string,
	lead *crm.Lead,
	accountName string,
) (string, int64, float64, error) {
	if a.llmProvider == nil {
		return "", 0, 0, ErrLLMNotConfigured
	}
	resp, err := a.llmProvider.ChatCompletion(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "Redacta emails de prospección breves, personalizados y profesionales."},
			{Role: "user", Content: fmt.Sprintf("Idioma: %s. Empresa: %s. Estado lead: %s. Fuente: %s. Redacta un email de outreach de máximo 120 palabras.", language, accountName, lead.Status, safePtr(lead.Source))},
		},
		Temperature: 0.2,
		MaxTokens:   180,
	})
	if err != nil {
		return "", 0, 0, err
	}
	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return "", 0, 0, ErrEmptyDraft
	}
	tokens := int64(resp.Tokens)
	if tokens == 0 {
		tokens = int64(len(strings.Fields(content)))
	}
	cost := float64(tokens) * 0.0001
	if cost < 0.1 {
		cost = 0.1
	}
	return content, tokens, cost, nil
}

func (a *ProspectingAgent) checkDailyLimits(ctx context.Context, workspaceID string) error {
	if a.db == nil {
		return nil
	}
	const maxDailyLeads = 50
	const maxDailyCost = 10.0

	var runsToday int
	if err := a.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM agent_run
		WHERE workspace_id = ?
		  AND agent_definition_id = 'prospecting-agent'
		  AND date(created_at) = date('now')
	`, workspaceID).Scan(&runsToday); err != nil {
		return err
	}
	if runsToday >= maxDailyLeads {
		return ErrProspectingDailyLeadLimitExceeded
	}

	var dailyCost float64
	if err := a.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_cost), 0)
		FROM agent_run
		WHERE workspace_id = ?
		  AND agent_definition_id = 'prospecting-agent'
		  AND date(created_at) = date('now')
	`, workspaceID).Scan(&dailyCost); err != nil {
		return err
	}
	if dailyCost >= maxDailyCost {
		return ErrProspectingDailyCostLimitExceeded
	}
	return nil
}

func mustJSON(v any) json.RawMessage {
	raw, _ := json.Marshal(v)
	return raw
}

func workspaceFromCtx(ctx context.Context) string {
	ws, _ := ctx.Value(ctxkeys.WorkspaceID).(string)
	return ws
}

func safePtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// Task 4.5b — domain errors.
var (
	ErrLeadIDRequired                    = &ProspectingError{message: "lead_id is required"}
	ErrLeadNotFound                      = &ProspectingError{message: "lead not found"}
	ErrAccountNotFound                   = &ProspectingError{message: "account not found"}
	ErrAccountRequired                   = &ProspectingError{message: "account_id is required to create follow-up task"}
	ErrTaskCreationFailed                = &ProspectingError{message: "failed to create follow-up task"}
	ErrLLMNotConfigured                  = &ProspectingError{message: "llm provider not configured"}
	ErrEmptyDraft                        = &ProspectingError{message: "llm returned empty draft"}
	ErrProspectingDailyLeadLimitExceeded = &ProspectingError{message: "daily lead limit exceeded"}
	ErrProspectingDailyCostLimitExceeded = &ProspectingError{message: "daily cost limit exceeded"}
)

// ProspectingError is the typed error for the prospecting agent.
type ProspectingError struct {
	message string
}

func (e *ProspectingError) Error() string { return e.message }
