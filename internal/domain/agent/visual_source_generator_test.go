package agent

import (
	"strings"
	"testing"
)

func TestGenerateDSLSourceFromVisualAuthoringGraphCoreStatements(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("sales_followup")
	graph.AddNode(NewVisualAuthoringNode("workflow", SemanticNodeWorkflow, "sales_followup", WorkflowVisualPosition{X: 0, Y: 0}))
	trigger := NewVisualAuthoringNode("trigger", SemanticNodeTrigger, "deal.updated", WorkflowVisualPosition{X: 260, Y: 0})
	trigger.Data.Event = "deal.updated"
	graph.AddNode(trigger)
	set := NewVisualAuthoringNode("set-status", SemanticNodeAction, "set status", WorkflowVisualPosition{X: 0, Y: 160})
	set.Data.Target = "deal.status"
	set.Data.Value = "qualified"
	graph.AddNode(set)
	notify := NewVisualAuthoringNode("notify-owner", SemanticNodeAction, "notify owner", WorkflowVisualPosition{X: 260, Y: 160})
	notify.Data.Action = "notify"
	notify.Data.Target = "owner"
	notify.Data.Value = "review deal"
	graph.AddNode(notify)

	result, err := GenerateDSLSourceFromVisualAuthoringGraph(graph)
	if err != nil {
		t.Fatalf("GenerateDSLSourceFromVisualAuthoringGraph() error = %v", err)
	}

	want := "WORKFLOW sales_followup\nON deal.updated\nSET deal.status = \"qualified\"\nNOTIFY owner WITH \"review deal\"\n"
	if result.DSLSource != want {
		t.Fatalf("DSLSource = %q, want %q", result.DSLSource, want)
	}
	if _, err := ParseAndValidateDSL(result.DSLSource); err != nil {
		t.Fatalf("generated DSL should parse and validate: %v\n%s", err, result.DSLSource)
	}
}

func TestGenerateDSLSourceFromVisualAuthoringGraphDecisionBranch(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("route_deal")
	graph.AddNode(NewVisualAuthoringNode("workflow", SemanticNodeWorkflow, "route_deal", WorkflowVisualPosition{X: 0, Y: 0}))
	trigger := NewVisualAuthoringNode("trigger", SemanticNodeTrigger, "deal.updated", WorkflowVisualPosition{X: 260, Y: 0})
	trigger.Data.Event = "deal.updated"
	graph.AddNode(trigger)
	decision := NewVisualAuthoringNode("decision", SemanticNodeDecision, "deal.value > 1000", WorkflowVisualPosition{X: 0, Y: 160})
	decision.Data.Expression = "deal.value > 1000"
	graph.AddNode(decision)
	action := NewVisualAuthoringNode("set-priority", SemanticNodeAction, "set priority", WorkflowVisualPosition{X: 0, Y: 320})
	action.Data.Target = "deal.priority"
	action.Data.Value = "high"
	graph.AddNode(action)
	graph.AddEdge(NewVisualAuthoringEdge("decision-action", decision.ID, action.ID, SemanticEdgeBranches))

	result, err := GenerateDSLSourceFromVisualAuthoringGraph(graph)
	if err != nil {
		t.Fatalf("GenerateDSLSourceFromVisualAuthoringGraph() error = %v", err)
	}

	if !strings.Contains(result.DSLSource, "IF deal.value > 1000:\n  SET deal.priority = \"high\"") {
		t.Fatalf("DSLSource = %q, want indented decision branch", result.DSLSource)
	}
	if _, err := ParseAndValidateDSL(result.DSLSource); err != nil {
		t.Fatalf("generated DSL should parse and validate: %v\n%s", err, result.DSLSource)
	}
}

func TestGenerateDSLSourceFromVisualAuthoringGraphAgentPayload(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("agent_review")
	graph.AddNode(NewVisualAuthoringNode("workflow", SemanticNodeWorkflow, "agent_review", WorkflowVisualPosition{}))
	trigger := NewVisualAuthoringNode("trigger", SemanticNodeTrigger, "deal.updated", WorkflowVisualPosition{X: 260})
	trigger.Data.Event = "deal.updated"
	graph.AddNode(trigger)
	action := NewVisualAuthoringNode("agent", SemanticNodeAction, "agent review", WorkflowVisualPosition{Y: 160})
	action.Data.Action = "agent"
	action.Data.AgentName = "review_agent"
	action.Data.Payload = map[string]any{"entity": "deal", "mode": "fast"}
	graph.AddNode(action)

	result, err := GenerateDSLSourceFromVisualAuthoringGraph(graph)
	if err != nil {
		t.Fatalf("GenerateDSLSourceFromVisualAuthoringGraph() error = %v", err)
	}

	if !strings.Contains(result.DSLSource, `AGENT review_agent WITH {"entity":"deal","mode":"fast"}`) {
		t.Fatalf("DSLSource = %q, want AGENT payload", result.DSLSource)
	}
	if _, err := ParseAndValidateDSL(result.DSLSource); err != nil {
		t.Fatalf("generated DSL should parse and validate: %v\n%s", err, result.DSLSource)
	}
}

func TestGenerateDSLSourceFromVisualAuthoringGraphRejectsInvalidVisualGraph(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("bad")
	graph.AddNode(NewVisualAuthoringNode("workflow", SemanticNodeWorkflow, "bad", WorkflowVisualPosition{}))

	_, err := GenerateDSLSourceFromVisualAuthoringGraph(graph)
	if err == nil {
		t.Fatal("expected error for graph without trigger")
	}
}

func TestGenerateDSLSourceFromVisualAuthoringGraphRejectsEmptyDecisionBranch(t *testing.T) {
	t.Parallel()

	graph := NewVisualAuthoringGraph("bad_decision")
	graph.AddNode(NewVisualAuthoringNode("workflow", SemanticNodeWorkflow, "bad_decision", WorkflowVisualPosition{}))
	graph.AddNode(NewVisualAuthoringNode("trigger", SemanticNodeTrigger, "deal.updated", WorkflowVisualPosition{X: 260}))
	graph.AddNode(NewVisualAuthoringNode("decision", SemanticNodeDecision, "deal.value > 1000", WorkflowVisualPosition{Y: 160}))

	_, err := GenerateDSLSourceFromVisualAuthoringGraph(graph)
	if err == nil {
		t.Fatal("expected error for decision without branch child")
	}
}

func TestGenerateSourcesFromVisualAuthoringGraphCartaGovernance(t *testing.T) {
	t.Parallel()

	graph := visualSourceBaseGraph("governed_followup")
	agent := NewVisualAuthoringNode("agent", SemanticNodeAction, "agent", WorkflowVisualPosition{Y: 160})
	agent.Data.Action = "agent"
	agent.Data.AgentName = "sales_agent"
	graph.AddNode(agent)
	grounds := NewVisualAuthoringNode("grounds", SemanticNodeGrounds, "grounds", WorkflowVisualPosition{Y: 320})
	grounds.Data.Grounds = map[string]any{"min_sources": 2, "min_confidence": "medium", "max_staleness": "30 days", "types": []string{"case", "kb_article"}}
	graph.AddNode(grounds)
	permit := NewVisualAuthoringNode("permit", SemanticNodePermit, "send_reply", WorkflowVisualPosition{X: 260, Y: 320})
	permit.Data.Permit = "send_reply"
	graph.AddNode(permit)
	delegate := NewVisualAuthoringNode("delegate", SemanticNodeDelegate, "Enterprise review", WorkflowVisualPosition{X: 520, Y: 320})
	delegate.Data.Expression = `case.tier == "enterprise"`
	delegate.Data.Value = "Enterprise review"
	graph.AddNode(delegate)
	invariant := NewVisualAuthoringNode("invariant", SemanticNodeInvariant, "send_pii", WorkflowVisualPosition{X: 780, Y: 320})
	invariant.Data.Invariant = "send_pii"
	graph.AddNode(invariant)
	budget := NewVisualAuthoringNode("budget", SemanticNodeBudget, "budget", WorkflowVisualPosition{Y: 480})
	budget.Data.Budget = map[string]any{"daily_tokens": 50000, "daily_cost_usd": 5.0, "executions_per_day": 100, "on_exceed": "pause"}
	graph.AddNode(budget)

	result, err := GenerateSourcesFromVisualAuthoringGraph(graph)
	if err != nil {
		t.Fatalf("GenerateSourcesFromVisualAuthoringGraph() error = %v", err)
	}
	if result.SpecSource == "" {
		t.Fatal("SpecSource is empty, want Carta source")
	}
	if !strings.Contains(result.SpecSource, "CARTA governed_followup\nBUDGET\n") {
		t.Fatalf("SpecSource = %q, want CARTA + BUDGET", result.SpecSource)
	}
	if !strings.Contains(result.SpecSource, "AGENT sales_agent\n  GROUNDS\n") {
		t.Fatalf("SpecSource = %q, want AGENT sales_agent with GROUNDS", result.SpecSource)
	}
	if !strings.Contains(result.SpecSource, "  PERMIT send_reply\n") {
		t.Fatalf("SpecSource = %q, want PERMIT", result.SpecSource)
	}
	if !strings.Contains(result.SpecSource, "  DELEGATE TO HUMAN\n    when: case.tier == \"enterprise\"\n    reason: \"Enterprise review\"") {
		t.Fatalf("SpecSource = %q, want DELEGATE clauses", result.SpecSource)
	}
	if !strings.Contains(result.SpecSource, "  INVARIANT\n    never: \"send_pii\"") {
		t.Fatalf("SpecSource = %q, want INVARIANT", result.SpecSource)
	}
	if _, err := ParseCarta(result.SpecSource); err != nil {
		t.Fatalf("generated Carta should parse: %v\n%s", err, result.SpecSource)
	}
}

func TestGenerateSourcesFromVisualAuthoringGraphNoGovernanceOmitsSpecSource(t *testing.T) {
	t.Parallel()

	result, err := GenerateSourcesFromVisualAuthoringGraph(visualSourceBaseGraph("plain_followup"))
	if err != nil {
		t.Fatalf("GenerateSourcesFromVisualAuthoringGraph() error = %v", err)
	}
	if result.SpecSource != "" {
		t.Fatalf("SpecSource = %q, want empty when no governance nodes exist", result.SpecSource)
	}
}

func TestGenerateSourcesFromVisualAuthoringGraphRejectsBudgetOnlyCarta(t *testing.T) {
	t.Parallel()

	graph := visualSourceBaseGraph("budget_only")
	budget := NewVisualAuthoringNode("budget", SemanticNodeBudget, "budget", WorkflowVisualPosition{Y: 160})
	budget.Data.Budget = map[string]any{"daily_tokens": 50000}
	graph.AddNode(budget)

	_, err := GenerateSourcesFromVisualAuthoringGraph(graph)
	if err == nil {
		t.Fatal("expected error for Carta source without agent governance directive")
	}
}

func visualSourceBaseGraph(name string) *VisualAuthoringGraph {
	graph := NewVisualAuthoringGraph(name)
	graph.AddNode(NewVisualAuthoringNode("workflow", SemanticNodeWorkflow, name, WorkflowVisualPosition{}))
	trigger := NewVisualAuthoringNode("trigger", SemanticNodeTrigger, "deal.updated", WorkflowVisualPosition{X: 260})
	trigger.Data.Event = "deal.updated"
	graph.AddNode(trigger)
	action := NewVisualAuthoringNode("set-status", SemanticNodeAction, "set status", WorkflowVisualPosition{Y: 160})
	action.Data.Target = "deal.status"
	action.Data.Value = "open"
	graph.AddNode(action)
	return graph
}
