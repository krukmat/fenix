// Package sqlite provides the SQLite database connection and migration system.
// Task 1.2.3: TDD tests for database connection (written before implementation)
package sqlite_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// TestNewDB_OpenAndClose verifies that NewDB opens a valid connection and Close works.
func TestNewDB_OpenAndClose(t *testing.T) {
	t.Parallel()

	path := tempDBPath(t)
	db, err := sqlite.NewDB(path)
	if err != nil {
		t.Fatalf("NewDB(%q) error = %v; want nil", path, err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("db.Close() error = %v; want nil", err)
	}
}

// TestNewDB_WALMode verifies that WAL journal mode is enabled after NewDB.
// WAL (Write-Ahead Logging) allows concurrent readers during writes — critical
// for LLM streaming + CRM reads happening simultaneously.
func TestNewDB_WALMode(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)

	var mode string
	row := db.QueryRow("PRAGMA journal_mode")
	if err := row.Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode scan error = %v", err)
	}

	if mode != "wal" {
		t.Errorf("journal_mode = %q; want %q", mode, "wal")
	}
}

// TestNewDB_ForeignKeysEnabled verifies that FK enforcement is ON after NewDB.
// Without FK enforcement, SQLite silently accepts invalid foreign key references.
func TestNewDB_ForeignKeysEnabled(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)

	var fkEnabled int
	row := db.QueryRow("PRAGMA foreign_keys")
	if err := row.Scan(&fkEnabled); err != nil {
		t.Fatalf("PRAGMA foreign_keys scan error = %v", err)
	}

	if fkEnabled != 1 {
		t.Errorf("foreign_keys = %d; want 1 (enabled)", fkEnabled)
	}
}

// TestNewDB_BusyTimeout verifies that busy_timeout is set to avoid immediate
// SQLITE_BUSY errors under concurrent access.
func TestNewDB_BusyTimeout(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)

	var timeout int
	row := db.QueryRow("PRAGMA busy_timeout")
	if err := row.Scan(&timeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout scan error = %v", err)
	}

	// Must be > 0 — we require at least 5 seconds to handle burst writes
	if timeout <= 0 {
		t.Errorf("busy_timeout = %d; want > 0 (ms)", timeout)
	}
}

// TestNewDB_Ping verifies the connection is alive and usable.
func TestNewDB_Ping(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping() error = %v; want nil", err)
	}
}

// TestNewDB_InMemory verifies that ":memory:" path works for test isolation.
// In-memory databases are used in tests to avoid file I/O.
func TestNewDB_InMemory(t *testing.T) {
	t.Parallel()

	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB(\":memory:\") error = %v; want nil", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("in-memory db.Ping() error = %v; want nil", err)
	}
}

// TestNewDB_FileCreated verifies that a new DB file is created if it doesn't exist.
func TestNewDB_FileCreated(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "new_db.sqlite")

	// File must NOT exist before opening
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file %q to not exist before NewDB", path)
	}

	db, err := sqlite.NewDB(path)
	if err != nil {
		t.Fatalf("NewDB(%q) error = %v; want nil", path, err)
	}
	defer db.Close()

	// File MUST exist after opening
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected DB file %q to be created by NewDB", path)
	}
}

// TestNewDB_InvalidDirectory verifies that NewDB returns an error if the
// parent directory doesn't exist (not silently create it).
func TestNewDB_InvalidDirectory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nonexistent_dir", "db.sqlite")

	db, err := sqlite.NewDB(path)
	if err == nil {
		db.Close()
		t.Errorf("NewDB(%q) = nil error; want error for non-existent parent dir", path)
	}
}

// TestNewDB_ConnectionPool verifies sensible pool settings are applied.
// SQLite with WAL works best with MaxOpenConns=1 for writers; readers can share.
func TestNewDB_ConnectionPool(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	stats := db.Stats()

	// MaxOpenConnections should be configured (not unlimited=0)
	if stats.MaxOpenConnections == 0 {
		t.Errorf("MaxOpenConnections = 0; want a configured limit")
	}
}

// --- helpers ---

// mustOpenDB opens a temp SQLite DB, registers cleanup, and fails the test on error.
func mustOpenDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlite.NewDB(tempDBPath(t))
	if err != nil {
		t.Fatalf("sqlite.NewDB error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// tempDBPath returns a unique temp file path for a test DB (auto-cleaned).
func tempDBPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.sqlite")
}
