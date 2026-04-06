-- Migration 029: usage and quota domain foundation

CREATE TABLE IF NOT EXISTS usage_event (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    actor_id       TEXT NOT NULL,
    actor_type     TEXT NOT NULL,
    run_id         TEXT REFERENCES agent_run(id) ON DELETE SET NULL,
    tool_name      TEXT,
    model_name     TEXT,
    input_units    INTEGER NOT NULL DEFAULT 0,
    output_units   INTEGER NOT NULL DEFAULT 0,
    estimated_cost REAL NOT NULL DEFAULT 0,
    latency_ms     INTEGER,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_usage_event_workspace_created_at
    ON usage_event(workspace_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_usage_event_workspace_run
    ON usage_event(workspace_id, run_id);

CREATE INDEX IF NOT EXISTS idx_usage_event_workspace_actor
    ON usage_event(workspace_id, actor_id, actor_type);

CREATE TABLE IF NOT EXISTS quota_policy (
    id               TEXT PRIMARY KEY,
    workspace_id     TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    policy_type      TEXT NOT NULL,
    scope_type       TEXT NOT NULL DEFAULT 'workspace',
    scope_id         TEXT,
    metric_name      TEXT NOT NULL,
    limit_value      REAL NOT NULL,
    reset_period     TEXT NOT NULL,
    enforcement_mode TEXT NOT NULL DEFAULT 'soft',
    is_active        INTEGER NOT NULL DEFAULT 1,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quota_policy_workspace_active
    ON quota_policy(workspace_id, is_active);

CREATE INDEX IF NOT EXISTS idx_quota_policy_workspace_metric
    ON quota_policy(workspace_id, metric_name);

CREATE TABLE IF NOT EXISTS quota_state (
    id               TEXT PRIMARY KEY,
    workspace_id     TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    quota_policy_id  TEXT NOT NULL REFERENCES quota_policy(id) ON DELETE CASCADE,
    current_value    REAL NOT NULL DEFAULT 0,
    period_start     DATETIME NOT NULL,
    period_end       DATETIME NOT NULL,
    last_event_at    DATETIME,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(quota_policy_id, period_start, period_end)
);

CREATE INDEX IF NOT EXISTS idx_quota_state_workspace_policy
    ON quota_state(workspace_id, quota_policy_id);
