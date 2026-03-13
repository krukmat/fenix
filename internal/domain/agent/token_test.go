package agent

import "testing"

func TestLookupTokenTypeRecognizesKeywords(t *testing.T) {
	t.Parallel()

	cases := map[string]TokenType{
		"WORKFLOW": TokenWorkflow,
		"on":       TokenOn,
		"If":       TokenIf,
		"SET":      TokenSet,
		"notify":   TokenNotify,
		"WITH":     TokenWith,
		"AGENT":    TokenAgent,
		"in":       TokenIn,
		"true":     TokenBoolean,
		"FALSE":    TokenBoolean,
		"null":     TokenNull,
	}

	for input, want := range cases {
		if got := LookupTokenType(input); got != want {
			t.Fatalf("LookupTokenType(%q) = %s, want %s", input, got, want)
		}
	}
}

func TestLookupTokenTypeRecognizesReservedKeywords(t *testing.T) {
	t.Parallel()

	cases := map[string]TokenType{
		"WAIT":     TokenWait,
		"dispatch": TokenDispatch,
		"Surface":  TokenSurface,
	}

	for input, want := range cases {
		if got := LookupTokenType(input); got != want {
			t.Fatalf("LookupTokenType(%q) = %s, want %s", input, got, want)
		}
	}
}

func TestLookupTokenTypeFallsBackToIdentifier(t *testing.T) {
	t.Parallel()

	cases := []string{
		"resolve_support_case",
		"case.status",
		"salesperson",
	}

	for _, input := range cases {
		if got := LookupTokenType(input); got != TokenIdentifier {
			t.Fatalf("LookupTokenType(%q) = %s, want %s", input, got, TokenIdentifier)
		}
	}
}

func TestTokenClassificationHelpers(t *testing.T) {
	t.Parallel()

	if !IsKeyword(TokenWorkflow) {
		t.Fatal("expected TokenWorkflow to be keyword")
	}
	if !IsKeyword(TokenWait) {
		t.Fatal("expected TokenWait to be executable keyword")
	}
	if IsReservedKeyword(TokenWait) {
		t.Fatal("expected TokenWait not to be reserved keyword")
	}
	if !IsLiteralToken(TokenString) || !IsLiteralToken(TokenIdentifier) {
		t.Fatal("expected literal token classification for string and identifier")
	}
	if !IsComparisonOperator(TokenIn) || !IsComparisonOperator(TokenGTE) {
		t.Fatal("expected comparison operators to be recognized")
	}
	if !IsStructuralToken(TokenIndent) || !IsStructuralToken(TokenColon) {
		t.Fatal("expected structural tokens to be recognized")
	}
	if IsStructuralToken(TokenSet) {
		t.Fatal("did not expect TokenSet to be structural")
	}
}
