-- Reference copy of local migration 038. No production migration is applied here.
-- Service identities are inert until an operator-controlled S2S route is enabled.
CREATE TABLE service_principal (
    id TEXT PRIMARY KEY NOT NULL,
    consumer TEXT NOT NULL CHECK (consumer = 'dubbridge'),
    workspace_id TEXT NOT NULL REFERENCES workspace(id) ON DELETE RESTRICT,
    dubbridge_org_id TEXT NOT NULL,
    dubbridge_project_id TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    capability_allowlist_json TEXT NOT NULL DEFAULT '[]' CHECK (capability_allowlist_json = '[]'),
    created_at_unix INTEGER NOT NULL CHECK (created_at_unix > 0),
    revoked_at_unix INTEGER CHECK (revoked_at_unix > 0),
    CHECK ((status = 'active' AND revoked_at_unix IS NULL) OR
           (status = 'revoked' AND revoked_at_unix IS NOT NULL))
);
CREATE INDEX idx_service_principal_workspace_org
    ON service_principal(workspace_id, dubbridge_org_id);

CREATE TABLE service_credential (
    id TEXT PRIMARY KEY NOT NULL,
    principal_id TEXT NOT NULL REFERENCES service_principal(id) ON DELETE RESTRICT,
    version INTEGER NOT NULL CHECK (version > 0),
    token_sha256 BLOB NOT NULL UNIQUE CHECK (length(token_sha256) = 32),
    issued_at_unix INTEGER NOT NULL CHECK (issued_at_unix > 0),
    expires_at_unix INTEGER NOT NULL CHECK (expires_at_unix > issued_at_unix),
    revoked_at_unix INTEGER,
    UNIQUE (principal_id, version),
    CHECK (revoked_at_unix IS NULL OR revoked_at_unix >= issued_at_unix)
);
CREATE INDEX idx_service_credential_principal ON service_credential(principal_id);
CREATE UNIQUE INDEX idx_service_credential_one_active
    ON service_credential(principal_id) WHERE revoked_at_unix IS NULL;

-- Append-only lifecycle linkage. No bearer or digest is recorded here.
CREATE TABLE service_identity_audit (
    id INTEGER PRIMARY KEY,
    principal_id TEXT NOT NULL REFERENCES service_principal(id) ON DELETE RESTRICT,
    credential_id TEXT REFERENCES service_credential(id) ON DELETE RESTRICT,
    action TEXT NOT NULL CHECK (action IN
        ('register_principal', 'store_credential', 'rotate_credential',
         'revoke_credential', 'revoke_principal')),
    occurred_at_unix INTEGER NOT NULL CHECK (occurred_at_unix > 0)
);
CREATE INDEX idx_service_identity_audit_principal
    ON service_identity_audit(principal_id, id);
CREATE TRIGGER service_identity_audit_no_update
    BEFORE UPDATE ON service_identity_audit
    BEGIN SELECT RAISE(ABORT, 'service identity audit is append-only'); END;
CREATE TRIGGER service_identity_audit_no_delete
    BEFORE DELETE ON service_identity_audit
    BEGIN SELECT RAISE(ABORT, 'service identity audit is append-only'); END;
