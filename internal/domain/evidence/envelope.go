package evidence

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

const SchemaVersion = "1"

var ErrEnvelopeInvalid = errors.New("invalid evidence envelope")

type PolicyDecision string

const (
	PolicyAllow PolicyDecision = "ALLOW"
	PolicyDeny  PolicyDecision = "DENY"
)

type OutcomeStatus string

const (
	OutcomeSucceeded     OutcomeStatus = "succeeded"
	OutcomeFailed        OutcomeStatus = "failed"
	OutcomeDenied        OutcomeStatus = "denied"
	OutcomeCancelled     OutcomeStatus = "cancelled"
	OutcomeIndeterminate OutcomeStatus = "indeterminate"
)

type ActorRef struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type CapabilityRef struct {
	Name            string               `json:"name"`
	Version         string               `json:"version"`
	Operation       string               `json:"operation"`
	SideEffectClass tool.SideEffectClass `json:"side_effect_class"`
}

type AuthorizationEvidence struct {
	PolicyID    string         `json:"policy_id"`
	PolicyVersion string       `json:"policy_version"`
	Decision    PolicyDecision `json:"decision"`
	Reason      string         `json:"reason"`
	ContextHash string         `json:"context_hash"`
}

type ApprovalEvidence struct {
	ApprovalID string `json:"approval_id"`
	Decision   string `json:"decision"`
}

type ExecutionOutcome struct {
	Status    OutcomeStatus `json:"status"`
	ErrorCode string        `json:"error_code,omitempty"`
}

type DigestRef struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type EvidenceEnvelope struct {
	SchemaVersion string                 `json:"schema_version"`
	StreamID      string                 `json:"stream_id"`
	WorkspaceID   string                 `json:"workspace_id"`
	TraceID       string                 `json:"trace_id"`
	ExecutionID   string                 `json:"execution_id"`
	RunID         string                 `json:"run_id,omitempty"`
	Actor         ActorRef               `json:"actor"`
	Capability    CapabilityRef          `json:"capability"`
	Authorization AuthorizationEvidence  `json:"authorization"`
	Approval      *ApprovalEvidence      `json:"approval,omitempty"`
	Outcome       ExecutionOutcome       `json:"outcome"`
	InputDigest   *DigestRef             `json:"input_digest,omitempty"`
	OutputDigest  *DigestRef             `json:"output_digest,omitempty"`
	OccurredAt    time.Time              `json:"occurred_at"`
}

func (e EvidenceEnvelope) Validate() error {
	if !e.hasValidCore() || !e.hasValidOptionalEvidence() {
		return ErrEnvelopeInvalid
	}
	return nil
}

func (e EvidenceEnvelope) hasValidCore() bool {
	return e.SchemaVersion == SchemaVersion &&
		hasRequiredEnvelopeIdentity(e) &&
		validActor(e.Actor) &&
		validCapability(e.Capability) &&
		validAuthorization(e.Authorization) &&
		validOutcome(e.Outcome) &&
		!e.OccurredAt.IsZero()
}

func (e EvidenceEnvelope) hasValidOptionalEvidence() bool {
	if e.Approval != nil && !validApproval(*e.Approval) {
		return false
	}
	return validOptionalDigest(e.InputDigest) && validOptionalDigest(e.OutputDigest)
}

func hasRequiredEnvelopeIdentity(e EvidenceEnvelope) bool {
	return nonEmpty(e.StreamID) &&
		nonEmpty(e.WorkspaceID) &&
		nonEmpty(e.TraceID) &&
		nonEmpty(e.ExecutionID)
}

func validActor(actor ActorRef) bool {
	return nonEmpty(actor.ID) && nonEmpty(actor.Type)
}

func validCapability(capability CapabilityRef) bool {
	return nonEmpty(capability.Name) &&
		nonEmpty(capability.Version) &&
		nonEmpty(capability.Operation) &&
		validSideEffectClass(capability.SideEffectClass)
}

func validAuthorization(auth AuthorizationEvidence) bool {
	return nonEmpty(auth.PolicyID) &&
		nonEmpty(auth.PolicyVersion) &&
		nonEmpty(auth.Reason) &&
		(auth.Decision == PolicyAllow || auth.Decision == PolicyDeny) &&
		isSHA256Hex(auth.ContextHash)
}

func validApproval(approval ApprovalEvidence) bool {
	return nonEmpty(approval.ApprovalID) && nonEmpty(approval.Decision)
}

func validOutcome(outcome ExecutionOutcome) bool {
	switch outcome.Status {
	case OutcomeSucceeded, OutcomeFailed, OutcomeDenied, OutcomeCancelled, OutcomeIndeterminate:
		return true
	default:
		return false
	}
}

func validOptionalDigest(digest *DigestRef) bool {
	if digest == nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(digest.Algorithm), "sha256") &&
		isSHA256Hex(digest.Value)
}

func validSideEffectClass(class tool.SideEffectClass) bool {
	switch class {
	case tool.SideEffectRead,
		tool.SideEffectTransform,
		tool.SideEffectVerify,
		tool.SideEffectMutate,
		tool.SideEffectIrreversible:
		return true
	default:
		return false
	}
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func nonEmpty(value string) bool {
	return strings.TrimSpace(value) != ""
}
