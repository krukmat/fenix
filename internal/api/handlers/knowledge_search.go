// Task 2.5: HTTP handler for hybrid knowledge search.
// POST /api/v1/knowledge/search — runs BM25 + vector search and returns ranked results.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/matiasleandrokruk/fenix/internal/domain/knowledge"
)

// KnowledgeSearchHandler handles knowledge search HTTP requests (Task 2.5).
type KnowledgeSearchHandler struct {
	searchService *knowledge.SearchService
}

// NewKnowledgeSearchHandler creates a KnowledgeSearchHandler.
func NewKnowledgeSearchHandler(svc *knowledge.SearchService) *KnowledgeSearchHandler {
	return &KnowledgeSearchHandler{searchService: svc}
}

// searchRequest is the JSON request body for POST /api/v1/knowledge/search.
type searchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// searchResultItem is a single item in the search response.
type searchResultItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Score   float64 `json:"score"`
	Method  string `json:"method"`
}

// searchResponse is the JSON response body for POST /api/v1/knowledge/search.
type searchResponse struct {
	Results []searchResultItem `json:"results"`
	Query   string             `json:"query"`
}

// Search handles POST /api/v1/knowledge/search.
func (h *KnowledgeSearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wsID, wsErr := getWorkspaceID(ctx)
	if wsErr != nil {
		writeError(w, http.StatusUnauthorized, "missing workspace context")
		return
	}

	var req searchRequest
	if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	results, searchErr := h.searchService.HybridSearch(ctx, knowledge.SearchInput{
		Query:       req.Query,
		WorkspaceID: wsID,
		Limit:       req.Limit,
	})
	if searchErr != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	items := make([]searchResultItem, len(results.Items))
	for i, r := range results.Items {
		items[i] = searchResultItem{
			ID:      r.KnowledgeItemID,
			Title:   r.Title,
			Snippet: r.Snippet,
			Score:   r.Score,
			Method:  string(r.Method),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(searchResponse{
		Results: items,
		Query:   results.Query,
	}); encodeErr != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}
