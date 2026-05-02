package eval

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	textContractFieldForbiddenClaim  = "forbidden_claim"
	textContractFieldLength          = "max_length"
	textContractFieldSection         = "required_section"
	textContractFieldSourceID        = "required_source_id"
	textContractFieldSourceReference = "permitted_source_id"
	textContractFieldUncertainty     = "uncertainty_statement"
	textContractActualMissing        = "missing"
	textContractExpectedUncertainty  = "uncertainty statement present"
	textContractHighConfidenceClaim  = "unsupported high-confidence claim"
)

var (
	sectionLinePrefix = regexp.MustCompile(`^[#*\-\d.\)\s]+`)
	sourceIDPattern   = regexp.MustCompile(`\b[A-Za-z]{2,}-[A-Za-z0-9]+\b`)
)

var defaultUncertaintyMarkers = []string{
	"uncertainty:",
	"confidence:",
	"cannot confirm",
	"not enough information",
	"based on the available information",
	"may ",
	"might ",
	"unknown",
	"unclear",
}

var defaultUnsupportedHighConfidenceClaims = []string{
	"guaranteed",
	"definitely resolved",
	"certainly resolved",
	"no risk",
	"without any risk",
}

// ResponseContract defines deterministic structural and lexical rules for generated text.
type ResponseContract struct {
	RequiredSections             []string `yaml:"required_sections" json:"required_sections"`
	RequiredSourceIDs            []string `yaml:"required_source_ids" json:"required_source_ids"`
	PermittedSourceIDs           []string `yaml:"permitted_source_ids,omitempty" json:"permitted_source_ids,omitempty"`
	ForbiddenClaims              []string `yaml:"forbidden_claims" json:"forbidden_claims"`
	MaxLength                    int      `yaml:"max_length" json:"max_length"`
	RequiresUncertaintyStatement bool     `yaml:"requires_uncertainty_statement" json:"requires_uncertainty_statement"`
}

// TextContractViolation records one deterministic contract failure.
type TextContractViolation struct {
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Evidence string `json:"evidence"`
}

// TextContractResult reports whether a generated text satisfies the contract.
type TextContractResult struct {
	Pass       bool                    `json:"pass"`
	Violations []TextContractViolation `json:"violations,omitempty"`
}

// ValidateTextOutput checks generated text against a structural and lexical contract.
func ValidateTextOutput(contract ResponseContract, text string) TextContractResult {
	var violations []TextContractViolation

	violations = append(violations, validateRequiredSections(contract.RequiredSections, text)...)
	violations = append(violations, validateRequiredSourceIDs(contract.RequiredSourceIDs, text)...)
	violations = append(violations, validatePermittedSourceIDs(contract.PermittedSourceIDs, text)...)
	violations = append(violations, validateForbiddenClaims(contract.ForbiddenClaims, text)...)
	violations = append(violations, validateUnsupportedHighConfidenceClaims(text)...)
	violations = append(violations, validateMaxLength(contract.MaxLength, text)...)
	violations = append(violations, validateUncertaintyStatement(contract.RequiresUncertaintyStatement, text)...)

	sortTextContractViolations(violations)
	return TextContractResult{
		Pass:       len(violations) == 0,
		Violations: cloneTextContractViolations(violations),
	}
}

func validateRequiredSections(required []string, text string) []TextContractViolation {
	out := make([]TextContractViolation, 0, len(required))
	for _, section := range required {
		if hasRequiredSection(text, section) {
			continue
		}
		out = append(out, TextContractViolation{
			Field:    textContractFieldSection,
			Expected: section,
			Actual:   textContractActualMissing,
			Evidence: fmt.Sprintf("required section %q not found in generated text", section),
		})
	}
	return out
}

func validateRequiredSourceIDs(required []string, text string) []TextContractViolation {
	actualSet := extractedSourceIDSet(text)
	out := make([]TextContractViolation, 0, len(required))
	for _, sourceID := range required {
		if _, ok := actualSet[strings.ToUpper(strings.TrimSpace(sourceID))]; ok {
			continue
		}
		out = append(out, TextContractViolation{
			Field:    textContractFieldSourceID,
			Expected: sourceID,
			Actual:   textContractActualMissing,
			Evidence: fmt.Sprintf("required source ID %q not referenced in generated text", sourceID),
		})
	}
	return out
}

func validatePermittedSourceIDs(permitted []string, text string) []TextContractViolation {
	if len(permitted) == 0 {
		return nil
	}

	permittedSet := stringSet(upperTrimmedStrings(permitted))
	actualIDs := sortedStringSetKeys(extractedSourceIDSet(text))
	out := make([]TextContractViolation, 0, len(actualIDs))
	for _, sourceID := range actualIDs {
		if _, ok := permittedSet[sourceID]; ok {
			continue
		}
		out = append(out, TextContractViolation{
			Field:    textContractFieldSourceReference,
			Expected: fmt.Sprintf("one of %v", sortedStringSetKeys(permittedSet)),
			Actual:   sourceID,
			Evidence: fmt.Sprintf("source ID %q is referenced but not permitted by the contract", sourceID),
		})
	}
	return out
}

func validateForbiddenClaims(forbidden []string, text string) []TextContractViolation {
	normalized := strings.ToLower(text)
	out := make([]TextContractViolation, 0, len(forbidden))
	for _, claim := range forbidden {
		needle := strings.ToLower(strings.TrimSpace(claim))
		if needle == "" || !strings.Contains(normalized, needle) {
			continue
		}
		out = append(out, TextContractViolation{
			Field:    textContractFieldForbiddenClaim,
			Expected: fmt.Sprintf(fmtAbsentValue, claim),
			Actual:   claim,
			Evidence: fmt.Sprintf("forbidden claim %q found in generated text", claim),
		})
	}
	return out
}

func validateUnsupportedHighConfidenceClaims(text string) []TextContractViolation {
	normalized := strings.ToLower(text)
	out := make([]TextContractViolation, 0, len(defaultUnsupportedHighConfidenceClaims))
	for _, claim := range defaultUnsupportedHighConfidenceClaims {
		if !strings.Contains(normalized, claim) {
			continue
		}
		out = append(out, TextContractViolation{
			Field:    textContractFieldForbiddenClaim,
			Expected: fmt.Sprintf(fmtAbsentValue, textContractHighConfidenceClaim),
			Actual:   claim,
			Evidence: fmt.Sprintf("unsupported high-confidence claim %q found in generated text", claim),
		})
	}
	return out
}

func validateMaxLength(maxLength int, text string) []TextContractViolation {
	if maxLength <= 0 {
		return nil
	}

	actualLength := utf8.RuneCountInString(text)
	if actualLength <= maxLength {
		return nil
	}
	return []TextContractViolation{{
		Field:    textContractFieldLength,
		Expected: fmt.Sprintf("<= %d", maxLength),
		Actual:   fmt.Sprintf("%d", actualLength),
		Evidence: fmt.Sprintf("generated text length %d exceeds max_length %d", actualLength, maxLength),
	}}
}

func validateUncertaintyStatement(required bool, text string) []TextContractViolation {
	if !required || hasUncertaintyStatement(text) {
		return nil
	}
	return []TextContractViolation{{
		Field:    textContractFieldUncertainty,
		Expected: textContractExpectedUncertainty,
		Actual:   textContractActualMissing,
		Evidence: "required uncertainty or confidence statement not found in generated text",
	}}
}

func hasRequiredSection(text, section string) bool {
	normalizedSection := normalizeSectionLabel(section)
	for _, line := range strings.Split(text, "\n") {
		if normalizeSectionLabel(line) == normalizedSection {
			return true
		}
	}
	return false
}

func hasUncertaintyStatement(text string) bool {
	normalized := strings.ToLower(text)
	for _, marker := range defaultUncertaintyMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func normalizeSectionLabel(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	trimmed = sectionLinePrefix.ReplaceAllString(trimmed, "")
	trimmed = strings.TrimSuffix(trimmed, ":")
	return strings.TrimSpace(trimmed)
}

func extractedSourceIDSet(text string) map[string]struct{} {
	matches := sourceIDPattern.FindAllString(text, -1)
	out := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		out[strings.ToUpper(strings.TrimSpace(match))] = struct{}{}
	}
	return out
}

func upperTrimmedStrings(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out = append(out, strings.ToUpper(trimmed))
	}
	return out
}

func sortTextContractViolations(items []TextContractViolation) {
	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.Field != right.Field {
			return left.Field < right.Field
		}
		if left.Expected != right.Expected {
			return left.Expected < right.Expected
		}
		if left.Actual != right.Actual {
			return left.Actual < right.Actual
		}
		return left.Evidence < right.Evidence
	})
}

func cloneTextContractViolations(in []TextContractViolation) []TextContractViolation {
	if len(in) == 0 {
		return nil
	}
	out := make([]TextContractViolation, len(in))
	copy(out, in)
	return out
}
