package agent

import (
	"errors"
	"fmt"
	"strings"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

const (
	judgeViolationDSLSyntax   = "dsl_syntax_error"
	judgeViolationDSLValidate = "dsl_validation_error"
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
			judgeViolationDSLSyntax,
			"workflow is required",
			"",
			Position{},
		))
	}
	if strings.TrimSpace(workflow.DSLSource) == "" {
		return failedWorkflowSyntaxValidation(newSyntaxViolation(
			judgeViolationDSLSyntax,
			"dsl_source is required",
			"workflow "+workflow.ID,
			Position{},
		))
	}

	program, err := ParseDSL(workflow.DSLSource)
	if err != nil {
		return failedWorkflowSyntaxValidation(mapSyntaxViolation(workflow.ID, err))
	}
	validateErr := ValidateDSLProgram(program)
	if validateErr != nil {
		return failedWorkflowSyntaxValidation(mapSyntaxViolation(workflow.ID, validateErr))
	}

	return &WorkflowSyntaxValidationResult{
		Passed:  true,
		Program: program,
	}
}

func mapSyntaxViolation(workflowID string, err error) Violation {
	var syntaxErr SyntaxError
	if errors.As(err, &syntaxErr) {
		pos := syntaxErr.Position()
		return newSyntaxViolation(
			judgeViolationDSLSyntax,
			syntaxErr.Message(),
			locationForSyntax(workflowID, pos),
			pos,
		)
	}

	var validationErr *DSLValidationError
	if errors.As(err, &validationErr) {
		return newSyntaxViolation(
			judgeViolationDSLValidate,
			validationErr.Reason,
			locationForSyntax(workflowID, validationErr.Position),
			validationErr.Position,
		)
	}

	return newSyntaxViolation(
		judgeViolationDSLSyntax,
		err.Error(),
		"workflow "+workflowID,
		Position{},
	)
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
