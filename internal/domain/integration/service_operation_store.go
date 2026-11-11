package integration

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// ServiceOperationStore owns only an inert Fenix SQLite operation journal.
// It neither authenticates the bearer nor mounts a route or invokes a provider.
type ServiceOperationStore struct{ db *sql.DB }

// ServiceOperationScope MUST be supplied from the Fenix-verified S2S context.
// The store independently checks all fields against the current live principal.
type ServiceOperationScope struct {
	PrincipalID string
	WorkspaceID string
	OrgID       string
	ProjectID   string
}

// ServiceOperationRequest is a verified, bounded request for durable reservation.
// RequestSHA256 must come from CanonicalOperationDigest over the approved envelope.
// ConfigSHA256 is populated by Reserve from the resolved Fenix capability binding;
// any caller-supplied value is ignored. Full policy fingerprinting belongs to B2.B4.
type ServiceOperationRequest struct {
	Scope         ServiceOperationScope
	OperationID   string
	CapabilityKey string
	SchemaVersion string
	TraceID       string
	RequestSHA256 [sha256.Size]byte
	ConfigSHA256  [sha256.Size]byte
	// ExpectedConfigSHA256 is a transient operator-policy snapshot guard;
	// it is NEVER accepted from the external B1 JSON and is not persisted.
	ExpectedConfigSHA256 [sha256.Size]byte
	// Managed policy snapshot is internal-only, never accepted from B1 JSON.
	// Used to recheck the enabled policy inside the SAME SQLite transaction
	// as the operation reservation, eliminating disable/replay TOCTOU.
	ManagedSchemaSHA256 [sha256.Size]byte
	ManagedHandlerID    string
}

type ServiceOperationRecord struct {
	ServiceOperationRequest
	Status         string
	RunID          string
	LeaseOwner     string
	LeaseUntilUnix int64
	CreatedAtUnix  int64
	UpdatedAtUnix  int64
}

var (
	ErrServiceOperationDenied      = errors.New("service operation unavailable")
	ErrServiceOperationConflict    = errors.New("service operation conflict")
	ErrServiceOperationUnavailable = errors.New("service operation storage unavailable")
)

var b2cOperationIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,96}$`)
var b2cTraceIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,96}$`)

func NewServiceOperationStore(db *sql.DB) *ServiceOperationStore {
	return &ServiceOperationStore{db: db}
}

// Reserve creates one accepted operation per (principal, operation_id), or
// returns the identical stored record as a replay. It never dispatches work.
// Caller MUST first use actual service authentication; this second check
// rejects a revoked principal or a forged source mapping even if caller errs.
func (s *ServiceOperationStore) Reserve(
	ctx context.Context, req ServiceOperationRequest, registry *S2SCapabilityRegistry, now time.Time,
) (ServiceOperationRecord, bool, error) {
	if !validOperationReservation(req, now) {
		return ServiceOperationRecord{}, false, ErrServiceOperationInvalid
	}
	if registry == nil {
		return ServiceOperationRecord{}, false, ErrServiceOperationDenied
	}
	binding, err := registry.ResolveContext(ctx, req.Scope.PrincipalID, req.Scope.WorkspaceID,
		req.CapabilityKey, req.SchemaVersion)
	if err != nil {
		return ServiceOperationRecord{}, false, ErrServiceOperationDenied
	}
	fingerprint := serviceCapabilityFingerprint(binding)
	var zero [sha256.Size]byte
	if registry.durable &&
		(req.ManagedSchemaSHA256 == zero || req.ManagedHandlerID == "" ||
			req.ManagedSchemaSHA256 != binding.SchemaSHA256 ||
			req.ManagedHandlerID != binding.HandlerID) {
		return ServiceOperationRecord{}, false, ErrServiceOperationDenied
	}
	if req.ExpectedConfigSHA256 != zero &&
		subtle.ConstantTimeCompare(req.ExpectedConfigSHA256[:], fingerprint[:]) != 1 {
		return ServiceOperationRecord{}, false, ErrServiceOperationDenied
	}
	req.ConfigSHA256 = fingerprint
	return s.reserveOperationTransaction(ctx, req, now)
}

func (s *ServiceOperationStore) reserveOperationTransaction(
	ctx context.Context, req ServiceOperationRequest, now time.Time,
) (ServiceOperationRecord, bool, error) {
	if s == nil || s.db == nil {
		return ServiceOperationRecord{}, false, ErrServiceOperationUnavailable
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return ServiceOperationRecord{}, false, ErrServiceOperationUnavailable
	}
	defer func() { _ = conn.Close() }()
	return reserveOperationOnConnection(ctx, conn, req, now)
}

func reserveOperationOnConnection(
	ctx context.Context, conn *sql.Conn, req ServiceOperationRequest, now time.Time,
) (result ServiceOperationRecord, replay bool, resultErr error) {
	if _, beginErr := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); beginErr != nil {
		return ServiceOperationRecord{}, false, ErrServiceOperationUnavailable
	}
	committed := false
	defer func() {
		rollbackServiceOperationTransaction(conn, committed, &resultErr)
	}()
	if policyErr := validateManagedPolicyInTransaction(ctx, conn, req); policyErr != nil {
		return ServiceOperationRecord{}, false, policyErr
	}
	created, rowErr := reserveOperationRow(ctx, conn, req, now)
	if rowErr != nil {
		return ServiceOperationRecord{}, false, rowErr
	}
	record, readErr := readServiceOperation(ctx, conn, req.Scope, req.OperationID)
	if readErr != nil {
		return ServiceOperationRecord{}, false, readErr
	}
	if !sameOperationRequest(record.ServiceOperationRequest, req) {
		return ServiceOperationRecord{}, false, ErrServiceOperationConflict
	}
	if _, commitErr := conn.ExecContext(ctx, "COMMIT"); commitErr != nil {
		return ServiceOperationRecord{}, false, ErrServiceOperationUnavailable
	}
	committed = true
	return record, !created, nil
}

// validateManagedPolicyInTransaction reads active operator policy from the
// exact SQLite transaction/connection that reserves or replays the operation.
// BEGIN IMMEDIATE serializes policy updates, so disabling the capability
// cannot race between this check and the row INSERT/lookup.
func validateManagedPolicyInTransaction(
	ctx context.Context, conn *sql.Conn, req ServiceOperationRequest,
) error {
	var empty [sha256.Size]byte
	if req.ManagedSchemaSHA256 == empty {
		return nil // legacy C2 inert in-memory registry mode only
	}
	var active int
	err := conn.QueryRowContext(ctx, `SELECT COUNT(*)
        FROM service_s2s_capability_policy c
        JOIN service_principal p ON p.id=c.principal_id
        WHERE c.principal_id=? AND c.workspace_id=?
          AND c.capability_key=? AND c.schema_version=?
          AND c.handler_id=? AND c.schema_sha256=?
          AND c.enabled=1 AND p.consumer='dubbridge' AND p.status='active'
          AND p.revoked_at_unix IS NULL AND p.workspace_id=c.workspace_id`,
		req.Scope.PrincipalID, req.Scope.WorkspaceID,
		req.CapabilityKey, req.SchemaVersion,
		req.ManagedHandlerID, req.ManagedSchemaSHA256[:]).Scan(&active)
	if err != nil {
		return ErrServiceOperationUnavailable
	}
	if active != 1 {
		return ErrServiceOperationDenied
	}
	return nil
}

func rollbackServiceOperationTransaction(conn *sql.Conn, committed bool, resultErr *error) {
	if committed {
		return
	}
	if _, rollbackErr := conn.ExecContext(context.Background(), "ROLLBACK"); rollbackErr != nil {
		*resultErr = errors.Join(*resultErr, ErrServiceOperationUnavailable,
			fmt.Errorf("rollback service operation transaction: %w", rollbackErr))
	}
}

func validOperationScope(scope ServiceOperationScope) bool {
	return validScopePart(scope.PrincipalID) && validScopePart(scope.WorkspaceID) &&
		validScopePart(scope.OrgID) && (scope.ProjectID == "" || validScopePart(scope.ProjectID))
}

func validOperationReservation(req ServiceOperationRequest, now time.Time) bool {
	if !validOperationScope(req.Scope) ||
		!b2cOperationIDPattern.MatchString(req.OperationID) ||
		!b2cTraceIDPattern.MatchString(req.TraceID) ||
		req.CapabilityKey == "" || req.SchemaVersion == "" || now.Unix() <= 0 {
		return false
	}
	var empty [sha256.Size]byte
	return subtle.ConstantTimeCompare(req.RequestSHA256[:], empty[:]) != 1
}

func reserveOperationRow(ctx context.Context, conn *sql.Conn, req ServiceOperationRequest, now time.Time) (bool, error) {
	// INSERT SELECT always derives scope from Fenix's current persisted principal.
	// No user-provided organization/project is copied to a new operation row.
	res, err := conn.ExecContext(ctx, `INSERT INTO service_operation
        (principal_id,operation_id,workspace_id,dubbridge_org_id,dubbridge_project_id,
         capability_key,schema_version,request_sha256,config_sha256,trace_id,
         created_at_unix,updated_at_unix)
        SELECT p.id,?,p.workspace_id,p.dubbridge_org_id,COALESCE(p.dubbridge_project_id,''),
               ?,?,?,?,?,?,?
        FROM service_principal p
        WHERE p.id=? AND p.consumer='dubbridge' AND p.status='active'
          AND p.revoked_at_unix IS NULL
          AND p.workspace_id=? AND p.dubbridge_org_id=?
          AND COALESCE(p.dubbridge_project_id,'')=?
        ON CONFLICT(principal_id,operation_id) DO NOTHING`,
		req.OperationID, req.CapabilityKey, req.SchemaVersion,
		req.RequestSHA256[:], req.ConfigSHA256[:], req.TraceID,
		now.UTC().Unix(), now.UTC().Unix(),
		req.Scope.PrincipalID, req.Scope.WorkspaceID, req.Scope.OrgID, req.Scope.ProjectID)
	if err != nil {
		return false, ErrServiceOperationUnavailable
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, ErrServiceOperationUnavailable
	}
	return n == 1, nil
}

func readServiceOperation(
	ctx context.Context, conn *sql.Conn, scope ServiceOperationScope, operationID string,
) (ServiceOperationRecord, error) {
	var record ServiceOperationRecord
	var requestHash, configHash []byte
	var runID, leaseOwner sql.NullString
	var leaseUntil sql.NullInt64
	err := conn.QueryRowContext(ctx, `SELECT o.principal_id,o.operation_id,o.workspace_id,
        o.dubbridge_org_id,o.dubbridge_project_id,o.capability_key,o.schema_version,
        o.request_sha256,o.config_sha256,o.trace_id,o.status,o.run_id,
        o.lease_owner,o.lease_until_unix,o.created_at_unix,o.updated_at_unix
        FROM service_operation o
        JOIN service_principal p ON o.principal_id=p.id
        WHERE o.principal_id=? AND o.operation_id=? AND o.workspace_id=?
          AND o.dubbridge_org_id=? AND o.dubbridge_project_id=?
          AND p.status='active' AND p.revoked_at_unix IS NULL AND p.consumer='dubbridge'
          AND p.workspace_id=o.workspace_id AND p.dubbridge_org_id=o.dubbridge_org_id
          AND COALESCE(p.dubbridge_project_id,'')=o.dubbridge_project_id`,
		scope.PrincipalID, operationID, scope.WorkspaceID, scope.OrgID, scope.ProjectID).Scan(
		&record.Scope.PrincipalID, &record.OperationID, &record.Scope.WorkspaceID,
		&record.Scope.OrgID, &record.Scope.ProjectID,
		&record.CapabilityKey, &record.SchemaVersion, &requestHash, &configHash,
		&record.TraceID, &record.Status, &runID, &leaseOwner, &leaseUntil,
		&record.CreatedAtUnix, &record.UpdatedAtUnix,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ServiceOperationRecord{}, ErrServiceOperationDenied
	}
	if err != nil {
		return ServiceOperationRecord{}, ErrServiceOperationUnavailable
	}
	if !hydrateServiceOperationRecord(&record, requestHash, configHash, runID, leaseOwner, leaseUntil) {
		return ServiceOperationRecord{}, ErrServiceOperationUnavailable
	}
	return record, nil
}

func hydrateServiceOperationRecord(
	record *ServiceOperationRecord, requestHash, configHash []byte,
	runID, leaseOwner sql.NullString, leaseUntil sql.NullInt64,
) bool {
	if len(requestHash) != sha256.Size || len(configHash) != sha256.Size {
		return false
	}
	copy(record.RequestSHA256[:], requestHash)
	copy(record.ConfigSHA256[:], configHash)
	if runID.Valid {
		record.RunID = runID.String
	}
	if leaseOwner.Valid {
		record.LeaseOwner = leaseOwner.String
	}
	if leaseUntil.Valid {
		record.LeaseUntilUnix = leaseUntil.Int64
	}
	return true
}

func serviceCapabilityFingerprint(binding S2SCapabilityBinding) [sha256.Size]byte {
	// Preserve B2.C legacy in-memory hash while binding persisted B2.B4
	// reservations to the immutable versioned schema digest.
	identity := binding.PrincipalID + "\x00" + binding.WorkspaceID + "\x00" +
		binding.CapabilityKey + "\x00" + binding.InputSchemaVersion + "\x00" + binding.HandlerID
	raw := []byte(identity)
	var zero [sha256.Size]byte
	if binding.SchemaSHA256 != zero {
		raw = append(raw, 0)
		raw = append(raw, binding.SchemaSHA256[:]...)
	}
	return sha256.Sum256(raw)
}

func sameOperationRequest(a, b ServiceOperationRequest) bool {
	return a.Scope == b.Scope &&
		a.OperationID == b.OperationID &&
		a.TraceID == b.TraceID &&
		a.CapabilityKey == b.CapabilityKey &&
		a.SchemaVersion == b.SchemaVersion &&
		subtle.ConstantTimeCompare(a.RequestSHA256[:], b.RequestSHA256[:]) == 1 &&
		subtle.ConstantTimeCompare(a.ConfigSHA256[:], b.ConfigSHA256[:]) == 1
}
