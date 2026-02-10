-- Migration 009: Auth index — composite lookup for login
-- Task 1.6.5: Adds composite index (workspace_id, email) on user_account.
-- Justification: login queries filter by email globally (unique), but future
-- multi-workspace support or admin queries will filter by (workspace_id, email).
-- The existing idx_user_account_email covers single-column lookups.
-- This index covers the join pattern: WHERE workspace_id = ? AND email = ?

CREATE INDEX IF NOT EXISTS idx_user_account_workspace_email
    ON user_account (workspace_id, email);
