package flowinterop

import (
	"errors"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
)

func TestCatalog_ExposesOnlyW2ATransportNeutralCapabilities(t *testing.T) {
	catalog := Catalog()
	if len(catalog) != 4 {
		t.Fatalf("catalog size = %d, want 4", len(catalog))
	}

	expected := map[Operation]tool.SideEffectClass{
		OperationImport:   tool.SideEffectTransform,
		OperationExport:   tool.SideEffectTransform,
		OperationValidate: tool.SideEffectVerify,
		OperationCompare:  tool.SideEffectVerify,
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
	if got := MaxSupport(OperationCompare, FlowFamilyAutolaunched).Level; got != FidelityGuaranteed {
		t.Fatalf("autolaunched compare = %q, want guaranteed", got)
	}
	if got := MaxSupport(OperationCompare, FlowFamilyOrchestrated).Level; got != FidelityUnsupported {
		t.Fatalf("orchestrated compare = %q, want unsupported", got)
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


func TestRequestValidate_CompareRequiresSecondArtifact(t *testing.T) {
	request := Request{
		ContractVersion: ContractVersion,
		Operation:       OperationCompare,
		Input: Artifact{
			Format:  FormatSalesforceFlowXML,
			Content: "<Flow />",
		},
	}
	if err := request.Validate(); !errors.Is(err, ErrComparisonArtifact) {
		t.Fatalf("expected ErrComparisonArtifact, got %v", err)
	}

	request.CompareTo = &Artifact{
		Format:  FormatMermaid,
		Content: "flowchart TD",
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("valid compare rejected: %v", err)
	}
}

func TestSemanticDiff_RejectsContradictoryState(t *testing.T) {
	diff := SemanticDiff{
		Equal: true,
		Changes: []SemanticChange{
			{Path: "elements.Assignment_1", Kind: SemanticChangeChanged},
		},
	}
	if err := diff.Validate(); !errors.Is(err, ErrSemanticDiff) {
		t.Fatalf("expected ErrSemanticDiff, got %v", err)
	}

	diff = SemanticDiff{
		Equal: false,
		Changes: []SemanticChange{
			{Path: "elements.Assignment_1", Kind: SemanticChangeChanged},
		},
	}
	if err := diff.Validate(); err != nil {
		t.Fatalf("valid semantic diff rejected: %v", err)
	}
}

func TestDiagnosticValidate_PreservesStableProviderCodes(t *testing.T) {
	diagnostic := Diagnostic{
		Code:        "M2SF-SF-008",
		Severity:    DiagnosticError,
		Stage:       DiagnosticStageValidate,
		Message:     "RecordBeforeSave element is not allowed.",
		ElementID:   "Update_Account",
		Recoverable: true,
	}
	if err := diagnostic.Validate(); err != nil {
		t.Fatalf("valid diagnostic rejected: %v", err)
	}

	diagnostic.Stage = "unknown"
	if err := diagnostic.Validate(); !errors.Is(err, ErrDiagnostic) {
		t.Fatalf("expected ErrDiagnostic, got %v", err)
	}
}

func TestAgentSurface_SeparatesInvocationFromResultTrust(t *testing.T) {
	for _, operation := range []Operation{
		OperationImport,
		OperationExport,
		OperationValidate,
		OperationCompare,
	} {
		if !AgentCallable(operation) {
			t.Fatalf("operation %s should be agent-callable", operation)
		}
	}

	guaranteed := Result{
		Status: ResultSucceeded,
		Fidelity: FidelityReport{
			Level: FidelityGuaranteed,
		},
	}
	if got := EvaluateAgentResult(guaranteed).Disposition; got != AgentUse {
		t.Fatalf("guaranteed disposition = %q, want use", got)
	}

	partial := guaranteed
	partial.Fidelity.Level = FidelityPartial
	if got := EvaluateAgentResult(partial).Disposition; got != AgentReviewOnly {
		t.Fatalf("partial disposition = %q, want review_only", got)
	}

	unsupported := guaranteed
	unsupported.Fidelity.Level = FidelityUnsupported
	if got := EvaluateAgentResult(unsupported).Disposition; got != AgentAbstain {
		t.Fatalf("unsupported disposition = %q, want abstain", got)
	}
}
