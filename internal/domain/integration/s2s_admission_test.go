package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

const (
	b2eOrgID = "123e4567-e89b-42d3-a456-426614174000"
	b2eProjectID = "123e4567-e89b-42d3-a456-426614174001"
)

type b2eInputValidator struct{}

func (b2eInputValidator) ValidateS2SInput(
	_ context.Context, _ integration.S2SCapabilityBinding, input map[string]any,
) error {
	if len(input) != 1 || input["v"] != json.Number("1") {
		return errors.New("unapproved capability input")
	}
	return nil
}

type b2eFixture struct {
	b2cFixture
	creds *domainauth.ServiceCredentialStore
	token string
}

func newB2EFixture(t *testing.T) b2eFixture {
	t.Helper()
	f := newB2CFixture(t)
	now := time.Now().UTC()
	creds := domainauth.NewServiceCredentialStore(f.db)
	identity := domainauth.ServiceIdentity{
		ServicePrincipal: domainauth.ServicePrincipal{
			ID: "b2e-principal", Consumer: "dubbridge", WorkspaceID: "b2c-w1",
		},
		DubBridgeOrgID: b2eOrgID, DubBridgeProjectID: b2eProjectID,
	}
	if err := creds.RegisterServicePrincipal(context.Background(), identity, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	token, digest, err := domainauth.GenerateServiceToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := creds.StoreServiceCredential(context.Background(), "b2e-credential",
		identity.ID, digest, now.Add(-time.Minute), now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := f.registry.Register(integration.S2SCapabilityBinding{
		PrincipalID: identity.ID, WorkspaceID: identity.WorkspaceID,
		CapabilityKey: "integration.diagnostics", InputSchemaVersion: "1",
		HandlerID: "test-diagnostic-only", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	return b2eFixture{b2cFixture: f, creds: creds, token: token}
}

func (f b2eFixture) admission() *integration.S2SAdmission {
	return &integration.S2SAdmission{
		Credentials: f.creds, Registry: f.registry,
		InputSchema: b2eInputValidator{}, Operations: f.store,
	}
}

func b2eEnvelope() string {
	return `{"contract_version":"1","operation_id":"operation_00000001","capability_key":"integration.diagnostics","input_schema_version":"1","trace_id":"trace_b2e_test","source":{"product":"dubbridge","org_id":"123e4567-e89b-42d3-a456-426614174000","project_id":"123e4567-e89b-42d3-a456-426614174001"},"input":{"v":1}}`
}

func b2eCount(t *testing.T, f b2eFixture) int {
	t.Helper()
	var count int
	if err := f.db.QueryRow("SELECT COUNT(*) FROM service_operation WHERE principal_id='b2e-principal'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestB2EAdmissionReservesAndReplaysWithoutExecution(t *testing.T) {
	f := newB2EFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	admit := f.admission()
	first, err := admit.Admit(ctx, f.token, []byte(b2eEnvelope()), now)
	if err != nil || first.Replayed || first.OperationID != "operation_00000001" {
		t.Fatalf("first receipt: %+v err=%v", first, err)
	}
	second, err := admit.Admit(ctx, f.token, []byte(b2eEnvelope()), now.Add(time.Second))
	if err != nil || !second.Replayed || second.OperationID != first.OperationID {
		t.Fatalf("replay receipt: %+v err=%v", second, err)
	}
	if b2eCount(t, f) != 1 {
		t.Fatal("idempotent replay created a second row")
	}
	var runs int
	if err := f.db.QueryRow("SELECT COUNT(*) FROM agent_run").Scan(&runs); err != nil || runs != 0 {
		t.Fatalf("admission dispatched an agent: runs=%d err=%v", runs, err)
	}
	if strings.Contains(fmt.Sprintf("%+v", first), "b2e-principal") {
		t.Fatal("receipt exposed persisted identity")
	}
}

func TestB2EAdmissionRejectsMalformedAndForgedEnvelope(t *testing.T) {
	base := b2eEnvelope()
	cases := []struct {
		name string
		raw string
		wantErr error
	}{
		{"unknown field", strings.Replace(base, `"input":{"v":1}`, `"other":1,"input":{"v":1}`, 1), integration.ErrS2SAdmissionInvalid},
		{"missing required", strings.Replace(base, `"contract_version":"1",`, "", 1), integration.ErrS2SAdmissionInvalid},
		{"wrong version", strings.Replace(base, `"contract_version":"1"`, `"contract_version":"2"`, 1), integration.ErrS2SAdmissionInvalid},
		{"invalid UUID", strings.Replace(base, b2eOrgID, "not-a-uuid", 1), integration.ErrS2SAdmissionInvalid},
		{"forged org", strings.Replace(base, b2eOrgID, "123e4567-e89b-42d3-a456-426614174022", 1), integration.ErrS2SAdmissionDenied},
		{"forged project", strings.Replace(base, b2eProjectID, "123e4567-e89b-42d3-a456-426614174023", 1), integration.ErrS2SAdmissionDenied},
		{"missing project", strings.Replace(base, `,"project_id":"`+b2eProjectID+`"`, "", 1), integration.ErrS2SAdmissionDenied},
		{"wrong product", strings.Replace(base, `"product":"dubbridge"`, `"product":"other"`, 1), integration.ErrS2SAdmissionInvalid},
		{"extra source field", strings.Replace(base, `"product":"dubbridge"`, `"product":"dubbridge","workspace_id":"forged"`, 1), integration.ErrS2SAdmissionInvalid},
		{"null input", strings.Replace(base, `"input":{"v":1}`, `"input":null`, 1), integration.ErrS2SAdmissionInvalid},
		{"duplicate key", strings.Replace(base, `"input":{"v":1}`, `"input":{"v":1,"v":2}`, 1), integration.ErrS2SAdmissionInvalid},
		{"lone surrogate", strings.Replace(base, `"input":{"v":1}`, `"input":{"v":"\ud800"}`, 1), integration.ErrS2SAdmissionInvalid},
		{"caller digest", strings.Replace(base, `"input":{"v":1}`, `"request_sha256":"fake","input":{"v":1}`, 1), integration.ErrS2SAdmissionInvalid},
		{"unregistered version", strings.Replace(base, `"input_schema_version":"1"`, `"input_schema_version":"2"`, 1), integration.ErrS2SAdmissionDenied},
		{"unregistered capability", strings.Replace(base, `"capability_key":"integration.diagnostics"`, `"capability_key":"integration.unknown"`, 1), integration.ErrS2SAdmissionDenied},
		{"input schema denied", strings.Replace(base, `"input":{"v":1}`, `"input":{"v":2}`, 1), integration.ErrS2SAdmissionDenied},
		{"oversized JSON", base + strings.Repeat(" ", 64<<10), integration.ErrS2SAdmissionInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newB2EFixture(t)
			receipt, err := f.admission().Admit(context.Background(), f.token, []byte(tc.raw), time.Now().UTC())
			if !errors.Is(err, tc.wantErr) || receipt.OperationID != "" || b2eCount(t, f) != 0 {
				t.Fatalf("invalid request reserved: receipt=%+v err=%v", receipt, err)
			}
		})
	}
}

func TestB2EAdmissionDeniesDisabledRegistryAndRevokedIdentity(t *testing.T) {
	f := newB2EFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	raw := []byte(b2eEnvelope())
	for _, token := range []string{"", "invalid-service-token"} {
		if _, err := f.admission().Admit(ctx, token, raw, now); !errors.Is(err, integration.ErrS2SAdmissionDenied) {
			t.Fatalf("invalid bearer admitted: %v", err)
		}
	}
	nilSchema := f.admission()
	nilSchema.InputSchema = nil
	if _, err := nilSchema.Admit(ctx, f.token, raw, now); !errors.Is(err, integration.ErrS2SAdmissionDenied) {
		t.Fatalf("nil schema admitted: %v", err)
	}
	if err := f.registry.Disable("b2e-principal", "b2c-w1", "integration.diagnostics"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.admission().Admit(ctx, f.token, raw, now); !errors.Is(err, integration.ErrS2SAdmissionDenied) {
		t.Fatalf("disabled capability admitted: %v", err)
	}
	if err := f.creds.RevokeServicePrincipal(ctx, "b2e-principal", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.admission().Admit(ctx, f.token, raw, now.Add(2*time.Second)); !errors.Is(err, integration.ErrS2SAdmissionDenied) {
		t.Fatalf("revoked principal admitted: %v", err)
	}
	if b2eCount(t, f) != 0 {
		t.Fatal("denied admission changed journal")
	}
}

func TestB2EAdmissionConflictsWithoutRewritingJournal(t *testing.T) {
	f := newB2EFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := f.admission().Admit(ctx, f.token, []byte(b2eEnvelope()), now); err != nil {
		t.Fatal(err)
	}
	changedTrace := strings.Replace(b2eEnvelope(), `"trace_id":"trace_b2e_test"`, `"trace_id":"trace_b2e_changed"`, 1)
	if _, err := f.admission().Admit(ctx, f.token, []byte(changedTrace), now); !errors.Is(err, integration.ErrS2SAdmissionConflict) {
		t.Fatalf("changed trace replayed: %v", err)
	}
	if b2eCount(t, f) != 1 {
		t.Fatal("conflict rewrote journal")
	}
}

func TestB2EPublicErrorProjectionNeverLeaksDetails(t *testing.T) {
	for _, tc := range []struct {
		err error
		code string
	}{
		{fmt.Errorf("bearer-sensitive: %w", integration.ErrS2SAdmissionDenied), "REQUEST_DENIED"},
		{fmt.Errorf("private-payload: %w", integration.ErrS2SAdmissionInvalid), "INVALID_REQUEST"},
		{fmt.Errorf("internal-scope: %w", integration.ErrS2SAdmissionConflict), "OPERATION_CONFLICT"},
		{integration.ErrS2SAdmissionUnavailable, "PROVIDER_UNAVAILABLE"},
		{errors.New("database /secret principal"), "PROVIDER_UNAVAILABLE"},
	} {
		p := integration.ProjectS2SAdmissionError(tc.err)
		if p.Code != tc.code || p.Message != "Request could not be accepted" {
			t.Fatalf("unsafe error projection: %+v", p)
		}
	}
}
