package agent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type VisualSourceResult struct {
	DSLSource  string `json:"dsl_source"`
	SpecSource string `json:"spec_source,omitempty"`
}

const (
	visualIndentUnit  = "  "
	visualEmptyObject = "{}"
)

func GenerateDSLSourceFromVisualAuthoringGraph(graph *VisualAuthoringGraph) (VisualSourceResult, error) {
	result, err := GenerateSourcesFromVisualAuthoringGraph(graph)
	if err != nil {
		return VisualSourceResult{}, err
	}
	result.SpecSource = ""
	return result, nil
}

func GenerateSourcesFromVisualAuthoringGraph(graph *VisualAuthoringGraph) (VisualSourceResult, error) {
	validation := ValidateVisualAuthoringGraph(graph)
	if !validation.Passed {
		return VisualSourceResult{}, fmt.Errorf("visual authoring graph is invalid: %s", validation.Violations[0].Code)
	}
	nodes := visualAuthoringNodesByID(graph.Nodes)
	workflow, trigger, err := visualAuthoringWorkflowTrigger(graph.Nodes)
	if err != nil {
		return VisualSourceResult{}, err
	}
	if validationErr := validateVisualDecisionBranches(graph, nodes); validationErr != nil {
		return VisualSourceResult{}, validationErr
	}
	lines := []string{
		"WORKFLOW " + visualWorkflowName(graph, workflow),
		"ON " + visualNodeValue(trigger.Data.Event, trigger.Label),
	}
	lines = append(lines, renderVisualStatements(visualRootStatementNodes(graph, nodes), graph, nodes, 0)...)
	specSource, err := renderVisualCartaSource(graph, workflow)
	if err != nil {
		return VisualSourceResult{}, err
	}
	return VisualSourceResult{
		DSLSource:  strings.Join(lines, "\n") + "\n",
		SpecSource: specSource,
	}, nil
}

func visualAuthoringNodesByID(nodes []VisualAuthoringNode) map[VisualAuthoringNodeID]VisualAuthoringNode {
	out := make(map[VisualAuthoringNodeID]VisualAuthoringNode, len(nodes))
	for _, node := range nodes {
		out[node.ID] = node
	}
	return out
}

func visualAuthoringWorkflowTrigger(nodes []VisualAuthoringNode) (VisualAuthoringNode, VisualAuthoringNode, error) {
	var workflow VisualAuthoringNode
	var trigger VisualAuthoringNode
	for _, node := range nodes {
		switch {
		case node.Kind == SemanticNodeWorkflow && workflow.ID == "":
			workflow = node
		case node.Kind == SemanticNodeTrigger && trigger.ID == "":
			trigger = node
		}
	}
	if err := requireVisualAuthoringRootNodes(workflow, trigger); err != nil {
		return VisualAuthoringNode{}, VisualAuthoringNode{}, err
	}
	return workflow, trigger, nil
}

func requireVisualAuthoringRootNodes(workflow, trigger VisualAuthoringNode) error {
	if workflow.ID == "" {
		return fmt.Errorf("visual authoring graph must include a workflow node")
	}
	if trigger.ID == "" {
		return fmt.Errorf("visual authoring graph must include a trigger node")
	}
	return nil
}

func visualWorkflowName(graph *VisualAuthoringGraph, workflow VisualAuthoringNode) string {
	return visualNodeValue(workflow.Data.WorkflowName, visualNodeValue(graph.WorkflowName, workflow.Label))
}

func visualRootStatementNodes(graph *VisualAuthoringGraph, nodes map[VisualAuthoringNodeID]VisualAuthoringNode) []VisualAuthoringNode {
	children := map[VisualAuthoringNodeID]struct{}{}
	for _, edge := range graph.Edges {
		if edge.ConnectionType == SemanticEdgeBranches {
			children[edge.To] = struct{}{}
		}
	}
	out := []VisualAuthoringNode{}
	for _, node := range graph.Nodes {
		if !isVisualDSLStatementNode(node.Kind) {
			continue
		}
		if _, isChild := children[node.ID]; isChild {
			continue
		}
		out = append(out, nodes[node.ID])
	}
	sortVisualAuthoringNodes(out)
	return out
}

func renderVisualStatements(nodes []VisualAuthoringNode, graph *VisualAuthoringGraph, indexed map[VisualAuthoringNodeID]VisualAuthoringNode, indent int) []string {
	lines := []string{}
	for _, node := range nodes {
		lines = append(lines, renderVisualStatement(node, graph, indexed, indent)...)
	}
	return lines
}

func renderVisualStatement(node VisualAuthoringNode, graph *VisualAuthoringGraph, indexed map[VisualAuthoringNodeID]VisualAuthoringNode, indent int) []string {
	prefix := strings.Repeat(visualIndentUnit, indent)
	switch node.Kind {
	case SemanticNodeDecision:
		return renderVisualDecision(node, graph, indexed, indent)
	case SemanticNodeAction:
		return []string{prefix + renderVisualAction(node)}
	default:
		return nil
	}
}

func renderVisualDecision(node VisualAuthoringNode, graph *VisualAuthoringGraph, indexed map[VisualAuthoringNodeID]VisualAuthoringNode, indent int) []string {
	children := visualDecisionChildren(node.ID, graph, indexed)
	lines := []string{strings.Repeat(visualIndentUnit, indent) + "IF " + visualNodeValue(node.Data.Expression, node.Label) + ":"}
	lines = append(lines, renderVisualStatements(children, graph, indexed, indent+1)...)
	return lines
}

func validateVisualDecisionBranches(graph *VisualAuthoringGraph, indexed map[VisualAuthoringNodeID]VisualAuthoringNode) error {
	for _, node := range graph.Nodes {
		if node.Kind != SemanticNodeDecision {
			continue
		}
		if len(visualDecisionChildren(node.ID, graph, indexed)) == 0 {
			return fmt.Errorf("decision node %s must have at least one branch child", node.ID)
		}
	}
	return nil
}

func visualDecisionChildren(id VisualAuthoringNodeID, graph *VisualAuthoringGraph, indexed map[VisualAuthoringNodeID]VisualAuthoringNode) []VisualAuthoringNode {
	out := []VisualAuthoringNode{}
	for _, edge := range graph.Edges {
		if edge.From != id || edge.ConnectionType != SemanticEdgeBranches {
			continue
		}
		if node, ok := indexed[edge.To]; ok && isVisualDSLStatementNode(node.Kind) {
			out = append(out, node)
		}
	}
	sortVisualAuthoringNodes(out)
	return out
}

func renderVisualAction(node VisualAuthoringNode) string {
	switch strings.ToLower(node.Data.Action) {
	case string(SemanticEffectNotify):
		return "NOTIFY " + visualNodeValue(node.Data.Target, "contact") + " WITH " + visualExpression(node.Data.Value)
	case string(RuntimeOperationAgent):
		return "AGENT " + visualNodeValue(node.Data.AgentName, node.Data.Target) + visualOptionalWith(node.Data.Payload, node.Data.Value)
	default:
		return "SET " + visualNodeValue(node.Data.Target, node.Label) + " = " + visualExpression(node.Data.Value)
	}
}

func visualOptionalWith(payload map[string]any, value string) string {
	if len(payload) != 0 {
		return " WITH " + visualJSON(payload)
	}
	if strings.TrimSpace(value) != "" {
		return " WITH " + visualExpression(value)
	}
	return ""
}

func visualExpression(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return strconv.Quote("")
	}
	if looksLikeVisualExpression(trimmed) {
		return trimmed
	}
	return strconv.Quote(trimmed)
}

func visualJSON(value map[string]any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return visualEmptyObject
	}
	return string(raw)
}

func looksLikeVisualExpression(value string) bool {
	if strings.HasPrefix(value, "\"") || strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") {
		return true
	}
	return value == "true" || value == "false" || value == "null" || isVisualNumber(value)
}

func isVisualNumber(value string) bool {
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func visualNodeValue(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func isVisualDSLStatementNode(kind SemanticNodeKind) bool {
	return kind == SemanticNodeAction || kind == SemanticNodeDecision
}

func sortVisualAuthoringNodes(nodes []VisualAuthoringNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Position.Y == nodes[j].Position.Y {
			return nodes[i].Position.X < nodes[j].Position.X
		}
		return nodes[i].Position.Y < nodes[j].Position.Y
	})
}

func renderVisualCartaSource(graph *VisualAuthoringGraph, workflow VisualAuthoringNode) (string, error) {
	governance := visualGovernanceNodes(graph.Nodes)
	if len(governance) == 0 {
		return "", nil
	}
	if !hasVisualCartaAgentDirective(governance) {
		return "", fmt.Errorf("Carta source requires at least one agent governance directive")
	}
	lines := []string{"CARTA " + visualWorkflowName(graph, workflow)}
	lines = append(lines, renderVisualBudgetNodes(governance)...)
	lines = append(lines, "AGENT "+visualCartaAgentName(graph))
	for _, node := range governance {
		lines = append(lines, renderVisualCartaAgentNode(node)...)
	}
	return strings.Join(nonEmptyLines(lines), "\n") + "\n", nil
}

func visualGovernanceNodes(nodes []VisualAuthoringNode) []VisualAuthoringNode {
	out := []VisualAuthoringNode{}
	for _, node := range nodes {
		if isVisualCartaNode(node.Kind) {
			out = append(out, node)
		}
	}
	sortVisualAuthoringNodes(out)
	return out
}

func renderVisualBudgetNodes(nodes []VisualAuthoringNode) []string {
	lines := []string{}
	for _, node := range nodes {
		if node.Kind == SemanticNodeBudget {
			lines = append(lines, string(TokenBudget))
			lines = append(lines, renderVisualMapFields(node.Data.Budget, 2)...)
		}
	}
	return lines
}

func renderVisualCartaAgentNode(node VisualAuthoringNode) []string {
	switch node.Kind {
	case SemanticNodeGrounds:
		return append([]string{"  GROUNDS"}, renderVisualMapFields(node.Data.Grounds, 4)...)
	case SemanticNodePermit:
		return renderVisualPermit(node)
	case SemanticNodeDelegate:
		return renderVisualDelegate(node)
	case SemanticNodeInvariant:
		return []string{"  INVARIANT", "    never: " + strconv.Quote(visualNodeValue(node.Data.Invariant, node.Label))}
	default:
		return nil
	}
}

func renderVisualPermit(node VisualAuthoringNode) []string {
	permit := visualNodeValue(node.Data.Permit, node.Label)
	if strings.HasPrefix(permit, "PERMIT ") {
		permit = strings.TrimSpace(strings.TrimPrefix(permit, "PERMIT "))
	}
	return []string{"  PERMIT " + permit}
}

func renderVisualDelegate(node VisualAuthoringNode) []string {
	lines := []string{"  DELEGATE TO HUMAN"}
	if node.Data.Expression != "" {
		lines = append(lines, "    when: "+node.Data.Expression)
	}
	lines = append(lines, "    reason: "+strconv.Quote(visualNodeValue(node.Data.DelegateTo, visualNodeValue(node.Data.Value, node.Label))))
	return lines
}

func renderVisualMapFields(fields map[string]any, indent int) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, strings.Repeat(" ", indent)+key+": "+visualCartaScalar(fields[key]))
	}
	return lines
}

func visualCartaScalar(value any) string {
	switch typed := value.(type) {
	case string:
		if strings.HasPrefix(typed, "[") || strings.Contains(typed, " ") {
			return typed
		}
		return typed
	case []string:
		quoted := make([]string, 0, len(typed))
		for _, item := range typed {
			quoted = append(quoted, strconv.Quote(item))
		}
		return "[" + strings.Join(quoted, ", ") + "]"
	default:
		return fmt.Sprint(typed)
	}
}

func visualCartaAgentName(graph *VisualAuthoringGraph) string {
	for _, node := range graph.Nodes {
		if node.Kind == SemanticNodeAction && node.Data.Action == string(RuntimeOperationAgent) {
			return visualNodeValue(node.Data.AgentName, "visual_author")
		}
	}
	return "visual_author"
}

func isVisualCartaNode(kind SemanticNodeKind) bool {
	switch kind {
	case SemanticNodeGrounds, SemanticNodePermit, SemanticNodeDelegate, SemanticNodeInvariant, SemanticNodeBudget:
		return true
	default:
		return false
	}
}

func hasVisualCartaAgentDirective(nodes []VisualAuthoringNode) bool {
	for _, node := range nodes {
		switch node.Kind {
		case SemanticNodeGrounds, SemanticNodePermit, SemanticNodeDelegate, SemanticNodeInvariant:
			return true
		default:
			continue
		}
	}
	return false
}

func nonEmptyLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
