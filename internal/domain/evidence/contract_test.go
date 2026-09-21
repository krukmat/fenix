package evidence

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

const testDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func validEnvelope() EvidenceEnvelope {
	return EvidenceEnvelope{
		SchemaVersion: SchemaVersion,
		StreamID:      "workspace/ws-1",
		WorkspaceID:   "ws-1",
		TraceID:       "trace-1",
		ExecutionID:   "exec-1",
		RunID:         "run-1",
		Actor:         ActorRef{ID: "user-1", Type: "user"},
		Capability: CapabilityRef{
			Name:            "salesforce.flow.export",
			Version:         "1",
			Operation:       "export",
			SideEffectClass: tool.SideEffectTransform,
		},
		Authorization: AuthorizationEvidence{
			PolicyID:      "fenix.policy",
			PolicyVersion: "7",
			Decision:      PolicyAllow,
			Reason:        "allowed by Fenix policy",
			ContextHash:   testDigest,
		},
		Approval: &ApprovalEvidence{
			ApprovalID: "approval-1",
			Decision:   "APPROVED",
		},
		Outcome:    ExecutionOutcome{Status: OutcomeSucceeded},
		InputDigest: &DigestRef{Algorithm: "sha256", Value: testDigest},
		OccurredAt: time.Now().UTC(),
	}
}

func TestEvidenceEnvelope_ValidatesMinimalVerifiableContext(t *testing.T) {
	envelope := validEnvelope()
	if err := envelope.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestEvidenceEnvelope_RejectsFullPayloadInsteadOfDigestContract(t *testing.T) {
	envelope := validEnvelope()
	envelope.InputDigest.Value = "not-a-digest"
	if err := envelope.Validate(); !errors.Is(err, ErrEnvelopeInvalid) {
		t.Fatalf("expected ErrEnvelopeInvalid, got %v", err)
	}
}

func TestEvidenceEnvelope_RequiresExecutionCorrelation(t *testing.T) {
	envelope := validEnvelope()
	envelope.ExecutionID = ""
	if err := envelope.Validate(); !errors.Is(err, ErrEnvelopeInvalid) {
		t.Fatalf("expected ErrEnvelopeInvalid, got %v", err)
	}
}

func TestProofReference_RequiresExecutionCorrelationAndEventEvidence(t *testing.T) {
	ref := ProofReference{
		SchemaVersion:      SchemaVersion,
		Provider:           "verifiable-event-ledger",
		ExecutionID:        "exec-1",
		StreamID:           "workspace/ws-1",
		EventID:            "event-1",
		EventHash:          testDigest,
		KeyID:              "key-1",
		Sequence:           1,
		VerificationStatus: VerificationRecorded,
	}
	if err := ref.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	ref.ExecutionID = ""
	if err := ref.Validate(); !errors.Is(err, ErrProofReferenceInvalid) {
		t.Fatalf("expected ErrProofReferenceInvalid, got %v", err)
	}
}

func TestProofReference_ValidatesCheckpointShape(t *testing.T) {
	ref := ProofReference{
		SchemaVersion: SchemaVersion,
		Provider:      "verifiable-event-ledger",
		ExecutionID:   "exec-1",
		StreamID:      "workspace/ws-1",
		EventID:       "event-1",
		EventHash:     testDigest,
		KeyID:         "key-1",
		Sequence:      1,
		Checkpoint: &CheckpointReference{
			CheckpointID:   "checkpoint-1",
			CheckpointHash: testDigest,
			MerkleRoot:     strings.Repeat("b", 64),
			TreeSize:       1,
		},
		VerificationStatus: VerificationVerified,
	}
	if err := ref.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}
