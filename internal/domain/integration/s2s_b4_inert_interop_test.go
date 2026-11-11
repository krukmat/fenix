package integration_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

// This exercises the in-process Go provider admitted by B2.E and the proposed
// B4 acknowledgement projection. It is NOT an HTTP/E2E test and dispatches
// no worker. The actual Fenix S2S transport route remains unmounted.
func TestB4InertAdmissionWireReservationReplayAndConflict(t *testing.T) {
	f := newB2EFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	raw := []byte(b2eEnvelope())
	first, err := f.admission().Admit(ctx, f.token, raw, now)
	if err != nil {
		t.Fatal(err)
	}
	firstStatus, firstWire, err := integration.ProjectS2SAdmissionWireAck(first)
	if err != nil || firstStatus != 202 || firstWire.Replayed ||
		firstWire.OperationID != "operation_00000001" ||
		firstWire.TraceID != "trace_b2e_test" {
		t.Fatalf("first operation not durably acknowledged: status=%d response=%+v err=%v",
			firstStatus, firstWire, err)
	}
	second, err := f.admission().Admit(ctx, f.token, raw, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	replayStatus, replayWire, err := integration.ProjectS2SAdmissionWireAck(second)
	if err != nil || replayStatus != 200 || !replayWire.Replayed ||
		replayWire.OperationID != firstWire.OperationID || replayWire.TraceID != firstWire.TraceID {
		t.Fatalf("replayed operation was not correlated: status=%d ack=%+v err=%v",
			replayStatus, replayWire, err)
	}
	changed := strings.Replace(b2eEnvelope(), `"trace_id":"trace_b2e_test"`,
		`"trace_id":"trace_b4_conflict"`, 1)
	if _, err = f.admission().Admit(ctx, f.token, []byte(changed), now.Add(2*time.Second)); !errors.Is(err, integration.ErrS2SAdmissionConflict) {
		t.Fatalf("changed trace was not rejected as conflict: %v", err)
	}
	status, failure := integration.ProjectS2SAdmissionWireFailure(err)
	if status != 409 || failure.ErrorCode != "OPERATION_CONFLICT" {
		t.Fatalf("conflict wire projection mismatch: status=%d failure=%+v", status, failure)
	}
	if b2eCount(t, f) != 1 {
		t.Fatal("idempotent reservation or conflict inserted duplicate operation")
	}
	var runs int
	if err := f.db.QueryRow("SELECT COUNT(*) FROM agent_run").Scan(&runs); err != nil || runs != 0 {
		t.Fatalf("inert admission dispatched an agent: count=%d err=%v", runs, err)
	}
}

func TestB4InertAdmissionWireWrongTenantUnknownCapabilityAndBearer(t *testing.T) {
	raw := b2eEnvelope()
	cases := []struct {
		name string
		bearerOverride string
		request string
		expected error
	}{
		{"wrong org", "", strings.Replace(raw, b2eOrgID,
			"123e4567-e89b-42d3-a456-426614174022", 1), integration.ErrS2SAdmissionDenied},
		{"wrong project", "", strings.Replace(raw, b2eProjectID,
			"123e4567-e89b-42d3-a456-426614174023", 1), integration.ErrS2SAdmissionDenied},
		{"unknown capability", "", strings.Replace(raw, "integration.diagnostics",
			"integration.unknown", 1), integration.ErrS2SAdmissionDenied},
		{"forged workspace", "", strings.Replace(raw, `"input":{}`,
			`"workspace_id":"another-workspace","input":{}`, 1), integration.ErrS2SAdmissionInvalid},
		{"invalid credential", "bad-token", raw, integration.ErrS2SAdmissionDenied},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newB2EFixture(t)
			bearer := f.token
			if tc.bearerOverride != "" {
				bearer = tc.bearerOverride
			}
			_, err := f.admission().Admit(
				context.Background(), bearer, []byte(tc.request), time.Now().UTC(),
			)
			if !errors.Is(err, tc.expected) {
				t.Fatalf("unexpected denial: got=%v want=%v", err, tc.expected)
			}
			status, wire := integration.ProjectS2SAdmissionWireFailure(err)
			if status != 400 && status != 403 {
				t.Fatalf("unexpected public refusal HTTP class: %d %+v", status, wire)
			}
			if wire.Message != "Request could not be accepted" || b2eCount(t, f) != 0 {
				t.Fatalf("denied request persisted or exposed internals: %+v", wire)
			}
		})
	}
}

func TestB4InertAdmissionWireRevokedOrDisabledDeniesWithoutRun(t *testing.T) {
	for _, tc := range []struct {
		name string
		revoke bool
		disable bool
	}{
		{"revoked principal", true, false},
		{"disabled capability", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newB2EFixture(t)
			ctx := context.Background()
			now := time.Now().UTC()
			if tc.revoke {
				if err := f.creds.RevokeServicePrincipal(ctx, "b2e-principal", now.Add(time.Second)); err != nil {
					t.Fatal(err)
				}
			}
			if tc.disable {
				if err := f.registry.Disable("b2e-principal", "b2c-w1", "integration.diagnostics"); err != nil {
					t.Fatal(err)
				}
			}
			effectiveNow := now
			if tc.revoke {
				effectiveNow = now.Add(2 * time.Second)
			}
			_, err := f.admission().Admit(ctx, f.token, []byte(b2eEnvelope()), effectiveNow)
			if !errors.Is(err, integration.ErrS2SAdmissionDenied) {
				t.Fatalf("blocked state admitted: %v", err)
			}
			status, wire := integration.ProjectS2SAdmissionWireFailure(err)
			if status != 403 || wire.ErrorCode != "REQUEST_DENIED" || b2eCount(t, f) != 0 {
				t.Fatalf("blocked state created operation: status=%d wire=%+v", status, wire)
			}
		})
	}
}
