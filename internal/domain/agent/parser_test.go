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

func TestParseDSLParsesWaitStatement(t *testing.T) {
	t.Parallel()

	program, err := ParseDSL("WORKFLOW x\nON case.created\nWAIT 48 hours")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	waitStmt, ok := program.Workflow.Body[0].(*WaitStatement)
	if !ok {
		t.Fatalf("body[0] type = %T, want *WaitStatement", program.Workflow.Body[0])
	}
	if waitStmt.Amount != 48 || waitStmt.Unit != "hours" {
		t.Fatalf("wait = %#v, want amount=48 unit=hours", waitStmt)
	}
}

func TestParseDSLParsesDispatchStatement(t *testing.T) {
	t.Parallel()

	program, err := ParseDSL("WORKFLOW delegate_case\nON case.created\nDISPATCH TO support_agent WITH {\"case_id\":\"case-1\"}")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	dispatchStmt, ok := program.Workflow.Body[0].(*DispatchStatement)
	if !ok {
		t.Fatalf("body[0] type = %T, want *DispatchStatement", program.Workflow.Body[0])
	}
	if dispatchStmt.Target == nil || dispatchStmt.Target.Name != "support_agent" {
		t.Fatalf("target = %#v, want support_agent", dispatchStmt.Target)
	}
	if _, ok := dispatchStmt.Payload.(*ObjectLiteralExpr); !ok {
		t.Fatalf("payload type = %T, want *ObjectLiteralExpr", dispatchStmt.Payload)
	}
}

func TestParseDSLParsesSurfaceStatement(t *testing.T) {
	t.Parallel()

	program, err := ParseDSL("WORKFLOW surface_case\nON case.created\nSURFACE case TO salesperson.view WITH {\"value\":\"review\",\"metadata\":{\"channel\":\"triage\"}}")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	surfaceStmt, ok := program.Workflow.Body[0].(*SurfaceStatement)
	if !ok {
		t.Fatalf("body[0] type = %T, want *SurfaceStatement", program.Workflow.Body[0])
	}
	if surfaceStmt.Entity == nil || surfaceStmt.Entity.Name != "case" {
		t.Fatalf("entity = %#v, want case", surfaceStmt.Entity)
	}
	if surfaceStmt.View == nil || surfaceStmt.View.Name != "salesperson.view" {
		t.Fatalf("view = %#v, want salesperson.view", surfaceStmt.View)
	}
	if _, ok := surfaceStmt.Payload.(*ObjectLiteralExpr); !ok {
		t.Fatalf("payload type = %T, want *ObjectLiteralExpr", surfaceStmt.Payload)
	}
}

func TestParserErrorMessageAndError(t *testing.T) {
	t.Parallel()

	e := &ParserError{Line: 5, Column: 2, Reason: msgExpectedWorkflowDecl}
	if e.Message() != msgExpectedWorkflowDecl {
		t.Fatalf("Message() = %q, want %q", e.Message(), msgExpectedWorkflowDecl)
	}
	want := "parser error at line 5, column 2: " + msgExpectedWorkflowDecl
	if e.Error() != want {
		t.Fatalf("Error() = %q, want %q", e.Error(), want)
	}
}

func TestParserParsesNumberLiterals(t *testing.T) {
	t.Parallel()

	source := `WORKFLOW count_check
ON lead.created
SET lead.score = 42`
	program, err := ParseDSL(source)
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	if len(program.Workflow.Body) != 1 {
		t.Fatalf("body len = %d, want 1", len(program.Workflow.Body))
	}
	set, ok := program.Workflow.Body[0].(*SetStatement)
	if !ok {
		t.Fatalf("expected *SetStatement, got %T", program.Workflow.Body[0])
	}
	lit, ok := set.Value.(*LiteralExpr)
	if !ok {
		t.Fatalf("expected *LiteralExpr, got %T", set.Value)
	}
	if lit.Value != 42 {
		t.Fatalf("literal value = %v, want 42", lit.Value)
	}
}

func TestParseDSLParsesCallWithInputAndAlias(t *testing.T) { // CLSF-52
	t.Parallel()

	program, err := ParseDSL("WORKFLOW x\nON case.created\nCALL search WITH query AS result")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	stmt, ok := program.Workflow.Body[0].(*CallStatement)
	if !ok {
		t.Fatalf("body[0] type = %T, want *CallStatement", program.Workflow.Body[0])
	}
	if stmt.Tool == nil || stmt.Tool.Name != "search" {
		t.Fatalf("Tool.Name = %q, want search", stmt.Tool.Name)
	}
	if stmt.Input == nil {
		t.Fatal("Input must not be nil when WITH is present")
	}
	if stmt.Alias == nil || stmt.Alias.Name != "result" {
		t.Fatalf("Alias.Name = %q, want result", stmt.Alias.Name)
	}
}

func TestParseDSLParsesCallBareNoWithNoAs(t *testing.T) { // CLSF-52
	t.Parallel()

	program, err := ParseDSL("WORKFLOW x\nON case.created\nCALL ping")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	stmt, ok := program.Workflow.Body[0].(*CallStatement)
	if !ok {
		t.Fatalf("body[0] type = %T, want *CallStatement", program.Workflow.Body[0])
	}
	if stmt.Tool == nil || stmt.Tool.Name != "ping" {
		t.Fatalf("Tool.Name = %q, want ping", stmt.Tool.Name)
	}
	if stmt.Input != nil {
		t.Fatal("Input must be nil when WITH is absent")
	}
	if stmt.Alias != nil {
		t.Fatal("Alias must be nil when AS is absent")
	}
}

func TestParseDSLParsesCallWithInputNoAlias(t *testing.T) { // CLSF-52
	t.Parallel()

	program, err := ParseDSL("WORKFLOW x\nON case.created\nCALL search WITH query")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	stmt, ok := program.Workflow.Body[0].(*CallStatement)
	if !ok {
		t.Fatalf("body[0] type = %T, want *CallStatement", program.Workflow.Body[0])
	}
	if stmt.Input == nil {
		t.Fatal("Input must not be nil when WITH is present")
	}
	if stmt.Alias != nil {
		t.Fatal("Alias must be nil when AS is absent")
	}
}

func TestParseDSLRejectsCallMissingToolName(t *testing.T) { // CLSF-52
	t.Parallel()

	_, err := ParseDSL("WORKFLOW x\nON case.created\nCALL")
	if err == nil {
		t.Fatal("expected parser error for CALL without tool name")
	}
	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
}

func TestParseDSLParsesApproveWithRole(t *testing.T) { // CLSF-53
	t.Parallel()

	program, err := ParseDSL("WORKFLOW x\nON case.created\nAPPROVE send_email role manager")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	stmt, ok := program.Workflow.Body[0].(*ApproveStatement)
	if !ok {
		t.Fatalf("body[0] type = %T, want *ApproveStatement", program.Workflow.Body[0])
	}
	if stmt.Stage == nil || stmt.Stage.Name != "send_email" {
		t.Fatalf("Stage.Name = %q, want send_email", stmt.Stage.Name)
	}
	if stmt.Role == nil || stmt.Role.Name != "manager" {
		t.Fatalf("Role.Name = %q, want manager", stmt.Role.Name)
	}
}

func TestParseDSLParsesApproveBareNoRole(t *testing.T) { // CLSF-53
	t.Parallel()

	program, err := ParseDSL("WORKFLOW x\nON case.created\nAPPROVE send_email")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	stmt, ok := program.Workflow.Body[0].(*ApproveStatement)
	if !ok {
		t.Fatalf("body[0] type = %T, want *ApproveStatement", program.Workflow.Body[0])
	}
	if stmt.Stage == nil || stmt.Stage.Name != "send_email" {
		t.Fatalf("Stage.Name = %q, want send_email", stmt.Stage.Name)
	}
	if stmt.Role != nil {
		t.Fatal("Role must be nil when role keyword is absent")
	}
}

func TestParseDSLRejectsApproveMissingStageName(t *testing.T) { // CLSF-53
	t.Parallel()

	_, err := ParseDSL("WORKFLOW x\nON case.created\nAPPROVE")
	if err == nil {
		t.Fatal("expected parser error for APPROVE without stage name")
	}
	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
}

func TestParseDSLRejectsApproveRoleMissingName(t *testing.T) { // CLSF-53
	t.Parallel()

	_, err := ParseDSL("WORKFLOW x\nON case.created\nAPPROVE send_email role")
	if err == nil {
		t.Fatal("expected parser error for APPROVE role without name")
	}
	var parseErr *ParserError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParserError, got %T", err)
	}
}

func TestParseDSLApproveCarriesPosition(t *testing.T) { // CLSF-53
	t.Parallel()

	program, err := ParseDSL("WORKFLOW x\nON case.created\nAPPROVE send_email role manager")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	stmt := program.Workflow.Body[0].(*ApproveStatement)
	if stmt.Pos().Line == 0 {
		t.Fatal("ApproveStatement must carry a non-zero position")
	}
}

func TestParseDSLCallCarriesPosition(t *testing.T) { // CLSF-52
	t.Parallel()

	program, err := ParseDSL("WORKFLOW x\nON case.created\nCALL search WITH query AS result")
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}
	stmt := program.Workflow.Body[0].(*CallStatement)
	if stmt.Pos().Line == 0 {
		t.Fatal("CallStatement must carry a non-zero position")
	}
}
