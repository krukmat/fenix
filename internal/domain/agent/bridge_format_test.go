package agent

import (
	"errors"
	"testing"
)

func TestBridgeWorkflowValidateAcceptsMinimalSupportedFormat(t *testing.T) {
	t.Parallel()

	wf := BridgeWorkflow{
		Name: "qualify_lead_bridge",
		Trigger: BridgeTrigger{
			Event: "lead.created",
		},
		Steps: []BridgeStep{
			{
				ID: "step_1",
				Action: BridgeAction{
					Verb:   BridgeVerbAgent,
					Target: "evaluate_intent",
				},
			},
			{
				ID: "step_2",
				Condition: &BridgeCondition{
					Left:     "lead.score",
					Operator: BridgeOpGTE,
					Right:    0.8,
				},
				Action: BridgeAction{
					Verb:   BridgeVerbSet,
					Target: "lead.status",
					Args: map[string]any{
						"value": "qualified",
					},
				},
			},
		},
	}

	if err := wf.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestBridgeWorkflowValidateRejectsMissingName(t *testing.T) {
	t.Parallel()

	wf := BridgeWorkflow{
		Trigger: BridgeTrigger{Event: "lead.created"},
		Steps: []BridgeStep{{
			ID: "step_1",
			Action: BridgeAction{
				Verb:   BridgeVerbSet,
				Target: "lead.status",
			},
		}},
	}

	err := wf.Validate()
	if !errors.Is(err, ErrBridgeWorkflowInvalid) {
		t.Fatalf("expected ErrBridgeWorkflowInvalid, got %v", err)
	}
}

func TestBridgeWorkflowValidateRejectsEmptySteps(t *testing.T) {
	t.Parallel()

	wf := BridgeWorkflow{
		Name:    "empty_steps",
		Trigger: BridgeTrigger{Event: "lead.created"},
	}

	err := wf.Validate()
	if !errors.Is(err, ErrBridgeWorkflowInvalid) {
		t.Fatalf("expected ErrBridgeWorkflowInvalid, got %v", err)
	}
}

func TestBridgeWorkflowValidateRejectsDuplicateStepIDs(t *testing.T) {
	t.Parallel()

	wf := BridgeWorkflow{
		Name:    "dup_ids",
		Trigger: BridgeTrigger{Event: "lead.created"},
		Steps: []BridgeStep{
			{
				ID: "step_1",
				Action: BridgeAction{
					Verb:   BridgeVerbSet,
					Target: "lead.status",
				},
			},
			{
				ID: "step_1",
				Action: BridgeAction{
					Verb:   BridgeVerbNotify,
					Target: "salesperson",
				},
			},
		},
	}

	err := wf.Validate()
	if !errors.Is(err, ErrBridgeWorkflowInvalid) {
		t.Fatalf("expected ErrBridgeWorkflowInvalid, got %v", err)
	}
}

func TestBridgeStepValidateRejectsUnsupportedVerb(t *testing.T) {
	t.Parallel()

	step := BridgeStep{
		ID: "step_1",
		Action: BridgeAction{
			Verb:   "WAIT",
			Target: "48h",
		},
	}

	err := step.Validate()
	if !errors.Is(err, ErrBridgeStepInvalid) {
		t.Fatalf("expected ErrBridgeStepInvalid, got %v", err)
	}
}

func TestBridgeConditionValidateRejectsUnknownOperator(t *testing.T) {
	t.Parallel()

	condition := BridgeCondition{
		Left:     "lead.score",
		Operator: "MATCHES",
		Right:    1,
	}

	err := condition.Validate()
	if !errors.Is(err, ErrBridgeConditionInvalid) {
		t.Fatalf("expected ErrBridgeConditionInvalid, got %v", err)
	}
}
