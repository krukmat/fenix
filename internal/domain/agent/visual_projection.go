package agent

import "fmt"

const (
	workflowVisualColumnWidth = 260
	workflowVisualRowHeight   = 160
	workflowVisualColumns     = 5
)

type WorkflowVisualProjection struct {
	WorkflowName string               `json:"workflow_name,omitempty"`
	Nodes        []WorkflowVisualNode `json:"nodes"`
	Edges        []WorkflowVisualEdge `json:"edges"`
	Conformance  ConformanceResult    `json:"conformance"`
}

type WorkflowVisualNode struct {
	ID         SemanticNodeID         `json:"id"`
	Kind       SemanticNodeKind       `json:"kind"`
	Label      string                 `json:"label"`
	Color      string                 `json:"color"`
	Position   WorkflowVisualPosition `json:"position"`
	Source     SemanticSourceKind     `json:"source"`
	Effect     SemanticEffectKind     `json:"effect,omitempty"`
	Properties map[string]any         `json:"properties,omitempty"`
}

type WorkflowVisualPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type WorkflowVisualEdge struct {
	ID             string           `json:"id"`
	From           SemanticNodeID   `json:"from"`
	To             SemanticNodeID   `json:"to"`
	ConnectionType SemanticEdgeKind `json:"connection_type"`
	Properties     map[string]any   `json:"properties,omitempty"`
}

func ProjectWorkflowSemanticGraph(graph *WorkflowSemanticGraph, conformance ConformanceResult) WorkflowVisualProjection {
	projection := WorkflowVisualProjection{
		Nodes:       []WorkflowVisualNode{},
		Edges:       []WorkflowVisualEdge{},
		Conformance: conformance,
	}
	if graph == nil {
		return projection
	}

	projection.WorkflowName = graph.WorkflowName
	projection.Nodes = make([]WorkflowVisualNode, 0, len(graph.Nodes))
	for i, node := range graph.Nodes {
		projection.Nodes = append(projection.Nodes, projectWorkflowVisualNode(node, i))
	}
	projection.Edges = make([]WorkflowVisualEdge, 0, len(graph.Edges))
	for _, edge := range graph.Edges {
		projection.Edges = append(projection.Edges, projectWorkflowVisualEdge(edge))
	}
	return projection
}

func projectWorkflowVisualNode(node WorkflowSemanticNode, index int) WorkflowVisualNode {
	return WorkflowVisualNode{
		ID:         node.ID,
		Kind:       node.Kind,
		Label:      node.Label,
		Color:      visualNodeColor(node.Kind),
		Position:   visualNodePosition(index),
		Source:     node.Source,
		Effect:     node.Effect,
		Properties: node.Properties,
	}
}

func projectWorkflowVisualEdge(edge WorkflowSemanticEdge) WorkflowVisualEdge {
	return WorkflowVisualEdge{
		ID:             workflowVisualEdgeID(edge),
		From:           edge.From,
		To:             edge.To,
		ConnectionType: edge.Kind,
		Properties:     edge.Properties,
	}
}

func visualNodePosition(index int) WorkflowVisualPosition {
	if index < 0 {
		index = 0
	}
	return WorkflowVisualPosition{
		X: (index % workflowVisualColumns) * workflowVisualColumnWidth,
		Y: (index / workflowVisualColumns) * workflowVisualRowHeight,
	}
}

func visualNodeColor(kind SemanticNodeKind) string {
	color, ok := visualNodeColors[kind]
	if !ok {
		return "#64748b"
	}
	return color
}

var visualNodeColors = map[SemanticNodeKind]string{
	SemanticNodeWorkflow:  "#2563eb",
	SemanticNodeTrigger:   "#16a34a",
	SemanticNodeAction:    "#f59e0b",
	SemanticNodeDecision:  "#7c3aed",
	SemanticNodeGrounds:   "#0891b2",
	SemanticNodePermit:    "#dc2626",
	SemanticNodeDelegate:  "#db2777",
	SemanticNodeInvariant: "#475569",
	SemanticNodeBudget:    "#0d9488",
	SemanticNodeCall:      "#ea580c",
	SemanticNodeApprove:   "#9333ea",
}

func workflowVisualEdgeID(edge WorkflowSemanticEdge) string {
	return fmt.Sprintf("%s:%s:%s", edge.Kind, edge.From, edge.To)
}
