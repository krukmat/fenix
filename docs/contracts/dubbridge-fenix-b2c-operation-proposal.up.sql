-- PROPOSAL ONLY (B2.C). Not part of Fenix embedded migrations.
-- Requires B2.A2 proposed service_principal in Fenix SQLite.
CREATE TABLE service_operation (
    principal_id       TEXT NOT NULL REFERENCES service_principal(id) ON DELETE RESTRICT,
    operation_id       TEXT NOT NULL CHECK (length(operation_id) BETWEEN 16 AND 96),
    workspace_id       TEXT NOT NULL REFERENCES workspace(id) ON DELETE RESTRICT,
    capability_key     TEXT NOT NULL,
    schema_version     TEXT NOT NULL,
    request_sha256     BLOB NOT NULL CHECK (length(request_sha256) = 32),
    trace_id           TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'accepted'
                          CHECK (status IN ('accepted','running','awaiting_approval',
                              'handed_off','succeeded','completed_with_warnings',
                              'abstained','failed','denied','indeterminate')),
    run_id             TEXT,
    lease_owner        TEXT,
    lease_until_unix   INTEGER,
    created_at_unix    INTEGER NOT NULL CHECK (created_at_unix > 0),
    updated_at_unix    INTEGER NOT NULL CHECK (updated_at_unix >= created_at_unix),
    PRIMARY KEY (principal_id, operation_id),
    CHECK (status NOT IN ('running','awaiting_approval','handed_off',
                        'succeeded','completed_with_warnings','abstained') OR run_id IS NOT NULL),
    CHECK ((lease_owner IS NULL AND lease_until_unix IS NULL) OR
           (lease_owner IS NOT NULL AND lease_until_unix IS NOT NULL))
);
CREATE INDEX idx_service_operation_workspace_created ON service_operation(workspace_id,created_at_unix);
CREATE INDEX idx_service_operation_status ON service_operation(status,lease_until_unix);
