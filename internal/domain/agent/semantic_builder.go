package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	semanticDepthPartFormat = "depth:%d"
	semanticJoinSeparator   = ", "
	semanticNullLabel       = "null"
)

func BuildWorkflowSemanticGraphFromDSL(source string) (*WorkflowSemanticGraph, error) {
	program, err := ParseDSL(source)
	if err != nil {
		return nil, err
	}
	return BuildWorkflowSemanticGraph(program), nil
}

func BuildWorkflowSemanticGraphFromSources(dslSource string, cartaSource string) (*WorkflowSemanticGraph, error) {
	program, err := ParseDSL(dslSource)
	if err != nil {
		return nil, err
	}
	carta, err := ParseCarta(cartaSource)
	if err != nil {
		return nil, err
	}
	return BuildWorkflowSemanticGraphWithCarta(program, carta), nil
}

func BuildWorkflowSemanticGraph(program *Program) *WorkflowSemanticGraph {
	return BuildWorkflowSemanticGraphWithCarta(program, nil)
}

func BuildWorkflowSemanticGraphWithCarta(program *Program, carta *CartaSummary) *WorkflowSemanticGraph {
	if program == nil || program.Workflow == nil {
		builder := semanticGraphBuilder{
			graph: NewWorkflowSemanticGraph(cartaWorkflowName(carta)),
			scope: cartaWorkflowName(carta),
		}
		builder.addCarta(carta)
		return builder.graph
	}

	builder := semanticGraphBuilder{
		graph: NewWorkflowSemanticGraph(program.Workflow.Name),
		scope: program.Workflow.Name,
	}
	builder.addWorkflow(program.Workflow)
	builder.addCarta(carta)
	return builder.graph
}

type semanticGraphBuilder struct {
	graph      *WorkflowSemanticGraph
	scope      string
	workflowID SemanticNodeID
}

func (b *semanticGraphBuilder) addWorkflow(workflow *WorkflowDecl) {
	workflowID := b.nodeID(SemanticNodeWorkflow, 0, "WORKFLOW", workflow.Name)
	workflowNode := NewWorkflowSemanticNode(workflowID, SemanticNodeWorkflow, SemanticSourceDSL)
	workflowNode.Label = workflow.Name
	workflowNode.Position = workflow.Position
	workflowNode.Properties = map[string]any{
		"statement": "WORKFLOW",
		"name":      workflow.Name,
	}
	b.graph.AddNode(workflowNode)
	b.workflowID = workflowID

	var previous SemanticNodeID
	if workflow.Trigger != nil {
		triggerID := b.addTrigger(workflowID, workflow.Trigger)
		previous = triggerID
	}

	b.addStatements(workflowID, workflow.Body, &previous, 0)
}

func (b *semanticGraphBuilder) addCarta(carta *CartaSummary) {
	if carta == nil {
		return
	}
	if b.scope == "" {
		b.scope = carta.Name
	}
	parent := b.workflowID
	if parent == "" {
		parent = b.addCartaWorkflow(carta)
	}
	b.addCartaNodes(parent, carta)
}

func (b *semanticGraphBuilder) addCartaNodes(parent SemanticNodeID, carta *CartaSummary) {
	for ordinal, agent := range carta.Agents {
		b.addCartaAgent(parent, agent, ordinal)
	}
	if carta.Grounds != nil {
		b.addCartaGrounds(parent, carta.Grounds)
	}
	for ordinal, permit := range carta.Permits {
		b.addCartaPermit(parent, permit, ordinal)
	}
	for ordinal, delegate := range carta.Delegates {
		b.addCartaDelegate(parent, delegate, ordinal)
	}
	for ordinal, invariant := range carta.Invariants {
		b.addCartaInvariant(parent, invariant, ordinal)
	}
	if carta.Budget != nil {
		b.addCartaBudget(parent, carta.Budget)
	}
}

func (b *semanticGraphBuilder) addCartaWorkflow(carta *CartaSummary) SemanticNodeID {
	workflowID := b.nodeIDFrom(SemanticSourceCarta, SemanticNodeWorkflow, 0, "CARTA", carta.Name)
	node := NewWorkflowSemanticNode(workflowID, SemanticNodeWorkflow, SemanticSourceCarta)
	node.Label = carta.Name
	node.Properties = map[string]any{
		"statement": "CARTA",
		"name":      carta.Name,
	}
	b.graph.AddNode(node)
	b.workflowID = workflowID
	return workflowID
}

func (b *semanticGraphBuilder) addCartaAgent(parent SemanticNodeID, agent CartaAgent, ordinal int) SemanticNodeID {
	nodeID := b.nodeIDFrom(SemanticSourceCarta, SemanticNodeDelegate, ordinal, "AGENT", agent.Name)
	node := NewWorkflowSemanticNode(nodeID, SemanticNodeDelegate, SemanticSourceCarta)
	node.Label = strings.TrimSpace("CARTA AGENT " + agent.Name)
	node.Effect = SemanticEffectDelegate
	node.Properties = map[string]any{
		"statement": "AGENT",
		"name":      agent.Name,
	}
	b.graph.AddNode(node)
	b.addEdge(nodeID, parent, SemanticEdgeGoverns, nil)
	return nodeID
}

func (b *semanticGraphBuilder) addCartaGrounds(parent SemanticNodeID, grounds *CartaGrounds) SemanticNodeID {
	groundsToken := string(TokenGrounds)
	nodeID := b.nodeIDFrom(SemanticSourceCarta, SemanticNodeGrounds, 0,
		groundsToken,
		fmt.Sprintf("min_sources:%d", grounds.MinSources),
		string(grounds.MinConfidence),
		fmt.Sprintf("max_staleness:%d %s", grounds.MaxStaleness, grounds.MaxAgeUnit),
		strings.Join(grounds.Types, ","),
	)
	node := NewWorkflowSemanticNode(nodeID, SemanticNodeGrounds, SemanticSourceCarta)
	node.Label = groundsToken
	node.Effect = SemanticEffectGovernance
	node.Properties = map[string]any{
		"statement":      groundsToken,
		"min_sources":    grounds.MinSources,
		"min_confidence": string(grounds.MinConfidence),
		"max_staleness":  grounds.MaxStaleness,
		"max_age_unit":   grounds.MaxAgeUnit,
		"types":          grounds.Types,
	}
	b.graph.AddNode(node)
	b.addEdge(parent, nodeID, SemanticEdgeRequires, nil)
	return nodeID
}

func (b *semanticGraphBuilder) addCartaPermit(parent SemanticNodeID, permit CartaPermit, ordinal int) SemanticNodeID {
	nodeID := b.nodeIDFrom(SemanticSourceCarta, SemanticNodePermit, ordinal, "PERMIT", permit.Tool, permit.When, semanticPermitRateLabel(permit.Rate), semanticPermitApprovalLabel(permit.Approval))
	node := NewWorkflowSemanticNode(nodeID, SemanticNodePermit, SemanticSourceCarta)
	node.Label = strings.TrimSpace("PERMIT " + permit.Tool)
	node.Effect = SemanticEffectGovernance
	node.Properties = map[string]any{
		"statement": "PERMIT",
		"tool":      permit.Tool,
	}
	if permit.When != "" {
		node.Properties["when"] = permit.When
	}
	if permit.Rate != nil {
		node.Properties["rate"] = map[string]any{"value": permit.Rate.Value, "unit": permit.Rate.Unit}
	}
	if permit.Approval != nil {
		node.Properties["approval"] = permit.Approval.Mode
	}
	b.graph.AddNode(node)
	b.addEdge(nodeID, parent, SemanticEdgeGoverns, nil)
	return nodeID
}

func (b *semanticGraphBuilder) addCartaDelegate(parent SemanticNodeID, delegate CartaDelegate, ordinal int) SemanticNodeID {
	nodeID := b.nodeIDFrom(SemanticSourceCarta, SemanticNodeDelegate, ordinal, "DELEGATE", delegate.When, delegate.Reason, strings.Join(delegate.Package, ","))
	node := NewWorkflowSemanticNode(nodeID, SemanticNodeDelegate, SemanticSourceCarta)
	node.Label = "DELEGATE TO HUMAN"
	node.Effect = SemanticEffectDelegate
	node.Properties = map[string]any{
		"statement": "DELEGATE",
		"target":    "human",
		"package":   delegate.Package,
	}
	if delegate.When != "" {
		node.Properties["when"] = delegate.When
	}
	if delegate.Reason != "" {
		node.Properties["reason"] = delegate.Reason
	}
	b.graph.AddNode(node)
	b.addEdge(nodeID, parent, SemanticEdgeGoverns, nil)
	return nodeID
}

func (b *semanticGraphBuilder) addCartaInvariant(parent SemanticNodeID, invariant CartaInvariant, ordinal int) SemanticNodeID {
	nodeID := b.nodeIDFrom(SemanticSourceCarta, SemanticNodeInvariant, ordinal, "INVARIANT", invariant.Mode, invariant.Statement)
	node := NewWorkflowSemanticNode(nodeID, SemanticNodeInvariant, SemanticSourceCarta)
	node.Label = strings.TrimSpace("INVARIANT " + invariant.Mode)
	node.Effect = SemanticEffectGovernance
	node.Properties = map[string]any{
		"statement": "INVARIANT",
		"mode":      invariant.Mode,
		"value":     invariant.Statement,
	}
	b.graph.AddNode(node)
	b.addEdge(nodeID, parent, SemanticEdgeGoverns, nil)
	return nodeID
}

func (b *semanticGraphBuilder) addCartaBudget(parent SemanticNodeID, budget *CartaBudget) SemanticNodeID {
	budgetToken := string(TokenBudget)
	nodeID := b.nodeIDFrom(SemanticSourceCarta, SemanticNodeBudget, 0,
		budgetToken,
		fmt.Sprintf("daily_tokens:%d", budget.DailyTokens),
		fmt.Sprintf("daily_cost_usd:%g", budget.DailyCostUSD),
		fmt.Sprintf("executions_per_day:%d", budget.ExecutionsPerDay),
		budget.OnExceed,
	)
	node := NewWorkflowSemanticNode(nodeID, SemanticNodeBudget, SemanticSourceCarta)
	node.Label = budgetToken
	node.Effect = SemanticEffectGovernance
	node.Properties = map[string]any{
		"statement":          budgetToken,
		"daily_tokens":       budget.DailyTokens,
		"daily_cost_usd":     budget.DailyCostUSD,
		"executions_per_day": budget.ExecutionsPerDay,
		"on_exceed":          budget.OnExceed,
	}
	b.graph.AddNode(node)
	b.addEdge(nodeID, parent, SemanticEdgeGoverns, nil)
	return nodeID
}

func (b *semanticGraphBuilder) addTrigger(parent SemanticNodeID, trigger *OnDecl) SemanticNodeID {
	triggerID := b.nodeID(SemanticNodeTrigger, 0, "ON", trigger.Event)
	triggerNode := NewWorkflowSemanticNode(triggerID, SemanticNodeTrigger, SemanticSourceDSL)
	triggerNode.Label = trigger.Event
	triggerNode.Position = trigger.Position
	triggerNode.Effect = SemanticEffectRead
	triggerNode.Properties = map[string]any{
		"statement": "ON",
		"event":     trigger.Event,
	}
	b.graph.AddNode(triggerNode)
	b.addEdge(parent, triggerID, SemanticEdgeContains, nil)
	return triggerID
}

func (b *semanticGraphBuilder) addStatements(parent SemanticNodeID, statements []Statement, previous *SemanticNodeID, depth int) {
	for ordinal, statement := range statements {
		nodeID := b.addStatement(parent, statement, ordinal, depth)
		if nodeID == "" {
			continue
		}
		if previous != nil && *previous != "" {
			b.addEdge(*previous, nodeID, SemanticEdgeNext, nil)
		}
		if previous != nil {
			*previous = nodeID
		}
	}
}

func (b *semanticGraphBuilder) addStatement(parent SemanticNodeID, statement Statement, ordinal int, depth int) SemanticNodeID {
	switch stmt := statement.(type) {
	case *IfStatement:
		return b.addIfStatement(parent, stmt, ordinal, depth)
	case *SetStatement:
		return b.addSetStatement(parent, stmt, ordinal, depth)
	case *NotifyStatement:
		return b.addNotifyStatement(parent, stmt, ordinal, depth)
	case *AgentStatement:
		return b.addAgentStatement(parent, stmt, ordinal, depth)
	case *CallStatement:
		return b.addCallStatement(parent, stmt, ordinal, depth) // CLSF-54
	case *ApproveStatement:
		return b.addApproveStatement(parent, stmt, ordinal, depth) // CLSF-54
	default:
		return ""
	}
}

func (b *semanticGraphBuilder) addIfStatement(parent SemanticNodeID, stmt *IfStatement, ordinal int, depth int) SemanticNodeID {
	condition := semanticExpressionLabel(stmt.Condition)
	nodeID := b.nodeID(SemanticNodeDecision, ordinal, string(TokenIf), condition, fmt.Sprintf(semanticDepthPartFormat, depth))
	node := NewWorkflowSemanticNode(nodeID, SemanticNodeDecision, SemanticSourceDSL)
	node.Label = strings.TrimSpace("IF " + condition)
	node.Position = stmt.Position
	node.Effect = SemanticEffectRead
	node.Properties = map[string]any{
		"statement": "IF",
		"condition": condition,
	}
	b.graph.AddNode(node)
	b.addEdge(parent, nodeID, SemanticEdgeContains, nil)

	var nestedPrevious SemanticNodeID
	b.addStatements(nodeID, stmt.Body, &nestedPrevious, depth+1)
	return nodeID
}

func (b *semanticGraphBuilder) addSetStatement(parent SemanticNodeID, stmt *SetStatement, ordinal int, depth int) SemanticNodeID {
	target := semanticIdentifierName(stmt.Target)
	value := semanticExpressionLabel(stmt.Value)
	return b.addStatementNode(parent, semanticStatementNode{
		kind:       SemanticNodeAction,
		ordinal:    ordinal,
		statement:  string(TokenSet),
		labelValue: target,
		position:   stmt.Position,
		effect:     SemanticEffectWrite,
		parts:      []string{target, value, fmt.Sprintf(semanticDepthPartFormat, depth)},
		properties: map[string]any{
			"statement": string(TokenSet),
			"target":    target,
			"value":     value,
		},
	})
}

func (b *semanticGraphBuilder) addNotifyStatement(parent SemanticNodeID, stmt *NotifyStatement, ordinal int, depth int) SemanticNodeID {
	target := semanticIdentifierName(stmt.Target)
	value := semanticExpressionLabel(stmt.Value)
	return b.addStatementNode(parent, semanticStatementNode{
		kind:       SemanticNodeAction,
		ordinal:    ordinal,
		statement:  string(TokenNotify),
		labelValue: target,
		position:   stmt.Position,
		effect:     SemanticEffectNotify,
		parts:      []string{target, value, fmt.Sprintf(semanticDepthPartFormat, depth)},
		properties: map[string]any{
			"statement": string(TokenNotify),
			"target":    target,
			"value":     value,
		},
	})
}

func (b *semanticGraphBuilder) addAgentStatement(parent SemanticNodeID, stmt *AgentStatement, ordinal int, depth int) SemanticNodeID {
	name := semanticIdentifierName(stmt.Name)
	input := semanticExpressionLabel(stmt.Input)
	properties := map[string]any{
		"statement": string(TokenAgent),
		"name":      name,
	}
	if input != "" {
		properties["input"] = input
	}
	return b.addStatementNode(parent, semanticStatementNode{
		kind:       SemanticNodeDelegate,
		ordinal:    ordinal,
		statement:  string(TokenAgent),
		labelValue: name,
		position:   stmt.Position,
		effect:     SemanticEffectDelegate,
		parts:      []string{name, input, fmt.Sprintf(semanticDepthPartFormat, depth)},
		properties: properties,
	})
}

func (b *semanticGraphBuilder) addCallStatement(parent SemanticNodeID, stmt *CallStatement, ordinal int, depth int) SemanticNodeID { // CLSF-54
	tool := semanticIdentifierName(stmt.Tool)
	alias := semanticIdentifierName(stmt.Alias)
	properties := map[string]any{"statement": string(TokenCall), "tool": tool}
	if alias != "" {
		properties["alias"] = alias
	}
	return b.addStatementNode(parent, semanticStatementNode{
		kind:       SemanticNodeCall,
		ordinal:    ordinal,
		statement:  string(TokenCall),
		labelValue: tool,
		position:   stmt.Position,
		effect:     SemanticEffectDelegate,
		parts:      []string{tool, alias, fmt.Sprintf(semanticDepthPartFormat, depth)},
		properties: properties,
	})
}

func (b *semanticGraphBuilder) addApproveStatement(parent SemanticNodeID, stmt *ApproveStatement, ordinal int, depth int) SemanticNodeID { // CLSF-54
	stage := semanticIdentifierName(stmt.Stage)
	role := semanticIdentifierName(stmt.Role)
	properties := map[string]any{"statement": string(TokenApprove), "stage": stage}
	if role != "" {
		properties["role"] = role
	}
	return b.addStatementNode(parent, semanticStatementNode{
		kind:       SemanticNodeApprove,
		ordinal:    ordinal,
		statement:  string(TokenApprove),
		labelValue: stage,
		position:   stmt.Position,
		effect:     SemanticEffectWrite,
		parts:      []string{stage, role, fmt.Sprintf(semanticDepthPartFormat, depth)},
		properties: properties,
	})
}

type semanticStatementNode struct {
	kind       SemanticNodeKind
	ordinal    int
	statement  string
	labelValue string
	position   Position
	effect     SemanticEffectKind
	parts      []string
	properties map[string]any
}

func (b *semanticGraphBuilder) addStatementNode(parent SemanticNodeID, spec semanticStatementNode) SemanticNodeID {
	parts := append([]string{spec.statement}, spec.parts...)
	nodeID := b.nodeID(spec.kind, spec.ordinal, parts...)
	node := NewWorkflowSemanticNode(nodeID, spec.kind, SemanticSourceDSL)
	node.Label = strings.TrimSpace(spec.statement + " " + spec.labelValue)
	node.Position = spec.position
	node.Effect = spec.effect
	node.Properties = spec.properties
	b.graph.AddNode(node)
	b.addEdge(parent, nodeID, SemanticEdgeContains, nil)
	return nodeID
}

func (b *semanticGraphBuilder) nodeID(kind SemanticNodeKind, ordinal int, parts ...string) SemanticNodeID {
	return b.nodeIDFrom(SemanticSourceDSL, kind, ordinal, parts...)
}

func (b *semanticGraphBuilder) nodeIDFrom(source SemanticSourceKind, kind SemanticNodeKind, ordinal int, parts ...string) SemanticNodeID {
	return NewSemanticNodeID(SemanticIDInput{
		Kind:    kind,
		Source:  source,
		Scope:   b.scope,
		Ordinal: ordinal,
		Parts:   parts,
	})
}

func (b *semanticGraphBuilder) addEdge(from SemanticNodeID, to SemanticNodeID, kind SemanticEdgeKind, properties map[string]any) {
	b.graph.AddEdge(WorkflowSemanticEdge{
		From:       from,
		To:         to,
		Kind:       kind,
		Properties: properties,
	})
}

func semanticIdentifierName(identifier *IdentifierExpr) string {
	if identifier == nil {
		return ""
	}
	return identifier.Name
}

func semanticExpressionLabel(expr Expression) string {
	switch typed := expr.(type) {
	case nil:
		return ""
	case *IdentifierExpr:
		return typed.Name
	case *LiteralExpr:
		return semanticLiteralLabel(typed.Value)
	case *ArrayLiteralExpr:
		return semanticArrayLabel(typed.Elements)
	case *ObjectLiteralExpr:
		return semanticObjectLabel(typed.Fields)
	case *ComparisonExpr:
		return strings.TrimSpace(fmt.Sprintf("%s %s %s",
			semanticExpressionLabel(typed.Left),
			string(typed.Operator),
			semanticExpressionLabel(typed.Right),
		))
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func semanticArrayLabel(elements []Expression) string {
	parts := make([]string, 0, len(elements))
	for _, element := range elements {
		parts = append(parts, semanticExpressionLabel(element))
	}
	return "[" + strings.Join(parts, semanticJoinSeparator) + "]"
}

func semanticObjectLabel(fields []ObjectField) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, field.Key+": "+semanticExpressionLabel(field.Value))
	}
	return "{" + strings.Join(parts, semanticJoinSeparator) + "}"
}

func semanticLiteralLabel(value any) string {
	if value == nil {
		return semanticNullLabel
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}

func cartaWorkflowName(carta *CartaSummary) string {
	if carta == nil {
		return ""
	}
	return carta.Name
}

func semanticPermitRateLabel(rate *CartaRate) string {
	if rate == nil {
		return ""
	}
	return fmt.Sprintf("%d/%s", rate.Value, rate.Unit)
}

func semanticPermitApprovalLabel(approval *CartaApprovalConfig) string {
	if approval == nil {
		return ""
	}
	return approval.Mode
}
