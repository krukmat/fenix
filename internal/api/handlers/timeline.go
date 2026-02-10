package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matiasleandrokruk/fenix/internal/domain/crm"
)

type TimelineHandler struct{ service *crm.TimelineService }

func NewTimelineHandler(service *crm.TimelineService) *TimelineHandler {
	return &TimelineHandler{service: service}
}

func (h *TimelineHandler) ListTimeline(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	page := parsePaginationParams(r)
	items, total, err := h.service.List(r.Context(), wsID, crm.ListTimelineInput{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list timeline: %v", err))
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: total, Limit: page.Limit, Offset: page.Offset}})
}

func (h *TimelineHandler) ListTimelineByEntity(w http.ResponseWriter, r *http.Request) {
	wsID, err := getWorkspaceID(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing workspace_id in context")
		return
	}
	entityType := chi.URLParam(r, "entity_type")
	entityID := chi.URLParam(r, "entity_id")
	page := parsePaginationParams(r)
	items, err := h.service.ListByEntity(r.Context(), wsID, entityType, entityID, crm.ListTimelineInput{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list timeline by entity: %v", err))
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: len(items), Limit: page.Limit, Offset: page.Offset}})
}
