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
