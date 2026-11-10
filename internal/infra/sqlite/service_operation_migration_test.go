package sqlite_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func TestB2CServiceOperationSchemaConstraints(t *testing.T) {
	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace(id,name,slug,created_at,updated_at)
        VALUES ('b2c-w1','One','b2c-one','2026-01-01','2026-01-01'),
               ('b2c-w2','Two','b2c-two','2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO service_principal
        (id,consumer,workspace_id,dubbridge_org_id,dubbridge_project_id,created_at_unix,status,revoked_at_unix)
        VALUES ('b2c-p1','dubbridge','b2c-w1','org-1','project-1',2000000000,'active',NULL),
               ('b2c-p2','dubbridge','b2c-w2','org-2',NULL,2000000000,'active',NULL),
               ('b2c-p3','dubbridge','b2c-w1','org-1','project-1',2000000000,'revoked',2000000010)`); err != nil {
		t.Fatal(err)
	}
	insert := `INSERT INTO service_operation
        (principal_id,operation_id,workspace_id,dubbridge_org_id,dubbridge_project_id,
         capability_key,schema_version,request_sha256,config_sha256,trace_id,
         created_at_unix,updated_at_unix) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
	base := []any{"b2c-p1", "operation_00000001", "b2c-w1", "org-1", "project-1",
		"integration.diagnostics", "1", bytes.Repeat([]byte{1}, 32),
		bytes.Repeat([]byte{2}, 32), "trace_12345678", int64(2000000000), int64(2000000000)}
	if _, err := db.Exec(insert, base...); err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct{ index int; value any }{
		{0, "b2c-p3"}, {2, "b2c-w2"}, {3, "org-2"}, {4, ""},
		{7, []byte{1}}, {1, "short"},
	} {
		params := append([]any(nil), base...)
		params[1] = "operation_00000002"
		params[change.index] = change.value
		if _, err := db.Exec(insert, params...); err == nil {
			t.Fatalf("invalid operation accepted: changed column %d", change.index)
		}
	}
	for _, change := range []string{
		"UPDATE service_operation SET dubbridge_org_id='other'",
		"UPDATE service_operation SET request_sha256=zeroblob(32)",
		"UPDATE service_operation SET status='running'",
		"UPDATE service_operation SET run_id='fabricated-run'",
		"UPDATE service_operation SET status='running',run_id=''",
		"UPDATE service_operation SET lease_owner='worker'",
	} {
		if _, err := db.Exec(change); err == nil {
			t.Fatalf("invalid mutation accepted: %s", change)
		}
	}
	if _, err := db.Exec(`UPDATE service_operation SET status='indeterminate',
        updated_at_unix=2000000010 WHERE principal_id='b2c-p1'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE service_operation SET status='accepted'
        WHERE principal_id='b2c-p1'`); err == nil {
		t.Fatal("ambiguous terminal operation reaccepted")
	}
	if _, err := db.Exec(insert, append([]any{"b2c-p2", "operation_00000001", "b2c-w2", "org-2", "",
		"integration.diagnostics", "1", bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32),
		"trace_12345678", int64(2000000000), int64(2000000000)})...); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM service_operation").Scan(&count); err != nil || count != 2 {
		t.Fatalf("principal-scoped uniqueness count=%d err=%v", count, err)
	}
}

func TestB2CServiceOperationLocalRollback(t *testing.T) {
	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version=39").Scan(&n); err != nil || n != 1 {
		t.Fatalf("migration 39 missing: %d %v", n, err)
	}
	down, err := os.ReadFile("migrations/039_service_operation.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='service_operation'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("operation table survived local rollback: %d %v", n, err)
	}
	assertTableExists(t, db, "service_principal")
	assertTableExists(t, db, "workspace")
}
