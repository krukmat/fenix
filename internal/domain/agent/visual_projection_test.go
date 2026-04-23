package agent

import "testing"

func TestProjectWorkflowSemanticGraphReturnsRenderableNodesEdgesAndConformance(t *testing.T) {
	t.Parallel()

	graph := NewWorkflowSemanticGraph("resolve_support_case")
	workflowID := SemanticNodeID("workflow:resolve_support_case")
	triggerID := SemanticNodeID("trigger:case.created")
	actionID := SemanticNodeID("action:set_status")
	graph.AddNode(WorkflowSemanticNode{
		ID:     workflowID,
		Kind:   SemanticNodeWorkflow,
		Label:  "resolve_support_case",
		Source: SemanticSourceDSL,
		Effect: SemanticEffectNone,
	})
	graph.AddNode(WorkflowSemanticNode{
		ID:     triggerID,
		Kind:   SemanticNodeTrigger,
		Label:  "case.created",
		Source: SemanticSourceDSL,
		Effect: SemanticEffectRead,
	})
	graph.AddNode(WorkflowSemanticNode{
		ID:     actionID,
		Kind:   SemanticNodeAction,
		Label:  "SET case.status",
		Source: SemanticSourceDSL,
		Effect: SemanticEffectWrite,
		Properties: map[string]any{
			"statement": "SET",
		},
	})
	graph.AddEdge(WorkflowSemanticEdge{From: workflowID, To: triggerID, Kind: SemanticEdgeContains})
	graph.AddEdge(WorkflowSemanticEdge{From: triggerID, To: actionID, Kind: SemanticEdgeNext})

	conformance := ConformanceResult{
		Profile: ConformanceProfileSafe,
		Details: []ConformanceDetail{{
			Code:     "missing_spec_source",
			Severity: ConformanceSeverityWarning,
			Message:  "missing spec_source is compatible but has no Carta graph nodes",
		}},
	}

	projection := ProjectWorkflowSemanticGraph(graph, conformance)

	if projection.WorkflowName != "resolve_support_case" {
		t.Fatalf("WorkflowName = %q, want resolve_support_case", projection.WorkflowName)
	}
	if projection.Conformance.Profile != ConformanceProfileSafe {
		t.Fatalf("Conformance.Profile = %q, want %q", projection.Conformance.Profile, ConformanceProfileSafe)
	}
	if len(projection.Conformance.Details) != 1 || projection.Conformance.Details[0].Code != "missing_spec_source" {
		t.Fatalf("Conformance.Details = %#v", projection.Conformance.Details)
	}
	if len(projection.Nodes) != 3 {
		t.Fatalf("len(Nodes) = %d, want 3", len(projection.Nodes))
	}
	if len(projection.Edges) != 2 {
		t.Fatalf("len(Edges) = %d, want 2", len(projection.Edges))
	}

	assertVisualNode(t, projection.Nodes[0], workflowID, SemanticNodeWorkflow, "resolve_support_case", "#2563eb", 0, 0)
	assertVisualNode(t, projection.Nodes[1], triggerID, SemanticNodeTrigger, "case.created", "#16a34a", 260, 0)
	assertVisualNode(t, projection.Nodes[2], actionID, SemanticNodeAction, "SET case.status", "#f59e0b", 520, 0)

	if projection.Edges[0].ID == "" {
		t.Fatal("first edge ID is empty")
	}
	if projection.Edges[0].From != workflowID || projection.Edges[0].To != triggerID {
		t.Fatalf("first edge endpoints = %q -> %q", projection.Edges[0].From, projection.Edges[0].To)
	}
	if projection.Edges[0].ConnectionType != SemanticEdgeContains {
		t.Fatalf("first edge ConnectionType = %q, want %q", projection.Edges[0].ConnectionType, SemanticEdgeContains)
	}
	if projection.Edges[1].ConnectionType != SemanticEdgeNext {
		t.Fatalf("second edge ConnectionType = %q, want %q", projection.Edges[1].ConnectionType, SemanticEdgeNext)
	}
}

func TestProjectWorkflowSemanticGraphMapsGovernanceKindsAndWrapsRows(t *testing.T) {
	t.Parallel()

	graph := NewWorkflowSemanticGraph("governed")
	kinds := []SemanticNodeKind{
		SemanticNodeWorkflow,
		SemanticNodeTrigger,
		SemanticNodeDecision,
		SemanticNodeGrounds,
		SemanticNodePermit,
		SemanticNodeDelegate,
		SemanticNodeInvariant,
		SemanticNodeBudget,
		SemanticNodeCall,
		SemanticNodeApprove,
		SemanticNodeKind("future"),
	}
	for _, kind := range kinds {
		graph.AddNode(WorkflowSemanticNode{
			ID:     SemanticNodeID(string(kind)),
			Kind:   kind,
			Label:  string(kind),
			Source: SemanticSourceDSL,
		})
	}

	projection := ProjectWorkflowSemanticGraph(graph, EvaluateGraphConformance(graph))

	wantColors := []string{
		"#2563eb",
		"#16a34a",
		"#7c3aed",
		"#0891b2",
		"#dc2626",
		"#db2777",
		"#475569",
		"#0d9488",
		"#ea580c",
		"#9333ea",
		"#64748b",
	}
	for i, wantColor := range wantColors {
		if projection.Nodes[i].Color != wantColor {
			t.Fatalf("node %d color = %q, want %q", i, projection.Nodes[i].Color, wantColor)
		}
	}
	if projection.Nodes[5].Position.X != 0 || projection.Nodes[5].Position.Y != 160 {
		t.Fatalf("wrapped node position = %#v, want {X:0 Y:160}", projection.Nodes[5].Position)
	}
	if projection.Conformance.Profile != ConformanceProfileExtended {
		t.Fatalf("Conformance.Profile = %q, want %q", projection.Conformance.Profile, ConformanceProfileExtended)
	}
}

func TestProjectWorkflowSemanticGraphHandlesNilGraph(t *testing.T) {
	t.Parallel()

	projection := ProjectWorkflowSemanticGraph(nil, ConformanceResult{Profile: ConformanceProfileInvalid})

	if projection.WorkflowName != "" {
		t.Fatalf("WorkflowName = %q, want empty", projection.WorkflowName)
	}
	if len(projection.Nodes) != 0 || len(projection.Edges) != 0 {
		t.Fatalf("projection = %#v, want empty nodes and edges", projection)
	}
	if projection.Conformance.Profile != ConformanceProfileInvalid {
		t.Fatalf("Conformance.Profile = %q, want invalid", projection.Conformance.Profile)
	}
}

func assertVisualNode(t *testing.T, node WorkflowVisualNode, id SemanticNodeID, kind SemanticNodeKind, label string, color string, x int, y int) {
	t.Helper()

	if node.ID != id {
		t.Fatalf("node.ID = %q, want %q", node.ID, id)
	}
	if node.Kind != kind {
		t.Fatalf("node.Kind = %q, want %q", node.Kind, kind)
	}
	if node.Label != label {
		t.Fatalf("node.Label = %q, want %q", node.Label, label)
	}
	if node.Color != color {
		t.Fatalf("node.Color = %q, want %q", node.Color, color)
	}
	if node.Position.X != x || node.Position.Y != y {
		t.Fatalf("node.Position = %#v, want X=%d Y=%d", node.Position, x, y)
	}
}
