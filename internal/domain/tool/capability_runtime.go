package tool

import (
	"context"
	"encoding/json"
	"strings"
)

// RuntimeEvidenceRequirement is the runtime projection of the W4 evidence policy.
type RuntimeEvidenceRequirement string

const (
	RuntimeEvidenceNone     RuntimeEvidenceRequirement = "none"
	RuntimeEvidenceOptional RuntimeEvidenceRequirement = "optional"
	RuntimeEvidenceRequired RuntimeEvidenceRequirement = "required"
)

// CapabilityGovernanceFacts are W1 facts already known by the governed runtime.
type CapabilityGovernanceFacts struct {
	GovernorRequired bool
	GovernorPassed   bool
	ApprovalRequired bool
	ApprovalGranted  bool
	GovernanceError  string
}

// CapabilityRuntimeDecision is the tool-layer projection of the single W4 GovernanceDecision.
type CapabilityRuntimeDecision struct {
	Allowed             bool
	DenialReason        string
	ApprovalRequired    bool
	EvidenceRequirement RuntimeEvidenceRequirement
	EvidencePlanned     bool
	EvidenceReason      string
	PolicyReference     string
}

// CapabilityGovernancePlanner resolves the single runtime governance decision for one execution.
type CapabilityGovernancePlanner interface {
	PlanCapability(
		ctx context.Context,
		descriptor CapabilityDescriptor,
		facts CapabilityGovernanceFacts,
	) (CapabilityRuntimeDecision, error)
}

// CapabilityEvidenceRequest contains in-memory execution data needed to construct minimized evidence.
// Recorder implementations must digest payloads before crossing an external evidence boundary.
type CapabilityEvidenceRequest struct {
	WorkspaceID string
	TraceID     string
	ExecutionID string
	RunID       string
	ApprovalID  string
	ActorID     string
	ActorType   string
	Descriptor  CapabilityDescriptor
	Governance  CapabilityRuntimeDecision
	Status      CapabilityExecutionStatus
	ErrorCode   string
	Input       json.RawMessage
	Output      json.RawMessage
}

// CapabilityEvidenceResult is the evidence lifecycle projection returned to the tool runtime.
type CapabilityEvidenceResult struct {
	State         string
	AuditMetadata map[string]any
}

// CapabilityEvidenceRecorder records or initiates evidence without owning business execution.
type CapabilityEvidenceRecorder interface {
	RecordCapabilityEvidence(
		ctx context.Context,
		request CapabilityEvidenceRequest,
	) (CapabilityEvidenceResult, error)
}

func defaultRuntimeDecision(
	descriptor CapabilityDescriptor,
	facts CapabilityGovernanceFacts,
) CapabilityRuntimeDecision {
	allowed := (!facts.GovernorRequired || facts.GovernorPassed) &&
		(!facts.ApprovalRequired || facts.ApprovalGranted)
	denialReason := ""
	if facts.GovernorRequired && !facts.GovernorPassed {
		denialReason = "governance_unavailable"
	} else if facts.ApprovalRequired && !facts.ApprovalGranted {
		denialReason = "approval_denied"
	}
	return CapabilityRuntimeDecision{
		Allowed:             allowed,
		DenialReason:        denialReason,
		ApprovalRequired:    facts.ApprovalRequired,
		EvidenceRequirement: RuntimeEvidenceNone,
		EvidencePlanned:     false,
		PolicyReference:     "fenix:w1-runtime",
	}
}

func validateRuntimeDecision(
	decision CapabilityRuntimeDecision,
	facts CapabilityGovernanceFacts,
) error {
	if strings.TrimSpace(decision.PolicyReference) == "" ||
		!validRuntimeEvidenceRequirement(decision.EvidenceRequirement) {
		return ErrCapabilityGovernanceRequired
	}
	if decision.Allowed && facts.GovernorRequired && !facts.GovernorPassed {
		return ErrCapabilityGovernanceRequired
	}
	if decision.Allowed && facts.ApprovalRequired && !facts.ApprovalGranted {
		return ErrCapabilityGovernanceRequired
	}
	if decision.EvidencePlanned && decision.EvidenceRequirement == RuntimeEvidenceNone {
		return ErrCapabilityGovernanceRequired
	}
	return nil
}

func validRuntimeEvidenceRequirement(requirement RuntimeEvidenceRequirement) bool {
	return requirement == RuntimeEvidenceNone ||
		requirement == RuntimeEvidenceOptional ||
		requirement == RuntimeEvidenceRequired
}
