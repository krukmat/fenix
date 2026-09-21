package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

type runtimeSinkStub struct {
	ref         ProofReference
	recordErr   error
	lookupRef   *ProofReference
	lookupErr   error
	recordCalls int
	lookupCalls int
	envelopes   []Envelope
}

func (s *runtimeSinkStub) RecordEvidence(
	_ context.Context,
	envelope Envelope,
) (ProofReference, error) {
	s.recordCalls++
	s.envelopes = append(s.envelopes, envelope)
	return s.ref, s.recordErr
}

func (s *runtimeSinkStub) LookupEvidence(
	_ context.Context,
	_, _ string,
) (*ProofReference, error) {
	s.lookupCalls++
	return s.lookupRef, s.lookupErr
}

func runtimeEvidenceRequest() tool.CapabilityEvidenceRequest {
	return tool.CapabilityEvidenceRequest{
		WorkspaceID: "ws-1",
		TraceID:     "trace-1",
		ExecutionID: testExecutionID,
		RunID:       "run-1",
		ActorID:     "user-1",
		ActorType:   "user",
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
			EvidenceReason:      "test evidence",
			PolicyReference:     "policy:test",
		},
		Status: tool.CapabilityStatusSucceeded,
		Input:  json.RawMessage(`{"secret":"input"}`),
		Output: json.RawMessage(`{"secret":"output"}`),
	}
}

func runtimeProof(status VerificationStatus) ProofReference {
	ref := validProofReference()
	ref.ExecutionID = testExecutionID
	ref.StreamID = "workspace/ws-1"
	ref.VerificationStatus = status
	if status == VerificationRecorded || status == VerificationPendingCheckpoint {
		ref.Checkpoint = nil
	}
	return ref
}

func TestBuildRuntimeEnvelope_MinimizesProviderPayload(t *testing.T) {
	request := runtimeEvidenceRequest()
	envelope, err := BuildRuntimeEnvelope(request)
	if err != nil {
		t.Fatalf("BuildRuntimeEnvelope returned error: %v", err)
	}
	if envelope.ExecutionID != request.ExecutionID || envelope.TraceID != request.TraceID {
		t.Fatalf("correlation was not preserved: %#v", envelope)
	}
	if envelope.InputDigest == nil || envelope.OutputDigest == nil {
		t.Fatal("input/output digests are required for non-empty runtime payloads")
	}
	if envelope.InputDigest.Value == string(request.Input) ||
		envelope.OutputDigest.Value == string(request.Output) {
		t.Fatal("raw provider payload leaked into digest fields")
	}
	if len(envelope.Authorization.ContextHash) != 64 {
		t.Fatalf("context hash length = %d", len(envelope.Authorization.ContextHash))
	}
}

func TestRuntimeRecorder_ReconcilesIndeterminateAppendByExecutionID(t *testing.T) {
	ref := runtimeProof(VerificationVerified)
	sink := &runtimeSinkStub{
		recordErr: NewIndeterminateRecordError(errors.New("lost response")),
		lookupRef: &ref,
	}
	recorder := NewRuntimeRecorder(sink)

	result, err := recorder.RecordCapabilityEvidence(context.Background(), runtimeEvidenceRequest())
	if err != nil {
		t.Fatalf("RecordCapabilityEvidence returned error: %v", err)
	}
	if sink.recordCalls != 1 || sink.lookupCalls != 1 {
		t.Fatalf("record=%d lookup=%d", sink.recordCalls, sink.lookupCalls)
	}
	if result.State != string(DeliveryVerified) {
		t.Fatalf("state = %q, want verified", result.State)
	}
	if result.AuditMetadata["execution_id"] != testExecutionID {
		t.Fatalf("unexpected audit projection: %#v", result.AuditMetadata)
	}
}

func TestRuntimeRecorder_DeterministicRecordFailureDoesNotLookup(t *testing.T) {
	sink := &runtimeSinkStub{recordErr: errors.New("invalid evidence")}
	recorder := NewRuntimeRecorder(sink)

	_, err := recorder.RecordCapabilityEvidence(context.Background(), runtimeEvidenceRequest())
	if err == nil {
		t.Fatal("expected deterministic evidence error")
	}
	if sink.lookupCalls != 0 {
		t.Fatalf("deterministic failure triggered %d lookups", sink.lookupCalls)
	}
}

func TestRuntimeRecorder_ExposesVerificationFailureWithoutRetry(t *testing.T) {
	sink := &runtimeSinkStub{ref: runtimeProof(VerificationFailed)}
	recorder := NewRuntimeRecorder(sink)

	result, err := recorder.RecordCapabilityEvidence(context.Background(), runtimeEvidenceRequest())
	if err != nil {
		t.Fatalf("RecordCapabilityEvidence returned error: %v", err)
	}
	if result.State != string(DeliveryVerificationFailed) {
		t.Fatalf("state = %q, want verification_failed", result.State)
	}
	if sink.recordCalls != 1 || sink.lookupCalls != 0 {
		t.Fatalf("record=%d lookup=%d", sink.recordCalls, sink.lookupCalls)
	}
}
