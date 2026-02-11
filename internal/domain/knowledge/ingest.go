// Package knowledge — Task 2.2: IngestService for the ingestion pipeline.
// IngestService transforms raw content into a knowledge_item with chunked
// embedding_document records, then publishes a knowledge.ingested event.
package knowledge

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// TopicKnowledgeIngested is the event bus topic published after a successful ingest.
const TopicKnowledgeIngested = "knowledge.ingested"

// IngestedEventPayload carries identifiers for the downstream embedder (Task 2.4).
type IngestedEventPayload struct {
	KnowledgeItemID string
	WorkspaceID     string
	ChunkCount      int
}

// DefaultChunkSize and DefaultChunkOverlap are the ingestion defaults (Task 2.2).
const (
	DefaultChunkSize    = 512
	DefaultChunkOverlap = 50
)

// IngestService handles knowledge item creation and chunking (Task 2.2).
type IngestService struct {
	db  *sql.DB
	bus eventbus.EventBus
	q   *sqlcgen.Queries
}

// NewIngestService creates an IngestService backed by the given DB and event bus.
func NewIngestService(db *sql.DB, bus eventbus.EventBus) *IngestService {
	return &IngestService{
		db:  db,
		bus: bus,
		q:   sqlcgen.New(db),
	}
}

// Ingest creates (or updates) a knowledge_item, splits the raw content into
// chunks, inserts embedding_document rows with status=pending, and publishes
// a knowledge.ingested event.
//
// Idempotency: if a knowledge_item already exists for the same
// (workspace_id, entity_type, entity_id), the existing item is updated and
// its old chunks are replaced.
func (s *IngestService) Ingest(ctx context.Context, input CreateKnowledgeItemInput) (*KnowledgeItem, error) {
	now := time.Now()
	normalized := normalizeContent(input.RawContent)
	existingID := s.findExistingItemID(ctx, input)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	qtx := sqlcgen.New(tx)
	itemID, err := s.upsertKnowledgeItem(ctx, tx, qtx, existingID, input, normalized, now)
	if err != nil {
		return nil, err
	}

	chunks := Chunk(input.RawContent, DefaultChunkSize, DefaultChunkOverlap)
	if err := insertChunks(ctx, qtx, itemID, input.WorkspaceID, chunks, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.bus.Publish(TopicKnowledgeIngested, IngestedEventPayload{
		KnowledgeItemID: itemID,
		WorkspaceID:     input.WorkspaceID,
		ChunkCount:      len(chunks),
	})

	return &KnowledgeItem{
		ID:                itemID,
		WorkspaceID:       input.WorkspaceID,
		SourceType:        input.SourceType,
		Title:             input.Title,
		RawContent:        input.RawContent,
		NormalizedContent: ptrFromStr(normalized),
		EntityType:        input.EntityType,
		EntityID:          input.EntityID,
		Metadata:          input.Metadata,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// upsertKnowledgeItem inserts a new item or updates+clears chunks of an existing one.
// Returns the item ID (new or existing).
func (s *IngestService) upsertKnowledgeItem(
	ctx context.Context, tx *sql.Tx, qtx *sqlcgen.Queries,
	existingID string, input CreateKnowledgeItemInput, normalized string, now time.Time,
) (string, error) {
	if existingID == "" {
		return s.insertKnowledgeItem(ctx, qtx, input, normalized, now)
	}
	return existingID, s.updateKnowledgeItem(ctx, tx, qtx, existingID, input, normalized, now)
}

// insertKnowledgeItem inserts a new knowledge_item row and returns its ID.
func (s *IngestService) insertKnowledgeItem(
	ctx context.Context, qtx *sqlcgen.Queries,
	input CreateKnowledgeItemInput, normalized string, now time.Time,
) (string, error) {
	itemID := uuid.NewV7().String()
	err := qtx.CreateKnowledgeItem(ctx, sqlcgen.CreateKnowledgeItemParams{
		ID:                itemID,
		WorkspaceID:       input.WorkspaceID,
		SourceType:        string(input.SourceType),
		Title:             input.Title,
		RawContent:        input.RawContent,
		NormalizedContent: ptrFromStr(normalized),
		EntityType:        input.EntityType,
		EntityID:          input.EntityID,
		Metadata:          input.Metadata,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	return itemID, err
}

// updateKnowledgeItem updates content fields and removes old chunks for re-chunking.
func (s *IngestService) updateKnowledgeItem(
	ctx context.Context, tx *sql.Tx, qtx *sqlcgen.Queries,
	itemID string, input CreateKnowledgeItemInput, normalized string, now time.Time,
) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE knowledge_item SET title=?, raw_content=?, normalized_content=?, updated_at=? WHERE id=? AND workspace_id=?`,
		input.Title, input.RawContent, normalized, now, itemID, input.WorkspaceID,
	); err != nil {
		return err
	}
	return qtx.DeleteEmbeddingDocumentsByKnowledgeItem(ctx, sqlcgen.DeleteEmbeddingDocumentsByKnowledgeItemParams{
		KnowledgeItemID: itemID,
		WorkspaceID:     input.WorkspaceID,
	})
}

// insertChunks inserts embedding_document rows for each chunk with status=pending.
func insertChunks(ctx context.Context, qtx *sqlcgen.Queries, itemID, workspaceID string, chunks []string, now time.Time) error {
	for i, chunkText := range chunks {
		tokenCount := int64(len(strings.Fields(chunkText)))
		if err := qtx.CreateEmbeddingDocument(ctx, sqlcgen.CreateEmbeddingDocumentParams{
			ID:              uuid.NewV7().String(),
			KnowledgeItemID: itemID,
			WorkspaceID:     workspaceID,
			ChunkIndex:      int64(i),
			ChunkText:       chunkText,
			TokenCount:      &tokenCount,
			EmbeddingStatus: string(EmbeddingStatusPending),
			CreatedAt:       now,
		}); err != nil {
			return err
		}
	}
	return nil
}

// findExistingItemID returns the ID of an existing knowledge_item for the same
// entity (workspace+entity_type+entity_id), or empty string if not found.
func (s *IngestService) findExistingItemID(ctx context.Context, input CreateKnowledgeItemInput) string {
	if input.EntityType == nil || input.EntityID == nil {
		return ""
	}
	item, err := s.q.GetKnowledgeItemByEntity(ctx, sqlcgen.GetKnowledgeItemByEntityParams{
		WorkspaceID: input.WorkspaceID,
		EntityType:  input.EntityType,
		EntityID:    input.EntityID,
	})
	if err != nil {
		return ""
	}
	return item.ID
}

// normalizeContent strips HTML tags and trims whitespace from raw content.
// Returns the raw content unchanged when no HTML is detected (MVP).
func normalizeContent(raw string) string {
	return strings.TrimSpace(raw)
}

// ptrFromStr returns nil for empty string, otherwise a pointer to the value.
func ptrFromStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
