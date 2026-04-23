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
		"to":       TokenTo,
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

func TestLookupTokenTypeRecognizesExtendedKeywords(t *testing.T) {
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

func TestLookupTokenTypeRecognizesCallAndApprove(t *testing.T) { // CLSF-50
	t.Parallel()

	cases := map[string]TokenType{
		"CALL":    TokenCall,
		"call":    TokenCall,
		"Call":    TokenCall,
		"APPROVE": TokenApprove,
		"approve": TokenApprove,
		"Approve": TokenApprove,
	}

	for input, want := range cases {
		if got := LookupTokenType(input); got != want {
			t.Fatalf("LookupTokenType(%q) = %s, want %s", input, got, want)
		}
	}
}

func TestCallAndApproveAreReservedNotActive(t *testing.T) { // CLSF-50
	t.Parallel()

	if IsKeyword(TokenCall) {
		t.Fatal("TokenCall must not be an active v0 keyword")
	}
	if IsKeyword(TokenApprove) {
		t.Fatal("TokenApprove must not be an active v0 keyword")
	}
	if !IsReservedKeyword(TokenCall) {
		t.Fatal("TokenCall must be a reserved keyword")
	}
	if !IsReservedKeyword(TokenApprove) {
		t.Fatal("TokenApprove must be a reserved keyword")
	}
}

func TestCallAndApproveDoNotBreakV0Identifiers(t *testing.T) { // CLSF-50
	t.Parallel()

	// Existing v0 keywords must be unaffected.
	v0 := map[string]TokenType{
		"WORKFLOW": TokenWorkflow,
		"SET":      TokenSet,
		"NOTIFY":   TokenNotify,
		"AGENT":    TokenAgent,
		"ON":       TokenOn,
		"IF":       TokenIf,
	}
	for input, want := range v0 {
		if got := LookupTokenType(input); got != want {
			t.Fatalf("v0 regression: LookupTokenType(%q) = %s, want %s", input, got, want)
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
	if !IsKeyword(TokenDispatch) {
		t.Fatal("expected TokenDispatch to be executable keyword")
	}
	if !IsKeyword(TokenSurface) {
		t.Fatal("expected TokenSurface to be executable keyword")
	}
	if IsReservedKeyword(TokenWait) {
		t.Fatal("expected TokenWait not to be reserved keyword")
	}
	if IsReservedKeyword(TokenSurface) {
		t.Fatal("expected TokenSurface not to be reserved keyword")
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
