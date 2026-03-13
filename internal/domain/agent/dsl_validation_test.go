package agent

import (
	"errors"
	"testing"
)

func TestValidateDSLProgramAcceptsValidV0Program(t *testing.T) {
	t.Parallel()

	program, err := ParseDSL(`WORKFLOW resolve_support_case
ON case.created
IF case.priority IN ["high", "urgent"]:
  NOTIFY salesperson WITH "review this case"
SET case.status = "resolved"`)
	if err != nil {
		t.Fatalf("ParseDSL() error = %v", err)
	}

	if err := ValidateDSLProgram(program); err != nil {
		t.Fatalf("ValidateDSLProgram() error = %v", err)
	}
}

func TestValidateDSLProgramRejectsSetWithoutDottedTarget(t *testing.T) {
	t.Parallel()

	program := &Program{
		Workflow: &WorkflowDecl{
			Name:     "x",
			Trigger:  &OnDecl{Event: "case.created", Position: Position{Line: 2, Column: 1}},
			Position: Position{Line: 1, Column: 1},
			Body: []Statement{
				&SetStatement{
					Target:   &IdentifierExpr{Name: "status", Position: Position{Line: 3, Column: 5}},
					Value:    &LiteralExpr{Value: "resolved", Position: Position{Line: 3, Column: 14}},
					Position: Position{Line: 3, Column: 1},
				},
			},
		},
	}

	err := ValidateDSLProgram(program)
	if err == nil {
		t.Fatal("expected validation error")
	}
	var validationErr *DSLValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected DSLValidationError, got %T", err)
	}
	if validationErr.Position.Line != 3 {
		t.Fatalf("validationErr.Position = %+v, want line 3", validationErr.Position)
	}
}

func TestValidateDSLProgramRejectsEmptyWorkflowBody(t *testing.T) {
	t.Parallel()

	program := &Program{
		Workflow: &WorkflowDecl{
			Name:     "x",
			Trigger:  &OnDecl{Event: "case.created", Position: Position{Line: 2, Column: 1}},
			Position: Position{Line: 1, Column: 1},
		},
	}

	err := ValidateDSLProgram(program)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestParseAndValidateDSLAcceptsWaitStatement(t *testing.T) {
	t.Parallel()

	_, err := ParseAndValidateDSL(`WORKFLOW follow_up_case
ON case.created
WAIT 48 hours`)
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}
}

func TestParseAndValidateDSLAcceptsDispatchStatement(t *testing.T) {
	t.Parallel()

	_, err := ParseAndValidateDSL("WORKFLOW delegate_case\nON case.created\nDISPATCH TO support_agent WITH {\"case_id\":\"case-1\"}")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}
}

func TestParseAndValidateDSLAcceptsSurfaceStatement(t *testing.T) {
	t.Parallel()

	_, err := ParseAndValidateDSL("WORKFLOW surface_case\nON case.created\nSURFACE case TO salesperson.view WITH {\"value\":\"review\"}")
	if err != nil {
		t.Fatalf("ParseAndValidateDSL() error = %v", err)
	}
}

func TestParseAndValidateDSLRejectsUnsupportedSurfaceEntity(t *testing.T) {
	t.Parallel()

	_, err := ParseAndValidateDSL("WORKFLOW surface_case\nON case.created\nSURFACE invoice TO salesperson.view WITH {\"value\":\"review\"}")
	if err == nil {
		t.Fatal("expected error")
	}
	var validationErr *DSLValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected DSLValidationError, got %T", err)
	}
}

func TestParseAndValidateDSLRejectsNegativeWaitDuration(t *testing.T) {
	t.Parallel()

	_, err := ParseAndValidateDSL(`WORKFLOW follow_up_case
ON case.created
WAIT -1 hours`)
	if err == nil {
		t.Fatal("expected error")
	}
	var validationErr *DSLValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected DSLValidationError, got %T", err)
	}
}

func TestDSLValidationErrorMessage(t *testing.T) {
	t.Parallel()

	e := &DSLValidationError{Position: Position{Line: 2, Column: 5}, Reason: "SET target must use dotted path"}
	want := "dsl validation error at line 2, column 5: SET target must use dotted path"
	if e.Error() != want {
		t.Fatalf("Error() = %q, want %q", e.Error(), want)
	}
}
