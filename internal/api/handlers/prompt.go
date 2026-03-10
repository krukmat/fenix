package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

// PromptVersionService covers prompt version lifecycle operations.
type PromptVersionService interface {
	CreatePromptVersion(ctx context.Context, input agent.CreatePromptVersionInput) (*agent.PromptVersion, error)
	GetActivePrompt(ctx context.Context, workspaceID, agentID string) (*agent.PromptVersion, error)
	ListPromptVersions(ctx context.Context, workspaceID, agentID string) ([]*agent.PromptVersion, error)
	GetPromptVersionByID(ctx context.Context, workspaceID, promptVersionID string) (*agent.PromptVersion, error)
	PromotePrompt(ctx context.Context, workspaceID, promptVersionID string) error
	RollbackPrompt(ctx context.Context, workspaceID, promptVersionID string) error
}

// PromptExperimentService covers A/B experiment operations (ISP: separated from version lifecycle).
type PromptExperimentService interface {
	StartPromptExperiment(ctx context.Context, input agent.StartPromptExperimentInput) (*agent.PromptExperiment, error)
	ListPromptExperiments(ctx context.Context, workspaceID, agentID string) ([]*agent.PromptExperiment, error)
	StopPromptExperiment(ctx context.Context, input agent.StopPromptExperimentInput) (*agent.PromptExperiment, error)
}

type PromptHandler struct {
	service     PromptVersionService
	experiments PromptExperimentService
	authz       ActionAuthorizer
}

func NewPromptHandler(service PromptVersionService, experiments PromptExperimentService) *PromptHandler {
	return &PromptHandler{service: service, experiments: experiments}
}

func NewPromptHandlerWithAuthorizer(service PromptVersionService, experiments PromptExperimentService, authz ActionAuthorizer) *PromptHandler {
	return &PromptHandler{service: service, experiments: experiments, authz: authz}
}

type CreatePromptVersionRequest struct {
	AgentDefinitionID  string  `json:"agent_definition_id"`
	SystemPrompt       string  `json:"system_prompt"`
	UserPromptTemplate *string `json:"user_prompt_template,omitempty"`
	Config             *string `json:"config,omitempty"`
}

type PromptVersionResponse struct {
	ID                 string                 `json:"id"`
	AgentDefinitionID  string                 `json:"agent_definition_id"`
	VersionNumber      int                    `json:"version_number"`
	SystemPrompt       string                 `json:"system_prompt"`
	UserPromptTemplate *string                `json:"user_prompt_template,omitempty"`
	Config             map[string]interface{} `json:"config,omitempty"`
	Status             string                 `json:"status"`
	CreatedAt          string                 `json:"created_at"`
}

func (h *PromptHandler) List(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.prompts.list") {
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

	versions, err := h.service.ListPromptVersions(r.Context(), workspaceID, agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	if encodeErr := json.NewEncoder(w).Encode(map[string]any{"data": toPromptVersionResponses(versions)}); encodeErr != nil {
		http.Error(w, errFailedToEncode, http.StatusInternalServerError)
	}
}

func (h *PromptHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.prompts.create") {
		return
	}

	workspaceID, ok := requirePromptWorkspaceID(r)
	if !ok {
		http.Error(w, errMissingWorkspaceShort, http.StatusUnauthorized)
		return
	}
	userID, _ := r.Context().Value(ctxkeys.UserID).(string)

	req, err := decodeCreatePromptRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pv, err := h.service.CreatePromptVersion(r.Context(), agent.CreatePromptVersionInput{
		WorkspaceID:        workspaceID,
		AgentDefinitionID:  req.AgentDefinitionID,
		SystemPrompt:       req.SystemPrompt,
		UserPromptTemplate: req.UserPromptTemplate,
		Config:             resolvePromptConfig(req.Config),
		CreatedBy:          &userID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": toPromptVersionResponse(pv)})
}

func decodeCreatePromptRequest(r *http.Request) (CreatePromptVersionRequest, error) {
	var req CreatePromptVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, errors.New(errInvalidBody)
	}
	if req.AgentDefinitionID == "" || req.SystemPrompt == "" {
		return req, fmt.Errorf("missing required fields")
	}
	return req, nil
}

func resolvePromptConfig(config *string) string {
	if config == nil {
		return errEmptyJSON
	}
	return *config
}

func (h *PromptHandler) Promote(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.prompts.promote") {
		return
	}

	workspaceID, ok := requirePromptWorkspaceID(r)
	if !ok {
		http.Error(w, errMissingWorkspaceShort, http.StatusUnauthorized)
		return
	}
	promptVersionID, ok := getPromptVersionIDParam(w, r)
	if !ok {
		return
	}

	err := h.service.PromotePrompt(r.Context(), workspaceID, promptVersionID)
	if err != nil {
		writePromoteError(w, err)
		return
	}
	h.respondWithPromptVersion(w, r, workspaceID, promptVersionID)
}

func requirePromptWorkspaceID(r *http.Request) (string, bool) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		return "", false
	}
	return workspaceID, true
}

func getPromptVersionIDParam(w http.ResponseWriter, r *http.Request) (string, bool) {
	promptVersionID := chi.URLParam(r, paramID)
	if promptVersionID == "" {
		http.Error(w, "missing id param", http.StatusBadRequest)
		return "", false
	}
	return promptVersionID, true
}

func writePromoteError(w http.ResponseWriter, err error) {
	switch {
	case isPromptNotFoundError(err):
		http.Error(w, "prompt version not found", http.StatusNotFound)
	case errors.Is(err, agent.ErrPromptPromotionEvalMissing), errors.Is(err, agent.ErrPromptPromotionEvalFailed):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func isPromptNotFoundError(err error) bool {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, agent.ErrPromptVersionNotFound) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "no rows") || strings.Contains(msg, "not found")
}

func (h *PromptHandler) respondWithPromptVersion(w http.ResponseWriter, r *http.Request, workspaceID, promptVersionID string) {
	pv, err := h.service.GetPromptVersionByID(r.Context(), workspaceID, promptVersionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	if encodeErr := json.NewEncoder(w).Encode(map[string]any{"data": toPromptVersionResponse(pv)}); encodeErr != nil {
		http.Error(w, errFailedToEncode, http.StatusInternalServerError)
	}
}

func (h *PromptHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.prompts.rollback") {
		return
	}

	workspaceID, ok := requirePromptWorkspaceID(r)
	if !ok {
		http.Error(w, errMissingWorkspaceShort, http.StatusUnauthorized)
		return
	}
	promptVersionID, ok := getPromptVersionIDParam(w, r)
	if !ok {
		return
	}

	err := h.service.RollbackPrompt(r.Context(), workspaceID, promptVersionID)
	if err != nil {
		writeRollbackError(w, err)
		return
	}
	h.respondWithPromptVersion(w, r, workspaceID, promptVersionID)
}

func writeRollbackError(w http.ResponseWriter, err error) {
	if errors.Is(err, agent.ErrPromptRollbackInvalid) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if isPromptNotFoundError(err) {
		http.Error(w, "prompt version not found", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func toPromptVersionResponse(pv *agent.PromptVersion) *PromptVersionResponse {
	if pv == nil {
		return nil
	}

	config := make(map[string]interface{})
	if pv.Config.Temperature > 0 {
		config["temperature"] = pv.Config.Temperature
	}
	if pv.Config.MaxTokens > 0 {
		config["max_tokens"] = pv.Config.MaxTokens
	}

	return &PromptVersionResponse{
		ID:                 pv.ID,
		AgentDefinitionID:  pv.AgentDefinitionID,
		VersionNumber:      pv.VersionNumber,
		SystemPrompt:       pv.SystemPrompt,
		UserPromptTemplate: pv.UserPromptTemplate,
		Config:             config,
		Status:             string(pv.Status),
		CreatedAt:          pv.CreatedAt.Format(timeFormatISO),
	}
}

func toPromptVersionResponses(pvs []*agent.PromptVersion) []*PromptVersionResponse {
	responses := make([]*PromptVersionResponse, 0, len(pvs))
	for _, pv := range pvs {
		responses = append(responses, toPromptVersionResponse(pv))
	}
	return responses
}
