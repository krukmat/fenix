-- Migration 032: Relationship Memory Engine schema (Task B.1)
-- Introduces stakeholder relationship cognition layer (ADR-101).
-- CRM entity references are loose (no FK) to survive CRM record deletion.

CREATE TABLE IF NOT EXISTS relationship_memory (
    id           TEXT     NOT NULL PRIMARY KEY,
    workspace_id TEXT     NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    -- Loose CRM reference — no FK; survives contact/deal/case deletion
    entity_type  TEXT     NOT NULL
                          CHECK(entity_type IN ('account', 'contact', 'lead', 'deal', 'case')),
    entity_id    TEXT     NOT NULL,
    summary      TEXT     NOT NULL DEFAULT '',
    inferred_intent TEXT,
    tone         TEXT     CHECK(tone IN ('positive', 'neutral', 'negative', 'mixed')),
    trajectory   TEXT     CHECK(trajectory IN ('improving', 'stable', 'declining')),
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, entity_type, entity_id)
);

CREATE INDEX IF NOT EXISTS idx_relationship_memory_workspace
    ON relationship_memory(workspace_id);

CREATE INDEX IF NOT EXISTS idx_relationship_memory_entity
    ON relationship_memory(workspace_id, entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_relationship_memory_tone
    ON relationship_memory(workspace_id, tone);

-- Each CRM interaction (email, call, meeting, note, etc.) produces one signal
CREATE TABLE IF NOT EXISTS interaction_signal (
    id                     TEXT     NOT NULL PRIMARY KEY,
    relationship_memory_id TEXT     NOT NULL REFERENCES relationship_memory(id) ON DELETE CASCADE,
    signal_type            TEXT     NOT NULL
                                    CHECK(signal_type IN ('email', 'call', 'meeting', 'note', 'case_update', 'deal_update')),
    sentiment              TEXT     CHECK(sentiment IN ('positive', 'neutral', 'negative')),
    summary                TEXT     NOT NULL DEFAULT '',
    -- Loose back-reference to the originating CRM record (no FK)
    source_entity_type     TEXT,
    source_entity_id       TEXT,
    occurred_at            DATETIME NOT NULL,
    created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_interaction_signal_memory
    ON interaction_signal(relationship_memory_id);

CREATE INDEX IF NOT EXISTS idx_interaction_signal_memory_type
    ON interaction_signal(relationship_memory_id, signal_type);

CREATE INDEX IF NOT EXISTS idx_interaction_signal_memory_occurred
    ON interaction_signal(relationship_memory_id, occurred_at DESC);

-- Directed influence edge between two CRM entities within a workspace
CREATE TABLE IF NOT EXISTS stakeholder_graph (
    id               TEXT     NOT NULL PRIMARY KEY,
    workspace_id     TEXT     NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    from_entity_type TEXT     NOT NULL
                              CHECK(from_entity_type IN ('account', 'contact', 'lead', 'deal', 'case')),
    from_entity_id   TEXT     NOT NULL,
    to_entity_type   TEXT     NOT NULL
                              CHECK(to_entity_type IN ('account', 'contact', 'lead', 'deal', 'case')),
    to_entity_id     TEXT     NOT NULL,
    influence_type   TEXT     NOT NULL
                              CHECK(influence_type IN ('reports_to', 'influences', 'blocks', 'collaborates', 'approves')),
    strength         REAL     NOT NULL DEFAULT 0.5
                              CHECK(strength >= 0.0 AND strength <= 1.0),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_stakeholder_graph_workspace
    ON stakeholder_graph(workspace_id);

CREATE INDEX IF NOT EXISTS idx_stakeholder_graph_from
    ON stakeholder_graph(workspace_id, from_entity_type, from_entity_id);

CREATE INDEX IF NOT EXISTS idx_stakeholder_graph_to
    ON stakeholder_graph(workspace_id, to_entity_type, to_entity_id);

-- One trust_score per relationship_memory (1:1 enforced by UNIQUE)
CREATE TABLE IF NOT EXISTS trust_score (
    id                     TEXT     NOT NULL PRIMARY KEY,
    relationship_memory_id TEXT     NOT NULL UNIQUE REFERENCES relationship_memory(id) ON DELETE CASCADE,
    score                  REAL     NOT NULL DEFAULT 0.5
                                    CHECK(score >= 0.0 AND score <= 1.0),
    confidence             TEXT     NOT NULL DEFAULT 'low'
                                    CHECK(confidence IN ('high', 'medium', 'low')),
    decay_factor           REAL     NOT NULL DEFAULT 1.0
                                    CHECK(decay_factor >= 0.0 AND decay_factor <= 1.0),
    last_scored_at         DATETIME NOT NULL,
    created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_trust_score_memory
    ON trust_score(relationship_memory_id);

CREATE INDEX IF NOT EXISTS idx_trust_score_score
    ON trust_score(score DESC);
