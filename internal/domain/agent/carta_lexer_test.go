package agent

import "testing"

func TestCartaLexerLexTokenizesGroundsBlock(t *testing.T) {
	t.Parallel()

	source := "GROUNDS\n  min_sources: 2\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	expectedTypes := []TokenType{
		TokenGrounds, TokenNewline,
		TokenIndent, TokenIdentifier, TokenColon, TokenNumber, TokenNewline,
		TokenDedent, TokenEOF,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("len(tokens) = %d, want %d", len(tokens), len(expectedTypes))
	}

	for i, want := range expectedTypes {
		if tokens[i].Type != want {
			t.Fatalf("token[%d].Type = %s, want %s", i, tokens[i].Type, want)
		}
	}

	if tokens[3].Literal != "min_sources" {
		t.Fatalf("token[3].Literal = %q, want %q", tokens[3].Literal, "min_sources")
	}
	if tokens[5].Literal != "2" {
		t.Fatalf("token[5].Literal = %q, want %q", tokens[5].Literal, "2")
	}
}

func TestCartaLexerLexTokenizesPermitRateSlash(t *testing.T) {
	t.Parallel()

	source := "PERMIT send_reply\n  rate: 10 / hour\n"
	tokens, err := NewCartaLexer().Lex(source)
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	for _, token := range tokens {
		if token.Type == TokenSlash {
			if token.Literal != "/" {
				t.Fatalf("slash literal = %q, want /", token.Literal)
			}
			return
		}
	}

	t.Fatal("expected TokenSlash in permit rate syntax")
}
