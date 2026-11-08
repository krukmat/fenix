package auth_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/auth"
	fxsqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

func serviceStoreDB(t *testing.T) (*sql.DB, *auth.ServiceCredentialStore) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "service-identity.db")
	db, err := fxsqlite.NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := fxsqlite.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO workspace(id,name,slug,created_at,updated_at)
        VALUES ('w1','One','one','2026-01-01','2026-01-01'),
               ('w2','Two','two','2026-01-01','2026-01-01')`)
	if err != nil {
		t.Fatal(err)
	}
	return db, auth.NewServiceCredentialStore(db)
}

func registerService(t *testing.T, store *auth.ServiceCredentialStore, id, workspace, org string, at time.Time) {
	t.Helper()
	identity := auth.ServiceIdentity{
		ServicePrincipal: auth.ServicePrincipal{ID: id, Consumer: "dubbridge", WorkspaceID: workspace},
		DubBridgeOrgID:   org,
	}
	if err := store.RegisterServicePrincipal(context.Background(), identity, at); err != nil {
		t.Fatal(err)
	}
}

func issueServiceToken(t *testing.T, store *auth.ServiceCredentialStore, credentialID, principalID string, at time.Time) string {
	t.Helper()
	secret, digest, err := auth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.StoreServiceCredential(context.Background(), credentialID, principalID, digest, at, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	return secret
}

func assertDenied(t *testing.T, store *auth.ServiceCredentialStore, secret string, at time.Time) {
	t.Helper()
	_, err := store.AuthenticateServiceToken(context.Background(), secret, at)
	if !errors.Is(err, auth.ErrServiceIdentityDenied) {
		t.Fatalf("want denied; got %v", err)
	}
}

func TestServiceCredentialSQLiteLifecycle(t *testing.T) {
	db, store := serviceStoreDB(t)
	ctx := context.Background()
	now := time.Unix(2_000_000_000, 0)
	registerService(t, store, "p1", "w1", "org-one", now)
	registerService(t, store, "p2", "w2", "org-two", now)
	secret := issueServiceToken(t, store, "c1", "p1", now)
	other := issueServiceToken(t, store, "c2", "p2", now)

	identity, err := store.AuthenticateServiceToken(ctx, secret, now)
	if err != nil || identity.ID != "p1" || identity.WorkspaceID != "w1" || identity.DubBridgeOrgID != "org-one" {
		t.Fatalf("wrong trusted scope: identity=%+v err=%v", identity, err)
	}
	otherIdentity, err := store.AuthenticateServiceToken(ctx, other, now)
	if err != nil || otherIdentity.WorkspaceID != "w2" || otherIdentity.DubBridgeOrgID != "org-two" {
		t.Fatalf("cross-workspace scope: identity=%+v err=%v", otherIdentity, err)
	}
	assertDenied(t, store, secret, now.Add(-time.Second))
	assertDenied(t, store, secret, now.Add(time.Hour))
	assertDenied(t, store, "aaa.bbb.ccc", now)

	var allowlist string
	if err := db.QueryRow(`SELECT capability_allowlist_json FROM service_principal WHERE id='p1'`).Scan(&allowlist); err != nil || allowlist != "[]" {
		t.Fatalf("capability allowlist must be empty: %q %v", allowlist, err)
	}
	if _, err := db.Exec(`UPDATE service_principal SET capability_allowlist_json='["unsafe"]' WHERE id='p1'`); err == nil {
		t.Fatal("capability grant must be impossible in this schema")
	}
	if _, err := db.Exec(`INSERT INTO service_principal(id,consumer,workspace_id,dubbridge_org_id,created_at_unix)
        VALUES('orphan','dubbridge','missing','org',?)`, now.Unix()); err == nil {
		t.Fatal("workspace foreign key not enforced")
	}
	if err := store.RevokeServiceCredential(ctx, "c1", now.Add(-5*time.Second)); err != nil {
		t.Fatalf("clock-skew credential revoke: %v", err)
	}
	assertDenied(t, store, secret, now.Add(5*time.Second))
	if err := store.RevokeServicePrincipal(ctx, "p2", now.Add(-5*time.Second)); err != nil {
		t.Fatalf("clock-skew principal revoke: %v", err)
	}
	assertDenied(t, store, other, now.Add(5*time.Second))
	if err := store.RevokeServiceCredential(ctx, "c1", now); !errors.Is(err, auth.ErrServiceIdentityNotFound) {
		t.Fatalf("duplicate revoke: %v", err)
	}
	var revoked int64
	if err := db.QueryRow(`SELECT revoked_at_unix FROM service_credential WHERE id='c1'`).Scan(&revoked); err != nil || revoked != now.Unix() {
		t.Fatalf("revocation timestamp violates issue constraint: %d %v", revoked, err)
	}
	var events int
	if err := db.QueryRow(`SELECT COUNT(*) FROM service_identity_audit WHERE principal_id='p1'`).Scan(&events); err != nil || events != 3 {
		t.Fatalf("principal audit linkage: %d %v", events, err)
	}
	if _, err := db.Exec(`DELETE FROM service_identity_audit WHERE principal_id='p1'`); err == nil {
		t.Fatal("service identity audit must be append-only")
	}
	path := dbPath(t, db)
	for _, file := range []string{path, path + "-wal"} {
		raw, readErr := os.ReadFile(file)
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			t.Fatal(readErr)
		}
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatal("plaintext bearer persisted")
		}
	}
}

func dbPath(t *testing.T, db *sql.DB) string {
	t.Helper()
	var path string
	if err := db.QueryRow(`PRAGMA database_list`).Scan(new(int), new(string), &path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestServiceCredentialConcurrentRotationAndRollback(t *testing.T) {
	db, store := serviceStoreDB(t)
	ctx := context.Background()
	now := time.Unix(2_000_000_000, 0)
	registerService(t, store, "p1", "w1", "org-one", now)
	old := issueServiceToken(t, store, "initial", "p1", now)

	const rotators = 5
	secrets := make([]string, rotators)
	digests := make([]auth.ServiceTokenDigest, rotators)
	for i := range secrets {
		var err error
		secrets[i], digests[i], err = auth.GenerateServiceToken()
		if err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	errs := make(chan error, rotators)
	for i := range secrets {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- store.RotateServiceCredential(ctx, "rotation-"+string(rune('a'+i)), "p1",
				digests[i], now.Add(time.Duration(i+1)*time.Second), now.Add(time.Hour))
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent rotation: %v", err)
		}
	}
	assertDenied(t, store, old, now.Add(10*time.Second))
	var live int
	if err := db.QueryRow(`SELECT COUNT(*) FROM service_credential WHERE principal_id='p1' AND revoked_at_unix IS NULL`).Scan(&live); err != nil || live != 1 {
		t.Fatalf("active credential count = %d; err=%v", live, err)
	}
	var accepted string
	for _, secret := range secrets {
		if _, err := store.AuthenticateServiceToken(ctx, secret, now.Add(10*time.Second)); err == nil {
			accepted = secret
		}
	}
	if accepted == "" {
		t.Fatal("no rotated credential authenticates")
	}
	var acceptedCount int
	for _, secret := range secrets {
		if _, err := store.AuthenticateServiceToken(ctx, secret, now.Add(10*time.Second)); err == nil {
			acceptedCount++
		}
	}
	if acceptedCount != 1 {
		t.Fatalf("accepted rotated tokens = %d", acceptedCount)
	}

	_, duplicateDigest, err := auth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RotateServiceCredential(ctx, "initial", "p1", duplicateDigest, now.Add(11*time.Second), now.Add(time.Hour)); !errors.Is(err, auth.ErrServiceIdentityUnavailable) {
		t.Fatalf("duplicate ID must roll back: %v", err)
	}
	if _, err := store.AuthenticateServiceToken(ctx, accepted, now.Add(12*time.Second)); err != nil {
		t.Fatalf("prior token lost on rollback: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM service_credential WHERE principal_id='p1' AND revoked_at_unix IS NULL`).Scan(&live); err != nil || live != 1 {
		t.Fatalf("rollback active count = %d; err=%v", live, err)
	}
	var maxVersion int
	if err := db.QueryRow(`SELECT MAX(version) FROM service_credential WHERE principal_id='p1'`).Scan(&maxVersion); err != nil || maxVersion != rotators+1 {
		t.Fatalf("version sequence = %d; err=%v", maxVersion, err)
	}
	var rotations int
	if err := db.QueryRow(`SELECT COUNT(*) FROM service_identity_audit WHERE action='rotate_credential'`).Scan(&rotations); err != nil || rotations != rotators {
		t.Fatalf("rollback audit count = %d; err=%v", rotations, err)
	}
}

func TestServiceCredentialSanitizedFailures(t *testing.T) {
	_, store := serviceStoreDB(t)
	ctx := context.Background()
	now := time.Unix(2_000_000_000, 0)
	secret, digest, err := auth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.StoreServiceCredential(ctx, "missing", "missing", digest, now, now.Add(time.Hour)); !errors.Is(err, auth.ErrServiceIdentityNotFound) {
		t.Fatalf("missing principal: %v", err)
	}
	if _, err := store.AuthenticateServiceToken(ctx, secret, now); !errors.Is(err, auth.ErrServiceIdentityDenied) {
		t.Fatalf("missing credential: %v", err)
	}
	if err := store.RegisterServicePrincipal(ctx, auth.ServiceIdentity{}, now); !errors.Is(err, auth.ErrServiceIdentityConfig) {
		t.Fatalf("invalid principal: %v", err)
	}
	if err := store.StoreServiceCredential(ctx, "zero", "missing", auth.ServiceTokenDigest{}, now, now.Add(time.Hour)); !errors.Is(err, auth.ErrServiceIdentityConfig) {
		t.Fatalf("zero digest: %v", err)
	}
	if strings.Contains(auth.ErrServiceIdentityDenied.Error(), secret) ||
		strings.Contains(auth.ErrServiceIdentityUnavailable.Error(), secret) {
		t.Fatal("secret leaked in public error")
	}
}
