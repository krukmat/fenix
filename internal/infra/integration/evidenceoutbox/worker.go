package evidenceoutbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/matiasleandrokruk/fenix/internal/domain/evidence"
	"github.com/matiasleandrokruk/fenix/internal/infra/integration/telemetry"
)

// ReconcileOnce performs one restart-safe evidence delivery and verification pass.
func (r *Recorder) ReconcileOnce(ctx context.Context) error {
	if err := r.reconcileRecords(ctx); err != nil {
		return err
	}
	if err := r.reconcileVerification(ctx); err != nil {
		return err
	}
	r.refreshPendingMetric(ctx)
	return nil
}

func (r *Recorder) reconcileRecords(ctx context.Context) error {
	rows, err := r.loadDueRecordRows(ctx, r.now())
	if err != nil {
		return err
	}
	for _, row := range rows {
		if rowErr := r.reconcileRecordRow(ctx, row); rowErr != nil {
			return rowErr
		}
	}
	return nil
}

func (r *Recorder) reconcileRecordRow(ctx context.Context, row deliveryRow) error {
	if row.State == evidence.DeliveryIndeterminate {
		ref, err := r.sink.LookupEvidence(ctx, row.StreamID, row.ExecutionID)
		if err != nil {
			return r.scheduleWorkerRetry(ctx, row.ExecutionID, err, false)
		}
		if ref != nil {
			telemetry.Default.IncLifecycle("reconciliation", "resolved")
			return r.persistWorkerRecord(ctx, row.ExecutionID, *ref, false)
		}
	}
	return r.appendWorker(ctx, row)
}

func (r *Recorder) appendWorker(ctx context.Context, row deliveryRow) error {
	ref, err := r.sink.RecordEvidence(ctx, row.Envelope)
	if err == nil {
		return r.persistWorkerRecord(ctx, row.ExecutionID, ref, true)
	}
	var indeterminate *evidence.IndeterminateRecordError
	if errors.As(err, &indeterminate) {
		telemetry.Default.IncLifecycle("delivery", "indeterminate")
		return r.scheduleWorkerRetry(ctx, row.ExecutionID, err, true)
	}
	telemetry.Default.IncLifecycle("delivery", "failed")
	return r.scheduleState(ctx, row.ExecutionID, evidence.DeliveryIndeterminate, err, nil, true)
}

func (r *Recorder) persistWorkerRecord(
	ctx context.Context,
	executionID string,
	ref evidence.ProofReference,
	incrementAttempt bool,
) error {
	if err := r.markRecorded(ctx, executionID, ref, incrementAttempt); err != nil {
		return err
	}
	telemetry.Default.IncLifecycle("delivery", "recorded")
	return nil
}

func (r *Recorder) scheduleWorkerRetry(
	ctx context.Context,
	executionID string,
	cause error,
	incrementAttempt bool,
) error {
	due := r.now().Add(r.retryDelay)
	telemetry.Default.IncLifecycle("reconciliation", "pending")
	return r.scheduleState(
		ctx,
		executionID,
		evidence.DeliveryIndeterminate,
		cause,
		&due,
		incrementAttempt,
	)
}

func (r *Recorder) reconcileVerification(ctx context.Context) error {
	rows, err := r.loadDueVerificationRows(ctx, r.now())
	if err != nil {
		return err
	}
	for streamID, group := range groupByStream(rows) {
		if verifyErr := r.verifyStreamGroup(ctx, streamID, group); verifyErr != nil {
			return verifyErr
		}
	}
	return nil
}

func groupByStream(rows []deliveryRow) map[string][]deliveryRow {
	groups := make(map[string][]deliveryRow)
	for _, row := range rows {
		groups[row.StreamID] = append(groups[row.StreamID], row)
	}
	return groups
}

func (r *Recorder) verifyStreamGroup(
	ctx context.Context,
	streamID string,
	rows []deliveryRow,
) error {
	if err := r.markGroupPendingCheckpoint(ctx, rows); err != nil {
		return err
	}
	progress, err := r.sink.CheckpointAndVerify(ctx, streamID)
	if err != nil {
		telemetry.Default.IncLifecycle("verification", "error")
		return r.rescheduleVerificationGroup(ctx, rows, err)
	}
	if validationErr := progress.Validate(); validationErr != nil {
		return r.rescheduleVerificationGroup(ctx, rows, validationErr)
	}
	telemetry.Default.IncLifecycle("verification", string(progress.Status))
	return r.applyVerificationProgress(ctx, rows, progress)
}

func (r *Recorder) markGroupPendingCheckpoint(ctx context.Context, rows []deliveryRow) error {
	due := r.now().Add(r.retryDelay)
	for _, row := range rows {
		if scheduleErr := r.scheduleState(
			ctx,
			row.ExecutionID,
			evidence.DeliveryPendingCheckpoint,
			nil,
			&due,
			false,
		); scheduleErr != nil {
			return scheduleErr
		}
	}
	telemetry.Default.IncLifecycle("checkpoint", "requested")
	return nil
}

func (r *Recorder) rescheduleVerificationGroup(
	ctx context.Context,
	rows []deliveryRow,
	cause error,
) error {
	due := r.now().Add(r.retryDelay)
	for _, row := range rows {
		if scheduleErr := r.scheduleState(
			ctx,
			row.ExecutionID,
			evidence.DeliveryPendingCheckpoint,
			cause,
			&due,
			false,
		); scheduleErr != nil {
			return scheduleErr
		}
	}
	return nil
}

func (r *Recorder) applyVerificationProgress(
	ctx context.Context,
	rows []deliveryRow,
	progress evidence.VerificationProgress,
) error {
	for _, row := range rows {
		if applyErr := r.applyVerificationRow(ctx, row, progress); applyErr != nil {
			return applyErr
		}
	}
	return nil
}

func (r *Recorder) applyVerificationRow(
	ctx context.Context,
	row deliveryRow,
	progress evidence.VerificationProgress,
) error {
	if row.Proof == nil {
		return fmt.Errorf("verification row %s has no proof reference", row.ExecutionID)
	}
	if row.Proof.Sequence > progress.Checkpoint.TreeSize {
		return r.rescheduleUncovered(ctx, row.ExecutionID)
	}
	proof := *row.Proof
	proof.Checkpoint = &progress.Checkpoint
	proof.VerificationStatus = progress.Status
	proof.VerificationIssues = append([]string(nil), progress.Issues...)
	return r.markVerification(ctx, row.ExecutionID, proof)
}

func (r *Recorder) rescheduleUncovered(ctx context.Context, executionID string) error {
	due := r.now().Add(r.retryDelay)
	return r.scheduleState(
		ctx,
		executionID,
		evidence.DeliveryPendingCheckpoint,
		errors.New("checkpoint does not yet cover evidence sequence"),
		&due,
		false,
	)
}
