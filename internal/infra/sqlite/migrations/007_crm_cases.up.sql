-- Migration 007: CRM Cases (Support Tickets)
-- Task 1.5: case_ticket table (CRM core entity)
-- Represents customer support cases/tickets

CREATE TABLE IF NOT EXISTS case_ticket (
    id               TEXT    NOT NULL PRIMARY KEY,       -- UUID v7
    workspace_id     TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    account_id       TEXT    REFERENCES account(id) ON DELETE CASCADE,
                                                         -- Optional: linked account
    contact_id       TEXT    REFERENCES contact(id) ON DELETE SET NULL,
                                                         -- Optional: contact who reported
    pipeline_id      TEXT    REFERENCES pipeline(id) ON DELETE SET NULL,
                                                         -- Optional: support pipeline (for complex workflows)
    stage_id         TEXT    REFERENCES pipeline_stage(id) ON DELETE SET NULL,
                                                         -- Optional: current stage in pipeline
    owner_id         TEXT    NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
                                                         -- Assigned support agent
    subject          TEXT    NOT NULL,                   -- Case title/subject
    description      TEXT,                               -- Detailed description
    priority         TEXT    NOT NULL DEFAULT 'medium'
                         CHECK (priority IN ('low', 'medium', 'high', 'critical')),
                                                         -- Case priority
    status           TEXT    NOT NULL DEFAULT 'open'
                         CHECK (status IN ('open', 'in_progress', 'waiting', 'resolved', 'closed')),
                                                         -- Case lifecycle status
    channel          TEXT,                               -- Source channel (email, chat, phone, web)
    sla_config       TEXT,                               -- JSON: SLA rules for this case
    sla_deadline     TEXT,                               -- ISO 8601: when SLA expires
    metadata         TEXT,                               -- JSON: tags, category, etc.
    created_at       TEXT    NOT NULL,                   -- ISO 8601 UTC
    updated_at       TEXT    NOT NULL,
    deleted_at       TEXT                                -- NULL = active, populated = soft deleted
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_case_workspace          ON case_ticket (workspace_id);
CREATE INDEX IF NOT EXISTS idx_case_account            ON case_ticket (account_id) WHERE account_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_case_contact            ON case_ticket (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_case_pipeline           ON case_ticket (pipeline_id) WHERE pipeline_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_case_stage              ON case_ticket (stage_id) WHERE stage_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_case_owner              ON case_ticket (workspace_id, owner_id);
CREATE INDEX IF NOT EXISTS idx_case_priority           ON case_ticket (workspace_id, priority);
CREATE INDEX IF NOT EXISTS idx_case_status             ON case_ticket (workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_case_channel            ON case_ticket (workspace_id, channel);
CREATE INDEX IF NOT EXISTS idx_case_sla_deadline       ON case_ticket (workspace_id, sla_deadline) WHERE sla_deadline IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_case_deleted            ON case_ticket (workspace_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_case_created            ON case_ticket (workspace_id, created_at DESC);

-- Composite index for support queue (open cases by priority)
CREATE INDEX IF NOT EXISTS idx_case_queue
    ON case_ticket (workspace_id, status, priority, created_at DESC)
    WHERE deleted_at IS NULL AND status IN ('open', 'in_progress', 'waiting');
