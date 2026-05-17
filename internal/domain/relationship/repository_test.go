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
