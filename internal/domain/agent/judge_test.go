package agent

import (
	"context"
	"testing"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

func TestWorkflowJudgeVerify_PassesForValidDSL(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := "CONTEXT\n  system = crm\nACTORS\n  admin\nBEHAVIOR resolve_support_case\n  GIVEN a workflow\nCONSTRAINTS\n  one active per name"
	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-valid",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Passed {
		t.Fatalf("Passed = false, want true; violations = %#v", result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("len(Violations) = %d, want 0", len(result.Violations))
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("len(Warnings) = %d, want 0", len(result.Warnings))
	}
}

func TestWorkflowJudgeVerify_AddsWarningWhenSpecSourceMissing(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:        "wf-no-spec",
		DSLSource: "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Passed {
		t.Fatalf("Passed = false, want true; violations = %#v", result.Violations)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want 1", len(result.Warnings))
	}
	got := result.Warnings[0]
	if got.Code != "missing_spec_source" {
		t.Fatalf("Code = %q, want missing_spec_source", got.Code)
	}
}

func TestWorkflowJudgeVerify_EmptySpecSourceAlsoWarns(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := "   "
	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-empty-spec",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Passed {
		t.Fatalf("Passed = false, want true; violations = %#v", result.Violations)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want 1", len(result.Warnings))
	}
}

func TestWorkflowJudgeVerify_InvalidDSLStillWarnsButFails(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:        "wf-invalid-no-spec",
		DSLSource: "ON case.created\nSET case.status = \"resolved\"",
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(result.Violations))
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want 1", len(result.Warnings))
	}
}

func TestWorkflowJudgeVerify_AddsConsistencyViolationWhenBehaviorHasNoCoverage(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := "BEHAVIOR notify_salesperson"
	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-mismatch",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if len(result.Violations) == 0 {
		t.Fatal("expected consistency violation")
	}
	found := false
	for _, violation := range result.Violations {
		if violation.Code == "behavior_no_coverage" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Violations = %#v", result.Violations)
	}
}

func TestWorkflowJudgeVerify_PassesWhenBehaviorCoverageMatches(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := "BEHAVIOR notify_salesperson"
	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-match",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Passed {
		t.Fatalf("Passed = false, want true; violations = %#v", result.Violations)
	}
}

func TestWorkflowJudgeVerify_LegacySpecMentioningCartaUsesLegacyPath(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := `CONTEXT
  system = crm with CARTA migration notes
ACTORS
  admin
BEHAVIOR notify_salesperson
  GIVEN legacy free-form spec text
CONSTRAINTS
  do not parse as Carta unless source starts with CARTA`

	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-legacy-carta-mention",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Passed {
		t.Fatalf("Passed = false, want true; violations = %#v", result.Violations)
	}
	for _, violation := range result.Violations {
		if violation.Code == "carta_parse_error" {
			t.Fatalf("legacy spec used Carta path: %#v", result.Violations)
		}
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("Warnings = %#v, want none", result.Warnings)
	}
}

func TestWorkflowJudgeVerify_AddsProtocolFindings(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:        "wf-protocol",
		DSLSource: "WORKFLOW route_case\nON case.created\nDISPATCH TO support_agent WITH {\"case_id\":\"case-1\"}\nSURFACE case TO salesperson WITH {\"value\":\"review\"}",
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	foundViolation := false
	foundWarning := false
	for _, violation := range result.Violations {
		if violation.Code == "dispatch_contract_missing" {
			foundViolation = true
		}
	}
	for _, warning := range result.Warnings {
		if warning.Code == "surface_view_ambiguous" {
			foundWarning = true
		}
	}
	if !foundViolation {
		t.Fatalf("Violations = %#v", result.Violations)
	}
	if !foundWarning {
		t.Fatalf("Warnings = %#v", result.Warnings)
	}
}

func TestWorkflowJudgeVerify_ReturnsViolationsForInvalidDSL(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:        "wf-invalid",
		DSLSource: "ON case.created\nSET case.status = \"resolved\"",
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(result.Violations))
	}
	got := result.Violations[0]
	if got.Code != "dsl_syntax_error" {
		t.Fatalf("Code = %q, want dsl_syntax_error", got.Code)
	}
	if got.Line != 1 || got.Column != 1 {
		t.Fatalf("position = %d:%d, want 1:1", got.Line, got.Column)
	}
}

func TestWorkflowJudgeVerify_RejectsNilWorkflow(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	result, err := judge.Verify(context.Background(), nil)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(result.Violations))
	}
	if result.Violations[0].Description != "workflow is required" {
		t.Fatalf("Description = %q", result.Violations[0].Description)
	}
}

func TestWorkflowJudgeVerify_UsesCartaBranchForCartaSpecs(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2`

	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-carta",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	for _, violation := range result.Violations {
		if violation.Code == "carta_parse_error" {
			t.Fatalf("unexpected carta parse error: %#v", result.Violations)
		}
	}
}

func TestWorkflowJudgeVerify_AddsCartaParseErrorWhenCartaIsInvalid(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_confidence: invalid`

	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-carta-invalid",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	found := false
	for _, violation := range result.Violations {
		if violation.Code == "carta_parse_error" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Violations = %#v, want carta_parse_error", result.Violations)
	}
}

func TestWorkflowJudgeVerify_AddsCartaParseErrorWhenCartaHeaderIsBare(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := "CARTA"

	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-carta-bare",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	found := false
	for _, violation := range result.Violations {
		if violation.Code == "carta_parse_error" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Violations = %#v, want carta_parse_error", result.Violations)
	}
}

func TestWorkflowJudgeVerify_FlagsToolNotPermittedForCarta(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  PERMIT send_reply`

	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-carta-missing-permit",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	found := false
	for _, violation := range result.Violations {
		if violation.Code == "tool_not_permitted" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Violations = %#v, want tool_not_permitted", result.Violations)
	}
}
