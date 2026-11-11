package integration

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"time"
)

// S2SCapabilityPolicyStore persists operator-provisioned ADMISSION policy.
// It does not register a ToolRegistry executor, grant tool permissions,
// provision secrets, enable HTTP routes or bypass runtime approval policy.
// S2SCapabilityOperatorAuthorizer delegates operator permission to the
// existing Fenix policy engine. It must verify the operator's own workspace
// matches the target, and CheckActionPermission("s2s_capabilities","manage").
// No operator identity is ever derived from DubBridge's bearer or JSON.
type S2SCapabilityOperatorAuthorizer interface {
	AuthorizeS2SPolicyChange(context.Context, string, string) error
}

type S2SCapabilityPolicyStore struct {
	db *sql.DB
	operator S2SCapabilityOperatorAuthorizer // nil = read-only/default deny
}

type S2SCapabilityPolicyRegistration struct {
	Binding    S2SCapabilityBinding
	SchemaJSON []byte
	OperatorID string
}

func NewS2SCapabilityPolicyStore(db *sql.DB) *S2SCapabilityPolicyStore {
	return &S2SCapabilityPolicyStore{db: db}
}

// NewS2SCapabilityPolicyStoreWithOperator is for trusted management code
// after Fenix RBAC/ABAC operator authorization has been wired.
// The normal S2S HTTP composition uses the READ-ONLY constructor above.
func NewS2SCapabilityPolicyStoreWithOperator(
	db *sql.DB, authorizer S2SCapabilityOperatorAuthorizer,
) *S2SCapabilityPolicyStore {
	return &S2SCapabilityPolicyStore{db: db, operator: authorizer}
}

// Provision only INSERTs a DISABLED version; schema/handler identity is
// immutable. Changing schema creates a separate version, never an overwrite.
func (s *S2SCapabilityPolicyStore) Provision(
	ctx context.Context, input S2SCapabilityPolicyRegistration, now time.Time,
) error {
	b := input.Binding
	if s == nil || s.db == nil || s.operator == nil || b.Enabled ||
		!validCapabilityBinding(b) || b.HandlerID == "" ||
		!validScopePart(input.OperatorID) || now.Unix() <= 0 {
		return ErrCapabilityConfiguration
	}
	if err := s.operator.AuthorizeS2SPolicyChange(ctx, input.OperatorID, b.WorkspaceID); err != nil {
		return ErrCapabilityConfiguration
	}
	canonical, digest, err := normalizeS2SCapabilitySchema(input.SchemaJSON)
	if err != nil {
		return ErrCapabilityConfiguration
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO service_s2s_capability_policy
        (principal_id,workspace_id,capability_key,schema_version,handler_id,
         schema_json,schema_sha256,enabled,revision,updated_by,created_at_unix,updated_at_unix)
        VALUES(?,?,?,?,?,?,?,0,1,?,?,?)`,
		b.PrincipalID, b.WorkspaceID, b.CapabilityKey, b.InputSchemaVersion,
		b.HandlerID, string(canonical), digest[:], input.OperatorID,
		now.UTC().Unix(), now.UTC().Unix())
	if err != nil {
		return ErrCapabilityConfiguration
	}
	return nil
}

// SetEnabled is an OPERATOR-ONLY code API. The caller must authorize the
// operator before calling it; no S2S caller/HTTP endpoint may invoke it.
// SQLite enforces an active principal on enable, single enabled schema
// version per capability and atomic append-only revision auditing.
func (s *S2SCapabilityPolicyStore) SetEnabled(
	ctx context.Context, principal, workspace, key, version string,
	enabled bool, operatorID string, now time.Time,
) error {
	if s == nil || s.db == nil || s.operator == nil || !validScopePart(principal) ||
		!validScopePart(workspace) || !validScopePart(operatorID) ||
		!capabilityKeyPattern.MatchString(key) ||
		!schemaVersionPattern.MatchString(version) || now.Unix() <= 0 {
		return ErrCapabilityConfiguration
	}
	if err := s.operator.AuthorizeS2SPolicyChange(ctx, operatorID, workspace); err != nil {
		return ErrCapabilityConfiguration
	}
	value := 0
	if enabled {
		value = 1
	}
	res, err := s.db.ExecContext(ctx, `UPDATE service_s2s_capability_policy
        SET enabled=?,revision=revision+1,updated_by=?,updated_at_unix=?
        WHERE principal_id=? AND workspace_id=? AND capability_key=?
          AND schema_version=? AND enabled<>?`,
		value, operatorID, now.UTC().Unix(), principal, workspace, key, version, value)
	if err != nil {
		return ErrCapabilityUnavailable
	}
	affected, err := res.RowsAffected()
	if err != nil || affected != 1 {
		return ErrCapabilityUnavailable
	}
	return nil
}

// ResolveEnabled reads policy from SQLite afresh and validates the current
// principal, schema digest and exact scope. The raw schema never crosses a
// public response; it is consumed solely by the trusted input validator.
func (s *S2SCapabilityPolicyStore) ResolveEnabled(
	ctx context.Context, principal, workspace, key, version string,
) (S2SCapabilityBinding, []byte, error) {
	if s == nil || s.db == nil || !validScopePart(principal) ||
		!validScopePart(workspace) || !capabilityKeyPattern.MatchString(key) ||
		!schemaVersionPattern.MatchString(version) {
		return S2SCapabilityBinding{}, nil, ErrCapabilityUnavailable
	}
	var handlerID string
	var schemaJSON string
	var storedDigest []byte
	err := s.db.QueryRowContext(ctx, `SELECT c.handler_id,c.schema_json,c.schema_sha256
        FROM service_s2s_capability_policy c
        JOIN service_principal p ON p.id=c.principal_id
        WHERE c.principal_id=? AND c.workspace_id=? AND c.capability_key=?
          AND c.schema_version=? AND c.enabled=1
          AND p.consumer='dubbridge' AND p.status='active'
          AND p.revoked_at_unix IS NULL AND p.workspace_id=c.workspace_id`,
		principal, workspace, key, version).Scan(&handlerID, &schemaJSON, &storedDigest)
	if err != nil || len(storedDigest) != sha256.Size {
		return S2SCapabilityBinding{}, nil, ErrCapabilityUnavailable
	}
	_, digest, err := normalizeS2SCapabilitySchema([]byte(schemaJSON))
	if err != nil || subtle.ConstantTimeCompare(digest[:], storedDigest) != 1 {
		return S2SCapabilityBinding{}, nil, ErrCapabilityUnavailable
	}
	binding := S2SCapabilityBinding{
		PrincipalID: principal, WorkspaceID: workspace,
		CapabilityKey: key, InputSchemaVersion: version,
		HandlerID: handlerID, SchemaSHA256: digest, Enabled: true,
	}
	if !validCapabilityBinding(binding) {
		return S2SCapabilityBinding{}, nil, ErrCapabilityUnavailable
	}
	return binding, []byte(schemaJSON), nil
}

// PersistedS2SInputValidator is an admission-only schema verifier. Runtime
// tool permissions and approvals remain exclusively in Fenix ToolRegistry.
type PersistedS2SInputValidator struct {
	Policies *S2SCapabilityPolicyStore
}

func NewPersistedS2SInputValidator(store *S2SCapabilityPolicyStore) *PersistedS2SInputValidator {
	return &PersistedS2SInputValidator{Policies: store}
}

func (v *PersistedS2SInputValidator) ValidateS2SInput(
	ctx context.Context, binding S2SCapabilityBinding, input map[string]any,
) error {
	if v == nil || v.Policies == nil {
		return ErrCapabilityUnavailable
	}
	current, schema, err := v.Policies.ResolveEnabled(ctx, binding.PrincipalID,
		binding.WorkspaceID, binding.CapabilityKey, binding.InputSchemaVersion)
	if err != nil || current != binding || !validateS2SInputSchema(schema, input) {
		return ErrCapabilityUnavailable
	}
	return nil
}
