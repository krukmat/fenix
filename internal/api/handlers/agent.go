package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent/agents"
	tooldomain "github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

const (
	defaultAgentLanguage = "es"
	errQueryRequired     = "query is required"
	queryWorkflowID      = "workflow_id"
	dispatchReasonKey    = "reason"
	rejectionReasonKey   = "rejection_reason"
)

// AgentHandler handles agent-related HTTP requests
type AgentHandler struct {
	orchestrator *agent.Orchestrator
}

// NewAgentHandler creates a new AgentHandler
func NewAgentHandler(orchestrator *agent.Orchestrator) *AgentHandler {
	return &AgentHandler{orchestrator: orchestrator}
}

// Request/Response types

type triggerAgentRequest struct {
	AgentID        string          `json:"agent_id"`
	TriggerType    string          `json:"trigger_type"`
	TriggerContext json.RawMessage `json:"trigger_context,omitempty"`
	Inputs         json.RawMessage `json:"inputs,omitempty"`
}

type agentRunResponse struct {
	ID                string          `json:"id"`
	WorkspaceID       string          `json:"workspaceId"`
	AgentDefinitionID string          `json:"agentDefinitionId"`
	TriggeredByUserID *string         `json:"triggeredByUserId,omitempty"`
	TriggerType       string          `json:"triggerType"`
	Status            string          `json:"status"`
	RuntimeStatus     string          `json:"runtime_status,omitempty"`
	Inputs            json.RawMessage `json:"inputs,omitempty"`
	Output            json.RawMessage `json:"output,omitempty"`
	ToolCalls         json.RawMessage `json:"toolCalls,omitempty"`
	ReasoningTrace    json.RawMessage `json:"reasoningTrace,omitempty"`
	TotalTokens       *int64          `json:"totalTokens,omitempty"`
	TotalCost         *float64        `json:"totalCost,omitempty"`
	LatencyMs         *int64          `json:"latencyMs,omitempty"`
	TraceID           *string         `json:"traceId,omitempty"`
	WorkflowID        *string         `json:"workflow_id,omitempty"`
	EntityType        *string         `json:"entity_type,omitempty"`
	EntityID          *string         `json:"entity_id,omitempty"`
	RejectionReason   *string         `json:"rejection_reason,omitempty"`
	StartedAt         string          `json:"startedAt"`
	CompletedAt       *string         `json:"completedAt,omitempty"`
	CreatedAt         string          `json:"createdAt"`
}

type agentDefinitionResponse struct {
	ID           string          `json:"id"`
	WorkspaceID  string          `json:"workspaceId"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	AgentType    string          `json:"agentType"`
	Objective    json.RawMessage `json:"objective,omitempty"`
	AllowedTools []string        `json:"allowedTools,omitempty"`
	Status       string          `json:"status"`
	CreatedAt    string          `json:"createdAt"`
	UpdatedAt    string          `json:"updatedAt"`
}

// buildTriggerInput converts an HTTP request into a domain TriggerAgentInput.
// Applies defaults and nil-guards for optional fields.
func buildTriggerInput(req triggerAgentRequest, workspaceID, userID string) agent.TriggerAgentInput {
	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = agent.TriggerTypeManual
	}
	var triggeredBy *string
	if userID != "" {
		triggeredBy = &userID
	}
	return agent.TriggerAgentInput{
		AgentID:        req.AgentID,
		WorkspaceID:    workspaceID,
		TriggeredBy:    triggeredBy,
		TriggerType:    triggerType,
		TriggerContext: req.TriggerContext,
		Inputs:         req.Inputs,
	}
}

// TriggerAgent handles POST /api/v1/agents/trigger
// Traces: FR-230, FR-231
func (h *AgentHandler) TriggerAgent(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return
	}

	userID, _ := r.Context().Value(ctxkeys.UserID).(string)

	var req triggerAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return
	}

	if req.AgentID == "" {
		writeError(w, http.StatusBadRequest, "agent_id is required")
		return
	}

	run, err := h.orchestrator.TriggerAgent(r.Context(), buildTriggerInput(req, workspaceID, userID))
	if err != nil {
		h.handleTriggerError(w, err)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": agentRunToResponse(run)})
}

// GetAgentRun handles GET /api/v1/agents/runs/{id}
func (h *AgentHandler) GetAgentRun(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return
	}

	runID := chi.URLParam(r, paramID)
	if runID == "" {
		writeError(w, http.StatusBadRequest, "run id is required")
		return
	}

	run, err := h.orchestrator.GetAgentRun(r.Context(), workspaceID, runID)
	if err != nil {
		if errors.Is(err, agent.ErrAgentRunNotFound) {
			writeError(w, http.StatusNotFound, errAgentRunNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get agent run")
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": agentRunToResponse(run)})
}

// parsePageParams extracts limit and offset from query string with defaults.
func parsePageParams(r *http.Request) (limit, offset int64) {
	limit, offset = 25, 0
	if l, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64); err == nil && o > 0 {
		offset = o
	}
	return
}

// ListAgentRuns handles GET /api/v1/agents/runs
func (h *AgentHandler) ListAgentRuns(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return
	}

	limit, offset := parsePageParams(r)
	filters := parseRunFilters(r)

	runs, total, err := h.orchestrator.ListAgentRuns(r.Context(), workspaceID, agent.ListRunsInput{
		Limit:      limit,
		Offset:     offset,
		Status:     filters.status,
		EntityType: filters.entityType,
		EntityID:   filters.entityID,
		WorkflowID: filters.workflowID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list agent runs")
		return
	}

	out := make([]agentRunResponse, 0, len(runs))
	for _, run := range runs {
		out = append(out, agentRunToResponse(run))
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": out,
		"meta": map[string]any{"total": total, "limit": limit, "offset": offset},
	})
}

type runFilters struct {
	status     string
	entityType string
	entityID   string
	workflowID string
}

func parseRunFilters(r *http.Request) runFilters {
	query := r.URL.Query()
	return runFilters{
		status:     query.Get(queryStatus),
		entityType: query.Get(paramEntityType),
		entityID:   query.Get(paramEntityID),
		workflowID: query.Get(queryWorkflowID),
	}
}

// ListAgentDefinitions handles GET /api/v1/agents/definitions
func (h *AgentHandler) ListAgentDefinitions(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return
	}

	definitions, err := h.orchestrator.ListAgentDefinitions(r.Context(), workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list agent definitions")
		return
	}

	out := make([]agentDefinitionResponse, 0, len(definitions))
	for _, def := range definitions {
		out = append(out, agentDefinitionResponse{
			ID:           def.ID,
			WorkspaceID:  def.WorkspaceID,
			Name:         def.Name,
			Description:  def.Description,
			AgentType:    def.AgentType,
			Objective:    def.Objective,
			AllowedTools: def.AllowedTools,
			Status:       def.Status,
			CreatedAt:    def.CreatedAt.Format(http.TimeFormat),
			UpdatedAt:    def.UpdatedAt.Format(http.TimeFormat),
		})
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": out})
}

// CancelAgentRun handles POST /api/v1/agents/runs/{id}/cancel
func (h *AgentHandler) CancelAgentRun(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return
	}

	runID := chi.URLParam(r, paramID)
	if runID == "" {
		writeError(w, http.StatusBadRequest, "run id is required")
		return
	}

	run, err := h.orchestrator.UpdateAgentRunStatus(r.Context(), workspaceID, runID, agent.StatusFailed)
	if err != nil {
		if errors.Is(err, agent.ErrAgentRunNotFound) {
			writeError(w, http.StatusNotFound, errAgentRunNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to cancel agent run")
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": agentRunToResponse(run)})
}

// Helper functions

func agentRunToResponse(run *agent.Run) agentRunResponse {
	meta := agentExtractRunContextMetadata(run)
	resp := agentRunResponse{
		ID:                run.ID,
		WorkspaceID:       run.WorkspaceID,
		AgentDefinitionID: run.DefinitionID,
		TriggeredByUserID: run.TriggeredByUserID,
		TriggerType:       run.TriggerType,
		Status:            agent.PublicRunOutcome(run),
		RuntimeStatus:     run.Status,
		Inputs:            run.Inputs,
		Output:            run.Output,
		ToolCalls:         run.ToolCalls,
		ReasoningTrace:    run.ReasoningTrace,
		TotalTokens:       run.TotalTokens,
		TotalCost:         run.TotalCost,
		LatencyMs:         run.LatencyMs,
		TraceID:           run.TraceID,
		StartedAt:         run.StartedAt.Format(http.TimeFormat),
		CreatedAt:         run.CreatedAt.Format(http.TimeFormat),
	}
	if meta.workflowID != "" {
		resp.WorkflowID = &meta.workflowID
	}
	if meta.entityType != "" {
		resp.EntityType = &meta.entityType
	}
	if meta.entityID != "" {
		resp.EntityID = &meta.entityID
	}
	if meta.rejectionReason != "" {
		resp.RejectionReason = &meta.rejectionReason
	}
	if run.CompletedAt != nil {
		completedAt := run.CompletedAt.Format(http.TimeFormat)
		resp.CompletedAt = &completedAt
	}
	return resp
}

func agentExtractRunContextMetadata(run *agent.Run) struct {
	workflowID      string
	entityType      string
	entityID        string
	rejectionReason string
} {
	meta := struct {
		workflowID      string
		entityType      string
		entityID        string
		rejectionReason string
	}{}
	if run == nil {
		return meta
	}

	meta.workflowID = firstJSONStringFromRaw(run.Output, queryWorkflowID)
	if meta.workflowID == "" {
		meta.workflowID = firstJSONStringFromRaw(run.TriggerContext, queryWorkflowID)
	}
	meta.entityType = firstNonEmptyString(
		firstJSONStringFromRaw(run.Output, paramEntityType),
		firstJSONStringFromRaw(run.TriggerContext, paramEntityType),
		firstNestedEntityTypeFromRaw(run.TriggerContext),
	)
	meta.entityID = firstNonEmptyString(
		firstJSONStringFromRaw(run.Output, paramEntityID),
		firstJSONStringFromRaw(run.TriggerContext, paramEntityID),
		firstNestedEntityIDFromRaw(run.TriggerContext),
	)
	if run.Status == agent.StatusRejected {
		meta.rejectionReason = firstNonEmptyString(
			firstJSONStringFromRaw(run.Output, dispatchReasonKey),
			firstJSONStringFromRaw(run.Output, rejectionReasonKey),
		)
		if meta.rejectionReason == "" && run.AbstentionReason != nil {
			meta.rejectionReason = *run.AbstentionReason
		}
	}
	return meta
}

func firstJSONStringFromRaw(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	str, _ := data[key].(string)
	return str
}

func firstNestedEntityTypeFromRaw(raw json.RawMessage) string {
	for _, entityType := range []string{"account", "contact", "deal", "case", "lead"} {
		if firstNestedEntityIDForTypeFromRaw(raw, entityType) != "" {
			return entityType
		}
	}
	return ""
}

func firstNestedEntityIDFromRaw(raw json.RawMessage) string {
	for _, entityType := range []string{"account", "contact", "deal", "case", "lead"} {
		if entityID := firstNestedEntityIDForTypeFromRaw(raw, entityType); entityID != "" {
			return entityID
		}
	}
	return ""
}

func firstNestedEntityIDForTypeFromRaw(raw json.RawMessage, entityType string) string {
	if len(raw) == 0 {
		return ""
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return ""
	}
	nested, _ := data[entityType].(map[string]any)
	entityID, _ := nested["id"].(string)
	return entityID
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (h *AgentHandler) handleTriggerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, agent.ErrAgentNotFound):
		writeError(w, http.StatusNotFound, "agent definition not found")
	case errors.Is(err, agent.ErrAgentNotActive):
		writeError(w, http.StatusBadRequest, "agent is not active")
	case errors.Is(err, agent.ErrInvalidTriggerType):
		writeError(w, http.StatusBadRequest, "invalid trigger type")
	default:
		writeError(w, http.StatusInternalServerError, "failed to trigger agent")
	}
}

// SupportAgentHandler handles Support Agent specific endpoints
type SupportAgentHandler struct {
	supportAgent *agents.SupportAgent
}

// NewSupportAgentHandler creates a new SupportAgentHandler
func NewSupportAgentHandler(supportAgent *agents.SupportAgent) *SupportAgentHandler {
	return &SupportAgentHandler{supportAgent: supportAgent}
}

type supportAgentRequest struct {
	CaseID        string `json:"case_id"`
	CustomerQuery string `json:"customer_query"`
	Language      string `json:"language,omitempty"`
	Priority      string `json:"priority,omitempty"`
}

type prospectingAgentRequest struct {
	LeadID   string `json:"lead_id"`
	Language string `json:"language,omitempty"`
}

type kbAgentRequest struct {
	CaseID   string `json:"case_id"`
	Language string `json:"language,omitempty"`
}

type insightsAgentRequest struct {
	Query         string `json:"query"`
	DateFrom      string `json:"date_from,omitempty"`
	DateTo        string `json:"date_to,omitempty"`
	Language      string `json:"language,omitempty"`
	ShadowMode    bool   `json:"shadow_mode,omitempty"`
	ShadowAgentID string `json:"shadow_agent_id,omitempty"`
}

// buildSupportConfig validates and converts an HTTP request into a SupportAgentConfig.
// Returns ("", nil) on validation error after writing the HTTP error response.
func buildSupportConfig(w http.ResponseWriter, req supportAgentRequest, workspaceID string) (agents.SupportAgentConfig, bool) {
	if req.CaseID == "" {
		writeError(w, http.StatusBadRequest, "case_id is required")
		return agents.SupportAgentConfig{}, false
	}
	if req.CustomerQuery == "" {
		writeError(w, http.StatusBadRequest, "customer_query is required")
		return agents.SupportAgentConfig{}, false
	}
	return agents.SupportAgentConfig{
		WorkspaceID:   workspaceID,
		CaseID:        req.CaseID,
		CustomerQuery: req.CustomerQuery,
		Language:      req.Language,
		Priority:      req.Priority,
	}, true
}

// TriggerSupportAgent handles POST /api/v1/agents/support/trigger
// Traces: FR-230, FR-231
func (h *SupportAgentHandler) TriggerSupportAgent(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return
	}

	var req supportAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return
	}

	config, valid := buildSupportConfig(w, req, workspaceID)
	if !valid {
		return
	}

	run, err := h.supportAgent.Run(r.Context(), config)
	if err != nil {
		if errors.Is(err, agents.ErrCaseIDRequired) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to run support agent")
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": agentRunToResponse(run)})
}

// extractAgentContext pulls workspace and user IDs from the request context.
// Returns ok=false and writes an error response when workspace is missing.
func extractAgentContext(w http.ResponseWriter, r *http.Request) (workspaceID, userID string, ok bool) {
	wid, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || wid == "" {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return "", "", false
	}
	uid, _ := r.Context().Value(ctxkeys.UserID).(string)
	return wid, uid, true
}

// decodeAgentRequest decodes a JSON request body into dst.
// Returns false and writes a 400 error response on decode failure.
func decodeAgentRequest[T any](w http.ResponseWriter, r *http.Request, dst *T) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return false
	}
	return true
}

// writeAgentQueuedResponse writes a 201 Created JSON response for a queued agent run.
func writeAgentQueuedResponse(w http.ResponseWriter, runID, agentName string) {
	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"run_id": runID,
		"status": "queued",
		"agent":  agentName,
	})
}

// ProspectingAgentHandler handles Prospecting Agent specific endpoints.
// Task 4.5b — FR-231: Prospecting Agent trigger endpoint.
type ProspectingAgentHandler struct {
	prospectingAgent *agents.ProspectingAgent
}

// NewProspectingAgentHandler creates a new ProspectingAgentHandler.
func NewProspectingAgentHandler(prospectingAgent *agents.ProspectingAgent) *ProspectingAgentHandler {
	return &ProspectingAgentHandler{prospectingAgent: prospectingAgent}
}

// buildProspectingConfig validates and converts an HTTP request into a ProspectingAgentConfig.
func buildProspectingConfig(w http.ResponseWriter, req prospectingAgentRequest, workspaceID string) (agents.ProspectingAgentConfig, bool) {
	if req.LeadID == "" {
		writeError(w, http.StatusBadRequest, "lead_id is required")
		return agents.ProspectingAgentConfig{}, false
	}
	language := req.Language
	if language == "" {
		language = defaultAgentLanguage
	}
	return agents.ProspectingAgentConfig{
		WorkspaceID: workspaceID,
		LeadID:      req.LeadID,
		Language:    language,
	}, true
}

func withProspectingTriggeredBy(config agents.ProspectingAgentConfig, userID string) agents.ProspectingAgentConfig {
	if userID == "" {
		return config
	}
	config.TriggeredByUserID = &userID
	return config
}

// TriggerProspectingAgent handles POST /api/v1/agents/prospecting/trigger.
func (h *ProspectingAgentHandler) TriggerProspectingAgent(w http.ResponseWriter, r *http.Request) {
	config, ok := prepareTriggeredAgentConfig(w, r, buildProspectingConfig, withProspectingTriggeredBy)
	if !ok {
		return
	}
	runQueuedAgent(w, r, config, h.prospectingAgent.Run, handleProspectingRunError, "failed to run prospecting agent", "prospecting")
}

func handleProspectingRunError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, agents.ErrLeadIDRequired) {
		writeError(w, http.StatusBadRequest, err.Error())
		return true
	}
	if errors.Is(err, agents.ErrLeadNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return true
	}
	if errors.Is(err, agents.ErrProspectingDailyLeadLimitExceeded) || errors.Is(err, agents.ErrProspectingDailyCostLimitExceeded) {
		writeError(w, http.StatusTooManyRequests, err.Error())
		return true
	}
	return false
}

// KBAgentHandler handles KB Agent specific endpoints.
// Task 4.5c — FR-231: KB Agent trigger endpoint.
type KBAgentHandler struct {
	kbAgent *agents.KBAgent
}

// NewKBAgentHandler creates a new KBAgentHandler.
func NewKBAgentHandler(kbAgent *agents.KBAgent) *KBAgentHandler {
	return &KBAgentHandler{kbAgent: kbAgent}
}

func buildKBConfig(w http.ResponseWriter, req kbAgentRequest, workspaceID string) (agents.KBAgentConfig, bool) {
	if req.CaseID == "" {
		writeError(w, http.StatusBadRequest, "case_id is required")
		return agents.KBAgentConfig{}, false
	}
	language := req.Language
	if language == "" {
		language = defaultAgentLanguage
	}
	return agents.KBAgentConfig{
		WorkspaceID: workspaceID,
		CaseID:      req.CaseID,
		Language:    language,
	}, true
}

// Task 4.5c — withKBTriggeredBy propagates user for audit trail.
func withKBTriggeredBy(config agents.KBAgentConfig, userID string) agents.KBAgentConfig {
	if userID == "" {
		return config
	}
	config.TriggeredByUserID = &userID
	return config
}

// TriggerKBAgent handles POST /api/v1/agents/kb/trigger.
func (h *KBAgentHandler) TriggerKBAgent(w http.ResponseWriter, r *http.Request) {
	config, ok := prepareTriggeredAgentConfig(w, r, buildKBConfig, withKBTriggeredBy)
	if !ok {
		return
	}
	runQueuedAgent(w, r, config, h.kbAgent.Run, handleKBRunError, "failed to run kb agent", "kb")
}

func handleKBRunError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, agents.ErrKBCaseIDRequired) {
		writeError(w, http.StatusBadRequest, err.Error())
		return true
	}
	if errors.Is(err, agents.ErrCaseNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return true
	}
	if errors.Is(err, agents.ErrCaseNotResolved) {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return true
	}
	if errors.Is(err, agents.ErrKBDailyLimitExceeded) {
		writeError(w, http.StatusTooManyRequests, err.Error())
		return true
	}
	return false
}

// InsightsAgentHandler handles Insights Agent specific endpoints.
// Task 4.5d — FR-231: Insights Agent trigger endpoint.
type InsightsAgentHandler struct {
	insightsAgent *agents.InsightsAgent
	shadow        *insightsShadowExecutor
	db            *sql.DB
}

// NewInsightsAgentHandler creates a new InsightsAgentHandler.
func NewInsightsAgentHandler(insightsAgent *agents.InsightsAgent) *InsightsAgentHandler {
	return &InsightsAgentHandler{insightsAgent: insightsAgent}
}

// NewInsightsAgentHandlerWithShadow creates an InsightsAgentHandler with
// optional shadow-mode execution support for the declarative pilot.
func NewInsightsAgentHandlerWithShadow(
	insightsAgent *agents.InsightsAgent,
	shadowRunner *agent.DSLRunner,
	orchestrator *agent.Orchestrator,
	toolRegistry *tooldomain.ToolRegistry,
	groundsValidator *agent.GroundsValidator,
	db *sql.DB,
) *InsightsAgentHandler {
	return &InsightsAgentHandler{
		insightsAgent: insightsAgent,
		shadow:        newInsightsShadowExecutor(shadowRunner, orchestrator, toolRegistry, groundsValidator, db),
		db:            db,
	}
}

func buildInsightsConfig(w http.ResponseWriter, req insightsAgentRequest, workspaceID string) (agents.InsightsAgentConfig, bool) {
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, errQueryRequired)
		return agents.InsightsAgentConfig{}, false
	}
	language := req.Language
	if language == "" {
		language = defaultAgentLanguage
	}
	config := agents.InsightsAgentConfig{
		WorkspaceID: workspaceID,
		Query:       req.Query,
		Language:    language,
	}
	if req.DateFrom != "" {
		t, err := parseDateTimeValue(req.DateFrom)
		if err != nil {
			writeError(w, http.StatusBadRequest, "date_from must be RFC3339")
			return agents.InsightsAgentConfig{}, false
		}
		config.DateFrom = t
	}
	if req.DateTo != "" {
		t, err := parseDateTimeValue(req.DateTo)
		if err != nil {
			writeError(w, http.StatusBadRequest, "date_to must be RFC3339")
			return agents.InsightsAgentConfig{}, false
		}
		config.DateTo = t
	}
	return config, true
}

// Task 4.5d — withInsightsTriggeredBy propagates user for audit trail.
func withInsightsTriggeredBy(config agents.InsightsAgentConfig, userID string) agents.InsightsAgentConfig {
	if userID == "" {
		return config
	}
	config.TriggeredByUserID = &userID
	return config
}

func parseDateTimeValue(v string) (*time.Time, error) {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TriggerInsightsAgent handles POST /api/v1/agents/insights/trigger.
func (h *InsightsAgentHandler) TriggerInsightsAgent(w http.ResponseWriter, r *http.Request) {
	workspaceID, req, config, ok := h.prepareInsightsRequest(w, r)
	if !ok {
		return
	}
	rollout := loadInsightsRolloutConfig(r.Context(), h.db, workspaceID)
	if rollout.Enabled && rollout.DeclarativePrimary {
		h.triggerInsightsDeclarativePrimary(w, r, config, rollout)
		return
	}

	run, ok := h.runInsightsPrimary(w, r, config)
	if !ok {
		return
	}

	response := buildInsightsPrimaryResponse(run)
	shadow := buildInsightsShadowPayload(h, r, config, req, run)
	enrichInsightsPrimaryResponse(response, rollout, run, req.ShadowMode, shadow)

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *InsightsAgentHandler) prepareInsightsRequest(
	w http.ResponseWriter,
	r *http.Request,
) (string, insightsAgentRequest, agents.InsightsAgentConfig, bool) {
	workspaceID, userID, req, config, ok := prepareTriggeredAgentRequest(w, r, buildInsightsConfig)
	if !ok {
		return "", insightsAgentRequest{}, agents.InsightsAgentConfig{}, false
	}
	return workspaceID, req, withInsightsTriggeredBy(config, userID), true
}

func prepareTriggeredAgentConfig[Req any, Config any](
	w http.ResponseWriter,
	r *http.Request,
	build func(http.ResponseWriter, Req, string) (Config, bool),
	withTriggeredBy func(Config, string) Config,
) (Config, bool) {
	_, userID, _, config, ok := prepareTriggeredAgentRequest(w, r, build)
	if !ok {
		return config, false
	}
	return withTriggeredBy(config, userID), true
}

func prepareTriggeredAgentRequest[Req any, Config any](
	w http.ResponseWriter,
	r *http.Request,
	build func(http.ResponseWriter, Req, string) (Config, bool),
) (string, string, Req, Config, bool) {
	workspaceID, userID, ok := extractAgentContext(w, r)
	if !ok {
		var zeroReq Req
		var zeroConfig Config
		return "", "", zeroReq, zeroConfig, false
	}

	var req Req
	if !decodeAgentRequest(w, r, &req) {
		var zeroConfig Config
		return "", "", req, zeroConfig, false
	}

	config, valid := build(w, req, workspaceID)
	return workspaceID, userID, req, config, valid
}

func runQueuedAgent[Config any](
	w http.ResponseWriter,
	r *http.Request,
	config Config,
	run func(context.Context, Config) (*agent.Run, error),
	handleErr func(http.ResponseWriter, error) bool,
	internalMsg string,
	agentName string,
) {
	runResult, err := run(r.Context(), config)
	if err != nil {
		if handled := handleErr(w, err); handled {
			return
		}
		writeError(w, http.StatusInternalServerError, internalMsg)
		return
	}
	writeAgentQueuedResponse(w, runResult.ID, agentName)
}

func (h *InsightsAgentHandler) runInsightsPrimary(
	w http.ResponseWriter,
	r *http.Request,
	config agents.InsightsAgentConfig,
) (*agent.Run, bool) {
	run, err := h.insightsAgent.Run(r.Context(), config)
	if err == nil {
		return run, true
	}
	if handled := handleInsightsRunError(w, err); handled {
		return nil, false
	}
	writeError(w, http.StatusInternalServerError, "failed to run insights agent")
	return nil, false
}

func buildInsightsPrimaryResponse(run *agent.Run) map[string]any {
	return map[string]any{
		"run_id": run.ID,
		"status": "queued",
		"agent":  "insights",
	}
}

func buildInsightsShadowPayload(
	h *InsightsAgentHandler,
	r *http.Request,
	config agents.InsightsAgentConfig,
	req insightsAgentRequest,
	run *agent.Run,
) map[string]any {
	if !req.ShadowMode {
		return nil
	}
	return h.executeInsightsShadow(r.Context(), config, req.ShadowAgentID, run)
}

func enrichInsightsPrimaryResponse(
	response map[string]any,
	rollout insightsRolloutConfig,
	run *agent.Run,
	shadowMode bool,
	shadow map[string]any,
) {
	if rollout.Enabled {
		response["rollout"] = buildInsightsRolloutResponse(rollout, run, run)
	}
	if shadowMode {
		response["shadow"] = shadow
	}
}

func (h *InsightsAgentHandler) triggerInsightsDeclarativePrimary(
	w http.ResponseWriter,
	r *http.Request,
	config agents.InsightsAgentConfig,
	rollout insightsRolloutConfig,
) {
	if h == nil || h.shadow == nil {
		writeError(w, http.StatusInternalServerError, "declarative insights rollout is not configured")
		return
	}
	execution, err := h.shadow.ExecutePrimary(r.Context(), config, rollout.AgentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to run declarative insights workflow")
		return
	}
	run := execution.WrapperRun
	effective := execution.EffectiveRun
	if effective == nil {
		effective = run
	}
	response := map[string]any{
		"run_id":  run.ID,
		"status":  "queued",
		"agent":   "insights",
		"rollout": buildInsightsRolloutResponse(rollout, run, effective),
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func handleInsightsRunError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, agents.ErrInsightsQueryRequired) {
		writeError(w, http.StatusBadRequest, err.Error())
		return true
	}
	if errors.Is(err, agents.ErrInsightsDailyLimitExceeded) {
		writeError(w, http.StatusTooManyRequests, err.Error())
		return true
	}
	return false
}
