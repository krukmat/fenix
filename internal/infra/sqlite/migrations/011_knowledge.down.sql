-- Rollback Migration 011: Knowledge Layer — Core Tables
-- NOTE: FTS5 + triggers rollback is in 012_knowledge_fts.down.sql
-- Related to: Task 2.1

-- Drop indexes
DROP INDEX IF EXISTS idx_evidence_score;
DROP INDEX IF EXISTS idx_evidence_method;
DROP INDEX IF EXISTS idx_evidence_workspace;
DROP INDEX IF EXISTS idx_evidence_knowledge_item;

DROP INDEX IF EXISTS idx_embedding_ws_status;
DROP INDEX IF EXISTS idx_embedding_status;
DROP INDEX IF EXISTS idx_embedding_workspace;
DROP INDEX IF EXISTS idx_embedding_knowledge_item;

DROP INDEX IF EXISTS idx_knowledge_source;
DROP INDEX IF EXISTS idx_knowledge_deleted;
DROP INDEX IF EXISTS idx_knowledge_created;
DROP INDEX IF EXISTS idx_knowledge_entity;
DROP INDEX IF EXISTS idx_knowledge_workspace;

-- Drop core tables (order matters for FK constraints)
DROP TABLE IF EXISTS evidence;
DROP TABLE IF EXISTS embedding_document;
DROP TABLE IF EXISTS knowledge_item;
