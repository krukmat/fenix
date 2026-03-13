package agent

import (
	"context"
	"fmt"
)

var ErrDSLRuntimeFailed = fmt.Errorf("dsl runtime failed")

type RuntimeOperationExecutor interface {
	Execute(ctx context.Context, op *RuntimeOperation, evalCtx map[string]any) (RuntimeExecutionResult, error)
}

type RuntimeWaitExecutor interface {
	ExecuteWait(ctx context.Context, stmt *WaitStatement, nextStatementIndex int, evalCtx map[string]any) (RuntimeExecutionResult, error)
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
	return r.ExecuteProgramFromIndex(ctx, program, 0, evalCtx, executor)
}

func (r *DSLRuntime) ExecuteProgramFromIndex(ctx context.Context, program *Program, startIndex int, evalCtx map[string]any, executor RuntimeOperationExecutor) (*DSLRuntimeResult, error) {
	if program == nil || program.Workflow == nil {
		return nil, fmt.Errorf("%w: program is required", ErrDSLRuntimeFailed)
	}
	if startIndex < 0 {
		return nil, fmt.Errorf("%w: start index must be >= 0", ErrDSLRuntimeFailed)
	}
	if startIndex > len(program.Workflow.Body) {
		return nil, fmt.Errorf("%w: start index %d out of range", ErrDSLRuntimeFailed, startIndex)
	}

	result := &DSLRuntimeResult{
		WorkflowName: program.Workflow.Name,
	}
	if program.Workflow.Trigger != nil {
		result.TriggerEvent = program.Workflow.Trigger.Event
	}

	statements := program.Workflow.Body
	cursor := 0
	if _, err := r.executeStatements(ctx, statements, evalCtx, executor, &result.Statements, startIndex, &cursor); err != nil {
		return result, err
	}
	return result, nil
}

func (r *DSLRuntime) executeStatements(ctx context.Context, statements []Statement, evalCtx map[string]any, executor RuntimeOperationExecutor, out *[]DSLStatementResult, startIndex int, cursor *int) (bool, error) {
	for _, stmt := range statements {
		index := *cursor
		*cursor++
		if index < startIndex && shouldDescendIntoSkippedStatement(stmt) {
			stop, err := r.executeSkippedSubtree(ctx, stmt, evalCtx, executor, out, startIndex, cursor)
			if err != nil {
				return false, err
			}
			if stop {
				return true, nil
			}
			continue
		}
		if index < startIndex {
			continue
		}

		stop, err := r.executeStatement(ctx, stmt, evalCtx, executor, out, startIndex, cursor)
		if err != nil {
			return false, err
		}
		if stop {
			return true, nil
		}
	}
	return false, nil
}

func shouldDescendIntoSkippedStatement(stmt Statement) bool {
	_, ok := stmt.(*IfStatement)
	return ok
}

func (r *DSLRuntime) executeSkippedSubtree(ctx context.Context, stmt Statement, evalCtx map[string]any, executor RuntimeOperationExecutor, out *[]DSLStatementResult, startIndex int, cursor *int) (bool, error) {
	ifStmt, ok := stmt.(*IfStatement)
	if !ok {
		return false, nil
	}
	return r.executeStatements(ctx, ifStmt.Body, evalCtx, executor, out, startIndex, cursor)
}

func (r *DSLRuntime) executeStatement(ctx context.Context, stmt Statement, evalCtx map[string]any, executor RuntimeOperationExecutor, out *[]DSLStatementResult, startIndex int, cursor *int) (bool, error) {
	switch node := stmt.(type) {
	case *IfStatement:
		return r.executeIf(ctx, node, evalCtx, executor, out, startIndex, cursor)
	case *WaitStatement:
		return r.executeWait(ctx, node, evalCtx, executor, out, *cursor)
	case *SetStatement, *NotifyStatement, *AgentStatement:
		return r.executeMappedStatement(ctx, stmt, evalCtx, executor, out)
	default:
		return false, fmt.Errorf("%w: unsupported statement type", ErrDSLRuntimeFailed)
	}
}

func startStatementTrace(ctx context.Context, executor RuntimeOperationExecutor, stmt Statement) (string, error) {
	tracer, ok := executor.(RuntimeStatementTracer)
	if !ok {
		return "", nil
	}
	return tracer.StartStatementTrace(ctx, stmt)
}

func finishStatementTrace(ctx context.Context, executor RuntimeOperationExecutor, traceID string, result DSLStatementResult, err error) {
	if traceID == "" {
		return
	}
	if tracer, ok := executor.(RuntimeStatementTracer); ok {
		_ = tracer.FinishStatementTrace(ctx, traceID, result, err)
	}
}

func (r *DSLRuntime) executeIf(ctx context.Context, stmt *IfStatement, evalCtx map[string]any, executor RuntimeOperationExecutor, out *[]DSLStatementResult, startIndex int, cursor *int) (bool, error) {
	traceID, traceErr := startStatementTrace(ctx, executor, stmt)
	if traceErr != nil {
		return false, traceErr
	}
	condOK, err := r.evaluator.EvaluateCondition(stmt.Condition, evalCtx)
	if err != nil {
		result := DSLStatementResult{Type: "IF", Status: StepStatusFailed, Position: stmt.Pos()}
		*out = append(*out, result)
		finishStatementTrace(ctx, executor, traceID, result, err)
		return false, err
	}
	if !condOK {
		result := DSLStatementResult{Type: "IF", Status: StepStatusSkipped, Position: stmt.Pos()}
		*out = append(*out, result)
		finishStatementTrace(ctx, executor, traceID, result, nil)
		return false, nil
	}
	result := DSLStatementResult{Type: "IF", Status: StepStatusSuccess, Position: stmt.Pos()}
	*out = append(*out, result)
	finishStatementTrace(ctx, executor, traceID, result, nil)
	return r.executeStatements(ctx, stmt.Body, evalCtx, executor, out, startIndex, cursor)
}

func (r *DSLRuntime) executeWait(ctx context.Context, stmt *WaitStatement, evalCtx map[string]any, executor RuntimeOperationExecutor, out *[]DSLStatementResult, nextStatementIndex int) (bool, error) {
	traceID, traceErr := startStatementTrace(ctx, executor, stmt)
	if traceErr != nil {
		return false, traceErr
	}
	waitExecutor, ok := executor.(RuntimeWaitExecutor)
	if !ok {
		err := fmt.Errorf("%w: WAIT requires scheduler executor", ErrDSLRuntimeFailed)
		result := DSLStatementResult{Type: "WAIT", Target: formatWaitTarget(stmt), Status: StepStatusFailed, Position: stmt.Pos()}
		*out = append(*out, result)
		finishStatementTrace(ctx, executor, traceID, result, err)
		return false, err
	}
	execResult, execErr := waitExecutor.ExecuteWait(ctx, stmt, nextStatementIndex, evalCtx)
	result := DSLStatementResult{
		Type:     "WAIT",
		Target:   formatWaitTarget(stmt),
		Status:   StepStatusSuccess,
		Position: stmt.Pos(),
		Output:   execResult.Output,
	}
	if execResult.Status != "" {
		result.Status = execResult.Status
	}
	if execErr != nil {
		result.Status = StepStatusFailed
		*out = append(*out, result)
		finishStatementTrace(ctx, executor, traceID, result, execErr)
		return false, execErr
	}
	*out = append(*out, result)
	finishStatementTrace(ctx, executor, traceID, result, nil)
	return execResult.Stop, nil
}

func (r *DSLRuntime) executeMappedStatement(ctx context.Context, stmt Statement, evalCtx map[string]any, executor RuntimeOperationExecutor, out *[]DSLStatementResult) (bool, error) {
	traceID, traceErr := startStatementTrace(ctx, executor, stmt)
	if traceErr != nil {
		return false, traceErr
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
		finishStatementTrace(ctx, executor, traceID, result, err)
		return false, err
	}
	if executor == nil {
		*out = append(*out, result)
		finishStatementTrace(ctx, executor, traceID, result, nil)
		return false, nil
	}
	stop, execErr := r.applyExecutorResult(ctx, executor, op, evalCtx, traceID, &result, out)
	return stop, execErr
}

func (r *DSLRuntime) applyExecutorResult(ctx context.Context, executor RuntimeOperationExecutor, op *RuntimeOperation, evalCtx map[string]any, traceID string, result *DSLStatementResult, out *[]DSLStatementResult) (bool, error) {
	execResult, execErr := executor.Execute(ctx, op, evalCtx)
	result.Output = execResult.Output
	if execResult.Status != "" {
		result.Status = execResult.Status
	}
	if execErr != nil {
		result.Status = StepStatusFailed
		*out = append(*out, *result)
		finishStatementTrace(ctx, executor, traceID, *result, execErr)
		return false, execErr
	}
	*out = append(*out, *result)
	finishStatementTrace(ctx, executor, traceID, *result, nil)
	return execResult.Stop, nil
}

func runtimeStatementType(stmt Statement) string {
	switch stmt.(type) {
	case *SetStatement:
		return "SET"
	case *NotifyStatement:
		return "NOTIFY"
	case *AgentStatement:
		return "AGENT"
	case *WaitStatement:
		return "WAIT"
	case *IfStatement:
		return "IF"
	default:
		return "UNKNOWN"
	}
}

func runtimeStatementTarget(stmt Statement) string {
	switch node := stmt.(type) {
	case *SetStatement:
		return identifierName(node.Target)
	case *NotifyStatement:
		return identifierName(node.Target)
	case *AgentStatement:
		return identifierName(node.Name)
	case *WaitStatement:
		return formatWaitTarget(node)
	}
	return ""
}

func identifierName(id *IdentifierExpr) string {
	if id == nil {
		return ""
	}
	return id.Name
}

func formatWaitTarget(stmt *WaitStatement) string {
	if stmt == nil {
		return ""
	}
	if stmt.Unit == "" {
		return fmt.Sprintf("%d", stmt.Amount)
	}
	return fmt.Sprintf("%d %s", stmt.Amount, stmt.Unit)
}
