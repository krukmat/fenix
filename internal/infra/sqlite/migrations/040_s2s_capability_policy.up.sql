-- B2.B4: durable, versioned S2S *admission* policy, not ToolRegistry execution.
-- Provisioned records start disabled; no production capability is created.
CREATE TABLE service_s2s_capability_policy (
    principal_id        TEXT NOT NULL REFERENCES service_principal(id) ON DELETE RESTRICT,
    workspace_id        TEXT NOT NULL REFERENCES workspace(id) ON DELETE RESTRICT,
    capability_key      TEXT NOT NULL CHECK (length(capability_key) BETWEEN 3 AND 96),
    schema_version      TEXT NOT NULL CHECK (length(schema_version) BETWEEN 1 AND 3),
    handler_id          TEXT NOT NULL CHECK (length(handler_id) BETWEEN 1 AND 128),
    schema_json         TEXT NOT NULL CHECK (json_valid(schema_json) AND length(schema_json) <= 65536),
    schema_sha256       BLOB NOT NULL CHECK (length(schema_sha256) = 32),
    enabled             INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    revision            INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    updated_by          TEXT NOT NULL CHECK (length(updated_by) BETWEEN 1 AND 128),
    created_at_unix     INTEGER NOT NULL CHECK (created_at_unix > 0),
    updated_at_unix     INTEGER NOT NULL CHECK (updated_at_unix >= created_at_unix),
    PRIMARY KEY(principal_id,workspace_id,capability_key,schema_version)
);
-- A principal can enable only one schema version per logical capability.
CREATE UNIQUE INDEX idx_service_s2s_capability_active
    ON service_s2s_capability_policy(principal_id,workspace_id,capability_key)
    WHERE enabled = 1;

CREATE TABLE service_s2s_capability_audit (
    id                INTEGER PRIMARY KEY,
    principal_id      TEXT NOT NULL,
    workspace_id      TEXT NOT NULL,
    capability_key    TEXT NOT NULL,
    schema_version    TEXT NOT NULL,
    action            TEXT NOT NULL CHECK (action IN ('provision','enable','disable')),
    revision          INTEGER NOT NULL CHECK (revision > 0),
    actor_id          TEXT NOT NULL,
    occurred_at_unix  INTEGER NOT NULL CHECK (occurred_at_unix > 0),
    schema_sha256     BLOB NOT NULL CHECK (length(schema_sha256) = 32)
);
CREATE INDEX idx_service_s2s_capability_audit_scope
    ON service_s2s_capability_audit(principal_id,workspace_id,capability_key,id);

CREATE TRIGGER service_s2s_capability_insert_guard
BEFORE INSERT ON service_s2s_capability_policy
BEGIN
    SELECT CASE WHEN NEW.enabled != 0 OR NEW.revision != 1 OR NOT EXISTS (
        SELECT 1 FROM service_principal p
        WHERE p.id = NEW.principal_id AND p.consumer = 'dubbridge'
          AND p.status = 'active' AND p.revoked_at_unix IS NULL
          AND p.workspace_id = NEW.workspace_id
    ) THEN RAISE(ABORT,'service capability provision denied') END;
END;

CREATE TRIGGER service_s2s_capability_immutable_fields
BEFORE UPDATE ON service_s2s_capability_policy
WHEN NEW.principal_id IS NOT OLD.principal_id
  OR NEW.workspace_id IS NOT OLD.workspace_id
  OR NEW.capability_key IS NOT OLD.capability_key
  OR NEW.schema_version IS NOT OLD.schema_version
  OR NEW.handler_id IS NOT OLD.handler_id
  OR NEW.schema_json IS NOT OLD.schema_json
  OR NEW.schema_sha256 IS NOT OLD.schema_sha256
  OR NEW.created_at_unix IS NOT OLD.created_at_unix
  OR NEW.revision != OLD.revision + 1
  OR NEW.enabled = OLD.enabled
  OR NEW.updated_at_unix < OLD.updated_at_unix
BEGIN SELECT RAISE(ABORT,'service capability revision invalid'); END;

CREATE TRIGGER service_s2s_capability_enable_guard
BEFORE UPDATE OF enabled ON service_s2s_capability_policy
WHEN NEW.enabled = 1 AND NOT EXISTS (
    SELECT 1 FROM service_principal p
    WHERE p.id = NEW.principal_id AND p.consumer = 'dubbridge'
      AND p.status = 'active' AND p.revoked_at_unix IS NULL
      AND p.workspace_id = NEW.workspace_id
)
BEGIN SELECT RAISE(ABORT,'service capability principal inactive'); END;

CREATE TRIGGER service_s2s_capability_audit_insert
AFTER INSERT ON service_s2s_capability_policy
BEGIN
    INSERT INTO service_s2s_capability_audit(
        principal_id,workspace_id,capability_key,schema_version,
        action,revision,actor_id,occurred_at_unix,schema_sha256)
    VALUES(NEW.principal_id,NEW.workspace_id,NEW.capability_key,NEW.schema_version,
           'provision',NEW.revision,NEW.updated_by,NEW.updated_at_unix,NEW.schema_sha256);
END;

CREATE TRIGGER service_s2s_capability_audit_update
AFTER UPDATE ON service_s2s_capability_policy
BEGIN
    INSERT INTO service_s2s_capability_audit(
        principal_id,workspace_id,capability_key,schema_version,
        action,revision,actor_id,occurred_at_unix,schema_sha256)
    VALUES(NEW.principal_id,NEW.workspace_id,NEW.capability_key,NEW.schema_version,
           CASE WHEN NEW.enabled = 1 THEN 'enable' ELSE 'disable' END,
           NEW.revision,NEW.updated_by,NEW.updated_at_unix,NEW.schema_sha256);
END;

CREATE TRIGGER service_s2s_capability_no_delete
BEFORE DELETE ON service_s2s_capability_policy
BEGIN SELECT RAISE(ABORT,'service capability policy immutable'); END;

CREATE TRIGGER service_s2s_capability_audit_no_update
BEFORE UPDATE ON service_s2s_capability_audit
BEGIN SELECT RAISE(ABORT,'service capability audit append-only'); END;

CREATE TRIGGER service_s2s_capability_audit_no_delete
BEFORE DELETE ON service_s2s_capability_audit
BEGIN SELECT RAISE(ABORT,'service capability audit append-only'); END;
