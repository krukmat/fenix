-- Migration 009 rollback: Drop composite auth index
-- Task 1.6.5: Reverses 009_auth_index.up.sql

DROP INDEX IF EXISTS idx_user_account_workspace_email;
