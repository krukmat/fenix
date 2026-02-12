// Task 2.4: Integration tests for EmbedderService.
// Uses real in-memory SQLite DB with all migrations applied.
// LLMProvider is a stub (no real Ollama needed).
package knowledge

import (
	"context"
	"errors"
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

	// Wait for async embedder to process (1s budget — LLM retries in other tests take ~300ms)
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		var status string
		db.QueryRowContext(context.Background(),
			`SELECT embedding_status FROM embedding_document WHERE knowledge_item_id = ? LIMIT 1`,
			item.ID,
		).Scan(&status) //nolint:errcheck
		if status == string(EmbeddingStatusEmbedded) {
			return // success
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("timeout: embedder did not process knowledge.ingested event within 500ms")
}
