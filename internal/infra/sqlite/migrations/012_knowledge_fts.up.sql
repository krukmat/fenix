-- Migration 012: Knowledge Layer — FTS5 Virtual Table + Sync Triggers
-- Creates knowledge_item_fts virtual table for full-text BM25 search
-- Related to: Task 2.1, FR-092
-- NOTE: Separated from 011 because sqlc cannot parse CREATE VIRTUAL TABLE syntax.
-- This file is applied by MigrateUp at runtime but excluded from sqlc schema parsing.
-- SECURITY: workspace_id is UNINDEXED — all FTS5 queries MUST filter on workspace_id.

-- ============================================================================
-- FTS5 VIRTUAL TABLE: Full-Text Search via BM25
-- BM25 scoring is built-in to FTS5 (no manual calculation needed)
-- workspace_id is UNINDEXED (filter column, not full-text searchable)
-- ============================================================================

CREATE VIRTUAL TABLE knowledge_item_fts USING fts5(
    id UNINDEXED,
    workspace_id UNINDEXED,
    title,
    normalized_content,
    tokenize = 'unicode61'
);

-- ============================================================================
-- TRIGGERS: Keep FTS5 in sync with knowledge_item
-- Note: modernc.org/sqlite requires plain DELETE+INSERT for FTS5 updates.
-- The FTS5 special 'delete' row syntax is not supported by this driver.
-- ============================================================================

-- Trigger: index new knowledge_item on INSERT
CREATE TRIGGER knowledge_item_ai
AFTER INSERT ON knowledge_item
BEGIN
    INSERT INTO knowledge_item_fts (id, workspace_id, title, normalized_content)
    VALUES (new.id, new.workspace_id, new.title, COALESCE(new.normalized_content, new.raw_content));
END;

-- Trigger: re-index knowledge_item on UPDATE (DELETE old entry, INSERT new)
CREATE TRIGGER knowledge_item_au
AFTER UPDATE ON knowledge_item
BEGIN
    DELETE FROM knowledge_item_fts WHERE id = old.id;
    INSERT INTO knowledge_item_fts (id, workspace_id, title, normalized_content)
    VALUES (new.id, new.workspace_id, new.title, COALESCE(new.normalized_content, new.raw_content));
END;

-- Trigger: remove knowledge_item from FTS5 on DELETE
CREATE TRIGGER knowledge_item_ad
AFTER DELETE ON knowledge_item
BEGIN
    DELETE FROM knowledge_item_fts WHERE id = old.id;
END;
