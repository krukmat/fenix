package agent

import (
	"fmt"
	"strings"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

type WorkflowSyntaxValidationResult struct {
	Passed     bool
	Violations []Violation
	Warnings   []Warning
	Program    *Program
}

func ValidateWorkflowDSLSyntax(workflow *workflowdomain.Workflow) *WorkflowSyntaxValidationResult {
	if workflow == nil {
		return failedWorkflowSyntaxValidation(newSyntaxViolation(
			"dsl_syntax_error",
			"workflow is required",
			"",
			Position{},
		))
	}
	if strings.TrimSpace(workflow.DSLSource) == "" {
		return failedWorkflowSyntaxValidation(newSyntaxViolation(
			"dsl_syntax_error",
			"dsl_source is required",
			"workflow "+workflow.ID,
			Position{},
		))
	}

	program, err := ParseDSL(workflow.DSLSource)
	if err != nil {
		return failedWorkflowSyntaxValidation(mapSyntaxViolation(workflow.ID, err))
	}
	if err := ValidateDSLProgram(program); err != nil {
		return failedWorkflowSyntaxValidation(mapSyntaxViolation(workflow.ID, err))
	}

	return &WorkflowSyntaxValidationResult{
		Passed:  true,
		Program: program,
	}
}

func mapSyntaxViolation(workflowID string, err error) Violation {
	switch e := err.(type) {
	case SyntaxError:
		pos := e.Position()
		return newSyntaxViolation(
			"dsl_syntax_error",
			e.Message(),
			locationForSyntax(workflowID, pos),
			pos,
		)
	case *DSLValidationError:
		return newSyntaxViolation(
			"dsl_validation_error",
			e.Reason,
			locationForSyntax(workflowID, e.Position),
			e.Position,
		)
	default:
		return newSyntaxViolation(
			"dsl_syntax_error",
			err.Error(),
			"workflow "+workflowID,
			Position{},
		)
	}
}

func locationForSyntax(workflowID string, pos Position) string {
	if pos.Line <= 0 || pos.Column <= 0 {
		if strings.TrimSpace(workflowID) == "" {
			return "DSL"
		}
		return fmt.Sprintf("workflow %s DSL", workflowID)
	}
	return fmt.Sprintf("DSL line %d, column %d", pos.Line, pos.Column)
}

func newSyntaxViolation(kind, description, location string, pos Position) Violation {
	return normalizeViolation(Violation{
		Code:        kind,
		Type:        kind,
		Description: description,
		Location:    location,
		Line:        pos.Line,
		Column:      pos.Column,
	})
}

func failedWorkflowSyntaxValidation(violation Violation) *WorkflowSyntaxValidationResult {
	return &WorkflowSyntaxValidationResult{
		Passed:     false,
		Violations: []Violation{violation},
	}
}
