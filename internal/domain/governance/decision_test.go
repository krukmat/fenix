package governance

import (
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

func profile(class tool.SideEffectClass, approval bool, evidence EvidenceRequirement) Profile {
	return Profile{
		Capability: tool.CapabilityDescriptor{
			Name:             "example.capability",
			Version:          "1",
			Operation:        "execute",
			SideEffectClass:  class,
			ApprovalRequired: approval,
		},
		EvidenceRequirement: evidence,
		EvidenceReason:      "governance profile",
	}
}

func TestResolve_SeparatesApprovalAndEvidenceDimensions(t *testing.T) {
	decision, err := Resolve(Input{
		Profile:           profile(tool.SideEffectTransform, false, EvidenceOptional),
		PolicyAllowed:     true,
		PolicyReference:   "policy:v1",
		ApprovalState:     ApprovalNotRequired,
		OptionalEvidence: true,
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !decision.Allowed || decision.ApprovalRequired || !decision.EvidencePlanned {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestResolve_RequiredEvidenceIsPlannedBeforeExecution(t *testing.T) {
	decision, err := Resolve(Input{
		Profile:         profile(tool.SideEffectMutate, false, EvidenceRequired),
		PolicyAllowed:   true,
		PolicyReference: "policy:v1",
		ApprovalState:   ApprovalNotRequired,
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !decision.Allowed || !decision.EvidencePlanned {
		t.Fatalf("required evidence not planned: %#v", decision)
	}
}

func TestResolve_IrreversibleCapabilityRequiresApprovedRequest(t *testing.T) {
	decision, err := Resolve(Input{
		Profile:         profile(tool.SideEffectIrreversible, false, EvidenceRequired),
		PolicyAllowed:   true,
		PolicyReference: "policy:v1",
		ApprovalState:   ApprovalMissing,
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if decision.Allowed || !decision.ApprovalRequired || decision.DenialReason != "approval_missing" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestResolve_PolicyDenialWinsBeforeProviderExecution(t *testing.T) {
	decision, err := Resolve(Input{
		Profile:         profile(tool.SideEffectVerify, false, EvidenceNone),
		PolicyAllowed:   false,
		PolicyReference: "policy:v1",
		ApprovalState:   ApprovalNotRequired,
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if decision.Allowed || decision.DenialReason != "policy_denied" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
}

func TestResolve_NoneEvidenceRejectsOptionalSelection(t *testing.T) {
	_, err := Resolve(Input{
		Profile:           profile(tool.SideEffectRead, false, EvidenceNone),
		PolicyAllowed:     true,
		PolicyReference:   "policy:v1",
		ApprovalState:     ApprovalNotRequired,
		OptionalEvidence: true,
	})
	if !errors.Is(err, ErrDecisionInputInvalid) {
		t.Fatalf("expected ErrDecisionInputInvalid, got %v", err)
	}
}


func TestResolve_RejectsEvidenceRequirementWithoutReason(t *testing.T) {
	p := profile(tool.SideEffectTransform, false, EvidenceOptional)
	p.EvidenceReason = ""

	_, err := Resolve(Input{
		Profile:         p,
		PolicyAllowed:   true,
		PolicyReference: "policy:v1",
		ApprovalState:   ApprovalNotRequired,
	})
	if !errors.Is(err, ErrDecisionInputInvalid) {
		t.Fatalf("expected ErrDecisionInputInvalid, got %v", err)
	}
}

func TestResolve_RejectsUnknownSideEffectClassification(t *testing.T) {
	p := profile(tool.SideEffectClass("unknown"), false, EvidenceNone)

	_, err := Resolve(Input{
		Profile:         p,
		PolicyAllowed:   true,
		PolicyReference: "policy:v1",
		ApprovalState:   ApprovalNotRequired,
	})
	if !errors.Is(err, ErrDecisionInputInvalid) {
		t.Fatalf("expected ErrDecisionInputInvalid, got %v", err)
	}
}
