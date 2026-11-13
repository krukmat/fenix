//go:build s2s_nonproduction

package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/matiasleandrokruk/fenix/internal/domain/integration"
)

// NonProductionDiagnosticCapabilityKey is an infrastructure-only test seam.
// No business capability, workflow, tool, agent or operator policy is registered.
const NonProductionDiagnosticCapabilityKey = "integration.diagnostics"
const nonProductionDiagnosticHandlerID = "b4w2-diagnostics-v1"

var errNonProductionDiagnosticInput = errors.New("invalid diagnostic input")

// NonProductionS2SDiagnosticHandler is a pure, deterministic test-only handler.
// It MUST be called only by a separately operator-authorized, schema-validated
// nonproduction harness AFTER Fenix's persisted S2S admission succeeds.
// This function never reserves, dispatches, registers or returns an agent run.
type NonProductionS2SDiagnosticHandler struct{}

type NonProductionDiagnosticResult struct {
	Digest string
}

// Handle computes a domain-separated digest without exposing a request or
// secret. No HTTP router calls this handler, even under the build tag.
func (NonProductionS2SDiagnosticHandler) Handle(input map[string]any) (NonProductionDiagnosticResult, error) {
	if len(input) != 1 {
		return NonProductionDiagnosticResult{}, errNonProductionDiagnosticInput
	}
	nonce, ok := input["nonce"].(string)
	if !ok || !utf8.ValidString(nonce) || utf8.RuneCountInString(nonce) < 1 ||
		utf8.RuneCountInString(nonce) > 64 {
		return NonProductionDiagnosticResult{}, errNonProductionDiagnosticInput
	}
	digest := sha256.Sum256([]byte("fenix-b4w2-diagnostic-v1\x00" + nonce))
	return NonProductionDiagnosticResult{Digest: hex.EncodeToString(digest[:])}, nil
}


// AdmitAndHandle is a nonproduction-only diagnostic *harness seam*, not an
// HTTP handler. It intentionally cannot authorize itself: the supplied
// S2SHTTPAdmission must first authenticate a persisted Fenix service
// principal, validate the operator-enabled exact capability policy and input
// schema, and reserve the durable operation. The diagnostic runs only AFTER
// successful admission, never on an unknown/disabled capability or denial.
// No normal Fenix router or production service references this function.
func (h NonProductionS2SDiagnosticHandler) AdmitAndHandle(
	ctx context.Context, admission S2SHTTPAdmission, bearer string, raw []byte,
	operationID string, now time.Time,
) (integration.S2SAdmissionReceipt, NonProductionDiagnosticResult, error) {
	if admission == nil || len(raw) == 0 || len(raw) > int(maxS2SAdmissionJSONBytes) {
		return integration.S2SAdmissionReceipt{}, NonProductionDiagnosticResult{}, errNonProductionDiagnosticInput
	}
	var envelope struct {
		CapabilityKey string         `json:"capability_key"`
		SchemaVersion string         `json:"input_schema_version"`
		Input         map[string]any `json:"input"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil ||
		envelope.CapabilityKey != NonProductionDiagnosticCapabilityKey ||
		envelope.SchemaVersion != "1" {
		return integration.S2SAdmissionReceipt{}, NonProductionDiagnosticResult{}, errNonProductionDiagnosticInput
	}
	receipt, err := admission.AdmitWithIdempotencyKey(ctx, bearer, raw, operationID, now)
	if err != nil {
		return integration.S2SAdmissionReceipt{}, NonProductionDiagnosticResult{}, err
	}
	result, err := h.Handle(envelope.Input)
	if err != nil {
		return integration.S2SAdmissionReceipt{}, NonProductionDiagnosticResult{}, err
	}
	return receipt, result, nil
}
