-- Rollback Migration 029: usage and quota domain foundation

DROP INDEX IF EXISTS idx_quota_state_workspace_policy;
DROP TABLE IF EXISTS quota_state;

DROP INDEX IF EXISTS idx_quota_policy_workspace_metric;
DROP INDEX IF EXISTS idx_quota_policy_workspace_active;
DROP TABLE IF EXISTS quota_policy;

DROP INDEX IF EXISTS idx_usage_event_workspace_actor;
DROP INDEX IF EXISTS idx_usage_event_workspace_run;
DROP INDEX IF EXISTS idx_usage_event_workspace_created_at;
DROP TABLE IF EXISTS usage_event;
