package sqlite_test

import (
	"os"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func TestServiceIdentityMigrationAndLocalRollback(t *testing.T) {
	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"service_principal", "service_credential", "service_identity_audit"} {
		assertTableExists(t, db, name)
	}
	var version int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=38`).Scan(&version); err != nil || version != 1 {
		t.Fatalf("migration version 38: %d %v", version, err)
	}
	down, err := os.ReadFile("migrations/038_service_identity.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("local down migration: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table'
        AND name IN ('service_principal','service_credential','service_identity_audit')`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("S2S tables remain after rollback: %d %v", count, err)
	}
	assertTableExists(t, db, "workspace")
}
