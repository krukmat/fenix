package flowinterop

import (
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

func TestCatalog_ExposesOnlyW2ATransportNeutralCapabilities(t *testing.T) {
	catalog := Catalog()
	if len(catalog) != 3 {
		t.Fatalf("catalog size = %d, want 3", len(catalog))
	}

	expected := map[Operation]tool.SideEffectClass{
		OperationImport:   tool.SideEffectTransform,
		OperationExport:   tool.SideEffectTransform,
		OperationValidate: tool.SideEffectVerify,
	}
	for operation, sideEffect := range expected {
		spec, ok := Lookup(operation)
		if !ok {
			t.Fatalf("missing operation %s", operation)
		}
		if spec.Descriptor.Version != ContractVersion {
			t.Fatalf("%s version = %q, want %q", operation, spec.Descriptor.Version, ContractVersion)
		}
		if spec.Descriptor.SideEffectClass != sideEffect {
			t.Fatalf("%s side effect = %q, want %q", operation, spec.Descriptor.SideEffectClass, sideEffect)
		}
	}
}

func TestRequestValidate_EnforcesOperationFormats(t *testing.T) {
	valid := Request{
		ContractVersion: ContractVersion,
		Operation:       OperationImport,
		Input: Artifact{
			Format:  FormatSalesforceFlowXML,
			Content: "<Flow />",
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid import rejected: %v", err)
	}

	invalid := valid
	invalid.Input.Format = FormatMermaid
	if err := invalid.Validate(); !errors.Is(err, ErrArtifactFormat) {
		t.Fatalf("expected ErrArtifactFormat, got %v", err)
	}
}

func TestRequestValidate_RejectsUnknownContractVersion(t *testing.T) {
	request := Request{
		ContractVersion: "2",
		Operation:       OperationValidate,
		Input: Artifact{
			Format:  FormatMermaid,
			Content: "flowchart TD",
		},
	}
	if err := request.Validate(); !errors.Is(err, ErrContractVersion) {
		t.Fatalf("expected ErrContractVersion, got %v", err)
	}
}

func TestMaxSupport_ReflectsMermaid2SFPublishedCeiling(t *testing.T) {
	families := []FlowFamily{
		FlowFamilyAutolaunched,
		FlowFamilyScreen,
		FlowFamilyRecordTriggeredAfterSave,
		FlowFamilyRecordTriggeredBeforeSave,
		FlowFamilyRecordTriggeredBeforeDelete,
		FlowFamilyScheduleTriggered,
		FlowFamilyPlatformEventTriggered,
	}
	for _, family := range families {
		if got := MaxSupport(OperationExport, family).Level; got != FidelityGuaranteed {
			t.Fatalf("export support for %s = %q, want guaranteed", family, got)
		}
		if got := MaxSupport(OperationImport, family).Level; got != FidelityGuaranteed {
			t.Fatalf("import support for %s = %q, want guaranteed", family, got)
		}
	}

	if got := MaxSupport(OperationImport, FlowFamilyOrchestrated).Level; got != FidelityPartial {
		t.Fatalf("orchestrated import = %q, want partial", got)
	}
	if got := MaxSupport(OperationExport, FlowFamilyOrchestrated).Level; got != FidelityUnsupported {
		t.Fatalf("orchestrated export = %q, want unsupported", got)
	}
}

func TestFidelityReport_GuaranteedCannotHideUnsupportedFeatures(t *testing.T) {
	report := FidelityReport{
		ContractVersion:     ContractVersion,
		Level:               FidelityGuaranteed,
		FlowFamily:          FlowFamilyAutolaunched,
		UnsupportedFeatures: []string{"apex_action"},
	}
	if err := report.Validate(); !errors.Is(err, ErrFidelity) {
		t.Fatalf("expected ErrFidelity, got %v", err)
	}
}

func TestFidelityReport_PartialCanExposeBoundary(t *testing.T) {
	report := FidelityReport{
		ContractVersion:     ContractVersion,
		Level:               FidelityPartial,
		FlowFamily:          FlowFamilyAutolaunched,
		SupportedFeatures:   []string{"assignment", "decision"},
		UnsupportedFeatures: []string{"apex_action"},
		Warnings:            []string{"unsupported metadata was detected"},
	}
	if err := report.Validate(); err != nil {
		t.Fatalf("partial fidelity report rejected: %v", err)
	}
}
