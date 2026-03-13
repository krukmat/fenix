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
	waits  []stubWaitCall
}

type stubWaitCall struct {
	Amount    int64
	Unit      string
	NextIndex int
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

func (s *stubRuntimeExecutor) ExecuteWait(_ context.Context, stmt *WaitStatement, nextStatementIndex int, _ map[string]any) (RuntimeExecutionResult, error) {
	s.waits = append(s.waits, stubWaitCall{
		Amount:    stmt.Amount,
		Unit:      stmt.Unit,
		NextIndex: nextStatementIndex,
	})
	if s.err != nil {
		return RuntimeExecutionResult{}, s.err
	}
	return RuntimeExecutionResult{
		Output: s.output,
		Status: s.status,
		Stop:   s.stop,
	}, nil
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

func TestDSLRuntimeExecuteProgramSchedulesWaitAndStops(t *testing.T) {
	t.Parallel()

	runtime := NewDSLRuntime()
	executor := &stubRuntimeExecutor{
		output: map[string]any{"action": "waiting", "resume_step_index": 1},
		status: StatusAccepted,
		stop:   true,
	}
	program := &Program{
		Workflow: &WorkflowDecl{
			Name: "wait_case",
			Body: []Statement{
				&WaitStatement{Amount: 48, Unit: "hours"},
				&NotifyStatement{
					Target: &IdentifierExpr{Name: "contact"},
					Value:  &LiteralExpr{Value: "done"},
				},
			},
		},
	}

	result, err := runtime.ExecuteProgram(context.Background(), program, nil, executor)
	if err != nil {
		t.Fatalf("ExecuteProgram() error = %v", err)
	}
	if len(result.Statements) != 1 {
		t.Fatalf("len(statements) = %d, want 1", len(result.Statements))
	}
	if result.Statements[0].Type != "WAIT" || result.Statements[0].Status != StatusAccepted {
		t.Fatalf("unexpected WAIT result = %#v", result.Statements[0])
	}
	if len(executor.waits) != 1 {
		t.Fatalf("len(waits) = %d, want 1", len(executor.waits))
	}
	if executor.waits[0].NextIndex != 1 {
		t.Fatalf("nextIndex = %d, want 1", executor.waits[0].NextIndex)
	}
	if len(executor.ops) != 0 {
		t.Fatalf("expected no mapped ops after WAIT, got %d", len(executor.ops))
	}
}

func TestDSLRuntimeExecuteProgramDispatchesAndStops(t *testing.T) {
	t.Parallel()

	runtime := NewDSLRuntime()
	executor := &stubRuntimeExecutor{
		output: map[string]any{"dispatch_result": dispatchResultDelegated},
		status: StatusDelegated,
		stop:   true,
	}
	program := &Program{
		Workflow: &WorkflowDecl{
			Name: "delegate_case",
			Body: []Statement{
				&DispatchStatement{
					Target:  &IdentifierExpr{Name: "support_agent"},
					Payload: &ObjectLiteralExpr{},
				},
				&NotifyStatement{
					Target: &IdentifierExpr{Name: "contact"},
					Value:  &LiteralExpr{Value: "done"},
				},
			},
		},
	}

	result, err := runtime.ExecuteProgram(context.Background(), program, nil, executor)
	if err != nil {
		t.Fatalf("ExecuteProgram() error = %v", err)
	}
	if len(result.Statements) != 1 {
		t.Fatalf("len(statements) = %d, want 1", len(result.Statements))
	}
	if result.Statements[0].Type != "DISPATCH" || result.Statements[0].Status != StatusDelegated {
		t.Fatalf("unexpected dispatch result = %#v", result.Statements[0])
	}
	if len(executor.ops) != 1 || executor.ops[0].Kind != RuntimeOperationDispatch {
		t.Fatalf("unexpected operations = %#v", executor.ops)
	}
}

func TestDSLRuntimeExecuteProgramSurfacesSignal(t *testing.T) {
	t.Parallel()

	runtime := NewDSLRuntime()
	executor := &stubRuntimeExecutor{
		output: map[string]any{"signal_id": "signal-1"},
	}
	program := &Program{
		Workflow: &WorkflowDecl{
			Name: "surface_case",
			Body: []Statement{
				&SurfaceStatement{
					Entity:  &IdentifierExpr{Name: "case"},
					View:    &IdentifierExpr{Name: "salesperson.view"},
					Payload: &ObjectLiteralExpr{},
				},
			},
		},
	}

	result, err := runtime.ExecuteProgram(context.Background(), program, map[string]any{
		"case": map[string]any{"id": "case-1"},
	}, executor)
	if err != nil {
		t.Fatalf("ExecuteProgram() error = %v", err)
	}
	if len(result.Statements) != 1 {
		t.Fatalf("len(statements) = %d, want 1", len(result.Statements))
	}
	if result.Statements[0].Type != "SURFACE" || result.Statements[0].Target != "salesperson.view" {
		t.Fatalf("unexpected SURFACE result = %#v", result.Statements[0])
	}
	if len(executor.ops) != 1 || executor.ops[0].Kind != RuntimeOperationSurface {
		t.Fatalf("unexpected operations = %#v", executor.ops)
	}
}
