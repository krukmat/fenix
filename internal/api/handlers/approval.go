package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/api/ctxkeys"
	"github.com/matiasleandrokruk/fenix/internal/domain/policy"
)

type ApprovalHandler struct {
	service *policy.ApprovalService
}

func NewApprovalHandler(service *policy.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{service: service}
}

type decideApprovalRequest struct {
	Decision string `json:"decision"`
}

type approvalResponse struct {
	ID           string  `json:"id"`
	WorkspaceID  string  `json:"workspaceId"`
	RequestedBy  string  `json:"requestedBy"`
	ApproverID   string  `json:"approverId"`
	Action       string  `json:"action"`
	ResourceType *string `json:"resourceType,omitempty"`
	ResourceID   *string `json:"resourceId,omitempty"`
	Status       string  `json:"status"`
	ExpiresAt    string  `json:"expiresAt"`
	CreatedAt    string  `json:"createdAt"`
}

func (h *ApprovalHandler) ListPendingApprovals(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxkeys.UserID).(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return
	}

	items, err := h.service.GetPendingApprovals(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list approvals")
		return
	}

	out := make([]approvalResponse, 0, len(items))
	for _, item := range items {
		out = append(out, approvalResponse{
			ID:           item.ID,
			WorkspaceID:  item.WorkspaceID,
			RequestedBy:  item.RequestedBy,
			ApproverID:   item.ApproverID,
			Action:       item.Action,
			ResourceType: item.ResourceType,
			ResourceID:   item.ResourceID,
			Status:       string(item.Status),
			ExpiresAt:    item.ExpiresAt.Format(time.RFC3339),
			CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": out, "meta": map[string]int{"total": len(out)}})
}

func (h *ApprovalHandler) DecideApproval(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(ctxkeys.UserID).(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "approval id is required")
		return
	}

	var req decideApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.DecideApprovalRequest(r.Context(), id, req.Decision, userID); err != nil {
		h.handleDecisionError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ApprovalHandler) handleDecisionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, policy.ErrInvalidDecision):
		writeError(w, http.StatusBadRequest, "invalid decision")
	case errors.Is(err, policy.ErrApprovalNotFound):
		writeError(w, http.StatusNotFound, "approval request not found")
	case errors.Is(err, policy.ErrApprovalForbidden):
		writeError(w, http.StatusForbidden, "approval request is not assigned to current user")
	case errors.Is(err, policy.ErrApprovalExpired):
		writeError(w, http.StatusConflict, "approval request is expired")
	case errors.Is(err, policy.ErrApprovalAlreadyClosed):
		writeError(w, http.StatusConflict, "approval request is already decided")
	default:
		writeError(w, http.StatusInternalServerError, "failed to decide approval request")
	}
}
