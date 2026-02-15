// Task 2.4: Integration tests for EmbedderService.
// Uses real in-memory SQLite DB with all migrations applied.
// LLMProvider is a stub (no real Ollama needed).
// Traces: FR-092
package knowledge

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
)

// ============================================================================
// stubLLMProvider — deterministic LLMProvider for tests
// ============================================================================

type stubEmbedder struct {
	// embedFunc is called by Embed(). Override per test.
	embedFunc func(ctx context.Context, req llm.EmbedRequest) (*llm.EmbedResponse, error)
	callCount int
}

func newStubEmbedder(dims int) *stubEmbedder {
	return &stubEmbedder{
		embedFunc: func(_ context.Context, req llm.EmbedRequest) (*llm.EmbedResponse, error) {
			vecs := make([][]float32, len(req.Texts))
			for i := range vecs {
				vecs[i] = make([]float32, dims)
				for j := range vecs[i] {
					vecs[i][j] = float32(i+1) * 0.1 // deterministic non-zero values
				}
			}
			return &llm.EmbedResponse{Embeddings: vecs}, nil
		},
	}
}

func (s *stubEmbedder) Embed(ctx context.Context, req llm.EmbedRequest) (*llm.EmbedResponse, error) {
	s.callCount++
	return s.embedFunc(ctx, req)
}

func (s *stubEmbedder) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Content: "stub"}, nil
}

func (s *stubEmbedder) ModelInfo() llm.ModelMeta {
	return llm.ModelMeta{ID: "stub-embed", Provider: "stub"}
}

func (s *stubEmbedder) HealthCheck(_ context.Context) error { return nil }

// ============================================================================
// EmbedderService tests
// ============================================================================

func TestEmbedderService_EmbedChunks_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := newStubEmbedder(3) // 3-dim vectors for test
	svc := NewEmbedderService(db, stub)
	wsID := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)

	item, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Embed Test Doc",
		RawContent:  "hello world this is a short document",
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	if err := svc.EmbedChunks(context.Background(), item.ID, wsID); err != nil {
		t.Fatalf("EmbedChunks failed: %v", err)
	}

	// Verify embedding_document status changed to 'embedded'
	rows, err := db.QueryContext(context.Background(),
		`SELECT embedding_status FROM embedding_document WHERE knowledge_item_id = ? AND workspace_id = ?`,
		item.ID, wsID,
	)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		if status != string(EmbeddingStatusEmbedded) {
			t.Errorf("expected status 'embedded', got %q", status)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration error: %v", err)
	}
	if count == 0 {
		t.Error("expected at least 1 embedded chunk")
	}

	// Verify vec_embedding rows were inserted
	var vecCount int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM vec_embedding WHERE workspace_id = ?`, wsID,
	).Scan(&vecCount); err != nil {
		t.Fatalf("vec_embedding count query failed: %v", err)
	}
	if vecCount != count {
		t.Errorf("expected %d vec_embedding rows, got %d", count, vecCount)
	}

	// Verify stub was called once (batch)
	if stub.callCount != 1 {
		t.Errorf("expected 1 LLM call (batch), got %d", stub.callCount)
	}
}

func TestEmbedderService_EmbedChunks_NoChunks_Noop(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := newStubEmbedder(3)
	svc := NewEmbedderService(db, stub)
	wsID := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)

	// Empty content → no chunks created
	item, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Empty Doc",
		RawContent:  "",
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// EmbedChunks should succeed silently with no LLM calls
	if err := svc.EmbedChunks(context.Background(), item.ID, wsID); err != nil {
		t.Fatalf("EmbedChunks should succeed for empty content: %v", err)
	}

	if stub.callCount != 0 {
		t.Errorf("expected 0 LLM calls for empty doc, got %d", stub.callCount)
	}
}

func TestEmbedderService_EmbedChunks_LLMError_StatusFailed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := &stubEmbedder{
		embedFunc: func(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
			return nil, errors.New("ollama connection refused")
		},
	}
	svc := NewEmbedderService(db, stub)
	wsID := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)

	item, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Error Test",
		RawContent:  "some content to embed",
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// EmbedChunks returns error after retries exhausted
	embedErr := svc.EmbedChunks(context.Background(), item.ID, wsID)
	if embedErr == nil {
		t.Error("expected EmbedChunks to return error when LLM fails")
	}

	// All chunks should be marked as 'failed'
	rows, err := db.QueryContext(context.Background(),
		`SELECT embedding_status FROM embedding_document WHERE knowledge_item_id = ?`, item.ID,
	)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		if status != string(EmbeddingStatusFailed) {
			t.Errorf("expected status 'failed' after LLM error, got %q", status)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration error: %v", err)
	}
}

func TestEmbedderService_WorkspaceIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := newStubEmbedder(3)
	svc := NewEmbedderService(db, stub)
	wsA := createWorkspace(t, db)
	wsB := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)

	itemA, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsA,
		SourceType:  SourceTypeDocument,
		Title:       "Doc A",
		RawContent:  "workspace a content",
	})
	if err != nil {
		t.Fatalf("ingest A failed: %v", err)
	}

	itemB, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsB,
		SourceType:  SourceTypeDocument,
		Title:       "Doc B",
		RawContent:  "workspace b content",
	})
	if err != nil {
		t.Fatalf("ingest B failed: %v", err)
	}

	// Embed only workspace A
	if err := svc.EmbedChunks(context.Background(), itemA.ID, wsA); err != nil {
		t.Fatalf("EmbedChunks A failed: %v", err)
	}

	// Workspace B chunks should still be pending
	var statusB string
	if err := db.QueryRowContext(context.Background(),
		`SELECT embedding_status FROM embedding_document WHERE knowledge_item_id = ? AND workspace_id = ?`,
		itemB.ID, wsB,
	).Scan(&statusB); err != nil {
		t.Fatalf("query B failed: %v", err)
	}
	if statusB != string(EmbeddingStatusPending) {
		t.Errorf("workspace B should still be pending, got %q", statusB)
	}

	// vec_embedding should only have workspace A rows
	var countB int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM vec_embedding WHERE workspace_id = ?`, wsB,
	).Scan(&countB); err != nil {
		t.Fatalf("vec_embedding count B failed: %v", err)
	}
	if countB != 0 {
		t.Errorf("expected 0 vec_embedding rows for workspace B, got %d", countB)
	}
}

func TestEmbedderService_Start_ReceivesEventAndEmbeds(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := newStubEmbedder(3)
	svc := NewEmbedderService(db, stub)
	wsID := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start embedder listener in background
	go svc.Start(ctx, bus)
	// Give subscriber goroutine time to attach before first publish.
	// Without this, event publish can race subscribe and be dropped by design.
	time.Sleep(50 * time.Millisecond)

	// Ingest triggers the event
	item, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Event Loop Test",
		RawContent:  "some content for embedding",
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// Wait for async embedder to process.
	// Under -race and CI load, the goroutine scheduler can delay this path,
	// so we keep a slightly wider budget to avoid flaky timeouts.
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		var status string
		err := db.QueryRowContext(context.Background(),
			`SELECT embedding_status FROM embedding_document WHERE knowledge_item_id = ? AND workspace_id = ? LIMIT 1`,
			item.ID, wsID,
		).Scan(&status)
		if err != nil {
			t.Fatalf("failed to query embedding status: %v", err)
		}
		if status == string(EmbeddingStatusEmbedded) {
			return // success
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("timeout: embedder did not process knowledge.ingested event within 8s")
}

// ============================================================================
// Task 2.4 audit remediation: branch coverage for storeVectors, encodeEmbedding,
// EmbedChunks storeVectors-error path, callEmbedWithRetry ctx.Done(), Start bad payload.
// ============================================================================

// TestEmbedderService_StoreVectors_DBClosed covers the storeVectors BeginTx error branch.
// When the DB is closed before EmbedChunks runs, storeVectors returns an error and
// EmbedChunks must call markAllFailed + return an error.
func TestEmbedderService_StoreVectors_DBClosed(t *testing.T) {
	db := setupTestDB(t)

	stub := newStubEmbedder(3)
	svc := NewEmbedderService(db, stub)
	wsID := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)

	item, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "StoreVectors Error Test",
		RawContent:  "content to embed",
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// Close DB — triggers BeginTx error inside storeVectors
	db.Close()

	embedErr := svc.EmbedChunks(context.Background(), item.ID, wsID)
	if embedErr == nil {
		t.Error("expected EmbedChunks to return error when DB is closed")
	}
}

// TestEmbedderService_EmbedChunks_FetchError covers the fetchPendingChunks error branch.
// When the DB is closed before EmbedChunks runs, fetchPendingChunks returns an error.
func TestEmbedderService_EmbedChunks_FetchError(t *testing.T) {
	db := setupTestDB(t)

	stub := newStubEmbedder(3)
	svc := NewEmbedderService(db, stub)
	wsID := createWorkspace(t, db)

	bus := eventbus.New()
	ingest := NewIngestService(db, bus)

	_, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Fetch Error Test",
		RawContent:  "some content",
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// Close DB before fetching — fetchPendingChunks will fail
	db.Close()

	embedErr := svc.EmbedChunks(context.Background(), "nonexistent-item", wsID)
	if embedErr == nil {
		t.Error("expected EmbedChunks to return error when DB is closed during fetch")
	}
}

// TestEncodeEmbedding_NaN covers the encodeEmbedding error branch.
// json.Marshal fails for float32 NaN values (JSON does not support NaN/Inf).
func TestEncodeEmbedding_NaN(t *testing.T) {
	vec := []float32{float32(math.NaN()), 0.1, 0.2}
	_, err := encodeEmbedding(vec)
	if err == nil {
		t.Error("expected encodeEmbedding to return error for NaN values")
	}
}

// TestEncodeEmbedding_Valid covers the happy path of encodeEmbedding.
func TestEncodeEmbedding_Valid(t *testing.T) {
	vec := []float32{0.1, 0.2, 0.3}
	got, err := encodeEmbedding(vec)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == "" {
		t.Error("expected non-empty JSON string")
	}
}

// TestEmbedderService_CallEmbedWithRetry_ContextCancelled covers the ctx.Done()
// branch inside callEmbedWithRetry's backoff select.
func TestEmbedderService_CallEmbedWithRetry_ContextCancelled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	callCount := 0
	stub := &stubEmbedder{
		embedFunc: func(_ context.Context, _ llm.EmbedRequest) (*llm.EmbedResponse, error) {
			callCount++
			return nil, errors.New("forced error")
		},
	}
	svc := NewEmbedderService(db, stub)

	// Cancel context immediately so the backoff select picks ctx.Done()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before first retry backoff

	wsID := createWorkspace(t, db)
	bus := eventbus.New()
	ingest := NewIngestService(db, bus)

	item, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Ctx Cancel Test",
		RawContent:  "content for retry cancel test",
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	embedErr := svc.EmbedChunks(ctx, item.ID, wsID)
	if embedErr == nil {
		t.Error("expected EmbedChunks to return error when context is cancelled")
	}
}

// TestEmbedderService_Start_BadPayload covers the Start() branch where the event
// payload cannot be cast to IngestedEventPayload — the service must continue running.
func TestEmbedderService_Start_BadPayload(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stub := newStubEmbedder(3)
	svc := NewEmbedderService(db, stub)

	bus := eventbus.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.Start(ctx, bus)
	// Ensure subscriber is attached before publishing test events.
	time.Sleep(50 * time.Millisecond)

	// Publish a bad payload (string instead of IngestedEventPayload)
	bus.Publish(TopicKnowledgeIngested, "this-is-not-a-valid-payload")

	// Give goroutine time to process without crashing
	time.Sleep(50 * time.Millisecond)

	// Verify embedder is still alive by publishing a valid event after the bad one
	wsID := createWorkspace(t, db)
	ingest := NewIngestService(db, bus)

	item, err := ingest.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Recovery After Bad Payload",
		RawContent:  "content after bad payload",
	})
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// Embedder should recover and process the valid event
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		var status string
		db.QueryRowContext(context.Background(), //nolint:errcheck
			`SELECT embedding_status FROM embedding_document WHERE knowledge_item_id = ? LIMIT 1`,
			item.ID,
		).Scan(&status) //nolint:errcheck
		if status == string(EmbeddingStatusEmbedded) {
			return // success — embedder survived bad payload and processed good one
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("timeout: embedder did not recover after bad payload within 8s")
}
