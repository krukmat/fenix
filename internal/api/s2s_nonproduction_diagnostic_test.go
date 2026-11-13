//go:build s2s_nonproduction

package api

import "testing"

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
