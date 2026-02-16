-- Migration 019: Add 'escalated' to case_ticket status CHECK constraint.
-- Task 3.8: Handoff Manager requires case status = 'escalated' when AI agent escalates.
--
-- SQLite does not support ALTER TABLE ... MODIFY COLUMN, so we recreate the table.
-- The migration runner wraps each file in its own transaction — no explicit BEGIN/COMMIT.
-- PRAGMA foreign_keys cannot run inside a transaction; no FK constraints reference
-- case_ticket from other tables, so this rename/recreate is safe without disabling FKs.

ALTER TABLE case_ticket RENAME TO case_ticket_old;

CREATE TABLE IF NOT EXISTS case_ticket (
    id               TEXT    PRIMARY KEY,
    workspace_id     TEXT    NOT NULL,
    account_id       TEXT,
    contact_id       TEXT,
    pipeline_id      TEXT,
    stage_id         TEXT,
    owner_id         TEXT    NOT NULL,
    subject          TEXT    NOT NULL,
    description      TEXT,
    priority         TEXT    NOT NULL DEFAULT 'medium'
                         CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    status           TEXT    NOT NULL DEFAULT 'open'
                         CHECK (status IN ('open', 'in_progress', 'waiting', 'resolved', 'closed', 'escalated')),
    channel          TEXT,
    sla_config       TEXT,
    sla_deadline     TEXT,
    metadata         TEXT,
    created_at       TEXT    NOT NULL,
    updated_at       TEXT    NOT NULL,
    deleted_at       TEXT
);

INSERT INTO case_ticket SELECT * FROM case_ticket_old;

DROP TABLE case_ticket_old;

CREATE INDEX IF NOT EXISTS idx_case_workspace     ON case_ticket (workspace_id);
CREATE INDEX IF NOT EXISTS idx_case_account       ON case_ticket (workspace_id, account_id);
CREATE INDEX IF NOT EXISTS idx_case_contact       ON case_ticket (workspace_id, contact_id);
CREATE INDEX IF NOT EXISTS idx_case_owner         ON case_ticket (workspace_id, owner_id);
CREATE INDEX IF NOT EXISTS idx_case_status        ON case_ticket (workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_case_pipeline      ON case_ticket (workspace_id, pipeline_id, stage_id);
CREATE INDEX IF NOT EXISTS idx_case_open_priority ON case_ticket (workspace_id, status, priority, created_at DESC)
    WHERE deleted_at IS NULL AND status IN ('open', 'in_progress', 'waiting', 'escalated');
