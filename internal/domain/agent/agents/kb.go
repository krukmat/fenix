// Package agents provides concrete agent implementations.
// Task 4.5c — FR-231: KB Agent
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

// Task 4.5c — FR-231: KB Agent config.
type KBAgentConfig struct {
	WorkspaceID       string  `json:"workspace_id"`
	CaseID            string  `json:"case_id"`
	Language          string  `json:"language,omitempty"`
	TriggeredByUserID *string `json:"-"`
}

const kbBaseRunCostEuros = 0.02 // Task 4.5c — baseline per non-LLM KB run.

// KBCaseGetter abstracts case retrieval for testability.
type KBCaseGetter interface {
	Get(ctx context.Context, workspaceID, caseID string) (*crm.CaseTicket, error)
}

// KBAgent implements FR-231 knowledge base extraction flow.
type KBAgent struct {
	orchestrator    *agent.Orchestrator
	toolRegistry    *tool.ToolRegistry
	knowledgeSearch KnowledgeSearchInterface
	llmProvider     llm.LLMProvider
	caseService     KBCaseGetter
	db              *sql.DB
}

// NewKBAgent creates a KB agent.
func NewKBAgent(
	orchestrator *agent.Orchestrator,
	toolRegistry *tool.ToolRegistry,
	knowledgeSearch KnowledgeSearchInterface,
	llmProvider llm.LLMProvider,
	caseService KBCaseGetter,
	db *sql.DB,
) *KBAgent {
	return &KBAgent{
		orchestrator:    orchestrator,
		toolRegistry:    toolRegistry,
		knowledgeSearch: knowledgeSearch,
		llmProvider:     llmProvider,
		caseService:     caseService,
		db:              db,
	}
}

// AllowedTools returns tools allowed to the KB agent.
func (a *KBAgent) AllowedTools() []string {
	return []string{"search_knowledge", "create_knowledge_item", "update_knowledge_item"}
}

// Objective returns objective payload used by the runtime.
func (a *KBAgent) Objective() json.RawMessage {
	objective := map[string]any{
		"role": "knowledge_curator",
		"goal": "convert_case_resolution_into_kb_article",
		"instructions": []string{
			"1. Validate case is resolved or closed",
			"2. Search similar KB articles",
			"3. If score > 0.85 update existing article",
			"4. Otherwise create a new KB article",
		},
	}
	obj, _ := json.Marshal(objective)
	return obj
}

// Run executes KB flow and persists agent_run updates.
func (a *KBAgent) Run(ctx context.Context, config KBAgentConfig) (*agent.Run, error) {
	normalized, err := a.normalizeConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	triggerContext, _ := json.Marshal(map[string]any{
		"case_id":      normalized.CaseID,
		"language":     normalized.Language,
		"agent_type":   "kb",
		"capabilities": a.AllowedTools(),
	})
	inputs, _ := json.Marshal(normalized)

	run, err := a.orchestrator.TriggerAgent(ctx, agent.TriggerAgentInput{
		AgentID:        "kb-agent",
		WorkspaceID:    normalized.WorkspaceID,
		TriggeredBy:    normalized.TriggeredByUserID,
		TriggerType:    agent.TriggerTypeManual,
		TriggerContext: triggerContext,
		Inputs:         inputs,
	})
	if err != nil {
		return nil, err
	}

	toolCtx := context.WithValue(ctx, ctxkeys.WorkspaceID, normalized.WorkspaceID) // Task 4.5c — toolCtx workspace propagation.
	result, err := a.executeKBFlow(toolCtx, toolCtx, normalized) // Task 4.5c — propagate enriched ctx to all downstream calls.
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
		Completed:   true,
		LatencyMs:   result.LatencyMs,
		TotalTokens: result.TotalTokens,
	})
	if err != nil {
		return run, err
	}

	return run, nil
}

func (a *KBAgent) normalizeConfig(ctx context.Context, config KBAgentConfig) (KBAgentConfig, error) {
	if config.CaseID == "" {
		return KBAgentConfig{}, ErrKBCaseIDRequired
	}
	if config.Language == "" {
		config.Language = "es"
	}
	if err := a.checkDailyLimits(ctx, config.WorkspaceID); err != nil {
		return KBAgentConfig{}, err
	}
	return config, nil
}

func (a *KBAgent) markRunFailed(ctx context.Context, run *agent.Run) error {
	_, err := a.orchestrator.UpdateAgentRunStatus(ctx, run.WorkspaceID, run.ID, agent.StatusFailed)
	return err
}

// KBResult holds runtime update payload.
type KBResult struct {
	Status      string
	Output      json.RawMessage
	ToolCalls   json.RawMessage
	TotalTokens *int64
	TotalCost   *float64
	LatencyMs   *int64
}

func (a *KBAgent) executeKBFlow(ctx context.Context, toolCtx context.Context, config KBAgentConfig) (*KBResult, error) {
	start := time.Now()
	tokens := int64(0)
	cost := kbBaseRunCostEuros

	caseTicket, scopedToolCtx, err := a.loadEligibleCase(ctx, toolCtx, config)
	if err != nil {
		return nil, err
	}
	content := safePtr(caseTicket.Description)
	if content == "" {
		content = caseTicket.Subject
	}
	query := caseTicket.Subject + " " + content
	topID, topScore := a.searchSimilarArticles(scopedToolCtx, config.WorkspaceID, query)

	articleID, action, reason, toolCalls, mutErr := a.resolveArticleMutation(scopedToolCtx, config.WorkspaceID, caseTicket.Subject, content, topID, topScore)
	if mutErr != nil {
		return nil, mutErr
	}
	output := buildKBOutput(caseTicket.Metadata, articleID, action, reason)
	calls, _ := json.Marshal(toolCalls)
	latency := time.Since(start).Milliseconds()

	return &KBResult{
		Status:      agent.StatusSuccess,
		Output:      output,
		ToolCalls:   calls,
		TotalTokens: &tokens,
		TotalCost:   &cost,
		LatencyMs:   &latency,
	}, nil
}

func (a *KBAgent) loadEligibleCase(
	ctx context.Context,
	toolCtx context.Context,
	config KBAgentConfig,
) (*crm.CaseTicket, context.Context, error) {
	caseTicket, err := a.caseService.Get(ctx, config.WorkspaceID, config.CaseID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrCaseNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if !isResolvedOrClosedCase(caseTicket.Status) {
		return nil, nil, ErrCaseNotResolved
	}
	userID := resolveKBAgentUserID(caseTicket.OwnerID, config.TriggeredByUserID)
	return caseTicket, context.WithValue(toolCtx, ctxkeys.UserID, userID), nil // Task 4.5c — toolCtx user propagation.
}

func isResolvedOrClosedCase(status string) bool {
	return status == "resolved" || status == "closed"
}

func resolveKBAgentUserID(ownerID string, triggeredBy *string) string {
	if triggeredBy == nil || *triggeredBy == "" {
		return ownerID
	}
	return *triggeredBy
}

func buildKBOutput(metadata *string, articleID, action, reason string) json.RawMessage {
	if caseIsHighSensitivity(metadata) {
		action = "pending_approval"
		reason = "high_sensitivity"
	}
	output, _ := json.Marshal(map[string]any{
		"action":     action,
		"article_id": articleID,
		"reason":     reason,
	})
	return output
}

func (a *KBAgent) resolveArticleMutation(
	ctx context.Context,
	workspaceID, subject, content, topID string,
	topScore float64,
) (string, string, string, []map[string]any, error) {
	if topScore > 0.85 {
		updatedID, err := a.updateKnowledgeArticle(ctx, topID, subject, content)
		if err != nil {
			return "", "", "", nil, err
		}
		return updatedID, "updated", "duplicate_found", []map[string]any{{"tool_name": tool.BuiltinUpdateKnowledgeItem}}, nil
	}
	createdID, err := a.createKnowledgeArticle(ctx, workspaceID, subject, content)
	if err != nil {
		return "", "", "", nil, err
	}
	return createdID, "created", "new_article", []map[string]any{{"tool_name": tool.BuiltinCreateKnowledgeItem}}, nil
}

func (a *KBAgent) searchSimilarArticles(ctx context.Context, workspaceID, query string) (string, float64) {
	results, err := a.knowledgeSearch.HybridSearch(ctx, knowledge.SearchInput{ // Task 4.5c — search with toolCtx.
		WorkspaceID: workspaceID,
		Query:       query,
		Limit:       5,
	})
	if err != nil || results == nil || len(results.Items) == 0 {
		return "", 0
	}
	return results.Items[0].KnowledgeItemID, results.Items[0].Score
}

func (a *KBAgent) updateKnowledgeArticle(ctx context.Context, articleID, subject, content string) (string, error) {
	exec, err := a.toolRegistry.Get(tool.BuiltinUpdateKnowledgeItem)
	if err != nil {
		return "", fmt.Errorf("update_knowledge_item tool not registered: %w", err)
	}
	raw, err := exec.Execute(ctx, mustJSON(map[string]any{
		"id":      articleID,
		"title":   "Solución: " + subject,
		"content": content,
	}))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKBArticleUpdateFailed, err)
	}
	var parsed struct {
		KnowledgeItemID string `json:"knowledge_item_id"`
	}
	if unmarshalErr := json.Unmarshal(raw, &parsed); unmarshalErr != nil {
		return "", unmarshalErr
	}
	if parsed.KnowledgeItemID == "" {
		return articleID, nil
	}
	return parsed.KnowledgeItemID, nil
}

func (a *KBAgent) createKnowledgeArticle(ctx context.Context, workspaceID, subject, content string) (string, error) {
	exec, err := a.toolRegistry.Get(tool.BuiltinCreateKnowledgeItem)
	if err != nil {
		return "", fmt.Errorf("create_knowledge_item tool not registered: %w", err)
	}
	raw, err := exec.Execute(ctx, mustJSON(map[string]any{
		"workspace_id": workspaceID,
		"source_type":  "kb_article",
		"title":        "Solución: " + subject,
		"content":      content,
	}))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKBArticleCreationFailed, err)
	}
	var parsed struct {
		KnowledgeItemID string `json:"knowledge_item_id"`
	}
	if unmarshalErr := json.Unmarshal(raw, &parsed); unmarshalErr != nil {
		return "", unmarshalErr
	}
	if parsed.KnowledgeItemID == "" {
		return "", ErrKBArticleCreationFailed
	}
	return parsed.KnowledgeItemID, nil
}

func (a *KBAgent) checkDailyLimits(ctx context.Context, workspaceID string) error {
	if a.db == nil {
		return nil
	}
	const maxDailyArticles = 10
	var runsToday int
	if err := a.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM agent_run
		WHERE workspace_id = ?
		  AND agent_definition_id = 'kb-agent'
		  AND date(created_at) = date('now')
	`, workspaceID).Scan(&runsToday); err != nil {
		return err
	}
	if runsToday >= maxDailyArticles {
		return ErrKBDailyLimitExceeded
	}
	return nil
}

func caseIsHighSensitivity(metadata *string) bool {
	if metadata == nil {
		return false
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(*metadata), &m); err != nil {
		return false
	}
	return m["sensitivity"] == "high"
}

var (
	ErrKBCaseIDRequired       = &KBError{message: "case_id is required"}
	ErrCaseNotFound            = &KBError{message: "case not found"}
	ErrCaseNotResolved         = &KBError{message: "case is not resolved or closed"}
	ErrKBDailyLimitExceeded    = &KBError{message: "daily article creation limit exceeded (max 10/day)"}
	ErrKBArticleCreationFailed = &KBError{message: "failed to create knowledge article"}
	ErrKBArticleUpdateFailed   = &KBError{message: "failed to update knowledge article"}
)

// KBError is the typed error for the KB agent.
type KBError struct{ message string }

func (e *KBError) Error() string { return e.message }
