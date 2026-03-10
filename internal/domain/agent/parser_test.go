package agent

import (
	"errors"
	"testing"
)

func TestParseDSLBuildsProgramForWorkflowV0(t *testing.T) {
	t.Parallel()

	source := `WORKFLOW resolve_support_case
ON case.created
IF case.priority IN ["high", "urgent"]:
  NOTIFY salesperson WITH "review this case"
SET case.status = "resolved"
AGENT search_knowledge WITH case`

	program, err := ParseDSL(source)
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	if program.Workflow == nil {
		t.Fatal("expected workflow")
	}
	if program.Workflow.Name != "resolve_support_case" {
		t.Fatalf("workflow name = %s", program.Workflow.Name)
	}
	if program.Workflow.Trigger == nil || program.Workflow.Trigger.Event != "case.created" {
		t.Fatalf("unexpected trigger = %#v", program.Workflow.Trigger)
	}
	if len(program.Workflow.Body) != 3 {
		t.Fatalf("len(body) = %d, want 3", len(program.Workflow.Body))
	}
	if _, ok := program.Workflow.Body[0].(*IfStatement); !ok {
		t.Fatalf("body[0] type = %T, want *IfStatement", program.Workflow.Body[0])
	}
	if _, ok := program.Workflow.Body[1].(*SetStatement); !ok {
		t.Fatalf("body[1] type = %T, want *SetStatement", program.Workflow.Body[1])
	}
	if _, ok := program.Workflow.Body[2].(*AgentStatement); !ok {
		t.Fatalf("body[2] type = %T, want *AgentStatement", program.Workflow.Body[2])
	}
}

func TestParseDSLParsesObjectLiteralInAgentInput(t *testing.T) {
	t.Parallel()

	source := `WORKFLOW resolve_support_case
ON case.created
AGENT evaluate_intent WITH {"entity":"case","mode":"fast"}`

	program, err := ParseDSL(source)
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	stmt, ok := program.Workflow.Body[0].(*AgentStatement)
	if !ok {
		t.Fatalf("body[0] type = %T, want *AgentStatement", program.Workflow.Body[0])
	}
	if _, ok := stmt.Input.(*ObjectLiteralExpr); !ok {
		t.Fatalf("agent input type = %T, want *ObjectLiteralExpr", stmt.Input)
	}
}

func TestParseDSLRejectsMissingWorkflowHeader(t *testing.T) {
	t.Parallel()

	_, err := ParseDSL("ON case.created\nSET case.status = \"resolved\"")
	if err == nil {
		t.Fatal("expected parser error")
	}
	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
	if parseErr.Stage() != SyntaxStageParser {
		t.Fatalf("Stage() = %s, want %s", parseErr.Stage(), SyntaxStageParser)
	}
	if parseErr.Position().Line != 1 || parseErr.Position().Column != 1 {
		t.Fatalf("Position() = %+v, want 1:1", parseErr.Position())
	}
	if parseErr.UnexpectedToken().Type != TokenOn {
		t.Fatalf("UnexpectedToken.Type = %s, want %s", parseErr.UnexpectedToken().Type, TokenOn)
	}
}

func TestParseDSLRejectsIfWithoutIndentedBlock(t *testing.T) {
	t.Parallel()

	_, err := ParseDSL("WORKFLOW x\nON case.created\nIF case.priority == \"high\":\nSET case.status = \"resolved\"")
	if err == nil {
		t.Fatal("expected parser error")
	}
	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
	if parseErr.UnexpectedToken().Type != TokenSet {
		t.Fatalf("UnexpectedToken.Type = %s, want %s", parseErr.UnexpectedToken().Type, TokenSet)
	}
}

func TestParseDSLRejectsReservedStatementInV0(t *testing.T) {
	t.Parallel()

	_, err := ParseDSL("WORKFLOW x\nON case.created\nWAIT 48")
	if err == nil {
		t.Fatal("expected parser error")
	}
	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
	if parseErr.UnexpectedToken().Type != TokenWait {
		t.Fatalf("UnexpectedToken.Type = %s, want %s", parseErr.UnexpectedToken().Type, TokenWait)
	}
}
