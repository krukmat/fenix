package agent

import "testing"

func TestSemanticIDDeterministicForSameInput(t *testing.T) {
	t.Parallel()

	input := SemanticIDInput{
		Kind:    SemanticNodeAction,
		Source:  SemanticSourceDSL,
		Scope:   "resolve_support_case",
		Ordinal: 2,
		Parts:   []string{"NOTIFY", "salesperson", "review this case"},
	}

	first := NewSemanticNodeID(input)
	second := NewSemanticNodeID(input)
	if first != second {
		t.Fatalf("IDs differ for same input: %q != %q", first, second)
	}
}

func TestSemanticIDIgnoresWhitespaceAndCaseInParts(t *testing.T) {
	t.Parallel()

	first := NewSemanticNodeID(SemanticIDInput{
		Kind:    SemanticNodeAction,
		Source:  SemanticSourceDSL,
		Scope:   " resolve_support_case ",
		Ordinal: 1,
		Parts:   []string{"NOTIFY", "salesperson", "review   this\ncase"},
	})
	second := NewSemanticNodeID(SemanticIDInput{
		Kind:    SemanticNodeAction,
		Source:  SemanticSourceDSL,
		Scope:   "RESOLVE_SUPPORT_CASE",
		Ordinal: 1,
		Parts:   []string{" notify ", "SalesPerson", "review this case"},
	})

	if first != second {
		t.Fatalf("IDs should ignore whitespace and case: %q != %q", first, second)
	}
}

func TestSemanticIDDifferentOrdinalChangesID(t *testing.T) {
	t.Parallel()

	first := NewSemanticNodeID(SemanticIDInput{
		Kind:    SemanticNodeAction,
		Source:  SemanticSourceDSL,
		Scope:   "resolve_support_case",
		Ordinal: 1,
		Parts:   []string{"SET", "case.status", "resolved"},
	})
	second := NewSemanticNodeID(SemanticIDInput{
		Kind:    SemanticNodeAction,
		Source:  SemanticSourceDSL,
		Scope:   "resolve_support_case",
		Ordinal: 2,
		Parts:   []string{"SET", "case.status", "resolved"},
	})

	if first == second {
		t.Fatalf("IDs should differ by ordinal: %q", first)
	}
}

func TestSemanticIDIncludesNodeKindPrefix(t *testing.T) {
	t.Parallel()

	id := NewSemanticNodeID(SemanticIDInput{
		Kind:   SemanticNodePermit,
		Source: SemanticSourceCarta,
		Scope:  "resolve_support_case",
		Parts:  []string{"send_reply"},
	})

	if got := string(id); len(got) <= len("permit:") || got[:len("permit:")] != "permit:" {
		t.Fatalf("ID = %q, want permit prefix", id)
	}
}
