package agent

import (
	"errors"
	"testing"
)

func TestLexerLexTokenizesWorkflowSource(t *testing.T) {
	t.Parallel()

	source := `WORKFLOW resolve_support_case
ON case.created
IF case.priority IN ["high", "urgent"]:
  SET case.status = "resolved"
  NOTIFY contact WITH "done"
AGENT search_knowledge WITH case`

	tokens, err := NewLexer(source).Lex()
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	expectedTypes := []TokenType{
		TokenWorkflow, TokenIdentifier, TokenNewline,
		TokenOn, TokenIdentifier, TokenNewline,
		TokenIf, TokenIdentifier, TokenIn, TokenLBracket, TokenString, TokenComma, TokenString, TokenRBracket, TokenColon, TokenNewline,
		TokenIndent,
		TokenSet, TokenIdentifier, TokenAssign, TokenString, TokenNewline,
		TokenNotify, TokenIdentifier, TokenWith, TokenString, TokenNewline,
		TokenDedent,
		TokenAgent, TokenIdentifier, TokenWith, TokenIdentifier, TokenNewline,
		TokenEOF,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("len(tokens) = %d, want %d", len(tokens), len(expectedTypes))
	}

	for i, want := range expectedTypes {
		if tokens[i].Type != want {
			t.Fatalf("token[%d].Type = %s, want %s", i, tokens[i].Type, want)
		}
	}
}

func TestLexerLexPreservesLineAndColumn(t *testing.T) {
	t.Parallel()

	source := "WORKFLOW x\n  SET case.status = \"resolved\""
	tokens, err := NewLexer(source).Lex()
	if err != nil {
		t.Fatalf("Lex() error = %v", err)
	}

	if tokens[0].Line != 1 || tokens[0].Column != 1 {
		t.Fatalf("WORKFLOW token position = %d:%d, want 1:1", tokens[0].Line, tokens[0].Column)
	}

	foundSet := false
	for _, token := range tokens {
		if token.Type == TokenSet {
			foundSet = true
			if token.Line != 2 || token.Column != 3 {
				t.Fatalf("SET token position = %d:%d, want 2:3", token.Line, token.Column)
			}
		}
	}
	if !foundSet {
		t.Fatal("expected SET token")
	}
}

func TestLexerLexRejectsUnterminatedString(t *testing.T) {
	t.Parallel()

	_, err := NewLexer(`NOTIFY contact WITH "oops`).Lex()
	if err == nil {
		t.Fatal("expected lexer error")
	}

	var lexErr *LexerError
	if !errors.As(err, &lexErr) {
		t.Fatalf("expected LexerError, got %T", err)
	}
	if lexErr.Line != 1 {
		t.Fatalf("lexErr.Line = %d, want 1", lexErr.Line)
	}
	if lexErr.Stage() != SyntaxStageLexer {
		t.Fatalf("Stage() = %s, want %s", lexErr.Stage(), SyntaxStageLexer)
	}
	if lexErr.UnexpectedToken().Type != TokenString {
		t.Fatalf("UnexpectedToken.Type = %s, want %s", lexErr.UnexpectedToken().Type, TokenString)
	}
}

func TestLexerLexRejectsInconsistentIndentation(t *testing.T) {
	t.Parallel()

	source := "WORKFLOW x\n  IF case.priority == \"high\":\n    SET case.status = \"resolved\"\n SET case.priority = \"high\""
	_, err := NewLexer(source).Lex()
	if err == nil {
		t.Fatal("expected lexer error")
	}

	var lexErr *LexerError
	if !errors.As(err, &lexErr) {
		t.Fatalf("expected LexerError, got %T", err)
	}
	if lexErr.Reason != "inconsistent indentation" {
		t.Fatalf("lexErr.Reason = %q, want %q", lexErr.Reason, "inconsistent indentation")
	}
	if lexErr.Position().Line != 4 || lexErr.Position().Column != 1 {
		t.Fatalf("Position() = %+v, want 4:1", lexErr.Position())
	}
}
