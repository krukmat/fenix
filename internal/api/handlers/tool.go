package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

type ToolHandler struct {
	registry *tool.ToolRegistry
}

func NewToolHandler(registry *tool.ToolRegistry) *ToolHandler {
	return &ToolHandler{registry: registry}
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": out, "meta": map[string]int{"total": len(out)}})
}

func (h *ToolHandler) CreateTool(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}

	var req createToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	var createdBy *string
	if userID, ok := r.Context().Value(ctxkeys.UserID).(string); ok && userID != "" {
		createdBy = &userID
	}

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toToolResponse(item))
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
