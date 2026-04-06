-- Migration 030 rollback: remove connector provenance fields from knowledge_item.

DROP TRIGGER IF EXISTS knowledge_item_ai;
DROP TRIGGER IF EXISTS knowledge_item_au;
DROP TRIGGER IF EXISTS knowledge_item_ad;

ALTER TABLE knowledge_item RENAME TO knowledge_item_new;

CREATE TABLE knowledge_item (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK(source_type IN ('email', 'document', 'kb_article', 'api', 'note', 'call', 'case', 'ticket', 'other')),
    title TEXT NOT NULL,
    raw_content TEXT NOT NULL,
    normalized_content TEXT,
    entity_type TEXT,
    entity_id TEXT,
    metadata TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    FOREIGN KEY (workspace_id) REFERENCES workspace(id),
    UNIQUE(workspace_id, entity_type, entity_id)
);

INSERT INTO knowledge_item (
    id, workspace_id, source_type, title, raw_content,
    normalized_content, entity_type, entity_id, metadata,
    created_at, updated_at, deleted_at
)
SELECT
    id, workspace_id, source_type, title, raw_content,
    normalized_content, entity_type, entity_id, metadata,
    created_at, updated_at, deleted_at
FROM knowledge_item_new;

DROP TABLE knowledge_item_new;

CREATE INDEX idx_knowledge_workspace ON knowledge_item(workspace_id);
CREATE INDEX idx_knowledge_entity ON knowledge_item(entity_type, entity_id);
CREATE INDEX idx_knowledge_created ON knowledge_item(created_at);
CREATE INDEX idx_knowledge_deleted ON knowledge_item(deleted_at);
CREATE INDEX idx_knowledge_source ON knowledge_item(workspace_id, source_type);

DELETE FROM knowledge_item_fts;

INSERT INTO knowledge_item_fts (id, workspace_id, title, normalized_content)
SELECT id, workspace_id, title, COALESCE(normalized_content, raw_content)
FROM knowledge_item
WHERE deleted_at IS NULL;

CREATE TRIGGER knowledge_item_ai
AFTER INSERT ON knowledge_item
BEGIN
    INSERT INTO knowledge_item_fts (id, workspace_id, title, normalized_content)
    VALUES (new.id, new.workspace_id, new.title, COALESCE(new.normalized_content, new.raw_content));
END;

CREATE TRIGGER knowledge_item_au
AFTER UPDATE ON knowledge_item
BEGIN
    DELETE FROM knowledge_item_fts WHERE id = old.id;
    INSERT INTO knowledge_item_fts (id, workspace_id, title, normalized_content)
    VALUES (new.id, new.workspace_id, new.title, COALESCE(new.normalized_content, new.raw_content));
END;

CREATE TRIGGER knowledge_item_ad
AFTER DELETE ON knowledge_item
BEGIN
    DELETE FROM knowledge_item_fts WHERE id = old.id;
END;
