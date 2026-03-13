-- Migration 027: Scheduled jobs for AGENT_SPEC Phase 6

CREATE TABLE IF NOT EXISTS scheduled_job (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    job_type     TEXT NOT NULL CHECK(job_type IN ('workflow_resume')),
    payload      JSON NOT NULL DEFAULT '{}',
    execute_at   DATETIME NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'executed', 'cancelled')),
    source_id    TEXT NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    executed_at  DATETIME
);

CREATE INDEX IF NOT EXISTS idx_scheduled_job_due
    ON scheduled_job(status, execute_at);

CREATE INDEX IF NOT EXISTS idx_scheduled_job_workspace_source
    ON scheduled_job(workspace_id, source_id);

CREATE INDEX IF NOT EXISTS idx_scheduled_job_workspace_type
    ON scheduled_job(workspace_id, job_type);
