package agent

import "testing"

func TestNewJudgeResult_PassedDependsOnViolations(t *testing.T) {
	t.Parallel()

	withWarnings := NewJudgeResult(nil, []Warning{{Code: "missing_spec", Description: "spec_source is missing"}})
	if !withWarnings.Passed {
		t.Fatal("Passed = false, want true when only warnings exist")
	}

	withViolations := NewJudgeResult([]Violation{{Code: "dsl_syntax_error", Description: "unexpected token"}}, nil)
	if withViolations.Passed {
		t.Fatal("Passed = true, want false when violations exist")
	}
}

func TestJudgeResult_AddViolationAndWarning(t *testing.T) {
	t.Parallel()

	result := NewJudgeResult(nil, nil)
	if !result.Passed {
		t.Fatal("Passed = false, want true")
	}

	result.AddWarning(Warning{Code: " missing_spec ", Description: " spec missing "})
	if !result.Passed {
		t.Fatal("Passed = false, want true with warnings only")
	}
	if len(result.Warnings) != 1 || result.Warnings[0].Code != "missing_spec" {
		t.Fatalf("Warnings = %#v", result.Warnings)
	}

	result.AddViolation(Violation{Code: " dsl_syntax_error ", Type: " syntax ", Description: " bad syntax ", Location: " DSL line 1 "})
	if result.Passed {
		t.Fatal("Passed = true, want false after violation")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(result.Violations))
	}
	got := result.Violations[0]
	if got.Code != "dsl_syntax_error" || got.Type != "syntax" || got.Description != "bad syntax" || got.Location != "DSL line 1" {
		t.Fatalf("Violation = %#v", got)
	}
}
