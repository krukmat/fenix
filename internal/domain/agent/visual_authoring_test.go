package agent

import (
	"encoding/json"
	"testing"
)

func TestNewVisualAuthoringGraphInitializesEmptySlices(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("sales_followup")

	if graph.WorkflowName != "sales_followup" {
		t.Fatalf("WorkflowName = %q, want sales_followup", graph.WorkflowName)
	}
	if graph.Nodes == nil || len(graph.Nodes) != 0 {
		t.Fatalf("Nodes = %#v, want empty initialized slice", graph.Nodes)
	}
	if graph.Edges == nil || len(graph.Edges) != 0 {
		t.Fatalf("Edges = %#v, want empty initialized slice", graph.Edges)
	}
}

func TestVisualAuthoringGraphAddsSupportedNodesAndEdges(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("sales_followup")
	workflow := NewVisualAuthoringNode("node-workflow", SemanticNodeWorkflow, "sales_followup", WorkflowVisualPosition{X: 0, Y: 0})
	trigger := NewVisualAuthoringNode("node-trigger", SemanticNodeTrigger, "deal.updated", WorkflowVisualPosition{X: 260, Y: 0})
	trigger.Data.Event = "deal.updated"
	graph.AddNode(workflow)
	graph.AddNode(trigger)
	graph.AddEdge(NewVisualAuthoringEdge("edge-workflow-trigger", workflow.ID, trigger.ID, SemanticEdgeContains))

	if len(graph.Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2", len(graph.Nodes))
	}
	if graph.Nodes[1].Data.Event != "deal.updated" {
		t.Fatalf("trigger Event = %q, want deal.updated", graph.Nodes[1].Data.Event)
	}
	if len(graph.Edges) != 1 || graph.Edges[0].ConnectionType != SemanticEdgeContains {
		t.Fatalf("Edges = %#v, want contains edge", graph.Edges)
	}
}

func TestSupportedVisualAuthoringNodeKindsIncludesWave7ScopeOnly(t *testing.T) {
	t.Parallel()

	for _, kind := range []SemanticNodeKind{
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
		if !IsSupportedVisualAuthoringNodeKind(kind) {
			t.Fatalf("kind %q should be supported", kind)
		}
	}

	if IsSupportedVisualAuthoringNodeKind(SemanticNodeCall) {
		t.Fatal("call should not be supported by initial visual authoring schema")
	}
	if IsSupportedVisualAuthoringNodeKind(SemanticNodeApprove) {
		t.Fatal("approve should not be supported by initial visual authoring schema")
	}
}

func TestVisualAuthoringGraphJSONShape(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("sales_followup")
	node := NewVisualAuthoringNode("node-permit", SemanticNodePermit, "PERMIT send_reply", WorkflowVisualPosition{X: 0, Y: 160})
	node.Data.Permit = "send_reply"
	graph.AddNode(node)

	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["workflow_name"] != "sales_followup" {
		t.Fatalf("workflow_name = %#v, want sales_followup", decoded["workflow_name"])
	}
	nodes, ok := decoded["nodes"].([]any)
	if !ok || len(nodes) != 1 {
		t.Fatalf("nodes = %#v, want one JSON node", decoded["nodes"])
	}
	first, ok := nodes[0].(map[string]any)
	if !ok || first["kind"] != string(SemanticNodePermit) {
		t.Fatalf("first node = %#v, want permit kind", nodes[0])
	}
}

func TestValidateVisualAuthoringGraphPassesForSupportedShape(t *testing.T) {
	t.Parallel()

	graph := validVisualAuthoringGraph()

	result := ValidateVisualAuthoringGraph(graph)

	if !result.Passed {
		t.Fatalf("Passed = false, violations = %#v", result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("Violations = %#v, want none", result.Violations)
	}
}

func TestValidateVisualAuthoringGraphRejectsMissingTrigger(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("sales_followup")
	graph.AddNode(NewVisualAuthoringNode("workflow", SemanticNodeWorkflow, "sales_followup", WorkflowVisualPosition{}))

	result := ValidateVisualAuthoringGraph(graph)

	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if !hasVisualAuthoringViolation(result, "visual_trigger_missing") {
		t.Fatalf("Violations = %#v, want visual_trigger_missing", result.Violations)
	}
}

func TestValidateVisualAuthoringGraphRejectsUnsupportedNodeKind(t *testing.T) {
	t.Parallel()

	graph := validVisualAuthoringGraph()
	graph.AddNode(NewVisualAuthoringNode("call", SemanticNodeCall, "CALL tool", WorkflowVisualPosition{}))

	result := ValidateVisualAuthoringGraph(graph)

	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if !hasVisualAuthoringViolation(result, "visual_node_unsupported") {
		t.Fatalf("Violations = %#v, want visual_node_unsupported", result.Violations)
	}
}

func TestValidateVisualAuthoringGraphRejectsInvalidEdgeEndpoints(t *testing.T) {
	t.Parallel()

	graph := validVisualAuthoringGraph()
	graph.AddEdge(NewVisualAuthoringEdge("bad-edge", "missing-from", "missing-to", SemanticEdgeNext))

	result := ValidateVisualAuthoringGraph(graph)

	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if !hasVisualAuthoringViolation(result, "visual_edge_from_missing") {
		t.Fatalf("Violations = %#v, want visual_edge_from_missing", result.Violations)
	}
	if !hasVisualAuthoringViolation(result, "visual_edge_to_missing") {
		t.Fatalf("Violations = %#v, want visual_edge_to_missing", result.Violations)
	}
}

func TestValidateVisualAuthoringGraphRejectsNilGraph(t *testing.T) {
	t.Parallel()

	result := ValidateVisualAuthoringGraph(nil)

	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if !hasVisualAuthoringViolation(result, "visual_graph_missing") {
		t.Fatalf("Violations = %#v, want visual_graph_missing", result.Violations)
	}
}

func validVisualAuthoringGraph() *VisualAuthoringGraph {
	graph := NewVisualAuthoringGraph("sales_followup")
	graph.AddNode(NewVisualAuthoringNode("workflow", SemanticNodeWorkflow, "sales_followup", WorkflowVisualPosition{}))
	graph.AddNode(NewVisualAuthoringNode("trigger", SemanticNodeTrigger, "deal.updated", WorkflowVisualPosition{}))
	graph.AddNode(NewVisualAuthoringNode("action", SemanticNodeAction, "notify owner", WorkflowVisualPosition{}))
	graph.AddNode(NewVisualAuthoringNode("permit", SemanticNodePermit, "PERMIT send_reply", WorkflowVisualPosition{}))
	graph.AddEdge(NewVisualAuthoringEdge("workflow-trigger", "workflow", "trigger", SemanticEdgeContains))
	graph.AddEdge(NewVisualAuthoringEdge("trigger-action", "trigger", "action", SemanticEdgeNext))
	graph.AddEdge(NewVisualAuthoringEdge("permit-action", "permit", "action", SemanticEdgeGoverns))
	return graph
}

func hasVisualAuthoringViolation(result VisualAuthoringValidationResult, code string) bool {
	for _, violation := range result.Violations {
		if violation.Code == code {
			return true
		}
	}
	return false
}
