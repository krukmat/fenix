// Package knowledge — Task 2.4: EmbedderService.
// Consumes knowledge.ingested events from the event bus, calls LLMProvider.Embed()
// in batch per knowledge_item, stores vectors in vec_embedding, and marks
// embedding_document rows as 'embedded' or 'failed'.
package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"

	"database/sql"
)

const (
	embedMaxRetries = 3
	embedBaseDelay  = 100 * time.Millisecond
)

// EmbedderService processes pending embedding_document rows (Task 2.4).
type EmbedderService struct {
	db  *sql.DB
	q   *sqlcgen.Queries
	llm llm.LLMProvider
}

// NewEmbedderService creates an EmbedderService backed by the given DB and LLM provider.
func NewEmbedderService(db *sql.DB, provider llm.LLMProvider) *EmbedderService {
	return &EmbedderService{
		db:  db,
		q:   sqlcgen.New(db),
		llm: provider,
	}
}

// Start subscribes to TopicKnowledgeIngested and runs EmbedChunks for each event.
// Runs in the calling goroutine — launch with: go svc.Start(ctx, bus)
// Stops when ctx is cancelled.
func (s *EmbedderService) Start(ctx context.Context, bus eventbus.EventBus) {
	ch := bus.Subscribe(TopicKnowledgeIngested)
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-ch:
			payload, ok := evt.Payload.(IngestedEventPayload)
			if !ok {
				continue
			}
			// Best-effort: log error but keep running
			_ = s.EmbedChunks(ctx, payload.KnowledgeItemID, payload.WorkspaceID)
		}
	}
}

// EmbedChunks fetches all pending chunks for a knowledge_item, calls LLM.Embed()
// in a single batch, stores vectors in vec_embedding, and marks status='embedded'.
// If the LLM call fails after all retries, marks chunks as 'failed' and returns an error.
func (s *EmbedderService) EmbedChunks(ctx context.Context, knowledgeItemID, workspaceID string) error {
	chunks, err := s.fetchPendingChunks(ctx, knowledgeItemID, workspaceID)
	if err != nil {
		return fmt.Errorf("embedder: fetch chunks: %w", err)
	}
	if len(chunks) == 0 {
		return nil // nothing to embed
	}

	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.ChunkText
	}

	vecs, err := s.callEmbedWithRetry(ctx, texts)
	if err != nil {
		s.markAllFailed(ctx, chunks)
		return fmt.Errorf("embedder: LLM.Embed: %w", err)
	}

	if storeErr := s.storeVectors(ctx, chunks, vecs, workspaceID); storeErr != nil {
		s.markAllFailed(ctx, chunks)
		return fmt.Errorf("embedder: store vectors: %w", storeErr)
	}
	return nil
}

// fetchPendingChunks returns all embedding_document rows with status='pending'
// for the given knowledge_item within the workspace.
func (s *EmbedderService) fetchPendingChunks(ctx context.Context, itemID, wsID string) ([]sqlcgen.EmbeddingDocument, error) {
	return s.q.ListEmbeddingDocumentsByKnowledgeItem(ctx, sqlcgen.ListEmbeddingDocumentsByKnowledgeItemParams{
		KnowledgeItemID: itemID,
		WorkspaceID:     wsID,
	})
}

// callEmbedWithRetry calls LLMProvider.Embed() with exponential backoff.
// Attempts: maxRetries (100ms, 200ms, 400ms delays).
func (s *EmbedderService) callEmbedWithRetry(ctx context.Context, texts []string) ([][]float32, error) {
	var lastErr error
	delay := embedBaseDelay
	for attempt := 0; attempt < embedMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay *= 2
			}
		}
		resp, err := s.llm.Embed(ctx, llm.EmbedRequest{Texts: texts})
		if err == nil {
			return resp.Embeddings, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("all %d retries failed: %w", embedMaxRetries, lastErr)
}

// storeVectors inserts float32 vectors into vec_embedding and marks each
// embedding_document as 'embedded'. Runs in a single transaction.
func (s *EmbedderService) storeVectors(ctx context.Context, chunks []sqlcgen.EmbeddingDocument, vecs [][]float32, workspaceID string) error {
	now := time.Now()
	tx, txErr := s.db.BeginTx(ctx, nil)
	if txErr != nil {
		return txErr
	}
	defer tx.Rollback() //nolint:errcheck

	qtx := sqlcgen.New(tx)
	for i, chunk := range chunks {
		embJSON, encErr := encodeEmbedding(vecs[i])
		if encErr != nil {
			return fmt.Errorf("encode embedding[%d]: %w", i, encErr)
		}

		if insErr := qtx.InsertVecEmbedding(ctx, sqlcgen.InsertVecEmbeddingParams{
			ID:          chunk.ID,
			WorkspaceID: workspaceID,
			Embedding:   embJSON,
			CreatedAt:   now,
		}); insErr != nil {
			return fmt.Errorf("insert vec_embedding[%d]: %w", i, insErr)
		}

		if updErr := qtx.UpdateEmbeddingDocumentStatus(ctx, sqlcgen.UpdateEmbeddingDocumentStatusParams{
			EmbeddingStatus: string(EmbeddingStatusEmbedded),
			EmbeddedAt:      &now,
			ID:              chunk.ID,
			WorkspaceID:     workspaceID,
		}); updErr != nil {
			return fmt.Errorf("update embedding_document[%d]: %w", i, updErr)
		}
	}
	return tx.Commit()
}

// markAllFailed sets embedding_status='failed' on all given chunks.
// Called after all retries are exhausted. Errors are silently ignored to avoid
// masking the original embed error.
func (s *EmbedderService) markAllFailed(ctx context.Context, chunks []sqlcgen.EmbeddingDocument) {
	for _, chunk := range chunks {
		_ = s.q.UpdateEmbeddingDocumentStatus(ctx, sqlcgen.UpdateEmbeddingDocumentStatusParams{
			EmbeddingStatus: string(EmbeddingStatusFailed),
			EmbeddedAt:      nil,
			ID:              chunk.ID,
			WorkspaceID:     chunk.WorkspaceID,
		})
	}
}

// encodeEmbedding serialises a float32 slice to JSON TEXT for storage.
// e.g. [0.1, 0.2, 0.3] → "[0.1,0.2,0.3]"
func encodeEmbedding(vec []float32) (string, error) {
	b, err := json.Marshal(vec)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
