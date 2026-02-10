-- Migration 003: CRM Contacts
-- Task 1.4: contact table (CRM core entity)

CREATE TABLE IF NOT EXISTS contact (
    id            TEXT    NOT NULL PRIMARY KEY,          -- UUID v7
    workspace_id  TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    account_id    TEXT    NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    first_name    TEXT    NOT NULL,
    last_name     TEXT    NOT NULL,
    email         TEXT,
    phone         TEXT,
    title         TEXT,
    status        TEXT    NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active', 'inactive', 'churned')),
    owner_id      TEXT    NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
    metadata      TEXT,                                  -- JSON blob
    created_at    TEXT    NOT NULL,                      -- ISO 8601 UTC
    updated_at    TEXT    NOT NULL,
    deleted_at    TEXT                                   -- soft delete
);

CREATE INDEX IF NOT EXISTS idx_contact_workspace      ON contact (workspace_id);
CREATE INDEX IF NOT EXISTS idx_contact_account        ON contact (workspace_id, account_id);
CREATE INDEX IF NOT EXISTS idx_contact_owner          ON contact (workspace_id, owner_id);
CREATE INDEX IF NOT EXISTS idx_contact_deleted        ON contact (workspace_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_contact_created        ON contact (workspace_id, created_at DESC);
