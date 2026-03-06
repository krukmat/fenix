package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

type ActionAuthorizer interface {
	CheckActionPermission(
		ctx context.Context,
		userID, resource, action string,
		attrs map[string]string,
	) (bool, error)
}

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
	if !h.checkAuthorization(w, r, "api", "admin.tools.list") {
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
	if !h.checkAuthorization(w, r, "api", "admin.tools.create") {
		return
	}

	workspaceID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}

	req, ok := decodeCreateToolRequest(w, r)
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

func decodeCreateToolRequest(w http.ResponseWriter, r *http.Request) (createToolRequest, bool) {
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

func (h *ToolHandler) checkAuthorization(w http.ResponseWriter, r *http.Request, resource, action string) bool {
	if h.authz == nil {
		return true
	}

	userID, ok := r.Context().Value(ctxkeys.UserID).(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user_id in context")
		return false
	}

	allowed, err := h.authz.CheckActionPermission(r.Context(), userID, resource, action, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "authorization failed")
		return false
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "forbidden")
		return false
	}

	return true
}
