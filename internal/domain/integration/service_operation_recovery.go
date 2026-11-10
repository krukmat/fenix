package integration

import (
	"context"
	"time"
)

// Lookup retrieves only a current active Fenix principal's own durable operation.
// It is NOT an HTTP response projection; C4 must omit request/config digests.
func (s *ServiceOperationStore) Lookup(
	ctx context.Context, scope ServiceOperationScope, operationID string,
) (ServiceOperationRecord, error) {
	if s == nil || s.db == nil {
		return ServiceOperationRecord{}, ErrServiceOperationUnavailable
	}
	if !validScopePart(scope.PrincipalID) || !validScopePart(scope.WorkspaceID) ||
		!validScopePart(scope.OrgID) || (scope.ProjectID != "" && !validScopePart(scope.ProjectID)) ||
		!b2cOperationIDPattern.MatchString(operationID) {
		return ServiceOperationRecord{}, ErrServiceOperationInvalid
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return ServiceOperationRecord{}, ErrServiceOperationUnavailable
	}
	defer func() { _ = conn.Close() }()
	return readServiceOperation(ctx, conn, scope, operationID)
}

// AcquireLease records a single bounded claim on an already accepted row.
// No provider, agent, capability or background worker is started here.
// Expired claims are never reacquired; only an explicit reconciliation may
// transition them to indeterminate without side effects.
func (s *ServiceOperationStore) AcquireLease(
	ctx context.Context, scope ServiceOperationScope, operationID, workerID string,
	registry *S2SCapabilityRegistry, now time.Time, ttl time.Duration,
) (bool, error) {
	if !validScopePart(workerID) || now.Unix() <= 0 || ttl < time.Second || ttl > 10*time.Minute {
		return false, ErrServiceOperationInvalid
	}
	record, err := s.Lookup(ctx, scope, operationID)
	if err != nil {
		return false, err
	}
	if registry == nil {
		return false, ErrServiceOperationDenied
	}
	if _, err = registry.Resolve(scope.PrincipalID, scope.WorkspaceID, record.CapabilityKey, record.SchemaVersion); err != nil {
		return false, ErrServiceOperationDenied
	}
	until := now.Add(ttl).Unix()
	if until <= now.Unix() {
		return false, ErrServiceOperationInvalid
	}
	res, err := s.db.ExecContext(ctx, `UPDATE service_operation SET lease_owner=?,lease_until_unix=?,updated_at_unix=?
        WHERE principal_id=? AND operation_id=? AND workspace_id=?
          AND dubbridge_org_id=? AND dubbridge_project_id=?
          AND status='accepted' AND lease_owner IS NULL AND lease_until_unix IS NULL
          AND EXISTS (
             SELECT 1 FROM service_principal p WHERE p.id=service_operation.principal_id
               AND p.consumer='dubbridge' AND p.status='active'
               AND p.revoked_at_unix IS NULL AND p.workspace_id=service_operation.workspace_id
               AND p.dubbridge_org_id=service_operation.dubbridge_org_id
               AND COALESCE(p.dubbridge_project_id,'')=service_operation.dubbridge_project_id
          )`, workerID, until, now.Unix(),
		scope.PrincipalID, operationID, scope.WorkspaceID, scope.OrgID, scope.ProjectID)
	if err != nil {
		return false, ErrServiceOperationUnavailable
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, ErrServiceOperationUnavailable
	}
	return rows == 1, nil
}

// MarkIndeterminate records known uncertainty under the exact leasing worker.
// It never resumes an accepted operation or repeats side effects.
func (s *ServiceOperationStore) MarkIndeterminate(
	ctx context.Context, scope ServiceOperationScope, operationID, workerID string, now time.Time,
) (bool, error) {
	if !validScopePart(workerID) || now.Unix() <= 0 {
		return false, ErrServiceOperationInvalid
	}
	if _, err := s.Lookup(ctx, scope, operationID); err != nil {
		return false, err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE service_operation
        SET status='indeterminate',lease_owner=NULL,lease_until_unix=NULL,updated_at_unix=?
        WHERE principal_id=? AND operation_id=? AND workspace_id=?
          AND dubbridge_org_id=? AND dubbridge_project_id=?
          AND status='accepted' AND lease_owner=?`,
		now.Unix(), scope.PrincipalID, operationID, scope.WorkspaceID,
		scope.OrgID, scope.ProjectID, workerID)
	if err != nil {
		return false, ErrServiceOperationUnavailable
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, ErrServiceOperationUnavailable
	}
	return n == 1, nil
}

// ReconcileExpiredLeases is an explicit operator/scheduler action, NOT a
// background worker. Any ambiguous claim is made indeterminate, never replayed.
// It may reconcile operations of revoked principals for audit consistency.
func (s *ServiceOperationStore) ReconcileExpiredLeases(
	ctx context.Context, now time.Time,
) (int64, error) {
	if now.Unix() <= 0 {
		return 0, ErrServiceOperationInvalid
	}
	if s == nil || s.db == nil {
		return 0, ErrServiceOperationUnavailable
	}
	res, err := s.db.ExecContext(ctx, `UPDATE service_operation
        SET status='indeterminate',lease_owner=NULL,lease_until_unix=NULL,updated_at_unix=?
        WHERE status='accepted' AND lease_owner IS NOT NULL
          AND lease_until_unix IS NOT NULL AND lease_until_unix<=?`,
		now.Unix(), now.Unix())
	if err != nil {
		return 0, ErrServiceOperationUnavailable
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, ErrServiceOperationUnavailable
	}
	return n, nil
}
