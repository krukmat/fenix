package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

const b4HTTPEnvelope = `{"contract_version":"1","operation_id":"operation_00000001","capability_key":"integration.diagnostics","input_schema_version":"1","trace_id":"trace_b4_http","source":{"product":"dubbridge","org_id":"123e4567-e89b-42d3-a456-426614174000","project_id":"123e4567-e89b-42d3-a456-426614174001"},"input":{"v":1}}`

type b4HTTPInputSchema struct{}

func (b4HTTPInputSchema) ValidateS2SInput(
	_ context.Context, _ integration.S2SCapabilityBinding, input map[string]any,
) error {
	if len(input) != 1 || input["v"] != json.Number("1") {
		return errors.New("invalid test capability payload")
	}
	return nil
}

type b4HTTPDatabaseFixture struct {
	db       *sql.DB
	handler  http.Handler
	token    string
	creds    *domainauth.ServiceCredentialStore
	registry *integration.S2SCapabilityRegistry
}

func b4NewHTTPDatabaseFixture(t *testing.T) b4HTTPDatabaseFixture {
	t.Helper()
	db := mustOpenAPITestDB(t)
	if _, err := db.Exec(`INSERT INTO workspace(id,name,slug,created_at,updated_at)
        VALUES('b4-http-w1','HTTP workspace','b4-http','2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	identity := domainauth.ServiceIdentity{
		ServicePrincipal: domainauth.ServicePrincipal{
			ID: "b4-http-principal", Consumer: "dubbridge", WorkspaceID: "b4-http-w1",
		},
		DubBridgeOrgID: "123e4567-e89b-42d3-a456-426614174000",
		DubBridgeProjectID: "123e4567-e89b-42d3-a456-426614174001",
	}
	creds := domainauth.NewServiceCredentialStore(db)
	if err := creds.RegisterServicePrincipal(context.Background(), identity, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	token, digest, err := domainauth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := creds.StoreServiceCredential(context.Background(), "b4-http-cred", identity.ID,
		digest, now.Add(-time.Minute), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	registry := integration.NewS2SCapabilityRegistry()
	if err := registry.Register(integration.S2SCapabilityBinding{
		PrincipalID: identity.ID, WorkspaceID: identity.WorkspaceID,
		CapabilityKey: "integration.diagnostics", InputSchemaVersion: "1",
		HandlerID: "test-diagnostic-only", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	admission := &integration.S2SAdmission{
		Credentials: creds, Registry: registry, InputSchema: b4HTTPInputSchema{},
		Operations: integration.NewServiceOperationStore(db),
	}
	return b4HTTPDatabaseFixture{
		db: db, handler: NewDormantS2SAdmissionHandler(admission), token: token,
		creds: creds, registry: registry,
	}
}

func (f b4HTTPDatabaseFixture) send(t *testing.T, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, DormantS2SAdmissionPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Idempotency-Key", key)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	return w
}

func (f b4HTTPDatabaseFixture) count(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow("SELECT COUNT(*) FROM service_operation WHERE principal_id='b4-http-principal'").
		Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestB4S2SHTTPPersistedCredentialKeyMismatchBeforeReserve(t *testing.T) {
	f := b4NewHTTPDatabaseFixture(t)
	w := f.send(t, "operation_00000009", b4HTTPEnvelope)
	v := testS2SBody(t, w)
	if w.Code != http.StatusBadRequest || v["error_code"] != "INVALID_REQUEST" || f.count(t) != 0 {
		t.Fatalf("mismatched header reserved a row: %d %+v", w.Code, v)
	}
}

func TestB4S2SHTTPPersistedCredentialNewReplayConflict(t *testing.T) {
	f := b4NewHTTPDatabaseFixture(t)
	first := f.send(t, "operation_00000001", b4HTTPEnvelope)
	second := f.send(t, "operation_00000001", b4HTTPEnvelope)
	if first.Code != http.StatusAccepted || second.Code != http.StatusOK || f.count(t) != 1 {
		t.Fatalf("new/replay failed: first=%d second=%d count=%d",
			first.Code, second.Code, f.count(t))
	}
	changed := strings.Replace(b4HTTPEnvelope, `"trace_id":"trace_b4_http"`,
		`"trace_id":"trace_b4_other"`, 1)
	conflict := f.send(t, "operation_00000001", changed)
	if conflict.Code != http.StatusConflict || f.count(t) != 1 {
		t.Fatalf("conflicting replay persisted: %d %s", conflict.Code, conflict.Body.String())
	}
	var runs int
	if err := f.db.QueryRow("SELECT COUNT(*) FROM agent_run").Scan(&runs); err != nil || runs != 0 {
		t.Fatalf("HTTP S2S admission unexpectedly executed agent: %d %v", runs, err)
	}
}

func TestB4S2SHTTPRevokedAndDisabledPrincipalDeny(t *testing.T) {
	for _, disable := range []bool{false, true} {
		t.Run(map[bool]string{false: "revoked", true: "disabled"}[disable], func(t *testing.T) {
			f := b4NewHTTPDatabaseFixture(t)
			if disable {
				if err := f.registry.Disable("b4-http-principal", "b4-http-w1",
					"integration.diagnostics"); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := f.creds.RevokeServicePrincipal(context.Background(),
					"b4-http-principal", time.Now().UTC()); err != nil {
					t.Fatal(err)
				}
			}
			w := f.send(t, "operation_00000001", b4HTTPEnvelope)
			if w.Code != http.StatusForbidden || f.count(t) != 0 {
				t.Fatalf("revoked/disabled allowed HTTP reservation: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestB4S2SHTTPForgedScopeAndCapabilityInputDeny(t *testing.T) {
	for _, tc := range []struct{
		name string
		raw string
		status int
	}{
		{"wrong organization", strings.Replace(b4HTTPEnvelope, "123e4567-e89b-42d3-a456-426614174000",
			"123e4567-e89b-42d3-a456-426614174022", 1), 403},
		{"wrong project", strings.Replace(b4HTTPEnvelope, "123e4567-e89b-42d3-a456-426614174001",
			"123e4567-e89b-42d3-a456-426614174023", 1), 403},
		{"invalid schema input", strings.Replace(b4HTTPEnvelope, `"input":{"v":1}`,
			`"input":{"v":2}`, 1), 403},
		{"unknown root field", strings.Replace(b4HTTPEnvelope, `"input":{"v":1}`,
			`"request_sha256":"forged","input":{"v":1}`, 1), 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := b4NewHTTPDatabaseFixture(t)
			w := f.send(t, "operation_00000001", tc.raw)
			if w.Code != tc.status || f.count(t) != 0 {
				t.Fatalf("unsafe HTTP envelope accepted: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
