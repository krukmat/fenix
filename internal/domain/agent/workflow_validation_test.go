package agent

import (
	"context"
	"testing"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

func TestValidateWorkflowForToolingReturnsJudgeConformanceAndGraph(t *testing.T) {
	t.Parallel()

	spec := `CARTA resolve_support_case
BUDGET
  daily_tokens: 50000
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  PERMIT update_case`

	result, err := ValidateWorkflowForTooling(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-tooling",
		DSLSource:  `WORKFLOW resolve_support_case` + "\n" + `ON case.created` + "\n" + `SET case.status = "resolved"`,
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("ValidateWorkflowForTooling() error = %v", err)
	}

	if result.WorkflowID != "wf-tooling" {
		t.Fatalf("WorkflowID = %q, want wf-tooling", result.WorkflowID)
	}
	if result.Judge == nil || !result.Judge.Passed {
		t.Fatalf("Judge = %#v, want passed judge", result.Judge)
	}
	if result.Conformance.Profile != ConformanceProfileSafe {
		t.Fatalf("Conformance.Profile = %q, want %q", result.Conformance.Profile, ConformanceProfileSafe)
	}
	if result.SemanticGraph == nil || result.SemanticGraph.WorkflowName != "resolve_support_case" {
		t.Fatalf("SemanticGraph = %#v, want resolve_support_case graph", result.SemanticGraph)
	}
	if result.SemanticGraph != result.Conformance.Graph {
		t.Fatal("SemanticGraph does not reference Conformance.Graph")
	}
}

func TestValidateWorkflowForToolingKeepsJudgeDiagnosticsSeparateFromConformance(t *testing.T) {
	t.Parallel()

	spec := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  PERMIT send_reply`

	result, err := ValidateWorkflowForTooling(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-tooling-judge-fail",
		DSLSource:  `WORKFLOW resolve_support_case` + "\n" + `ON case.created` + "\n" + `NOTIFY salesperson WITH "review"`,
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("ValidateWorkflowForTooling() error = %v", err)
	}

	if result.Judge == nil || result.Judge.Passed {
		t.Fatalf("Judge = %#v, want judge violation", result.Judge)
	}
	if !hasJudgeViolation(result.Judge, "tool_not_permitted") {
		t.Fatalf("Judge.Violations = %#v, want tool_not_permitted", result.Judge.Violations)
	}
	if result.Conformance.Profile != ConformanceProfileSafe {
		t.Fatalf("Conformance.Profile = %q, want %q", result.Conformance.Profile, ConformanceProfileSafe)
	}
}

func TestValidateWorkflowForToolingInvalidWorkflowReturnsInvalidConformance(t *testing.T) {
	t.Parallel()

	result, err := ValidateWorkflowForTooling(context.Background(), nil)
	if err != nil {
		t.Fatalf("ValidateWorkflowForTooling() error = %v", err)
	}

	if result.Judge == nil || result.Judge.Passed {
		t.Fatalf("Judge = %#v, want failed judge", result.Judge)
	}
	if result.Conformance.Profile != ConformanceProfileInvalid {
		t.Fatalf("Conformance.Profile = %q, want %q", result.Conformance.Profile, ConformanceProfileInvalid)
	}
	if result.SemanticGraph != nil {
		t.Fatalf("SemanticGraph = %#v, want nil graph for invalid source", result.SemanticGraph)
	}
}

func hasJudgeViolation(result *JudgeResult, code string) bool {
	if result == nil {
		return false
	}
	for _, violation := range result.Violations {
		if violation.Code == code {
			return true
		}
	}
	return false
}
