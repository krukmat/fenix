-- Migration 025: enforce one active workflow per workspace + name

CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_one_active_per_name
    ON workflow(workspace_id, name)
    WHERE status = 'active';
