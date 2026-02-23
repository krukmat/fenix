-- 020_eval.up.sql
-- Task 4.7: FR-242 Eval Service Basic — eval_suite + eval_run tables

CREATE TABLE IF NOT EXISTS eval_suite (
    id           TEXT NOT NULL PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    domain       TEXT NOT NULL CHECK (domain IN ('support', 'sales', 'general')),
    test_cases   TEXT NOT NULL DEFAULT '[]',  -- JSON: [{input, expected_keywords, should_abstain}]
    thresholds   TEXT NOT NULL DEFAULT '{}',  -- JSON: {groundedness, exactitude, abstention, policy}
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_eval_suite_workspace
    ON eval_suite(workspace_id);

CREATE TABLE IF NOT EXISTS eval_run (
    id                TEXT NOT NULL PRIMARY KEY,
    workspace_id      TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    eval_suite_id     TEXT NOT NULL REFERENCES eval_suite(id) ON DELETE CASCADE,
    prompt_version_id TEXT,               -- nullable: references prompt_version if provided
    status            TEXT NOT NULL DEFAULT 'running'
                          CHECK (status IN ('running', 'passed', 'failed')),
    scores            TEXT NOT NULL DEFAULT '{}',  -- JSON: {groundedness, exactitude, abstention, policy_adherence}
    details           TEXT NOT NULL DEFAULT '[]',  -- JSON: per-test-case results
    triggered_by      TEXT,               -- nullable: user_account.id
    started_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at      DATETIME,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_eval_run_workspace ON eval_run(workspace_id);
CREATE INDEX IF NOT EXISTS idx_eval_run_suite     ON eval_run(eval_suite_id);
CREATE INDEX IF NOT EXISTS idx_eval_run_status    ON eval_run(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_eval_run_prompt    ON eval_run(prompt_version_id);