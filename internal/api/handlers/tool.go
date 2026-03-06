package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

type ToolHandler struct {
	registry *tool.ToolRegistry
	authz    ActionAuthorizer
}

func NewToolHandler(registry *tool.ToolRegistry) *ToolHandler {
	return &ToolHandler{registry: registry}
}

func NewToolHandlerWithAuthorizer(registry *tool.ToolRegistry, authz ActionAuthorizer) *ToolHandler {
	return &ToolHandler{registry: registry, authz: authz}
}

type createToolRequest struct {
	Name                string          `json:"name"`
	Description         *string         `json:"description,omitempty"`
	InputSchema         json.RawMessage `json:"inputSchema"`
	RequiredPermissions []string        `json:"requiredPermissions,omitempty"`
}

type toolResponse struct {
	ID                  string          `json:"id"`
	WorkspaceID         string          `json:"workspaceId"`
	Name                string          `json:"name"`
	Description         *string         `json:"description,omitempty"`
	InputSchema         json.RawMessage `json:"inputSchema"`
	RequiredPermissions []string        `json:"requiredPermissions"`
	IsActive            bool            `json:"isActive"`
	CreatedAt           string          `json:"createdAt"`
	UpdatedAt           string          `json:"updatedAt"`
}

func (h *ToolHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.tools.list") {
		return
	}

	workspaceID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	items, err := h.registry.ListToolDefinitions(r.Context(), workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tools")
		return
	}

	out := make([]toolResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toToolResponse(item))
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": out, "meta": map[string]int{"total": len(out)}})
}

func (h *ToolHandler) CreateTool(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.tools.create") {
		return
	}

	workspaceID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}

	req, ok := decodeToolRequest(w, r)
	if !ok {
		return
	}
	createdBy := getCreatedByFromContext(r.Context())

	item, err := h.registry.CreateToolDefinition(r.Context(), tool.CreateToolDefinitionInput{
		WorkspaceID:         workspaceID,
		Name:                req.Name,
		Description:         req.Description,
		InputSchema:         req.InputSchema,
		RequiredPermissions: req.RequiredPermissions,
		CreatedBy:           createdBy,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toToolResponse(item))
}

func (h *ToolHandler) UpdateTool(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.tools.update") {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, paramID)
	if id == "" {
		writeError(w, http.StatusBadRequest, errToolIDRequired)
		return
	}

	req, ok := decodeToolRequest(w, r)
	if !ok {
		return
	}

	item, err := h.registry.UpdateToolDefinition(r.Context(), tool.UpdateToolDefinitionInput{
		ID:                  id,
		WorkspaceID:         workspaceID,
		Name:                req.Name,
		Description:         req.Description,
		InputSchema:         req.InputSchema,
		RequiredPermissions: req.RequiredPermissions,
	})
	if err != nil {
		writeToolError(w, err)
		return
	}

	writeJSONOr500(w, toToolResponse(item))
}

func (h *ToolHandler) ActivateTool(w http.ResponseWriter, r *http.Request) {
	h.setToolActive(w, r, true, "admin.tools.activate")
}

func (h *ToolHandler) DeactivateTool(w http.ResponseWriter, r *http.Request) {
	h.setToolActive(w, r, false, "admin.tools.deactivate")
}

func (h *ToolHandler) DeleteTool(w http.ResponseWriter, r *http.Request) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, "admin.tools.delete") {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, paramID)
	if id == "" {
		writeError(w, http.StatusBadRequest, errToolIDRequired)
		return
	}

	if err := h.registry.DeleteToolDefinition(r.Context(), workspaceID, id); err != nil {
		writeToolError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ToolHandler) setToolActive(w http.ResponseWriter, r *http.Request, isActive bool, action string) {
	if !checkActionAuthorization(w, r, h.authz, resourceAPI, action) {
		return
	}

	workspaceID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}

	id := chi.URLParam(r, paramID)
	if id == "" {
		writeError(w, http.StatusBadRequest, errToolIDRequired)
		return
	}

	item, err := h.registry.SetToolDefinitionActive(r.Context(), workspaceID, id, isActive)
	if err != nil {
		writeToolError(w, err)
		return
	}

	writeJSONOr500(w, toToolResponse(item))
}

func decodeToolRequest(w http.ResponseWriter, r *http.Request) (createToolRequest, bool) {
	var req createToolRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, errInvalidBody)
		return createToolRequest{}, false
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return createToolRequest{}, false
	}
	return req, true
}

func writeToolError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		return
	case err == tool.ErrToolDefinitionNotFound:
		writeError(w, http.StatusNotFound, "tool definition not found")
	case err == tool.ErrToolDefinitionInvalid:
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func getCreatedByFromContext(ctx context.Context) *string {
	if userID, ok := ctx.Value(ctxkeys.UserID).(string); ok && userID != "" {
		return &userID
	}
	return nil
}

func toToolResponse(item *tool.ToolDefinition) toolResponse {
	return toolResponse{
		ID:                  item.ID,
		WorkspaceID:         item.WorkspaceID,
		Name:                item.Name,
		Description:         item.Description,
		InputSchema:         item.InputSchema,
		RequiredPermissions: item.RequiredPermissions,
		IsActive:            item.IsActive,
		CreatedAt:           item.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:           item.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
