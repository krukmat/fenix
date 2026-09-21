package governance

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

// RuntimePolicySelection contains Fenix-owned policy/evidence facts not supplied by the provider.
type RuntimePolicySelection struct {
	Allowed          bool
	PolicyReference  string
	OptionalEvidence bool
}

// RuntimePolicySelector supplies the current Fenix policy result to the runtime planner.
type RuntimePolicySelector interface {
	SelectRuntimePolicy(
		ctx context.Context,
		profile Profile,
	) (RuntimePolicySelection, error)
}

// RuntimePlanner adapts the W4 GovernanceDecision contract to the W1 tool runtime port.
type RuntimePlanner struct {
	mu       sync.RWMutex
	profiles map[string]Profile
	selector RuntimePolicySelector
}

// NewRuntimePlanner creates a planner. A nil selector uses the already-passed W1 policy gate.
func NewRuntimePlanner(selector RuntimePolicySelector) *RuntimePlanner {
	return &RuntimePlanner{
		profiles: make(map[string]Profile),
		selector: selector,
	}
}

// RegisterProfile installs the Fenix-owned governance profile for a capability.
func (p *RuntimePlanner) RegisterProfile(profile Profile) error {
	if strings.TrimSpace(profile.Capability.Name) == "" {
		return ErrDecisionInputInvalid
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.profiles[profile.Capability.Name] = profile
	return nil
}

// PlanCapability resolves exactly one W4 decision for the logical execution.
func (p *RuntimePlanner) PlanCapability(
	ctx context.Context,
	descriptor tool.CapabilityDescriptor,
	facts tool.CapabilityGovernanceFacts,
) (tool.CapabilityRuntimeDecision, error) {
	profile := p.profileFor(descriptor)
	selection, err := p.policySelection(ctx, profile)
	if err != nil {
		return tool.CapabilityRuntimeDecision{}, err
	}
	decision, err := Resolve(Input{
		Profile:           profile,
		PolicyAllowed:     selection.Allowed,
		PolicyReference:   selection.PolicyReference,
		ApprovalState:     runtimeApprovalState(facts),
		OptionalEvidence: selection.OptionalEvidence,
	})
	if err != nil {
		return tool.CapabilityRuntimeDecision{}, err
	}
	if facts.GovernorRequired && !facts.GovernorPassed {
		decision.Allowed = false
		decision.DenialReason = "governance_unavailable"
	}
	runtime := runtimeDecision(decision)
	if !runtime.Allowed {
		runtime.DenialCause = facts.GovernanceCause
	}
	return runtime, nil
}

func (p *RuntimePlanner) profileFor(descriptor tool.CapabilityDescriptor) Profile {
	p.mu.RLock()
	profile, ok := p.profiles[descriptor.Name]
	p.mu.RUnlock()
	if ok {
		profile.Capability = descriptor
		return profile
	}
	return Profile{
		Capability:          descriptor,
		EvidenceRequirement: EvidenceNone,
	}
}

func (p *RuntimePlanner) policySelection(
	ctx context.Context,
	profile Profile,
) (RuntimePolicySelection, error) {
	if p.selector == nil {
		return RuntimePolicySelection{
			Allowed:         true,
			PolicyReference: "fenix:w1-policy-gate",
		}, nil
	}
	selection, err := p.selector.SelectRuntimePolicy(ctx, profile)
	if err != nil {
		return RuntimePolicySelection{}, fmt.Errorf("select runtime governance policy: %w", err)
	}
	return selection, nil
}

func runtimeApprovalState(facts tool.CapabilityGovernanceFacts) ApprovalState {
	if !facts.ApprovalRequired {
		return ApprovalNotRequired
	}
	if facts.ApprovalGranted {
		return ApprovalApproved
	}
	return ApprovalDenied
}

func runtimeDecision(decision Decision) tool.CapabilityRuntimeDecision {
	return tool.CapabilityRuntimeDecision{
		Allowed:             decision.Allowed,
		DenialReason:        decision.DenialReason,
		ApprovalRequired:    decision.ApprovalRequired,
		EvidenceRequirement: tool.RuntimeEvidenceRequirement(decision.EvidenceRequirement),
		EvidencePlanned:     decision.EvidencePlanned,
		EvidenceReason:      decision.EvidenceReason,
		PolicyReference:     decision.PolicyReference,
	}
}
