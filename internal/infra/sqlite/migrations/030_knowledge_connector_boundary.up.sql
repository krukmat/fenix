-- Migration 030: Preserve minimum connector provenance on knowledge ingestion.
-- Strategic repositioning W2-T4: connector boundary contract.

ALTER TABLE knowledge_item ADD COLUMN source_system TEXT;
ALTER TABLE knowledge_item ADD COLUMN source_object_id TEXT;
ALTER TABLE knowledge_item ADD COLUMN refresh_strategy TEXT;
ALTER TABLE knowledge_item ADD COLUMN delete_behavior TEXT;
ALTER TABLE knowledge_item ADD COLUMN permission_context TEXT;

CREATE INDEX idx_knowledge_source_object
    ON knowledge_item(workspace_id, source_system, source_object_id);
