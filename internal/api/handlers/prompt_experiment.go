package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

type StartPromptExperimentRequest struct {
	ControlPromptVersionID   string `json:"control_prompt_version_id"`
	CandidatePromptVersionID string `json:"candidate_prompt_version_id"`
	ControlTrafficPercent    int    `json:"control_traffic_percent"`
	CandidateTrafficPercent  int    `json:"candidate_traffic_percent"`
}

type StopPromptExperimentRequest struct {
	WinnerPromptVersionID *string `json:"winner_prompt_version_id,omitempty"`
}

type PromptExperimentResponse struct {
	ID                       string  `json:"id"`
	AgentDefinitionID        string  `json:"agent_definition_id"`
	ControlPromptVersionID   string  `json:"control_prompt_version_id"`
	CandidatePromptVersionID string  `json:"candidate_prompt_version_id"`
	ControlTrafficPercent    int     `json:"control_traffic_percent"`
	CandidateTrafficPercent  int     `json:"candidate_traffic_percent"`
	Status                   string  `json:"status"`
	WinnerPromptVersionID    *string `json:"winner_prompt_version_id,omitempty"`
}

func (h *PromptHandler) ListExperiments(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.prompts.experiments.list") {
		return
	}
	workspaceID, ok := requirePromptWorkspaceID(r)
	if !ok {
		http.Error(w, errMissingWorkspaceShort, http.StatusUnauthorized)
		return
	}
	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "missing agent_id query param", http.StatusBadRequest)
		return
	}

	experiments, err := h.service.ListPromptExperiments(r.Context(), workspaceID, agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": toPromptExperimentResponses(experiments)})
}

func (h *PromptHandler) StartExperiment(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.prompts.experiments.create") {
		return
	}
	input, ok := buildStartExperimentInput(w, r)
	if !ok {
		return
	}
	experiment, err := h.service.StartPromptExperiment(r.Context(), input)
	if err != nil {
		writePromptExperimentError(w, err)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": toPromptExperimentResponse(experiment)})
}

func (h *PromptHandler) StopExperiment(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.prompts.experiments.stop") {
		return
	}
	input, ok := buildStopExperimentInput(w, r)
	if !ok {
		return
	}
	experiment, err := h.service.StopPromptExperiment(r.Context(), input)
	if err != nil {
		writePromptExperimentError(w, err)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": toPromptExperimentResponse(experiment)})
}

func buildStartExperimentInput(w http.ResponseWriter, r *http.Request) (agent.StartPromptExperimentInput, bool) {
	workspaceID, ok := requirePromptWorkspaceID(r)
	if !ok {
		http.Error(w, errMissingWorkspaceShort, http.StatusUnauthorized)
		return agent.StartPromptExperimentInput{}, false
	}
	req, err := decodeStartExperimentRequest(r)
	if err != nil {
		http.Error(w, errInvalidBody, http.StatusBadRequest)
		return agent.StartPromptExperimentInput{}, false
	}
	userID, _ := r.Context().Value(ctxkeys.UserID).(string)
	return agent.StartPromptExperimentInput{
		WorkspaceID:              workspaceID,
		ControlPromptVersionID:   req.ControlPromptVersionID,
		CandidatePromptVersionID: req.CandidatePromptVersionID,
		ControlTrafficPercent:    req.ControlTrafficPercent,
		CandidateTrafficPercent:  req.CandidateTrafficPercent,
		CreatedBy:                &userID,
	}, true
}

func decodeStartExperimentRequest(r *http.Request) (StartPromptExperimentRequest, error) {
	var req StartPromptExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return StartPromptExperimentRequest{}, err
	}
	return req, nil
}

func buildStopExperimentInput(w http.ResponseWriter, r *http.Request) (agent.StopPromptExperimentInput, bool) {
	workspaceID, ok := requirePromptWorkspaceID(r)
	if !ok {
		http.Error(w, errMissingWorkspaceShort, http.StatusUnauthorized)
		return agent.StopPromptExperimentInput{}, false
	}
	experimentID, ok := getPromptVersionIDParam(w, r)
	if !ok {
		return agent.StopPromptExperimentInput{}, false
	}
	req, err := decodeStopExperimentRequest(r)
	if err != nil {
		http.Error(w, errInvalidBody, http.StatusBadRequest)
		return agent.StopPromptExperimentInput{}, false
	}
	return agent.StopPromptExperimentInput{
		WorkspaceID:           workspaceID,
		ExperimentID:          experimentID,
		WinnerPromptVersionID: req.WinnerPromptVersionID,
	}, true
}

func decodeStopExperimentRequest(r *http.Request) (StopPromptExperimentRequest, error) {
	var req StopPromptExperimentRequest
	if r.Body == nil {
		return req, nil
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil && !errors.Is(err, io.EOF) {
		return StopPromptExperimentRequest{}, err
	}
	return req, nil
}

func writePromptExperimentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, agent.ErrPromptExperimentInvalidSplit),
		errors.Is(err, agent.ErrPromptExperimentSameVersion),
		errors.Is(err, agent.ErrPromptExperimentAgentMismatch):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, agent.ErrPromptExperimentAlreadyRunning):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, agent.ErrPromptExperimentNotFound), errors.Is(err, agent.ErrPromptVersionNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func toPromptExperimentResponse(experiment *agent.PromptExperiment) *PromptExperimentResponse {
	if experiment == nil {
		return nil
	}
	return &PromptExperimentResponse{
		ID:                       experiment.ID,
		AgentDefinitionID:        experiment.AgentDefinitionID,
		ControlPromptVersionID:   experiment.ControlPromptVersionID,
		CandidatePromptVersionID: experiment.CandidatePromptVersionID,
		ControlTrafficPercent:    experiment.ControlTrafficPercent,
		CandidateTrafficPercent:  experiment.CandidateTrafficPercent,
		Status:                   string(experiment.Status),
		WinnerPromptVersionID:    experiment.WinnerPromptVersionID,
	}
}

func toPromptExperimentResponses(experiments []*agent.PromptExperiment) []*PromptExperimentResponse {
	responses := make([]*PromptExperimentResponse, 0, len(experiments))
	for _, experiment := range experiments {
		responses = append(responses, toPromptExperimentResponse(experiment))
	}
	return responses
}
