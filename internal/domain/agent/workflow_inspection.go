package agent

import (
	"fmt"
	"sort"
	"strings"
)

type WorkflowScenarioCoverageRef struct {
	ScenarioID string `json:"scenario_id"`
	Title      string `json:"title,omitempty"`
}

type WorkflowAdjacencyNode struct {
	ID       SemanticNodeID     `json:"id"`
	Label    string             `json:"label"`
	Kind     SemanticNodeKind   `json:"kind"`
	Outgoing []SemanticNodeID   `json:"outgoing,omitempty"`
	Incoming []SemanticNodeID   `json:"incoming,omitempty"`
	Source   SemanticSourceKind `json:"source"`
}

type WorkflowInspectionSurface struct {
	WorkflowID       string                        `json:"workflow_id,omitempty"`
	WorkflowName     string                        `json:"workflow_name,omitempty"`
	Conformance      ConformanceResult             `json:"conformance"`
	VisualProjection WorkflowVisualProjection      `json:"visual_projection"`
	Adjacency        []WorkflowAdjacencyNode       `json:"adjacency"`
	Mermaid          string                        `json:"mermaid"`
	Coverage         []DSLCoverageLabel            `json:"coverage,omitempty"`
	ScenarioCoverage []WorkflowScenarioCoverageRef `json:"scenario_coverage,omitempty"`
}

func BuildWorkflowInspectionSurface(
	validation *WorkflowValidationResult,
	dslSource string,
	scenarioCoverage []WorkflowScenarioCoverageRef,
) WorkflowInspectionSurface {
	surface := WorkflowInspectionSurface{
		Adjacency:        []WorkflowAdjacencyNode{},
		ScenarioCoverage: cloneWorkflowScenarioCoverageRefs(scenarioCoverage),
	}
	if validation == nil {
		return surface
	}

	surface.WorkflowID = validation.WorkflowID
	surface.WorkflowName = workflowInspectionName(validation)
	surface.Conformance = validation.Conformance
	surface.VisualProjection = ProjectWorkflowSemanticGraph(validation.SemanticGraph, validation.Conformance)
	surface.Adjacency = workflowAdjacencyList(validation.SemanticGraph)
	surface.Mermaid = surface.VisualProjection.ToMermaid()

	if program, err := ParseAndValidateDSL(dslSource); err == nil {
		summary := BuildDSLCoverageSummary(program)
		surface.Coverage = append(surface.Coverage, summary.Labels...)
	}

	return surface
}

func (p WorkflowVisualProjection) ToMermaid() string {
	if len(p.Nodes) == 0 {
		return "flowchart LR\n"
	}

	var b strings.Builder
	b.WriteString("flowchart LR\n")
	b.WriteString(fmt.Sprintf("  workflow_status[%q]\n", "Conformance: "+string(p.Conformance.Profile)))

	for _, node := range p.Nodes {
		b.WriteString(fmt.Sprintf("  %s[%q]\n", mermaidNodeID(node.ID), workflowMermaidLabel(node)))
	}
	for _, edge := range p.Edges {
		b.WriteString(fmt.Sprintf("  %s %s %s\n",
			mermaidNodeID(edge.From),
			mermaidEdgeOperator(edge.ConnectionType),
			mermaidNodeID(edge.To),
		))
	}

	root := mermaidNodeID(p.Nodes[0].ID)
	for _, node := range p.Nodes {
		if node.Kind == SemanticNodeWorkflow {
			root = mermaidNodeID(node.ID)
			break
		}
	}
	b.WriteString(fmt.Sprintf("  %s -.-> workflow_status\n", root))
	return b.String()
}

func workflowInspectionName(validation *WorkflowValidationResult) string {
	if validation == nil || validation.SemanticGraph == nil {
		return ""
	}
	return validation.SemanticGraph.WorkflowName
}

func workflowAdjacencyList(graph *WorkflowSemanticGraph) []WorkflowAdjacencyNode {
	if graph == nil {
		return []WorkflowAdjacencyNode{}
	}

	outgoing := make(map[SemanticNodeID][]SemanticNodeID, len(graph.Nodes))
	incoming := make(map[SemanticNodeID][]SemanticNodeID, len(graph.Nodes))
	for _, edge := range graph.Edges {
		outgoing[edge.From] = append(outgoing[edge.From], edge.To)
		incoming[edge.To] = append(incoming[edge.To], edge.From)
	}

	nodes := make([]WorkflowAdjacencyNode, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes = append(nodes, WorkflowAdjacencyNode{
			ID:       node.ID,
			Label:    node.Label,
			Kind:     node.Kind,
			Outgoing: sortedNodeIDs(outgoing[node.ID]),
			Incoming: sortedNodeIDs(incoming[node.ID]),
			Source:   node.Source,
		})
	}
	return nodes
}

func sortedNodeIDs(ids []SemanticNodeID) []SemanticNodeID {
	if len(ids) == 0 {
		return nil
	}
	out := append([]SemanticNodeID(nil), ids...)
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

func workflowMermaidLabel(node WorkflowVisualNode) string {
	return fmt.Sprintf("%s (%s)", node.Label, node.Kind)
}

func mermaidNodeID(id SemanticNodeID) string {
	raw := strings.ToLower(string(id))
	var b strings.Builder
	for _, r := range raw {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	text := strings.Trim(b.String(), "_")
	if text == "" {
		return "node"
	}
	return text
}

func mermaidEdgeOperator(kind SemanticEdgeKind) string {
	switch kind {
	case SemanticEdgeGoverns:
		return "-.->"
	case SemanticEdgeRequires:
		return "-->"
	case SemanticEdgeBranches:
		return "-->"
	case SemanticEdgeContains:
		return "-->"
	case SemanticEdgeNext:
		return "-->"
	default:
		return "-->"
	}
}

func cloneWorkflowScenarioCoverageRefs(items []WorkflowScenarioCoverageRef) []WorkflowScenarioCoverageRef {
	if len(items) == 0 {
		return nil
	}
	out := append([]WorkflowScenarioCoverageRef(nil), items...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScenarioID != out[j].ScenarioID {
			return out[i].ScenarioID < out[j].ScenarioID
		}
		return out[i].Title < out[j].Title
	})
	return out
}
