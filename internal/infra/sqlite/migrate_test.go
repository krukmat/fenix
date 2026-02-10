// Task 1.2.5: TDD tests for the migration system (written before implementation)
package sqlite_test

import (
	"database/sql"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

// TestMigrate_RunsAllMigrations verifies that MigrateUp applies all pending migrations.
func TestMigrate_RunsAllMigrations(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)

	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v; want nil", err)
	}

	// After migration, schema_migrations table must exist with at least 1 row
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM schema_migrations")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("SELECT COUNT(*) FROM schema_migrations error = %v", err)
	}

	if count == 0 {
		t.Error("schema_migrations has 0 rows after MigrateUp; want > 0")
	}
}

// TestMigrate_Idempotent verifies that running MigrateUp twice does not fail.
// Migrations must be idempotent — re-running on an already-migrated DB is safe.
func TestMigrate_Idempotent(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)

	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() first run error = %v; want nil", err)
	}

	// Second run must not fail (already-applied migrations are skipped)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() second run error = %v; want nil (idempotent)", err)
	}
}

// TestMigrate_WorkspaceTableCreated verifies the workspace table exists after migration.
func TestMigrate_WorkspaceTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "workspace")
}

// TestMigrate_UserAccountTableCreated verifies the user_account table exists.
func TestMigrate_UserAccountTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "user_account")
}

// TestMigrate_RoleTableCreated verifies the role table exists.
func TestMigrate_RoleTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "role")
}

// TestMigrate_UserRoleTableCreated verifies the user_role table exists.
func TestMigrate_UserRoleTableCreated(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	assertTableExists(t, db, "user_role")
}

// TestMigrate_ForeignKeyConstraintEnforced verifies that FK constraints are active.
// Inserting a user_account with a non-existent workspace_id must fail.
func TestMigrate_ForeignKeyConstraintEnforced(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	// This insert references a workspace_id that does not exist — must fail
	_, err := db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES ('user-1', 'nonexistent-workspace', 'test@example.com', 'Test User', 'active', datetime('now'), datetime('now'))
	`)

	if err == nil {
		t.Error("INSERT with non-existent workspace_id succeeded; want FK constraint error")
	}
}

// TestMigrate_WorkspaceSlugUnique verifies the UNIQUE constraint on workspace.slug.
func TestMigrate_WorkspaceSlugUnique(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-1', 'Workspace One', 'acme', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("first workspace insert error = %v", err)
	}

	// Duplicate slug — must fail
	_, err = db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-2', 'Workspace Two', 'acme', datetime('now'), datetime('now'))
	`)

	if err == nil {
		t.Error("duplicate slug INSERT succeeded; want UNIQUE constraint error")
	}
}

// TestMigrate_UserEmailUnique verifies the UNIQUE constraint on user_account.email.
func TestMigrate_UserEmailUnique(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	_, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-1', 'Test Workspace', 'test', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("workspace insert error = %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES ('user-1', 'ws-1', 'alice@example.com', 'Alice', 'active', datetime('now'), datetime('now'))
	`)
	if err != nil {
		t.Fatalf("first user insert error = %v", err)
	}

	// Duplicate email — must fail
	_, err = db.Exec(`
		INSERT INTO user_account (id, workspace_id, email, display_name, status, created_at, updated_at)
		VALUES ('user-2', 'ws-1', 'alice@example.com', 'Alice 2', 'active', datetime('now'), datetime('now'))
	`)

	if err == nil {
		t.Error("duplicate email INSERT succeeded; want UNIQUE constraint error")
	}
}

// TestMigrate_Version returns the current applied migration version.
func TestMigrate_Version(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	version, err := sqlite.MigrationVersion(db)
	if err != nil {
		t.Fatalf("MigrationVersion() error = %v; want nil", err)
	}

	if version == 0 {
		t.Error("MigrationVersion() = 0; want > 0 after MigrateUp")
	}
}

// TestMigrate_OnlyAppliesPending verifies that already-applied migrations are NOT re-run.
func TestMigrate_OnlyAppliesPending(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() first error = %v", err)
	}

	var countBefore int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&countBefore); err != nil {
		t.Fatalf("count before: %v", err)
	}

	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() second error = %v", err)
	}

	var countAfter int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&countAfter); err != nil {
		t.Fatalf("count after: %v", err)
	}

	if countAfter != countBefore {
		t.Errorf("schema_migrations count changed from %d to %d; want unchanged", countBefore, countAfter)
	}
}

// TestMigrationVersion_NoMigrations verifies version is 0 on fresh DB.
func TestMigrationVersion_NoMigrations(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	// Do NOT call MigrateUp — fresh DB

	version, err := sqlite.MigrationVersion(db)
	if err != nil {
		t.Fatalf("MigrationVersion() error = %v", err)
	}

	if version != 0 {
		t.Errorf("MigrationVersion() = %d; want 0 on fresh DB", version)
	}
}

// TestMigrate_RoleUniquePerWorkspace verifies UNIQUE(workspace_id, name) on role.
func TestMigrate_RoleUniquePerWorkspace(t *testing.T) {
	t.Parallel()

	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO workspace (id, name, slug, created_at, updated_at)
		VALUES ('ws-1', 'Test', 'test', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("workspace insert: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO role (id, workspace_id, name, created_at, updated_at)
		VALUES ('r-1', 'ws-1', 'admin', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("first role insert: %v", err)
	}

	// Duplicate role name in same workspace — must fail
	_, err := db.Exec(`
		INSERT INTO role (id, workspace_id, name, created_at, updated_at)
		VALUES ('r-2', 'ws-1', 'admin', datetime('now'), datetime('now'))
	`)
	if err == nil {
		t.Error("duplicate role name in same workspace succeeded; want UNIQUE constraint error")
	}
}

// assertTableExists fails the test if the given table doesn't exist in the DB.
func assertTableExists(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()

	var name string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
		tableName,
	).Scan(&name)

	if err == sql.ErrNoRows {
		t.Errorf("table %q not found in sqlite_master after MigrateUp", tableName)
		return
	}
	if err != nil {
		t.Fatalf("assertTableExists(%q) query error = %v", tableName, err)
	}
}
