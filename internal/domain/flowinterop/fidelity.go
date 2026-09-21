package flowinterop

import "errors"

var ErrFidelity = errors.New("invalid flow fidelity report")

// FidelityLevel describes semantic preservation for the specific artifact,
// not merely whether Mermaid2SF can parse part of it.
type FidelityLevel string

const (
	FidelityGuaranteed  FidelityLevel = "guaranteed"
	FidelityPartial     FidelityLevel = "partial"
	FidelityUnsupported FidelityLevel = "unsupported"
)

// FlowFamily is the Fenix-facing classification derived from Mermaid2SF's
// documented flow-family support matrix.
type FlowFamily string

const (
	FlowFamilyAutolaunched              FlowFamily = "autolaunched"
	FlowFamilyScreen                    FlowFamily = "screen"
	FlowFamilyRecordTriggeredAfterSave  FlowFamily = "record_triggered_after_save"
	FlowFamilyRecordTriggeredBeforeSave FlowFamily = "record_triggered_before_save"
	FlowFamilyRecordTriggeredBeforeDelete FlowFamily = "record_triggered_before_delete"
	FlowFamilyScheduleTriggered         FlowFamily = "schedule_triggered"
	FlowFamilyPlatformEventTriggered    FlowFamily = "platform_event_triggered"
	FlowFamilyOrchestrated              FlowFamily = "orchestrated"
)

// VerificationScope distinguishes semantic guarantees from Salesforce-org evidence.
type VerificationScope string

const (
	VerificationNone             VerificationScope = "none"
	VerificationSemanticRoundTrip VerificationScope = "semantic_roundtrip"
	VerificationCanonicalOrgDryRun VerificationScope = "canonical_fixture_org_dry_run"
)

// FidelityReport is returned for the concrete artifact examined by Mermaid2SF.
// A family support ceiling must never be promoted automatically into Guaranteed.
type FidelityReport struct {
	ContractVersion     string            `json:"contract_version"`
	Level               FidelityLevel     `json:"level"`
	FlowFamily          FlowFamily        `json:"flow_family"`
	SupportedFeatures   []string          `json:"supported_features,omitempty"`
	UnsupportedFeatures []string          `json:"unsupported_features,omitempty"`
	Warnings            []string          `json:"warnings,omitempty"`
	VerificationScopes  []VerificationScope `json:"verification_scopes,omitempty"`
}

// SupportCeiling is static provider evidence: the best fidelity Mermaid2SF
// currently claims for a flow family/operation. It is not a runtime verdict.
type SupportCeiling struct {
	Level              FidelityLevel
	VerificationScopes []VerificationScope
}

// MaxSupport returns the documented ceiling from Mermaid2SF's current
// SUPPORTED_FEATURES contract. Runtime code must still inspect the artifact.
func MaxSupport(operation Operation, family FlowFamily) SupportCeiling {
	if family == FlowFamilyOrchestrated {
		if operation == OperationImport {
			return SupportCeiling{Level: FidelityPartial}
		}
		return SupportCeiling{Level: FidelityUnsupported}
	}
	if !isGuaranteedFamily(family) {
		return SupportCeiling{Level: FidelityUnsupported}
	}
	switch operation {
	case OperationImport, OperationExport, OperationValidate:
		return SupportCeiling{
			Level: FidelityGuaranteed,
			VerificationScopes: []VerificationScope{
				VerificationSemanticRoundTrip,
				VerificationCanonicalOrgDryRun,
			},
		}
	default:
		return SupportCeiling{Level: FidelityUnsupported}
	}
}

// Validate checks internal fidelity invariants only; it does not infer support.
func (f FidelityReport) Validate() error {
	if !f.hasValidIdentity() || !f.hasConsistentFeatureBoundary() {
		return ErrFidelity
	}
	return nil
}

func (f FidelityReport) hasValidIdentity() bool {
	return f.ContractVersion == ContractVersion &&
		isFidelityLevel(f.Level) &&
		isKnownFamily(f.FlowFamily)
}

func (f FidelityReport) hasConsistentFeatureBoundary() bool {
	if f.Level == FidelityGuaranteed {
		return len(f.UnsupportedFeatures) == 0
	}
	if f.Level == FidelityUnsupported {
		return len(f.SupportedFeatures) == 0
	}
	return true
}

func isGuaranteedFamily(family FlowFamily) bool {
	switch family {
	case FlowFamilyAutolaunched,
		FlowFamilyScreen,
		FlowFamilyRecordTriggeredAfterSave,
		FlowFamilyRecordTriggeredBeforeSave,
		FlowFamilyRecordTriggeredBeforeDelete,
		FlowFamilyScheduleTriggered,
		FlowFamilyPlatformEventTriggered:
		return true
	default:
		return false
	}
}

func isKnownFamily(family FlowFamily) bool {
	return isGuaranteedFamily(family) || family == FlowFamilyOrchestrated
}

func isFidelityLevel(level FidelityLevel) bool {
	return level == FidelityGuaranteed || level == FidelityPartial || level == FidelityUnsupported
}
