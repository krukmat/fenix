// Package evidenceoutbox provides restart-safe delivery and verification of Fenix evidence.
package evidenceoutbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/evidence"
	"github.com/matiasleandrokruk/fenix/internal/domain/tool"
	"github.com/matiasleandrokruk/fenix/internal/infra/integration/telemetry"
)

const (
	defaultWorkerInterval = 5 * time.Second
	defaultRetryDelay     = 5 * time.Second
)

var (
	ErrInvalidRecorderConfig = errors.New("invalid durable evidence recorder configuration")
	ErrEnvelopeConflict      = errors.New("durable evidence execution envelope conflict")
)

// LifecycleSink extends the append/lookup port with provider-owned checkpoint verification.
type LifecycleSink interface {
	evidence.RuntimeSink
	CheckpointAndVerify(ctx context.Context, streamID string) (evidence.VerificationProgress, error)
}

// Recorder persists minimized evidence before any provider call and reconciles it in background.
type Recorder struct {
	db             *sql.DB
	sink           LifecycleSink
	workerInterval time.Duration
	retryDelay     time.Duration
	now            func() time.Time
}

// NewRecorder creates a durable evidence recorder backed by the Fenix SQLite database.
func NewRecorder(db *sql.DB, sink LifecycleSink) (*Recorder, error) {
	if db == nil || sink == nil {
		return nil, ErrInvalidRecorderConfig
	}
	return &Recorder{
		db:             db,
		sink:           sink,
		workerInterval: defaultWorkerInterval,
		retryDelay:     defaultRetryDelay,
		now:            func() time.Time { return time.Now().UTC() },
	}, nil
}

// Start runs reconciliation until the router-owned background context is cancelled.
func (r *Recorder) Start(ctx context.Context) {
	r.runReconciliation(ctx)
	ticker := time.NewTicker(r.workerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runReconciliation(ctx)
		}
	}
}

func (r *Recorder) runReconciliation(ctx context.Context) {
	if err := r.ReconcileOnce(ctx); err != nil {
		telemetry.Default.IncLifecycle("worker", "error")
	}
}

// RecordCapabilityEvidence persists the envelope first, then attempts immediate evidence delivery.
func (r *Recorder) RecordCapabilityEvidence(
	ctx context.Context,
	request tool.CapabilityEvidenceRequest,
) (tool.CapabilityEvidenceResult, error) {
	envelope, err := evidence.BuildRuntimeEnvelope(request)
	if err != nil {
		return pendingResult("", ""), err
	}
	row, err := r.persistEnvelope(ctx, envelope)
	if err != nil {
		return pendingResult(envelope.ExecutionID, envelope.StreamID), err
	}
	r.refreshPendingMetric(ctx)
	return r.deliverInteractive(ctx, row)
}

func (r *Recorder) deliverInteractive(
	ctx context.Context,
	row deliveryRow,
) (tool.CapabilityEvidenceResult, error) {
	if row.Proof != nil && hasUsableProofState(row.State) {
		return evidence.RuntimeResultFromProof(row.Envelope, *row.Proof)
	}
	if row.State == evidence.DeliveryIndeterminate {
		return r.lookupInteractive(ctx, row, false)
	}
	return r.appendInteractive(ctx, row)
}

func (r *Recorder) appendInteractive(
	ctx context.Context,
	row deliveryRow,
) (tool.CapabilityEvidenceResult, error) {
	ref, err := r.sink.RecordEvidence(ctx, row.Envelope)
	if err == nil {
		return r.finishInteractiveRecord(ctx, row, ref, true)
	}
	var indeterminate *evidence.IndeterminateRecordError
	if errors.As(err, &indeterminate) {
		return r.lookupInteractive(ctx, row, true)
	}
	if updateErr := r.scheduleState(ctx, row.ExecutionID, evidence.DeliveryIndeterminate, err, nil, true); updateErr != nil {
		return pendingResult(row.ExecutionID, row.StreamID), updateErr
	}
	telemetry.Default.IncLifecycle("delivery", "failed")
	return pendingResult(row.ExecutionID, row.StreamID), err
}

func (r *Recorder) lookupInteractive(
	ctx context.Context,
	row deliveryRow,
	incrementAttempt bool,
) (tool.CapabilityEvidenceResult, error) {
	ref, err := r.sink.LookupEvidence(ctx, row.StreamID, row.ExecutionID)
	if err != nil || ref == nil {
		if scheduleErr := r.scheduleRetry(ctx, row.ExecutionID, err, incrementAttempt); scheduleErr != nil {
			return pendingResult(row.ExecutionID, row.StreamID), scheduleErr
		}
		telemetry.Default.IncLifecycle("reconciliation", "pending")
		return pendingResult(row.ExecutionID, row.StreamID), nil
	}
	telemetry.Default.IncLifecycle("reconciliation", "resolved")
	return r.finishInteractiveRecord(ctx, row, *ref, incrementAttempt)
}

func (r *Recorder) finishInteractiveRecord(
	ctx context.Context,
	row deliveryRow,
	ref evidence.ProofReference,
	incrementAttempt bool,
) (tool.CapabilityEvidenceResult, error) {
	if err := r.markRecorded(ctx, row.ExecutionID, ref, incrementAttempt); err != nil {
		return pendingResult(row.ExecutionID, row.StreamID), err
	}
	telemetry.Default.IncLifecycle("delivery", "recorded")
	r.refreshPendingMetric(ctx)
	return evidence.RuntimeResultFromProof(row.Envelope, ref)
}

func (r *Recorder) scheduleRetry(
	ctx context.Context,
	executionID string,
	cause error,
	incrementAttempt bool,
) error {
	due := r.now().Add(r.retryDelay)
	return r.scheduleState(
		ctx,
		executionID,
		evidence.DeliveryIndeterminate,
		cause,
		&due,
		incrementAttempt,
	)
}

func hasUsableProofState(state evidence.DeliveryState) bool {
	switch state {
	case evidence.DeliveryRecorded,
		evidence.DeliveryPendingCheckpoint,
		evidence.DeliveryVerified,
		evidence.DeliveryVerificationFailed:
		return true
	default:
		return false
	}
}

func pendingResult(executionID, streamID string) tool.CapabilityEvidenceResult {
	metadata := map[string]any{"reconciliation": "durable"}
	if executionID != "" {
		metadata["execution_id"] = executionID
	}
	if streamID != "" {
		metadata["stream_id"] = streamID
	}
	return tool.CapabilityEvidenceResult{
		State:         string(evidence.DeliveryIndeterminate),
		AuditMetadata: metadata,
	}
}

func (r *Recorder) refreshPendingMetric(ctx context.Context) {
	count, err := r.countPending(ctx)
	if err == nil {
		telemetry.Default.SetPending(count)
	}
}

func wrapDBError(operation string, err error) error {
	return fmt.Errorf("evidence outbox %s: %w", operation, err)
}
