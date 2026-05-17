-- Migration 031: Blackboard cognitive workspace domain (Task A.1)
-- Introduces shared cognitive workspace for multi-agent coordination (ADR-100)

CREATE TABLE IF NOT EXISTS cognitive_workspace (
    id           TEXT    NOT NULL PRIMARY KEY,
    workspace_id TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    agent_run_id TEXT    REFERENCES agent_run(id) ON DELETE SET NULL,
    status       TEXT    NOT NULL DEFAULT 'active'
                         CHECK(status IN ('active', 'closed', 'expired')),
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at    DATETIME
);

CREATE INDEX IF NOT EXISTS idx_cognitive_workspace_workspace
    ON cognitive_workspace(workspace_id);

CREATE INDEX IF NOT EXISTS idx_cognitive_workspace_agent_run
    ON cognitive_workspace(agent_run_id);

CREATE INDEX IF NOT EXISTS idx_cognitive_workspace_status
    ON cognitive_workspace(workspace_id, status);

-- Append-only log of reasoning events published by agents (source of truth for replay)
CREATE TABLE IF NOT EXISTS reasoning_event (
    id                     TEXT     NOT NULL PRIMARY KEY,
    cognitive_workspace_id TEXT     NOT NULL REFERENCES cognitive_workspace(id) ON DELETE CASCADE,
    actor_agent_id         TEXT,
    event_type             TEXT     NOT NULL
                                    CHECK(event_type IN ('hypothesis', 'observation', 'risk', 'recommendation', 'intent')),
    payload                TEXT     NOT NULL DEFAULT '{}',
    created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reasoning_event_workspace
    ON reasoning_event(cognitive_workspace_id);

CREATE INDEX IF NOT EXISTS idx_reasoning_event_workspace_type
    ON reasoning_event(cognitive_workspace_id, event_type);

CREATE INDEX IF NOT EXISTS idx_reasoning_event_workspace_created
    ON reasoning_event(cognitive_workspace_id, created_at);

-- Hypotheses posted by agents, subject to confidence arbitration (Phase D)
CREATE TABLE IF NOT EXISTS signal_hypothesis (
    id                     TEXT     NOT NULL PRIMARY KEY,
    cognitive_workspace_id TEXT     NOT NULL REFERENCES cognitive_workspace(id) ON DELETE CASCADE,
    source_agent_id        TEXT,
    content                TEXT     NOT NULL,
    confidence             REAL     NOT NULL CHECK(confidence >= 0.0 AND confidence <= 1.0),
    status                 TEXT     NOT NULL DEFAULT 'open'
                                    CHECK(status IN ('open', 'accepted', 'rejected', 'superseded')),
    created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at            DATETIME
);

CREATE INDEX IF NOT EXISTS idx_signal_hypothesis_workspace
    ON signal_hypothesis(cognitive_workspace_id);

CREATE INDEX IF NOT EXISTS idx_signal_hypothesis_workspace_status
    ON signal_hypothesis(cognitive_workspace_id, status);

-- Shared key-value memory accessible by all agents within a workspace
CREATE TABLE IF NOT EXISTS agent_memory (
    id                     TEXT     NOT NULL PRIMARY KEY,
    cognitive_workspace_id TEXT     NOT NULL REFERENCES cognitive_workspace(id) ON DELETE CASCADE,
    key                    TEXT     NOT NULL,
    value                  TEXT     NOT NULL DEFAULT '{}',
    scope                  TEXT     NOT NULL DEFAULT 'session'
                                    CHECK(scope IN ('session', 'persistent')),
    expires_at             DATETIME,
    created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cognitive_workspace_id, key)
);

CREATE INDEX IF NOT EXISTS idx_agent_memory_workspace
    ON agent_memory(cognitive_workspace_id);

CREATE INDEX IF NOT EXISTS idx_agent_memory_workspace_scope
    ON agent_memory(cognitive_workspace_id, scope);

-- Opt-in attachment: agent_run can be linked to a cognitive workspace
ALTER TABLE agent_run ADD COLUMN cognitive_workspace_id TEXT
    REFERENCES cognitive_workspace(id) ON DELETE SET NULL;
