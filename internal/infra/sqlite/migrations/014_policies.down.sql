-- Migration 014 rollback: Policy Engine foundation

DROP INDEX IF EXISTS idx_policy_version_status;
DROP INDEX IF EXISTS idx_policy_version_workspace;
DROP INDEX IF EXISTS idx_policy_version_set;
DROP TABLE IF EXISTS policy_version;

DROP INDEX IF EXISTS idx_policy_set_active;
DROP INDEX IF EXISTS idx_policy_set_workspace;
DROP TABLE IF EXISTS policy_set;
