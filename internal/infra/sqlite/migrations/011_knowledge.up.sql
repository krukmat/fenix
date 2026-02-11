-- Migration 011: Knowledge Layer — Core Tables
-- Creates knowledge_item, embedding_document, evidence tables
-- Related to: Task 2.1, FR-090, FR-092
-- NOTE: FTS5 virtual table + triggers are in 012_knowledge_fts.up.sql
-- (split to avoid sqlc parser issues with CREATE VIRTUAL TABLE syntax)

-- ============================================================================
-- CORE TABLES
-- ============================================================================

-- knowledge_item: Ingested documents (emails, KB articles, API data, etc.)
-- Source of truth for the knowledge layer. FTS5 and vec_embedding are derived.
CREATE TABLE knowledge_item (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK(source_type IN ('email', 'document', 'kb_article', 'api', 'other')),
    title TEXT NOT NULL,
    raw_content TEXT NOT NULL,                   -- Original unmodified content
    normalized_content TEXT,                     -- HTML-stripped, lowercase for indexing
    entity_type TEXT,                            -- Optional: 'account', 'contact', 'case', 'deal', 'lead'
    entity_id TEXT,                              -- Optional: UUID of linked CRM entity
    metadata TEXT,                               -- JSON: author, source_url, tags, etc.
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,                         -- Soft delete

    FOREIGN KEY (workspace_id) REFERENCES workspace(id),
    UNIQUE(workspace_id, entity_type, entity_id) -- Prevent duplicate ingestion of same CRM entity
);

-- embedding_document: Chunks of a knowledge_item with embedding metadata
-- One knowledge_item -> N embedding_document rows (one per chunk)
-- The actual vector is stored in vec_embedding virtual table (same id)
CREATE TABLE embedding_document (
    id TEXT PRIMARY KEY,                         -- Also used as vec_embedding.id (must match)
    knowledge_item_id TEXT NOT NULL,
    workspace_id TEXT NOT NULL,                  -- CRITICAL: Required for multi-tenant vector search
    chunk_index INTEGER NOT NULL,                -- 0, 1, 2, ... within the document
    chunk_text TEXT NOT NULL,                    -- The actual text chunk (max ~512 tokens)
    token_count INTEGER,                         -- Approximate token count for cost tracking
    embedding_status TEXT NOT NULL DEFAULT 'pending'
        CHECK(embedding_status IN ('pending', 'embedded', 'failed')),
    embedded_at DATETIME,                        -- When the embedding was successfully created
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (knowledge_item_id) REFERENCES knowledge_item(id),
    FOREIGN KEY (workspace_id) REFERENCES workspace(id)
);

-- evidence: Search result snapshots — ranked results from BM25/Vector search
-- Used for audit trail and as input to evidence pack builder (Task 2.6)
CREATE TABLE evidence (
    id TEXT PRIMARY KEY,
    knowledge_item_id TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    method TEXT NOT NULL CHECK(method IN ('bm25', 'vector', 'hybrid')),
    score REAL NOT NULL,                         -- BM25 rank or cosine similarity
    snippet TEXT,                                -- Short excerpt of matched content
    pii_redacted INTEGER NOT NULL DEFAULT 0,     -- 0=false, 1=true (bool via sqlc override)
    metadata TEXT,                               -- JSON: {distance, rank, rrf_score, confidence}
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (knowledge_item_id) REFERENCES knowledge_item(id),
    FOREIGN KEY (workspace_id) REFERENCES workspace(id)
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

CREATE INDEX idx_knowledge_workspace ON knowledge_item(workspace_id);
CREATE INDEX idx_knowledge_entity ON knowledge_item(entity_type, entity_id);
CREATE INDEX idx_knowledge_created ON knowledge_item(created_at);
CREATE INDEX idx_knowledge_deleted ON knowledge_item(deleted_at);
CREATE INDEX idx_knowledge_source ON knowledge_item(workspace_id, source_type);

CREATE INDEX idx_embedding_knowledge_item ON embedding_document(knowledge_item_id);
CREATE INDEX idx_embedding_workspace ON embedding_document(workspace_id);
CREATE INDEX idx_embedding_status ON embedding_document(embedding_status);
CREATE INDEX idx_embedding_ws_status ON embedding_document(workspace_id, embedding_status);

CREATE INDEX idx_evidence_knowledge_item ON evidence(knowledge_item_id);
CREATE INDEX idx_evidence_workspace ON evidence(workspace_id);
CREATE INDEX idx_evidence_method ON evidence(workspace_id, method);
CREATE INDEX idx_evidence_score ON evidence(workspace_id, score DESC);
