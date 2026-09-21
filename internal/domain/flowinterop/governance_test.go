package flowinterop

import (
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/governance"
)

func TestGovernanceProfile_DoesNotRequireVELForEveryM2SFOperation(t *testing.T) {
	validate, ok := GovernanceProfile(OperationValidate)
	if !ok {
		t.Fatal("validate governance profile missing")
	}
	if validate.EvidenceRequirement != governance.EvidenceNone {
		t.Fatalf("validate evidence = %q, want none", validate.EvidenceRequirement)
	}

	export, ok := GovernanceProfile(OperationExport)
	if !ok {
		t.Fatal("export governance profile missing")
	}
	if export.EvidenceRequirement != governance.EvidenceOptional {
		t.Fatalf("export evidence = %q, want optional", export.EvidenceRequirement)
	}
}

func TestGovernanceProfile_OptionalEvidenceIsDecidedByFenix(t *testing.T) {
	profile, ok := GovernanceProfile(OperationExport)
	if !ok {
		t.Fatal("export governance profile missing")
	}

	withoutEvidence, err := governance.Resolve(governance.Input{
		Profile:         profile,
		PolicyAllowed:   true,
		PolicyReference: "fenix:test",
		ApprovalState:   governance.ApprovalNotRequired,
	})
	if err != nil {
		t.Fatalf("Resolve without evidence returned error: %v", err)
	}
	if !withoutEvidence.Allowed || withoutEvidence.EvidencePlanned {
		t.Fatalf("unexpected decision without evidence: %#v", withoutEvidence)
	}

	withEvidence, err := governance.Resolve(governance.Input{
		Profile:           profile,
		PolicyAllowed:     true,
		PolicyReference:   "fenix:test",
		ApprovalState:     governance.ApprovalNotRequired,
		OptionalEvidence: true,
	})
	if err != nil {
		t.Fatalf("Resolve with evidence returned error: %v", err)
	}
	if !withEvidence.Allowed || !withEvidence.EvidencePlanned {
		t.Fatalf("unexpected decision with evidence: %#v", withEvidence)
	}
}
