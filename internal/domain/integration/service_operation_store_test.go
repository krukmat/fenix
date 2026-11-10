package integration_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
	fxsqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

type b2cFixture struct {
	db       *sql.DB
	store    *integration.ServiceOperationStore
	registry *integration.S2SCapabilityRegistry
}

func newB2CFixture(t *testing.T) b2cFixture {
	t.Helper()
	db, err := fxsqlite.NewDB(filepath.Join(t.TempDir(), "b2c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := fxsqlite.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO workspace(id,name,slug,created_at,updated_at)
        VALUES ('b2c-w1','B2C One','b2c-one','2026-01-01','2026-01-01'),
               ('b2c-w2','B2C Two','b2c-two','2026-01-01','2026-01-01')`)
	if err != nil {
		t.Fatal(err)
	}
	principals := domainauth.NewServiceCredentialStore(db)
	for _, p := range []domainauth.ServiceIdentity{
		{ServicePrincipal: domainauth.ServicePrincipal{ID: "b2c-p1", Consumer: "dubbridge", WorkspaceID: "b2c-w1"},
			DubBridgeOrgID: "org1", DubBridgeProjectID: "project1"},
		{ServicePrincipal: domainauth.ServicePrincipal{ID: "b2c-p2", Consumer: "dubbridge", WorkspaceID: "b2c-w2"},
			DubBridgeOrgID: "org2"},
	} {
		if err := principals.RegisterServicePrincipal(context.Background(), p, time.Now().Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	reg := integration.NewS2SCapabilityRegistry()
	for _, id := range []struct{ p, w string }{{"b2c-p1", "b2c-w1"}, {"b2c-p2", "b2c-w2"}} {
		err := reg.Register(integration.S2SCapabilityBinding{PrincipalID: id.p, WorkspaceID: id.w,
			CapabilityKey: "integration.diagnostics", InputSchemaVersion: "1",
			HandlerID: "test-diagnostic-only", Enabled: true})
		if err != nil {
			t.Fatal(err)
		}
	}
	return b2cFixture{db: db, store: integration.NewServiceOperationStore(db), registry: reg}
}

func b2cRequest(t *testing.T, principal, workspace, org, project, operationID string) integration.ServiceOperationRequest {
	t.Helper()
	digest, err := integration.CanonicalOperationDigest([]byte(`{"input":{"v":1},"source":{"product":"dubbridge"}}`))
	if err != nil {
		t.Fatal(err)
	}
	return integration.ServiceOperationRequest{
		Scope: integration.ServiceOperationScope{PrincipalID: principal, WorkspaceID: workspace, OrgID: org, ProjectID: project},
		OperationID: operationID, CapabilityKey: "integration.diagnostics", SchemaVersion: "1",
		TraceID: "trace_b2c_test", RequestSHA256: digest,
		// ConfigSHA256 is deliberately left zero: Reserve must derive it
		// from the active Fenix registry, never from caller-controlled input.
	}
}

func TestB2CReserveNewReplayConflictAndScope(t *testing.T) {
	f := newB2CFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	request := b2cRequest(t, "b2c-p1", "b2c-w1", "org1", "project1", "operation_00000001")
	first, replay, err := f.store.Reserve(ctx, request, f.registry, now)
	if err != nil || replay || first.Status != "accepted" || first.RunID != "" {
		t.Fatalf("first reserve: %+v replay=%v err=%v", first, replay, err)
	}
	same, replay, err := f.store.Reserve(ctx, request, f.registry, now.Add(time.Minute))
	if err != nil || !replay || same.CreatedAtUnix != first.CreatedAtUnix {
		t.Fatalf("replay changed record: %+v replay=%v err=%v", same, replay, err)
	}
	// Registry-owned configuration identity must win over any caller-supplied
	// fingerprint; a forged value cannot mutate or redefine a replay.
	forgedConfig := request
	forgedConfig.ConfigSHA256 = sha256.Sum256([]byte("caller-forged-configuration"))
	forged, replay, err := f.store.Reserve(ctx, forgedConfig, f.registry, now.Add(time.Minute))
	if err != nil || !replay || forged.ConfigSHA256 != first.ConfigSHA256 {
		t.Fatalf("caller config influenced reservation: replay=%v err=%v", replay, err)
	}
	different := request
	different.RequestSHA256 = sha256.Sum256([]byte("different"))
	if _, _, err := f.store.Reserve(ctx, different, f.registry, now); !errors.Is(err, integration.ErrServiceOperationConflict) {
		t.Fatalf("expected conflict: %v", err)
	}
	// A replay cannot silently replace the immutable trace identity even if an
	// upstream caller incorrectly supplies the original request fingerprint.
	changedTrace := request
	changedTrace.TraceID = "trace_b2c_other"
	if _, _, err := f.store.Reserve(ctx, changedTrace, f.registry, now); !errors.Is(err, integration.ErrServiceOperationConflict) {
		t.Fatalf("replayed operation with different trace identity: %v", err)
	}
	// An operator-updated handler binding changes the authoritative fingerprint,
	// even if the caller repeats the identical JSON request and operation ID.
	if err := f.registry.Register(integration.S2SCapabilityBinding{
		PrincipalID: "b2c-p1", WorkspaceID: "b2c-w1",
		CapabilityKey: "integration.diagnostics", InputSchemaVersion: "1",
		HandlerID: "new-test-handler", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.store.Reserve(ctx, request, f.registry, now); !errors.Is(err, integration.ErrServiceOperationConflict) {
		t.Fatalf("registered config fingerprint conflict: %v", err)
	}
	for _, bad := range []integration.ServiceOperationScope{
		{PrincipalID: "b2c-p1", WorkspaceID: "b2c-w2", OrgID: "org1", ProjectID: "project1"},
		{PrincipalID: "b2c-p1", WorkspaceID: "b2c-w1", OrgID: "org2", ProjectID: "project1"},
		{PrincipalID: "b2c-p1", WorkspaceID: "b2c-w1", OrgID: "org1"},
	} {
		rejected := request
		rejected.Scope = bad
		if _, _, err := f.store.Reserve(ctx, rejected, f.registry, now); !errors.Is(err, integration.ErrServiceOperationDenied) {
			t.Fatalf("forged scope accepted: %+v err=%v", bad, err)
		}
	}
	second := b2cRequest(t, "b2c-p2", "b2c-w2", "org2", "", request.OperationID)
	if _, replay, err := f.store.Reserve(ctx, second, f.registry, now); err != nil || replay {
		t.Fatalf("independent principal with same op id rejected: replay=%v err=%v", replay, err)
	}
}

func TestB2CReserveConcurrentOneRow(t *testing.T) {
	f := newB2CFixture(t)
	req := b2cRequest(t, "b2c-p1", "b2c-w1", "org1", "project1", "operation_00000002")
	var newCount int64
	var replayCount int64
	var wg sync.WaitGroup
	errorsCh := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, replay, err := f.store.Reserve(context.Background(), req, f.registry, time.Now().UTC())
			if err != nil {
				errorsCh <- err
				return
			}
			if replay {
				atomic.AddInt64(&replayCount, 1)
			} else {
				atomic.AddInt64(&newCount, 1)
			}
		}()
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Errorf("concurrent reserve failed: %v", err)
	}
	if newCount != 1 || replayCount != 19 {
		t.Fatalf("new=%d replay=%d", newCount, replayCount)
	}
}

func TestB2CCanonicalDigestLimitsAndDuplicateKeys(t *testing.T) {
	equal := []string{`{"a":1,"b":{"x":true,"y":[1,2]}}`, ` { "b":{"y":[1,2],"x":true}, "a":1 }`}
	a, err := integration.CanonicalOperationDigest([]byte(equal[0]))
	if err != nil {
		t.Fatal(err)
	}
	b, err := integration.CanonicalOperationDigest([]byte(equal[1]))
	if err != nil || a != b {
		t.Fatalf("canonical map order differs: %v", err)
	}
	// encoding/json otherwise replaces malformed UTF-8 with U+FFFD, which
	// could alias distinct inbound payload bytes to the same request digest.
	invalidUTF8 := append([]byte(`{"message":"`), byte(0xff))
	invalidUTF8 = append(invalidUTF8, []byte(`"}`)...)
	if _, err := integration.CanonicalOperationDigest(invalidUTF8); !errors.Is(err, integration.ErrServiceOperationInvalid) {
		t.Fatalf("malformed UTF-8 accepted as canonical request: %v", err)
	}
	for _, malformedSurrogate := range []string{
		`{"key":"\ud800"}`,
		`{"key":"\udfff"}`,
		`{"key":"\ud800\u0041"}`,
		`{"\udfff":"value"}`,
	} {
		if _, err := integration.CanonicalOperationDigest([]byte(malformedSurrogate)); !errors.Is(err, integration.ErrServiceOperationInvalid) {
			t.Fatalf("lone Unicode surrogate accepted: %q err=%v", malformedSurrogate, err)
		}
	}
	paired, err := integration.CanonicalOperationDigest([]byte(`{"key":"\ud83d\ude00"}`))
	if err != nil {
		t.Fatalf("valid UTF-16 surrogate pair rejected: %v", err)
	}
	literal, err := integration.CanonicalOperationDigest([]byte(`{"key":"😀"}`))
	if err != nil || paired != literal {
		t.Fatalf("valid Unicode surrogate pair did not normalize: %v", err)
	}
	if _, err := integration.CanonicalOperationDigest([]byte(`{"key":"\\ud800"}`)); err != nil {
		t.Fatalf("escaped literal backslash rejected: %v", err)
	}
	for _, raw := range []string{
		`{"a":1,"a":2}`, `{"nested":{"a":1,"a":2}}`,
		`{"a":1} {"b":2}`, `[1,2]`, `{"a":null} trailing`,
		`{}` + strings.Repeat(" ", 64<<10),
		`{"a":` + strings.Repeat("[", 33) + "1" + strings.Repeat("]", 33) + "}",
	} {
		if _, err := integration.CanonicalOperationDigest([]byte(raw)); !errors.Is(err, integration.ErrServiceOperationInvalid) {
			t.Fatalf("accepted invalid canonical input %q: %v", raw[:min(len(raw), 35)], err)
		}
	}
}

func TestB2CReserveDefaultDenyAndRevoked(t *testing.T) {
	f := newB2CFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	req := b2cRequest(t, "b2c-p1", "b2c-w1", "org1", "project1", "operation_00000003")
	empty := integration.NewS2SCapabilityRegistry()
	if _, _, err := f.store.Reserve(ctx, req, empty, now); !errors.Is(err, integration.ErrServiceOperationDenied) {
		t.Fatalf("empty registry allowed reserve: %v", err)
	}
	if _, _, err := f.store.Reserve(ctx, req, f.registry, now); err != nil {
		t.Fatal(err)
	}
	if err := f.registry.Disable(req.Scope.PrincipalID, req.Scope.WorkspaceID, req.CapabilityKey); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.store.Reserve(ctx, req, f.registry, now); !errors.Is(err, integration.ErrServiceOperationDenied) {
		t.Fatalf("disabled capability allowed replay: %v", err)
	}
}

func TestB2CReserveRejectsRevokedPrincipal(t *testing.T) {
	f := newB2CFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	req := b2cRequest(t, "b2c-p1", "b2c-w1", "org1", "project1", "operation_00000017")
	if _, _, err := f.store.Reserve(ctx, req, f.registry, now); err != nil {
		t.Fatal(err)
	}
	if err := domainauth.NewServiceCredentialStore(f.db).RevokeServicePrincipal(
		ctx, req.Scope.PrincipalID, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	// Even an exact pre-revocation replay may no longer read the stored row.
	if _, _, err := f.store.Reserve(ctx, req, f.registry, now.Add(2*time.Second)); !errors.Is(err, integration.ErrServiceOperationDenied) {
		t.Fatalf("revoked principal replay succeeded: %v", err)
	}
	newOperation := req
	newOperation.OperationID = "operation_00000018"
	if _, _, err := f.store.Reserve(ctx, newOperation, f.registry, now.Add(2*time.Second)); !errors.Is(err, integration.ErrServiceOperationDenied) {
		t.Fatalf("revoked principal accepted new operation: %v", err)
	}
	var count int
	if err := f.db.QueryRow("SELECT COUNT(*) FROM service_operation WHERE principal_id=?",
		req.Scope.PrincipalID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("revoked reservation changed journal: count=%d err=%v", count, err)
	}
}
