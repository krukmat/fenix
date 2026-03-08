CREATE TABLE IF NOT EXISTS prompt_experiment (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspace(id),
    agent_definition_id TEXT NOT NULL,
    control_prompt_version_id TEXT NOT NULL REFERENCES prompt_version(id) ON DELETE CASCADE,
    candidate_prompt_version_id TEXT NOT NULL REFERENCES prompt_version(id) ON DELETE CASCADE,
    control_traffic_percent INTEGER NOT NULL,
    candidate_traffic_percent INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'running', 'completed', 'cancelled')),
    winner_prompt_version_id TEXT REFERENCES prompt_version(id) ON DELETE SET NULL,
    created_by TEXT REFERENCES user_account(id),
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_prompt_experiment_agent_status
    ON prompt_experiment(workspace_id, agent_definition_id, status);
