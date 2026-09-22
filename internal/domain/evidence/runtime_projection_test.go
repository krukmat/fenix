package evidence

import (
	"errors"
	"testing"
)

func TestRuntimeResultFromProofProjectsVerifiedEvidence(t *testing.T) {
	envelope := validEnvelope()
	ref := validProofReference()

	result, err := RuntimeResultFromProof(envelope, ref)
	if err != nil {
		t.Fatalf("RuntimeResultFromProof: %v", err)
	}
	if result.State != string(DeliveryVerified) {
		t.Fatalf("state = %q", result.State)
	}
	if result.AuditMetadata["execution_id"] != envelope.ExecutionID {
		t.Fatalf("audit metadata = %#v", result.AuditMetadata)
	}
}

func TestRuntimeResultFromProofRejectsMismatchedExecution(t *testing.T) {
	envelope := validEnvelope()
	ref := validProofReference()
	ref.ExecutionID = "exec-other"

	result, err := RuntimeResultFromProof(envelope, ref)
	if !errors.Is(err, ErrRuntimeProofMismatch) {
		t.Fatalf("error = %v", err)
	}
	if result.State != string(DeliveryIndeterminate) {
		t.Fatalf("state = %q", result.State)
	}
}
