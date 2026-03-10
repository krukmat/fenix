package agent

import (
	"context"
	"fmt"
)

var ErrDSLRuntimeFailed = fmt.Errorf("dsl runtime failed")

type RuntimeOperationExecutor interface {
	Execute(ctx context.Context, op *RuntimeOperation, evalCtx map[string]any) (RuntimeExecutionResult, error)
}

type RuntimeStatementTracer interface {
	StartStatementTrace(ctx context.Context, stmt Statement) (string, error)
	FinishStatementTrace(ctx context.Context, traceID string, result DSLStatementResult, stepErr error) error
}

type RuntimeExecutionResult struct {
	Output any
	Status string
	Stop   bool
}

type DSLRuntime struct {
	evaluator *ExpressionEvaluator
	mapper    *VerbMapper
}

type DSLRuntimeResult struct {
	WorkflowName string               `json:"workflow_name"`
	TriggerEvent string               `json:"trigger_event,omitempty"`
	Statements   []DSLStatementResult `json:"statements"`
}

type DSLStatementResult struct {
	Type      string            `json:"type"`
	Target    string            `json:"target,omitempty"`
	Status    string            `json:"status"`
	Position  Position          `json:"position"`
	Operation *RuntimeOperation `json:"operation,omitempty"`
	Output    any               `json:"output,omitempty"`
}

func NewDSLRuntime() *DSLRuntime {
	return &DSLRuntime{
		evaluator: NewExpressionEvaluator(),
		mapper:    NewVerbMapper(),
	}
}

func (r *DSLRuntime) Mapper() *VerbMapper {
	return r.mapper
}

func (r *DSLRuntime) Evaluator() *ExpressionEvaluator {
	return r.evaluator
}

func (r *DSLRuntime) ExecuteProgram(ctx context.Context, program *Program, evalCtx map[string]any, executor RuntimeOperationExecutor) (*DSLRuntimeResult, error) {
	if program == nil || program.Workflow == nil {
		return nil, fmt.Errorf("%w: program is required", ErrDSLRuntimeFailed)
	}

	result := &DSLRuntimeResult{
		WorkflowName: program.Workflow.Name,
	}
	if program.Workflow.Trigger != nil {
		result.TriggerEvent = program.Workflow.Trigger.Event
	}

	if _, err := r.executeStatements(ctx, program.Workflow.Body, evalCtx, executor, &result.Statements); err != nil {
		return result, err
	}
	return result, nil
}

func (r *DSLRuntime) executeStatements(ctx context.Context, statements []Statement, evalCtx map[string]any, executor RuntimeOperationExecutor, out *[]DSLStatementResult) (bool, error) {
	for _, stmt := range statements {
		stop, err := r.executeStatement(ctx, stmt, evalCtx, executor, out)
		if err != nil {
			return false, err
		}
		if stop {
			return true, nil
		}
	}
	return false, nil
}

func (r *DSLRuntime) executeStatement(ctx context.Context, stmt Statement, evalCtx map[string]any, executor RuntimeOperationExecutor, out *[]DSLStatementResult) (bool, error) {
	switch node := stmt.(type) {
	case *IfStatement:
		return r.executeIf(ctx, node, evalCtx, executor, out)
	case *SetStatement, *NotifyStatement, *AgentStatement:
		return r.executeMappedStatement(ctx, stmt, evalCtx, executor, out)
	default:
		return false, fmt.Errorf("%w: unsupported statement type", ErrDSLRuntimeFailed)
	}
}

func (r *DSLRuntime) executeIf(ctx context.Context, stmt *IfStatement, evalCtx map[string]any, executor RuntimeOperationExecutor, out *[]DSLStatementResult) (bool, error) {
	var traceID string
	if tracer, ok := executor.(RuntimeStatementTracer); ok {
		var traceErr error
		traceID, traceErr = tracer.StartStatementTrace(ctx, stmt)
		if traceErr != nil {
			return false, traceErr
		}
	}
	ok, err := r.evaluator.EvaluateCondition(stmt.Condition, evalCtx)
	if err != nil {
		result := DSLStatementResult{
			Type:     "IF",
			Status:   StepStatusFailed,
			Position: stmt.Pos(),
		}
		*out = append(*out, result)
		if tracer, isTracer := executor.(RuntimeStatementTracer); isTracer && traceID != "" {
			_ = tracer.FinishStatementTrace(ctx, traceID, result, err)
		}
		return false, err
	}
	if !ok {
		result := DSLStatementResult{
			Type:     "IF",
			Status:   StepStatusSkipped,
			Position: stmt.Pos(),
		}
		*out = append(*out, result)
		if tracer, isTracer := executor.(RuntimeStatementTracer); isTracer && traceID != "" {
			_ = tracer.FinishStatementTrace(ctx, traceID, result, nil)
		}
		return false, nil
	}
	result := DSLStatementResult{
		Type:     "IF",
		Status:   StepStatusSuccess,
		Position: stmt.Pos(),
	}
	*out = append(*out, result)
	if tracer, isTracer := executor.(RuntimeStatementTracer); isTracer && traceID != "" {
		_ = tracer.FinishStatementTrace(ctx, traceID, result, nil)
	}
	return r.executeStatements(ctx, stmt.Body, evalCtx, executor, out)
}

func (r *DSLRuntime) executeMappedStatement(ctx context.Context, stmt Statement, evalCtx map[string]any, executor RuntimeOperationExecutor, out *[]DSLStatementResult) (bool, error) {
	var traceID string
	if tracer, ok := executor.(RuntimeStatementTracer); ok {
		var traceErr error
		traceID, traceErr = tracer.StartStatementTrace(ctx, stmt)
		if traceErr != nil {
			return false, traceErr
		}
	}
	op, err := r.mapper.MapStatement(stmt, evalCtx)
	result := DSLStatementResult{
		Type:      runtimeStatementType(stmt),
		Target:    runtimeStatementTarget(stmt),
		Status:    StepStatusSuccess,
		Position:  stmt.Pos(),
		Operation: op,
	}
	if err != nil {
		result.Status = StepStatusFailed
		*out = append(*out, result)
		if tracer, ok := executor.(RuntimeStatementTracer); ok && traceID != "" {
			_ = tracer.FinishStatementTrace(ctx, traceID, result, err)
		}
		return false, err
	}
	if executor != nil {
		execResult, execErr := executor.Execute(ctx, op, evalCtx)
		result.Output = execResult.Output
		if execResult.Status != "" {
			result.Status = execResult.Status
		}
		if execErr != nil {
			result.Status = StepStatusFailed
			*out = append(*out, result)
			if tracer, ok := executor.(RuntimeStatementTracer); ok && traceID != "" {
				_ = tracer.FinishStatementTrace(ctx, traceID, result, execErr)
			}
			return false, execErr
		}
		*out = append(*out, result)
		if tracer, ok := executor.(RuntimeStatementTracer); ok && traceID != "" {
			_ = tracer.FinishStatementTrace(ctx, traceID, result, nil)
		}
		return execResult.Stop, nil
	}
	*out = append(*out, result)
	if tracer, ok := executor.(RuntimeStatementTracer); ok && traceID != "" {
		_ = tracer.FinishStatementTrace(ctx, traceID, result, nil)
	}
	return false, nil
}

func runtimeStatementType(stmt Statement) string {
	switch stmt.(type) {
	case *SetStatement:
		return "SET"
	case *NotifyStatement:
		return "NOTIFY"
	case *AgentStatement:
		return "AGENT"
	case *IfStatement:
		return "IF"
	default:
		return "UNKNOWN"
	}
}

func runtimeStatementTarget(stmt Statement) string {
	switch node := stmt.(type) {
	case *SetStatement:
		if node.Target != nil {
			return node.Target.Name
		}
	case *NotifyStatement:
		if node.Target != nil {
			return node.Target.Name
		}
	case *AgentStatement:
		if node.Name != nil {
			return node.Name.Name
		}
	}
	return ""
}
