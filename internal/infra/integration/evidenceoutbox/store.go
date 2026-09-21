package evidenceoutbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/evidence"
)

const deliveryColumns = `execution_id, stream_id, envelope_json, proof_json, delivery_state, attempt_count`

type deliveryRow struct {
	ExecutionID string
	StreamID    string
	EnvelopeRaw string
	Envelope    evidence.Envelope
	Proof       *evidence.ProofReference
	State       evidence.DeliveryState
	Attempts    int
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *Recorder) persistEnvelope(ctx context.Context, envelope evidence.Envelope) (deliveryRow, error) {
	raw, err := json.Marshal(envelope)
	if err != nil {
		return deliveryRow{}, fmt.Errorf("marshal evidence envelope: %w", err)
	}
	now := r.now()
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO evidence_delivery (
			execution_id, workspace_id, trace_id, stream_id, envelope_json,
			delivery_state, attempt_count, next_attempt_at_ms, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, ?)
		ON CONFLICT(execution_id) DO NOTHING`,
		envelope.ExecutionID,
		envelope.WorkspaceID,
		envelope.TraceID,
		envelope.StreamID,
		string(raw),
		string(evidence.DeliveryPendingRecord),
		now.UnixMilli(),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return deliveryRow{}, wrapDBError("persist envelope", err)
	}
	row, err := r.loadByExecution(ctx, envelope.ExecutionID)
	if err != nil {
		return deliveryRow{}, err
	}
	if row.EnvelopeRaw != string(raw) {
		return deliveryRow{}, ErrEnvelopeConflict
	}
	return row, nil
}

func (r *Recorder) loadByExecution(ctx context.Context, executionID string) (deliveryRow, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT `+deliveryColumns+` FROM evidence_delivery WHERE execution_id = ?`,
		executionID,
	)
	decoded, err := scanDelivery(row)
	if err != nil {
		return deliveryRow{}, wrapDBError("load execution", err)
	}
	return decoded, nil
}

func (r *Recorder) loadDueRecordRows(ctx context.Context, now time.Time) ([]deliveryRow, error) {
	return r.queryRows(
		ctx,
		`SELECT `+deliveryColumns+` FROM evidence_delivery
		 WHERE delivery_state IN (?, ?)
		   AND next_attempt_at_ms IS NOT NULL
		   AND next_attempt_at_ms <= ?
		 ORDER BY next_attempt_at_ms, execution_id
		 LIMIT 100`,
		string(evidence.DeliveryPendingRecord),
		string(evidence.DeliveryIndeterminate),
		now.UnixMilli(),
	)
}

func (r *Recorder) loadDueVerificationRows(ctx context.Context, now time.Time) ([]deliveryRow, error) {
	return r.queryRows(
		ctx,
		`SELECT `+deliveryColumns+` FROM evidence_delivery
		 WHERE delivery_state IN (?, ?)
		   AND next_attempt_at_ms IS NOT NULL
		   AND next_attempt_at_ms <= ?
		 ORDER BY stream_id, execution_id
		 LIMIT 100`,
		string(evidence.DeliveryRecorded),
		string(evidence.DeliveryPendingCheckpoint),
		now.UnixMilli(),
	)
}

func (r *Recorder) queryRows(ctx context.Context, query string, args ...any) ([]deliveryRow, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, wrapDBError("query due rows", err)
	}
	defer rows.Close()

	result := make([]deliveryRow, 0)
	for rows.Next() {
		row, scanErr := scanDelivery(rows)
		if scanErr != nil {
			return nil, wrapDBError("scan due row", scanErr)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapDBError("iterate due rows", err)
	}
	return result, nil
}

func scanDelivery(scanner rowScanner) (deliveryRow, error) {
	var row deliveryRow
	var proofRaw sql.NullString
	var state string
	if err := scanner.Scan(
		&row.ExecutionID,
		&row.StreamID,
		&row.EnvelopeRaw,
		&proofRaw,
		&state,
		&row.Attempts,
	); err != nil {
		return deliveryRow{}, err
	}
	if err := json.Unmarshal([]byte(row.EnvelopeRaw), &row.Envelope); err != nil {
		return deliveryRow{}, fmt.Errorf("decode envelope: %w", err)
	}
	row.State = evidence.DeliveryState(state)
	if proofRaw.Valid && proofRaw.String != "" {
		var proof evidence.ProofReference
		if err := json.Unmarshal([]byte(proofRaw.String), &proof); err != nil {
			return deliveryRow{}, fmt.Errorf("decode proof reference: %w", err)
		}
		row.Proof = &proof
	}
	return row, nil
}

func (r *Recorder) markRecorded(
	ctx context.Context,
	executionID string,
	proof evidence.ProofReference,
	incrementAttempt bool,
) error {
	return r.saveProof(
		ctx,
		executionID,
		proof,
		evidence.DeliveryRecorded,
		timePtr(r.now()),
		incrementAttempt,
	)
}

func (r *Recorder) markVerification(
	ctx context.Context,
	executionID string,
	proof evidence.ProofReference,
) error {
	return r.saveProof(ctx, executionID, proof, evidence.DeliveryStateFromProof(proof), nil, false)
}

func (r *Recorder) saveProof(
	ctx context.Context,
	executionID string,
	proof evidence.ProofReference,
	state evidence.DeliveryState,
	next *time.Time,
	incrementAttempt bool,
) error {
	if err := proof.Validate(); err != nil {
		return fmt.Errorf("validate durable proof: %w", err)
	}
	raw, err := json.Marshal(proof)
	if err != nil {
		return fmt.Errorf("marshal durable proof: %w", err)
	}
	_, err = r.db.ExecContext(
		ctx,
		`UPDATE evidence_delivery
		 SET proof_json = ?, delivery_state = ?, last_error = NULL,
		     next_attempt_at_ms = ?, attempt_count = attempt_count + ?, updated_at = ?
		 WHERE execution_id = ?`,
		string(raw),
		string(state),
		nextMillis(next),
		boolInt(incrementAttempt),
		r.now().Format(time.RFC3339Nano),
		executionID,
	)
	if err != nil {
		return wrapDBError("save proof", err)
	}
	return nil
}

func (r *Recorder) scheduleState(
	ctx context.Context,
	executionID string,
	state evidence.DeliveryState,
	cause error,
	next *time.Time,
	incrementAttempt bool,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE evidence_delivery
		 SET delivery_state = ?, last_error = ?, next_attempt_at_ms = ?,
		     attempt_count = attempt_count + ?, updated_at = ?
		 WHERE execution_id = ?`,
		string(state),
		errorText(cause),
		nextMillis(next),
		boolInt(incrementAttempt),
		r.now().Format(time.RFC3339Nano),
		executionID,
	)
	if err != nil {
		return wrapDBError("schedule state", err)
	}
	return nil
}

func (r *Recorder) countPending(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM evidence_delivery
		 WHERE delivery_state IN (?, ?, ?, ?)`,
		string(evidence.DeliveryPendingRecord),
		string(evidence.DeliveryIndeterminate),
		string(evidence.DeliveryRecorded),
		string(evidence.DeliveryPendingCheckpoint),
	).Scan(&count)
	if err != nil {
		return 0, wrapDBError("count pending", err)
	}
	return count, nil
}

func nextMillis(next *time.Time) any {
	if next == nil {
		return nil
	}
	return next.UnixMilli()
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func errorText(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}
