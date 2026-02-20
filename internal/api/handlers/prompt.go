// Task 3.9: Prompt Versioning
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/agent"
)

// PromptVersionService interface para inyección de dependencias
type PromptVersionService interface {
	CreatePromptVersion(ctx context.Context, input agent.CreatePromptVersionInput) (*agent.PromptVersion, error)
	GetActivePrompt(ctx context.Context, workspaceID, agentID string) (*agent.PromptVersion, error)
	ListPromptVersions(ctx context.Context, workspaceID, agentID string) ([]*agent.PromptVersion, error)
	GetPromptVersionByID(ctx context.Context, workspaceID, promptVersionID string) (*agent.PromptVersion, error)
	PromotePrompt(ctx context.Context, workspaceID, promptVersionID string) error
	RollbackPrompt(ctx context.Context, workspaceID, agentID string) error
}

// PromptHandler gestiona endpoints de prompts
type PromptHandler struct {
	service PromptVersionService
}

// NewPromptHandler crea un nuevo handler
func NewPromptHandler(service PromptVersionService) *PromptHandler {
	return &PromptHandler{service: service}
}

// CreatePromptVersionRequest es el body para crear versión
type CreatePromptVersionRequest struct {
	AgentDefinitionID  string  `json:"agent_definition_id"`
	SystemPrompt       string  `json:"system_prompt"`
	UserPromptTemplate *string `json:"user_prompt_template,omitempty"`
	Config             *string `json:"config,omitempty"`
}

// PromptVersionResponse representa una versión en respuesta
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

// List lista todas las versiones de un agente
// GET /api/v1/admin/prompts?agent_id={id}
func (h *PromptHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		http.Error(w, errMissingWorkspaceShort, http.StatusUnauthorized)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "missing agent_id query param", http.StatusBadRequest)
		return
	}

	versions, listErr := h.service.ListPromptVersions(r.Context(), workspaceID, agentID)
	if listErr != nil {
		http.Error(w, listErr.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"data": toPromptVersionResponses(versions),
	}
	w.Header().Set(headerContentType, mimeJSON)
	if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
		http.Error(w, errFailedToEncode, http.StatusInternalServerError)
	}
}

// Create crea una nueva versión de prompt
// POST /api/v1/admin/prompts
func (h *PromptHandler) Create(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
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

	resp := map[string]interface{}{
		"data": toPromptVersionResponse(pv),
	}
	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp) // header already sent, error unrecoverable
}

// decodeCreatePromptRequest decodifica y valida el body del request de creación
func decodeCreatePromptRequest(r *http.Request) (CreatePromptVersionRequest, error) {
	var req CreatePromptVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, fmt.Errorf("invalid request body")
	}
	if req.AgentDefinitionID == "" || req.SystemPrompt == "" {
		return req, fmt.Errorf("missing required fields")
	}
	return req, nil
}

// resolvePromptConfig retorna el config o '{}' si es nil
func resolvePromptConfig(config *string) string {
	if config == nil {
		return errEmptyJSON
	}
	return *config
}

// Promote activa una versión
// PUT /api/v1/admin/prompts/{id}/promote
func (h *PromptHandler) Promote(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := getWorkspaceIDFromContext(r)
	if !ok {
		http.Error(w, errMissingWorkspaceShort, http.StatusUnauthorized)
		return
	}

	promptVersionID, ok := getPromptVersionIDParam(w, r)
	if !ok {
		return
	}

	if !h.promotePromptVersion(w, r, workspaceID, promptVersionID) {
		return
	}

	h.respondWithPromptVersion(w, r, workspaceID, promptVersionID)
}

func getWorkspaceIDFromContext(r *http.Request) (string, bool) {
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

func (h *PromptHandler) promotePromptVersion(w http.ResponseWriter, r *http.Request, workspaceID, promptVersionID string) bool {
	promErr := h.service.PromotePrompt(r.Context(), workspaceID, promptVersionID)
	if promErr == nil {
		return true
	}
	writePromoteError(w, promErr)
	return false
}

func writePromoteError(w http.ResponseWriter, err error) {
	if isPromptNotFoundError(err) {
		http.Error(w, "prompt version not found", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func isPromptNotFoundError(err error) bool {
	if err == sql.ErrNoRows {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "no rows") || strings.Contains(msg, "not found")
}

func (h *PromptHandler) respondWithPromptVersion(w http.ResponseWriter, r *http.Request, workspaceID, promptVersionID string) {
	pv, getErr := h.service.GetPromptVersionByID(r.Context(), workspaceID, promptVersionID)
	if getErr != nil {
		http.Error(w, getErr.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"data": toPromptVersionResponse(pv),
	}
	w.Header().Set(headerContentType, mimeJSON)
	if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
		http.Error(w, errFailedToEncode, http.StatusInternalServerError)
	}
}

// Rollback reactiva la versión anterior
// PUT /api/v1/admin/prompts/{id}/rollback (nota: {id} aquí es agent_id, no prompt_version_id)
func (h *PromptHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		http.Error(w, errMissingWorkspaceShort, http.StatusUnauthorized)
		return
	}

	agentID := chi.URLParam(r, paramID)
	if agentID == "" {
		http.Error(w, "missing id param", http.StatusBadRequest)
		return
	}

	if rollErr := h.service.RollbackPrompt(r.Context(), workspaceID, agentID); rollErr != nil {
		writeRollbackError(w, rollErr)
		return
	}

	// Obtén la versión reactivada
	pv, getErr := h.service.GetActivePrompt(r.Context(), workspaceID, agentID)
	if getErr != nil {
		http.Error(w, getErr.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"data": toPromptVersionResponse(pv),
	}
	w.Header().Set(headerContentType, mimeJSON)
	if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
		http.Error(w, errFailedToEncode, http.StatusInternalServerError)
	}
}

// writeRollbackError escribe el error HTTP apropiado para un rollback fallido
func writeRollbackError(w http.ResponseWriter, err error) {
	if strings.Contains(err.Error(), "no archived prompt") {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// Helpers

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
	var responses []*PromptVersionResponse
	for _, pv := range pvs {
		responses = append(responses, toPromptVersionResponse(pv))
	}
	return responses
}
