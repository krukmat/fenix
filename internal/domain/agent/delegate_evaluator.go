package agent

import (
	"fmt"
)

type DelegateEvaluator struct {
	evaluator *ExpressionEvaluator
}

type DelegateDecision struct {
	Matched  bool
	Delegate *CartaDelegate
}

func NewDelegateEvaluator() *DelegateEvaluator {
	return &DelegateEvaluator{evaluator: NewExpressionEvaluator()}
}

func (e *DelegateEvaluator) EvaluateDelegate(delegates []CartaDelegate, evalCtx map[string]any) (*DelegateDecision, error) {
	if len(delegates) == 0 {
		return &DelegateDecision{}, nil
	}
	if e == nil || e.evaluator == nil {
		return nil, fmt.Errorf("delegate evaluator requires expression evaluator")
	}

	for i := range delegates {
		decision, err := e.evaluateSingleDelegate(delegates[i], evalCtx)
		if err != nil {
			return nil, err
		}
		if decision != nil {
			return decision, nil
		}
	}

	return &DelegateDecision{}, nil
}

func (e *DelegateEvaluator) evaluateSingleDelegate(delegate CartaDelegate, evalCtx map[string]any) (*DelegateDecision, error) {
	if delegate.When == "" {
		return &DelegateDecision{Matched: true, Delegate: &delegate}, nil
	}

	expr, err := parseDelegateCondition(delegate.When)
	if err != nil {
		return nil, err
	}
	matched, err := e.evaluator.EvaluateCondition(expr, evalCtx)
	if err != nil {
		return nil, err
	}
	if matched {
		return &DelegateDecision{Matched: true, Delegate: &delegate}, nil
	}
	return nil, nil
}

func parseDelegateCondition(condition string) (Expression, error) {
	source := fmt.Sprintf("WORKFLOW delegate_condition\nON case.created\nIF %s:\n  WAIT 1 minutes\n", condition)
	program, err := ParseDSL(source)
	if err != nil {
		return nil, err
	}
	if program == nil || program.Workflow == nil || len(program.Workflow.Body) == 0 {
		return nil, fmt.Errorf("delegate condition did not produce an IF statement")
	}
	stmt, ok := program.Workflow.Body[0].(*IfStatement)
	if !ok || stmt.Condition == nil {
		return nil, fmt.Errorf("delegate condition did not parse as IF condition")
	}
	return stmt.Condition, nil
}
