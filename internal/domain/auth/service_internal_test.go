package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite/sqlcgen"
	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

func TestInsertWorkspaceAndUser_RollsBackBootstrapFailure(t *testing.T) {
	t.Parallel()

	db := mustOpenInternalDB(t)
	injectedErr := errors.New("injected bootstrap failure")
	svc := &authService{
		db: db,
		bootstrapRegistration: func(ctx context.Context, q *sqlcgen.Queries, p insertParams, now string) error {
			if err := q.CreateRole(ctx, sqlcgen.CreateRoleParams{
				ID:          uuid.NewV7().String(),
				WorkspaceID: p.workspaceID,
				Name:        defaultRoleName,
				Description: strPtr(defaultRoleDescription),
				Permissions: defaultWorkspaceOwnerPermissions,
				CreatedAt:   now,
				UpdatedAt:   now,
			}); err != nil {
				return err
			}
			return injectedErr
		},
	}

	err := svc.insertWorkspaceAndUser(context.Background(), insertParams{
		workspaceID:   uuid.NewV7().String(),
		userID:        uuid.NewV7().String(),
		workspaceName: "Rollback Workspace",
		email:         "rollback@example.com",
		passwordHash:  "hashed-password",
		displayName:   "Rollback User",
	})
	if !errors.Is(err, injectedErr) {
		t.Fatalf("insertWorkspaceAndUser() error = %v; want injected bootstrap failure", err)
	}

	for _, table := range []string{"workspace", "user_account", "role", "user_role", "pipeline", "pipeline_stage"} {
		if got := countInternalRows(t, db, table); got != 0 {
			t.Fatalf("%s rows after failed bootstrap = %d; want 0", table, got)
		}
	}
}

func mustOpenInternalDB(t *testing.T) *sql.DB {
	t.Helper()

	db := mustCreateInternalSQLite(t)
	mustApplyInternalMigrations(t, db)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func mustCreateInternalSQLite(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "auth-internal-test.sqlite")
	opened, openErr := sqlite.NewDB(dbPath)
	if openErr != nil {
		t.Fatalf("sqlite.NewDB(%q) error = %v", dbPath, openErr)
	}
	return opened
}

func mustApplyInternalMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	if migrateErr := sqlite.MigrateUp(db); migrateErr != nil {
		t.Fatalf("MigrateUp error = %v", migrateErr)
	}
}

func countInternalRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	allowedTables := map[string]struct{}{
		"workspace":      {},
		"user_account":   {},
		"role":           {},
		"user_role":      {},
		"pipeline":       {},
		"pipeline_stage": {},
	}
	if _, ok := allowedTables[table]; !ok {
		t.Fatalf("unsupported count table %q", table)
	}

	var n int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}
