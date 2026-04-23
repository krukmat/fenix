package agent

import (
	"context"
	"strings"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

type WorkflowValidationResult struct {
	WorkflowID    string                 `json:"workflow_id,omitempty"`
	Judge         *JudgeResult           `json:"judge"`
	Conformance   ConformanceResult      `json:"conformance"`
	SemanticGraph *WorkflowSemanticGraph `json:"semantic_graph,omitempty"`
}

func ValidateWorkflowForTooling(ctx context.Context, workflow *workflowdomain.Workflow) (*WorkflowValidationResult, error) {
	judgeResult, err := NewJudge().Verify(ctx, workflow)
	if err != nil {
		return nil, err
	}

	conformance := EvaluateWorkflowConformance(workflowDSLSource(workflow), workflowSpecSource(workflow))
	return &WorkflowValidationResult{
		WorkflowID:    workflowID(workflow),
		Judge:         judgeResult,
		Conformance:   conformance,
		SemanticGraph: conformance.Graph,
	}, nil
}

func workflowID(workflow *workflowdomain.Workflow) string {
	if workflow == nil {
		return ""
	}
	return workflow.ID
}

func workflowDSLSource(workflow *workflowdomain.Workflow) string {
	if workflow == nil {
		return ""
	}
	return workflow.DSLSource
}

func workflowSpecSource(workflow *workflowdomain.Workflow) string {
	if workflow == nil || workflow.SpecSource == nil {
		return ""
	}
	return strings.TrimSpace(*workflow.SpecSource)
}
