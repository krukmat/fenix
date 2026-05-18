// Task B.2.2 — Integration tests for SQLiteSignalRepository.
// Uses a real SQLite file (via mustOpenDB + MigrateUp) — no mocks.
package relationship_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/relationship"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// mustOpenDB opens a real SQLite file and applies all migrations.
// The file is cleaned up automatically when the test ends.
func mustOpenDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("sqlite.NewDB: %v", err)
	}
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// seedWorkspace inserts a minimal workspace row required by the FK constraint
// on relationship_memory.workspace_id.
func seedWorkspace(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, id, "test-ws-"+id, "slug-"+id)
	if err != nil {
		t.Fatalf("seedWorkspace(%s): %v", id, err)
	}
}

func seedInteractionSignal(t *testing.T, db *sql.DB, workspaceID string, entityType relationship.EntityType, entityID string) string {
	t.Helper()

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	mem, err := repo.UpsertMemory(ctx, workspaceID, entityType, entityID, "seed summary")
	if err != nil {
		t.Fatalf("seedInteractionSignal UpsertMemory(%s): %v", workspaceID, err)
	}

	signalID, err := repo.InsertSignal(
		ctx,
		mem.ID,
		relationship.SignalNote,
		relationship.SentimentNeutral,
		"seed signal summary",
		"note",
		"seed-note-"+entityID,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("seedInteractionSignal InsertSignal(%s): %v", workspaceID, err)
	}
	return signalID
}

func TestRepository_UpsertMemoryCreates(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-create")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	mem, err := repo.UpsertMemory(ctx, "ws-create", relationship.EntityTypeAccount, "acc-001", "first summary")
	if err != nil {
		t.Fatalf("UpsertMemory error = %v", err)
	}
	if mem.ID == "" {
		t.Error("expected non-empty Memory.ID")
	}
	if mem.Summary != "first summary" {
		t.Errorf("Summary = %q; want %q", mem.Summary, "first summary")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM relationship_memory WHERE workspace_id='ws-create'`).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("row count = %d; want 1", count)
	}
}

func TestRepository_UpsertMemoryUpdates(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-update")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	if _, err := repo.UpsertMemory(ctx, "ws-update", relationship.EntityTypeContact, "con-001", "first summary"); err != nil {
		t.Fatalf("first UpsertMemory error = %v", err)
	}
	mem, err := repo.UpsertMemory(ctx, "ws-update", relationship.EntityTypeContact, "con-001", "updated summary")
	if err != nil {
		t.Fatalf("second UpsertMemory error = %v", err)
	}
	if mem.Summary != "updated summary" {
		t.Errorf("Summary = %q; want %q", mem.Summary, "updated summary")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM relationship_memory WHERE workspace_id='ws-update'`).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("row count after double upsert = %d; want 1", count)
	}
}

func TestRepository_InsertSignal(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-signal")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	mem, err := repo.UpsertMemory(ctx, "ws-signal", relationship.EntityTypeDeal, "deal-001", "deal summary")
	if err != nil {
		t.Fatalf("UpsertMemory error = %v", err)
	}

	signalID, err := repo.InsertSignal(ctx,
		mem.ID,
		relationship.SignalNote,
		relationship.SentimentPositive,
		"customer replied",
		"note", "note-001",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("InsertSignal error = %v", err)
	}
	if signalID == "" {
		t.Fatal("expected non-empty signal ID")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM interaction_signal WHERE relationship_memory_id=?`, mem.ID).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("interaction_signal row count = %d; want 1", count)
	}
}

func TestRepository_UpsertTrustScoreCreates(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-trust-create")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	mem, err := repo.UpsertMemory(ctx, "ws-trust-create", relationship.EntityTypeContact, "con-trust-001", "trust summary")
	if err != nil {
		t.Fatalf("UpsertMemory error = %v", err)
	}

	lastScoredAt := time.Date(2026, time.May, 18, 10, 0, 0, 0, time.UTC)
	err = repo.UpsertTrustScore(ctx, mem.ID, 0.81, relationship.ConfidenceHigh, 0.92, lastScoredAt)
	if err != nil {
		t.Fatalf("UpsertTrustScore error = %v", err)
	}

	var (
		score        float64
		confidence   string
		decayFactor  float64
		lastScored   string
		count        int
	)
	if err := db.QueryRow(`
		SELECT COUNT(*), score, confidence, decay_factor, last_scored_at
		FROM trust_score
		WHERE relationship_memory_id = ?
	`, mem.ID).Scan(&count, &score, &confidence, &decayFactor, &lastScored); err != nil {
		t.Fatalf("trust_score query: %v", err)
	}
	if count != 1 {
		t.Fatalf("trust_score row count = %d; want 1", count)
	}
	if score != 0.81 {
		t.Errorf("score = %v; want 0.81", score)
	}
	if confidence != string(relationship.ConfidenceHigh) {
		t.Errorf("confidence = %q; want %q", confidence, relationship.ConfidenceHigh)
	}
	if decayFactor != 0.92 {
		t.Errorf("decay_factor = %v; want 0.92", decayFactor)
	}
	if lastScored != lastScoredAt.Format(time.RFC3339) {
		t.Errorf("last_scored_at = %q; want %q", lastScored, lastScoredAt.Format(time.RFC3339))
	}
}

func TestRepository_UpsertTrustScoreUpdatesAndPreservesCreatedAt(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-trust-update")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	mem, err := repo.UpsertMemory(ctx, "ws-trust-update", relationship.EntityTypeAccount, "acc-trust-001", "trust summary")
	if err != nil {
		t.Fatalf("UpsertMemory error = %v", err)
	}

	firstScoredAt := time.Date(2026, time.May, 18, 9, 0, 0, 0, time.UTC)
	if err := repo.UpsertTrustScore(ctx, mem.ID, 0.55, relationship.ConfidenceLow, 0.70, firstScoredAt); err != nil {
		t.Fatalf("first UpsertTrustScore error = %v", err)
	}

	var createdAtBefore string
	if err := db.QueryRow(`SELECT created_at FROM trust_score WHERE relationship_memory_id = ?`, mem.ID).Scan(&createdAtBefore); err != nil {
		t.Fatalf("query created_at before update: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	secondScoredAt := time.Date(2026, time.May, 18, 11, 30, 0, 0, time.UTC)
	if err := repo.UpsertTrustScore(ctx, mem.ID, 0.88, relationship.ConfidenceMedium, 0.95, secondScoredAt); err != nil {
		t.Fatalf("second UpsertTrustScore error = %v", err)
	}

	var (
		rowID          string
		score          float64
		confidence     string
		decayFactor    float64
		lastScoredAt   string
		createdAtAfter string
		count          int
	)
	if err := db.QueryRow(`
		SELECT COUNT(*), id, score, confidence, decay_factor, last_scored_at, created_at
		FROM trust_score
		WHERE relationship_memory_id = ?
	`, mem.ID).Scan(&count, &rowID, &score, &confidence, &decayFactor, &lastScoredAt, &createdAtAfter); err != nil {
		t.Fatalf("trust_score update query: %v", err)
	}

	if count != 1 {
		t.Fatalf("trust_score row count after update = %d; want 1", count)
	}
	if rowID == "" {
		t.Fatal("expected non-empty trust_score id")
	}
	if score != 0.88 {
		t.Errorf("score = %v; want 0.88", score)
	}
	if confidence != string(relationship.ConfidenceMedium) {
		t.Errorf("confidence = %q; want %q", confidence, relationship.ConfidenceMedium)
	}
	if decayFactor != 0.95 {
		t.Errorf("decay_factor = %v; want 0.95", decayFactor)
	}
	if lastScoredAt != secondScoredAt.Format(time.RFC3339) {
		t.Errorf("last_scored_at = %q; want %q", lastScoredAt, secondScoredAt.Format(time.RFC3339))
	}
	if createdAtAfter != createdAtBefore {
		t.Errorf("created_at changed from %q to %q; want preserved", createdAtBefore, createdAtAfter)
	}
}

func TestRepository_UpsertTrustScoreRejectsInvalidConfidence(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-trust-confidence")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	mem, err := repo.UpsertMemory(ctx, "ws-trust-confidence", relationship.EntityTypeLead, "lead-trust-001", "trust summary")
	if err != nil {
		t.Fatalf("UpsertMemory error = %v", err)
	}

	err = repo.UpsertTrustScore(ctx, mem.ID, 0.60, relationship.ConfidenceLevel("invalid"), 0.85, time.Now().UTC())
	if err == nil {
		t.Fatal("UpsertTrustScore with invalid confidence returned nil error; want CHECK constraint error")
	}
}

func TestRepository_UpsertTrustScoreRejectsOutOfRangeScore(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-trust-score")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	mem, err := repo.UpsertMemory(ctx, "ws-trust-score", relationship.EntityTypeDeal, "deal-trust-001", "trust summary")
	if err != nil {
		t.Fatalf("UpsertMemory error = %v", err)
	}

	err = repo.UpsertTrustScore(ctx, mem.ID, 1.25, relationship.ConfidenceHigh, 0.90, time.Now().UTC())
	if err == nil {
		t.Fatal("UpsertTrustScore with score > 1 returned nil error; want CHECK constraint error")
	}
}

func TestRepository_UpsertTrustScoreRejectsMissingRelationshipMemory(t *testing.T) {
	db := mustOpenDB(t)
	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	err := repo.UpsertTrustScore(ctx, "missing-memory-id", 0.50, relationship.ConfidenceLow, 1.0, time.Now().UTC())
	if err == nil {
		t.Fatal("UpsertTrustScore with missing relationship_memory_id returned nil error; want FK error")
	}
}

func TestRepository_UpsertEdgeCreates(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-edge-create")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	err := repo.UpsertEdge(ctx, "ws-edge-create", "contact", "from-001", "account", "to-001", relationship.InfluenceApproves, 0.60)
	if err != nil {
		t.Fatalf("UpsertEdge error = %v", err)
	}

	var (
		count         int
		influenceType string
		strength      float64
	)
	if err := db.QueryRow(`
		SELECT COUNT(*), influence_type, strength
		FROM stakeholder_graph
		WHERE workspace_id = ? AND from_entity_type = ? AND from_entity_id = ? AND to_entity_type = ? AND to_entity_id = ?
	`, "ws-edge-create", "contact", "from-001", "account", "to-001").Scan(&count, &influenceType, &strength); err != nil {
		t.Fatalf("stakeholder_graph query: %v", err)
	}
	if count != 1 {
		t.Fatalf("stakeholder_graph row count = %d; want 1", count)
	}
	if influenceType != string(relationship.InfluenceApproves) {
		t.Errorf("influence_type = %q; want %q", influenceType, relationship.InfluenceApproves)
	}
	if strength != 0.60 {
		t.Errorf("strength = %v; want 0.60", strength)
	}
}

func TestRepository_UpsertEdgeUpdatesAndPreservesCreatedAt(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-edge-update")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	err := repo.UpsertEdge(ctx, "ws-edge-update", "user", "user-001", "case", "case-001", relationship.InfluenceApproves, 0.60)
	if err != nil {
		t.Fatalf("first UpsertEdge error = %v", err)
	}

	var createdAtBefore string
	if err := db.QueryRow(`
		SELECT created_at
		FROM stakeholder_graph
		WHERE workspace_id = ? AND from_entity_type = ? AND from_entity_id = ? AND to_entity_type = ? AND to_entity_id = ? AND influence_type = ?
	`, "ws-edge-update", "user", "user-001", "case", "case-001", string(relationship.InfluenceApproves)).Scan(&createdAtBefore); err != nil {
		t.Fatalf("query created_at before update: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	err = repo.UpsertEdge(ctx, "ws-edge-update", "user", "user-001", "case", "case-001", relationship.InfluenceApproves, 0.90)
	if err != nil {
		t.Fatalf("second UpsertEdge error = %v", err)
	}

	var (
		count          int
		strength       float64
		createdAtAfter string
	)
	if err := db.QueryRow(`
		SELECT COUNT(*), strength, created_at
		FROM stakeholder_graph
		WHERE workspace_id = ? AND from_entity_type = ? AND from_entity_id = ? AND to_entity_type = ? AND to_entity_id = ? AND influence_type = ?
	`, "ws-edge-update", "user", "user-001", "case", "case-001", string(relationship.InfluenceApproves)).Scan(&count, &strength, &createdAtAfter); err != nil {
		t.Fatalf("stakeholder_graph update query: %v", err)
	}
	if count != 1 {
		t.Fatalf("stakeholder_graph row count after update = %d; want 1", count)
	}
	if strength != 0.90 {
		t.Errorf("strength = %v; want 0.90", strength)
	}
	if createdAtAfter != createdAtBefore {
		t.Errorf("created_at changed from %q to %q; want preserved", createdAtBefore, createdAtAfter)
	}
}

func TestRepository_UpsertEdgeAcceptsUserFromEntityType(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-edge-user")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	err := repo.UpsertEdge(ctx, "ws-edge-user", "user", "user-002", "contact", "contact-001", relationship.InfluenceReportsTo, 0.50)
	if err != nil {
		t.Fatalf("UpsertEdge with user from_entity_type error = %v", err)
	}
}

func TestRepository_UpsertEdgeRejectsOutOfRangeStrength(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-edge-strength")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	err := repo.UpsertEdge(ctx, "ws-edge-strength", "contact", "from-002", "account", "to-002", relationship.InfluenceInfluences, 1.50)
	if err == nil {
		t.Fatal("UpsertEdge with strength > 1 returned nil error; want CHECK constraint error")
	}
}

func TestRepository_UpsertEdgeDifferentInfluenceTypeCreatesNewRow(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-edge-multi")

	repo := relationship.NewSQLiteSignalRepository(db)
	ctx := context.Background()

	err := repo.UpsertEdge(ctx, "ws-edge-multi", "contact", "from-003", "account", "to-003", relationship.InfluenceApproves, 0.60)
	if err != nil {
		t.Fatalf("first UpsertEdge error = %v", err)
	}
	err = repo.UpsertEdge(ctx, "ws-edge-multi", "contact", "from-003", "account", "to-003", relationship.InfluenceInfluences, 0.40)
	if err != nil {
		t.Fatalf("second UpsertEdge error = %v", err)
	}

	var count int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM stakeholder_graph
		WHERE workspace_id = ? AND from_entity_type = ? AND from_entity_id = ? AND to_entity_type = ? AND to_entity_id = ?
	`, "ws-edge-multi", "contact", "from-003", "account", "to-003").Scan(&count); err != nil {
		t.Fatalf("stakeholder_graph count query: %v", err)
	}
	if count != 2 {
		t.Fatalf("stakeholder_graph row count for distinct influence types = %d; want 2", count)
	}
}

func TestRepository_UpsertSignalEmbeddingCreatesRows(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-embed-create")

	repo := relationship.NewSQLiteSignalRepositoryWithEmbeddingDim(db, 3)
	ctx := context.Background()
	signalID := seedInteractionSignal(t, db, "ws-embed-create", "contact", "embed-entity-001")

	err := repo.UpsertSignalEmbedding(ctx, "ws-embed-create", signalID, []float32{0.1, 0.2, 0.3})
	if err != nil {
		t.Fatalf("UpsertSignalEmbedding error = %v", err)
	}

	var (
		linkCount   int
		vecID       string
		dim         int
		vecCount    int
		embeddingJS string
	)
	if err := db.QueryRow(`
		SELECT COUNT(*), vec_embedding_id, dim
		FROM interaction_signal_embedding
		WHERE workspace_id = ? AND signal_id = ?
	`, "ws-embed-create", signalID).Scan(&linkCount, &vecID, &dim); err != nil {
		t.Fatalf("interaction_signal_embedding query: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("interaction_signal_embedding row count = %d; want 1", linkCount)
	}
	if dim != 3 {
		t.Errorf("dim = %d; want 3", dim)
	}
	if err := db.QueryRow(`SELECT COUNT(*), embedding FROM vec_embedding WHERE id = ? AND workspace_id = ?`, vecID, "ws-embed-create").Scan(&vecCount, &embeddingJS); err != nil {
		t.Fatalf("vec_embedding query: %v", err)
	}
	if vecCount != 1 {
		t.Fatalf("vec_embedding row count = %d; want 1", vecCount)
	}
	if embeddingJS != "[0.1,0.2,0.3]" {
		t.Errorf("embedding JSON = %q; want %q", embeddingJS, "[0.1,0.2,0.3]")
	}
}

func TestRepository_UpsertSignalEmbeddingReplacesExistingVector(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-embed-update")

	repo := relationship.NewSQLiteSignalRepositoryWithEmbeddingDim(db, 3)
	ctx := context.Background()
	signalID := seedInteractionSignal(t, db, "ws-embed-update", "account", "embed-entity-002")

	if err := repo.UpsertSignalEmbedding(ctx, "ws-embed-update", signalID, []float32{0.1, 0.2, 0.3}); err != nil {
		t.Fatalf("first UpsertSignalEmbedding error = %v", err)
	}

	var firstVecID string
	if err := db.QueryRow(`SELECT vec_embedding_id FROM interaction_signal_embedding WHERE workspace_id = ? AND signal_id = ?`, "ws-embed-update", signalID).Scan(&firstVecID); err != nil {
		t.Fatalf("query first vec id: %v", err)
	}

	if err := repo.UpsertSignalEmbedding(ctx, "ws-embed-update", signalID, []float32{0.4, 0.5, 0.6}); err != nil {
		t.Fatalf("second UpsertSignalEmbedding error = %v", err)
	}

	var (
		linkCount   int
		secondVecID string
		vecCountOld int
		vecCountNew int
	)
	if err := db.QueryRow(`SELECT COUNT(*), vec_embedding_id FROM interaction_signal_embedding WHERE workspace_id = ? AND signal_id = ?`, "ws-embed-update", signalID).Scan(&linkCount, &secondVecID); err != nil {
		t.Fatalf("query updated vec id: %v", err)
	}
	if linkCount != 1 {
		t.Fatalf("interaction_signal_embedding row count after update = %d; want 1", linkCount)
	}
	if firstVecID == secondVecID {
		t.Fatalf("vec_embedding_id did not change after replacement; both = %q", firstVecID)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM vec_embedding WHERE id = ?`, firstVecID).Scan(&vecCountOld); err != nil {
		t.Fatalf("count old vec embedding: %v", err)
	}
	if vecCountOld != 0 {
		t.Fatalf("old vec_embedding row count = %d; want 0", vecCountOld)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM vec_embedding WHERE id = ?`, secondVecID).Scan(&vecCountNew); err != nil {
		t.Fatalf("count new vec embedding: %v", err)
	}
	if vecCountNew != 1 {
		t.Fatalf("new vec_embedding row count = %d; want 1", vecCountNew)
	}
}

func TestRepository_UpsertSignalEmbeddingWorkspaceIsolation(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-embed-a")
	seedWorkspace(t, db, "ws-embed-b")

	repo := relationship.NewSQLiteSignalRepositoryWithEmbeddingDim(db, 3)
	ctx := context.Background()
	signalA := seedInteractionSignal(t, db, "ws-embed-a", "contact", "embed-entity-a")
	signalB := seedInteractionSignal(t, db, "ws-embed-b", "contact", "embed-entity-b")

	if err := repo.UpsertSignalEmbedding(ctx, "ws-embed-a", signalA, []float32{0.1, 0.2, 0.3}); err != nil {
		t.Fatalf("UpsertSignalEmbedding A error = %v", err)
	}
	if err := repo.UpsertSignalEmbedding(ctx, "ws-embed-b", signalB, []float32{0.4, 0.5, 0.6}); err != nil {
		t.Fatalf("UpsertSignalEmbedding B error = %v", err)
	}

	var countA, countB int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM interaction_signal_embedding ise
		JOIN vec_embedding v ON v.id = ise.vec_embedding_id
		WHERE ise.workspace_id = ? AND v.workspace_id = ?
	`, "ws-embed-a", "ws-embed-a").Scan(&countA); err != nil {
		t.Fatalf("workspace A join count query: %v", err)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM interaction_signal_embedding ise
		JOIN vec_embedding v ON v.id = ise.vec_embedding_id
		WHERE ise.workspace_id = ? AND v.workspace_id = ?
	`, "ws-embed-a", "ws-embed-b").Scan(&countB); err != nil {
		t.Fatalf("cross-tenant join count query: %v", err)
	}
	if countA != 1 {
		t.Fatalf("workspace A join count = %d; want 1", countA)
	}
	if countB != 0 {
		t.Fatalf("cross-tenant join count = %d; want 0", countB)
	}
}

func TestRepository_UpsertSignalEmbeddingRejectsDimMismatchBeforeWrite(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-embed-dim")

	repo := relationship.NewSQLiteSignalRepositoryWithEmbeddingDim(db, 3)
	ctx := context.Background()
	signalID := seedInteractionSignal(t, db, "ws-embed-dim", "lead", "embed-entity-dim")

	err := repo.UpsertSignalEmbedding(ctx, "ws-embed-dim", signalID, []float32{0.1, 0.2})
	if err == nil {
		t.Fatal("UpsertSignalEmbedding dim mismatch returned nil error; want validation error")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM interaction_signal_embedding WHERE workspace_id = ? AND signal_id = ?`, "ws-embed-dim", signalID).Scan(&count); err != nil {
		t.Fatalf("count join rows after dim mismatch: %v", err)
	}
	if count != 0 {
		t.Fatalf("interaction_signal_embedding rows after dim mismatch = %d; want 0", count)
	}
}

func TestRepository_UpsertSignalEmbeddingRollbackLeavesNoOrphanVector(t *testing.T) {
	db := mustOpenDB(t)
	seedWorkspace(t, db, "ws-embed-rollback")

	repo := relationship.NewSQLiteSignalRepositoryWithEmbeddingDim(db, 3)
	ctx := context.Background()

	err := repo.UpsertSignalEmbedding(ctx, "ws-embed-rollback", "missing-signal-id", []float32{0.1, 0.2, 0.3})
	if err == nil {
		t.Fatal("UpsertSignalEmbedding with missing signal_id returned nil error; want FK failure")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM vec_embedding WHERE workspace_id = ?`, "ws-embed-rollback").Scan(&count); err != nil {
		t.Fatalf("count vec_embedding rows after rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("vec_embedding rows after rollback = %d; want 0", count)
	}
}
