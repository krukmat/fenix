package agent

type VisualAuthoringNodeID string
type VisualAuthoringEdgeID string

type VisualAuthoringGraph struct {
	WorkflowName string                `json:"workflow_name,omitempty"`
	Nodes        []VisualAuthoringNode `json:"nodes"`
	Edges        []VisualAuthoringEdge `json:"edges"`
	Metadata     map[string]any        `json:"metadata,omitempty"`
}

type VisualAuthoringNode struct {
	ID         VisualAuthoringNodeID   `json:"id"`
	Kind       SemanticNodeKind        `json:"kind"`
	Label      string                  `json:"label"`
	Position   WorkflowVisualPosition  `json:"position"`
	Data       VisualAuthoringNodeData `json:"data,omitempty"`
	Properties map[string]any          `json:"properties,omitempty"`
}

type VisualAuthoringNodeData struct {
	WorkflowName string         `json:"workflow_name,omitempty"`
	Event        string         `json:"event,omitempty"`
	Expression   string         `json:"expression,omitempty"`
	Action       string         `json:"action,omitempty"`
	Target       string         `json:"target,omitempty"`
	Value        string         `json:"value,omitempty"`
	AgentName    string         `json:"agent_name,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
	Permit       string         `json:"permit,omitempty"`
	Grounds      map[string]any `json:"grounds,omitempty"`
	DelegateTo   string         `json:"delegate_to,omitempty"`
	Invariant    string         `json:"invariant,omitempty"`
	Budget       map[string]any `json:"budget,omitempty"`
}

type VisualAuthoringEdge struct {
	ID             VisualAuthoringEdgeID `json:"id"`
	From           VisualAuthoringNodeID `json:"from"`
	To             VisualAuthoringNodeID `json:"to"`
	ConnectionType SemanticEdgeKind      `json:"connection_type"`
	Properties     map[string]any        `json:"properties,omitempty"`
}

type VisualAuthoringValidationResult struct {
	Passed     bool        `json:"passed"`
	Violations []Violation `json:"violations,omitempty"`
}

func NewVisualAuthoringGraph(workflowName string) *VisualAuthoringGraph {
	return &VisualAuthoringGraph{
		WorkflowName: workflowName,
		Nodes:        []VisualAuthoringNode{},
		Edges:        []VisualAuthoringEdge{},
	}
}

func NewVisualAuthoringNode(id VisualAuthoringNodeID, kind SemanticNodeKind, label string, position WorkflowVisualPosition) VisualAuthoringNode {
	return VisualAuthoringNode{
		ID:       id,
		Kind:     kind,
		Label:    label,
		Position: position,
	}
}

func NewVisualAuthoringEdge(id VisualAuthoringEdgeID, from VisualAuthoringNodeID, to VisualAuthoringNodeID, connectionType SemanticEdgeKind) VisualAuthoringEdge {
	return VisualAuthoringEdge{
		ID:             id,
		From:           from,
		To:             to,
		ConnectionType: connectionType,
	}
}

func (g *VisualAuthoringGraph) AddNode(node VisualAuthoringNode) {
	if g == nil {
		return
	}
	g.Nodes = append(g.Nodes, node)
}

func (g *VisualAuthoringGraph) AddEdge(edge VisualAuthoringEdge) {
	if g == nil {
		return
	}
	g.Edges = append(g.Edges, edge)
}

func SupportedVisualAuthoringNodeKinds() []SemanticNodeKind {
	return []SemanticNodeKind{
		SemanticNodeWorkflow,
		SemanticNodeTrigger,
		SemanticNodeAction,
		SemanticNodeDecision,
		SemanticNodeGrounds,
		SemanticNodePermit,
		SemanticNodeDelegate,
		SemanticNodeInvariant,
		SemanticNodeBudget,
	}
}

func IsSupportedVisualAuthoringNodeKind(kind SemanticNodeKind) bool {
	for _, supported := range SupportedVisualAuthoringNodeKinds() {
		if kind == supported {
			return true
		}
	}
	return false
}

func ValidateVisualAuthoringGraph(graph *VisualAuthoringGraph) VisualAuthoringValidationResult {
	violations := validateVisualAuthoringGraph(graph)
	return VisualAuthoringValidationResult{
		Passed:     len(violations) == 0,
		Violations: violations,
	}
}

func validateVisualAuthoringGraph(graph *VisualAuthoringGraph) []Violation {
	if graph == nil {
		return []Violation{visualAuthoringViolation("visual_graph_missing", "visual authoring graph is required", "graph")}
	}
	violations, nodeIDs, hasTrigger := validateVisualAuthoringNodes(graph.Nodes)
	if !hasTrigger {
		violations = append(violations, visualAuthoringViolation("visual_trigger_missing", "visual authoring graph must include a trigger node", "nodes"))
	}
	violations = append(violations, validateVisualAuthoringEdges(graph.Edges, nodeIDs)...)
	return violations
}

func validateVisualAuthoringNodes(nodes []VisualAuthoringNode) ([]Violation, map[VisualAuthoringNodeID]struct{}, bool) {
	violations := []Violation{}
	nodeIDs := make(map[VisualAuthoringNodeID]struct{}, len(nodes))
	hasTrigger := false
	for _, node := range nodes {
		nodeViolations, nodeValid := validateVisualAuthoringNode(node, nodeIDs)
		violations = append(violations, nodeViolations...)
		if !nodeValid {
			continue
		}
		if node.Kind == SemanticNodeTrigger {
			hasTrigger = true
		}
	}
	return violations, nodeIDs, hasTrigger
}

func validateVisualAuthoringNode(node VisualAuthoringNode, nodeIDs map[VisualAuthoringNodeID]struct{}) ([]Violation, bool) {
	if node.ID == "" {
		return []Violation{visualAuthoringViolation("visual_node_id_missing", "visual authoring node id is required", "nodes")}, false
	}
	violations := []Violation{}
	if _, exists := nodeIDs[node.ID]; exists {
		violations = append(violations, visualAuthoringViolation("visual_node_duplicate", "visual authoring node id must be unique", "node "+string(node.ID)))
	}
	nodeIDs[node.ID] = struct{}{}
	if !IsSupportedVisualAuthoringNodeKind(node.Kind) {
		violations = append(violations, visualAuthoringViolation("visual_node_unsupported", "visual authoring node kind is not supported", "node "+string(node.ID)))
	}
	return violations, true
}

func validateVisualAuthoringEdges(edges []VisualAuthoringEdge, nodeIDs map[VisualAuthoringNodeID]struct{}) []Violation {
	violations := []Violation{}
	for _, edge := range edges {
		violations = append(violations, validateVisualAuthoringEdge(edge, nodeIDs)...)
	}
	return violations
}

func validateVisualAuthoringEdge(edge VisualAuthoringEdge, nodeIDs map[VisualAuthoringNodeID]struct{}) []Violation {
	violations := []Violation{}
	if edge.ID == "" {
		violations = append(violations, visualAuthoringViolation("visual_edge_id_missing", "visual authoring edge id is required", "edges"))
	}
	if _, ok := nodeIDs[edge.From]; !ok {
		violations = append(violations, visualAuthoringViolation("visual_edge_from_missing", "visual authoring edge source node does not exist", "edge "+string(edge.ID)))
	}
	if _, ok := nodeIDs[edge.To]; !ok {
		violations = append(violations, visualAuthoringViolation("visual_edge_to_missing", "visual authoring edge target node does not exist", "edge "+string(edge.ID)))
	}
	if !isSupportedVisualAuthoringEdgeKind(edge.ConnectionType) {
		violations = append(violations, visualAuthoringViolation("visual_edge_unsupported", "visual authoring edge connection_type is not supported", "edge "+string(edge.ID)))
	}
	return violations
}

func isSupportedVisualAuthoringEdgeKind(kind SemanticEdgeKind) bool {
	switch kind {
	case SemanticEdgeContains, SemanticEdgeNext, SemanticEdgeBranches, SemanticEdgeGoverns, SemanticEdgeRequires:
		return true
	default:
		return false
	}
}

func visualAuthoringViolation(code string, description string, location string) Violation {
	return normalizeViolation(Violation{
		Code:        code,
		Type:        "visual_authoring_validation",
		Description: description,
		Location:    location,
	})
}
