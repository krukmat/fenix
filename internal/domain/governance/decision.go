package governance

import (
	"errors"
	"strings"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

var ErrDecisionInputInvalid = errors.New("invalid cross-platform governance input")

// EvidenceRequirement classifies whether immutable evidence participates in a governed execution.
type EvidenceRequirement string

const (
	EvidenceNone     EvidenceRequirement = "none"
	EvidenceOptional EvidenceRequirement = "optional"
	EvidenceRequired EvidenceRequirement = "required"
)

// ApprovalState is the approval status supplied to the cross-platform governance resolver.
type ApprovalState string

const (
	ApprovalNotRequired ApprovalState = "not_required"
	ApprovalApproved    ApprovalState = "approved"
	ApprovalPending     ApprovalState = "pending"
	ApprovalDenied      ApprovalState = "denied"
	ApprovalMissing     ApprovalState = "missing"
)

// Profile adds cross-platform governance metadata without coupling the capability descriptor to VEL.
type Profile struct {
	Capability          tool.CapabilityDescriptor
	EvidenceRequirement EvidenceRequirement
	EvidenceReason      string
}

// Input contains the Fenix-owned facts needed to resolve one execution before provider invocation.
type Input struct {
	Profile           Profile
	PolicyAllowed     bool
	PolicyReference   string
	ApprovalState     ApprovalState
	OptionalEvidence bool
}

// Decision is the single Fenix-owned governance result consumed by later runtime integration.
type Decision struct {
	Allowed             bool                `json:"allowed"`
	DenialReason        string              `json:"denial_reason,omitempty"`
	ApprovalRequired    bool                `json:"approval_required"`
	EvidenceRequirement EvidenceRequirement `json:"evidence_requirement"`
	EvidencePlanned     bool                `json:"evidence_planned"`
	EvidenceReason      string              `json:"evidence_reason,omitempty"`
	SideEffectClass     tool.SideEffectClass `json:"side_effect_class"`
	CapabilityName      string              `json:"capability_name"`
	CapabilityVersion   string              `json:"capability_version"`
	PolicyReference     string              `json:"policy_reference"`
}

// Resolve computes the governance result before capability execution.
// Provider semantics and cryptographic evidence do not participate in this decision.
func Resolve(input Input) (Decision, error) {
	if err := validateInput(input); err != nil {
		return Decision{}, err
	}

	decision := baseDecision(input)
	if !input.PolicyAllowed {
		decision.DenialReason = "policy_denied"
		return decision, nil
	}
	if decision.ApprovalRequired && input.ApprovalState != ApprovalApproved {
		decision.DenialReason = approvalDenialReason(input.ApprovalState)
		return decision, nil
	}
	decision.Allowed = true
	return decision, nil
}

func baseDecision(input Input) Decision {
	return Decision{
		ApprovalRequired:    input.Profile.Capability.RequiresApproval(),
		EvidenceRequirement: input.Profile.EvidenceRequirement,
		EvidencePlanned:     evidencePlanned(input.Profile.EvidenceRequirement, input.OptionalEvidence),
		EvidenceReason:      strings.TrimSpace(input.Profile.EvidenceReason),
		SideEffectClass:     input.Profile.Capability.SideEffectClass,
		CapabilityName:      input.Profile.Capability.Name,
		CapabilityVersion:   input.Profile.Capability.Version,
		PolicyReference:     strings.TrimSpace(input.PolicyReference),
	}
}

func validateInput(input Input) error {
	if strings.TrimSpace(input.Profile.Capability.Name) == "" ||
		strings.TrimSpace(input.Profile.Capability.Version) == "" ||
		strings.TrimSpace(input.Profile.Capability.Operation) == "" ||
		strings.TrimSpace(input.PolicyReference) == "" ||
		!validEvidenceRequirement(input.Profile.EvidenceRequirement) ||
		!validApprovalState(input.ApprovalState) {
		return ErrDecisionInputInvalid
	}
	if input.Profile.EvidenceRequirement == EvidenceNone && input.OptionalEvidence {
		return ErrDecisionInputInvalid
	}
	return nil
}

func evidencePlanned(requirement EvidenceRequirement, optionalRequested bool) bool {
	return requirement == EvidenceRequired ||
		(requirement == EvidenceOptional && optionalRequested)
}

func approvalDenialReason(state ApprovalState) string {
	switch state {
	case ApprovalPending:
		return "approval_pending"
	case ApprovalDenied:
		return "approval_denied"
	case ApprovalMissing, ApprovalNotRequired:
		return "approval_missing"
	default:
		return "approval_invalid"
	}
}

func validEvidenceRequirement(requirement EvidenceRequirement) bool {
	return requirement == EvidenceNone ||
		requirement == EvidenceOptional ||
		requirement == EvidenceRequired
}

func validApprovalState(state ApprovalState) bool {
	return state == ApprovalNotRequired ||
		state == ApprovalApproved ||
		state == ApprovalPending ||
		state == ApprovalDenied ||
		state == ApprovalMissing
}
