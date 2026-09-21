package flowinterop

import (
	"github.com/matiasleandrokruk/fenix/internal/domain/governance"
)

// GovernanceProfile returns the Fenix-owned cross-platform governance profile
// for a Mermaid2SF semantic capability.
func GovernanceProfile(operation Operation) (governance.Profile, bool) {
	spec, ok := Lookup(operation)
	if !ok {
		return governance.Profile{}, false
	}

	requirement, reason := evidenceRequirement(operation)
	return governance.Profile{
		Capability:          spec.Descriptor,
		EvidenceRequirement: requirement,
		EvidenceReason:      reason,
	}, true
}

func evidenceRequirement(operation Operation) (governance.EvidenceRequirement, string) {
	switch operation {
	case OperationImport, OperationExport:
		return governance.EvidenceOptional, "transformation evidence may be requested by Fenix policy"
	case OperationValidate, OperationCompare:
		return governance.EvidenceNone, ""
	default:
		return governance.EvidenceNone, ""
	}
}
