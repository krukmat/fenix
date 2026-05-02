package eval_test

import (
	"strings"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/eval"
)

func TestTextContractMissingSection(t *testing.T) {
	t.Parallel()

	contract := eval.ResponseContract{
		RequiredSections: []string{"summary", "evidence"},
	}
	text := "## Summary\nDone.\n\n## Recommendation\nProceed."

	result := eval.ValidateTextOutput(contract, text)

	if result.Pass {
		t.Fatal("expected Pass=false when a required section is missing")
	}
	if !hasTextContractViolation(result.Violations, "required_section", "evidence", "missing") {
		t.Fatalf("expected missing required_section violation, got %#v", result.Violations)
	}
}

func TestTextContractForbiddenClaimGuaranteed(t *testing.T) {
	t.Parallel()

	contract := eval.ResponseContract{
		ForbiddenClaims: []string{"guaranteed"},
	}
	text := "This is guaranteed to solve the issue."

	result := eval.ValidateTextOutput(contract, text)

	if result.Pass {
		t.Fatal("expected Pass=false when a forbidden claim is present")
	}
	if !hasTextContractViolation(result.Violations, "forbidden_claim", "absent: guaranteed", "guaranteed") {
		t.Fatalf("expected forbidden_claim violation, got %#v", result.Violations)
	}
}

func TestTextContractSourceIDMissing(t *testing.T) {
	t.Parallel()

	contract := eval.ResponseContract{
		RequiredSourceIDs: []string{"KB-102"},
	}
	text := "## Summary\nUsed KB-101 for the answer."

	result := eval.ValidateTextOutput(contract, text)

	if result.Pass {
		t.Fatal("expected Pass=false when a required source ID is missing")
	}
	if !hasTextContractViolation(result.Violations, "required_source_id", "KB-102", "missing") {
		t.Fatalf("expected required_source_id violation, got %#v", result.Violations)
	}
}

func TestTextContractExceedsMaxLength(t *testing.T) {
	t.Parallel()

	contract := eval.ResponseContract{
		MaxLength: 20,
	}
	text := "This answer is definitely longer than twenty characters."

	result := eval.ValidateTextOutput(contract, text)

	if result.Pass {
		t.Fatal("expected Pass=false when max_length is exceeded")
	}
	if !hasTextContractViolation(result.Violations, "max_length", "<= 20", "56") {
		t.Fatalf("expected max_length violation, got %#v", result.Violations)
	}
}

func TestTextContractMissingUncertaintyStatement(t *testing.T) {
	t.Parallel()

	contract := eval.ResponseContract{
		RequiresUncertaintyStatement: true,
	}
	text := "## Summary\nIssue appears resolved.\n\n## Evidence\nKB-102"

	result := eval.ValidateTextOutput(contract, text)

	if result.Pass {
		t.Fatal("expected Pass=false when the uncertainty statement is missing")
	}
	if !hasTextContractViolation(result.Violations, "uncertainty_statement", "uncertainty statement present", "missing") {
		t.Fatalf("expected uncertainty_statement violation, got %#v", result.Violations)
	}
}

func TestTextContractUnpermittedSourceID(t *testing.T) {
	t.Parallel()

	contract := eval.ResponseContract{
		PermittedSourceIDs: []string{"KB-102"},
	}
	text := "## Evidence\nKB-999 supports this answer."

	result := eval.ValidateTextOutput(contract, text)

	if result.Pass {
		t.Fatal("expected Pass=false when an unpermitted source ID is referenced")
	}
	if !hasTextContractViolation(result.Violations, "permitted_source_id", "one of [KB-102]", "KB-999") {
		t.Fatalf("expected permitted_source_id violation, got %#v", result.Violations)
	}
}

func TestTextContractUnsupportedHighConfidenceClaim(t *testing.T) {
	t.Parallel()

	contract := eval.ResponseContract{}
	text := "The incident is definitely resolved and there is no risk remaining."

	result := eval.ValidateTextOutput(contract, text)

	if result.Pass {
		t.Fatal("expected Pass=false when unsupported high-confidence claims are present")
	}
	if !hasTextContractViolation(result.Violations, "forbidden_claim", "absent: unsupported high-confidence claim", "definitely resolved") {
		t.Fatalf("expected unsupported high-confidence claim violation, got %#v", result.Violations)
	}
}

func TestTextContractAllPass(t *testing.T) {
	t.Parallel()

	contract := eval.ResponseContract{
		RequiredSections:             []string{"summary", "evidence", "recommendation", "uncertainty"},
		RequiredSourceIDs:            []string{"KB-102"},
		PermittedSourceIDs:           []string{"KB-102"},
		ForbiddenClaims:              []string{"guaranteed", "no risk"},
		MaxLength:                    400,
		RequiresUncertaintyStatement: true,
	}
	text := strings.Join([]string{
		"## Summary",
		"The issue appears partially mitigated.",
		"",
		"## Evidence",
		"KB-102 documents the fallback behavior.",
		"",
		"## Recommendation",
		"Roll out the workaround and monitor the next run.",
		"",
		"## Uncertainty",
		"Based on the available information, I cannot confirm a permanent fix yet.",
	}, "\n")

	result := eval.ValidateTextOutput(contract, text)

	if !result.Pass {
		t.Fatalf("expected Pass=true, got violations %#v", result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("expected no violations, got %#v", result.Violations)
	}
}

func hasTextContractViolation(violations []eval.TextContractViolation, field, expected, actual string) bool {
	for _, violation := range violations {
		if violation.Field == field && violation.Expected == expected && violation.Actual == actual {
			return true
		}
	}
	return false
}
