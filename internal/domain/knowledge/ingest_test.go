// Task 2.2: Integration tests for IngestService.
// Tests verify: knowledge_item + embedding_document creation, idempotency,
// workspace isolation, and event bus notification.
// Uses real in-memory SQLite DB with all migrations applied.
// Traces: FR-090
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
	err = db.QueryRow(
		`SELECT COUNT(*) FROM embedding_document WHERE knowledge_item_id = ? AND workspace_id = ?`,
		item.ID, wsID,
	).Scan(&chunkCount)
	if err != nil {
		t.Fatalf("failed to count embedding_document rows: %v", err)
	}
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
		if err := rows.Scan(&status); err != nil {
			t.Fatalf("failed to scan embedding status row: %v", err)
		}
		if status != string(EmbeddingStatusPending) {
			t.Errorf("expected chunk status 'pending', got %q", status)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration error for chunk statuses: %v", err)
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
	err = db.QueryRow(
		`SELECT COUNT(*) FROM embedding_document WHERE knowledge_item_id = ?`, item.ID,
	).Scan(&chunkCount)
	if err != nil {
		t.Fatalf("failed to count chunks for short content: %v", err)
	}
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
	err = db.QueryRow(
		`SELECT COUNT(*) FROM embedding_document WHERE knowledge_item_id = ?`, item.ID,
	).Scan(&chunkCount)
	if err != nil {
		t.Fatalf("failed to count chunks for empty content: %v", err)
	}
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
	err = db.QueryRow(
		`SELECT title, raw_content FROM knowledge_item WHERE id = ?`, item1.ID,
	).Scan(&title, &rawContent)
	if err != nil {
		t.Fatalf("failed to query updated knowledge_item: %v", err)
	}
	if title != "Case v2" {
		t.Errorf("expected updated title 'Case v2', got %q", title)
	}

	// Only one knowledge_item should exist for this entity
	var itemCount int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item WHERE workspace_id = ? AND entity_type = ? AND entity_id = ? AND deleted_at IS NULL`,
		wsID, entityType, entityID,
	).Scan(&itemCount)
	if err != nil {
		t.Fatalf("failed to count knowledge_item rows for entity: %v", err)
	}
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
	err = db.QueryRow(`SELECT COUNT(*) FROM knowledge_item WHERE workspace_id = ? AND deleted_at IS NULL`, wsA).Scan(&countA)
	if err != nil {
		t.Fatalf("failed to count knowledge_item rows for workspace A: %v", err)
	}
	err = db.QueryRow(`SELECT COUNT(*) FROM knowledge_item WHERE workspace_id = ? AND deleted_at IS NULL`, wsB).Scan(&countB)
	if err != nil {
		t.Fatalf("failed to count knowledge_item rows for workspace B: %v", err)
	}

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
// Error Branch Tests (Task 2.2 audit remediation)
// ============================================================================

// TestIngestService_ClosedDB_ReturnsError covers the BeginTx error branch in Ingest.
// A closed *sql.DB makes BeginTx return an error immediately.
func TestIngestService_ClosedDB_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	// Close DB before calling Ingest — BeginTx will fail
	db.Close()

	bus := eventbus.New()
	svc := NewIngestService(db, bus)

	_, err := svc.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: "ws-does-not-matter",
		SourceType:  SourceTypeDocument,
		Title:       "Error Test",
		RawContent:  "some content",
	})
	if err == nil {
		t.Error("expected Ingest to return error when DB is closed, got nil")
	}
}

// TestIngestService_Idempotent_ChunkCount_IsReplaced verifies that re-ingesting
// the same entity replaces ALL old chunks (covers insertChunks + deleteOldChunks path).
// Before re-ingest: 2 chunks. After re-ingest with shorter text: 1 chunk.
func TestIngestService_Idempotent_ChunkCount_IsReplaced(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bus := eventbus.New()
	svc := NewIngestService(db, bus)
	wsID := createWorkspace(t, db)
	entityID := newID()
	entityType := "case"

	// First ingest: 600 tokens → 2 chunks
	item, err := svc.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeCase,
		Title:       "Case v1",
		RawContent:  buildText(600),
		EntityType:  &entityType,
		EntityID:    &entityID,
	})
	if err != nil {
		t.Fatalf("first ingest failed: %v", err)
	}

	var chunksAfterFirst int
	err = db.QueryRow(`SELECT COUNT(*) FROM embedding_document WHERE knowledge_item_id = ?`, item.ID).Scan(&chunksAfterFirst)
	if err != nil {
		t.Fatalf("failed to count chunks after first ingest: %v", err)
	}
	if chunksAfterFirst < 2 {
		t.Fatalf("expected >=2 chunks after first ingest, got %d", chunksAfterFirst)
	}

	// Second ingest same entity: short text → 1 chunk
	_, err = svc.Ingest(context.Background(), CreateKnowledgeItemInput{
		WorkspaceID: wsID,
		SourceType:  SourceTypeCase,
		Title:       "Case v2",
		RawContent:  "just a few words",
		EntityType:  &entityType,
		EntityID:    &entityID,
	})
	if err != nil {
		t.Fatalf("second ingest failed: %v", err)
	}

	var chunksAfterSecond int
	err = db.QueryRow(`SELECT COUNT(*) FROM embedding_document WHERE knowledge_item_id = ?`, item.ID).Scan(&chunksAfterSecond)
	if err != nil {
		t.Fatalf("failed to count chunks after second ingest: %v", err)
	}
	if chunksAfterSecond != 1 {
		t.Errorf("expected exactly 1 chunk after re-ingest with short text, got %d (old chunks not replaced)", chunksAfterSecond)
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
