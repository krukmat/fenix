-- Task 2.4: vec_embedding table stores float32 vectors as JSON TEXT.
-- Uses standard SQLite (no sqlite-vec extension) to remain pure-Go compatible.
-- Each row shares its id with the corresponding embedding_document row.
-- Multi-tenant: workspace_id on every row (mandatory for Task 2.5 vector search isolation).
CREATE TABLE IF NOT EXISTS vec_embedding (
    id           TEXT     PRIMARY KEY,                -- Same as embedding_document.id
    workspace_id TEXT     NOT NULL,                   -- CRITICAL: multi-tenant isolation
    embedding    TEXT     NOT NULL,                   -- JSON float32 array: "[0.1,0.2,...]"
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id) REFERENCES embedding_document(id),
    FOREIGN KEY (workspace_id) REFERENCES workspace(id)
);

-- Index for workspace-scoped vector scans (Task 2.5: cosine similarity queries)
CREATE INDEX IF NOT EXISTS idx_vec_workspace ON vec_embedding(workspace_id);
