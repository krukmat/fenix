-- SQL queries for attachment table
-- Task 1.5: File attachment management queries

-- name: CreateAttachment :exec
INSERT INTO attachment (id, workspace_id, entity_type, entity_id, uploader_id, filename, content_type, size_bytes, storage_path, sensitivity, metadata, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAttachmentByID :one
SELECT id, workspace_id, entity_type, entity_id, uploader_id, filename, content_type, size_bytes, storage_path, sensitivity, metadata, created_at
FROM attachment
WHERE id = ?
  AND workspace_id = ?
LIMIT 1;

-- name: ListAttachmentsByWorkspace :many
SELECT id, workspace_id, entity_type, entity_id, uploader_id, filename, content_type, size_bytes, storage_path, sensitivity, metadata, created_at
FROM attachment
WHERE workspace_id = ?
ORDER BY created_at DESC
LIMIT ?
OFFSET ?;

-- name: ListAttachmentsByEntity :many
SELECT id, workspace_id, entity_type, entity_id, uploader_id, filename, content_type, size_bytes, storage_path, sensitivity, metadata, created_at
FROM attachment
WHERE workspace_id = ?
  AND entity_type = ?
  AND entity_id = ?
ORDER BY created_at DESC;

-- name: ListAttachmentsByUploader :many
SELECT id, workspace_id, entity_type, entity_id, uploader_id, filename, content_type, size_bytes, storage_path, sensitivity, metadata, created_at
FROM attachment
WHERE workspace_id = ?
  AND uploader_id = ?
ORDER BY created_at DESC;

-- name: DeleteAttachment :exec
DELETE FROM attachment
WHERE id = ?
  AND workspace_id = ?;

-- name: CountAttachmentsByWorkspace :one
SELECT COUNT(*) FROM attachment
WHERE workspace_id = ?;

-- name: CountAttachmentsByEntity :one
SELECT COUNT(*) FROM attachment
WHERE workspace_id = ?
  AND entity_type = ?
  AND entity_id = ?;

-- name: GetTotalAttachmentSizeByWorkspace :one
SELECT COALESCE(SUM(size_bytes), 0) FROM attachment
WHERE workspace_id = ?;
