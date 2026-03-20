-- Migration 026: Signal foundation for AGENT_SPEC Phase 2

CREATE TABLE IF NOT EXISTS signal (
    id                TEXT PRIMARY KEY,
    workspace_id      TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    entity_type       TEXT NOT NULL,
    entity_id         TEXT NOT NULL,
    signal_type       TEXT NOT NULL,
    confidence        REAL NOT NULL CHECK(confidence >= 0.0 AND confidence <= 1.0),
    evidence_ids      JSON NOT NULL DEFAULT '[]',
    source_type       TEXT NOT NULL,
    source_id         TEXT NOT NULL,
    metadata          JSON NOT NULL DEFAULT '{}',
    status            TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'dismissed', 'expired')),
    dismissed_by      TEXT REFERENCES user_account(id) ON DELETE SET NULL,
    dismissed_at      DATETIME,
    expires_at        DATETIME,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_signal_workspace
    ON signal(workspace_id);

CREATE INDEX IF NOT EXISTS idx_signal_workspace_status
    ON signal(workspace_id, status);

CREATE INDEX IF NOT EXISTS idx_signal_entity
    ON signal(workspace_id, entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_signal_type
    ON signal(workspace_id, signal_type);

CREATE INDEX IF NOT EXISTS idx_signal_source
    ON signal(workspace_id, source_type, source_id);

CREATE INDEX IF NOT EXISTS idx_signal_expires_at
    ON signal(expires_at);
