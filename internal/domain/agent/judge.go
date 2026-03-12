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

	result := ValidateWorkflowDSLSyntax(workflow)
	judgeResult := NewJudgeResult(result.Violations, result.Warnings)
	if len(judgeResult.Violations) == 0 && workflow != nil && !missingSpecSource(workflow) {
		specSummary := ParsePartialSpec(*workflow.SpecSource)
		for _, warning := range specSummary.Warnings {
			judgeResult.AddWarning(warning)
		}
		violations, warnings := RunInitialSpecDSLChecks(specSummary, result.Program)
		for _, violation := range violations {
			judgeResult.AddViolation(violation)
		}
		for _, warning := range warnings {
			judgeResult.AddWarning(warning)
		}
	}
	if workflow != nil && missingSpecSource(workflow) {
		judgeResult.AddWarning(Warning{
			Code:        "missing_spec_source",
			Description: "spec_source is not provided; spec-to-dsl consistency checks are skipped",
			Location:    "workflow " + workflow.ID,
		})
	}
	return judgeResult, nil
}

func missingSpecSource(workflow *workflowdomain.Workflow) bool {
	return workflow.SpecSource == nil || strings.TrimSpace(*workflow.SpecSource) == ""
}
