// Task 2.1: Knowledge Tables — Integration Tests
// Tests verify: schema, FTS5 sync, sqlite-vec virtual tables, multi-tenant isolation
// TDD: These tests are written BEFORE the migration exists — they will fail first.
// Traces: FR-090
package knowledge

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

// newID generates a new UUID v7 string for tests
func newID() string { return uuid.NewV7().String() }

// TestMain sets up test environment (JWT_SECRET required by MigrateUp chain)
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!")
	code := m.Run()
	os.Exit(code)
}

// setupTestDB creates an in-memory SQLite DB with all migrations applied
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	// IMPORTANT: with ":memory:" each SQLite connection has its own isolated DB.
	// Restrict pool to a single connection so async goroutines in tests (embedder)
	// see the same schema/data and avoid intermittent "no such table" failures.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	return db
}

// createWorkspace inserts a workspace row needed by FK constraints
func createWorkspace(t *testing.T, db *sql.DB) string {
	t.Helper()
	id := newID()
	_, err := db.Exec(
		`INSERT INTO workspace (id, name, slug, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, "Test Workspace", "test-ws-"+id, time.Now(), time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	return id
}

// insertKnowledgeItem helper for tests
func insertKnowledgeItem(t *testing.T, db *sql.DB, id, workspaceID, title, rawContent, normalizedContent string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO knowledge_item
		 (id, workspace_id, source_type, title, raw_content, normalized_content, created_at, updated_at)
		 VALUES (?, ?, 'document', ?, ?, ?, ?, ?)`,
		id, workspaceID, title, rawContent, normalizedContent, time.Now(), time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to insert knowledge_item: %v", err)
	}
}

// insertEmbeddingDocument helper for tests
func insertEmbeddingDocument(t *testing.T, db *sql.DB, id, knowledgeItemID, workspaceID, chunkText string, chunkIndex int) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO embedding_document
		 (id, knowledge_item_id, workspace_id, chunk_index, chunk_text, embedding_status, created_at)
		 VALUES (?, ?, ?, ?, ?, 'pending', ?)`,
		id, knowledgeItemID, workspaceID, chunkIndex, chunkText, time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to insert embedding_document: %v", err)
	}
}

// ============================================================================
// Schema Existence Tests
// ============================================================================

func TestKnowledgeItem_TableExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='knowledge_item'`,
	).Scan(&name)
	if err != nil || name != "knowledge_item" {
		t.Errorf("expected knowledge_item table to exist, got: %v (err: %v)", name, err)
	}
}

func TestEmbeddingDocument_TableExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='embedding_document'`,
	).Scan(&name)
	if err != nil || name != "embedding_document" {
		t.Errorf("expected embedding_document table to exist, got: %v (err: %v)", name, err)
	}
}

func TestEvidence_TableExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='evidence'`,
	).Scan(&name)
	if err != nil || name != "evidence" {
		t.Errorf("expected evidence table to exist, got: %v (err: %v)", name, err)
	}
}

func TestFTS5_VirtualTableExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='knowledge_item_fts'`,
	).Scan(&name)
	if err != nil || name != "knowledge_item_fts" {
		t.Errorf("expected knowledge_item_fts virtual table to exist, got: %v (err: %v)", name, err)
	}
}

// ============================================================================
// KnowledgeItem CRUD Tests
// ============================================================================

func TestKnowledgeItem_Insert_And_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	itemID := newID()

	insertKnowledgeItem(t, db, itemID, wsID, "Test Doc", "<p>hello world</p>", "hello world")

	var id, title string
	err := db.QueryRow(
		`SELECT id, title FROM knowledge_item WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL`,
		itemID, wsID,
	).Scan(&id, &title)
	if err != nil {
		t.Fatalf("failed to query knowledge_item: %v", err)
	}
	if id != itemID {
		t.Errorf("expected id %s, got %s", itemID, id)
	}
	if title != "Test Doc" {
		t.Errorf("expected title 'Test Doc', got %s", title)
	}
}

func TestKnowledgeItem_SoftDelete_ExcludedFromQueries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	itemID := newID()
	insertKnowledgeItem(t, db, itemID, wsID, "To Delete", "content", "content")

	// Soft delete
	_, err := db.Exec(
		`UPDATE knowledge_item SET deleted_at = ? WHERE id = ? AND workspace_id = ?`,
		time.Now(), itemID, wsID,
	)
	if err != nil {
		t.Fatalf("failed to soft delete: %v", err)
	}

	// Should not appear in standard query
	var count int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item WHERE id = ? AND deleted_at IS NULL`, itemID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count soft-deleted knowledge_item: %v", err)
	}
	if count != 0 {
		t.Errorf("expected soft-deleted item to be excluded, got count=%d", count)
	}
}

func TestKnowledgeItem_UniqueConstraint_EntityType_EntityID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	entityID := newID()

	// First insert should succeed
	id1 := newID()
	_, err := db.Exec(
		`INSERT INTO knowledge_item
		 (id, workspace_id, source_type, title, raw_content, entity_type, entity_id, created_at, updated_at)
		 VALUES (?, ?, 'document', 'Doc 1', 'content', 'case', ?, ?, ?)`,
		id1, wsID, entityID, time.Now(), time.Now(),
	)
	if err != nil {
		t.Fatalf("first insert should succeed: %v", err)
	}

	// Second insert with same entity_type + entity_id should fail
	id2 := newID()
	_, err = db.Exec(
		`INSERT INTO knowledge_item
		 (id, workspace_id, source_type, title, raw_content, entity_type, entity_id, created_at, updated_at)
		 VALUES (?, ?, 'document', 'Doc 2', 'content', 'case', ?, ?, ?)`,
		id2, wsID, entityID, time.Now(), time.Now(),
	)
	if err == nil {
		t.Error("expected UNIQUE constraint violation for duplicate entity_type+entity_id, but insert succeeded")
	}
}

func TestKnowledgeItem_WorkspaceIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsA := createWorkspace(t, db)
	wsB := createWorkspace(t, db)

	insertKnowledgeItem(t, db, newID(), wsA, "Doc A", "content a", "content a")
	insertKnowledgeItem(t, db, newID(), wsB, "Doc B", "content b", "content b")

	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item WHERE workspace_id = ? AND deleted_at IS NULL`, wsA,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count workspace A knowledge items: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 item in workspace A, got %d", count)
	}

	err = db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item WHERE workspace_id = ? AND deleted_at IS NULL`, wsB,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count workspace B knowledge items: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 item in workspace B, got %d", count)
	}
}

// ============================================================================
// FTS5 Sync Tests (via triggers)
// ============================================================================

func TestFTS5_AutoSync_OnInsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	itemID := newID()

	insertKnowledgeItem(t, db, itemID, wsID, "Pricing Strategy", "pricing discount policy", "pricing discount policy")

	// FTS5 should be queryable immediately via trigger
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item_fts WHERE knowledge_item_fts MATCH 'pricing' AND workspace_id = ?`,
		wsID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("FTS5 query failed: %v", err)
	}
	if count == 0 {
		t.Error("expected FTS5 index to contain 'pricing' after insert trigger, got 0")
	}
}

func TestFTS5_AutoSync_OnUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	itemID := newID()

	insertKnowledgeItem(t, db, itemID, wsID, "Old Title", "original content", "original content")

	// Update normalized_content
	_, err := db.Exec(
		`UPDATE knowledge_item SET normalized_content = ?, updated_at = ? WHERE id = ?`,
		"updated content with new keywords", time.Now(), itemID,
	)
	if err != nil {
		t.Fatalf("failed to update knowledge_item: %v", err)
	}

	// Old content should no longer match
	var count int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item_fts WHERE knowledge_item_fts MATCH 'original' AND workspace_id = ?`,
		wsID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query FTS5 old content count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected old content 'original' to be removed from FTS5 after update, got count=%d", count)
	}

	// New content should match
	err = db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item_fts WHERE knowledge_item_fts MATCH 'keywords' AND workspace_id = ?`,
		wsID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query FTS5 new content count: %v", err)
	}
	if count == 0 {
		t.Error("expected new content 'keywords' to be indexed in FTS5 after update, got 0")
	}
}

func TestFTS5_AutoSync_OnDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	itemID := newID()

	insertKnowledgeItem(t, db, itemID, wsID, "Temporary Doc", "temporary searchable content", "temporary searchable content")

	// Verify it's indexed
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item_fts WHERE knowledge_item_fts MATCH 'temporary' AND workspace_id = ?`,
		wsID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query FTS5 count before delete: %v", err)
	}
	if count == 0 {
		t.Fatal("expected FTS5 to index 'temporary' before delete")
	}

	// Delete the item
	_, err = db.Exec(`DELETE FROM knowledge_item WHERE id = ?`, itemID)
	if err != nil {
		t.Fatalf("failed to delete knowledge_item: %v", err)
	}

	// FTS5 should no longer contain it
	err = db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item_fts WHERE knowledge_item_fts MATCH 'temporary' AND workspace_id = ?`,
		wsID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query FTS5 count after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("expected FTS5 to remove 'temporary' after delete trigger, got count=%d", count)
	}
}

func TestFTS5_WorkspaceIsolation(t *testing.T) {
	// SECURITY TEST: FTS5 search must not return results from other workspaces
	db := setupTestDB(t)
	defer db.Close()

	wsA := createWorkspace(t, db)
	wsB := createWorkspace(t, db)

	insertKnowledgeItem(t, db, newID(), wsA, "Secret Doc A", "confidential alpha data", "confidential alpha data")
	insertKnowledgeItem(t, db, newID(), wsB, "Public Doc B", "public beta content", "public beta content")

	// Search in wsA should NOT return wsB results
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item_fts WHERE knowledge_item_fts MATCH 'beta' AND workspace_id = ?`,
		wsA,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query FTS5 beta count for workspace A: %v", err)
	}
	if count != 0 {
		t.Errorf("SECURITY VIOLATION: workspace A FTS5 search returned workspace B results (count=%d)", count)
	}

	// wsA can find its own content
	err = db.QueryRow(
		`SELECT COUNT(*) FROM knowledge_item_fts WHERE knowledge_item_fts MATCH 'alpha' AND workspace_id = ?`,
		wsA,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query FTS5 alpha count for workspace A: %v", err)
	}
	if count == 0 {
		t.Error("workspace A should find its own content via FTS5")
	}
}

// ============================================================================
// EmbeddingDocument Tests
// ============================================================================

func TestEmbeddingDocument_Insert_And_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	itemID := newID()
	insertKnowledgeItem(t, db, itemID, wsID, "Doc", "content", "content")

	chunk1 := newID()
	chunk2 := newID()
	insertEmbeddingDocument(t, db, chunk1, itemID, wsID, "first chunk text", 0)
	insertEmbeddingDocument(t, db, chunk2, itemID, wsID, "second chunk text", 1)

	rows, err := db.Query(
		`SELECT id, chunk_index, chunk_text, embedding_status FROM embedding_document
		 WHERE knowledge_item_id = ? AND workspace_id = ? ORDER BY chunk_index ASC`,
		itemID, wsID,
	)
	if err != nil {
		t.Fatalf("failed to query embedding_document: %v", err)
	}
	defer rows.Close()

	var chunks []struct {
		ID     string
		Index  int
		Text   string
		Status string
	}
	for rows.Next() {
		var c struct {
			ID     string
			Index  int
			Text   string
			Status string
		}
		if err := rows.Scan(&c.ID, &c.Index, &c.Text, &c.Status); err != nil {
			t.Fatalf("failed to scan embedding_document row: %v", err)
		}
		chunks = append(chunks, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration error for embedding_document: %v", err)
	}

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Status != "pending" {
		t.Errorf("expected status 'pending', got %s", chunks[0].Status)
	}
	if chunks[0].Index != 0 || chunks[1].Index != 1 {
		t.Errorf("expected chunk indexes 0,1 but got %d,%d", chunks[0].Index, chunks[1].Index)
	}
}

func TestEmbeddingDocument_StatusTransition_ToEmbedded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	itemID := newID()
	insertKnowledgeItem(t, db, itemID, wsID, "Doc", "content", "content")

	chunkID := newID()
	insertEmbeddingDocument(t, db, chunkID, itemID, wsID, "chunk text", 0)

	now := time.Now()
	_, err := db.Exec(
		`UPDATE embedding_document SET embedding_status = 'embedded', embedded_at = ? WHERE id = ?`,
		now, chunkID,
	)
	if err != nil {
		t.Fatalf("failed to update embedding status: %v", err)
	}

	var status string
	var embeddedAt sql.NullTime
	err = db.QueryRow(
		`SELECT embedding_status, embedded_at FROM embedding_document WHERE id = ?`, chunkID,
	).Scan(&status, &embeddedAt)
	if err != nil {
		t.Fatalf("failed to query embedding_document status transition: %v", err)
	}

	if status != "embedded" {
		t.Errorf("expected status 'embedded', got %s", status)
	}
	if !embeddedAt.Valid {
		t.Error("expected embedded_at to be set after status transition")
	}
}

func TestEmbeddingDocument_ForeignKey_ToKnowledgeItem(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	fakeItemID := newID() // does not exist in knowledge_item

	_, err := db.Exec(
		`INSERT INTO embedding_document
		 (id, knowledge_item_id, workspace_id, chunk_index, chunk_text, embedding_status, created_at)
		 VALUES (?, ?, ?, 0, 'text', 'pending', ?)`,
		newID(), fakeItemID, wsID, time.Now(),
	)
	if err == nil {
		t.Error("expected foreign key constraint violation for non-existent knowledge_item_id")
	}
}

func TestEmbeddingDocument_WorkspaceIsolation(t *testing.T) {
	// SECURITY TEST: embedding_document has workspace_id column,
	// vector search MUST join on it to prevent cross-tenant leaks
	db := setupTestDB(t)
	defer db.Close()

	wsA := createWorkspace(t, db)
	wsB := createWorkspace(t, db)

	itemA := newID()
	itemB := newID()
	insertKnowledgeItem(t, db, itemA, wsA, "Doc A", "content a", "content a")
	insertKnowledgeItem(t, db, itemB, wsB, "Doc B", "content b", "content b")

	chunkA := newID()
	chunkB := newID()
	insertEmbeddingDocument(t, db, chunkA, itemA, wsA, "workspace A secret data", 0)
	insertEmbeddingDocument(t, db, chunkB, itemB, wsB, "workspace B data", 0)

	// Searching with workspace_id filter must NOT return other workspace's chunks
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM embedding_document WHERE workspace_id = ?`, wsA,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count embedding_document rows for workspace A: %v", err)
	}
	if count != 1 {
		t.Errorf("expected only 1 embedding_document for workspace A, got %d", count)
	}

	// This is the pattern used in safe vector search (Task 2.5):
	// JOIN embedding_document ON vec_embedding.id = embedding_document.id WHERE embedding_document.workspace_id = ?
	err = db.QueryRow(
		`SELECT COUNT(*) FROM embedding_document WHERE workspace_id = ?`, wsB,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count embedding_document rows for workspace B: %v", err)
	}
	if count != 1 {
		t.Errorf("expected only 1 embedding_document for workspace B, got %d", count)
	}
}

// ============================================================================
// Evidence Table Tests
// ============================================================================

func TestEvidence_Insert_And_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	itemID := newID()
	insertKnowledgeItem(t, db, itemID, wsID, "Evidence Source", "content", "content")

	evidenceID := newID()
	_, err := db.Exec(
		`INSERT INTO evidence
		 (id, knowledge_item_id, workspace_id, method, score, snippet, created_at)
		 VALUES (?, ?, ?, 'bm25', 0.87, 'relevant snippet...', ?)`,
		evidenceID, itemID, wsID, time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to insert evidence: %v", err)
	}

	var score float64
	var method string
	err = db.QueryRow(
		`SELECT score, method FROM evidence WHERE id = ? AND workspace_id = ?`, evidenceID, wsID,
	).Scan(&score, &method)
	if err != nil {
		t.Fatalf("failed to query evidence: %v", err)
	}
	if method != "bm25" {
		t.Errorf("expected method 'bm25', got %s", method)
	}
	if score < 0.86 || score > 0.88 {
		t.Errorf("expected score ~0.87, got %f", score)
	}
}

func TestEvidence_Method_CheckConstraint(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	itemID := newID()
	insertKnowledgeItem(t, db, itemID, wsID, "Doc", "content", "content")

	_, err := db.Exec(
		`INSERT INTO evidence (id, knowledge_item_id, workspace_id, method, score, created_at)
		 VALUES (?, ?, ?, 'invalid_method', 0.5, ?)`,
		newID(), itemID, wsID, time.Now(),
	)
	if err == nil {
		t.Error("expected CHECK constraint violation for invalid method, but insert succeeded")
	}
}

func TestEvidence_ListByKnowledgeItem_OrderedByScore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	wsID := createWorkspace(t, db)
	itemID := newID()
	insertKnowledgeItem(t, db, itemID, wsID, "Doc", "content", "content")

	for _, score := range []float64{0.3, 0.9, 0.6} {
		_, err := db.Exec(
			`INSERT INTO evidence (id, knowledge_item_id, workspace_id, method, score, created_at)
			 VALUES (?, ?, ?, 'vector', ?, ?)`,
			newID(), itemID, wsID, score, time.Now(),
		)
		if err != nil {
			t.Fatalf("failed to insert evidence: %v", err)
		}
	}

	rows, err := db.Query(
		`SELECT score FROM evidence WHERE knowledge_item_id = ? AND workspace_id = ? ORDER BY score DESC`,
		itemID, wsID,
	)
	if err != nil {
		t.Fatalf("failed to query evidence: %v", err)
	}
	defer rows.Close()

	var scores []float64
	for rows.Next() {
		var s float64
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("failed to scan evidence row: %v", err)
		}
		scores = append(scores, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration error for evidence list: %v", err)
	}

	if len(scores) != 3 {
		t.Fatalf("expected 3 evidence rows, got %d", len(scores))
	}
	if scores[0] < scores[1] || scores[1] < scores[2] {
		t.Errorf("expected scores ordered DESC, got %v", scores)
	}
}
