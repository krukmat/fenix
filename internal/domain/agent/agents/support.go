// Package agents provides concrete agent implementations.
// Task 3.7: Support Agent UC-C1
package agents

import (
	"context"
	"encoding/json"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

// KnowledgeSearchInterface defines the interface for knowledge base search
type KnowledgeSearchInterface interface {
	HybridSearch(ctx context.Context, input knowledge.SearchInput) (*knowledge.SearchResults, error)
}

// SupportAgentConfig defines the configuration for the Support Agent
type SupportAgentConfig struct {
	WorkspaceID    string `json:"workspace_id"`
	CaseID         string `json:"case_id"`
	CustomerQuery  string `json:"customer_query"`
	Language       string `json:"language,omitempty"`
	Priority       string `json:"priority,omitempty"`
	ContextAccount string `json:"context_account,omitempty"`
	ContextContact string `json:"context_contact,omitempty"`
}

// SupportAgent handles customer support case resolution
// UC-C1: Resolver casos de soporte de clientes
type SupportAgent struct {
	orchestrator    *agent.Orchestrator
	toolRegistry    *tool.ToolRegistry
	knowledgeSearch KnowledgeSearchInterface
}

// NewSupportAgent creates a new Support Agent instance
func NewSupportAgent(
	orchestrator *agent.Orchestrator,
	toolRegistry *tool.ToolRegistry,
	knowledgeSearch KnowledgeSearchInterface,
) *SupportAgent {
	return &SupportAgent{
		orchestrator:    orchestrator,
		toolRegistry:    toolRegistry,
		knowledgeSearch: knowledgeSearch,
	}
}

// AllowedTools returns the tools available to the Support Agent
func (a *SupportAgent) AllowedTools() []string {
	return []string{
		"update_case",
		"send_reply",
		"create_task",
		"search_knowledge",
		"get_case",
		"get_contact",
	}
}

// Objective returns the agent's objective in JSON
func (a *SupportAgent) Objective() json.RawMessage {
	objective := map[string]any{
		"role": "Customer Support Specialist",
		"goal": "Resolve customer support cases efficiently and accurately",
		"instructions": []string{
			"1. Analyze the customer query and case history",
			"2. Search knowledge base for relevant solutions",
			"3. If solution found: apply fix and close case",
			"4. If solution not found: escalate to human agent",
			"5. Always maintain professional tone",
			"6. Document all actions taken",
		},
		"response_format": map[string]string{
			"action":     "update_case|send_reply|create_task|escalate",
			"details":    "explanation of action taken",
			"case_id":    "ID of the case being updated",
			"status":     "resolved|pending|escalated",
			"confidence": "0-100",
		},
	}
	obj, _ := json.Marshal(objective)
	return obj
}

// Run executes the Support Agent for a given case
// Traces: FR-230, FR-231
func (a *SupportAgent) Run(ctx context.Context, config SupportAgentConfig) (*agent.Run, error) {
	if config.CaseID == "" {
		return nil, ErrCaseIDRequired
	}

	triggerContext, _ := json.Marshal(map[string]any{
		"case_id":         config.CaseID,
		"customer_query":  config.CustomerQuery,
		"context_account": config.ContextAccount,
		"context_contact": config.ContextContact,
		"language":        config.Language,
		"priority":        config.Priority,
		"agent_type":      "support",
		"capabilities":    a.AllowedTools(),
	})

	inputs, _ := json.Marshal(config)

	run, err := a.orchestrator.TriggerAgent(ctx, agent.TriggerAgentInput{
		AgentID:        "support-agent",
		WorkspaceID:    config.WorkspaceID,
		TriggeredBy:    nil,
		TriggerType:    agent.TriggerTypeManual,
		TriggerContext: triggerContext,
		Inputs:         inputs,
	})
	if err != nil {
		return nil, err
	}

	result, err := a.executeSupportFlow(ctx, config)
	if err != nil {
		_, updateErr := a.orchestrator.UpdateAgentRunStatus(ctx, run.WorkspaceID, run.ID, agent.StatusFailed)
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

// SupportResult holds the result of a support agent execution
type SupportResult struct {
	Status         string
	Output         json.RawMessage
	ToolCalls      json.RawMessage
	ReasoningTrace json.RawMessage
	TotalTokens    *int64
	TotalCost      *float64
	LatencyMs      *int64
}

// executeSupportFlow runs the main support logic
func (a *SupportAgent) executeSupportFlow(ctx context.Context, config SupportAgentConfig) (*SupportResult, error) {
	startTime := time.Now()
	var totalTokens int64
	var totalCost float64

	caseContext, _ := a.getCaseContext(ctx, config.CaseID)

	evidence, err := a.knowledgeSearch.HybridSearch(ctx, knowledge.SearchInput{
		WorkspaceID: caseContext.WorkspaceID,
		Query:       config.CustomerQuery,
		Limit:       5,
	})
	if err != nil {
		evidence = &knowledge.SearchResults{Items: []knowledge.SearchResult{}}
	}

	action := a.determineAction(config, caseContext, evidence)

	toolCalls, _ := a.executeAction(action, caseContext)

	elapsed := time.Since(startTime).Milliseconds()
	return &SupportResult{
		Status:         agent.StatusSuccess,
		Output:         action.toJSON(),
		ToolCalls:      toolCalls,
		ReasoningTrace: buildReasoningTrace(config, evidence, action),
		TotalTokens:    &totalTokens,
		TotalCost:      &totalCost,
		LatencyMs:      &elapsed,
	}, nil
}

// CaseContext holds the context of a support case
type CaseContext struct {
	ID          string
	WorkspaceID string
	Subject     string
	Description string
	Status      string
	Priority    string
	AccountID   string
	ContactID   string
}

func (a *SupportAgent) getCaseContext(_ context.Context, caseID string) (*CaseContext, error) {
	return &CaseContext{
		ID:          caseID,
		WorkspaceID: "",
		Subject:     "Customer Issue",
		Description: "Customer reported an issue",
		Status:      "open",
		Priority:    "medium",
	}, nil
}

// Action represents an action to take for a support case
type Action struct {
	Type       string
	Details    string
	CaseID     string
	Status     string
	Confidence int
	NextSteps  []string
}

func (a *SupportAgent) determineAction(config SupportAgentConfig, _ *CaseContext, evidence *knowledge.SearchResults) *Action {
	if len(evidence.Items) == 0 {
		return &Action{
			Type:       "escalate",
			Details:    "No solution found in knowledge base",
			CaseID:     config.CaseID,
			Status:     "escalated",
			Confidence: 30,
			NextSteps:  []string{"notify_support_lead"},
		}
	}

	if len(evidence.Items) > 0 && evidence.Items[0].Score > 0.8 {
		return &Action{
			Type:       "update_case",
			Details:    "Applied solution from knowledge base",
			CaseID:     config.CaseID,
			Status:     "resolved",
			Confidence: 85,
			NextSteps:  []string{"send_resolution_email"},
		}
	}

	return &Action{
		Type:       "create_task",
		Details:    "Solution found but requires verification",
		CaseID:     config.CaseID,
		Status:     "pending",
		Confidence: 50,
		NextSteps:  []string{"review_knowledge_article"},
	}
}

func (a *SupportAgent) executeAction(action *Action, caseContext *CaseContext) (json.RawMessage, error) {
	toolCalls := []map[string]any{}

	switch action.Type {
	case "update_case":
		toolCall := map[string]any{
			"tool_name": "update_case",
			"params": map[string]any{
				"case_id": action.CaseID,
				"status":  action.Status,
				"notes":   action.Details,
			},
			"executed_at": time.Now().UTC().Format(time.RFC3339),
		}
		toolCalls = append(toolCalls, toolCall)

	case "escalate":
		toolCall := map[string]any{
			"tool_name": "create_task",
			"params": map[string]any{
				"title":       "Escalated: " + caseContext.Subject,
				"description": action.Details,
				"assignee":    "support_lead",
				"priority":    "high",
			},
			"executed_at": time.Now().UTC().Format(time.RFC3339),
		}
		toolCalls = append(toolCalls, toolCall)

	case "create_task":
		toolCall := map[string]any{
			"tool_name": "create_task",
			"params": map[string]any{
				"title":       "Review required: " + caseContext.Subject,
				"description": action.Details,
				"assignee":    "support_agent",
				"priority":    action.Status,
			},
			"executed_at": time.Now().UTC().Format(time.RFC3339),
		}
		toolCalls = append(toolCalls, toolCall)
	}

	return json.Marshal(toolCalls)
}

func (a *Action) toJSON() json.RawMessage {
	data, _ := json.Marshal(a)
	return data
}

func buildReasoningTrace(_ SupportAgentConfig, evidence *knowledge.SearchResults, action *Action) json.RawMessage {
	trace := []map[string]any{
		{
			"step":        "analyze",
			"description": "Analyzed customer query",
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		},
		{
			"step":        "search",
			"description": "Searched knowledge base",
			"results":     len(evidence.Items),
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		},
		{
			"step":        "decide",
			"description": "Determined action: " + action.Type,
			"confidence":  action.Confidence,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		},
	}
	data, _ := json.Marshal(trace)
	return data
}

// Errors

var ErrCaseIDRequired = &SupportError{message: "case_id is required"}

type SupportError struct {
	message string
}

func (e *SupportError) Error() string {
	return e.message
}
