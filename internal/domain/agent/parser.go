package agent

import (
	"strconv"
	"strings"
)

const msgExpectedWorkflowDecl = "expected WORKFLOW declaration"

type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens}
}

func ParseDSL(source string) (*Program, error) {
	tokens, err := NewLexer(source).Lex()
	if err != nil {
		return nil, err
	}
	return NewParser(tokens).ParseProgram()
}

func (p *Parser) ParseProgram() (*Program, error) {
	p.skipNewlines()

	workflow, err := p.parseWorkflow()
	if err != nil {
		return nil, err
	}

	p.skipNewlines()
	if tok := p.current(); tok.Type != TokenEOF {
		return nil, p.errorAt(tok, "unexpected tokens after workflow body")
	}

	return &Program{Workflow: workflow}, nil
}

func (p *Parser) parseWorkflow() (*WorkflowDecl, error) {
	start, err := p.expect(TokenWorkflow, msgExpectedWorkflowDecl)
	if err != nil {
		return nil, err
	}
	name, err := p.expect(TokenIdentifier, "expected workflow name")
	if err != nil {
		return nil, err
	}
	if parseErr := p.expectNewline("expected newline after WORKFLOW header"); parseErr != nil {
		return nil, parseErr
	}

	on, err := p.parseOnDecl()
	if err != nil {
		return nil, err
	}

	body, err := p.parseStatementList(TokenEOF)
	if err != nil {
		return nil, err
	}

	return &WorkflowDecl{
		Name:     name.Literal,
		Trigger:  on,
		Body:     body,
		Position: positionFromToken(start),
	}, nil
}

func (p *Parser) parseOnDecl() (*OnDecl, error) {
	start, err := p.expect(TokenOn, "expected ON declaration")
	if err != nil {
		return nil, err
	}
	event, err := p.expect(TokenIdentifier, "expected trigger event after ON")
	if err != nil {
		return nil, err
	}
	if parseErr := p.expectNewline("expected newline after ON declaration"); parseErr != nil {
		return nil, parseErr
	}
	return &OnDecl{
		Event:    event.Literal,
		Position: positionFromToken(start),
	}, nil
}

func (p *Parser) parseStatementList(stop TokenType) ([]Statement, error) {
	statements := make([]Statement, 0)
	for {
		p.skipNewlines()
		current := p.current()
		if current.Type == stop || current.Type == TokenEOF || current.Type == TokenDedent {
			return statements, nil
		}
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		statements = append(statements, stmt)
	}
}

func (p *Parser) parseStatement() (Statement, error) {
	switch p.current().Type {
	case TokenIf:
		return p.parseIfStatement()
	case TokenSet:
		return p.parseSetStatement()
	case TokenNotify:
		return p.parseNotifyStatement()
	case TokenAgent:
		return p.parseAgentStatement()
	default:
		return nil, p.errorAt(p.current(), "unexpected statement")
	}
}

func (p *Parser) parseIfStatement() (Statement, error) {
	start, err := p.expect(TokenIf, "expected IF")
	if err != nil {
		return nil, err
	}
	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, parseErr := p.expect(TokenColon, "expected ':' after IF condition"); parseErr != nil {
		return nil, parseErr
	}
	if parseErr := p.expectNewline("expected newline after IF statement"); parseErr != nil {
		return nil, parseErr
	}
	body, err := p.parseIndentedStatementBlock(start)
	if err != nil {
		return nil, err
	}
	return &IfStatement{Condition: condition, Body: body, Position: positionFromToken(start)}, nil
}

func (p *Parser) parseIndentedStatementBlock(start Token) ([]Statement, error) {
	if _, parseErr := p.expect(TokenIndent, "expected indented block after IF"); parseErr != nil {
		return nil, parseErr
	}
	body, err := p.parseStatementList(TokenDedent)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, p.errorAt(start, "IF block must contain at least one statement")
	}
	if _, parseErr := p.expect(TokenDedent, "expected end of IF block"); parseErr != nil {
		return nil, parseErr
	}
	return body, nil
}

func (p *Parser) parseSetStatement() (Statement, error) {
	start, err := p.expect(TokenSet, "expected SET")
	if err != nil {
		return nil, err
	}
	target, err := p.expect(TokenIdentifier, "expected target after SET")
	if err != nil {
		return nil, err
	}
	if _, parseErr := p.expect(TokenAssign, "expected '=' after SET target"); parseErr != nil {
		return nil, parseErr
	}
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if parseErr := p.expectNewline("expected newline after SET statement"); parseErr != nil {
		return nil, parseErr
	}
	return &SetStatement{
		Target:   &IdentifierExpr{Name: target.Literal, Position: positionFromToken(target)},
		Value:    value,
		Position: positionFromToken(start),
	}, nil
}

func (p *Parser) parseNotifyStatement() (Statement, error) {
	start, err := p.expect(TokenNotify, "expected NOTIFY")
	if err != nil {
		return nil, err
	}
	target, err := p.expect(TokenIdentifier, "expected target after NOTIFY")
	if err != nil {
		return nil, err
	}
	if _, parseErr := p.expect(TokenWith, "expected WITH after NOTIFY target"); parseErr != nil {
		return nil, parseErr
	}
	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if parseErr := p.expectNewline("expected newline after NOTIFY statement"); parseErr != nil {
		return nil, parseErr
	}
	return &NotifyStatement{
		Target:   &IdentifierExpr{Name: target.Literal, Position: positionFromToken(target)},
		Value:    value,
		Position: positionFromToken(start),
	}, nil
}

func (p *Parser) parseAgentStatement() (Statement, error) {
	start, err := p.expect(TokenAgent, "expected AGENT")
	if err != nil {
		return nil, err
	}
	name, err := p.expect(TokenIdentifier, "expected agent name after AGENT")
	if err != nil {
		return nil, err
	}
	var input Expression
	if p.current().Type == TokenWith {
		p.advance()
		input, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}
	if parseErr := p.expectNewline("expected newline after AGENT statement"); parseErr != nil {
		return nil, parseErr
	}
	return &AgentStatement{
		Name:     &IdentifierExpr{Name: name.Literal, Position: positionFromToken(name)},
		Input:    input,
		Position: positionFromToken(start),
	}, nil
}

func (p *Parser) parseExpression() (Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	if IsComparisonOperator(p.current().Type) {
		operator := p.current()
		p.advance()
		right, parseErr := p.parsePrimary()
		if parseErr != nil {
			return nil, parseErr
		}
		return &ComparisonExpr{
			Left:     left,
			Operator: operator.Type,
			Right:    right,
			Position: left.Pos(),
		}, nil
	}
	return left, nil
}

func (p *Parser) parsePrimary() (Expression, error) {
	tok := p.current()
	switch tok.Type {
	case TokenIdentifier:
		p.advance()
		return &IdentifierExpr{Name: tok.Literal, Position: positionFromToken(tok)}, nil
	case TokenString:
		return p.parseStringPrimary(tok)
	case TokenNumber:
		p.advance()
		return parseNumberLiteral(tok)
	case TokenBoolean:
		p.advance()
		return &LiteralExpr{Value: strings.EqualFold(tok.Literal, "true"), Position: positionFromToken(tok)}, nil
	case TokenNull:
		p.advance()
		return &LiteralExpr{Value: nil, Position: positionFromToken(tok)}, nil
	case TokenLBracket, TokenLBrace:
		return p.parseCollectionLiteral(tok)
	default:
		return nil, p.errorAt(tok, "expected expression")
	}
}

func (p *Parser) parseStringPrimary(tok Token) (Expression, error) {
	p.advance()
	value, err := strconv.Unquote(tok.Literal)
	if err != nil {
		return nil, p.errorAt(tok, "invalid string literal")
	}
	return &LiteralExpr{Value: value, Position: positionFromToken(tok)}, nil
}

func (p *Parser) parseCollectionLiteral(tok Token) (Expression, error) {
	if tok.Type == TokenLBracket {
		return p.parseArrayLiteral()
	}
	return p.parseObjectLiteral()
}

func parseNumberLiteral(tok Token) (Expression, error) {
	if strings.Contains(tok.Literal, ".") {
		value, err := strconv.ParseFloat(tok.Literal, 64)
		if err != nil {
			return nil, &ParserError{Line: tok.Line, Column: tok.Column, Reason: "invalid number literal"}
		}
		return &LiteralExpr{Value: value, Position: positionFromToken(tok)}, nil
	}
	value, err := strconv.Atoi(tok.Literal)
	if err != nil {
		return nil, &ParserError{Line: tok.Line, Column: tok.Column, Reason: "invalid number literal"}
	}
	return &LiteralExpr{Value: value, Position: positionFromToken(tok)}, nil
}

func (p *Parser) parseArrayLiteral() (Expression, error) {
	start, err := p.expect(TokenLBracket, "expected '['")
	if err != nil {
		return nil, err
	}
	elements := make([]Expression, 0)
	for p.current().Type != TokenRBracket {
		expr, parseErr := p.parseExpression()
		if parseErr != nil {
			return nil, parseErr
		}
		elements = append(elements, expr)
		if p.current().Type == TokenComma {
			p.advance()
			continue
		}
		if p.current().Type != TokenRBracket {
			return nil, p.errorAt(p.current(), "expected ',' or ']' in array literal")
		}
	}
	if _, parseErr := p.expect(TokenRBracket, "expected ']'"); parseErr != nil {
		return nil, parseErr
	}
	return &ArrayLiteralExpr{Elements: elements, Position: positionFromToken(start)}, nil
}

func (p *Parser) parseObjectLiteral() (Expression, error) {
	start, err := p.expect(TokenLBrace, "expected '{'")
	if err != nil {
		return nil, err
	}
	fields := make([]ObjectField, 0)
	for p.current().Type != TokenRBrace {
		field, parseErr := p.parseObjectField()
		if parseErr != nil {
			return nil, parseErr
		}
		fields = append(fields, field)
		if p.current().Type == TokenComma {
			p.advance()
			continue
		}
		if p.current().Type != TokenRBrace {
			return nil, p.errorAt(p.current(), "expected ',' or '}' in object literal")
		}
	}
	if _, parseErr := p.expect(TokenRBrace, "expected '}'"); parseErr != nil {
		return nil, parseErr
	}
	return &ObjectLiteralExpr{Fields: fields, Position: positionFromToken(start)}, nil
}

func (p *Parser) parseObjectField() (ObjectField, error) {
	keyTok := p.current()
	if keyTok.Type != TokenIdentifier && keyTok.Type != TokenString {
		return ObjectField{}, p.errorAt(keyTok, "expected object key")
	}
	p.advance()
	key := keyTok.Literal
	if keyTok.Type == TokenString {
		var err error
		key, err = strconv.Unquote(keyTok.Literal)
		if err != nil {
			return ObjectField{}, p.errorAt(keyTok, "invalid object key")
		}
	}
	if _, err := p.expect(TokenColon, "expected ':' after object key"); err != nil {
		return ObjectField{}, err
	}
	value, err := p.parseExpression()
	if err != nil {
		return ObjectField{}, err
	}
	return ObjectField{Key: key, Value: value, Position: positionFromToken(keyTok)}, nil
}

func (p *Parser) expect(tokenType TokenType, reason string) (Token, error) {
	current := p.current()
	if current.Type != tokenType {
		return Token{}, p.errorAt(current, reason)
	}
	p.advance()
	return current, nil
}

func (p *Parser) expectNewline(reason string) error {
	if p.current().Type != TokenNewline {
		return p.errorAt(p.current(), reason)
	}
	p.skipNewlines()
	return nil
}

func (p *Parser) skipNewlines() {
	for p.current().Type == TokenNewline {
		p.advance()
	}
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) errorAt(tok Token, reason string) error {
	return &ParserError{
		Line:   tok.Line,
		Column: tok.Column,
		Reason: reason,
		Found:  tok,
	}
}

func positionFromToken(tok Token) Position {
	return Position{Line: tok.Line, Column: tok.Column}
}
