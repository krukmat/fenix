package agent

import "testing"

func TestSemanticDiffWhitespaceOnlyIsLayoutOnly(t *testing.T) {
	t.Parallel()

	before := mustBuildSemanticGraph(t, `WORKFLOW resolve_support_case
ON case.created
SET case.status = "triaged"
NOTIFY salesperson WITH "review"
`)
	after := mustBuildSemanticGraph(t, `
WORKFLOW resolve_support_case
ON case.created

SET case.status = "triaged"

NOTIFY salesperson WITH "review"
`)

	diff := DiffWorkflowSemanticGraphs(before, after)
	if diff.HasSemanticChanges {
		t.Fatalf("HasSemanticChanges = true, changes = %#v %#v", diff.NodeChanges, diff.EdgeChanges)
	}
	if !diff.LayoutOnly {
		t.Fatal("LayoutOnly = false, want true")
	}
	if len(diff.NodeChanges) != 0 || len(diff.EdgeChanges) != 0 {
		t.Fatalf("changes = %#v %#v, want none", diff.NodeChanges, diff.EdgeChanges)
	}
}

func TestSemanticDiffWorkflowRenameIsSemantic(t *testing.T) {
	t.Parallel()

	before := mustBuildSemanticGraph(t, `WORKFLOW resolve_support_case
ON case.created
SET case.status = "triaged"
`)
	after := mustBuildSemanticGraph(t, `WORKFLOW resolve_enterprise_case
ON case.created
SET case.status = "triaged"
`)

	diff := DiffWorkflowSemanticGraphs(before, after)
	if !diff.HasSemanticChanges {
		t.Fatal("HasSemanticChanges = false, want true")
	}
	if diff.LayoutOnly {
		t.Fatal("LayoutOnly = true, want false")
	}
	if countSemanticNodeChanges(diff, SemanticDiffRemoved) == 0 {
		t.Fatalf("removed node changes = 0, changes = %#v", diff.NodeChanges)
	}
	if countSemanticNodeChanges(diff, SemanticDiffAdded) == 0 {
		t.Fatalf("added node changes = 0, changes = %#v", diff.NodeChanges)
	}
}

func TestSemanticDiffNodePropertyChangeIsModified(t *testing.T) {
	t.Parallel()

	before := mustBuildSemanticGraph(t, `WORKFLOW resolve_support_case
ON case.created
SET case.status = "triaged"
`)
	after := mustBuildSemanticGraph(t, `WORKFLOW resolve_support_case
ON case.created
SET case.status = "resolved"
`)

	diff := DiffWorkflowSemanticGraphs(before, after)
	if !diff.HasSemanticChanges {
		t.Fatal("HasSemanticChanges = false, want true")
	}
	if countSemanticNodeChanges(diff, SemanticDiffRemoved) == 0 || countSemanticNodeChanges(diff, SemanticDiffAdded) == 0 {
		t.Fatalf("node changes = %#v, want add/remove because stable IDs include semantic value", diff.NodeChanges)
	}
}

func mustBuildSemanticGraph(t *testing.T, source string) *WorkflowSemanticGraph {
	t.Helper()

	graph, err := BuildWorkflowSemanticGraphFromDSL(source)
	if err != nil {
		t.Fatalf("BuildWorkflowSemanticGraphFromDSL() error = %v", err)
	}
	return graph
}

func countSemanticNodeChanges(diff WorkflowSemanticDiff, kind SemanticDiffChangeKind) int {
	count := 0
	for _, change := range diff.NodeChanges {
		if change.Kind == kind {
			count++
		}
	}
	return count
}
