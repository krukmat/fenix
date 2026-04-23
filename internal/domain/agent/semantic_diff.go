package agent

import (
	"encoding/json"
	"sort"
)

type SemanticDiffChangeKind string

const (
	SemanticDiffAdded    SemanticDiffChangeKind = "added"
	SemanticDiffRemoved  SemanticDiffChangeKind = "removed"
	SemanticDiffModified SemanticDiffChangeKind = "modified"

	semanticNodeFieldSource = "source"
)

type WorkflowSemanticDiff struct {
	HasSemanticChanges bool                         `json:"has_semantic_changes"`
	LayoutOnly         bool                         `json:"layout_only"`
	NodeChanges        []WorkflowSemanticNodeChange `json:"node_changes,omitempty"`
	EdgeChanges        []WorkflowSemanticEdgeChange `json:"edge_changes,omitempty"`
}

type WorkflowSemanticNodeChange struct {
	Kind   SemanticDiffChangeKind `json:"kind"`
	ID     SemanticNodeID         `json:"id"`
	Before *WorkflowSemanticNode  `json:"before,omitempty"`
	After  *WorkflowSemanticNode  `json:"after,omitempty"`
	Fields []string               `json:"fields,omitempty"`
}

type WorkflowSemanticEdgeChange struct {
	Kind   SemanticDiffChangeKind `json:"kind"`
	ID     string                 `json:"id"`
	Before *WorkflowSemanticEdge  `json:"before,omitempty"`
	After  *WorkflowSemanticEdge  `json:"after,omitempty"`
}

func DiffWorkflowSemanticGraphs(before *WorkflowSemanticGraph, after *WorkflowSemanticGraph) WorkflowSemanticDiff {
	nodeChanges, positionsChanged := diffSemanticNodes(before, after)
	edgeChanges := diffSemanticEdges(before, after)
	hasSemanticChanges := len(nodeChanges) > 0 || len(edgeChanges) > 0
	return WorkflowSemanticDiff{
		HasSemanticChanges: hasSemanticChanges,
		LayoutOnly:         !hasSemanticChanges && positionsChanged,
		NodeChanges:        nodeChanges,
		EdgeChanges:        edgeChanges,
	}
}

func diffSemanticNodes(before, after *WorkflowSemanticGraph) ([]WorkflowSemanticNodeChange, bool) {
	changes := []WorkflowSemanticNodeChange{}
	beforeNodes := semanticNodesByID(before)
	afterNodes := semanticNodesByID(after)
	positionsChanged := appendChangedOrRemovedSemanticNodes(&changes, beforeNodes, afterNodes)
	appendAddedSemanticNodes(&changes, beforeNodes, afterNodes)
	return changes, positionsChanged
}

func appendChangedOrRemovedSemanticNodes(changes *[]WorkflowSemanticNodeChange, beforeNodes, afterNodes map[SemanticNodeID]WorkflowSemanticNode) bool {
	positionsChanged := false
	for _, id := range sortedSemanticNodeIDs(beforeNodes) {
		beforeNode := beforeNodes[id]
		afterNode, ok := afterNodes[id]
		if !ok {
			*changes = append(*changes, WorkflowSemanticNodeChange{Kind: SemanticDiffRemoved, ID: id, Before: ptrWorkflowSemanticNode(beforeNode)})
			continue
		}
		if fields := changedSemanticNodeFields(beforeNode, afterNode); len(fields) > 0 {
			*changes = append(*changes, WorkflowSemanticNodeChange{Kind: SemanticDiffModified, ID: id, Before: ptrWorkflowSemanticNode(beforeNode), After: ptrWorkflowSemanticNode(afterNode), Fields: fields})
			continue
		}
		if beforeNode.Position != afterNode.Position {
			positionsChanged = true
		}
	}
	return positionsChanged
}

func appendAddedSemanticNodes(changes *[]WorkflowSemanticNodeChange, beforeNodes, afterNodes map[SemanticNodeID]WorkflowSemanticNode) {
	for _, id := range sortedSemanticNodeIDs(afterNodes) {
		if _, ok := beforeNodes[id]; ok {
			continue
		}
		*changes = append(*changes, WorkflowSemanticNodeChange{Kind: SemanticDiffAdded, ID: id, After: ptrWorkflowSemanticNode(afterNodes[id])})
	}
}

func diffSemanticEdges(before, after *WorkflowSemanticGraph) []WorkflowSemanticEdgeChange {
	changes := []WorkflowSemanticEdgeChange{}
	beforeEdges := semanticEdgesByID(before)
	afterEdges := semanticEdgesByID(after)

	for _, id := range sortedStringKeys(beforeEdges) {
		if _, ok := afterEdges[id]; ok {
			continue
		}
		changes = append(changes, WorkflowSemanticEdgeChange{Kind: SemanticDiffRemoved, ID: id, Before: ptrWorkflowSemanticEdge(beforeEdges[id])})
	}
	for _, id := range sortedStringKeys(afterEdges) {
		if _, ok := beforeEdges[id]; ok {
			continue
		}
		changes = append(changes, WorkflowSemanticEdgeChange{Kind: SemanticDiffAdded, ID: id, After: ptrWorkflowSemanticEdge(afterEdges[id])})
	}
	return changes
}

func semanticNodesByID(graph *WorkflowSemanticGraph) map[SemanticNodeID]WorkflowSemanticNode {
	out := map[SemanticNodeID]WorkflowSemanticNode{}
	if graph == nil {
		return out
	}
	for _, node := range graph.Nodes {
		out[node.ID] = node
	}
	return out
}

func semanticEdgesByID(graph *WorkflowSemanticGraph) map[string]WorkflowSemanticEdge {
	out := map[string]WorkflowSemanticEdge{}
	if graph == nil {
		return out
	}
	for _, edge := range graph.Edges {
		out[semanticEdgeID(edge)] = edge
	}
	return out
}

func semanticEdgeID(edge WorkflowSemanticEdge) string {
	return string(edge.From) + "\x00" + string(edge.To) + "\x00" + string(edge.Kind) + "\x00" + semanticJSON(edge.Properties)
}

func changedSemanticNodeFields(before WorkflowSemanticNode, after WorkflowSemanticNode) []string {
	fields := make([]string, 0, 5)
	if before.Kind != after.Kind {
		fields = append(fields, "kind")
	}
	if before.Label != after.Label {
		fields = append(fields, "label")
	}
	if before.Source != after.Source {
		fields = append(fields, semanticNodeFieldSource)
	}
	if before.Effect != after.Effect {
		fields = append(fields, "effect")
	}
	if semanticJSON(before.Properties) != semanticJSON(after.Properties) {
		fields = append(fields, "properties")
	}
	return fields
}

func sortedSemanticNodeIDs(values map[SemanticNodeID]WorkflowSemanticNode) []SemanticNodeID {
	ids := make([]SemanticNodeID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func sortedStringKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func semanticJSON(value any) string {
	if value == nil {
		return ""
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func ptrWorkflowSemanticNode(node WorkflowSemanticNode) *WorkflowSemanticNode {
	return &node
}

func ptrWorkflowSemanticEdge(edge WorkflowSemanticEdge) *WorkflowSemanticEdge {
	return &edge
}
