-- Migration 021: Agent runtime steps for FR-230

CREATE TABLE IF NOT EXISTS agent_run_step (
    id            TEXT PRIMARY KEY,
    workspace_id  TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    agent_run_id  TEXT NOT NULL REFERENCES agent_run(id) ON DELETE CASCADE,
    step_index    INTEGER NOT NULL,
    step_type     TEXT NOT NULL, -- retrieve_evidence|reason|tool_call|finalize
    status        TEXT NOT NULL DEFAULT 'pending', -- pending|running|success|failed|skipped|retrying
    attempt       INTEGER NOT NULL DEFAULT 1,
    input         JSON,
    output        JSON,
    error         TEXT,
    started_at    DATETIME,
    completed_at  DATETIME,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_run_id, step_index, attempt)
);

CREATE INDEX IF NOT EXISTS idx_agent_run_step_run
    ON agent_run_step(agent_run_id);

CREATE INDEX IF NOT EXISTS idx_agent_run_step_workspace
    ON agent_run_step(workspace_id);

CREATE INDEX IF NOT EXISTS idx_agent_run_step_status
    ON agent_run_step(status);

CREATE INDEX IF NOT EXISTS idx_agent_run_step_type
    ON agent_run_step(step_type);
