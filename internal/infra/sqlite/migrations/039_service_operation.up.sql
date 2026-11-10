-- DB-FENIX-B2.C1: local Fenix-owned durable operations, inert until approved ingress.
-- The caller must not construct scope from incoming DubBridge JSON. On writes,
-- a trigger enforces equality with the active stored S2S principal as defense in depth.
CREATE TABLE service_operation (
    principal_id          TEXT NOT NULL REFERENCES service_principal(id) ON DELETE RESTRICT,
    operation_id          TEXT NOT NULL CHECK (length(operation_id) BETWEEN 16 AND 96
                                   AND operation_id NOT GLOB '*[^A-Za-z0-9_-]*'),
    workspace_id          TEXT NOT NULL REFERENCES workspace(id) ON DELETE RESTRICT,
    dubbridge_org_id      TEXT NOT NULL CHECK (length(dubbridge_org_id)>0),
    dubbridge_project_id  TEXT NOT NULL DEFAULT '',
    capability_key        TEXT NOT NULL CHECK (length(capability_key) BETWEEN 3 AND 96),
    schema_version        TEXT NOT NULL CHECK (length(schema_version) BETWEEN 1 AND 3),
    request_sha256        BLOB NOT NULL CHECK (length(request_sha256)=32),
    config_sha256         BLOB NOT NULL CHECK (length(config_sha256)=32),
    trace_id              TEXT NOT NULL CHECK (length(trace_id) BETWEEN 8 AND 96),
    status                TEXT NOT NULL DEFAULT 'accepted'
                          CHECK (status IN ('accepted','running','awaiting_approval',
                              'handed_off','succeeded','completed_with_warnings',
                              'abstained','failed','denied','indeterminate')),
    run_id                TEXT,
    lease_owner           TEXT,
    lease_until_unix      INTEGER,
    created_at_unix       INTEGER NOT NULL CHECK (created_at_unix>0),
    updated_at_unix       INTEGER NOT NULL CHECK (updated_at_unix>=created_at_unix),
    PRIMARY KEY(principal_id,operation_id),
    CHECK (status NOT IN ('running','awaiting_approval','handed_off',
                         'succeeded','completed_with_warnings','abstained') OR run_id IS NOT NULL),
    CHECK ((lease_owner IS NULL AND lease_until_unix IS NULL)
       OR (lease_owner IS NOT NULL AND length(lease_owner)>0 AND lease_until_unix>0))
);
CREATE INDEX idx_service_operation_workspace_created
    ON service_operation(workspace_id,created_at_unix);
CREATE INDEX idx_service_operation_status_lease
    ON service_operation(status,lease_until_unix);

CREATE TRIGGER service_operation_scope_insert
BEFORE INSERT ON service_operation
BEGIN
    SELECT CASE WHEN NOT EXISTS (
        SELECT 1 FROM service_principal p
         WHERE p.id=NEW.principal_id AND p.consumer='dubbridge'
           AND p.status='active' AND p.revoked_at_unix IS NULL
           AND p.workspace_id=NEW.workspace_id
           AND p.dubbridge_org_id=NEW.dubbridge_org_id
           AND COALESCE(p.dubbridge_project_id,'')=NEW.dubbridge_project_id
    ) THEN RAISE(ABORT,'service operation scope invalid') END;
END;

CREATE TRIGGER service_operation_immutable_identity
BEFORE UPDATE OF principal_id,operation_id,workspace_id,
    dubbridge_org_id,dubbridge_project_id,capability_key,schema_version,
    request_sha256,config_sha256,trace_id,created_at_unix
ON service_operation
BEGIN SELECT RAISE(ABORT,'service operation identity immutable'); END;

-- A terminal outcome, including crash ambiguity, may not be silently reset
-- to accepted. Future authorized recovery must create a distinct operation ID.
CREATE TRIGGER service_operation_terminal_state
BEFORE UPDATE OF status ON service_operation
WHEN OLD.status IN ('succeeded','completed_with_warnings','abstained',
                    'failed','denied','indeterminate') AND NEW.status<>OLD.status
BEGIN SELECT RAISE(ABORT,'terminal service operation state immutable'); END;
