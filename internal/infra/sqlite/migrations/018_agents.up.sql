-- Migration 018: Agent definitions, skills, and runs (Task 3.7)
-- Agent Orchestrator + Support Agent UC-C1

-- Agent Definition: defines agents (support, prospecting, kb, insights, custom)
CREATE TABLE IF NOT EXISTS agent_definition (
    id                   TEXT PRIMARY KEY,
    workspace_id         TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    name                 TEXT NOT NULL,
    description          TEXT,
    agent_type           TEXT NOT NULL, -- support, prospecting, kb, insights, custom
    objective            JSON, -- agent objective/goals
    allowed_tools        JSON NOT NULL DEFAULT '[]', -- array of tool_definition IDs
    limits               JSON NOT NULL DEFAULT '{}', -- max_tokens_day, max_cost_day, max_runs_day
    trigger_config       JSON NOT NULL DEFAULT '{}', -- event|schedule|manual
    policy_set_id        TEXT REFERENCES policy_set(id) ON DELETE SET NULL,
    active_prompt_version_id TEXT REFERENCES prompt_version(id) ON DELETE SET NULL,
    status               TEXT NOT NULL DEFAULT 'active', -- active|paused|deprecated
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_agent_definition_workspace
    ON agent_definition(workspace_id);

CREATE INDEX IF NOT EXISTS idx_agent_definition_workspace_status
    ON agent_definition(workspace_id, status);

CREATE INDEX IF NOT EXISTS idx_agent_definition_type
    ON agent_definition(agent_type);

-- Skill Definition: defines skills (ordered array of tool calls + conditions)
CREATE TABLE IF NOT EXISTS skill_definition (
    id                   TEXT PRIMARY KEY,
    workspace_id         TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    name                 TEXT NOT NULL,
    description          TEXT,
    steps                JSON NOT NULL, -- ordered array of tool calls + conditions
    agent_definition_id  TEXT REFERENCES agent_definition(id) ON DELETE CASCADE,
    status               TEXT NOT NULL DEFAULT 'draft', -- draft|active|deprecated
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_skill_definition_workspace
    ON skill_definition(workspace_id);

CREATE INDEX IF NOT EXISTS idx_skill_definition_agent
    ON skill_definition(agent_definition_id);

CREATE INDEX IF NOT EXISTS idx_skill_definition_status
    ON skill_definition(status);

-- Agent Run: state machine for agent executions
CREATE TABLE IF NOT EXISTS agent_run (
    id                   TEXT PRIMARY KEY,
    workspace_id         TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    agent_definition_id  TEXT NOT NULL REFERENCES agent_definition(id) ON DELETE CASCADE,
    triggered_by_user_id TEXT REFERENCES user_account(id) ON DELETE SET NULL,
    trigger_type         TEXT NOT NULL, -- event|schedule|manual|copilot
    trigger_context      JSON, -- event payload, entity ref
    status               TEXT NOT NULL DEFAULT 'running', -- running|success|partial|abstained|failed|escalated
    inputs               JSON, -- agent input data
    retrieval_queries    JSON, -- queries made to knowledge
    retrieved_evidence_ids JSON, -- IDs of evidence retrieved
    reasoning_trace      JSON, -- LLM reasoning steps
    tool_calls           JSON, -- array of tool call records
    output               JSON, -- final output
    abstention_reason    TEXT, -- why agent abstained
    total_tokens         INTEGER,
    total_cost           REAL,
    latency_ms           INTEGER,
    trace_id             TEXT,
    started_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at         DATETIME,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_run_workspace
    ON agent_run(workspace_id);

CREATE INDEX IF NOT EXISTS idx_agent_run_agent
    ON agent_run(agent_definition_id);

CREATE INDEX IF NOT EXISTS idx_agent_run_status
    ON agent_run(status);

CREATE INDEX IF NOT EXISTS idx_agent_run_triggered_by
    ON agent_run(triggered_by_user_id);

CREATE INDEX IF NOT EXISTS idx_agent_run_created_at
    ON agent_run(created_at);
