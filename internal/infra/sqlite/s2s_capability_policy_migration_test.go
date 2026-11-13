package sqlite_test

import (
	"crypto/sha256"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func TestB2B4Migration040DeniedByDefaultAndAuditImmutable(t *testing.T) {
	db := mustOpenDB(t)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO workspace(id,name,slug,created_at,updated_at)
        VALUES('s2s-w','S2S Test','s2s-test','2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO service_principal
        (id,consumer,workspace_id,dubbridge_org_id,created_at_unix)
        VALUES('s2s-p','dubbridge','s2s-w','org1',2000000000),
              ('s2s-revoked','dubbridge','s2s-w','org2',2000000000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE service_principal SET status='revoked',revoked_at_unix=2000000005
        WHERE id='s2s-revoked'`); err != nil {
		t.Fatal(err)
	}
	rawSchema := `{"type":"object","properties":{"v":{"type":"integer"}},"required":["v"],"additionalProperties":false}`
	digest := sha256.Sum256([]byte(rawSchema))
	provision := `INSERT INTO service_s2s_capability_policy
        (principal_id,workspace_id,capability_key,schema_version,handler_id,
         schema_json,schema_sha256,updated_by,created_at_unix,updated_at_unix)
        VALUES(?,?,?,?,?,?,?,?,?,?)`
	provisionArgs := []any{"s2s-p", "s2s-w", "integration.diagnostics", "1", "diagnostics-handler",
		rawSchema, digest[:], "operator-test", int64(2000000010), int64(2000000010)}
	if _, err := db.Exec(provision, provisionArgs...); err != nil {
		t.Fatal(err)
	}
	var enabled int
	if err := db.QueryRow(`SELECT enabled FROM service_s2s_capability_policy
        WHERE principal_id='s2s-p'`).Scan(&enabled); err != nil || enabled != 0 {
		t.Fatalf("provision activated capability: enabled=%d err=%v", enabled, err)
	}
	var events int
	if err := db.QueryRow("SELECT COUNT(*) FROM service_s2s_capability_audit").Scan(&events); err != nil || events != 1 {
		t.Fatalf("provision not audited: events=%d err=%v", events, err)
	}
	for _, mutation := range []string{
		"UPDATE service_s2s_capability_policy SET handler_id='forged'",
		"UPDATE service_s2s_capability_policy SET schema_json='{}'",
		"UPDATE service_s2s_capability_policy SET schema_sha256=zeroblob(32)",
		"UPDATE service_s2s_capability_policy SET revision=revision+1",
		"DELETE FROM service_s2s_capability_policy",
		"UPDATE service_s2s_capability_audit SET action='enable'",
		"DELETE FROM service_s2s_capability_audit",
	} {
		if _, err := db.Exec(mutation); err == nil {
			t.Fatalf("forbidden policy/audit mutation accepted: %s", mutation)
		}
	}
	if _, err := db.Exec(`UPDATE service_s2s_capability_policy
        SET enabled=1,revision=2,updated_by='operator-test',updated_at_unix=2000000020
        WHERE principal_id='s2s-p' AND schema_version='1'`); err != nil {
		t.Fatal(err)
	}
	v2 := append([]any(nil), provisionArgs...)
	v2[3] = "2"
	if _, err := db.Exec(provision, v2...); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE service_s2s_capability_policy
        SET enabled=1,revision=2,updated_by='operator-test',updated_at_unix=2000000020
        WHERE principal_id='s2s-p' AND schema_version='2'`); err == nil {
		t.Fatal("two schema versions enabled for same capability")
	}
	if _, err := db.Exec(`UPDATE service_s2s_capability_policy
        SET enabled=0,revision=3,updated_by='operator-test',updated_at_unix=2000000030
        WHERE principal_id='s2s-p' AND schema_version='1'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE service_s2s_capability_policy
        SET enabled=1,revision=2,updated_by='operator-test',updated_at_unix=2000000040
        WHERE principal_id='s2s-p' AND schema_version='2'`); err != nil {
		t.Fatal(err)
	}
	v3 := append([]any(nil), provisionArgs...)
	v3[0] = "s2s-revoked"
	if _, err := db.Exec(provision, v3...); err == nil {
		t.Fatal("revoked principal could provision capability")
	}
	if _, err := db.Exec("SELECT 1 FROM service_s2s_capability_audit LIMIT 1"); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM service_s2s_capability_audit").Scan(&events); err != nil || events != 5 {
		t.Fatalf("missing atomic policy audit events: count=%d err=%v", events, err)
	}
}
