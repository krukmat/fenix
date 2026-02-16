// Task 3.9: Prompt Versioning
package handlers

import (
	"context"
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
	AgentDefinitionID  string `json:"agent_definition_id"`
	SystemPrompt       string `json:"system_prompt"`
	UserPromptTemplate *string `json:"user_prompt_template,omitempty"`
	Config             *string `json:"config,omitempty"`
}

// PromptVersionResponse representa una versión en respuesta
type PromptVersionResponse struct {
	ID                 string            `json:"id"`
	AgentDefinitionID  string            `json:"agent_definition_id"`
	VersionNumber      int               `json:"version_number"`
	SystemPrompt       string            `json:"system_prompt"`
	UserPromptTemplate *string           `json:"user_prompt_template,omitempty"`
	Config             map[string]interface{} `json:"config,omitempty"`
	Status             string            `json:"status"`
	CreatedAt          string            `json:"created_at"`
}

// List lista todas las versiones de un agente
// GET /api/v1/admin/prompts?agent_id={id}
func (h *PromptHandler) List(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		http.Error(w, "missing workspace_id", http.StatusUnauthorized)
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

	resp := map[string]interface{}{
		"data": toPromptVersionResponses(versions),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// Create crea una nueva versión de prompt
// POST /api/v1/admin/prompts
func (h *PromptHandler) Create(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		http.Error(w, "missing workspace_id", http.StatusUnauthorized)
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
	w.Header().Set("Content-Type", "application/json")
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
		return "{}"
	}
	return *config
}

// Promote activa una versión
// PUT /api/v1/admin/prompts/{id}/promote
func (h *PromptHandler) Promote(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		http.Error(w, "missing workspace_id", http.StatusUnauthorized)
		return
	}

	promptVersionID := chi.URLParam(r, "id")
	if promptVersionID == "" {
		http.Error(w, "missing id param", http.StatusBadRequest)
		return
	}

	err := h.service.PromotePrompt(r.Context(), workspaceID, promptVersionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Obtén la versión actualizada
	pv, err := h.service.GetPromptVersionByID(r.Context(), workspaceID, promptVersionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"data": toPromptVersionResponse(pv),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// Rollback reactiva la versión anterior
// PUT /api/v1/admin/prompts/{id}/rollback (nota: {id} aquí es agent_id, no prompt_version_id)
func (h *PromptHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := r.Context().Value(ctxkeys.WorkspaceID).(string)
	if !ok || workspaceID == "" {
		http.Error(w, "missing workspace_id", http.StatusUnauthorized)
		return
	}

	agentID := chi.URLParam(r, "id")
	if agentID == "" {
		http.Error(w, "missing id param", http.StatusBadRequest)
		return
	}

	if err := h.service.RollbackPrompt(r.Context(), workspaceID, agentID); err != nil {
		writeRollbackError(w, err)
		return
	}

	// Obtén la versión reactivada
	pv, err := h.service.GetActivePrompt(r.Context(), workspaceID, agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"data": toPromptVersionResponse(pv),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
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
		CreatedAt:          pv.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toPromptVersionResponses(pvs []*agent.PromptVersion) []*PromptVersionResponse {
	var responses []*PromptVersionResponse
	for _, pv := range pvs {
		responses = append(responses, toPromptVersionResponse(pv))
	}
	return responses
}
