package agent

import "testing"

func TestRunInitialSpecDSLChecks_NoViolationsWhenBehaviorIsCovered(t *testing.T) {
	t.Parallel()

	spec := ParsePartialSpec(`BEHAVIOR resolve_support_case`)
	program, err := ParseAndValidateDSL("WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}

	violations, warnings := RunInitialSpecDSLChecks(spec, program)
	if len(violations) != 0 {
		t.Fatalf("Violations = %#v, want none", violations)
	}
	if len(warnings) != 0 {
		t.Fatalf("Warnings = %#v, want none", warnings)
	}
}

func TestRunInitialSpecDSLChecks_DetectsMissingBehaviorCoverage(t *testing.T) {
	t.Parallel()

	spec := ParsePartialSpec(`BEHAVIOR notify_salesperson`)
	program, err := ParseAndValidateDSL("WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}

	violations, _ := RunInitialSpecDSLChecks(spec, program)
	if len(violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(violations))
	}
	got := violations[0]
	if got.Code != "behavior_no_coverage" {
		t.Fatalf("Code = %q, want behavior_no_coverage", got.Code)
	}
	if got.CheckID != judgeCheckBehaviorCoverage {
		t.Fatalf("CheckID = %d, want %d", got.CheckID, judgeCheckBehaviorCoverage)
	}
}

func TestRunInitialSpecDSLChecks_UsesStatementCoverage(t *testing.T) {
	t.Parallel()

	spec := ParsePartialSpec(`BEHAVIOR notify_salesperson`)
	program, err := ParseAndValidateDSL("WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}

	violations, _ := RunInitialSpecDSLChecks(spec, program)
	if len(violations) != 0 {
		t.Fatalf("Violations = %#v, want none", violations)
	}
}
