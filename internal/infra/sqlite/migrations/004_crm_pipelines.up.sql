-- Migration 004: CRM Pipelines and Stages
-- Task 1.5: pipeline and pipeline_stage tables
-- Pipelines define stages for deals and cases

-- Pipeline: container for stages (can be for deals or cases)
CREATE TABLE IF NOT EXISTS pipeline (
    id            TEXT    NOT NULL PRIMARY KEY,          -- UUID v7
    workspace_id  TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    name          TEXT    NOT NULL,                      -- Pipeline name
    entity_type   TEXT    NOT NULL
                         CHECK (entity_type IN ('deal', 'case')),
                                                         -- What this pipeline is for
    settings      TEXT,                                  -- JSON: stage_colors, automation_rules, etc.
    created_at    TEXT    NOT NULL,                      -- ISO 8601 UTC
    updated_at    TEXT    NOT NULL
);

-- Pipeline Stage: individual stage within a pipeline
CREATE TABLE IF NOT EXISTS pipeline_stage (
    id               TEXT    NOT NULL PRIMARY KEY,       -- UUID v7
    pipeline_id      TEXT    NOT NULL REFERENCES pipeline(id) ON DELETE CASCADE,
    name             TEXT    NOT NULL,                   -- Stage name (e.g., "Qualified", "Negotiation")
    position         INTEGER NOT NULL DEFAULT 0,         -- Order within pipeline (0, 1, 2...)
    probability      REAL,                               -- Win probability % (for deals, e.g., 0.25 = 25%)
    sla_hours        INTEGER,                            -- SLA target for this stage (optional)
    required_fields  TEXT,                               -- JSON: array of required field names
    created_at       TEXT    NOT NULL,                   -- ISO 8601 UTC
    updated_at       TEXT    NOT NULL,

    UNIQUE (pipeline_id, position)                       -- No duplicate positions within a pipeline
);

-- Indexes for pipelines
CREATE INDEX IF NOT EXISTS idx_pipeline_workspace      ON pipeline (workspace_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_entity_type    ON pipeline (workspace_id, entity_type);
CREATE INDEX IF NOT EXISTS idx_pipeline_created        ON pipeline (workspace_id, created_at DESC);

-- Constraint: pipeline name must be unique within a workspace
CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_workspace_name
    ON pipeline (workspace_id, name);

-- Indexes for pipeline stages
CREATE INDEX IF NOT EXISTS idx_pipeline_stage_pipeline     ON pipeline_stage (pipeline_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_stage_position     ON pipeline_stage (pipeline_id, position);

-- Constraint: stage name must be unique within a pipeline
CREATE UNIQUE INDEX IF NOT EXISTS idx_pipeline_stage_name
    ON pipeline_stage (pipeline_id, name);
