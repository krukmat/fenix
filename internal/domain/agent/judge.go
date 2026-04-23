package agent

import (
	"context"
	"strings"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

type WorkflowJudge struct{}

func NewJudge() *WorkflowJudge {
	return &WorkflowJudge{}
}

func (j *WorkflowJudge) Verify(ctx context.Context, workflow *workflowdomain.Workflow) (*JudgeResult, error) {
	_ = ctx

	syntaxResult := ValidateWorkflowDSLSyntax(workflow)
	judgeResult := NewJudgeResult(syntaxResult.Violations, syntaxResult.Warnings)
	appendMissingSpecSourceWarning(judgeResult, workflow)
	appendInitialSpecConsistencyFindings(judgeResult, workflow, syntaxResult.Program)
	appendProtocolJudgeFindings(judgeResult, syntaxResult.Program)
	return judgeResult, nil
}

func missingSpecSource(workflow *workflowdomain.Workflow) bool {
	return workflow.SpecSource == nil || strings.TrimSpace(*workflow.SpecSource) == ""
}

func appendMissingSpecSourceWarning(result *JudgeResult, workflow *workflowdomain.Workflow) {
	if workflow == nil || !missingSpecSource(workflow) {
		return
	}
	result.AddWarning(Warning{
		Code:        "missing_spec_source",
		Description: "spec_source is not provided; spec-to-dsl consistency checks are skipped",
		Location:    "workflow " + workflow.ID,
	})
}

func appendInitialSpecConsistencyFindings(result *JudgeResult, workflow *workflowdomain.Workflow, program *Program) {
	if result == nil || len(result.Violations) != 0 || workflow == nil || missingSpecSource(workflow) {
		return
	}
	if isCartaSource(*workflow.SpecSource) {
		appendCartaSpecConsistencyFindings(result, workflow, program)
		return
	}
	specSummary := ParsePartialSpec(*workflow.SpecSource)
	appendWarnings(result, specSummary.Warnings)
	violations, warnings := RunInitialSpecDSLChecks(specSummary, program)
	appendViolations(result, violations)
	appendWarnings(result, warnings)
}

func isCartaSource(source string) bool {
	trimmed := strings.TrimSpace(source)
	return trimmed == string(TokenCarta) || strings.HasPrefix(trimmed, string(TokenCarta)+" ")
}

func appendCartaSpecConsistencyFindings(result *JudgeResult, workflow *workflowdomain.Workflow, program *Program) {
	cartaSummary, err := ParseCarta(*workflow.SpecSource)
	if err != nil {
		appendViolations(result, []Violation{{
			Code:        "carta_parse_error",
			Type:        "carta_parse_error",
			Description: err.Error(),
			Location:    "spec_source",
		}})
		return
	}
	appendWarnings(result, cartaSummary.Warnings)
	violations, warnings := RunCartaSpecDSLChecks(cartaSummary, program, nil)
	appendViolations(result, violations)
	appendWarnings(result, warnings)
}

func appendProtocolJudgeFindings(result *JudgeResult, program *Program) {
	if result == nil || program == nil {
		return
	}
	violations, warnings := RunProtocolJudgeChecks(program)
	appendViolations(result, violations)
	appendWarnings(result, warnings)
}

func appendViolations(result *JudgeResult, violations []Violation) {
	for _, violation := range violations {
		result.AddViolation(violation)
	}
}

func appendWarnings(result *JudgeResult, warnings []Warning) {
	for _, warning := range warnings {
		result.AddWarning(warning)
	}
}
