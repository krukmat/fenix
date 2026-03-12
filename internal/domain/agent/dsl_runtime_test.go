package agent

import (
	"context"
	"errors"
	"testing"
)

type stubRuntimeExecutor struct {
	ops    []*RuntimeOperation
	output any
	err    error
	stop   bool
	status string
}

func (s *stubRuntimeExecutor) Execute(_ context.Context, op *RuntimeOperation, _ map[string]any) (RuntimeExecutionResult, error) {
	s.ops = append(s.ops, op)
	if s.err != nil {
		return RuntimeExecutionResult{}, s.err
	}
	return RuntimeExecutionResult{
		Output: s.output,
		Status: s.status,
		Stop:   s.stop,
	}, nil
}

func TestDSLRuntimeExecuteProgramRunsMappedStatementsInOrder(t *testing.T) {
	t.Parallel()

	runtime := NewDSLRuntime()
	executor := &stubRuntimeExecutor{output: map[string]any{"result": "ok"}}
	program := &Program{
		Workflow: &WorkflowDecl{
			Name: "resolve_support_case",
			Trigger: &OnDecl{
				Event: "case.created",
			},
			Body: []Statement{
				&SetStatement{
					Target: &IdentifierExpr{Name: "case.status"},
					Value:  &LiteralExpr{Value: "resolved"},
				},
				&NotifyStatement{
					Target: &IdentifierExpr{Name: "contact"},
					Value:  &LiteralExpr{Value: "done"},
				},
			},
		},
	}

	result, err := runtime.ExecuteProgram(context.Background(), program, map[string]any{
		"case": map[string]any{"id": "case-1"},
	}, executor)
	if err != nil {
		t.Fatalf("ExecuteProgram returned error: %v", err)
	}
	if result.WorkflowName != "resolve_support_case" {
		t.Fatalf("unexpected workflow name: %s", result.WorkflowName)
	}
	if len(result.Statements) != 2 {
		t.Fatalf("statement count = %d, want 2", len(result.Statements))
	}
	if len(executor.ops) != 2 {
		t.Fatalf("executed ops = %d, want 2", len(executor.ops))
	}
	if executor.ops[0].ToolName != "update_case" {
		t.Fatalf("first op tool = %s", executor.ops[0].ToolName)
	}
	if executor.ops[1].ToolName != "send_reply" {
		t.Fatalf("second op tool = %s", executor.ops[1].ToolName)
	}
}

func TestDSLRuntimeExecuteProgramSkipsIfBodyWhenFalse(t *testing.T) {
	t.Parallel()

	runtime := NewDSLRuntime()
	executor := &stubRuntimeExecutor{}
	program := &Program{
		Workflow: &WorkflowDecl{
			Name: "resolve_support_case",
			Body: []Statement{
				&IfStatement{
					Condition: &ComparisonExpr{
						Left:     &IdentifierExpr{Name: "case.priority"},
						Operator: TokenEqual,
						Right:    &LiteralExpr{Value: "high"},
					},
					Body: []Statement{
						&NotifyStatement{
							Target: &IdentifierExpr{Name: "contact"},
							Value:  &LiteralExpr{Value: "done"},
						},
					},
				},
			},
		},
	}

	result, err := runtime.ExecuteProgram(context.Background(), program, map[string]any{
		"case": map[string]any{"priority": "low", "id": "case-1"},
	}, executor)
	if err != nil {
		t.Fatalf("ExecuteProgram returned error: %v", err)
	}
	if len(result.Statements) != 1 {
		t.Fatalf("statement count = %d, want 1", len(result.Statements))
	}
	if result.Statements[0].Status != StepStatusSkipped {
		t.Fatalf("IF status = %s, want %s", result.Statements[0].Status, StepStatusSkipped)
	}
	if len(executor.ops) != 0 {
		t.Fatalf("expected 0 ops, got %d", len(executor.ops))
	}
}

func TestDSLRuntimeExecuteProgramPropagatesExecutorError(t *testing.T) {
	t.Parallel()

	runtime := NewDSLRuntime()
	execErr := errors.New("tool failed")
	executor := &stubRuntimeExecutor{err: execErr}
	program := &Program{
		Workflow: &WorkflowDecl{
			Name: "resolve_support_case",
			Body: []Statement{
				&SetStatement{
					Target: &IdentifierExpr{Name: "case.status"},
					Value:  &LiteralExpr{Value: "resolved"},
				},
				&NotifyStatement{
					Target: &IdentifierExpr{Name: "contact"},
					Value:  &LiteralExpr{Value: "done"},
				},
			},
		},
	}

	result, err := runtime.ExecuteProgram(context.Background(), program, map[string]any{
		"case": map[string]any{"id": "case-1"},
	}, executor)
	if !errors.Is(err, execErr) {
		t.Fatalf("expected executor error, got %v", err)
	}
	if len(result.Statements) != 1 {
		t.Fatalf("statement count = %d, want 1", len(result.Statements))
	}
	if result.Statements[0].Status != StepStatusFailed {
		t.Fatalf("statement status = %s, want %s", result.Statements[0].Status, StepStatusFailed)
	}
}

func TestDSLRuntimeExecuteProgramFailsOnUnsupportedMapping(t *testing.T) {
	t.Parallel()

	runtime := NewDSLRuntime()
	program := &Program{
		Workflow: &WorkflowDecl{
			Name: "resolve_deal",
			Body: []Statement{
				&SetStatement{
					Target: &IdentifierExpr{Name: "deal.stage"},
					Value:  &LiteralExpr{Value: "won"},
				},
			},
		},
	}

	result, err := runtime.ExecuteProgram(context.Background(), program, nil, nil)
	if err == nil {
		t.Fatal("expected mapping error")
	}
	if len(result.Statements) != 1 {
		t.Fatalf("statement count = %d, want 1", len(result.Statements))
	}
	if result.Statements[0].Status != StepStatusFailed {
		t.Fatalf("statement status = %s, want %s", result.Statements[0].Status, StepStatusFailed)
	}
}

func TestDSLRuntimeMapperAndEvaluatorAccessors(t *testing.T) {
	t.Parallel()

	rt := NewDSLRuntime()
	if rt.Mapper() == nil {
		t.Fatal("Mapper() returned nil")
	}
	if rt.Evaluator() == nil {
		t.Fatal("Evaluator() returned nil")
	}
}
