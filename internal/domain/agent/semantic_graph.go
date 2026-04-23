package agent

type SemanticEdgeKind string

const (
	SemanticEdgeContains SemanticEdgeKind = "contains"
	SemanticEdgeNext     SemanticEdgeKind = "next"
	SemanticEdgeBranches SemanticEdgeKind = "branches"
	SemanticEdgeGoverns  SemanticEdgeKind = "governs"
	SemanticEdgeRequires SemanticEdgeKind = "requires"
)

type WorkflowSemanticEdge struct {
	From       SemanticNodeID   `json:"from"`
	To         SemanticNodeID   `json:"to"`
	Kind       SemanticEdgeKind `json:"kind"`
	Properties map[string]any   `json:"properties,omitempty"`
}

type WorkflowSemanticGraph struct {
	WorkflowName string                 `json:"workflow_name,omitempty"`
	Nodes        []WorkflowSemanticNode `json:"nodes"`
	Edges        []WorkflowSemanticEdge `json:"edges"`
}

func NewWorkflowSemanticGraph(workflowName string) *WorkflowSemanticGraph {
	return &WorkflowSemanticGraph{
		WorkflowName: workflowName,
		Nodes:        []WorkflowSemanticNode{},
		Edges:        []WorkflowSemanticEdge{},
	}
}

func (g *WorkflowSemanticGraph) AddNode(node WorkflowSemanticNode) {
	if g == nil {
		return
	}
	g.Nodes = append(g.Nodes, node)
}

func (g *WorkflowSemanticGraph) AddEdge(edge WorkflowSemanticEdge) {
	if g == nil {
		return
	}
	g.Edges = append(g.Edges, edge)
}

func (g *WorkflowSemanticGraph) FindNode(id SemanticNodeID) (WorkflowSemanticNode, bool) {
	if g == nil {
		return WorkflowSemanticNode{}, false
	}
	for _, node := range g.Nodes {
		if node.ID == id {
			return node, true
		}
	}
	return WorkflowSemanticNode{}, false
}
