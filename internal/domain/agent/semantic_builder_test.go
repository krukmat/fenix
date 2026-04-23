package agent

import "testing"

func TestSemanticBuilderProjectsDSLCoreNodes(t *testing.T) {
	t.Parallel()

	graph, err := BuildWorkflowSemanticGraphFromDSL(`WORKFLOW resolve_support_case
ON case.created
IF case.priority == "high":
  SET case.status = "triaged"
  NOTIFY salesperson WITH "review"
AGENT support_bot WITH case.id
`)
	if err != nil {
		t.Fatalf("BuildWorkflowSemanticGraphFromDSL() error = %v", err)
	}

	if graph.WorkflowName != "resolve_support_case" {
		t.Fatalf("WorkflowName = %q, want resolve_support_case", graph.WorkflowName)
	}

	assertSemanticNode(t, graph, SemanticNodeWorkflow, "resolve_support_case", SemanticEffectNone, "WORKFLOW")
	assertSemanticNode(t, graph, SemanticNodeTrigger, "case.created", SemanticEffectRead, "ON")
	assertSemanticNode(t, graph, SemanticNodeDecision, `IF case.priority == "high"`, SemanticEffectRead, "IF")
	assertSemanticNode(t, graph, SemanticNodeAction, "SET case.status", SemanticEffectWrite, "SET")
	assertSemanticNode(t, graph, SemanticNodeAction, "NOTIFY salesperson", SemanticEffectNotify, "NOTIFY")
	assertSemanticNode(t, graph, SemanticNodeDelegate, "AGENT support_bot", SemanticEffectDelegate, "AGENT")

	if len(graph.Nodes) != 6 {
		t.Fatalf("len(Nodes) = %d, want 6", len(graph.Nodes))
	}
	if countSemanticEdges(graph, SemanticEdgeContains) != 5 {
		t.Fatalf("contains edges = %d, want 5", countSemanticEdges(graph, SemanticEdgeContains))
	}
	if countSemanticEdges(graph, SemanticEdgeNext) != 3 {
		t.Fatalf("next edges = %d, want 3", countSemanticEdges(graph, SemanticEdgeNext))
	}
}

func TestSemanticBuilderStableAcrossWhitespaceOnlyChanges(t *testing.T) {
	t.Parallel()

	sourceA := `WORKFLOW resolve_support_case
ON case.created
SET case.status = "triaged"
NOTIFY salesperson WITH "review"
`
	sourceB := `
WORKFLOW resolve_support_case
ON case.created

SET case.status = "triaged"

NOTIFY salesperson WITH "review"
`

	graphA, err := BuildWorkflowSemanticGraphFromDSL(sourceA)
	if err != nil {
		t.Fatalf("BuildWorkflowSemanticGraphFromDSL(sourceA) error = %v", err)
	}
	graphB, err := BuildWorkflowSemanticGraphFromDSL(sourceB)
	if err != nil {
		t.Fatalf("BuildWorkflowSemanticGraphFromDSL(sourceB) error = %v", err)
	}

	idsA := semanticNodeIDs(graphA)
	idsB := semanticNodeIDs(graphB)
	if len(idsA) != len(idsB) {
		t.Fatalf("len(idsA) = %d, len(idsB) = %d", len(idsA), len(idsB))
	}
	for i := range idsA {
		if idsA[i] != idsB[i] {
			t.Fatalf("node id %d changed across whitespace-only source: %q != %q", i, idsA[i], idsB[i])
		}
	}
}

func TestSemanticBuilderProjectsCartaNodes(t *testing.T) {
	t.Parallel()

	graph, err := BuildWorkflowSemanticGraphFromSources(`WORKFLOW resolve_support_case
ON case.created
SET case.status = "triaged"
`, `CARTA resolve_support_case
BUDGET
  daily_tokens: 50000
  daily_cost_usd: 5.00
  executions_per_day: 100
  on_exceed: pause
AGENT search_knowledge
  GROUNDS
    min_sources: 2
    min_confidence: medium
    max_staleness: 30 days
    types: ["case", "kb_article"]
  PERMIT send_reply
    rate: 10 / hour
    approval: none
  DELEGATE TO HUMAN
    when: case.tier == "enterprise"
    reason: "Enterprise review"
    package: [evidence_ids, case_summary]
  INVARIANT
    never: "send_pii"
`)
	if err != nil {
		t.Fatalf("BuildWorkflowSemanticGraphFromSources() error = %v", err)
	}

	assertCartaSemanticNode(t, graph, SemanticNodeGrounds, "GROUNDS", SemanticEffectGovernance, "GROUNDS")
	assertCartaSemanticNode(t, graph, SemanticNodePermit, "PERMIT send_reply", SemanticEffectGovernance, "PERMIT")
	assertCartaSemanticNode(t, graph, SemanticNodeDelegate, "DELEGATE TO HUMAN", SemanticEffectDelegate, "DELEGATE")
	assertCartaSemanticNode(t, graph, SemanticNodeInvariant, "INVARIANT never", SemanticEffectGovernance, "INVARIANT")
	assertCartaSemanticNode(t, graph, SemanticNodeBudget, "BUDGET", SemanticEffectGovernance, "BUDGET")

	permit := findSemanticNode(t, graph, SemanticNodePermit, "PERMIT send_reply")
	if permit.Properties["tool"] != "send_reply" || permit.Properties["approval"] != "none" {
		t.Fatalf("permit properties = %#v", permit.Properties)
	}
	grounds := findSemanticNode(t, graph, SemanticNodeGrounds, "GROUNDS")
	if grounds.Properties["min_sources"] != 2 || grounds.Properties["min_confidence"] != "medium" {
		t.Fatalf("grounds properties = %#v", grounds.Properties)
	}

	if countSemanticEdges(graph, SemanticEdgeRequires) != 1 {
		t.Fatalf("requires edges = %d, want 1", countSemanticEdges(graph, SemanticEdgeRequires))
	}
	if countSemanticEdges(graph, SemanticEdgeGoverns) != 5 {
		t.Fatalf("governs edges = %d, want 5", countSemanticEdges(graph, SemanticEdgeGoverns))
	}
}

func assertSemanticNode(t *testing.T, graph *WorkflowSemanticGraph, kind SemanticNodeKind, label string, effect SemanticEffectKind, statement string) {
	t.Helper()

	for _, node := range graph.Nodes {
		if node.Kind != kind || node.Label != label {
			continue
		}
		if node.Source != SemanticSourceDSL {
			t.Fatalf("%s source = %q, want %q", label, node.Source, SemanticSourceDSL)
		}
		if node.Effect != effect {
			t.Fatalf("%s effect = %q, want %q", label, node.Effect, effect)
		}
		if node.Properties["statement"] != statement {
			t.Fatalf("%s statement = %#v, want %q", label, node.Properties["statement"], statement)
		}
		if node.Position.Line == 0 || node.Position.Column == 0 {
			t.Fatalf("%s position = %#v, want non-zero position", label, node.Position)
		}
		return
	}

	t.Fatalf("missing semantic node kind=%q label=%q", kind, label)
}

func assertCartaSemanticNode(t *testing.T, graph *WorkflowSemanticGraph, kind SemanticNodeKind, label string, effect SemanticEffectKind, statement string) {
	t.Helper()

	node := findSemanticNode(t, graph, kind, label)
	if node.Source != SemanticSourceCarta {
		t.Fatalf("%s source = %q, want %q", label, node.Source, SemanticSourceCarta)
	}
	if node.Effect != effect {
		t.Fatalf("%s effect = %q, want %q", label, node.Effect, effect)
	}
	if node.Properties["statement"] != statement {
		t.Fatalf("%s statement = %#v, want %q", label, node.Properties["statement"], statement)
	}
}

func findSemanticNode(t *testing.T, graph *WorkflowSemanticGraph, kind SemanticNodeKind, label string) WorkflowSemanticNode {
	t.Helper()

	for _, node := range graph.Nodes {
		if node.Kind == kind && node.Label == label {
			return node
		}
	}
	t.Fatalf("missing semantic node kind=%q label=%q", kind, label)
	return WorkflowSemanticNode{}
}

func countSemanticEdges(graph *WorkflowSemanticGraph, kind SemanticEdgeKind) int {
	count := 0
	for _, edge := range graph.Edges {
		if edge.Kind == kind {
			count++
		}
	}
	return count
}

func semanticNodeIDs(graph *WorkflowSemanticGraph) []SemanticNodeID {
	ids := make([]SemanticNodeID, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		ids = append(ids, node.ID)
	}
	return ids
}
