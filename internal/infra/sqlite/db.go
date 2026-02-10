// Package sqlite provides the SQLite database connection factory for FenixCRM.
// Uses modernc.org/sqlite — a pure-Go SQLite driver (no CGO required).
// Task 1.2.4: Database connection with WAL mode, FK enforcement, and connection pool.
package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// Register the modernc sqlite driver under the name "sqlite"
	_ "modernc.org/sqlite"
)

// NewDB opens (or creates) a SQLite database at path and configures it for production use:
//   - WAL journal mode (allows concurrent reads during writes)
//   - Foreign key enforcement (SQLite disables FKs by default)
//   - 5-second busy timeout (prevents SQLITE_BUSY errors under burst writes)
//   - Synchronous=NORMAL (safe + faster than FULL for WAL mode)
//
// Use ":memory:" as path for in-memory databases in tests.
// Returns an error if the parent directory does not exist (will not create it).
func NewDB(path string) (*sql.DB, error) {
	// Task 1.2.4: Validate parent directory exists (for non-memory paths)
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return nil, fmt.Errorf("sqlite.NewDB: parent directory %q does not exist", dir)
		}
	}

	// DSN with PRAGMAs applied at connection time via query parameters.
	// modernc.org/sqlite supports _pragma=... params in the DSN.
	dsn := path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(ON)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=cache_size(-64000)" + // 64MB page cache (negative = KB)
		"&_pragma=temp_store(MEMORY)"   // temp tables in RAM

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite.NewDB: open %q: %w", path, err)
	}

	// Task 1.2.4: Connection pool configuration for SQLite WAL.
	// WAL allows concurrent readers but serializes writers.
	// MaxOpenConns > 1 is safe for reads; writers are serialized by SQLite itself.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Verify the connection is alive and PRAGMAs were applied.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite.NewDB: ping %q: %w", path, err)
	}

	return db, nil
}
