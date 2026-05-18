-- Migration 036: join table for relationship interaction signal embeddings.

CREATE TABLE IF NOT EXISTS interaction_signal_embedding (
    id               TEXT     NOT NULL PRIMARY KEY,
    workspace_id     TEXT     NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    signal_id        TEXT     NOT NULL UNIQUE REFERENCES interaction_signal(id) ON DELETE CASCADE,
    vec_embedding_id TEXT     NOT NULL UNIQUE REFERENCES vec_embedding(id) ON DELETE CASCADE,
    dim              INTEGER  NOT NULL CHECK(dim > 0),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_interaction_signal_embedding_workspace
    ON interaction_signal_embedding(workspace_id);

CREATE INDEX IF NOT EXISTS idx_interaction_signal_embedding_signal
    ON interaction_signal_embedding(signal_id);
