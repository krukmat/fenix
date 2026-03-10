package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/matiasleandrokruk/fenix/pkg/uuid"
)

var (
	ErrInvalidRunTransition  = errors.New("invalid agent run status transition")
	ErrInvalidStepTransition = errors.New("invalid agent run step transition")
)

const (
	StepStatusPending  = "pending"
	StepStatusRunning  = "running"
	StepStatusSuccess  = "success"
	StepStatusFailed   = "failed"
	StepStatusSkipped  = "skipped"
	StepStatusRetrying = "retrying"
)

const (
	StepTypeRetrieveEvidence = "retrieve_evidence"
	StepTypeReason           = "reason"
	StepTypeToolCall         = "tool_call"
	StepTypeFinalize         = "finalize"
)

const maxStepRetries = 2

type RunStep struct {
	ID          string
	WorkspaceID string
	RunID       string
	StepIndex   int
	StepType    string
	Status      string
	Attempt     int
	Input       json.RawMessage
	Output      json.RawMessage
	Error       *string
	StartedAt   *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type runStepScanner interface {
	Scan(dest ...any) error
}

type runStepNullable struct {
	input       sql.NullString
	output      sql.NullString
	errorText   sql.NullString
	startedAt   sql.NullTime
	completedAt sql.NullTime
}

func (o *Orchestrator) ListRunSteps(ctx context.Context, workspaceID, runID string) ([]*RunStep, error) {
	rows, err := o.db.QueryContext(ctx, `
		SELECT id, workspace_id, agent_run_id, step_index, step_type, status, attempt,
		       input, output, error, started_at, completed_at, created_at, updated_at
		FROM agent_run_step
		WHERE workspace_id = ? AND agent_run_id = ?
		ORDER BY step_index ASC, attempt ASC, created_at ASC
	`, workspaceID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	steps := make([]*RunStep, 0)
	for rows.Next() {
		step, scanErr := scanRunStep(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		steps = append(steps, step)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}

	return steps, nil
}

func (o *Orchestrator) RecoverRun(ctx context.Context, workspaceID, runID string) (*Run, error) {
	run, err := o.GetAgentRun(ctx, workspaceID, runID)
	if err != nil {
		return nil, err
	}
	if !runNeedsRecovery(run) {
		return run, nil
	}
	return o.recoverRunningRun(ctx, run)
}

func (o *Orchestrator) createInitialRunStep(ctx context.Context, run *Run) error {
	_, err := o.db.ExecContext(ctx, `
		INSERT INTO agent_run_step (
			id, workspace_id, agent_run_id, step_index, step_type, status, attempt, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		uuid.NewV7().String(),
		run.WorkspaceID,
		run.ID,
		0,
		StepTypeRetrieveEvidence,
		StepStatusPending,
		1,
		time.Now().UTC(),
		time.Now().UTC(),
	)
	return err
}

func scanRunStep(scan runStepScanner) (*RunStep, error) {
	var step RunStep
	var n runStepNullable
	if err := scan.Scan(
		&step.ID,
		&step.WorkspaceID,
		&step.RunID,
		&step.StepIndex,
		&step.StepType,
		&step.Status,
		&step.Attempt,
		&n.input,
		&n.output,
		&n.errorText,
		&n.startedAt,
		&n.completedAt,
		&step.CreatedAt,
		&step.UpdatedAt,
	); err != nil {
		return nil, err
	}
	applyRunStepNullables(&step, &n)
	return &step, nil
}

func applyRunStepNullables(step *RunStep, n *runStepNullable) {
	if n.input.Valid {
		step.Input = json.RawMessage(n.input.String)
	}
	if n.output.Valid {
		step.Output = json.RawMessage(n.output.String)
	}
	if n.errorText.Valid {
		step.Error = &n.errorText.String
	}
	if n.startedAt.Valid {
		step.StartedAt = &n.startedAt.Time
	}
	if n.completedAt.Valid {
		step.CompletedAt = &n.completedAt.Time
	}
}

func isTerminalRunStatus(status string) bool {
	switch status {
	case StatusRejected, StatusDelegated, StatusSuccess, StatusPartial, StatusAbstained, StatusFailed, StatusEscalated:
		return true
	default:
		return false
	}
}

func validateRunTransition(current, next string) error {
	if current == next {
		return nil
	}
	if isTerminalRunStatus(current) {
		return ErrInvalidRunTransition
	}

	switch current {
	case StatusRunning:
		switch next {
		case StatusAccepted:
			return nil
		default:
			if isTerminalRunStatus(next) {
				return nil
			}
		}
	case StatusAccepted:
		switch next {
		case StatusSuccess, StatusPartial, StatusAbstained, StatusFailed, StatusDelegated:
			return nil
		}
	}
	return ErrInvalidRunTransition
}

func stepStatusForRun(status string) string {
	if status == StatusFailed || status == StatusRejected {
		return StepStatusFailed
	}
	return StepStatusSuccess
}

func isRetryableStepType(stepType string) bool {
	switch stepType {
	case StepTypeRetrieveEvidence, StepTypeToolCall:
		return true
	default:
		return false
	}
}

func findRunningRunStep(steps []*RunStep) *RunStep {
	for _, step := range steps {
		if step.Status == StepStatusRunning {
			return step
		}
	}
	return nil
}

func hasMeaningfulPayload(raw json.RawMessage) bool {
	value, ok := decodePayload(raw)
	if !ok {
		return false
	}
	return payloadValueIsMeaningful(value)
}

func synthesizeRunSteps(ctx context.Context, tx *sql.Tx, run *Run, updates RunUpdates) error {
	if err := ensureRetrieveStepTx(ctx, tx, run, updates); err != nil {
		return err
	}

	nextIndex, err := nextRunStepIndexTx(ctx, tx, run.WorkspaceID, run.ID)
	if err != nil {
		return err
	}

	nextIndex, err = maybeInsertRunStepTx(ctx, tx, run, nextIndex, StepTypeReason, updates.ReasoningTrace, nil)
	if err != nil {
		return err
	}
	_, err = maybeInsertRunStepTx(ctx, tx, run, nextIndex, StepTypeToolCall, updates.ToolCalls, nil)
	if err != nil {
		return err
	}

	if !updates.Completed && !isTerminalRunStatus(updates.Status) {
		return nil
	}
	return ensureFinalizeStepTx(ctx, tx, run, updates.Status, updates.Output)
}

func ensureRetrieveStepTx(ctx context.Context, tx *sql.Tx, run *Run, updates RunUpdates) error {
	step, err := getLatestRunStepByTypeTx(ctx, tx, run.WorkspaceID, run.ID, StepTypeRetrieveEvidence)
	if err != nil {
		return err
	}

	if step == nil {
		return insertRunStepTx(ctx, tx, &RunStep{
			ID:          uuid.NewV7().String(),
			WorkspaceID: run.WorkspaceID,
			RunID:       run.ID,
			StepIndex:   0,
			StepType:    StepTypeRetrieveEvidence,
			Status:      StepStatusPending,
			Attempt:     1,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		})
	}

	if step.Status != StepStatusPending && step.Status != StepStatusRunning {
		return nil
	}

	status := StepStatusSkipped
	var output json.RawMessage
	if hasMeaningfulPayload(updates.RetrievedEvidenceIDs) || hasMeaningfulPayload(updates.RetrievalQueries) {
		status = StepStatusSuccess
		output = updates.RetrievedEvidenceIDs
	}

	return updateRunStepStateTx(ctx, tx, step.ID, run.WorkspaceID, status, updates.RetrievalQueries, output, nil)
}

func maybeInsertRunStepTx(ctx context.Context, tx *sql.Tx, run *Run, stepIndex int, stepType string, output json.RawMessage, input json.RawMessage) (int, error) {
	if !hasMeaningfulPayload(output) {
		return stepIndex, nil
	}
	existing, err := getLatestRunStepByTypeTx(ctx, tx, run.WorkspaceID, run.ID, stepType)
	if err != nil {
		return stepIndex, err
	}
	if existing != nil {
		return stepIndex, nil
	}
	err = insertRunStepTx(ctx, tx, &RunStep{
		ID:          uuid.NewV7().String(),
		WorkspaceID: run.WorkspaceID,
		RunID:       run.ID,
		StepIndex:   stepIndex,
		StepType:    stepType,
		Status:      StepStatusSuccess,
		Attempt:     1,
		Input:       input,
		Output:      output,
		StartedAt:   timePtr(run.StartedAt),
		CompletedAt: timePtr(time.Now().UTC()),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return stepIndex, err
	}
	return stepIndex + 1, nil
}

func ensureFinalizeStepTx(ctx context.Context, tx *sql.Tx, run *Run, runStatus string, output json.RawMessage) error {
	existing, err := getLatestRunStepByTypeTx(ctx, tx, run.WorkspaceID, run.ID, StepTypeFinalize)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	index, err := nextRunStepIndexTx(ctx, tx, run.WorkspaceID, run.ID)
	if err != nil {
		return err
	}
	return insertRunStepTx(ctx, tx, &RunStep{
		ID:          uuid.NewV7().String(),
		WorkspaceID: run.WorkspaceID,
		RunID:       run.ID,
		StepIndex:   index,
		StepType:    StepTypeFinalize,
		Status:      stepStatusForRun(runStatus),
		Attempt:     1,
		Output:      output,
		StartedAt:   timePtr(run.StartedAt),
		CompletedAt: timePtr(time.Now().UTC()),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	})
}

func finalizeRunStatusTx(ctx context.Context, tx *sql.Tx, workspaceID, runID, status string, latencyMs *int64) error {
	now := time.Now().UTC()
	_, err := tx.ExecContext(ctx, `
		UPDATE agent_run
		SET status = ?, completed_at = ?, latency_ms = COALESCE(?, latency_ms), updated_at = ?
		WHERE id = ? AND workspace_id = ?
	`, status, now, latencyMs, now, runID, workspaceID)
	return err
}

func reconcileOpenStepsTx(ctx context.Context, tx *sql.Tx, workspaceID, runID, terminalStepStatus string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE agent_run_step
		SET status = ?, completed_at = COALESCE(completed_at, ?), updated_at = ?
		WHERE workspace_id = ? AND agent_run_id = ? AND status IN (?, ?)
	`, terminalStepStatus, time.Now().UTC(), time.Now().UTC(), workspaceID, runID, StepStatusPending, StepStatusRunning)
	return err
}

func insertRunStepTx(ctx context.Context, tx *sql.Tx, step *RunStep) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO agent_run_step (
			id, workspace_id, agent_run_id, step_index, step_type, status, attempt,
			input, output, error, started_at, completed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		step.ID,
		step.WorkspaceID,
		step.RunID,
		step.StepIndex,
		step.StepType,
		step.Status,
		step.Attempt,
		nullJSON(step.Input),
		nullJSON(step.Output),
		step.Error,
		step.StartedAt,
		step.CompletedAt,
		step.CreatedAt,
		step.UpdatedAt,
	)
	return err
}

func listRunStepsTx(ctx context.Context, tx *sql.Tx, workspaceID, runID string) ([]*RunStep, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, workspace_id, agent_run_id, step_index, step_type, status, attempt,
		       input, output, error, started_at, completed_at, created_at, updated_at
		FROM agent_run_step
		WHERE workspace_id = ? AND agent_run_id = ?
		ORDER BY step_index ASC, attempt ASC, created_at ASC
	`, workspaceID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	steps := make([]*RunStep, 0)
	for rows.Next() {
		step, scanErr := scanRunStep(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		steps = append(steps, step)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, rowsErr
	}
	return steps, nil
}

func getLatestRunStepByTypeTx(ctx context.Context, tx *sql.Tx, workspaceID, runID, stepType string) (*RunStep, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, workspace_id, agent_run_id, step_index, step_type, status, attempt,
		       input, output, error, started_at, completed_at, created_at, updated_at
		FROM agent_run_step
		WHERE workspace_id = ? AND agent_run_id = ? AND step_type = ?
		ORDER BY step_index DESC, attempt DESC, created_at DESC
		LIMIT 1
	`, workspaceID, runID, stepType)

	step, err := scanRunStep(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return step, nil
}

func getLatestRunStepByIDTx(ctx context.Context, tx *sql.Tx, workspaceID, stepID string) (*RunStep, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, workspace_id, agent_run_id, step_index, step_type, status, attempt,
		       input, output, error, started_at, completed_at, created_at, updated_at
		FROM agent_run_step
		WHERE workspace_id = ? AND id = ?
	`, workspaceID, stepID)
	step, err := scanRunStep(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return step, nil
}

func nextRunStepIndexTx(ctx context.Context, tx *sql.Tx, workspaceID, runID string) (int, error) {
	var next int
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(step_index) + 1, 0)
		FROM agent_run_step
		WHERE workspace_id = ? AND agent_run_id = ?
	`, workspaceID, runID).Scan(&next); err != nil {
		return 0, err
	}
	return next, nil
}

func updateRunStepStatusTx(ctx context.Context, tx *sql.Tx, stepID, workspaceID, status string, output json.RawMessage, errText *string) error {
	step, err := getLatestRunStepByIDTx(ctx, tx, workspaceID, stepID)
	if err != nil {
		return err
	}
	if step == nil {
		return ErrInvalidStepTransition
	}
	return updateRunStepStateTx(ctx, tx, stepID, workspaceID, status, step.Input, output, errText)
}

func updateRunStepStateTx(ctx context.Context, tx *sql.Tx, stepID, workspaceID, status string, input, output json.RawMessage, errText *string) error {
	step, err := getLatestRunStepByIDTx(ctx, tx, workspaceID, stepID)
	if err != nil {
		return err
	}
	if step == nil {
		return ErrInvalidStepTransition
	}
	err = validateStepTransition(step.Status, status)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	startedAt, completedAt := deriveStepTimestamps(step, status, now)

	_, err = tx.ExecContext(ctx, `
		UPDATE agent_run_step
		SET status = ?, input = ?, output = ?, error = ?, started_at = ?, completed_at = ?, updated_at = ?
		WHERE id = ? AND workspace_id = ?
	`, status, nullJSON(input), nullJSON(output), errText, startedAt, completedAt, now, stepID, workspaceID)
	return err
}

func validateStepTransition(current, next string) error {
	if current == next || isAllowedStepTransition(current, next) {
		return nil
	}
	return ErrInvalidStepTransition
}

func nullJSON(raw json.RawMessage) any {
	if !hasMeaningfulPayload(raw) {
		return nil
	}
	return raw
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func calculateRunLatency(startedAt time.Time) *int64 {
	latency := time.Since(startedAt).Milliseconds()
	return &latency
}

func runNeedsRecovery(run *Run) bool {
	return run.Status == StatusRunning
}

func (o *Orchestrator) recoverRunningRun(ctx context.Context, run *Run) (*Run, error) {
	tx, current, err := o.loadRecoverableStep(ctx, run.WorkspaceID, run.ID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		_ = tx.Rollback()
		return run, nil
	}
	defer func() { _ = tx.Rollback() }()

	if shouldFailRecovery(current) {
		return o.failRecoveredRun(ctx, tx, run, current)
	}
	return o.retryRecoveredRun(ctx, tx, run, current)
}

func (o *Orchestrator) loadRecoverableStep(ctx context.Context, workspaceID, runID string) (*sql.Tx, *RunStep, error) {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	steps, err := listRunStepsTx(ctx, tx, workspaceID, runID)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	return tx, findRunningRunStep(steps), nil
}

func (o *Orchestrator) failRecoveredRun(ctx context.Context, tx *sql.Tx, run *Run, current *RunStep) (*Run, error) {
	if err := markRecoveredRunFailed(ctx, tx, run, current); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return o.GetAgentRun(ctx, run.WorkspaceID, run.ID)
}

func shouldFailRecovery(step *RunStep) bool {
	return !isRetryableStepType(step.StepType) || step.Attempt >= maxStepRetries
}

func markRecoveredRunFailed(ctx context.Context, tx *sql.Tx, run *Run, current *RunStep) error {
	if err := updateRunStepStatusTx(ctx, tx, current.ID, run.WorkspaceID, StepStatusFailed, current.Output, current.Error); err != nil {
		return err
	}
	if err := finalizeRunStatusTx(ctx, tx, run.WorkspaceID, run.ID, StatusFailed, nil); err != nil {
		return err
	}
	if err := reconcileOpenStepsTx(ctx, tx, run.WorkspaceID, run.ID, StepStatusFailed); err != nil {
		return err
	}
	return ensureFinalizeStepTx(ctx, tx, run, StatusFailed, nil)
}

func queueRetryStepTx(ctx context.Context, tx *sql.Tx, workspaceID, runID string, current *RunStep) error {
	if err := updateRunStepStatusTx(ctx, tx, current.ID, workspaceID, StepStatusRetrying, current.Output, current.Error); err != nil {
		return err
	}
	return insertRunStepTx(ctx, tx, retryStepFromCurrent(workspaceID, runID, current))
}

func (o *Orchestrator) retryRecoveredRun(ctx context.Context, tx *sql.Tx, run *Run, current *RunStep) (*Run, error) {
	if err := queueRetryStepTx(ctx, tx, run.WorkspaceID, run.ID, current); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return o.GetAgentRun(ctx, run.WorkspaceID, run.ID)
}

func retryStepFromCurrent(workspaceID, runID string, current *RunStep) *RunStep {
	now := time.Now().UTC()
	return &RunStep{
		ID:          uuid.NewV7().String(),
		WorkspaceID: workspaceID,
		RunID:       runID,
		StepIndex:   current.StepIndex,
		StepType:    current.StepType,
		Status:      StepStatusPending,
		Attempt:     current.Attempt + 1,
		Input:       current.Input,
		Output:      current.Output,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func decodePayload(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil, false
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false
	}
	return value, true
}

func payloadValueIsMeaningful(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	default:
		return true
	}
}

func deriveStepTimestamps(step *RunStep, status string, now time.Time) (*time.Time, *time.Time) {
	startedAt := step.StartedAt
	if shouldSetStepStartedAt(startedAt, status) {
		startedAt = &now
	}
	completedAt := step.CompletedAt
	if shouldSetStepCompletedAt(status) {
		completedAt = &now
	}
	return startedAt, completedAt
}

func shouldSetStepStartedAt(startedAt *time.Time, status string) bool {
	return startedAt == nil && status != StepStatusPending
}

func shouldSetStepCompletedAt(status string) bool {
	return status != StepStatusPending && status != StepStatusRunning
}

func isAllowedStepTransition(current, next string) bool {
	return nextStepStatusMap(current)[next]
}

func nextStepStatusMap(current string) map[string]bool {
	switch current {
	case StepStatusPending:
		return map[string]bool{
			StepStatusRunning: true,
			StepStatusSuccess: true,
			StepStatusSkipped: true,
			StepStatusFailed:  true,
		}
	case StepStatusRunning:
		return map[string]bool{
			StepStatusSuccess:  true,
			StepStatusFailed:   true,
			StepStatusRetrying: true,
		}
	case StepStatusRetrying:
		return map[string]bool{StepStatusPending: true}
	default:
		return map[string]bool{}
	}
}
