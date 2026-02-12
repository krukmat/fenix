// Task 2.5: Integration tests for SearchService (Hybrid Search BM25 + Vector + RRF).
// Uses real in-memory SQLite DB with all migrations applied.
// LLMProvider is a stub — no real Ollama required.
package knowledge

import (
	"context"
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

// errStubLLMFailed is used in tests that simulate LLM failures.
var errStubLLMFailed = errors.New("stub LLM unavailable")

// ingestAndEmbedDoc ingests a document and synchronously embeds its chunks.
// Uses IngestService + EmbedderService.EmbedChunks to ensure vec_embedding rows exist.
func ingestAndEmbedDoc(t *testing.T, ingest *IngestService, embedder *EmbedderService, wsID, title, content string) *KnowledgeItem {
	t.Helper()
	item, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       title,
		RawContent:  content,
	})
	if err != nil {
		t.Fatalf("ingest failed for %q: %v", title, err)
	}
	if err := embedder.EmbedChunks(context.Background(), item.ID, wsID); err != nil {
		t.Fatalf("EmbedChunks failed for %q: %v", title, err)
	}
	return item
}

// ============================================================================
// TestCosineSimilarity_Basic — unit test for the helper function
// ============================================================================

func TestCosineSimilarity_Basic(t *testing.T) {
	// Two identical vectors → similarity = 1.0
	a := []float32{1.0, 0.0, 0.0}
	b := []float32{1.0, 0.0, 0.0}
	got := cosineSimilarity(a, b)
	if got < 0.99 || got > 1.01 {
		t.Errorf("identical vectors: expected ~1.0, got %f", got)
	}

	// Orthogonal vectors → similarity = 0.0
	c := []float32{0.0, 1.0, 0.0}
	got = cosineSimilarity(a, c)
	if got > 0.01 {
		t.Errorf("orthogonal vectors: expected ~0.0, got %f", got)
	}

	// Zero vector → similarity = 0.0 (safe, no division by zero)
	zero := []float32{0.0, 0.0, 0.0}
	got = cosineSimilarity(a, zero)
	if got != 0.0 {
		t.Errorf("zero vector: expected 0.0, got %f", got)
	}
}

// ============================================================================
// TestDecodeEmbedding — unit test for JSON decode helper
// ============================================================================

func TestDecodeEmbedding_Valid(t *testing.T) {
	vec, err := decodeEmbedding("[0.1,0.2,0.3]")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(vec))
	}
	if vec[0] < 0.09 || vec[0] > 0.11 {
		t.Errorf("expected vec[0] ~0.1, got %f", vec[0])
	}
}

func TestDecodeEmbedding_Invalid(t *testing.T) {
	_, err := decodeEmbedding("not-json")
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

// ============================================================================
// TestRRFMerge — unit test for RRF ranking formula
// ============================================================================

func TestRRFMerge_RankingFormula(t *testing.T) {
	// Doc A appears in BM25 rank 1, not in vector
	// Doc B appears in BM25 rank 2, and vector rank 1
	// Doc C appears only in vector rank 2
	// Expected RRF order: B > A > C (B appears in both with high vector rank)

	bm25Results := []bm25Row{
		{id: "A", title: "Doc A", snippet: "snippet A", score: -1.0}, // rank 1
		{id: "B", title: "Doc B", snippet: "snippet B", score: -0.5}, // rank 2
	}
	vecResults := []vectorRow{
		{id: "chunk-B", knowledgeItemID: "B", similarity: 0.95}, // rank 1
		{id: "chunk-C", knowledgeItemID: "C", similarity: 0.80}, // rank 2
	}

	results := rrfMerge(bm25Results, vecResults, 10)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// B should be first (appears in both)
	if results[0].KnowledgeItemID != "B" {
		t.Errorf("expected B to be first (both methods), got %s", results[0].KnowledgeItemID)
	}

	// B's method should be hybrid
	if results[0].Method != EvidenceMethodHybrid {
		t.Errorf("expected method 'hybrid' for doc in both, got %s", results[0].Method)
	}

	// A and C should only have their single-method labels
	for _, r := range results[1:] {
		if r.KnowledgeItemID == "A" && r.Method != EvidenceMethodBM25 {
			t.Errorf("expected 'bm25' for doc A (only in BM25), got %s", r.Method)
		}
		if r.KnowledgeItemID == "C" && r.Method != EvidenceMethodVector {
			t.Errorf("expected 'vector' for doc C (only in vector), got %s", r.Method)
		}
	}
}

// ============================================================================
// Integration tests (real DB + real FTS5 + stub embedder)
// ============================================================================

func TestSearchService_BM25_ReturnsRelevantDocs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := newStubEmbedder(3)
	wsID := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)
	embedder := NewEmbedderService(db, stub)
	svc := NewSearchService(db, stub)

	ingestAndEmbedDoc(t, ingest, embedder, wsID, "Pricing Strategy", "our pricing discount policy for enterprise customers")
	ingestAndEmbedDoc(t, ingest, embedder, wsID, "Support Process", "how to handle customer support tickets efficiently")

	results, err := svc.HybridSearch(context.Background(), SearchInput{
		Query:       "pricing discount",
		WorkspaceID: wsID,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("HybridSearch failed: %v", err)
	}
	if len(results.Items) == 0 {
		t.Fatal("expected at least 1 result for 'pricing discount'")
	}

	// The pricing doc should appear in results
	found := false
	for _, r := range results.Items {
		if r.Title == "Pricing Strategy" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Pricing Strategy' doc in results, got: %+v", results.Items)
	}
}

func TestSearchService_Vector_ReturnsRelevantDocs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Use a stub that returns distinct vectors per chunk index
	stub := newStubEmbedder(4)
	wsID := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)
	embedder := NewEmbedderService(db, stub)
	svc := NewSearchService(db, stub)

	ingestAndEmbedDoc(t, ingest, embedder, wsID, "Vector Doc", "content for vector retrieval test")

	results, err := svc.HybridSearch(context.Background(), SearchInput{
		Query:       "vector retrieval",
		WorkspaceID: wsID,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("HybridSearch failed: %v", err)
	}
	// Stub returns non-zero vectors — vector search should find something
	if len(results.Items) == 0 {
		t.Fatal("expected at least 1 result for vector search")
	}
}

func TestSearchService_Hybrid_CombinesBothMethods(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := newStubEmbedder(3)
	wsID := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)
	embedder := NewEmbedderService(db, stub)
	svc := NewSearchService(db, stub)

	ingestAndEmbedDoc(t, ingest, embedder, wsID, "Hybrid Doc", "hybrid search combines keyword and semantic retrieval")

	results, err := svc.HybridSearch(context.Background(), SearchInput{
		Query:       "keyword semantic",
		WorkspaceID: wsID,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("HybridSearch failed: %v", err)
	}
	if len(results.Items) == 0 {
		t.Fatal("expected results for hybrid search")
	}

	// Verify query is echoed back
	if results.Query != "keyword semantic" {
		t.Errorf("expected Query to be 'keyword semantic', got %q", results.Query)
	}
}

func TestSearchService_WorkspaceIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := newStubEmbedder(3)
	wsA := createWorkspace(t, db)
	wsB := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)
	embedder := NewEmbedderService(db, stub)
	svc := NewSearchService(db, stub)

	// Ingest "secret" doc only in workspace B
	ingestAndEmbedDoc(t, ingest, embedder, wsB, "Secret Workspace B", "confidential secret data for workspace B only")

	// Search in workspace A — must not return workspace B docs
	results, err := svc.HybridSearch(context.Background(), SearchInput{
		Query:       "confidential secret",
		WorkspaceID: wsA,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("HybridSearch failed: %v", err)
	}
	for _, r := range results.Items {
		if r.Title == "Secret Workspace B" {
			t.Errorf("SECURITY VIOLATION: workspace A search returned workspace B doc %q", r.Title)
		}
	}
}

func TestSearchService_EmptyIndex_NoResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := newStubEmbedder(3)
	wsID := createWorkspace(t, db)
	svc := NewSearchService(db, stub)

	results, err := svc.HybridSearch(context.Background(), SearchInput{
		Query:       "anything",
		WorkspaceID: wsID,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("HybridSearch on empty index should not error: %v", err)
	}
	if len(results.Items) != 0 {
		t.Errorf("expected 0 results on empty index, got %d", len(results.Items))
	}
}

func TestSearchService_Limit_Respected(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := newStubEmbedder(3)
	wsID := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)
	embedder := NewEmbedderService(db, stub)
	svc := NewSearchService(db, stub)

	// Ingest 5 documents with same keyword
	for i := 0; i < 5; i++ {
		ingestAndEmbedDoc(t, ingest, embedder, wsID,
			"Document about retrieval",
			"retrieval augmented generation system for knowledge base",
		)
	}

	results, err := svc.HybridSearch(context.Background(), SearchInput{
		Query:       "retrieval",
		WorkspaceID: wsID,
		Limit:       2,
	})
	if err != nil {
		t.Fatalf("HybridSearch failed: %v", err)
	}
	if len(results.Items) > 2 {
		t.Errorf("expected at most 2 results (limit=2), got %d", len(results.Items))
	}
}

func TestSearchService_LLMEmbedFails_FallbackToBM25(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Embedder that succeeds for doc embedding but fails for query embedding
	callCount := 0
	stub := &stubEmbedder{
		embedFunc: func(_ context.Context, req llm.EmbedRequest) (*llm.EmbedResponse, error) {
			callCount++
			// First call (document ingestion/embedding): succeed
			if callCount <= 3 {
				vecs := make([][]float32, len(req.Texts))
				for i := range vecs {
					vecs[i] = []float32{0.1, 0.2, 0.3}
				}
				return &llm.EmbedResponse{Embeddings: vecs}, nil
			}
			// Subsequent calls (query embedding): fail
			return nil, errStubLLMFailed
		},
	}

	wsID := createWorkspace(t, db)
	bus := eventbus.New()
	ingest := NewIngestService(db, bus)
	embedder := NewEmbedderService(db, stub)
	svc := NewSearchService(db, stub)

	ingestAndEmbedDoc(t, ingest, embedder, wsID, "Fallback Doc", "pricing policy for customers")

	// Query embedding will fail — should degrade to BM25 only, not return error
	results, err := svc.HybridSearch(context.Background(), SearchInput{
		Query:       "pricing",
		WorkspaceID: wsID,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("HybridSearch should degrade gracefully on LLM failure, got error: %v", err)
	}
	// BM25 should still return the pricing doc
	if len(results.Items) == 0 {
		t.Error("expected BM25 fallback to return results even when vector search fails")
	}
}
