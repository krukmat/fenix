package evidenceoutbox

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/evidence"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
)

type lifecycleSinkStub struct {
	ref               evidence.ProofReference
	recordErr         error
	lookupRef         *evidence.ProofReference
	lookupErr         error
	progress          evidence.VerificationProgress
	verificationErr   error
	recordCalls       int
	lookupCalls       int
	verificationCalls int
}

func (s *lifecycleSinkStub) RecordEvidence(
	_ context.Context,
	_ evidence.Envelope,
) (evidence.ProofReference, error) {
	s.recordCalls++
	return s.ref, s.recordErr
}

func (s *lifecycleSinkStub) LookupEvidence(
	_ context.Context,
	_, _ string,
) (*evidence.ProofReference, error) {
	s.lookupCalls++
	return s.lookupRef, s.lookupErr
}

func (s *lifecycleSinkStub) CheckpointAndVerify(
	_ context.Context,
	_ string,
) (evidence.VerificationProgress, error) {
	s.verificationCalls++
	return s.progress, s.verificationErr
}

func TestRecorderRecoversIndeterminateAppendAfterRestartAndVerifies(t *testing.T) {
	db, err := sqlite.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := sqlite.MigrateUp(db); err != nil {
		t.Fatalf("MigrateUp: %v", err)
	}

	firstSink := &lifecycleSinkStub{
		recordErr: evidence.NewIndeterminateRecordError(errors.New("lost append response")),
	}
	first, err := NewRecorder(db, firstSink)
	if err != nil {
		t.Fatalf("NewRecorder first: %v", err)
	}
	first.retryDelay = 0

	result, err := first.RecordCapabilityEvidence(context.Background(), durableEvidenceRequest())
	if err != nil {
		t.Fatalf("RecordCapabilityEvidence: %v", err)
	}
	if result.State != string(evidence.DeliveryIndeterminate) {
		t.Fatalf("state = %q", result.State)
	}
	if firstSink.recordCalls != 1 || firstSink.lookupCalls != 1 {
		t.Fatalf("initial record=%d lookup=%d", firstSink.recordCalls, firstSink.lookupCalls)
	}

	ref := durableProof()
	secondSink := &lifecycleSinkStub{
		lookupRef: &ref,
		progress:  durableVerificationProgress(),
	}
	second, err := NewRecorder(db, secondSink)
	if err != nil {
		t.Fatalf("NewRecorder second: %v", err)
	}
	second.retryDelay = 0
	if err := second.ReconcileOnce(context.Background()); err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	if secondSink.recordCalls != 0 {
		t.Fatalf("restart reconciliation replayed append %d times", secondSink.recordCalls)
	}
	if secondSink.lookupCalls != 1 || secondSink.verificationCalls != 1 {
		t.Fatalf("restart lookup=%d verification=%d", secondSink.lookupCalls, secondSink.verificationCalls)
	}

	var state string
	var proofRaw string
	if err := db.QueryRow(
		"SELECT delivery_state, proof_json FROM evidence_delivery WHERE execution_id = ?",
		"exec-1",
	).Scan(&state, &proofRaw); err != nil {
		t.Fatalf("query durable row: %v", err)
	}
	if state != string(evidence.DeliveryVerified) {
		t.Fatalf("durable state = %q", state)
	}
	if !strings.Contains(proofRaw, ""verification_status":"verified"") {
		t.Fatalf("proof not verified: %s", proofRaw)
	}
}

func durableEvidenceRequest() tool.CapabilityEvidenceRequest {
	return tool.CapabilityEvidenceRequest{
		WorkspaceID: "ws-1",
		TraceID:     "trace-1",
		ExecutionID: "exec-1",
		RunID:       "run-1",
		ActorID:     "agent-1",
		ActorType:   "agent",
		Descriptor: tool.CapabilityDescriptor{
			Name:            "salesforce.flow.export",
			Version:         "1",
			Operation:       "export",
			SideEffectClass: tool.SideEffectTransform,
		},
		Governance: tool.CapabilityRuntimeDecision{
			Allowed:             true,
			EvidenceRequirement: tool.RuntimeEvidenceOptional,
			EvidencePlanned:     true,
			EvidenceReason:      "test",
			PolicyReference:     "fenix:w1-policy-gate",
		},
		Status: tool.CapabilityStatusSucceeded,
	}
}

func durableProof() evidence.ProofReference {
	return evidence.ProofReference{
		SchemaVersion:      evidence.SchemaVersion,
		Provider:           "vel-http",
		ExecutionID:        "exec-1",
		StreamID:           "workspace/ws-1",
		EventID:            "event-1",
		EventHash:          strings.Repeat("a", 64),
		KeyID:              "key-1",
		SignatureRef:       "vel://events/event-1#signature",
		Sequence:           1,
		VerificationStatus: evidence.VerificationRecorded,
	}
}

func durableVerificationProgress() evidence.VerificationProgress {
	return evidence.VerificationProgress{
		Checkpoint: evidence.CheckpointReference{
			CheckpointID:   "checkpoint-1",
			CheckpointHash: strings.Repeat("b", 64),
			MerkleRoot:     strings.Repeat("c", 64),
			TreeSize:       1,
		},
		Status: evidence.VerificationVerified,
	}
}
