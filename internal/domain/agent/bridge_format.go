package agent

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrBridgeWorkflowInvalid = errors.New("bridge workflow is invalid")
	ErrBridgeStepInvalid     = errors.New("bridge step is invalid")
	ErrBridgeConditionInvalid = errors.New("bridge condition is invalid")
)

const (
	BridgeVerbSet    = "SET"
	BridgeVerbNotify = "NOTIFY"
	BridgeVerbAgent  = "AGENT"
)

const (
	BridgeOpEQ  = "=="
	BridgeOpNEQ = "!="
	BridgeOpGT  = ">"
	BridgeOpLT  = "<"
	BridgeOpGTE = ">="
	BridgeOpLTE = "<="
	BridgeOpIn  = "IN"
)

// BridgeWorkflow is the transitional declarative format introduced in F3.1.
//
// It is intentionally smaller than the final DSL: one trigger, sequential
// steps, and a small conditional envelope per step. WAIT and DISPATCH remain
// explicitly out of scope for this bridge format.
type BridgeWorkflow struct {
	Name    string       `json:"name"`
	Trigger BridgeTrigger `json:"trigger"`
	Steps   []BridgeStep `json:"steps"`
}

type BridgeTrigger struct {
	Event string `json:"event"`
}

type BridgeStep struct {
	ID        string           `json:"id"`
	Name      string           `json:"name,omitempty"`
	Condition *BridgeCondition `json:"condition,omitempty"`
	Action    BridgeAction     `json:"action"`
}

type BridgeCondition struct {
	Left     string   `json:"left"`
	Operator string   `json:"operator"`
	Right    any      `json:"right"`
}

type BridgeAction struct {
	Verb    string         `json:"verb"`
	Target  string         `json:"target,omitempty"`
	Args    map[string]any `json:"args,omitempty"`
}

func (wf BridgeWorkflow) Validate() error {
	if strings.TrimSpace(wf.Name) == "" {
		return invalidBridgeWorkflow("name is required", nil)
	}
	if err := wf.Trigger.Validate(); err != nil {
		return invalidBridgeWorkflow("trigger is invalid", err)
	}
	if len(wf.Steps) == 0 {
		return invalidBridgeWorkflow("at least one step is required", nil)
	}

	seenIDs := make(map[string]struct{}, len(wf.Steps))
	for i, step := range wf.Steps {
		if err := step.Validate(); err != nil {
			return invalidBridgeWorkflow(fmt.Sprintf("step %d is invalid", i), err)
		}
		if step.ID == "" {
			continue
		}
		if _, exists := seenIDs[step.ID]; exists {
			return invalidBridgeWorkflow(fmt.Sprintf("duplicate step id: %s", step.ID), nil)
		}
		seenIDs[step.ID] = struct{}{}
	}

	return nil
}

func (t BridgeTrigger) Validate() error {
	if strings.TrimSpace(t.Event) == "" {
		return invalidBridgeWorkflow("trigger event is required", nil)
	}
	return nil
}

func (s BridgeStep) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return invalidBridgeStep("id is required", nil)
	}
	if s.Condition != nil {
		if err := s.Condition.Validate(); err != nil {
			return invalidBridgeStep("condition is invalid", err)
		}
	}
	if err := s.Action.Validate(); err != nil {
		return invalidBridgeStep("action is invalid", err)
	}
	return nil
}

func (c BridgeCondition) Validate() error {
	if strings.TrimSpace(c.Left) == "" {
		return invalidBridgeCondition("left operand is required", nil)
	}
	if c.Right == nil {
		return invalidBridgeCondition("right operand is required", nil)
	}

	switch strings.TrimSpace(strings.ToUpper(c.Operator)) {
	case BridgeOpEQ, BridgeOpNEQ, BridgeOpGT, BridgeOpLT, BridgeOpGTE, BridgeOpLTE, BridgeOpIn:
		return nil
	default:
		return invalidBridgeCondition("operator is invalid", nil)
	}
}

func (a BridgeAction) Validate() error {
	verb := strings.TrimSpace(strings.ToUpper(a.Verb))
	if verb == "" {
		return invalidBridgeStep("verb is required", nil)
	}

	switch verb {
	case BridgeVerbSet, BridgeVerbNotify, BridgeVerbAgent:
	default:
		return invalidBridgeStep("verb is not supported by bridge format", nil)
	}

	if strings.TrimSpace(a.Target) == "" {
		return invalidBridgeStep("target is required", nil)
	}

	return nil
}

func invalidBridgeWorkflow(reason string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrBridgeWorkflowInvalid, reason)
	}
	return fmt.Errorf("%w: %s: %w", ErrBridgeWorkflowInvalid, reason, err)
}

func invalidBridgeStep(reason string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrBridgeStepInvalid, reason)
	}
	return fmt.Errorf("%w: %s: %w", ErrBridgeStepInvalid, reason, err)
}

func invalidBridgeCondition(reason string, err error) error {
	if err == nil {
		return fmt.Errorf("%w: %s", ErrBridgeConditionInvalid, reason)
	}
	return fmt.Errorf("%w: %s: %w", ErrBridgeConditionInvalid, reason, err)
}
