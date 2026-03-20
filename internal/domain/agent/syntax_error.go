package agent

import "fmt"

type SyntaxStage string

const (
	SyntaxStageLexer  SyntaxStage = "lexer"
	SyntaxStageParser SyntaxStage = "parser"
)

type SyntaxError interface {
	error
	Stage() SyntaxStage
	Position() Position
	UnexpectedToken() Token
	Message() string
}

type LexerError struct {
	Line   int
	Column int
	Reason string
	Found  Token
}

func (e *LexerError) Error() string {
	return fmt.Sprintf("%s error at line %d, column %d: %s", e.Stage(), e.Line, e.Column, e.Reason)
}

func (e *LexerError) Stage() SyntaxStage { return SyntaxStageLexer }

func (e *LexerError) Position() Position {
	return Position{Line: e.Line, Column: e.Column}
}

func (e *LexerError) UnexpectedToken() Token { return e.Found }

func (e *LexerError) Message() string { return e.Reason }

type ParserError struct {
	Line   int
	Column int
	Reason string
	Found  Token
}

func (e *ParserError) Error() string {
	return fmt.Sprintf("%s error at line %d, column %d: %s", e.Stage(), e.Line, e.Column, e.Reason)
}

func (e *ParserError) Stage() SyntaxStage { return SyntaxStageParser }

func (e *ParserError) Position() Position {
	return Position{Line: e.Line, Column: e.Column}
}

func (e *ParserError) UnexpectedToken() Token { return e.Found }

func (e *ParserError) Message() string { return e.Reason }
