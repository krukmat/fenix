package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

type runtimePolicySelectorStub struct {
	selection RuntimePolicySelection
	err       error
	calls     int
}

func (s *runtimePolicySelectorStub) SelectRuntimePolicy(
	_ context.Context,
	_ Profile,
) (RuntimePolicySelection, error) {
	s.calls++
	return s.selection, s.err
}

func TestRuntimePlanner_UsesRegisteredEvidenceProfile(t *testing.T) {
	selector := &runtimePolicySelectorStub{
		selection: RuntimePolicySelection{
			Allowed:          true,
			PolicyReference:  "policy:test",
			OptionalEvidence: true,
		},
	}
	planner := NewRuntimePlanner(selector)
	p := profile(tool.SideEffectTransform, false, EvidenceOptional)
	if err := planner.RegisterProfile(p); err != nil {
		t.Fatalf("RegisterProfile returned error: %v", err)
	}

	decision, err := planner.PlanCapability(
		context.Background(),
		p.Capability,
		tool.CapabilityGovernanceFacts{
			GovernorPassed:  true,
			ApprovalGranted: true,
		},
	)
	if err != nil {
		t.Fatalf("PlanCapability returned error: %v", err)
	}
	if !decision.Allowed || !decision.EvidencePlanned {
		t.Fatalf("unexpected runtime decision: %#v", decision)
	}
	if selector.calls != 1 {
		t.Fatalf("selector calls = %d, want 1", selector.calls)
	}
}

func TestRuntimePlanner_GovernorFailureForcesDenial(t *testing.T) {
	planner := NewRuntimePlanner(nil)
	descriptor := tool.CapabilityDescriptor{
		Name:            "runtime.mutate",
		Version:         "1",
		Operation:       "execute",
		SideEffectClass: tool.SideEffectMutate,
	}

	decision, err := planner.PlanCapability(
		context.Background(),
		descriptor,
		tool.CapabilityGovernanceFacts{
			GovernorRequired: true,
			GovernorPassed:   false,
			GovernanceError:  "governor_unavailable",
		},
	)
	if err != nil {
		t.Fatalf("PlanCapability returned error: %v", err)
	}
	if decision.Allowed || decision.DenialReason != "governance_unavailable" {
		t.Fatalf("unexpected runtime decision: %#v", decision)
	}
}

func TestRuntimePlanner_PropagatesPolicySelectorFailure(t *testing.T) {
	selector := &runtimePolicySelectorStub{err: errors.New("policy unavailable")}
	planner := NewRuntimePlanner(selector)
	descriptor := tool.CapabilityDescriptor{
		Name:            "runtime.verify",
		Version:         "1",
		Operation:       "verify",
		SideEffectClass: tool.SideEffectVerify,
	}

	_, err := planner.PlanCapability(
		context.Background(),
		descriptor,
		tool.CapabilityGovernanceFacts{
			GovernorPassed:  true,
			ApprovalGranted: true,
		},
	)
	if err == nil {
		t.Fatal("expected policy selector error")
	}
}
