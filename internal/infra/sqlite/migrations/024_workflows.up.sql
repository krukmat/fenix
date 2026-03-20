-- Migration 024: Workflow foundation for AGENT_SPEC Phase 2

CREATE TABLE IF NOT EXISTS workflow (
    id                   TEXT PRIMARY KEY,
    workspace_id         TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    agent_definition_id  TEXT REFERENCES agent_definition(id) ON DELETE SET NULL,
    parent_version_id    TEXT REFERENCES workflow(id) ON DELETE SET NULL,
    name                 TEXT NOT NULL,
    description          TEXT,
    dsl_source           TEXT NOT NULL,
    spec_source          TEXT,
    version              INTEGER NOT NULL DEFAULT 1,
    status               TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft', 'testing', 'active', 'archived')),
    created_by_user_id   TEXT REFERENCES user_account(id) ON DELETE SET NULL,
    archived_at          DATETIME,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, name, version)
);

CREATE INDEX IF NOT EXISTS idx_workflow_workspace
    ON workflow(workspace_id);

CREATE INDEX IF NOT EXISTS idx_workflow_workspace_status
    ON workflow(workspace_id, status);

CREATE INDEX IF NOT EXISTS idx_workflow_workspace_name
    ON workflow(workspace_id, name);

CREATE INDEX IF NOT EXISTS idx_workflow_agent_definition
    ON workflow(agent_definition_id);

CREATE INDEX IF NOT EXISTS idx_workflow_parent_version
    ON workflow(parent_version_id);
