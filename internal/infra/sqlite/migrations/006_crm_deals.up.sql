-- Migration 006: CRM Deals
-- Task 1.5: deal table (CRM core entity)
-- Represents a sales opportunity tied to an account

CREATE TABLE IF NOT EXISTS deal (
    id               TEXT    NOT NULL PRIMARY KEY,       -- UUID v7
    workspace_id     TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    account_id       TEXT    NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    contact_id       TEXT    REFERENCES contact(id) ON DELETE SET NULL,
                                                         -- Primary contact for this deal
    pipeline_id      TEXT    NOT NULL REFERENCES pipeline(id) ON DELETE RESTRICT,
                                                         -- Cannot delete pipeline if deals exist
    stage_id         TEXT    NOT NULL REFERENCES pipeline_stage(id) ON DELETE RESTRICT,
                                                         -- Cannot delete stage if deals exist
    owner_id         TEXT    NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
                                                         -- Deal owner
    title            TEXT    NOT NULL,                   -- Deal name/title
    amount           REAL,                               -- Deal value
    currency         TEXT    DEFAULT 'USD',              -- ISO 4217 currency code
    expected_close   TEXT,                               -- Date (YYYY-MM-DD)
    status           TEXT    NOT NULL DEFAULT 'open'
                         CHECK (status IN ('open', 'won', 'lost')),
                                                         -- Deal outcome status
    metadata         TEXT,                               -- JSON: custom fields, products, etc.
    created_at       TEXT    NOT NULL,                   -- ISO 8601 UTC
    updated_at       TEXT    NOT NULL,
    deleted_at       TEXT                                -- NULL = active, populated = soft deleted
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_deal_workspace          ON deal (workspace_id);
CREATE INDEX IF NOT EXISTS idx_deal_account            ON deal (workspace_id, account_id);
CREATE INDEX IF NOT EXISTS idx_deal_contact            ON deal (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_deal_pipeline           ON deal (workspace_id, pipeline_id);
CREATE INDEX IF NOT EXISTS idx_deal_stage              ON deal (workspace_id, stage_id);
CREATE INDEX IF NOT EXISTS idx_deal_owner              ON deal (workspace_id, owner_id);
CREATE INDEX IF NOT EXISTS idx_deal_status             ON deal (workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_deal_expected_close     ON deal (workspace_id, expected_close);
CREATE INDEX IF NOT EXISTS idx_deal_deleted            ON deal (workspace_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_deal_created            ON deal (workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_deal_amount             ON deal (workspace_id, amount DESC) WHERE amount IS NOT NULL;

-- Composite index for pipeline board view (stage-based grouping)
CREATE INDEX IF NOT EXISTS idx_deal_pipeline_stage
    ON deal (workspace_id, pipeline_id, stage_id, status)
    WHERE deleted_at IS NULL;
