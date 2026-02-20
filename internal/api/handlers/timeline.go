package handlers

import (
	"context"
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
	handleListWithPagination(w, r, "failed to list timeline: %v", func(ctx context.Context, wsID string, limit, offset int) ([]*crm.TimelineEvent, int, error) {
		return h.service.List(ctx, wsID, crm.ListTimelineInput{Limit: limit, Offset: offset})
	})
}

func (h *TimelineHandler) ListTimelineByEntity(w http.ResponseWriter, r *http.Request) {
	wsID, wsErr := getWorkspaceID(r.Context())
	if wsErr != nil {
		writeError(w, http.StatusBadRequest, errMissingWorkspaceID)
		return
	}
	entityType := chi.URLParam(r, paramEntityType)
	entityID := chi.URLParam(r, paramEntityID)
	page := parsePaginationParams(r)
	items, listErr := h.service.ListByEntity(r.Context(), wsID, entityType, entityID, crm.ListTimelineInput{Limit: page.Limit, Offset: page.Offset})
	if listErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list timeline by entity: %v", listErr))
		return
	}
	w.Header().Set(headerContentType, mimeJSON)
	if encodeErr := json.NewEncoder(w).Encode(map[string]any{"data": items, "meta": Meta{Total: len(items), Limit: page.Limit, Offset: page.Offset}}); encodeErr != nil {
		writeError(w, http.StatusInternalServerError, errFailedToEncode)
		return
	}
}
