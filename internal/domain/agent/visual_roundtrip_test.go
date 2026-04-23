package agent

import (
	"strings"
	"testing"
)

func TestVisualAuthoringRoundTripEquivalentToTextWorkflow(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("sales_followup")
	graph.AddNode(NewVisualAuthoringNode("workflow", SemanticNodeWorkflow, "sales_followup", WorkflowVisualPosition{}))
	trigger := NewVisualAuthoringNode("trigger", SemanticNodeTrigger, "deal.updated", WorkflowVisualPosition{X: 260})
	trigger.Data.Event = "deal.updated"
	graph.AddNode(trigger)
	set := NewVisualAuthoringNode("set-status", SemanticNodeAction, "set status", WorkflowVisualPosition{Y: 160})
	set.Data.Target = "deal.status"
	set.Data.Value = "qualified"
	graph.AddNode(set)
	decision := NewVisualAuthoringNode("decision", SemanticNodeDecision, "deal.value > 1000", WorkflowVisualPosition{Y: 320})
	decision.Data.Expression = "deal.value > 1000"
	graph.AddNode(decision)
	notify := NewVisualAuthoringNode("notify-owner", SemanticNodeAction, "notify owner", WorkflowVisualPosition{X: 260, Y: 480})
	notify.Data.Action = "notify"
	notify.Data.Target = "owner"
	notify.Data.Value = "review deal"
	graph.AddNode(notify)
	agent := NewVisualAuthoringNode("agent", SemanticNodeAction, "agent review", WorkflowVisualPosition{Y: 640})
	agent.Data.Action = "agent"
	agent.Data.AgentName = "sales_agent"
	agent.Data.Value = "deal.id"
	graph.AddNode(agent)
	graph.AddEdge(NewVisualAuthoringEdge("decision-notify", decision.ID, notify.ID, SemanticEdgeBranches))

	result, err := GenerateSourcesFromVisualAuthoringGraph(graph)
	if err != nil {
		t.Fatalf("GenerateSourcesFromVisualAuthoringGraph() error = %v", err)
	}
	if result.SpecSource != "" {
		t.Fatalf("SpecSource = %q, want empty for DSL-only visual workflow", result.SpecSource)
	}

	authoredDSL := `
WORKFLOW sales_followup
ON deal.updated

SET deal.status = "qualified"

IF deal.value > 1000:
  NOTIFY owner WITH "review deal"

AGENT sales_agent WITH "deal.id"
`

	assertVisualRoundTripEquivalent(t, result.DSLSource, result.SpecSource, authoredDSL, "")
}

func TestVisualAuthoringRoundTripEquivalentToTextWorkflowWithCarta(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("governed_followup")
	graph.AddNode(NewVisualAuthoringNode("workflow", SemanticNodeWorkflow, "governed_followup", WorkflowVisualPosition{}))
	trigger := NewVisualAuthoringNode("trigger", SemanticNodeTrigger, "deal.updated", WorkflowVisualPosition{X: 260})
	trigger.Data.Event = "deal.updated"
	graph.AddNode(trigger)
	set := NewVisualAuthoringNode("set-status", SemanticNodeAction, "set status", WorkflowVisualPosition{Y: 160})
	set.Data.Target = "deal.status"
	set.Data.Value = "open"
	graph.AddNode(set)
	agent := NewVisualAuthoringNode("agent", SemanticNodeAction, "agent review", WorkflowVisualPosition{Y: 320})
	agent.Data.Action = "agent"
	agent.Data.AgentName = "sales_agent"
	agent.Data.Value = "deal.id"
	graph.AddNode(agent)
	grounds := NewVisualAuthoringNode("grounds", SemanticNodeGrounds, "grounds", WorkflowVisualPosition{Y: 480})
	grounds.Data.Grounds = map[string]any{
		"min_sources":    2,
		"min_confidence": "medium",
		"max_staleness":  "30 days",
		"types":          []string{"case", "kb_article"},
	}
	graph.AddNode(grounds)
	permit := NewVisualAuthoringNode("permit", SemanticNodePermit, "send_reply", WorkflowVisualPosition{X: 260, Y: 480})
	permit.Data.Permit = "send_reply"
	graph.AddNode(permit)
	delegate := NewVisualAuthoringNode("delegate", SemanticNodeDelegate, "Enterprise review", WorkflowVisualPosition{X: 520, Y: 480})
	delegate.Data.Expression = `case.tier == "enterprise"`
	delegate.Data.Value = "Enterprise review"
	graph.AddNode(delegate)
	invariant := NewVisualAuthoringNode("invariant", SemanticNodeInvariant, "send_pii", WorkflowVisualPosition{X: 780, Y: 480})
	invariant.Data.Invariant = "send_pii"
	graph.AddNode(invariant)
	budget := NewVisualAuthoringNode("budget", SemanticNodeBudget, "budget", WorkflowVisualPosition{Y: 640})
	budget.Data.Budget = map[string]any{
		"daily_cost_usd":     5.0,
		"daily_tokens":       50000,
		"executions_per_day": 100,
		"on_exceed":          "pause",
	}
	graph.AddNode(budget)

	result, err := GenerateSourcesFromVisualAuthoringGraph(graph)
	if err != nil {
		t.Fatalf("GenerateSourcesFromVisualAuthoringGraph() error = %v", err)
	}
	if result.SpecSource == "" {
		t.Fatal("SpecSource is empty, want Carta source")
	}

	authoredDSL := `
WORKFLOW governed_followup
ON deal.updated

SET deal.status = "open"
AGENT sales_agent WITH "deal.id"
`
	authoredCarta := `
CARTA governed_followup
AGENT sales_agent
  PERMIT send_reply
  GROUNDS
    max_staleness: 30 days
    types: ["case", "kb_article"]
    min_confidence: medium
    min_sources: 2
  INVARIANT
    never: "send_pii"
  DELEGATE TO HUMAN
    reason: "Enterprise review"
    when: case.tier == "enterprise"
BUDGET
  executions_per_day: 100
  daily_cost_usd: 5
  daily_tokens: 50000
  on_exceed: pause
`

	assertVisualRoundTripEquivalent(t, result.DSLSource, result.SpecSource, authoredDSL, authoredCarta)
}

func assertVisualRoundTripEquivalent(t *testing.T, generatedDSL string, generatedCarta string, authoredDSL string, authoredCarta string) {
	t.Helper()

	generatedGraph := mustBuildGraphFromTokenizedSources(t, generatedDSL, generatedCarta)
	authoredGraph := mustBuildGraphFromTokenizedSources(t, authoredDSL, authoredCarta)

	diff := DiffWorkflowSemanticGraphs(authoredGraph, generatedGraph)
	if diff.HasSemanticChanges {
		t.Fatalf("visual round-trip changed semantics:\nnode changes: %#v\nedge changes: %#v\ngenerated DSL:\n%s\ngenerated Carta:\n%s", diff.NodeChanges, diff.EdgeChanges, generatedDSL, generatedCarta)
	}
}

func mustBuildGraphFromTokenizedSources(t *testing.T, dslSource string, cartaSource string) *WorkflowSemanticGraph {
	t.Helper()

	tokens, err := NewLexer(dslSource).Lex()
	if err != nil {
		t.Fatalf("NewLexer().Lex() error = %v\n%s", err, dslSource)
	}
	program, err := NewParser(tokens).ParseProgram()
	if err != nil {
		t.Fatalf("NewParser(tokens).ParseProgram() error = %v\n%s", err, dslSource)
	}

	if strings.TrimSpace(cartaSource) == "" {
		return BuildWorkflowSemanticGraph(program)
	}
	carta, err := ParseCarta(cartaSource)
	if err != nil {
		t.Fatalf("ParseCarta() error = %v\n%s", err, cartaSource)
	}
	return BuildWorkflowSemanticGraphWithCarta(program, carta)
}
