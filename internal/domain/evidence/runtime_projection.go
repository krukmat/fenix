package evidence

import "github.com/matiasleandrokruk/fenix/internal/domain/tool"

// RuntimeResultFromProof exposes the canonical W3/W4 projection for durable runtime adapters.
func RuntimeResultFromProof(
	envelope Envelope,
	ref ProofReference,
) (tool.CapabilityEvidenceResult, error) {
	return runtimeResultFromProof(envelope, ref)
}
