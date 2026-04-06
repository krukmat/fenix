// Package agents provides concrete agent implementations.
// Task 3.7: Support Agent UC-C1
package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/audit"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/domain/usage"
)

type KnowledgeSearchInterface interface {
	HybridSearch(ctx context.Context, input knowledge.SearchInput) (*knowledge.SearchResults, error)
}

type SupportEvidenceBuilder interface {
	BuildEvidencePack(ctx context.Context, input knowledge.BuildEvidencePackInput) (*knowledge.EvidencePack, error)
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
const supportPendingApprovalAction = "pending_approval"
const supportSystemActorID = "system"

const (
	supportResolveThreshold  = 0.85
	supportEscalateThreshold = 0.55
)

// SupportAgent handles customer support case resolution
// UC-C1: Resolver casos de soporte de clientes
type SupportAgent struct {
	orchestrator    *agent.Orchestrator
	toolRegistry    *tool.ToolRegistry
	evidenceBuilder SupportEvidenceBuilder
	db              *sql.DB
	audit           supportAuditLogger
	usage           supportUsageRecorder
}

type supportAuditLogger interface {
	LogWithDetails(ctx context.Context, workspaceID, actorID string, actorType audit.ActorType, action string, entityType, entityID *string, details *audit.EventDetails, outcome audit.Outcome) error
}

type supportUsageRecorder interface {
	RecordEvent(ctx context.Context, input usage.RecordEventInput) (*usage.Event, error)
}

// NewSupportAgent creates a new Support Agent instance
func NewSupportAgent(
	orchestrator *agent.Orchestrator,
	toolRegistry *tool.ToolRegistry,
	evidenceBuilder SupportEvidenceBuilder,
) *SupportAgent {
	return NewSupportAgentWithDBAndUsage(orchestrator, toolRegistry, evidenceBuilder, nil, nil)
}

func NewSupportAgentWithDB(
	orchestrator *agent.Orchestrator,
	toolRegistry *tool.ToolRegistry,
	evidenceBuilder SupportEvidenceBuilder,
	db *sql.DB,
) *SupportAgent {
	return NewSupportAgentWithDBAndUsage(orchestrator, toolRegistry, evidenceBuilder, db, nil)
}

func NewSupportAgentWithDBAndUsage(
	orchestrator *agent.Orchestrator,
	toolRegistry *tool.ToolRegistry,
	evidenceBuilder SupportEvidenceBuilder,
	db *sql.DB,
	usage supportUsageRecorder,
) *SupportAgent {
	var auditLogger supportAuditLogger
	if db != nil {
		auditLogger = audit.NewAuditService(db)
	}
	return &SupportAgent{
		orchestrator:    orchestrator,
		toolRegistry:    toolRegistry,
		evidenceBuilder: evidenceBuilder,
		db:              db,
		audit:           auditLogger,
		usage:           usage,
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

	err = a.completeSupportRun(ctx, run, result)
	if err != nil {
		return run, err
	}

	return run, nil
}

// SupportResult holds the result of a support agent execution
type SupportResult struct {
	Status         string
	Output         json.RawMessage
	RetrievalQuery json.RawMessage
	EvidenceIDs    json.RawMessage
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

	evidence := a.loadSupportEvidencePack(ctx, caseContext.WorkspaceID, config.CustomerQuery)

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

func (a *SupportAgent) determineAction(config SupportAgentConfig, caseContext *CaseContext, evidence *knowledge.EvidencePack) *Action {
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
	toolCtx := supportToolContext(ctx, caseContext, runID)
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
		return nil, fmt.Errorf("%w: %w", ErrSupportCaseContextLoadFailed, err)
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

func buildReasoningTrace(_ SupportAgentConfig, evidence *knowledge.EvidencePack, action *Action) json.RawMessage {
	sourceCount := 0
	if evidence != nil {
		sourceCount = len(evidence.Sources)
	}
	trace := []map[string]any{
		{
			"step":        "analyze",
			"description": "Analyzed customer query",
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		},
		{
			"step":        "search",
			"description": "Built evidence pack from knowledge base",
			"results":     sourceCount,
			"confidence":  supportEvidenceConfidence(evidence),
			"query":       supportEvidenceQuery(evidence),
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		},
		{
			"step":              "policy",
			"description":       "Evaluated approval and execution policy gates",
			"requires_approval": actionRequiresApproval(action),
			"timestamp":         time.Now().UTC().Format(time.RFC3339),
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
	triggeredBy := supportUserID(ctx)
	var triggeredByPtr *string
	if triggeredBy != "" {
		triggeredByPtr = &triggeredBy
	}
	return a.orchestrator.TriggerAgent(ctx, agent.TriggerAgentInput{
		AgentID:        "support-agent",
		WorkspaceID:    config.WorkspaceID,
		TriggeredBy:    triggeredByPtr,
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
	a.auditSupportRun(ctx, run, nil, cause)
	a.recordSupportUsage(ctx, run, nil)
	return cause
}

func (a *SupportAgent) completeSupportRun(ctx context.Context, run *agent.Run, result *SupportResult) error {
	_, err := a.orchestrator.UpdateAgentRun(ctx, run.WorkspaceID, run.ID, agent.RunUpdates{
		Status:               result.Status,
		Output:               result.Output,
		RetrievalQueries:     result.RetrievalQuery,
		RetrievedEvidenceIDs: result.EvidenceIDs,
		ToolCalls:            result.ToolCalls,
		ReasoningTrace:       result.ReasoningTrace,
		TotalTokens:          result.TotalTokens,
		TotalCost:            result.TotalCost,
		LatencyMs:            result.LatencyMs,
		Completed:            true,
	})
	if err != nil {
		return err
	}
	a.auditSupportRun(ctx, run, result, nil)
	a.recordSupportUsage(ctx, run, result)
	return nil
}

func (a *SupportAgent) loadSupportEvidencePack(ctx context.Context, workspaceID, query string) *knowledge.EvidencePack {
	if a.evidenceBuilder == nil {
		return emptySupportEvidencePack(query)
	}

	evidence, err := a.evidenceBuilder.BuildEvidencePack(ctx, knowledge.BuildEvidencePackInput{
		Query:       query,
		WorkspaceID: workspaceID,
		Limit:       5,
	})
	if err != nil {
		return emptySupportEvidencePack(query)
	}
	return evidence
}

func (a *SupportAgent) buildApprovalEscalationResult(
	ctx context.Context,
	startTime time.Time,
	config SupportAgentConfig,
	caseContext *CaseContext,
	evidence *knowledge.EvidencePack,
	action *Action,
	totalTokens *int64,
	totalCost *float64,
) (*SupportResult, error) {
	approvalID, err := a.requestSupportApproval(ctx, caseContext, action)
	if err != nil {
		return nil, err
	}
	escalatedAction := &Action{
		Type:       supportPendingApprovalAction,
		Details:    "Sensitive action requires human approval",
		CaseID:     action.CaseID,
		Status:     supportPendingApprovalAction,
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
	evidence *knowledge.EvidencePack,
	action *Action,
	toolCalls json.RawMessage,
	totalTokens *int64,
	totalCost *float64,
) *SupportResult {
	elapsed := time.Since(startTime).Milliseconds()
	return &SupportResult{
		Status:         supportResultStatus(action.Type),
		Output:         action.toJSON(),
		RetrievalQuery: marshalSupportRetrievalQueries(config.CustomerQuery),
		EvidenceIDs:    marshalSupportEvidenceIDs(evidence),
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

func topEvidenceScore(evidence *knowledge.EvidencePack) float64 {
	if evidence == nil || len(evidence.Sources) == 0 {
		return 0
	}
	return evidence.Sources[0].Score
}

func shouldEscalateSupportAction(score float64, config SupportAgentConfig, caseContext *CaseContext) bool {
	if score < supportEscalateThreshold {
		return isHighPrioritySupportCase(firstNonEmptySupport(config.Priority, caseContext.Priority)) || caseContext.Status == agent.StatusEscalated
	}
	return false
}

func supportToolContext(ctx context.Context, caseContext *CaseContext, runID string) context.Context {
	toolCtx := context.WithValue(ctx, ctxkeys.WorkspaceID, caseContext.WorkspaceID)
	if runID != "" {
		toolCtx = context.WithValue(toolCtx, ctxkeys.RunID, runID)
	}
	if caseContext.OwnerID == "" {
		return toolCtx
	}
	return context.WithValue(toolCtx, ctxkeys.UserID, caseContext.OwnerID)
}

func (a *SupportAgent) recordSupportUsage(ctx context.Context, run *agent.Run, result *SupportResult) {
	if a.usage == nil || run == nil {
		return
	}

	actorID := firstNonEmptySupport(supportUserID(ctx), derefSupportString(run.TriggeredByUserID))
	actorType := string(auditActorTypeForSupport(actorID))
	latencyMs := supportLatency(run, result)
	inputUnits, outputUnits, estimatedCost := supportUsageTotals(result)
	runID := run.ID

	_, _ = a.usage.RecordEvent(ctx, usage.RecordEventInput{
		WorkspaceID:   run.WorkspaceID,
		ActorID:       firstNonEmptySupport(actorID, supportSystemActorID),
		ActorType:     actorType,
		RunID:         &runID,
		InputUnits:    inputUnits,
		OutputUnits:   outputUnits,
		EstimatedCost: estimatedCost,
		LatencyMs:     latencyMs,
	})
}

func (a *SupportAgent) auditSupportRun(ctx context.Context, run *agent.Run, result *SupportResult, cause error) {
	if a.audit == nil || run == nil {
		return
	}

	actorID := firstNonEmptySupport(supportUserID(ctx), derefSupportString(run.TriggeredByUserID), supportSystemActorID)
	actorType := audit.ActorTypeUser
	if actorID == supportSystemActorID {
		actorType = audit.ActorTypeSystem
	}
	caseID := firstNonEmptySupport(firstJSONStringFromRaw(run.TriggerContext, "case_id"), firstJSONStringFromRaw(run.Output, "CaseID"))
	actionType := firstJSONStringFromRaw(run.Output, "Type")
	entityType := "case_ticket"
	entityID := caseID
	metadata := map[string]any{
		"run_id":         run.ID,
		"status":         agent.PublicRunOutcome(run),
		"runtime_status": run.Status,
		"action_type":    actionType,
	}
	if result != nil {
		metadata["retrieval_query_count"] = len(rawStringArray(result.RetrievalQuery))
		metadata["evidence_count"] = len(rawStringArray(result.EvidenceIDs))
	}
	if cause != nil {
		metadata["error"] = cause.Error()
	}

	outcome := audit.OutcomeSuccess
	auditAction := "agent.support.run.completed"
	if cause != nil {
		outcome = audit.OutcomeError
		auditAction = "agent.support.run.failed"
	}

	_ = a.audit.LogWithDetails(
		ctx,
		run.WorkspaceID,
		actorID,
		actorType,
		auditAction,
		&entityType,
		nilIfEmpty(entityID),
		&audit.EventDetails{Metadata: metadata},
		outcome,
	)
}

func supportUserID(ctx context.Context) string {
	userID, _ := ctx.Value(ctxkeys.UserID).(string)
	return userID
}

func auditActorTypeForSupport(actorID string) string {
	if actorID == "" || actorID == supportSystemActorID {
		return supportSystemActorID
	}
	return "user"
}

func supportLatency(run *agent.Run, result *SupportResult) *int64 {
	if result != nil && result.LatencyMs != nil {
		return result.LatencyMs
	}
	if run == nil {
		return nil
	}
	elapsed := time.Since(run.StartedAt).Milliseconds()
	return &elapsed
}

func supportUsageTotals(result *SupportResult) (int64, int64, float64) {
	if result == nil {
		return 0, 0, 0
	}

	totalTokens := int64(0)
	if result.TotalTokens != nil {
		totalTokens = *result.TotalTokens
	}
	totalCost := 0.0
	if result.TotalCost != nil {
		totalCost = *result.TotalCost
	}
	return 0, totalTokens, totalCost
}

func marshalSupportRetrievalQueries(query string) json.RawMessage {
	queries, _ := json.Marshal([]string{query})
	return queries
}

func marshalSupportEvidenceIDs(results *knowledge.EvidencePack) json.RawMessage {
	if results == nil {
		return json.RawMessage("[]")
	}
	ids := make([]string, 0, len(results.Sources))
	for _, item := range results.Sources {
		if trimmed := strings.TrimSpace(item.ID); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	data, _ := json.Marshal(ids)
	return data
}

func rawStringArray(raw json.RawMessage) []string {
	var items []string
	if len(raw) == 0 {
		return items
	}
	_ = json.Unmarshal(raw, &items)
	return items
}

func emptySupportEvidencePack(query string) *knowledge.EvidencePack {
	return &knowledge.EvidencePack{
		SchemaVersion:        knowledge.EvidencePackSchemaVersion,
		Query:                query,
		Sources:              []knowledge.Evidence{},
		SourceCount:          0,
		DedupCount:           0,
		Confidence:           knowledge.ConfidenceLow,
		FilteredCount:        0,
		Warnings:             []string{"no sources found"},
		RetrievalMethodsUsed: []knowledge.EvidenceMethod{},
		BuiltAt:              time.Now().UTC(),
	}
}

func supportEvidenceConfidence(pack *knowledge.EvidencePack) string {
	if pack == nil {
		return string(knowledge.ConfidenceLow)
	}
	return string(pack.Confidence)
}

func supportEvidenceQuery(pack *knowledge.EvidencePack) string {
	if pack == nil {
		return ""
	}
	return pack.Query
}

func firstJSONStringFromRaw(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var payload map[string]any
	if json.Unmarshal(raw, &payload) != nil {
		return ""
	}
	value, _ := payload[key].(string)
	return value
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
