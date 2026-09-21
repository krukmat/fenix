package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

// ErrRuntimeSinkUnavailable indicates that evidence was planned but no VEL adapter is configured.
var ErrRuntimeSinkUnavailable = errors.New("evidence runtime sink is unavailable")

// IndeterminateRecordError marks an append outcome that must be reconciled by execution identity.
type IndeterminateRecordError struct {
	Err error
}

func (e *IndeterminateRecordError) Error() string {
	if e == nil || e.Err == nil {
		return "evidence append outcome is indeterminate"
	}
	return e.Err.Error()
}

func (e *IndeterminateRecordError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NewIndeterminateRecordError marks an evidence append whose persistence outcome is unknown.
func NewIndeterminateRecordError(err error) error {
	return &IndeterminateRecordError{Err: err}
}

// RuntimeSink is the transport-neutral VEL recording port.
type RuntimeSink interface {
	RecordEvidence(ctx context.Context, envelope Envelope) (ProofReference, error)
	LookupEvidence(ctx context.Context, streamID, executionID string) (*ProofReference, error)
}

// RuntimeRecorder adapts the W4 tool evidence port to the W3 evidence contract.
type RuntimeRecorder struct {
	sink RuntimeSink
}

// NewRuntimeRecorder creates a recorder backed by a transport-neutral evidence sink.
func NewRuntimeRecorder(sink RuntimeSink) *RuntimeRecorder {
	return &RuntimeRecorder{sink: sink}
}

// RecordCapabilityEvidence builds minimized evidence and records it through the configured sink.
func (r *RuntimeRecorder) RecordCapabilityEvidence(
	ctx context.Context,
	request tool.CapabilityEvidenceRequest,
) (tool.CapabilityEvidenceResult, error) {
	if r == nil || r.sink == nil {
		return tool.CapabilityEvidenceResult{State: string(DeliveryIndeterminate)}, ErrRuntimeSinkUnavailable
	}
	envelope, err := BuildRuntimeEnvelope(request)
	if err != nil {
		return tool.CapabilityEvidenceResult{State: string(DeliveryIndeterminate)}, err
	}
	ref, err := r.sink.RecordEvidence(ctx, envelope)
	if err != nil {
		ref, err = r.reconcileIndeterminateRecord(ctx, envelope, err)
		if err != nil {
			return tool.CapabilityEvidenceResult{State: string(DeliveryIndeterminate)}, err
		}
	}
	projection, err := NewAuditProjection(ref)
	if err != nil {
		return tool.CapabilityEvidenceResult{State: string(DeliveryIndeterminate)}, err
	}
	return tool.CapabilityEvidenceResult{
		State:         string(DeliveryStateFromProof(ref)),
		AuditMetadata: auditProjectionMetadata(projection),
	}, nil
}

// BuildRuntimeEnvelope maps one governed runtime outcome into the W3 EvidenceEnvelope contract.
func BuildRuntimeEnvelope(request tool.CapabilityEvidenceRequest) (Envelope, error) {
	envelope := Envelope{
		SchemaVersion: SchemaVersion,
		StreamID:      "workspace/" + strings.TrimSpace(request.WorkspaceID),
		WorkspaceID:   strings.TrimSpace(request.WorkspaceID),
		TraceID:       strings.TrimSpace(request.TraceID),
		ExecutionID:   strings.TrimSpace(request.ExecutionID),
		RunID:         strings.TrimSpace(request.RunID),
		Actor: ActorRef{
			ID:   strings.TrimSpace(request.ActorID),
			Type: strings.TrimSpace(request.ActorType),
		},
		Capability: CapabilityRef{
			Name:            request.Descriptor.Name,
			Version:         request.Descriptor.Version,
			Operation:       request.Descriptor.Operation,
			SideEffectClass: request.Descriptor.SideEffectClass,
		},
		Authorization: runtimeAuthorization(request),
		Approval:      runtimeApproval(request),
		Outcome:       runtimeOutcome(request),
		InputDigest:   digestBytes(request.Input),
		OutputDigest:  digestBytes(request.Output),
		OccurredAt:    time.Now().UTC(),
	}
	if err := envelope.Validate(); err != nil {
		return Envelope{}, err
	}
	return envelope, nil
}

func runtimeAuthorization(request tool.CapabilityEvidenceRequest) AuthorizationEvidence {
	decision := PolicyAllow
	reason := "authorized by Fenix governance"
	if !request.Governance.Allowed {
		decision = PolicyDeny
		reason = strings.TrimSpace(request.Governance.DenialReason)
		if reason == "" {
			reason = "denied by Fenix governance"
		}
	}
	return AuthorizationEvidence{
		PolicyID:      strings.TrimSpace(request.Governance.PolicyReference),
		PolicyVersion: "runtime",
		Decision:      decision,
		Reason:        reason,
		ContextHash:   runtimeContextHash(request),
	}
}

func runtimeApproval(request tool.CapabilityEvidenceRequest) *ApprovalEvidence {
	approvalID := strings.TrimSpace(request.ApprovalID)
	if approvalID == "" {
		return nil
	}
	decision := "APPROVED"
	if !request.Governance.Allowed && request.Governance.ApprovalRequired {
		decision = "DENIED"
	}
	return &ApprovalEvidence{ApprovalID: approvalID, Decision: decision}
}

func runtimeOutcome(request tool.CapabilityEvidenceRequest) ExecutionOutcome {
	status := OutcomeIndeterminate
	switch request.Status {
	case tool.CapabilityStatusSucceeded:
		status = OutcomeSucceeded
	case tool.CapabilityStatusDenied:
		status = OutcomeDenied
	case tool.CapabilityStatusFailed:
		status = OutcomeFailed
	case tool.CapabilityStatusIndeterminate:
		status = OutcomeIndeterminate
	}
	return ExecutionOutcome{Status: status, ErrorCode: request.ErrorCode}
}

func runtimeContextHash(request tool.CapabilityEvidenceRequest) string {
	contextMaterial := struct {
		WorkspaceID     string `json:"workspace_id"`
		TraceID         string `json:"trace_id"`
		ExecutionID     string `json:"execution_id"`
		ApprovalID      string `json:"approval_id,omitempty"`
		PolicyReference string `json:"policy_reference"`
	}{
		WorkspaceID:     strings.TrimSpace(request.WorkspaceID),
		TraceID:         strings.TrimSpace(request.TraceID),
		ExecutionID:     strings.TrimSpace(request.ExecutionID),
		ApprovalID:      strings.TrimSpace(request.ApprovalID),
		PolicyReference: strings.TrimSpace(request.Governance.PolicyReference),
	}
	raw, _ := json.Marshal(contextMaterial)
	return sha256Hex(raw)
}

func digestBytes(raw []byte) *DigestRef {
	if len(raw) == 0 {
		return nil
	}
	return &DigestRef{Algorithm: "sha256", Value: sha256Hex(raw)}
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func auditProjectionMetadata(projection AuditProjection) map[string]any {
	return map[string]any{
		"provider":            projection.Provider,
		"execution_id":        projection.ExecutionID,
		"stream_id":           projection.StreamID,
		"event_id":            projection.EventID,
		"event_hash":          projection.EventHash,
		"key_id":              projection.KeyID,
		"signature_ref":       projection.SignatureRef,
		"sequence":            projection.Sequence,
		"checkpoint_id":       projection.CheckpointID,
		"checkpoint_hash":     projection.CheckpointHash,
		"merkle_root":         projection.MerkleRoot,
		"tree_size":           projection.TreeSize,
		"verification_status": string(projection.VerificationStatus),
		"issue_count":         projection.IssueCount,
	}
}


func (r *RuntimeRecorder) reconcileIndeterminateRecord(
	ctx context.Context,
	envelope Envelope,
	recordErr error,
) (ProofReference, error) {
	var indeterminate *IndeterminateRecordError
	if !errors.As(recordErr, &indeterminate) {
		return ProofReference{}, recordErr
	}
	ref, lookupErr := r.sink.LookupEvidence(ctx, envelope.StreamID, envelope.ExecutionID)
	if lookupErr != nil {
		return ProofReference{}, lookupErr
	}
	if ref == nil {
		return ProofReference{}, recordErr
	}
	return *ref, nil
}
