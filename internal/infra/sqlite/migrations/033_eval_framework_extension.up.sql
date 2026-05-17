-- Migration 033: Deterministic eval framework extension (Task C.1)
-- Extends the eval foundation with benchmark_case, synthetic_org,
-- and structured replay provenance on eval_run.

CREATE TABLE IF NOT EXISTS synthetic_org (
    id           TEXT     NOT NULL PRIMARY KEY,
    workspace_id TEXT     NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    slug         TEXT     NOT NULL,
    name         TEXT     NOT NULL,
    version      INTEGER  NOT NULL DEFAULT 1 CHECK(version > 0),
    seed         INTEGER  NOT NULL DEFAULT 0,
    fixture_data TEXT     NOT NULL DEFAULT '{}',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (workspace_id, slug, version)
);

CREATE INDEX IF NOT EXISTS idx_synthetic_org_workspace
    ON synthetic_org(workspace_id);

CREATE INDEX IF NOT EXISTS idx_synthetic_org_workspace_slug
    ON synthetic_org(workspace_id, slug);

CREATE TABLE IF NOT EXISTS benchmark_case (
    id               TEXT     NOT NULL PRIMARY KEY,
    workspace_id     TEXT     NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    synthetic_org_id TEXT     REFERENCES synthetic_org(id) ON DELETE SET NULL,
    slug             TEXT     NOT NULL,
    name             TEXT     NOT NULL,
    domain           TEXT     NOT NULL CHECK (domain IN ('support', 'sales', 'general')),
    version          INTEGER  NOT NULL DEFAULT 1 CHECK(version > 0),
    input_payload    TEXT     NOT NULL DEFAULT '{}',
    expected_outcome TEXT     NOT NULL DEFAULT '{}',
    tags             TEXT     NOT NULL DEFAULT '[]',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (workspace_id, slug, version)
);

CREATE INDEX IF NOT EXISTS idx_benchmark_case_workspace
    ON benchmark_case(workspace_id);

CREATE INDEX IF NOT EXISTS idx_benchmark_case_workspace_domain
    ON benchmark_case(workspace_id, domain);

CREATE INDEX IF NOT EXISTS idx_benchmark_case_synthetic_org
    ON benchmark_case(synthetic_org_id);

ALTER TABLE eval_run ADD COLUMN benchmark_case_id TEXT
    REFERENCES benchmark_case(id) ON DELETE SET NULL;

ALTER TABLE eval_run ADD COLUMN synthetic_org_id TEXT
    REFERENCES synthetic_org(id) ON DELETE SET NULL;

ALTER TABLE eval_run ADD COLUMN source_agent_run_id TEXT
    REFERENCES agent_run(id) ON DELETE SET NULL;

ALTER TABLE eval_run ADD COLUMN source_cognitive_workspace_id TEXT
    REFERENCES cognitive_workspace(id) ON DELETE SET NULL;

ALTER TABLE eval_run ADD COLUMN source_trace_id TEXT;

ALTER TABLE eval_run ADD COLUMN replay_mode TEXT
    NOT NULL DEFAULT 'adhoc'
    CHECK (replay_mode IN ('adhoc', 'benchmark', 'replay'));

CREATE INDEX IF NOT EXISTS idx_eval_run_benchmark_case
    ON eval_run(benchmark_case_id);

CREATE INDEX IF NOT EXISTS idx_eval_run_synthetic_org
    ON eval_run(synthetic_org_id);

CREATE INDEX IF NOT EXISTS idx_eval_run_source_agent_run
    ON eval_run(source_agent_run_id);

CREATE INDEX IF NOT EXISTS idx_eval_run_source_cognitive_workspace
    ON eval_run(source_cognitive_workspace_id);

CREATE INDEX IF NOT EXISTS idx_eval_run_replay_mode
    ON eval_run(workspace_id, replay_mode);

CREATE INDEX IF NOT EXISTS idx_eval_run_source_trace
    ON eval_run(source_trace_id);
