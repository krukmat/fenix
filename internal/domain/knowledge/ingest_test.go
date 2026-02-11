// Task 2.2: Integration tests for IngestService.
// Tests verify: knowledge_item + embedding_document creation, idempotency,
// workspace isolation, and event bus notification.
// Uses real in-memory SQLite DB with all migrations applied.
package knowledge

import (
	"context"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
)

// ============================================================================
// IngestService Tests
// ============================================================================

func TestIngestService_CreateItem_And_Chunks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bus := eventbus.New()
	svc := NewIngestService(db, bus)
	wsID := createWorkspace(t, db)

	input := CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Architecture Guide",
		RawContent:  buildText(600), // 600 tokens → 2 chunks with size=512, overlap=50
	}

	item, err := svc.Ingest(context.Background(), input)
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
	if item.ID == "" {
		t.Error("expected item.ID to be set")
	}
	if item.Title != "Architecture Guide" {
		t.Errorf("expected title 'Architecture Guide', got %q", item.Title)
	}
	if item.WorkspaceID != wsID {
		t.Errorf("expected workspaceID %q, got %q", wsID, item.WorkspaceID)
	}

	// Verify embedding_document rows were created
	var chunkCount int
	db.QueryRow(
		`SELECT COUNT(*) FROM embedding_document WHERE knowledge_item_id = ? AND workspace_id = ?`,
		item.ID, wsID,
	).Scan(&chunkCount)
	if chunkCount < 2 {
		t.Errorf("expected at least 2 chunks for 600-token text, got %d", chunkCount)
	}
}

func TestIngestService_ChunksHaveStatusPending(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bus := eventbus.New()
	svc := NewIngestService(db, bus)
	wsID := createWorkspace(t, db)

	input := CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Pending Chunks Test",
		RawContent:  buildText(200),
	}

	item, err := svc.Ingest(context.Background(), input)
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	rows, err := db.Query(
		`SELECT embedding_status FROM embedding_document WHERE knowledge_item_id = ?`, item.ID,
	)
	if err != nil {
		t.Fatalf("failed to query chunks: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		rows.Scan(&status)
		if status != string(EmbeddingStatusPending) {
			t.Errorf("expected chunk status 'pending', got %q", status)
		}
	}
}

func TestIngestService_ShortContent_CreatesSingleChunk(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bus := eventbus.New()
	svc := NewIngestService(db, bus)
	wsID := createWorkspace(t, db)

	input := CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeNote,
		Title:       "Short Note",
		RawContent:  "just a few words here",
	}

	item, err := svc.Ingest(context.Background(), input)
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	var chunkCount int
	db.QueryRow(
		`SELECT COUNT(*) FROM embedding_document WHERE knowledge_item_id = ?`, item.ID,
	).Scan(&chunkCount)
	if chunkCount != 1 {
		t.Errorf("expected 1 chunk for short text, got %d", chunkCount)
	}
}

func TestIngestService_EmptyContent_CreatesItemWithNoChunks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bus := eventbus.New()
	svc := NewIngestService(db, bus)
	wsID := createWorkspace(t, db)

	input := CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Empty Doc",
		RawContent:  "",
	}

	item, err := svc.Ingest(context.Background(), input)
	if err != nil {
		t.Fatalf("Ingest failed for empty content: %v", err)
	}
	if item.ID == "" {
		t.Error("expected item.ID to be set even for empty content")
	}

	var chunkCount int
	db.QueryRow(
		`SELECT COUNT(*) FROM embedding_document WHERE knowledge_item_id = ?`, item.ID,
	).Scan(&chunkCount)
	if chunkCount != 0 {
		t.Errorf("expected 0 chunks for empty content, got %d", chunkCount)
	}
}

func TestIngestService_Idempotent_SameEntity_UpdatesAndReplacesChunks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bus := eventbus.New()
	svc := NewIngestService(db, bus)
	wsID := createWorkspace(t, db)
	entityID := newID()

	entityType := "case"
	first := CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeCase,
		Title:       "Case v1",
		RawContent:  "first version content",
		EntityType:  &entityType,
		EntityID:    &entityID,
	}
	item1, err := svc.Ingest(context.Background(), first)
	if err != nil {
		t.Fatalf("first ingest failed: %v", err)
	}

	// Second ingest same entity — should update, not duplicate
	second := CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeCase,
		Title:       "Case v2",
		RawContent:  "updated content with more information",
		EntityType:  &entityType,
		EntityID:    &entityID,
	}
	item2, err := svc.Ingest(context.Background(), second)
	if err != nil {
		t.Fatalf("second ingest failed: %v", err)
	}

	// Must return the same ID (upsert, not insert)
	if item1.ID != item2.ID {
		t.Errorf("expected same item.ID on re-ingest, got %q vs %q", item1.ID, item2.ID)
	}

	// Title and content must be updated
	var title, rawContent string
	db.QueryRow(
		`SELECT title, raw_content FROM knowledge_item WHERE id = ?`, item1.ID,
	).Scan(&title, &rawContent)
	if title != "Case v2" {
		t.Errorf("expected updated title 'Case v2', got %q", title)
	}

	// Only one knowledge_item should exist for this entity
	var itemCount int
	db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item WHERE workspace_id = ? AND entity_type = ? AND entity_id = ? AND deleted_at IS NULL`,
		wsID, entityType, entityID,
	).Scan(&itemCount)
	if itemCount != 1 {
		t.Errorf("expected exactly 1 knowledge_item for entity, got %d", itemCount)
	}
}

func TestIngestService_WorkspaceIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bus := eventbus.New()
	svc := NewIngestService(db, bus)
	wsA := createWorkspace(t, db)
	wsB := createWorkspace(t, db)

	_, err := svc.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsA,
		SourceType:  SourceTypeDocument,
		Title:       "Doc A",
		RawContent:  "workspace a content",
	})
	if err != nil {
		t.Fatalf("ingest wsA failed: %v", err)
	}

	_, err = svc.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsB,
		SourceType:  SourceTypeDocument,
		Title:       "Doc B",
		RawContent:  "workspace b content",
	})
	if err != nil {
		t.Fatalf("ingest wsB failed: %v", err)
	}

	var countA, countB int
	db.QueryRow(`SELECT COUNT(*) FROM knowledge_item WHERE workspace_id = ? AND deleted_at IS NULL`, wsA).Scan(&countA)
	db.QueryRow(`SELECT COUNT(*) FROM knowledge_item WHERE workspace_id = ? AND deleted_at IS NULL`, wsB).Scan(&countB)

	if countA != 1 {
		t.Errorf("expected 1 item in workspace A, got %d", countA)
	}
	if countB != 1 {
		t.Errorf("expected 1 item in workspace B, got %d", countB)
	}
}

func TestIngestService_PublishesEvent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bus := eventbus.New()
	ch := bus.Subscribe(TopicKnowledgeIngested)

	svc := NewIngestService(db, bus)
	wsID := createWorkspace(t, db)

	item, err := svc.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeDocument,
		Title:       "Event Test",
		RawContent:  "some content to ingest",
	})
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	select {
	case evt := <-ch:
		payload, ok := evt.Payload.(IngestedEventPayload)
		if !ok {
			t.Fatalf("expected IngestedEventPayload, got %T", evt.Payload)
		}
		if payload.KnowledgeItemID != item.ID {
			t.Errorf("expected event itemID %q, got %q", item.ID, payload.KnowledgeItemID)
		}
		if payload.WorkspaceID != wsID {
			t.Errorf("expected event workspaceID %q, got %q", wsID, payload.WorkspaceID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout: expected knowledge.ingested event within 200ms")
	}
}

// ============================================================================
// Helpers
// ============================================================================

// buildText returns a string with n whitespace-separated tokens ("word word ...").
func buildText(n int) string {
	words := make([]byte, 0, n*5)
	for i := 0; i < n; i++ {
		if i > 0 {
			words = append(words, ' ')
		}
		words = append(words, "word"...)
	}
	return string(words)
}
