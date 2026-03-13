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
	if validateErr := ValidateDSLProgram(program); validateErr != nil {
		return nil, validateErr
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
	if handled, err := validateLeafStatement(statement); handled {
		return err
	}
	return validateStructuredStatement(statement)
}

func validateLeafStatement(statement Statement) (bool, error) {
	switch stmt := statement.(type) {
	case *SetStatement:
		return true, validateSetStatement(stmt)
	case *NotifyStatement:
		return true, validateNotifyStatement(stmt)
	case *AgentStatement:
		return true, validateAgentStatement(stmt)
	case *DispatchStatement:
		return true, validateDispatchStatement(stmt)
	case *SurfaceStatement:
		return true, validateSurfaceStatement(stmt)
	case *WaitStatement:
		return true, validateWaitStatement(stmt)
	default:
		return false, nil
	}
}

func validateStructuredStatement(statement Statement) error {
	if stmt, ok := statement.(*IfStatement); ok {
		return validateIfStatement(stmt)
	}
	return validationError(statement.Pos(), "statement is not allowed in DSL v0")
}

func validateIfStatement(stmt *IfStatement) error {
	if stmt.Condition == nil {
		return validationError(stmt.Pos(), "IF requires condition")
	}
	if len(stmt.Body) == 0 {
		return validationError(stmt.Pos(), "IF block must contain at least one statement")
	}
	return validateStatementSlice(stmt.Body)
}

func validateSetStatement(stmt *SetStatement) error {
	if stmt.Target == nil || !strings.Contains(stmt.Target.Name, ".") {
		return validationError(stmt.Pos(), "SET target must be a dotted field reference")
	}
	if stmt.Value == nil {
		return validationError(stmt.Pos(), "SET requires value")
	}
	return nil
}

func validateNotifyStatement(stmt *NotifyStatement) error {
	if stmt.Target == nil || strings.TrimSpace(stmt.Target.Name) == "" {
		return validationError(stmt.Pos(), "NOTIFY target is required")
	}
	if stmt.Value == nil {
		return validationError(stmt.Pos(), "NOTIFY requires WITH payload")
	}
	return nil
}

func validateAgentStatement(stmt *AgentStatement) error {
	if stmt.Name == nil || strings.TrimSpace(stmt.Name.Name) == "" {
		return validationError(stmt.Pos(), "AGENT name is required")
	}
	return nil
}

func validateDispatchStatement(stmt *DispatchStatement) error {
	if stmt.Target == nil || strings.TrimSpace(stmt.Target.Name) == "" {
		return validationError(stmt.Pos(), "DISPATCH target is required")
	}
	if stmt.Payload == nil {
		return validationError(stmt.Pos(), "DISPATCH requires WITH payload")
	}
	return nil
}

func validateSurfaceStatement(stmt *SurfaceStatement) error {
	if stmt.Entity == nil || strings.TrimSpace(stmt.Entity.Name) == "" {
		return validationError(stmt.Pos(), "SURFACE entity is required")
	}
	if !isSupportedSurfaceEntity(stmt.Entity.Name) {
		return validationError(stmt.Pos(), "SURFACE entity is not supported")
	}
	if stmt.View == nil || strings.TrimSpace(stmt.View.Name) == "" {
		return validationError(stmt.Pos(), "SURFACE target view is required")
	}
	if stmt.Payload == nil {
		return validationError(stmt.Pos(), "SURFACE requires WITH payload")
	}
	return nil
}

func isSupportedSurfaceEntity(name string) bool {
	switch strings.TrimSpace(name) {
	case "contact", "lead", "deal", bridgeEntityCase:
		return true
	default:
		return false
	}
}

func validateWaitStatement(stmt *WaitStatement) error {
	if stmt.Amount < 0 {
		return validationError(stmt.Pos(), "WAIT duration must be >= 0")
	}
	if stmt.Amount > 0 && strings.TrimSpace(stmt.Unit) == "" {
		return validationError(stmt.Pos(), "WAIT duration unit is required for non-zero durations")
	}
	if strings.TrimSpace(stmt.Unit) == "" {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(stmt.Unit)) {
	case "s", "sec", "secs", "second", "seconds",
		"m", "min", "mins", "minute", "minutes",
		"h", "hr", "hrs", "hour", "hours",
		"d", "day", "days":
		return nil
	default:
		return validationError(stmt.Pos(), "WAIT duration unit is not supported")
	}
}

func validationError(pos Position, reason string) error {
	return &DSLValidationError{
		Position: pos,
		Reason:   reason,
	}
}
