package agent

import (
	"fmt"
	"strings"
)

type DSLValidationError struct {
	Position Position
	Reason   string
}

func (e *DSLValidationError) Error() string {
	return fmt.Sprintf("dsl validation error at line %d, column %d: %s", e.Position.Line, e.Position.Column, e.Reason)
}

func ParseAndValidateDSL(source string) (*Program, error) {
	program, err := ParseDSL(source)
	if err != nil {
		return nil, err
	}
	if err := ValidateDSLProgram(program); err != nil {
		return nil, err
	}
	return program, nil
}

func ValidateDSLProgram(program *Program) error {
	if program == nil || program.Workflow == nil {
		return &DSLValidationError{Reason: "workflow declaration is required"}
	}
	workflow := program.Workflow
	if strings.TrimSpace(workflow.Name) == "" {
		return &DSLValidationError{Position: workflow.Position, Reason: "workflow name is required"}
	}
	if workflow.Trigger == nil || strings.TrimSpace(workflow.Trigger.Event) == "" {
		return &DSLValidationError{Position: workflow.Position, Reason: "ON trigger is required"}
	}
	if len(workflow.Body) == 0 {
		return &DSLValidationError{Position: workflow.Position, Reason: "workflow body must contain at least one statement"}
	}
	return validateStatementSlice(workflow.Body)
}

func validateStatementSlice(statements []Statement) error {
	for _, statement := range statements {
		if err := validateStatement(statement); err != nil {
			return err
		}
	}
	return nil
}

func validateStatement(statement Statement) error {
	switch stmt := statement.(type) {
	case *IfStatement:
		if stmt.Condition == nil {
			return validationError(stmt.Pos(), "IF requires condition")
		}
		if len(stmt.Body) == 0 {
			return validationError(stmt.Pos(), "IF block must contain at least one statement")
		}
		return validateStatementSlice(stmt.Body)
	case *SetStatement:
		if stmt.Target == nil || !strings.Contains(stmt.Target.Name, ".") {
			return validationError(stmt.Pos(), "SET target must be a dotted field reference")
		}
		if stmt.Value == nil {
			return validationError(stmt.Pos(), "SET requires value")
		}
		return nil
	case *NotifyStatement:
		if stmt.Target == nil || strings.TrimSpace(stmt.Target.Name) == "" {
			return validationError(stmt.Pos(), "NOTIFY target is required")
		}
		if stmt.Value == nil {
			return validationError(stmt.Pos(), "NOTIFY requires WITH payload")
		}
		return nil
	case *AgentStatement:
		if stmt.Name == nil || strings.TrimSpace(stmt.Name.Name) == "" {
			return validationError(stmt.Pos(), "AGENT name is required")
		}
		return nil
	default:
		return validationError(statement.Pos(), "statement is not allowed in DSL v0")
	}
}

func validationError(pos Position, reason string) error {
	return &DSLValidationError{
		Position: pos,
		Reason:   reason,
	}
}
