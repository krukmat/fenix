package agent

import (
	"testing"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

func TestValidateWorkflowDSLSyntax_PassedForValidDSL(t *testing.T) {
	t.Parallel()

	result := ValidateWorkflowDSLSyntax(&workflowdomain.Workflow{
		ID:        "wf-valid",
		DSLSource: "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
	})

	if !result.Passed {
		t.Fatalf("Passed = false, want true; violations = %#v", result.Violations)
	}
	if result.Program == nil || result.Program.Workflow == nil {
		t.Fatal("expected parsed program")
	}
	if len(result.Violations) != 0 {
		t.Fatalf("len(Violations) = %d, want 0", len(result.Violations))
	}
}

func TestValidateWorkflowDSLSyntax_ParserErrorPreservesPosition(t *testing.T) {
	t.Parallel()

	result := ValidateWorkflowDSLSyntax(&workflowdomain.Workflow{
		ID:        "wf-parser-error",
		DSLSource: "ON case.created\nSET case.status = \"resolved\"",
	})

	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(result.Violations))
	}
	got := result.Violations[0]
	if got.Type != "dsl_syntax_error" {
		t.Fatalf("Type = %q, want dsl_syntax_error", got.Type)
	}
	if got.Line != 1 || got.Column != 1 {
		t.Fatalf("position = %d:%d, want 1:1", got.Line, got.Column)
	}
	if got.Location != "DSL line 1, column 1" {
		t.Fatalf("Location = %q", got.Location)
	}
}

func TestValidateWorkflowDSLSyntax_ValidationErrorPreservesPosition(t *testing.T) {
	t.Parallel()

	result := ValidateWorkflowDSLSyntax(&workflowdomain.Workflow{
		ID:        "wf-validation-error",
		DSLSource: "WORKFLOW resolve_support_case\nON case.created",
	})

	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(result.Violations))
	}
	got := result.Violations[0]
	if got.Type != "dsl_validation_error" {
		t.Fatalf("Type = %q, want dsl_validation_error", got.Type)
	}
	if got.Line != 1 || got.Column != 1 {
		t.Fatalf("position = %d:%d, want 1:1", got.Line, got.Column)
	}
	if got.Description == "" {
		t.Fatal("expected validation description")
	}
}

func TestValidateWorkflowDSLSyntax_RejectsMissingDSLSource(t *testing.T) {
	t.Parallel()

	result := ValidateWorkflowDSLSyntax(&workflowdomain.Workflow{
		ID:        "wf-empty",
		DSLSource: "   ",
	})

	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(result.Violations))
	}
	got := result.Violations[0]
	if got.Description != "dsl_source is required" {
		t.Fatalf("Description = %q", got.Description)
	}
}
