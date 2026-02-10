-- Migration 002: CRM Accounts
-- Task 1.3.1: account table (CRM core entity)
-- Represents a customer or organization account in FenixCRM

CREATE TABLE IF NOT EXISTS account (
    id            TEXT    NOT NULL PRIMARY KEY,          -- UUID v7
    workspace_id  TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    name          TEXT    NOT NULL,                      -- Organization name
    domain        TEXT,                                  -- Domain (example.com)
    industry      TEXT,                                  -- Industry vertical
    size_segment  TEXT    CHECK (size_segment IN ('smb', 'mid', 'enterprise')),
                                                         -- Market segment
    owner_id      TEXT    NOT NULL REFERENCES user_account(id) ON DELETE SET NULL,
                                                         -- Account owner (nullable on user delete)
    address       TEXT,                                  -- JSON: { "street", "city", "postal_code", "country" }
    metadata      TEXT,                                  -- JSON: extensible attributes
    created_at    TEXT    NOT NULL,                      -- ISO 8601 UTC
    updated_at    TEXT    NOT NULL,
    deleted_at    TEXT                                   -- NULL = active, populated = soft deleted
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_account_workspace     ON account (workspace_id);
CREATE INDEX IF NOT EXISTS idx_account_owner         ON account (owner_id);
CREATE INDEX IF NOT EXISTS idx_account_deleted       ON account (workspace_id, deleted_at);
                                                         -- For ListAccounts (filters deleted_at IS NULL)
CREATE INDEX IF NOT EXISTS idx_account_created       ON account (workspace_id, created_at DESC);
                                                         -- For ListAccounts with ordering

-- Constraint: account name must be unique within a workspace (upsert-safe)
CREATE UNIQUE INDEX IF NOT EXISTS idx_account_workspace_name
    ON account (workspace_id, name)
    WHERE deleted_at IS NULL;
                                                         -- Only active accounts, allows deleted + recreated
