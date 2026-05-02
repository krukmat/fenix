package agent

import (
	"context"
	"strings"
	"testing"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

func TestBuildWorkflowInspectionSurfaceIncludesProjectionCoverageAndScenarioLinks(t *testing.T) {
	t.Parallel()

	spec := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  PERMIT update_case`

	workflow := &workflowdomain.Workflow{
		ID:         "wf-support-graph",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"triaged\"",
		SpecSource: &spec,
	}

	validation, err := ValidateWorkflowForTooling(context.Background(), workflow)
	if err != nil {
		t.Fatalf("ValidateWorkflowForTooling() error = %v", err)
	}

	surface := BuildWorkflowInspectionSurface(validation, workflow.DSLSource, []WorkflowScenarioCoverageRef{
		{ScenarioID: "sc-support-004", Title: "approval path"},
		{ScenarioID: "sc-workflow-001", Title: "activation blocked"},
	})

	if surface.WorkflowID != "wf-support-graph" {
		t.Fatalf("WorkflowID = %q, want wf-support-graph", surface.WorkflowID)
	}
	if surface.WorkflowName != "resolve_support_case" {
		t.Fatalf("WorkflowName = %q, want resolve_support_case", surface.WorkflowName)
	}
	if surface.Conformance.Profile != ConformanceProfileSafe {
		t.Fatalf("Conformance.Profile = %q, want %q", surface.Conformance.Profile, ConformanceProfileSafe)
	}
	if len(surface.VisualProjection.Nodes) == 0 {
		t.Fatal("expected visual projection nodes")
	}
	if len(surface.Adjacency) == 0 {
		t.Fatal("expected adjacency nodes")
	}
	if len(surface.Coverage) == 0 {
		t.Fatal("expected DSL coverage labels")
	}
	if len(surface.ScenarioCoverage) != 2 {
		t.Fatalf("ScenarioCoverage len = %d, want 2", len(surface.ScenarioCoverage))
	}
	if !strings.Contains(surface.Mermaid, "Conformance: safe") {
		t.Fatalf("Mermaid = %q, want conformance annotation", surface.Mermaid)
	}
	if !strings.Contains(surface.Mermaid, "resolve_support_case (workflow)") {
		t.Fatalf("Mermaid = %q, want workflow node label", surface.Mermaid)
	}
	if surface.Adjacency[0].ID == "" {
		t.Fatalf("Adjacency[0] = %#v, want stable node identity", surface.Adjacency[0])
	}
}

func TestWorkflowVisualProjectionToMermaidHandlesEmptyProjection(t *testing.T) {
	t.Parallel()

	got := (WorkflowVisualProjection{}).ToMermaid()
	if got != "flowchart LR\n" {
		t.Fatalf("ToMermaid() = %q, want empty flowchart", got)
	}
}
