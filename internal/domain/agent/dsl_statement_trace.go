package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

const StepTypeDSLStatement = "dsl_statement"

func insertDSLRunStep(ctx context.Context, rc *RunContext, workspaceID, runID string, input json.RawMessage) (string, error) {
	if rc == nil || rc.DB == nil {
		return "", nil
	}
	tx, err := rc.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	index, err := nextRunStepIndexTx(ctx, tx, workspaceID, runID)
	if err != nil {
		return "", err
	}
	stepID := uuid.NewV7().String()
	now := time.Now().UTC()
	if insertErr := insertRunStepTx(ctx, tx, &RunStep{
		ID:          stepID,
		WorkspaceID: workspaceID,
		RunID:       runID,
		StepIndex:   index,
		StepType:    StepTypeDSLStatement,
		Status:      StepStatusRunning,
		Attempt:     1,
		Input:       input,
		StartedAt:   timePtr(now),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); insertErr != nil {
		return "", insertErr
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return "", commitErr
	}
	return stepID, nil
}

func updateDSLRunStep(ctx context.Context, rc *RunContext, workspaceID, stepID, status string, output json.RawMessage, stepErr error) error {
	if rc == nil || rc.DB == nil || stepID == "" {
		return nil
	}
	tx, err := rc.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var errText *string
	if stepErr != nil {
		msg := stepErr.Error()
		errText = &msg
	}
	if updateErr := updateRunStepStateTx(ctx, tx, stepID, workspaceID, status, nil, output, errText); updateErr != nil {
		return updateErr
	}
	return tx.Commit()
}

func marshalDSLStatementInput(stmt Statement) json.RawMessage {
	payload := map[string]any{
		"type":     runtimeStatementType(stmt),
		"target":   runtimeStatementTarget(stmt),
		"position": stmt.Pos(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return raw
}

func marshalDSLStatementOutput(result DSLStatementResult) json.RawMessage {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil
	}
	return raw
}

type tracedDSLExecutor struct {
	workspaceID string
	runID       string
	rc          *RunContext
	delegate    RuntimeOperationExecutor
}

func newTracedDSLExecutor(workspaceID, runID string, rc *RunContext, _ *DSLRuntime, delegate RuntimeOperationExecutor) *tracedDSLExecutor {
	return &tracedDSLExecutor{
		workspaceID: workspaceID,
		runID:       runID,
		rc:          rc,
		delegate:    delegate,
	}
}

func (e *tracedDSLExecutor) Execute(ctx context.Context, op *RuntimeOperation, evalCtx map[string]any) (RuntimeExecutionResult, error) {
	return e.delegate.Execute(ctx, op, evalCtx)
}

func (e *tracedDSLExecutor) ExecuteWait(ctx context.Context, stmt *WaitStatement, nextStatementIndex int, evalCtx map[string]any) (RuntimeExecutionResult, error) {
	waitExecutor, ok := e.delegate.(RuntimeWaitExecutor)
	if !ok {
		return RuntimeExecutionResult{}, ErrDSLRuntimeFailed
	}
	return waitExecutor.ExecuteWait(ctx, stmt, nextStatementIndex, evalCtx)
}

func (e *tracedDSLExecutor) StartStatementTrace(ctx context.Context, stmt Statement) (string, error) {
	return insertDSLRunStep(ctx, e.rc, e.workspaceID, e.runID, marshalDSLStatementInput(stmt))
}

func (e *tracedDSLExecutor) FinishStatementTrace(ctx context.Context, traceID string, result DSLStatementResult, stepErr error) error {
	if traceID == "" {
		return nil
	}
	return updateDSLRunStep(
		ctx,
		e.rc,
		e.workspaceID,
		traceID,
		normalizeDSLTraceStatus(result.Status, result.Output),
		marshalDSLStatementOutput(result),
		stepErr,
	)
}

func normalizeDSLTraceStatus(status string, output any) string {
	switch status {
	case "", StatusSuccess:
		return StepStatusSuccess
	case StatusAccepted:
		return acceptedTraceStatus(output)
	case StatusDelegated:
		return StepStatusSuccess
	case StatusRejected:
		return StepStatusFailed
	case StatusAbstained:
		return StepStatusSuccess
	default:
		return status
	}
}

func acceptedTraceStatus(output any) string {
	if indicatesPendingApproval(output) || indicatesPendingDispatchAccepted(output) {
		return StepStatusRunning
	}
	return StepStatusSuccess
}

func indicatesPendingApproval(output any) bool {
	switch v := output.(type) {
	case map[string]any:
		action, _ := v["action"].(string)
		if action == pendingApprovalAction {
			return true
		}
		status, _ := v["status"].(string)
		return status == pendingApprovalAction
	default:
		return false
	}
}

func indicatesPendingDispatchAccepted(output any) bool {
	payload, ok := output.(map[string]any)
	if !ok {
		return false
	}
	action, _ := payload["action"].(string)
	result, _ := payload["dispatch_result"].(string)
	return action == pendingDispatchAction && result == dispatchResultAccepted
}
