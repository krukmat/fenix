-- Rollback Migration 012: Knowledge Layer — FTS5 Virtual Table + Triggers
-- Related to: Task 2.1

DROP TRIGGER IF EXISTS knowledge_item_ad;
DROP TRIGGER IF EXISTS knowledge_item_au;
DROP TRIGGER IF EXISTS knowledge_item_ai;
DROP TABLE IF EXISTS knowledge_item_fts;
