-- Migration 005: CRM Leads
-- Task 1.5: lead table (CRM core entity)
-- Represents a potential customer before qualification

CREATE TABLE IF NOT EXISTS lead (
    id            TEXT    NOT NULL PRIMARY KEY,          -- UUID v7
    workspace_id  TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    contact_id    TEXT    REFERENCES contact(id) ON DELETE SET NULL,
                                                         -- Optional: linked contact if known
    account_id    TEXT    REFERENCES account(id) ON DELETE SET NULL,
                                                         -- Optional: linked account if known
    source        TEXT,                                  -- Lead source (e.g., "website", "referral", "trade_show")
    status        TEXT    NOT NULL DEFAULT 'new'
                         CHECK (status IN ('new', 'contacted', 'qualified', 'converted', 'lost')),
                                                         -- Lead lifecycle status
    owner_id      TEXT    NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
                                                         -- Assigned sales rep
    score         REAL,                                  -- Lead score (0-100, AI-calculated or manual)
    metadata      TEXT,                                  -- JSON: custom attributes, UTM params, etc.
    created_at    TEXT    NOT NULL,                      -- ISO 8601 UTC
    updated_at    TEXT    NOT NULL,
    deleted_at    TEXT                                   -- NULL = active, populated = soft deleted
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_lead_workspace          ON lead (workspace_id);
CREATE INDEX IF NOT EXISTS idx_lead_owner              ON lead (workspace_id, owner_id);
CREATE INDEX IF NOT EXISTS idx_lead_status             ON lead (workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_lead_source             ON lead (workspace_id, source);
CREATE INDEX IF NOT EXISTS idx_lead_contact            ON lead (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_lead_account            ON lead (account_id) WHERE account_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_lead_deleted            ON lead (workspace_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_lead_created            ON lead (workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lead_score              ON lead (workspace_id, score DESC) WHERE score IS NOT NULL;
