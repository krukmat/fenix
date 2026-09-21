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
	return CapabilityRuntimeDecision{
		Allowed:             runtimeGovernanceAllows(facts),
		DenialReason:        runtimeGovernanceDenialReason(facts),
		ApprovalRequired:    facts.ApprovalRequired,
		EvidenceRequirement: RuntimeEvidenceNone,
		EvidencePlanned:     false,
		PolicyReference:     "fenix:w1-runtime",
	}
}

func runtimeGovernanceAllows(facts CapabilityGovernanceFacts) bool {
	if facts.GovernorRequired && !facts.GovernorPassed {
		return false
	}
	return !facts.ApprovalRequired || facts.ApprovalGranted
}

func runtimeGovernanceDenialReason(facts CapabilityGovernanceFacts) string {
	if facts.GovernorRequired && !facts.GovernorPassed {
		return "governance_unavailable"
	}
	if facts.ApprovalRequired && !facts.ApprovalGranted {
		return "approval_denied"
	}
	return ""
}

func validateRuntimeDecision(
	decision CapabilityRuntimeDecision,
	facts CapabilityGovernanceFacts,
) error {
	if !hasValidRuntimeDecisionContract(decision) ||
		(decision.Allowed && !runtimeGovernanceAllows(facts)) ||
		hasInvalidRuntimeEvidencePlan(decision) {
		return ErrCapabilityGovernanceRequired
	}
	return nil
}

func hasValidRuntimeDecisionContract(decision CapabilityRuntimeDecision) bool {
	return strings.TrimSpace(decision.PolicyReference) != "" &&
		validRuntimeEvidenceRequirement(decision.EvidenceRequirement)
}

func hasInvalidRuntimeEvidencePlan(decision CapabilityRuntimeDecision) bool {
	return decision.EvidencePlanned &&
		decision.EvidenceRequirement == RuntimeEvidenceNone
}

func validRuntimeEvidenceRequirement(requirement RuntimeEvidenceRequirement) bool {
	return requirement == RuntimeEvidenceNone ||
		requirement == RuntimeEvidenceOptional ||
		requirement == RuntimeEvidenceRequired
}
