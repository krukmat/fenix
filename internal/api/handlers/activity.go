package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

type ActivityHandler struct{ service *crm.ActivityService }

func NewActivityHandler(service *crm.ActivityService) *ActivityHandler {
	return &ActivityHandler{service: service}
}

type CreateActivityRequest struct {
	ActivityType string `json:"activityType"`
	EntityType   string `json:"entityType"`
	EntityID     string `json:"entityId"`
	OwnerID      string `json:"ownerId"`
	AssignedTo   string `json:"assignedTo,omitempty"`
	Subject      string `json:"subject"`
	Body         string `json:"body,omitempty"`
	Status       string `json:"status,omitempty"`
	DueAt        string `json:"dueAt,omitempty"`
	CompletedAt  string `json:"completedAt,omitempty"`
	Metadata     string `json:"metadata,omitempty"`
}

type UpdateActivityRequest = CreateActivityRequest

func (h *ActivityHandler) CreateActivity(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	var req CreateActivityRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}
	if !isActivityRequestValid(req) {
		writeError(w, http.StatusBadRequest, "activityType, entityType, entityId, ownerId and subject are required")
		return
	}
	out, err := h.service.Create(r.Context(), crm.CreateActivityInput{
		WorkspaceID:  wsID,
		ActivityType: req.ActivityType,
		EntityType:   req.EntityType,
		EntityID:     req.EntityID,
		OwnerID:      req.OwnerID,
		AssignedTo:   req.AssignedTo,
		Subject:      req.Subject,
		Body:         req.Body,
		Status:       req.Status,
		DueAt:        req.DueAt,
		CompletedAt:  req.CompletedAt,
		Metadata:     req.Metadata,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create activity: %v", err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	if !writeJSONOr500(w, out) {
		return
	}
}

func (h *ActivityHandler) GetActivity(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	out, svcErr := h.service.Get(r.Context(), wsID, id)
	if handleGetError(w, svcErr, "activity not found", "failed to get activity: %v") {
		return
	}
	if !writeJSONOr500(w, out) {
		return
	}
}

func (h *ActivityHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	page := parsePaginationParams(r)
	items, total, svcErr := h.service.List(r.Context(), wsID, crm.ListActivitiesInput{Limit: page.Limit, Offset: page.Offset})
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list activities: %v", svcErr))
		return
	}
	if !writePaginatedOr500(w, items, total, page) {
		return
	}
}

func (h *ActivityHandler) UpdateActivity(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	existing, svcErr := h.service.Get(r.Context(), wsID, id)
	if handleGetError(w, svcErr, "activity not found", "failed to get activity: %v") {
		return
	}
	var req UpdateActivityRequest
	if !decodeBodyJSON(w, r, &req) {
		return
	}
	out, svcErr := h.service.Update(r.Context(), wsID, id, buildUpdateActivityInput(req, existing))
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update activity: %v", svcErr))
		return
	}
	_ = writeJSONOr500(w, out)
}

func (h *ActivityHandler) DeleteActivity(w http.ResponseWriter, r *http.Request) {
	wsID, ok := requireWorkspaceID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, paramID)
	if svcErr := h.service.Delete(r.Context(), wsID, id); svcErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete activity: %v", svcErr))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isActivityRequestValid checks required fields for CreateActivity.
// Task 1.6.15: Extracted to reduce cyclomatic complexity of CreateActivity (was 9).
func isActivityRequestValid(req CreateActivityRequest) bool {
	return req.ActivityType != "" && req.EntityType != "" && req.EntityID != "" &&
		req.OwnerID != "" && req.Subject != ""
}

// buildUpdateActivityInput merges update request with existing activity values.
// Task 1.6.15: Extracted to reduce cyclomatic complexity of UpdateActivity (was 11).
func buildUpdateActivityInput(req UpdateActivityRequest, existing *crm.Activity) crm.UpdateActivityInput {
	return crm.UpdateActivityInput{
		ActivityType: coalesce(req.ActivityType, existing.ActivityType),
		EntityType:   coalesce(req.EntityType, existing.EntityType),
		EntityID:     coalesce(req.EntityID, existing.EntityID),
		OwnerID:      coalesce(req.OwnerID, existing.OwnerID),
		AssignedTo:   req.AssignedTo,
		Subject:      coalesce(req.Subject, existing.Subject),
		Body:         req.Body,
		Status:       req.Status,
		DueAt:        req.DueAt,
		CompletedAt:  req.CompletedAt,
		Metadata:     req.Metadata,
	}
}
