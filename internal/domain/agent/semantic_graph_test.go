package agent

import "testing"

func TestSemanticSupportedNodeKindsIncludesWave2MVPKinds(t *testing.T) {
	t.Parallel()

	got := map[SemanticNodeKind]bool{}
	for _, kind := range SupportedSemanticNodeKinds() {
		got[kind] = true
	}

	for _, want := range []SemanticNodeKind{
		SemanticNodeWorkflow,
		SemanticNodeTrigger,
		SemanticNodeAction,
		SemanticNodeDecision,
		SemanticNodeGrounds,
		SemanticNodePermit,
		SemanticNodeDelegate,
		SemanticNodeInvariant,
		SemanticNodeBudget,
	} {
		if !got[want] {
			t.Fatalf("SupportedSemanticNodeKinds() missing %q", want)
		}
	}
}

func TestSemanticGraphStoresNodesAndEdges(t *testing.T) {
	t.Parallel()

	graph := NewWorkflowSemanticGraph("resolve_support_case")
	graph.AddNode(WorkflowSemanticNode{
		ID:     "workflow:resolve_support_case",
		Kind:   SemanticNodeWorkflow,
		Label:  "resolve_support_case",
		Source: SemanticSourceDSL,
	})
	graph.AddNode(WorkflowSemanticNode{
		ID:     "trigger:case.created",
		Kind:   SemanticNodeTrigger,
		Label:  "case.created",
		Source: SemanticSourceDSL,
	})
	graph.AddEdge(WorkflowSemanticEdge{
		From: "workflow:resolve_support_case",
		To:   "trigger:case.created",
		Kind: SemanticEdgeContains,
	})

	if graph.WorkflowName != "resolve_support_case" {
		t.Fatalf("WorkflowName = %q, want resolve_support_case", graph.WorkflowName)
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("len(Edges) = %d, want 1", len(graph.Edges))
	}

	node, ok := graph.FindNode("trigger:case.created")
	if !ok {
		t.Fatal("FindNode(trigger) returned false")
	}
	if node.Kind != SemanticNodeTrigger {
		t.Fatalf("node.Kind = %q, want %q", node.Kind, SemanticNodeTrigger)
	}
}

func TestSemanticNodeDefaultsEffect(t *testing.T) {
	t.Parallel()

	node := NewWorkflowSemanticNode("permit:send_reply", SemanticNodePermit, SemanticSourceCarta)
	if node.Effect != SemanticEffectNone {
		t.Fatalf("Effect = %q, want %q", node.Effect, SemanticEffectNone)
	}
}
