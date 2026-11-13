//go:build s2s_nonproduction

package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"unicode/utf8"
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
