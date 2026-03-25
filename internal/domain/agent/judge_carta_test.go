package agent

import (
	"context"
	"testing"

	workflowdomain "github.com/matiasleandrokruk/fenix/internal/domain/workflow"
)

func TestJudgeCartaScenarioA_PassesForCoveredCartaWorkflow(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  PERMIT update_case`

	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-carta-scenario-a",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nSET case.status = \"resolved\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Passed {
		t.Fatalf("Passed = false, want true; violations = %#v", result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("Violations = %#v, want none", result.Violations)
	}
}

func TestJudgeCartaScenarioB_FlagsToolNotPermitted(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := `CARTA resolve_support_case
AGENT search_knowledge
  GROUNDS
    min_sources: 2
  PERMIT send_reply`

	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-carta-scenario-b",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	found := false
	for _, violation := range result.Violations {
		if violation.Code == "tool_not_permitted" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Violations = %#v, want tool_not_permitted", result.Violations)
	}
}

func TestJudgeCartaScenarioC_FlagsBehaviorWithoutPermitOrDelegate(t *testing.T) {
	t.Parallel()

	violations := RunCartaCoverageChecks(
		&CartaSummary{Permits: []CartaPermit{{Tool: "send_reply"}}},
		&SpecSummary{Behaviors: []SpecBehavior{{Name: "escalate_unresolved", Line: 4}}},
	)
	if len(violations) != 1 {
		t.Fatalf("len(Violations) = %d, want 1", len(violations))
	}
	if violations[0].Code != "behavior_no_permit_or_delegate" {
		t.Fatalf("Code = %q, want behavior_no_permit_or_delegate", violations[0].Code)
	}
}

func TestJudgeCartaBackwardCompat_FreeFormatSpecStillPasses(t *testing.T) {
	t.Parallel()

	judge := NewJudge()
	spec := "BEHAVIOR notify_salesperson"

	result, err := judge.Verify(context.Background(), &workflowdomain.Workflow{
		ID:         "wf-free-format-compat",
		DSLSource:  "WORKFLOW resolve_support_case\nON case.created\nNOTIFY salesperson WITH \"review\"",
		SpecSource: &spec,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Passed {
		t.Fatalf("Passed = false, want true; violations = %#v", result.Violations)
	}
}
