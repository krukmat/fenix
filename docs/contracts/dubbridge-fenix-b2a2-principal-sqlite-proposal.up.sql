-- B2.A2 PROPOSAL ONLY. DO NOT place under internal/infra/sqlite/migrations/
-- until security peer review + integrated Fenix Go/SQLite QA.
-- Requires the existing workspace table and PRAGMA foreign_keys=ON.
CREATE TABLE service_principal (
    id TEXT PRIMARY KEY NOT NULL,
    consumer TEXT NOT NULL CHECK (consumer = 'dubbridge'),
    workspace_id TEXT NOT NULL REFERENCES workspace(id) ON DELETE RESTRICT,
    dubbridge_org_id TEXT NOT NULL,
    dubbridge_project_id TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    created_at_unix INTEGER NOT NULL CHECK (created_at_unix > 0),
    revoked_at_unix INTEGER,
    CHECK ((status = 'active' AND revoked_at_unix IS NULL) OR (status = 'revoked' AND revoked_at_unix IS NOT NULL))
);
CREATE INDEX idx_service_principal_workspace_org ON service_principal(workspace_id, dubbridge_org_id);

CREATE TABLE service_credential (
    id TEXT PRIMARY KEY NOT NULL,
    principal_id TEXT NOT NULL REFERENCES service_principal(id) ON DELETE RESTRICT,
    token_sha256 BLOB NOT NULL UNIQUE CHECK (length(token_sha256) = 32),
    issued_at_unix INTEGER NOT NULL CHECK (issued_at_unix > 0),
    expires_at_unix INTEGER NOT NULL CHECK (expires_at_unix > issued_at_unix),
    revoked_at_unix INTEGER,
    CHECK (revoked_at_unix IS NULL OR revoked_at_unix >= issued_at_unix)
);
CREATE INDEX idx_service_credential_principal ON service_credential(principal_id);
