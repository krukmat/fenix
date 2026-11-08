package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"time"
)

// ServiceCredentialStore uses a Fenix-owned SQLite DB with foreign keys enabled.
// Its caller is responsible for operator authorization and the DB lifecycle.
type ServiceCredentialStore struct{ db *sql.DB }

var (
	ErrServiceIdentityDenied      = errors.New("invalid service credentials")
	ErrServiceIdentityUnavailable = errors.New("service identity unavailable")
	ErrServiceIdentityConfig      = errors.New("invalid service identity configuration")
	ErrServiceIdentityNotFound    = errors.New("service identity not found or inactive")
)

// ServiceIdentity contains a verified principal and its Fenix-owned source scope.
type ServiceIdentity struct {
	ServicePrincipal
	DubBridgeOrgID     string
	DubBridgeProjectID string
}

// NewServiceCredentialStore never migrates, provisions or activates a route.
func NewServiceCredentialStore(db *sql.DB) *ServiceCredentialStore {
	return &ServiceCredentialStore{db: db}
}

// RegisterServicePrincipal persists an operator-approved identity with no capabilities.
func (s *ServiceCredentialStore) RegisterServicePrincipal(ctx context.Context, identity ServiceIdentity, at time.Time) error {
	if !s.ready() || !validServiceIdentity(identity, at) {
		return ErrServiceIdentityConfig
	}
	var project any
	if identity.DubBridgeProjectID != "" {
		project = identity.DubBridgeProjectID
	}
	return s.write(ctx, func(conn *sql.Conn) error {
		_, err := conn.ExecContext(ctx,
			`INSERT INTO service_principal
             (id,consumer,workspace_id,dubbridge_org_id,dubbridge_project_id,created_at_unix)
             VALUES (?,?,?,?,?,?)`, identity.ID, identity.Consumer, identity.WorkspaceID,
			identity.DubBridgeOrgID, project, at.UTC().Unix())
		if err != nil {
			return ErrServiceIdentityUnavailable
		}
		return appendServiceAudit(ctx, conn, identity.ID, nil, "register_principal", at)
	})
}

// StoreServiceCredential stores the first digest for an active principal.
// A later credential must use RotateServiceCredential; plaintext is never stored.
func (s *ServiceCredentialStore) StoreServiceCredential(ctx context.Context,
	credentialID, principalID string, digest ServiceTokenDigest,
	issuedAt, expiresAt time.Time) error {
	if !s.ready() || !validCredential(credentialID, principalID, digest, issuedAt, expiresAt) {
		return ErrServiceIdentityConfig
	}
	return s.write(ctx, func(conn *sql.Conn) error {
		res, err := conn.ExecContext(ctx,
			`INSERT INTO service_credential
             (id,principal_id,version,token_sha256,issued_at_unix,expires_at_unix)
             SELECT ?,p.id,1,?,?,? FROM service_principal p
             WHERE p.id=? AND p.consumer='dubbridge' AND p.status='active'
               AND NOT EXISTS (SELECT 1 FROM service_credential c WHERE c.principal_id=p.id)`,
			credentialID, digest[:], issuedAt.UTC().Unix(), expiresAt.UTC().Unix(), principalID)
		if changeErr := changedOne(res, err); changeErr != nil {
			return changeErr
		}
		return appendServiceAudit(ctx, conn, principalID, credentialID, "store_credential", issuedAt)
	})
}

// RotateServiceCredential atomically revokes the current digest and inserts the
// next version. An insertion or audit failure rolls back the revocation.
func (s *ServiceCredentialStore) RotateServiceCredential(ctx context.Context,
	credentialID, principalID string, digest ServiceTokenDigest,
	issuedAt, expiresAt time.Time) error {
	if !s.ready() || !validCredential(credentialID, principalID, digest, issuedAt, expiresAt) {
		return ErrServiceIdentityConfig
	}
	return s.write(ctx, func(conn *sql.Conn) error {
		res, err := conn.ExecContext(ctx,
			`UPDATE service_credential SET revoked_at_unix=max(?,issued_at_unix)
             WHERE principal_id=? AND revoked_at_unix IS NULL
               AND EXISTS (SELECT 1 FROM service_principal p
                           WHERE p.id=? AND p.consumer='dubbridge' AND p.status='active')`,
			issuedAt.UTC().Unix(), principalID, principalID)
		if changeErr := changedOne(res, err); changeErr != nil {
			return changeErr
		}
		res, err = conn.ExecContext(ctx,
			`INSERT INTO service_credential
             (id,principal_id,version,token_sha256,issued_at_unix,expires_at_unix)
             SELECT ?,p.id,COALESCE((SELECT MAX(c.version) FROM service_credential c
                                     WHERE c.principal_id=p.id),0)+1,?,?,?
             FROM service_principal p WHERE p.id=? AND p.consumer='dubbridge' AND p.status='active'`,
			credentialID, digest[:], issuedAt.UTC().Unix(), expiresAt.UTC().Unix(), principalID)
		if changeErr := changedOne(res, err); changeErr != nil {
			return changeErr
		}
		return appendServiceAudit(ctx, conn, principalID, credentialID, "rotate_credential", issuedAt)
	})
}

// AuthenticateServiceToken verifies digest, principal status, issue time and expiry.
// It does not authorize any requested tenant or capability.
func (s *ServiceCredentialStore) AuthenticateServiceToken(ctx context.Context, bearer string, now time.Time) (*ServiceIdentity, error) {
	if !s.ready() {
		return nil, ErrServiceIdentityUnavailable
	}
	digest, err := DigestServiceToken(bearer)
	if err != nil || !validTime(now) {
		return nil, ErrServiceIdentityDenied
	}
	return s.loadIdentity(ctx, bearer, digest, now)
}

func (s *ServiceCredentialStore) loadIdentity(ctx context.Context, bearer string, digest ServiceTokenDigest, now time.Time) (*ServiceIdentity, error) {
	var out ServiceIdentity
	var project sql.NullString
	var stored []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT p.id,p.consumer,p.workspace_id,p.dubbridge_org_id,p.dubbridge_project_id,c.token_sha256
         FROM service_credential c JOIN service_principal p ON c.principal_id=p.id
         WHERE c.token_sha256=? AND c.revoked_at_unix IS NULL
           AND c.issued_at_unix<=? AND c.expires_at_unix>?
           AND p.status='active' AND p.revoked_at_unix IS NULL
           AND p.consumer='dubbridge'`, digest[:], now.UTC().Unix(), now.UTC().Unix()).Scan(
		&out.ID, &out.Consumer, &out.WorkspaceID, &out.DubBridgeOrgID, &project, &stored)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrServiceIdentityDenied
	}
	if err != nil {
		return nil, ErrServiceIdentityUnavailable
	}
	if !validStoredIdentity(out, stored, digest, bearer) {
		return nil, ErrServiceIdentityDenied
	}
	if project.Valid {
		out.DubBridgeProjectID = project.String
	}
	return &out, nil
}

func validStoredIdentity(out ServiceIdentity, stored []byte, digest ServiceTokenDigest, bearer string) bool {
	return len(stored) == len(digest) && subtle.ConstantTimeCompare(stored, digest[:]) == 1 &&
		VerifyServiceToken(bearer, digest) && out.ID != "" && out.WorkspaceID != "" && out.DubBridgeOrgID != ""
}

// RevokeServiceCredential denies a live credential even under operator clock skew.
func (s *ServiceCredentialStore) RevokeServiceCredential(ctx context.Context, credentialID string, at time.Time) error {
	if !s.ready() || credentialID == "" || !validTime(at) {
		return ErrServiceIdentityConfig
	}
	return s.write(ctx, func(conn *sql.Conn) error {
		res, err := conn.ExecContext(ctx,
			`UPDATE service_credential SET revoked_at_unix=max(?,issued_at_unix)
             WHERE id=? AND revoked_at_unix IS NULL`, at.UTC().Unix(), credentialID)
		if changeErr := changedOne(res, err); changeErr != nil {
			return changeErr
		}
		return appendCredentialRevokeAudit(ctx, conn, credentialID, at)
	})
}

// RevokeServicePrincipal immediately denies all credentials for this principal.
func (s *ServiceCredentialStore) RevokeServicePrincipal(ctx context.Context, principalID string, at time.Time) error {
	if !s.ready() || principalID == "" || !validTime(at) {
		return ErrServiceIdentityConfig
	}
	return s.write(ctx, func(conn *sql.Conn) error {
		res, err := conn.ExecContext(ctx,
			`UPDATE service_principal SET status='revoked',revoked_at_unix=?
             WHERE id=? AND status='active'`, at.UTC().Unix(), principalID)
		if changeErr := changedOne(res, err); changeErr != nil {
			return changeErr
		}
		return appendServiceAudit(ctx, conn, principalID, nil, "revoke_principal", at)
	})
}

func (s *ServiceCredentialStore) ready() bool { return s != nil && s.db != nil }
func validTime(at time.Time) bool             { return !at.IsZero() && at.Unix() > 0 }
func validServiceIdentity(identity ServiceIdentity, at time.Time) bool {
	return identity.ID != "" && identity.Consumer == "dubbridge" &&
		identity.WorkspaceID != "" && identity.DubBridgeOrgID != "" && validTime(at)
}
func validCredential(id, principalID string, digest ServiceTokenDigest, issuedAt, expiresAt time.Time) bool {
	return id != "" && principalID != "" && validTime(issuedAt) && expiresAt.Unix() > issuedAt.Unix() &&
		subtle.ConstantTimeCompare(digest[:], make([]byte, len(digest))) != 1
}

func changedOne(res sql.Result, err error) error {
	if err != nil {
		return ErrServiceIdentityUnavailable
	}
	n, err := res.RowsAffected()
	if err != nil {
		return ErrServiceIdentityUnavailable
	}
	if n != 1 {
		return ErrServiceIdentityNotFound
	}
	return nil
}

// write uses an explicit SQLite write lock so competing rotations serialize
// before either reads or changes credential state.
func (s *ServiceCredentialStore) write(ctx context.Context, action func(*sql.Conn) error) (resultErr error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return ErrServiceIdentityUnavailable
	}
	defer func() { _ = conn.Close() }()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return ErrServiceIdentityUnavailable
	}
	committed := false
	defer rollbackServiceWrite(conn, &committed, &resultErr)
	if err = action(conn); err != nil {
		return err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return ErrServiceIdentityUnavailable
	}
	committed = true
	return nil
}

func rollbackServiceWrite(conn *sql.Conn, committed *bool, resultErr *error) {
	if *committed {
		return
	}
	if _, err := conn.ExecContext(context.Background(), "ROLLBACK"); err != nil {
		*resultErr = ErrServiceIdentityUnavailable
	}
}

func appendServiceAudit(ctx context.Context, conn *sql.Conn, principalID string, credentialID any, action string, at time.Time) error {
	_, err := conn.ExecContext(ctx,
		`INSERT INTO service_identity_audit(principal_id,credential_id,action,occurred_at_unix)
         VALUES (?,?,?,?)`, principalID, credentialID, action, at.UTC().Unix())
	if err != nil {
		return ErrServiceIdentityUnavailable
	}
	return nil
}

func appendCredentialRevokeAudit(ctx context.Context, conn *sql.Conn, credentialID string, at time.Time) error {
	res, err := conn.ExecContext(ctx,
		`INSERT INTO service_identity_audit(principal_id,credential_id,action,occurred_at_unix)
         SELECT principal_id,id,'revoke_credential',? FROM service_credential WHERE id=?`,
		at.UTC().Unix(), credentialID)
	return changedOne(res, err)
}
