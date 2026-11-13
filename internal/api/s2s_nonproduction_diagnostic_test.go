//go:build s2s_nonproduction

package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

func TestB4W2NonProductionDiagnosticIsDeterministicAndBounded(t *testing.T) {
	h := NonProductionS2SDiagnosticHandler{}
	one, err := h.Handle(map[string]any{"nonce": "fixed-case"})
	if err != nil || len(one.Digest) != 64 {
		t.Fatalf("diagnostic rejected valid input: %v", err)
	}
	replay, err := h.Handle(map[string]any{"nonce": "fixed-case"})
	if err != nil || replay != one {
		t.Fatalf("diagnostic changed across repeats: %v", err)
	}
	changed, err := h.Handle(map[string]any{"nonce": "different"})
	if err != nil || changed.Digest == one.Digest {
		t.Fatal("diagnostic digest failed to bind input")
	}
	for _, invalid := range []map[string]any{
		nil, {}, {"nonce": ""}, {"nonce": 7},
		{"nonce": "a", "extra": true},
		{"nonce": string([]byte{0xff})},
		{"nonce": "12345678901234567890123456789012345678901234567890123456789012345"},
	} {
		if _, err := h.Handle(invalid); err == nil {
			t.Fatalf("invalid diagnostic input was accepted: %+v", invalid)
		}
	}
}


func TestB4W2DiagnosticHarnessRequiresAdmissionBeforeExecution(t *testing.T) {
	h := NonProductionS2SDiagnosticHandler{}
	raw := []byte(`{"capability_key":"integration.diagnostics","input_schema_version":"1","input":{"nonce":"fixture"}}`)
	fake := &fakeS2SHTTPAdmission{
		result: integration.S2SAdmissionReceipt{OperationID: "fixture-operation", TraceID: "fixture-trace"},
	}
	receipt, output, err := h.AdmitAndHandle(
		context.Background(), fake, "opaque-bearer", raw, "fixture-operation", time.Now().UTC(),
	)
	if err != nil || receipt.OperationID != "fixture-operation" || len(output.Digest) != 64 || fake.calls != 1 {
		t.Fatalf("authorized diagnostic failed: receipt=%+v err=%v calls=%d", receipt, err, fake.calls)
	}

	fake.calls = 0
	fake.err = integration.ErrS2SAdmissionDenied
	receipt, output, err = h.AdmitAndHandle(
		context.Background(), fake, "revoked-bearer", raw, "fixture-operation", time.Now().UTC(),
	)
	if !errors.Is(err, integration.ErrS2SAdmissionDenied) ||
		receipt.OperationID != "" || output.Digest != "" || fake.calls != 1 {
		t.Fatalf("diagnostic ran after denial: receipt=%+v output=%+v err=%v", receipt, output, err)
	}

	fake.calls = 0
	fake.err = nil
	wrongKey := []byte(`{"capability_key":"workflow.run","input_schema_version":"1","input":{"nonce":"fixture"}}`)
	_, _, err = h.AdmitAndHandle(
		context.Background(), fake, "opaque-bearer", wrongKey, "fixture-operation", time.Now().UTC(),
	)
	if err == nil || fake.calls != 0 {
		t.Fatalf("wrong capability reached admission: err=%v calls=%d", err, fake.calls)
	}
}
