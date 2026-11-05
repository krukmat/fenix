package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"time"
)

// ServiceCredentialStore is Fenix's backend for nonhuman credentials. The
// caller supplies a Fenix-owned *sql.DB configured for SQLite FK enforcement.
// It is not a user account store and never accepts a human JWT.
type ServiceCredentialStore struct{ db *sql.DB }

var (
	ErrServiceIdentityDenied      = errors.New("invalid service credentials")
	ErrServiceIdentityUnavailable = errors.New("service identity unavailable")
	ErrServiceIdentityConfig      = errors.New("invalid service identity configuration")
	ErrServiceIdentityNotFound    = errors.New("service identity not found or inactive")
)

// ServiceIdentity contains the verified principal and the authenticated scope.
// The DubBridge organization and project are always read from Fenix's registry,
// never copied from a client's request body or unverified JWT.
type ServiceIdentity struct {
	ServicePrincipal
	DubBridgeOrgID     string
	DubBridgeProjectID string
}

// NewServiceCredentialStore wires the store but never migrates a database,
// provisions a token or activates any service endpoint.
func NewServiceCredentialStore(db *sql.DB) *ServiceCredentialStore {
	return &ServiceCredentialStore{db: db}
}

// RegisterServicePrincipal persists an operator-approved identity. It does not
// grant any capability; B2.B must bind an explicit allowlist before execution.
func (s *ServiceCredentialStore) RegisterServicePrincipal(ctx context.Context, identity ServiceIdentity, at time.Time) error {
	if s == nil || s.db == nil || identity.ID == "" || identity.Consumer != "dubbridge" ||
		identity.WorkspaceID == "" || identity.DubBridgeOrgID == "" || at.IsZero() || at.Unix() <= 0 {
		return ErrServiceIdentityConfig
	}
	var project any
	if identity.DubBridgeProjectID != "" {
		project = identity.DubBridgeProjectID
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO service_principal
         (id,consumer,workspace_id,dubbridge_org_id,dubbridge_project_id,created_at_unix)
         VALUES (?,?,?,?,?,?)`, identity.ID, identity.Consumer, identity.WorkspaceID,
		identity.DubBridgeOrgID, project, at.UTC().Unix())
	if err != nil {
		return ErrServiceIdentityUnavailable
	}
	return nil
}

// StoreServiceCredential stores only a digest. The bearer string must never be
// passed to this function or written into Fenix's SQLite database.
func (s *ServiceCredentialStore) StoreServiceCredential(ctx context.Context,
	credentialID, principalID string, digest ServiceTokenDigest,
	issuedAt, expiresAt time.Time) error {
	if s == nil || s.db == nil || credentialID == "" || principalID == "" ||
		issuedAt.IsZero() || expiresAt.IsZero() ||
		issuedAt.Unix() <= 0 || expiresAt.Unix() <= issuedAt.Unix() ||
		subtle.ConstantTimeCompare(digest[:], make([]byte, len(digest))) == 1 {
		return ErrServiceIdentityConfig
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO service_credential
         (id,principal_id,token_sha256,issued_at_unix,expires_at_unix)
         SELECT ?,p.id,?,?,? FROM service_principal p
         WHERE p.id=? AND p.consumer='dubbridge' AND p.status='active'`,
		credentialID, digest[:], issuedAt.UTC().Unix(), expiresAt.UTC().Unix(), principalID)
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

// AuthenticateServiceToken verifies a canonical opaque bearer against the
// digest-only store, active principal, revocation and expiry in one read.
// A match NEVER implies authorization for a requested tenant/capability: the
// service middleware must compare the trusted stored scope with the request.
func (s *ServiceCredentialStore) AuthenticateServiceToken(ctx context.Context, bearer string, now time.Time) (*ServiceIdentity, error) {
	if s == nil || s.db == nil {
		return nil, ErrServiceIdentityUnavailable
	}
	digest, err := DigestServiceToken(bearer)
	if err != nil || now.IsZero() {
		return nil, ErrServiceIdentityDenied
	}

	var out ServiceIdentity
	var project sql.NullString
	var stored []byte
	err = s.db.QueryRowContext(ctx,
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
	if len(stored) != len(digest) || subtle.ConstantTimeCompare(stored, digest[:]) != 1 ||
		!VerifyServiceToken(bearer, digest) || out.ID == "" || out.WorkspaceID == "" || out.DubBridgeOrgID == "" {
		return nil, ErrServiceIdentityDenied
	}
	if project.Valid {
		out.DubBridgeProjectID = project.String
	}
	return &out, nil
}

// RevokeServiceCredential invalidates a live credential; repeated revocation
// returns ErrServiceIdentityNotFound without leaking which principal owned it.
func (s *ServiceCredentialStore) RevokeServiceCredential(ctx context.Context, credentialID string, at time.Time) error {
	if s == nil || s.db == nil || credentialID == "" || at.IsZero() || at.Unix() <= 0 {
		return ErrServiceIdentityConfig
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE service_credential SET revoked_at_unix=?
         WHERE id=? AND revoked_at_unix IS NULL AND issued_at_unix<=?`,
		at.UTC().Unix(), credentialID, at.UTC().Unix())
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

// RevokeServicePrincipal immediately denies all of the principal's existing
// tokens. No human user or workspace data are touched.
func (s *ServiceCredentialStore) RevokeServicePrincipal(ctx context.Context, principalID string, at time.Time) error {
	if s == nil || s.db == nil || principalID == "" || at.IsZero() || at.Unix() <= 0 {
		return ErrServiceIdentityConfig
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE service_principal SET status='revoked',revoked_at_unix=?
         WHERE id=? AND status='active' AND created_at_unix<=?`,
		at.UTC().Unix(), principalID, at.UTC().Unix())
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
