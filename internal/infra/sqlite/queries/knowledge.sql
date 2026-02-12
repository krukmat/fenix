-- Queries for knowledge layer tables
-- Related to: Task 2.1, internal/domain/knowledge
-- SECURITY NOTE: All queries include workspace_id filter to enforce multi-tenant isolation.
-- Vector search (vec_embedding) is NOT handled here - modernc.org/sqlite requires
-- raw sql.DB queries for virtual tables. See Task 2.5 for safe vector query patterns.

-- ============================================================================
-- KNOWLEDGE ITEM QUERIES
-- ============================================================================

-- name: CreateKnowledgeItem :exec
-- Task 2.1/2.2: Insert a new knowledge item
INSERT INTO knowledge_item (
    id, workspace_id, source_type, title, raw_content,
    normalized_content, entity_type, entity_id, metadata,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetKnowledgeItemByID :one
-- Task 2.1/2.2: Retrieve a single knowledge item (excludes soft-deleted)
SELECT * FROM knowledge_item
WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL
LIMIT 1;

-- name: GetKnowledgeItemByEntity :one
-- Task 2.7: Find knowledge item linked to a CRM entity
SELECT * FROM knowledge_item
WHERE workspace_id = ? AND entity_type = ? AND entity_id = ? AND deleted_at IS NULL
LIMIT 1;

-- name: ListKnowledgeItemsByWorkspace :many
-- Task 2.2: List all knowledge items for a workspace (paginated)
SELECT * FROM knowledge_item
WHERE workspace_id = ? AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListKnowledgeItemsByEntity :many
-- Task 2.7: List knowledge items linked to a specific entity type
SELECT * FROM knowledge_item
WHERE workspace_id = ? AND entity_type = ? AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateKnowledgeItemNormalizedContent :exec
-- Task 2.7: Update normalized content after CDC reindex
-- Triggers knowledge_item_au which re-syncs FTS5 index
UPDATE knowledge_item
SET normalized_content = ?, updated_at = ?
WHERE id = ? AND workspace_id = ?;

-- name: SoftDeleteKnowledgeItem :exec
-- Task 2.2/2.7: Soft delete a knowledge item (preserves audit trail)
UPDATE knowledge_item
SET deleted_at = ?
WHERE id = ? AND workspace_id = ?;

-- name: CountKnowledgeItemsByWorkspace :one
-- Task 2.2: Count total knowledge items for a workspace
SELECT COUNT(*) FROM knowledge_item
WHERE workspace_id = ? AND deleted_at IS NULL;

-- ============================================================================
-- EMBEDDING DOCUMENT QUERIES
-- ============================================================================

-- name: CreateEmbeddingDocument :exec
-- Task 2.2/2.4: Insert a chunk with pending embedding status
INSERT INTO embedding_document (
    id, knowledge_item_id, workspace_id, chunk_index,
    chunk_text, token_count, embedding_status, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetEmbeddingDocumentByID :one
-- Task 2.4: Get a single embedding document
SELECT * FROM embedding_document
WHERE id = ? AND workspace_id = ?
LIMIT 1;

-- name: ListEmbeddingDocumentsByKnowledgeItem :many
-- Task 2.4: Get all chunks for a knowledge item (for embedding job)
SELECT * FROM embedding_document
WHERE knowledge_item_id = ? AND workspace_id = ?
ORDER BY chunk_index ASC;

-- name: ListPendingEmbeddingsByWorkspace :many
-- Task 2.4: Get all chunks waiting to be embedded (for background job)
SELECT * FROM embedding_document
WHERE workspace_id = ? AND embedding_status = 'pending'
ORDER BY created_at ASC
LIMIT ? OFFSET ?;

-- name: UpdateEmbeddingDocumentStatus :exec
-- Task 2.4: Mark a chunk as embedded (or failed)
UPDATE embedding_document
SET embedding_status = ?, embedded_at = ?
WHERE id = ? AND workspace_id = ?;

-- name: CountPendingEmbeddingsByWorkspace :one
-- Task 2.4/2.7: Count pending embeddings for a workspace
SELECT COUNT(*) FROM embedding_document
WHERE workspace_id = ? AND embedding_status = 'pending';

-- name: DeleteEmbeddingDocumentsByKnowledgeItem :exec
-- Task 2.7: Remove all chunks when knowledge item is deleted/reindexed
DELETE FROM embedding_document
WHERE knowledge_item_id = ? AND workspace_id = ?;

-- ============================================================================
-- EVIDENCE QUERIES
-- ============================================================================

-- name: CreateEvidence :exec
-- Task 2.6: Store a search result snapshot
INSERT INTO evidence (
    id, knowledge_item_id, workspace_id, method,
    score, snippet, pii_redacted, metadata, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetEvidenceByID :one
-- Task 2.6: Retrieve a single evidence record
SELECT * FROM evidence
WHERE id = ? AND workspace_id = ?
LIMIT 1;

-- name: ListEvidenceByKnowledgeItem :many
-- Task 2.6: List evidence for a knowledge item, ordered by score
SELECT * FROM evidence
WHERE knowledge_item_id = ? AND workspace_id = ?
ORDER BY score DESC;

-- name: ListEvidenceByMethod :many
-- Task 2.6: List evidence filtered by search method
SELECT * FROM evidence
WHERE workspace_id = ? AND method = ?
ORDER BY score DESC
LIMIT ? OFFSET ?;

-- ============================================================================
-- vec_embedding queries (Task 2.4)
-- ============================================================================

-- name: InsertVecEmbedding :exec
-- Task 2.4: Store a float32 vector as JSON TEXT for an embedding_document chunk.
INSERT INTO vec_embedding (id, workspace_id, embedding, created_at)
VALUES (?, ?, ?, ?);

-- name: DeleteVecEmbeddingsByKnowledgeItem :exec
-- Task 2.4: Remove vectors for all chunks of a knowledge_item (on re-ingest).
DELETE FROM vec_embedding
WHERE id IN (
    SELECT ed.id FROM embedding_document ed
    WHERE ed.knowledge_item_id = ? AND ed.workspace_id = ?
);

-- ============================================================================
-- search queries (Task 2.5)
-- ============================================================================

-- name: GetAllEmbeddedVectorsByWorkspace :many
-- Task 2.5: Fetches all embedded vectors for a workspace for in-memory cosine
-- distance calculation. workspace_id filter via JOIN (multi-tenant security).
-- Note: BM25/FTS5 query (SearchKnowledgeItemFTS) is executed as raw SQL in
-- search.go because sqlc does not support CREATE VIRTUAL TABLE fts5 syntax.
SELECT v.id, v.embedding, ed.knowledge_item_id
FROM vec_embedding v
JOIN embedding_document ed ON v.id = ed.id
WHERE ed.workspace_id = ?
  AND ed.embedding_status = 'embedded';
