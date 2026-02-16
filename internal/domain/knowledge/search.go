// Package knowledge — Task 2.5: SearchService (Hybrid Search BM25 + Vector + RRF).
// Combines FTS5 BM25 keyword search with in-memory cosine vector similarity.
// Results are merged via Reciprocal Rank Fusion (RRF, k=60).
package knowledge

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
)

const (
	rrfK         = 60 // RRF constant — industry standard
	defaultLimit = 20 // default search result limit
	maxLimit     = 50 // maximum search result limit
)

// SearchInput carries parameters for a hybrid search query.
type SearchInput struct {
	Query       string
	WorkspaceID string
	Limit       int // 0 → defaultLimit, capped at maxLimit
}

// SearchResult is a single ranked result from hybrid search.
type SearchResult struct {
	KnowledgeItemID string
	Title           string
	Snippet         string
	Score           float64
	Method          EvidenceMethod // bm25, vector, or hybrid
}

// SearchResults is the response from HybridSearch.
type SearchResults struct {
	Items []SearchResult
	Query string
}

// SearchService implements hybrid search (Task 2.5).
type SearchService struct {
	db  *sql.DB
	q   *sqlcgen.Queries
	llm llm.LLMProvider
}

// NewSearchService creates a SearchService backed by the given DB and LLM provider.
func NewSearchService(db *sql.DB, provider llm.LLMProvider) *SearchService {
	return &SearchService{
		db:  db,
		q:   sqlcgen.New(db),
		llm: provider,
	}
}

// HybridSearch runs BM25 + vector search in parallel and merges results via RRF.
// BM25 (FTS5) and LLM.Embed() run concurrently to overlap Ollama RTT with DB query.
// Graceful degradation: if LLM.Embed() fails, returns BM25-only results without error.
// Task 2.5 audit: switched from sequential to parallel execution.
func (s *SearchService) HybridSearch(ctx context.Context, input SearchInput) (*SearchResults, error) {
	limit := resolveLimit(input.Limit)

	var (
		bm25Results []bm25Row
		vecResults  []vectorRow
		bm25Err     error
		mu          sync.Mutex
		wg          sync.WaitGroup
	)

	wg.Add(2)

	// Goroutine 1: BM25 search via FTS5 (always available, no LLM required)
	go func() {
		defer wg.Done()
		res, err := s.bm25Search(ctx, input.Query, input.WorkspaceID, limit)
		mu.Lock()
		bm25Results, bm25Err = res, err
		mu.Unlock()
	}()

	// Goroutine 2: vector search — degrade gracefully if LLM embed fails
	go func() {
		defer wg.Done()
		vecResults = s.vectorSearchWithFallback(ctx, input.Query, input.WorkspaceID, limit)
	}()

	wg.Wait()

	if bm25Err != nil {
		return nil, fmt.Errorf("search: bm25: %w", bm25Err)
	}

	items := rrfMerge(bm25Results, vecResults, limit)
	return &SearchResults{Items: items, Query: input.Query}, nil
}

// vectorSearchWithFallback embeds the query and runs vector search.
// Returns empty slice on LLM failure (caller falls back to BM25-only).
func (s *SearchService) vectorSearchWithFallback(ctx context.Context, query, wsID string, limit int) []vectorRow {
	resp, err := s.llm.Embed(ctx, llm.EmbedRequest{Texts: []string{query}})
	if err != nil || len(resp.Embeddings) == 0 {
		return nil // graceful degradation
	}
	results, err := s.vectorSearch(ctx, wsID, resp.Embeddings[0], limit)
	if err != nil {
		return nil // graceful degradation
	}
	return results
}

// bm25Row holds a single BM25 result from FTS5 search.
type bm25Row struct {
	id      string
	title   string
	snippet string
	score   float64 // FTS5 bm25() — negative values, lower = better
}

// bm25Search executes FTS5 MATCH and returns results ordered by BM25 score.
// Note: FTS5 bm25() returns negative values (lower = better match).
// Raw SQL used because sqlc does not support CREATE VIRTUAL TABLE fts5 syntax.
func (s *SearchService) bm25Search(ctx context.Context, query, wsID string, limit int) ([]bm25Row, error) {
	const ftsQuery = `
		SELECT ki.id, ki.title,
		       snippet(knowledge_item_fts, 2, '', '', '...', 32) AS snippet,
		       bm25(knowledge_item_fts) AS score
		FROM knowledge_item_fts
		JOIN knowledge_item ki ON ki.id = knowledge_item_fts.id
		WHERE knowledge_item_fts MATCH ?
		  AND knowledge_item_fts.workspace_id = ?
		  AND ki.deleted_at IS NULL
		ORDER BY bm25(knowledge_item_fts)
		LIMIT ?`

	rows, err := s.db.QueryContext(ctx, ftsQuery, query, wsID, limit)
	if err != nil {
		// FTS5 MATCH with invalid syntax returns an error — treat as no results
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	var results []bm25Row
	for rows.Next() {
		var r bm25Row
		if scanErr := rows.Scan(&r.id, &r.title, &r.snippet, &r.score); scanErr != nil {
			return nil, fmt.Errorf("bm25Search scan: %w", scanErr)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// vectorRow holds a single result from vector (cosine) search.
type vectorRow struct {
	id              string // embedding_document.id
	knowledgeItemID string
	similarity      float32 // cosine similarity [0, 1]
}

// vectorSearch fetches all embedded vectors for the workspace, computes cosine
// similarity in-memory, and returns the top-limit results.
func (s *SearchService) vectorSearch(ctx context.Context, wsID string, queryVec []float32, limit int) ([]vectorRow, error) {
	rows, err := s.q.GetAllEmbeddedVectorsByWorkspace(ctx, wsID)
	if err != nil {
		return nil, fmt.Errorf("vectorSearch fetch: %w", err)
	}

	type scoredRow struct {
		row        sqlcgen.GetAllEmbeddedVectorsByWorkspaceRow
		similarity float32
	}

	scored := make([]scoredRow, 0, len(rows))
	for _, row := range rows {
		vec, decodeErr := decodeEmbedding(row.Embedding)
		if decodeErr != nil {
			continue // skip malformed vectors
		}
		sim := cosineSimilarity(queryVec, vec)
		scored = append(scored, scoredRow{row: row, similarity: sim})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].similarity > scored[j].similarity
	})

	results := make([]vectorRow, 0, min(limit, len(scored)))
	for i := 0; i < len(scored) && i < limit; i++ {
		results = append(results, vectorRow{
			id:              scored[i].row.ID,
			knowledgeItemID: scored[i].row.KnowledgeItemID,
			similarity:      scored[i].similarity,
		})
	}
	return results, nil
}

// rrfMerge combines BM25 and vector results via Reciprocal Rank Fusion (k=60).
// Documents present in both lists get a higher combined score (hybrid method).
func rrfMerge(bm25Results []bm25Row, vecResults []vectorRow, limit int) []SearchResult {
	type docInfo struct {
		title   string
		snippet string
		method  EvidenceMethod
	}

	scores := make(map[string]float64)
	docs := make(map[string]docInfo)

	// BM25 ranks contribute to RRF score
	for rank, r := range bm25Results {
		scores[r.id] += 1.0 / float64(rrfK+rank+1)
		docs[r.id] = docInfo{title: r.title, snippet: r.snippet, method: EvidenceMethodBM25}
	}

	// Vector ranks contribute to RRF score (keyed by knowledge_item_id for dedup)
	for rank, r := range vecResults {
		scores[r.knowledgeItemID] += 1.0 / float64(rrfK+rank+1)
		if existing, ok := docs[r.knowledgeItemID]; ok {
			// already in BM25 → upgrade method to hybrid
			existing.method = EvidenceMethodHybrid
			docs[r.knowledgeItemID] = existing
		} else {
			docs[r.knowledgeItemID] = docInfo{method: EvidenceMethodVector}
		}
	}

	type ranked struct {
		id    string
		score float64
	}
	all := make([]ranked, 0, len(scores))
	for id, score := range scores {
		all = append(all, ranked{id: id, score: score})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].score > all[j].score })

	results := make([]SearchResult, 0, min(limit, len(all)))
	for i := 0; i < len(all) && i < limit; i++ {
		id := all[i].id
		info := docs[id]
		results = append(results, SearchResult{
			KnowledgeItemID: id,
			Title:           info.title,
			Snippet:         info.snippet,
			Score:           all[i].score,
			Method:          info.method,
		})
	}
	return results
}

// cosineSimilarity computes cosine similarity between two float32 vectors.
// Returns 0 if either vector has zero magnitude.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return float32(dot / denom)
}

// decodeEmbedding deserialises a JSON TEXT vector back to []float32.
// e.g. "[0.1,0.2,0.3]" → []float32{0.1, 0.2, 0.3}
func decodeEmbedding(jsonStr string) ([]float32, error) {
	var vec []float32
	if err := json.Unmarshal([]byte(jsonStr), &vec); err != nil {
		return nil, fmt.Errorf("decodeEmbedding: %w", err)
	}
	return vec, nil
}

// resolveLimit returns the effective limit, applying default and max caps.
func resolveLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}
