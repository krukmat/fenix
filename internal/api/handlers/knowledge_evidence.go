// Task 2.6: HTTP handler for evidence pack building.
// POST /api/v1/knowledge/evidence — builds curated evidence from hybrid search.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

// KnowledgeEvidenceHandler handles evidence pack HTTP requests.
type KnowledgeEvidenceHandler struct {
	evidenceService *knowledge.EvidencePackService
}

// NewKnowledgeEvidenceHandler creates a KnowledgeEvidenceHandler.
func NewKnowledgeEvidenceHandler(svc *knowledge.EvidencePackService) *KnowledgeEvidenceHandler {
	return &KnowledgeEvidenceHandler{evidenceService: svc}
}

type evidenceRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type evidenceResponse struct {
	Data evidenceData `json:"data"`
}

type evidenceData struct {
	Sources         []evidenceSource `json:"sources"`
	Confidence      string           `json:"confidence"`
	TotalCandidates int              `json:"total_candidates"`
	FilteredCount   int              `json:"filtered_count"`
	Warnings        []string         `json:"warnings"`
}

type evidenceSource struct {
	KnowledgeItemID string   `json:"knowledge_item_id"`
	Method          string   `json:"method"`
	Score           float64  `json:"score"`
	Snippet         *string  `json:"snippet,omitempty"`
	PiiRedacted     bool     `json:"pii_redacted"`
	Metadata        *string  `json:"metadata,omitempty"`
	CreatedAt       string   `json:"created_at"`
}

// Build handles POST /api/v1/knowledge/evidence.
func (h *KnowledgeEvidenceHandler) Build(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wsID, wsErr := getWorkspaceID(ctx)
	if wsErr != nil {
		writeError(w, http.StatusUnauthorized, "missing workspace context")
		return
	}

	var req evidenceRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	pack, buildErr := h.evidenceService.BuildEvidencePack(ctx, knowledge.BuildEvidencePackInput{
		Query:       req.Query,
		WorkspaceID: wsID,
		Limit:       req.Limit,
	})
	if buildErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to build evidence pack")
		return
	}

	sources := make([]evidenceSource, len(pack.Sources))
	for i, src := range pack.Sources {
		sources[i] = evidenceSource{
			KnowledgeItemID: src.KnowledgeItemID,
			Method:          string(src.Method),
			Score:           src.Score,
			Snippet:         src.Snippet,
			PiiRedacted:     src.PiiRedacted,
			Metadata:        src.Metadata,
			CreatedAt:       src.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(evidenceResponse{
		Data: evidenceData{
			Sources:         sources,
			Confidence:      string(pack.Confidence),
			TotalCandidates: pack.TotalCandidates,
			FilteredCount:   pack.FilteredCount,
			Warnings:        pack.Warnings,
		},
	}); encodeErr != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}
