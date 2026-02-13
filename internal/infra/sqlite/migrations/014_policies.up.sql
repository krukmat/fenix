-- Migration 014: Policy Engine foundation (Task 3.1)
-- Adds policy_set and policy_version for RBAC/ABAC policy management.

CREATE TABLE IF NOT EXISTS policy_set (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    description   TEXT,
    is_active     INTEGER NOT NULL DEFAULT 1,
    created_by    TEXT,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_policy_set_workspace ON policy_set(workspace_id);
CREATE INDEX IF NOT EXISTS idx_policy_set_active ON policy_set(workspace_id, is_active);

CREATE TABLE IF NOT EXISTS policy_version (
    id                TEXT PRIMARY KEY,
    policy_set_id     TEXT NOT NULL REFERENCES policy_set(id) ON DELETE CASCADE,
    workspace_id      TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    version_number    INTEGER NOT NULL,
    policy_json       JSON NOT NULL,
    status            TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft', 'active', 'archived')),
    created_by        TEXT,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(policy_set_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_policy_version_set ON policy_version(policy_set_id);
CREATE INDEX IF NOT EXISTS idx_policy_version_workspace ON policy_version(workspace_id);
CREATE INDEX IF NOT EXISTS idx_policy_version_status ON policy_version(workspace_id, status);
