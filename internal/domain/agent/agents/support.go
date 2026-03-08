// Package agents provides concrete agent implementations.
// Task 3.7: Support Agent UC-C1
package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
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

const supportActionUpdateCase = "update_case"
const supportActionAbstain = "abstain"
const supportActionEscalate = "escalate"

const (
	supportResolveThreshold  = 0.85
	supportEscalateThreshold = 0.55
)

// SupportAgent handles customer support case resolution
// UC-C1: Resolver casos de soporte de clientes
type SupportAgent struct {
	orchestrator    *agent.Orchestrator
	toolRegistry    *tool.ToolRegistry
	knowledgeSearch KnowledgeSearchInterface
	db              *sql.DB
}

// NewSupportAgent creates a new Support Agent instance
func NewSupportAgent(
	orchestrator *agent.Orchestrator,
	toolRegistry *tool.ToolRegistry,
	knowledgeSearch KnowledgeSearchInterface,
) *SupportAgent {
	return NewSupportAgentWithDB(orchestrator, toolRegistry, knowledgeSearch, nil)
}

func NewSupportAgentWithDB(
	orchestrator *agent.Orchestrator,
	toolRegistry *tool.ToolRegistry,
	knowledgeSearch KnowledgeSearchInterface,
	db *sql.DB,
) *SupportAgent {
	return &SupportAgent{
		orchestrator:    orchestrator,
		toolRegistry:    toolRegistry,
		knowledgeSearch: knowledgeSearch,
		db:              db,
	}
}

// AllowedTools returns the tools available to the Support Agent
func (a *SupportAgent) AllowedTools() []string {
	return []string{
		supportActionUpdateCase,
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
	if err := validateSupportConfig(config); err != nil {
		return nil, err
	}

	run, err := a.triggerSupportRun(ctx, config)
	if err != nil {
		return nil, err
	}

	result, err := a.executeSupportFlow(ctx, run.ID, config)
	if err != nil {
		return run, a.failSupportRun(ctx, run, err)
	}

	if err := a.completeSupportRun(ctx, run, result); err != nil {
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
func (a *SupportAgent) executeSupportFlow(ctx context.Context, runID string, config SupportAgentConfig) (*SupportResult, error) {
	startTime := time.Now()
	var totalTokens int64
	var totalCost float64

	caseContext, err := a.getCaseContext(ctx, config.WorkspaceID, config.CaseID)
	if err != nil {
		return nil, err
	}

	evidence := a.loadSupportEvidence(ctx, caseContext.WorkspaceID, config.CustomerQuery)

	action := a.determineAction(config, caseContext, evidence)
	if actionRequiresApproval(action) {
		return a.buildApprovalEscalationResult(ctx, startTime, config, caseContext, evidence, action, &totalTokens, &totalCost)
	}

	toolCalls, handoffReason, err := a.executeAction(ctx, runID, action, caseContext)
	if err != nil {
		return nil, err
	}
	if handoffReason != "" {
		action.NextSteps = append(action.NextSteps, "handoff_created")
	}
	return buildSupportResult(startTime, config, evidence, action, toolCalls, &totalTokens, &totalCost), nil
}

// CaseContext holds the context of a support case
type CaseContext struct {
	ID           string
	WorkspaceID  string
	Subject      string
	Description  string
	Status       string
	Priority     string
	AccountID    string
	ContactID    string
	OwnerID      string
	ContactName  string
	ContactEmail string
}

func (a *SupportAgent) getCaseContext(ctx context.Context, workspaceID, caseID string) (*CaseContext, error) {
	if a.db == nil {
		return nil, ErrSupportDBNotConfigured
	}

	caseTicket, err := a.loadSupportCase(ctx, workspaceID, caseID)
	if err != nil {
		return nil, err
	}

	ctxOut := buildCaseContext(caseTicket)
	return a.enrichCaseContextWithContact(ctx, ctxOut)
}

func (a *SupportAgent) determineAction(config SupportAgentConfig, caseContext *CaseContext, evidence *knowledge.SearchResults) *Action {
	score := topEvidenceScore(evidence)
	if shouldResolveSupportAction(score) {
		return supportResolvedAction(config)
	}
	if shouldEscalateSupportAction(score, config, caseContext) {
		return supportEscalatedAction(config)
	}
	return supportAbstainedAction(config)
}

func (a *SupportAgent) executeAction(ctx context.Context, runID string, action *Action, caseContext *CaseContext) (json.RawMessage, string, error) {
	toolCtx := supportToolContext(ctx, caseContext)
	switch action.Type {
	case supportActionUpdateCase:
		return a.executeResolvedAction(toolCtx, action, caseContext)
	case supportActionAbstain:
		return a.executeAbstainedAction(toolCtx, action, caseContext)
	case supportActionEscalate:
		return a.executeEscalatedAction(toolCtx, runID, action, caseContext)
	default:
		raw, err := json.Marshal([]map[string]any{})
		return raw, "", err
	}
}

func (a *SupportAgent) loadSupportCase(ctx context.Context, workspaceID, caseID string) (*crm.CaseTicket, error) {
	caseTicket, err := crm.NewCaseService(a.db).Get(ctx, workspaceID, caseID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSupportCaseContextLoadFailed, err)
	}
	return caseTicket, nil
}

func buildCaseContext(caseTicket *crm.CaseTicket) *CaseContext {
	return &CaseContext{
		ID:          caseTicket.ID,
		WorkspaceID: caseTicket.WorkspaceID,
		Subject:     caseTicket.Subject,
		Description: derefSupportString(caseTicket.Description),
		Status:      caseTicket.Status,
		Priority:    caseTicket.Priority,
		AccountID:   derefSupportString(caseTicket.AccountID),
		ContactID:   derefSupportString(caseTicket.ContactID),
		OwnerID:     caseTicket.OwnerID,
	}
}

func (a *SupportAgent) enrichCaseContextWithContact(ctx context.Context, ctxOut *CaseContext) (*CaseContext, error) {
	if ctxOut.ContactID == "" {
		return ctxOut, nil
	}

	contact, contactErr := crm.NewContactService(a.db).Get(ctx, ctxOut.WorkspaceID, ctxOut.ContactID)
	if contactErr != nil {
		return ctxOut, nil
	}
	ctxOut.ContactName = supportContactName(contact)
	ctxOut.ContactEmail = derefSupportString(contact.Email)
	return ctxOut, nil
}

// Action represents an action to take for a support case
type Action struct {
	Type       string
	Details    string
	CaseID     string
	Status     string
	Confidence int
	NextSteps  []string
	ApprovalID string
	Metadata   string
}

func (a *SupportAgent) executeResolvedAction(toolCtx context.Context, action *Action, caseContext *CaseContext) (json.RawMessage, string, error) {
	toolCalls := []map[string]any{}
	if err := a.appendCaseUpdateToolCall(toolCtx, &toolCalls, action, caseContext); err != nil {
		return nil, "", err
	}
	if err := a.appendReplyToolCall(toolCtx, &toolCalls, action, caseContext); err != nil {
		return nil, "", err
	}
	raw, err := json.Marshal(toolCalls)
	return raw, "", err
}

func (a *SupportAgent) executeAbstainedAction(toolCtx context.Context, action *Action, caseContext *CaseContext) (json.RawMessage, string, error) {
	toolCalls := []map[string]any{}
	if err := a.appendReplyToolCall(toolCtx, &toolCalls, action, caseContext); err != nil {
		return nil, "", err
	}
	raw, err := json.Marshal(toolCalls)
	return raw, "", err
}

func (a *SupportAgent) executeEscalatedAction(toolCtx context.Context, runID string, action *Action, caseContext *CaseContext) (json.RawMessage, string, error) {
	toolCalls := []map[string]any{}
	if err := a.appendEscalationTaskToolCall(toolCtx, &toolCalls, caseContext); err != nil {
		return nil, "", err
	}
	if err := a.initiateSupportHandoff(toolCtx, runID, caseContext, action); err != nil {
		return nil, "", err
	}
	raw, err := json.Marshal(toolCalls)
	return raw, action.Details, err
}

func (a *Action) toJSON() json.RawMessage {
	data, _ := json.Marshal(a)
	return data
}

func nilIfEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func (a *SupportAgent) requestSupportApproval(ctx context.Context, caseContext *CaseContext, action *Action) (string, error) {
	if a.db == nil {
		return "", ErrSupportApprovalCreationFailed
	}
	requestedBy := requesterFromCtxOrDefault(ctx, "")
	if requestedBy == "" {
		requestedBy = "support_lead"
	}
	payload := map[string]any{
		"case_id":          caseContext.ID,
		"proposed_action":  action.Type,
		"proposed_status":  action.Status,
		"proposed_details": action.Details,
	}
	return createApprovalGateRequest(ctx, a.db, approvalGateInput{
		WorkspaceID:  caseContext.WorkspaceID,
		RequestedBy:  requestedBy,
		ApproverID:   requestedBy,
		Action:       "support.case.update",
		ResourceType: "case_ticket",
		ResourceID:   caseContext.ID,
		Reason:       "high_sensitivity",
		Payload:      payload,
		TTL:          24 * time.Hour,
	})
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

func validateSupportConfig(config SupportAgentConfig) error {
	if config.CaseID == "" {
		return ErrCaseIDRequired
	}
	if config.WorkspaceID == "" {
		return ErrWorkspaceIDRequired
	}
	return nil
}

func (a *SupportAgent) triggerSupportRun(ctx context.Context, config SupportAgentConfig) (*agent.Run, error) {
	triggerContext, inputs := supportRunPayloads(config, a.AllowedTools())
	return a.orchestrator.TriggerAgent(ctx, agent.TriggerAgentInput{
		AgentID:        "support-agent",
		WorkspaceID:    config.WorkspaceID,
		TriggeredBy:    nil,
		TriggerType:    agent.TriggerTypeManual,
		TriggerContext: triggerContext,
		Inputs:         inputs,
	})
}

func supportRunPayloads(config SupportAgentConfig, allowedTools []string) (json.RawMessage, json.RawMessage) {
	triggerContext, _ := json.Marshal(map[string]any{
		"case_id":         config.CaseID,
		"customer_query":  config.CustomerQuery,
		"context_account": config.ContextAccount,
		"context_contact": config.ContextContact,
		"language":        config.Language,
		"priority":        config.Priority,
		"agent_type":      "support",
		"capabilities":    allowedTools,
	})
	inputs, _ := json.Marshal(config)
	return triggerContext, inputs
}

func (a *SupportAgent) failSupportRun(ctx context.Context, run *agent.Run, cause error) error {
	_, err := a.orchestrator.UpdateAgentRunStatus(ctx, run.WorkspaceID, run.ID, agent.StatusFailed)
	if err != nil {
		return err
	}
	return cause
}

func (a *SupportAgent) completeSupportRun(ctx context.Context, run *agent.Run, result *SupportResult) error {
	_, err := a.orchestrator.UpdateAgentRun(ctx, run.WorkspaceID, run.ID, agent.RunUpdates{
		Status:         result.Status,
		Output:         result.Output,
		ToolCalls:      result.ToolCalls,
		ReasoningTrace: result.ReasoningTrace,
		TotalTokens:    result.TotalTokens,
		TotalCost:      result.TotalCost,
		LatencyMs:      result.LatencyMs,
		Completed:      true,
	})
	return err
}

func (a *SupportAgent) loadSupportEvidence(ctx context.Context, workspaceID, query string) *knowledge.SearchResults {
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

func (a *SupportAgent) buildApprovalEscalationResult(
	ctx context.Context,
	startTime time.Time,
	config SupportAgentConfig,
	caseContext *CaseContext,
	evidence *knowledge.SearchResults,
	action *Action,
	totalTokens *int64,
	totalCost *float64,
) (*SupportResult, error) {
	approvalID, err := a.requestSupportApproval(ctx, caseContext, action)
	if err != nil {
		return nil, err
	}
	escalatedAction := &Action{
		Type:       "pending_approval",
		Details:    "Sensitive action requires human approval",
		CaseID:     action.CaseID,
		Status:     "pending_approval",
		Confidence: action.Confidence,
		ApprovalID: approvalID,
	}
	toolCalls, _ := json.Marshal([]map[string]any{{"tool_name": "approval.requested"}})
	result := buildSupportResult(startTime, config, evidence, escalatedAction, toolCalls, totalTokens, totalCost)
	result.Status = agent.StatusEscalated
	return result, nil
}

func buildSupportResult(
	startTime time.Time,
	config SupportAgentConfig,
	evidence *knowledge.SearchResults,
	action *Action,
	toolCalls json.RawMessage,
	totalTokens *int64,
	totalCost *float64,
) *SupportResult {
	elapsed := time.Since(startTime).Milliseconds()
	return &SupportResult{
		Status:         supportResultStatus(action.Type),
		Output:         action.toJSON(),
		ToolCalls:      toolCalls,
		ReasoningTrace: buildReasoningTrace(config, evidence, action),
		TotalTokens:    totalTokens,
		TotalCost:      totalCost,
		LatencyMs:      &elapsed,
	}
}

func shouldResolveSupportAction(score float64) bool {
	return score >= supportResolveThreshold
}

func supportResolvedAction(config SupportAgentConfig) *Action {
	return &Action{
		Type:       supportActionUpdateCase,
		Details:    "Applied solution from knowledge base",
		CaseID:     config.CaseID,
		Status:     "resolved",
		Confidence: 90,
		NextSteps:  []string{"send_resolution_email"},
		Metadata:   config.Priority,
	}
}

func supportEscalatedAction(config SupportAgentConfig) *Action {
	return &Action{
		Type:       supportActionEscalate,
		Details:    "Insufficient confidence for autonomous resolution",
		CaseID:     config.CaseID,
		Status:     "escalated",
		Confidence: 30,
		NextSteps:  []string{"create_support_handoff"},
	}
}

func supportAbstainedAction(config SupportAgentConfig) *Action {
	return &Action{
		Type:       supportActionAbstain,
		Details:    "Evidence is not strong enough to resolve the case automatically",
		CaseID:     config.CaseID,
		Status:     "open",
		Confidence: 50,
		NextSteps:  []string{"await_human_review_if_customer_replies"},
	}
}

func (a *SupportAgent) appendCaseUpdateToolCall(
	toolCtx context.Context,
	toolCalls *[]map[string]any,
	action *Action,
	caseContext *CaseContext,
) error {
	updateOut, err := a.executeTool(toolCtx, caseContext.WorkspaceID, tool.BuiltinUpdateCase, map[string]any{
		"case_id":  action.CaseID,
		"status":   action.Status,
		"priority": caseContext.Priority,
	})
	if err != nil {
		return err
	}
	*toolCalls = append(*toolCalls, supportToolCall(tool.BuiltinUpdateCase, updateOut))
	return nil
}

func (a *SupportAgent) appendReplyToolCall(
	toolCtx context.Context,
	toolCalls *[]map[string]any,
	action *Action,
	caseContext *CaseContext,
) error {
	replyOut, err := a.executeTool(toolCtx, caseContext.WorkspaceID, tool.BuiltinSendReply, map[string]any{
		"case_id":     action.CaseID,
		"body":        buildSupportReply(caseContext, action),
		"is_internal": false,
	})
	if err != nil {
		return err
	}
	*toolCalls = append(*toolCalls, supportToolCall(tool.BuiltinSendReply, replyOut))
	return nil
}

func (a *SupportAgent) appendEscalationTaskToolCall(
	toolCtx context.Context,
	toolCalls *[]map[string]any,
	caseContext *CaseContext,
) error {
	taskOut, err := a.executeTool(toolCtx, caseContext.WorkspaceID, tool.BuiltinCreateTask, map[string]any{
		"owner_id":    caseContext.OwnerID,
		"title":       "Escalated support case: " + caseContext.Subject,
		"entity_type": "case",
		"entity_id":   caseContext.ID,
	})
	if err != nil {
		return err
	}
	*toolCalls = append(*toolCalls, supportToolCall(tool.BuiltinCreateTask, taskOut))
	return nil
}

func supportToolCall(toolName string, result json.RawMessage) map[string]any {
	return map[string]any{
		"tool_name":   toolName,
		"result":      rawJSONMap(result),
		"executed_at": time.Now().UTC().Format(time.RFC3339),
	}
}

// Errors

var ErrCaseIDRequired = &SupportError{message: "case_id is required"}
var ErrWorkspaceIDRequired = &SupportError{message: "workspace_id is required"}
var ErrSupportDBNotConfigured = &SupportError{message: "support agent db is required"}
var ErrSupportApprovalCreationFailed = &SupportError{message: "failed to create approval request"}
var ErrSupportCaseContextLoadFailed = &SupportError{message: "failed to load support case context"}

type SupportError struct {
	message string
}

func (e *SupportError) Error() string {
	return e.message
}

func supportResultStatus(actionType string) string {
	switch actionType {
	case supportActionEscalate:
		return agent.StatusEscalated
	case supportActionAbstain:
		return agent.StatusAbstained
	default:
		return agent.StatusSuccess
	}
}

func actionRequiresApproval(action *Action) bool {
	return action.Type == supportActionUpdateCase && isHighSensitivityMetadata(nilIfEmpty(action.Metadata))
}

func topEvidenceScore(evidence *knowledge.SearchResults) float64 {
	if evidence == nil || len(evidence.Items) == 0 {
		return 0
	}
	return evidence.Items[0].Score
}

func shouldEscalateSupportAction(score float64, config SupportAgentConfig, caseContext *CaseContext) bool {
	if score < supportEscalateThreshold {
		return isHighPrioritySupportCase(firstNonEmptySupport(config.Priority, caseContext.Priority)) || caseContext.Status == agent.StatusEscalated
	}
	return false
}

func supportToolContext(ctx context.Context, caseContext *CaseContext) context.Context {
	toolCtx := context.WithValue(ctx, ctxkeys.WorkspaceID, caseContext.WorkspaceID)
	if caseContext.OwnerID == "" {
		return toolCtx
	}
	return context.WithValue(toolCtx, ctxkeys.UserID, caseContext.OwnerID)
}

func (a *SupportAgent) executeTool(ctx context.Context, workspaceID, toolName string, payload map[string]any) (json.RawMessage, error) {
	raw, _ := json.Marshal(payload)
	return a.toolRegistry.Execute(ctx, workspaceID, toolName, raw)
}

func buildSupportReply(caseContext *CaseContext, action *Action) string {
	if action.Type == supportActionUpdateCase {
		if caseContext.ContactName != "" {
			return "Hola " + caseContext.ContactName + ", hemos aplicado una solucion y marcado el caso como resuelto."
		}
		return "Hemos aplicado una solucion y marcado el caso como resuelto."
	}
	return "No tengo evidencia suficiente para resolver el caso automaticamente. Un agente revisara el caso si necesitas mas ayuda."
}

func rawJSONMap(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var out any
	if json.Unmarshal(raw, &out) != nil {
		return string(raw)
	}
	return out
}

func derefSupportString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func supportContactName(contact *crm.Contact) string {
	name := contact.FirstName
	if contact.LastName != "" {
		if name != "" {
			name += " "
		}
		name += contact.LastName
	}
	return name
}

func firstNonEmptySupport(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func isHighPrioritySupportCase(priority string) bool {
	return priority == "high" || priority == "urgent"
}

func (a *SupportAgent) initiateSupportHandoff(ctx context.Context, runID string, caseContext *CaseContext, action *Action) error {
	if a.db == nil {
		return ErrSupportDBNotConfigured
	}
	handoffSvc := agent.NewHandoffService(a.db, crm.NewCaseService(a.db), nil)
	_, err := handoffSvc.InitiateHandoff(ctx, caseContext.WorkspaceID, runID, caseContext.ID, action.Details)
	if err != nil {
		return err
	}
	return nil
}
