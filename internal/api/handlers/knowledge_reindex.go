package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

// KnowledgeReindexHandler handles manual reindex operations.
type KnowledgeReindexHandler struct {
	reindexService *knowledge.ReindexService
}

func NewKnowledgeReindexHandler(svc *knowledge.ReindexService) *KnowledgeReindexHandler {
	return &KnowledgeReindexHandler{reindexService: svc}
}

type reindexRequest struct {
	EntityType *string `json:"entityType,omitempty"`
}

type reindexResponse struct {
	ItemsQueued   int    `json:"items_queued"`
	EstimatedTime string `json:"estimated_time"`
}

// Reindex handles POST /api/v1/knowledge/reindex.
func (h *KnowledgeReindexHandler) Reindex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wsID, wsErr := getWorkspaceID(ctx)
	if wsErr != nil {
		writeError(w, http.StatusUnauthorized, "missing workspace context")
		return
	}

	var req reindexRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	queued, queueErr := h.reindexService.QueueWorkspaceReindex(ctx, wsID, req.EntityType)
	if queueErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to queue reindex")
		return
	}

	estimated := time.Duration(queued) * 250 * time.Millisecond

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(reindexResponse{
		ItemsQueued:   queued,
		EstimatedTime: fmt.Sprintf("%ds", int(estimated.Seconds()+0.5)),
	}); encodeErr != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}
