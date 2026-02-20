package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent/agents"
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
	Inputs            json.RawMessage `json:"inputs,omitempty"`
	Output            json.RawMessage `json:"output,omitempty"`
	ToolCalls         json.RawMessage `json:"toolCalls,omitempty"`
	ReasoningTrace    json.RawMessage `json:"reasoningTrace,omitempty"`
	TotalTokens       *int64          `json:"totalTokens,omitempty"`
	TotalCost         *float64        `json:"totalCost,omitempty"`
	LatencyMs         *int64          `json:"latencyMs,omitempty"`
	TraceID           *string         `json:"traceId,omitempty"`
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

	runs, total, err := h.orchestrator.ListAgentRuns(r.Context(), workspaceID, limit, offset)
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
	resp := agentRunResponse{
		ID:                run.ID,
		WorkspaceID:       run.WorkspaceID,
		AgentDefinitionID: run.DefinitionID,
		TriggeredByUserID: run.TriggeredByUserID,
		TriggerType:       run.TriggerType,
		Status:            run.Status,
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
	if run.CompletedAt != nil {
		completedAt := run.CompletedAt.Format(http.TimeFormat)
		resp.CompletedAt = &completedAt
	}
	return resp
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
		language = "es"
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
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		writeError(w, http.StatusUnauthorized, errMissingWorkspaceContext)
		return
	}
	userID, _ := r.Context().Value(ctxkeys.UserID).(string)

	var req prospectingAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return
	}

	config, valid := buildProspectingConfig(w, req, workspaceID)
	if !valid {
		return
	}
	config = withProspectingTriggeredBy(config, userID)

	run, err := h.prospectingAgent.Run(r.Context(), config)
	if err != nil {
		if handled := handleProspectingRunError(w, err); handled {
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to run prospecting agent")
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"run_id": run.ID,
		"status": "queued",
		"agent":  "prospecting",
	})
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
