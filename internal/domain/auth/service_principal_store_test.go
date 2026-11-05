package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type credentialDriver struct {
	mu                sync.Mutex
	secret            []byte
	active            bool
	created           bool
	credentialActive  bool
	credentialExpires int64
	issued            int64
	revoked           bool
	errorRead         bool
	rows              int64
	lastStmt          string
	lastArgs          []driver.NamedValue
}
type credentialConn struct{ m *credentialDriver }
type credentialRows struct {
	cols []string
	vals []driver.Value
	done bool
}

func (r *credentialRows) Columns() []string { return r.cols }
func (r *credentialRows) Close() error      { return nil }
func (r *credentialRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	copy(dest, r.vals)
	r.done = true
	return nil
}
func (c *credentialConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *credentialConn) Close() error                         { return nil }
func (c *credentialConn) Begin() (driver.Tx, error)            { return nil, errors.New("unexpected transaction") }
func (d *credentialDriver) Open(_ string) (driver.Conn, error) { return &credentialConn{m: d}, nil }
func (c *credentialConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	m := c.m
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastStmt = query
	m.lastArgs = append([]driver.NamedValue(nil), args...)
	switch {
	case strings.Contains(query, "INSERT INTO service_principal"):
		if len(args) != 6 || args[1].Value != "dubbridge" {
			return nil, errors.New("bad principal")
		}
		m.created = true
		m.active = true
		m.rows = 1
	case strings.Contains(query, "INSERT INTO service_credential"):
		if !m.active {
			m.rows = 0
			break
		}
		raw, ok := args[1].Value.([]byte)
		if !ok {
			return nil, errors.New("expected digest BLOB")
		}
		m.secret = append([]byte(nil), raw...)
		m.issued = args[2].Value.(int64)
		m.credentialExpires = args[3].Value.(int64)
		m.credentialActive = true
		m.rows = 1
	case strings.Contains(query, "UPDATE service_credential"):
		if m.credentialActive {
			m.credentialActive = false
			m.rows = 1
		} else {
			m.rows = 0
		}
	case strings.Contains(query, "UPDATE service_principal"):
		if m.active {
			m.active = false
			m.rows = 1
		} else {
			m.rows = 0
		}
	default:
		return nil, errors.New("unexpected Exec SQL")
	}
	return driver.RowsAffected(m.rows), nil
}
func (c *credentialConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	m := c.m
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastStmt = query
	m.lastArgs = append([]driver.NamedValue(nil), args...)
	if m.errorRead {
		return nil, errors.New("injected unavailable")
	}
	if !strings.Contains(query, "service_credential c JOIN service_principal p") {
		return nil, errors.New("unexpected Query SQL")
	}
	got, ok := args[0].Value.([]byte)
	now := args[1].Value.(int64)
	if !ok || len(got) != 32 || !m.active || !m.credentialActive || now < m.issued || now >= m.credentialExpires {
		return &credentialRows{cols: []string{"id", "consumer", "workspace_id", "dubbridge_org_id", "dubbridge_project_id", "token_sha256"}, done: true}, nil
	}
	if string(got) != string(m.secret) {
		return &credentialRows{cols: []string{"id", "consumer", "workspace_id", "dubbridge_org_id", "dubbridge_project_id", "token_sha256"}, done: true}, nil
	}
	return &credentialRows{cols: []string{"id", "consumer", "workspace_id", "dubbridge_org_id", "dubbridge_project_id", "token_sha256"}, vals: []driver.Value{"p1", "dubbridge", "w1", "org1", nil, append([]byte(nil), m.secret...)}}, nil
}

var driverIndex int
var driverMu sync.Mutex

func fakeStore(t *testing.T) (*ServiceCredentialStore, *credentialDriver) {
	t.Helper()
	driverMu.Lock()
	driverIndex++
	name := "credential-test-" + string(rune('a'+driverIndex))
	driverMu.Unlock()
	m := &credentialDriver{}
	sql.Register(name, m)
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return NewServiceCredentialStore(db), m
}
func TestServiceCredentialStoreLifecycle(t *testing.T) {
	ctx := context.Background()
	repo, backend := fakeStore(t)
	now := time.Unix(2000000000, 0)
	ident := ServiceIdentity{ServicePrincipal: ServicePrincipal{ID: "p1", Consumer: "dubbridge", WorkspaceID: "w1"}, DubBridgeOrgID: "org1"}
	if err := repo.RegisterServicePrincipal(ctx, ident, now); err != nil {
		t.Fatal(err)
	}
	secret, digest, err := GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.StoreServiceCredential(ctx, "c1", "p1", digest, now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	backend.mu.Lock()
	if string(backend.secret) == secret {
		t.Error("plaintext stored")
	}
	backend.mu.Unlock()
	p, err := repo.AuthenticateServiceToken(ctx, secret, now.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "p1" || p.WorkspaceID != "w1" || p.DubBridgeOrgID != "org1" || p.DubBridgeProjectID != "" {
		t.Fatalf("unexpected verified identity: %+v", p)
	}
	if _, err = repo.AuthenticateServiceToken(ctx, secret, now.Add(time.Hour)); !errors.Is(err, ErrServiceIdentityDenied) {
		t.Fatalf("expected expiration: %v", err)
	}
	if _, err = repo.AuthenticateServiceToken(ctx, "aaa.bbb.ccc", now); !errors.Is(err, ErrServiceIdentityDenied) {
		t.Fatalf("must reject human JWT: %v", err)
	}
	if err = repo.RevokeServiceCredential(ctx, "c1", now.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.AuthenticateServiceToken(ctx, secret, now.Add(40*time.Second)); !errors.Is(err, ErrServiceIdentityDenied) {
		t.Fatalf("expected revoked credential denial: %v", err)
	}
	if err = repo.RevokeServiceCredential(ctx, "c1", now.Add(time.Minute)); !errors.Is(err, ErrServiceIdentityNotFound) {
		t.Fatalf("expected already revoked: %v", err)
	}
	if err = repo.RevokeServicePrincipal(ctx, "p1", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err = repo.RevokeServicePrincipal(ctx, "p1", now.Add(2*time.Minute)); !errors.Is(err, ErrServiceIdentityNotFound) {
		t.Fatalf("expected already revoked: %v", err)
	}
}
func TestServiceCredentialStoreDenyInvalidAndUnavailable(t *testing.T) {
	ctx := context.Background()
	repo, backend := fakeStore(t)
	now := time.Unix(2000000000, 0)
	if err := repo.RegisterServicePrincipal(ctx, ServiceIdentity{}, now); !errors.Is(err, ErrServiceIdentityConfig) {
		t.Fatal(err)
	}
	zero := ServiceTokenDigest{}
	if err := repo.StoreServiceCredential(ctx, "c", "p", zero, now, now.Add(time.Hour)); !errors.Is(err, ErrServiceIdentityConfig) {
		t.Fatal(err)
	}
	secret, digest, err := GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.StoreServiceCredential(ctx, "c", "missing", digest, now, now.Add(time.Hour)); !errors.Is(err, ErrServiceIdentityNotFound) {
		t.Fatalf("must fail closed on missing principal: %v", err)
	}
	backend.mu.Lock()
	backend.errorRead = true
	backend.mu.Unlock()
	if _, err = repo.AuthenticateServiceToken(ctx, secret, now); !errors.Is(err, ErrServiceIdentityUnavailable) {
		t.Fatalf("unavailable must differ from rejected: %v", err)
	}
	if _, err = repo.AuthenticateServiceToken(ctx, secret+"!", now); !errors.Is(err, ErrServiceIdentityDenied) {
		t.Fatalf("malformed: %v", err)
	}
	if _, err = repo.AuthenticateServiceToken(ctx, secret, time.Time{}); !errors.Is(err, ErrServiceIdentityDenied) {
		t.Fatal(err)
	}
	var missing *ServiceCredentialStore
	if _, err = missing.AuthenticateServiceToken(ctx, secret, now); !errors.Is(err, ErrServiceIdentityUnavailable) {
		t.Fatal(err)
	}
}
func TestCredentialStoreSQLHasRequiredGuards(t *testing.T) {
	ctx := context.Background()
	repo, m := fakeStore(t)
	now := time.Unix(2000000000, 0)
	id := ServiceIdentity{ServicePrincipal: ServicePrincipal{ID: "p1", Consumer: "dubbridge", WorkspaceID: "w1"}, DubBridgeOrgID: "org1"}
	if err := repo.RegisterServicePrincipal(ctx, id, now); err != nil {
		t.Fatal(err)
	}
	secret, digest, err := GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.StoreServiceCredential(ctx, "c1", "p1", digest, now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	p, err := repo.AuthenticateServiceToken(ctx, secret, now)
	if err != nil {
		t.Fatal(err)
	}
	_ = p
	m.mu.Lock()
	sqlText := m.lastStmt
	params := append([]driver.NamedValue(nil), m.lastArgs...)
	m.mu.Unlock()
	for _, clause := range []string{"c.revoked_at_unix IS NULL", "c.expires_at_unix>?", "p.status='active'", "p.consumer='dubbridge'"} {
		if !strings.Contains(sqlText, clause) {
			t.Fatalf("missing SQL guard: %s", clause)
		}
	}
	if len(params) != 3 {
		t.Fatalf("expected digest + time params: %d", len(params))
	}
	want := sha256.Sum256([]byte(secret))
	if got := params[0].Value.([]byte); string(got) != string(want[:]) {
		t.Fatal("must bind digest, not secret")
	}
}
