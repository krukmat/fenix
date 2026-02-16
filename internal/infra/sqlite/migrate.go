// Task 1.2.5: Migration system for FenixCRM SQLite.
// Uses embed.FS to bundle SQL files into the binary (zero runtime file deps).
// Tracks applied migrations in schema_migrations table (idempotent by design).
package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// migrations embeds all *.up.sql files from the migrations directory.
// The embed directive is relative to this file's package directory.
//
//go:embed migrations/*.up.sql
var migrations embed.FS

// MigrateUp applies all pending *.up.sql migrations in order.
// Already-applied migrations are skipped (idempotent).
// Uses a transaction per migration for atomicity.
func MigrateUp(db *sql.DB) error {
	// Task 1.2.5: Ensure schema_migrations table exists before querying it
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("migrate: ensure migrations table: %w", err)
	}

	// Load all migration files from embedded FS, sorted by name (001_, 002_, ...)
	files, err := loadMigrationFiles()
	if err != nil {
		return fmt.Errorf("migrate: load files: %w", err)
	}

	for _, f := range files {
		version := versionFromFilename(f.name)

		// Skip already-applied migrations
		applied, checkErr := isMigrationApplied(db, version)
		if checkErr != nil {
			return fmt.Errorf("migrate: check applied %d: %w", version, checkErr)
		}
		if applied {
			continue
		}

		// Apply migration in a transaction
		if applyErr := applyMigration(db, version, f.name, f.sql); applyErr != nil {
			return fmt.Errorf("migrate: apply %s: %w", f.name, applyErr)
		}
	}

	return nil
}

// MigrationVersion returns the highest migration version number currently applied.
// Returns 0 if no migrations have been applied yet.
func MigrationVersion(db *sql.DB) (int, error) {
	if err := ensureMigrationsTable(db); err != nil {
		return 0, fmt.Errorf("migrate: ensure migrations table: %w", err)
	}

	var version int
	row := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err := row.Scan(&version); err != nil {
		return 0, fmt.Errorf("migrate: query version: %w", err)
	}

	return version, nil
}

// --- internal ---

// migrationFile holds a parsed migration file ready to apply.
type migrationFile struct {
	name string // e.g. "001_init_schema.up.sql"
	sql  string // full SQL content
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist.
// This is always run first so we can track which migrations have been applied.
func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     INTEGER NOT NULL PRIMARY KEY,
			name        TEXT    NOT NULL,
			applied_at  TEXT    NOT NULL DEFAULT (datetime('now'))
		)
	`)
	return err
}

// loadMigrationFiles reads all *.up.sql files from the embedded FS and sorts them.
func loadMigrationFiles() ([]migrationFile, error) {
	var files []migrationFile

	err := fs.WalkDir(migrations, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".up.sql") {
			return nil
		}

		content, err := migrations.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		// Use just the filename (not the full path) as the name
		name := d.Name()
		files = append(files, migrationFile{name: name, sql: string(content)})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by filename (lexicographic = numeric order for 001_, 002_, ... prefix)
	sort.Slice(files, func(i, j int) bool {
		return files[i].name < files[j].name
	})

	return files, nil
}

// versionFromFilename extracts the numeric version prefix from a migration filename.
// "001_init_schema.up.sql" → 1
// "042_add_vector_index.up.sql" → 42
func versionFromFilename(name string) int {
	var version int
	if _, err := fmt.Sscanf(name, "%d_", &version); err != nil {
		return 0
	}
	return version
}

// isMigrationApplied checks if a migration version is already in schema_migrations.
func isMigrationApplied(db *sql.DB, version int) (bool, error) {
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version)
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// applyMigration executes a single migration SQL in a transaction and records it.
func applyMigration(db *sql.DB, version int, name, sqlContent string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() //nolint:errcheck // rollback on panic/error is intentional
	}()

	// Execute the migration SQL (may contain multiple statements)
	if _, execErr := tx.Exec(sqlContent); execErr != nil {
		return fmt.Errorf("exec SQL: %w", execErr)
	}

	// Record as applied
	if _, execErr := tx.Exec(
		"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
		version, name,
	); execErr != nil {
		return fmt.Errorf("record migration: %w", execErr)
	}

	return tx.Commit()
}
