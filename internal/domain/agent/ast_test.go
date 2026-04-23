package agent

import "testing"

func TestProgramPosReturnsWorkflowPosition(t *testing.T) {
	t.Parallel()

	program := &Program{
		Workflow: &WorkflowDecl{
			Name:     "resolve_support_case",
			Position: Position{Line: 1, Column: 1},
		},
	}

	pos := program.Pos()
	if pos.Line != 1 || pos.Column != 1 {
		t.Fatalf("program.Pos() = %+v, want 1:1", pos)
	}
}

func TestASTStatementsCarryPosition(t *testing.T) {
	t.Parallel()

	stmt := &SetStatement{
		Target:   &IdentifierExpr{Name: "case.status", Position: Position{Line: 4, Column: 5}},
		Value:    &LiteralExpr{Value: "resolved", Position: Position{Line: 4, Column: 19}},
		Position: Position{Line: 4, Column: 1},
	}

	if stmt.Pos().Line != 4 || stmt.Pos().Column != 1 {
		t.Fatalf("stmt.Pos() = %+v, want 4:1", stmt.Pos())
	}
	if stmt.Target.Pos().Column != 5 {
		t.Fatalf("target.Pos() = %+v, want column 5", stmt.Target.Pos())
	}
}

func TestProgramStatementsReturnsWorkflowBody(t *testing.T) {
	t.Parallel()

	body := []Statement{
		&NotifyStatement{
			Target:   &IdentifierExpr{Name: "contact", Position: Position{Line: 3, Column: 8}},
			Value:    &LiteralExpr{Value: "done", Position: Position{Line: 3, Column: 21}},
			Position: Position{Line: 3, Column: 1},
		},
	}

	program := &Program{
		Workflow: &WorkflowDecl{
			Name:     "resolve_support_case",
			Trigger:  &OnDecl{Event: "case.created", Position: Position{Line: 2, Column: 1}},
			Body:     body,
			Position: Position{Line: 1, Column: 1},
		},
	}

	got := ProgramStatements(program)
	if len(got) != 1 {
		t.Fatalf("len(ProgramStatements) = %d, want 1", len(got))
	}
	if got[0].Pos().Line != 3 {
		t.Fatalf("statement line = %d, want 3", got[0].Pos().Line)
	}
}

func TestASTNodeTypeMarkerMethods(t *testing.T) {
	t.Parallel()

	// OnDecl.Pos
	on := &OnDecl{Event: "case.created", Position: Position{Line: 2, Column: 1}}
	if on.Pos().Line != 2 {
		t.Fatalf("OnDecl.Pos().Line = %d, want 2", on.Pos().Line)
	}

	// IfStatement marker
	ifStmt := &IfStatement{Position: Position{Line: 3, Column: 1}}
	ifStmt.statementNode()
	if ifStmt.Pos().Line != 3 {
		t.Fatalf("IfStatement.Pos().Line = %d, want 3", ifStmt.Pos().Line)
	}

	// SetStatement marker
	setStmt := &SetStatement{Position: Position{Line: 4, Column: 1}}
	setStmt.statementNode()

	// NotifyStatement marker
	notifyStmt := &NotifyStatement{Position: Position{Line: 5, Column: 1}}
	notifyStmt.statementNode()

	// AgentStatement marker
	agentStmt := &AgentStatement{Position: Position{Line: 6, Column: 1}}
	agentStmt.statementNode()

	// IdentifierExpr marker and Pos
	ident := &IdentifierExpr{Name: "x", Position: Position{Line: 1, Column: 2}}
	ident.expressionNode()

	// LiteralExpr marker
	lit := &LiteralExpr{Value: "v", Position: Position{Line: 1, Column: 3}}
	lit.expressionNode()

	// ArrayLiteralExpr Pos and marker
	arr := &ArrayLiteralExpr{Position: Position{Line: 7, Column: 4}}
	arr.expressionNode()
	if arr.Pos().Column != 4 {
		t.Fatalf("ArrayLiteralExpr.Pos().Column = %d, want 4", arr.Pos().Column)
	}

	// ObjectLiteralExpr Pos and marker
	obj := &ObjectLiteralExpr{Position: Position{Line: 8, Column: 5}}
	obj.expressionNode()
	if obj.Pos().Column != 5 {
		t.Fatalf("ObjectLiteralExpr.Pos().Column = %d, want 5", obj.Pos().Column)
	}

	// ComparisonExpr marker
	cmp := &ComparisonExpr{Position: Position{Line: 9, Column: 1}}
	cmp.expressionNode()
}

func TestCallStatementImplementsStatementInterface(t *testing.T) { // CLSF-51
	t.Parallel()

	stmt := &CallStatement{
		Tool:     &IdentifierExpr{Name: "search", Position: Position{Line: 3, Column: 6}},
		Input:    &LiteralExpr{Value: "query", Position: Position{Line: 3, Column: 13}},
		Alias:    &IdentifierExpr{Name: "result", Position: Position{Line: 3, Column: 25}},
		Position: Position{Line: 3, Column: 1},
	}

	var _ Statement = stmt // compile-time interface check

	if stmt.Pos().Line != 3 || stmt.Pos().Column != 1 {
		t.Fatalf("CallStatement.Pos() = %+v, want 3:1", stmt.Pos())
	}
	if stmt.Tool.Name != "search" {
		t.Fatalf("Tool.Name = %q, want search", stmt.Tool.Name)
	}
	if stmt.Alias.Name != "result" {
		t.Fatalf("Alias.Name = %q, want result", stmt.Alias.Name)
	}
}

func TestCallStatementInputIsOptional(t *testing.T) { // CLSF-51
	t.Parallel()

	stmt := &CallStatement{
		Tool:     &IdentifierExpr{Name: "ping", Position: Position{Line: 5, Column: 6}},
		Position: Position{Line: 5, Column: 1},
	}

	if stmt.Input != nil {
		t.Fatal("CallStatement.Input must be nil when not provided")
	}
	if stmt.Alias != nil {
		t.Fatal("CallStatement.Alias must be nil when not provided")
	}
}

func TestApproveStatementImplementsStatementInterface(t *testing.T) { // CLSF-51
	t.Parallel()

	stmt := &ApproveStatement{
		Stage:    &IdentifierExpr{Name: "send_email", Position: Position{Line: 4, Column: 9}},
		Role:     &IdentifierExpr{Name: "manager", Position: Position{Line: 4, Column: 25}},
		Position: Position{Line: 4, Column: 1},
	}

	var _ Statement = stmt // compile-time interface check

	if stmt.Pos().Line != 4 || stmt.Pos().Column != 1 {
		t.Fatalf("ApproveStatement.Pos() = %+v, want 4:1", stmt.Pos())
	}
	if stmt.Stage.Name != "send_email" {
		t.Fatalf("Stage.Name = %q, want send_email", stmt.Stage.Name)
	}
	if stmt.Role.Name != "manager" {
		t.Fatalf("Role.Name = %q, want manager", stmt.Role.Name)
	}
}

func TestApproveStatementRoleIsOptional(t *testing.T) { // CLSF-51
	t.Parallel()

	stmt := &ApproveStatement{
		Stage:    &IdentifierExpr{Name: "send_email", Position: Position{Line: 6, Column: 9}},
		Position: Position{Line: 6, Column: 1},
	}

	if stmt.Role != nil {
		t.Fatal("ApproveStatement.Role must be nil when not provided")
	}
}

func TestCallAndApproveCanAppearInWorkflowBody(t *testing.T) { // CLSF-51
	t.Parallel()

	body := []Statement{
		&ApproveStatement{
			Stage:    &IdentifierExpr{Name: "send_email", Position: Position{Line: 3, Column: 9}},
			Role:     &IdentifierExpr{Name: "manager", Position: Position{Line: 3, Column: 25}},
			Position: Position{Line: 3, Column: 1},
		},
		&CallStatement{
			Tool:     &IdentifierExpr{Name: "send_email", Position: Position{Line: 4, Column: 6}},
			Position: Position{Line: 4, Column: 1},
		},
	}

	program := &Program{
		Workflow: &WorkflowDecl{
			Name:     "approval_flow",
			Body:     body,
			Position: Position{Line: 1, Column: 1},
		},
	}

	stmts := ProgramStatements(program)
	if len(stmts) != 2 {
		t.Fatalf("len(stmts) = %d, want 2", len(stmts))
	}
	if _, ok := stmts[0].(*ApproveStatement); !ok {
		t.Fatal("stmts[0] must be *ApproveStatement")
	}
	if _, ok := stmts[1].(*CallStatement); !ok {
		t.Fatal("stmts[1] must be *CallStatement")
	}
}

func TestComparisonExprSupportsBridgeAlignedOperators(t *testing.T) {
	t.Parallel()

	expr := &ComparisonExpr{
		Left:     &IdentifierExpr{Name: "case.priority", Position: Position{Line: 3, Column: 4}},
		Operator: TokenIn,
		Right: &ArrayLiteralExpr{
			Elements: []Expression{
				&LiteralExpr{Value: "high", Position: Position{Line: 3, Column: 21}},
				&LiteralExpr{Value: "urgent", Position: Position{Line: 3, Column: 29}},
			},
			Position: Position{Line: 3, Column: 20},
		},
		Position: Position{Line: 3, Column: 1},
	}

	if expr.Operator != TokenIn {
		t.Fatalf("expr.Operator = %s, want %s", expr.Operator, TokenIn)
	}
	if expr.Pos().Line != 3 {
		t.Fatalf("expr.Pos() = %+v, want line 3", expr.Pos())
	}
}
