package evidence

import "testing"

func validProofReference() ProofReference {
	return ProofReference{
		SchemaVersion: SchemaVersion,
		Provider:      "verifiable-event-ledger",
		ExecutionID:   "exec-1",
		StreamID:      "workspace/ws-1",
		EventID:       "event-1",
		EventHash:     testDigest,
		KeyID:         "key-1",
		SignatureRef:  "event:event-1#signature",
		Sequence:      1,
		Checkpoint: &CheckpointReference{
			CheckpointID:   "checkpoint-1",
			CheckpointHash: testDigest,
			MerkleRoot:     testDigest,
			TreeSize:       1,
		},
		VerificationStatus: VerificationVerified,
	}
}

func TestIdempotencyKey_UsesStableExecutionIdentity(t *testing.T) {
	envelope := validEnvelope()
	first := IdempotencyKey(envelope)
	second := IdempotencyKey(envelope)
	if first != "exec-1" || second != first {
		t.Fatalf("unexpected idempotency key: first=%q second=%q", first, second)
	}

	envelope.ExecutionID = "exec-2"
	if got := IdempotencyKey(envelope); got == first {
		t.Fatalf("different execution must produce different key: %q", got)
	}
}

func TestDecideReconciliation_NeverRepeatsBusinessAction(t *testing.T) {
	cases := map[DeliveryState]ReconciliationAction{
		DeliveryPendingRecord:      ReconcileRetryRecord,
		DeliveryRecorded:           ReconcileAwaitCheckpoint,
		DeliveryPendingCheckpoint:  ReconcileAwaitCheckpoint,
		DeliveryVerified:           ReconcileNone,
		DeliveryVerificationFailed: ReconcileEscalate,
		DeliveryIndeterminate:      ReconcileLookupExisting,
	}
	for state, expected := range cases {
		decision := DecideReconciliation(state)
		if decision.Action != expected {
			t.Fatalf("state %s action = %s, want %s", state, decision.Action, expected)
		}
		if decision.MayRepeatBusinessAction {
			t.Fatalf("state %s must never repeat business action", state)
		}
	}
}

func TestDeliveryStateFromProof_MapsVerificationLifecycle(t *testing.T) {
	ref := validProofReference()

	cases := map[VerificationStatus]DeliveryState{
		VerificationRecorded:          DeliveryRecorded,
		VerificationPendingCheckpoint: DeliveryPendingCheckpoint,
		VerificationVerified:          DeliveryVerified,
		VerificationFailed:            DeliveryVerificationFailed,
	}
	for status, expected := range cases {
		ref.VerificationStatus = status
		if got := DeliveryStateFromProof(ref); got != expected {
			t.Fatalf("status %s state = %s, want %s", status, got, expected)
		}
	}
}

func TestNewAuditProjection_StripsBundleAndIssueMessages(t *testing.T) {
	ref := validProofReference()
	ref.VerificationIssues = []string{"signature mismatch", "checkpoint mismatch"}

	projection, err := NewAuditProjection(ref)
	if err != nil {
		t.Fatalf("NewAuditProjection returned error: %v", err)
	}
	if projection.EventID != ref.EventID ||
		projection.EventHash != ref.EventHash ||
		projection.CheckpointID != ref.Checkpoint.CheckpointID ||
		projection.CheckpointHash != ref.Checkpoint.CheckpointHash ||
		projection.MerkleRoot != ref.Checkpoint.MerkleRoot {
		t.Fatalf("unexpected projection: %#v", projection)
	}
	if projection.IssueCount != 2 {
		t.Fatalf("issue count = %d, want 2", projection.IssueCount)
	}
}

func TestEvaluateForAgent_DoesNotOverclaimEvidenceIntegrity(t *testing.T) {
	ref := validProofReference()

	if got := EvaluateForAgent(DeliveryVerified, &ref).Disposition; got != AgentEvidenceUse {
		t.Fatalf("verified disposition = %q, want use", got)
	}
	if got := EvaluateForAgent(DeliveryPendingCheckpoint, &ref).Disposition; got != AgentEvidencePending {
		t.Fatalf("pending disposition = %q, want pending", got)
	}
	if got := EvaluateForAgent(DeliveryVerificationFailed, &ref).Disposition; got != AgentEvidenceAbstain {
		t.Fatalf("failed disposition = %q, want abstain", got)
	}
	if got := EvaluateForAgent(DeliveryIndeterminate, nil).Disposition; got != AgentEvidenceAbstain {
		t.Fatalf("indeterminate disposition = %q, want abstain", got)
	}

	ref.VerificationStatus = VerificationRecorded
	if got := EvaluateForAgent(DeliveryVerified, &ref).Disposition; got != AgentEvidenceAbstain {
		t.Fatalf("inconsistent verified state disposition = %q, want abstain", got)
	}
}
