-- SQL queries for note table
-- Task 1.5: Note/comment management queries

-- name: CreateNote :exec
INSERT INTO note (id, workspace_id, entity_type, entity_id, author_id, content, is_internal, metadata, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetNoteByID :one
SELECT id, workspace_id, entity_type, entity_id, author_id, content, is_internal, metadata, created_at, updated_at
FROM note
WHERE id = ?
  AND workspace_id = ?
LIMIT 1;

-- name: ListNotesByWorkspace :many
SELECT id, workspace_id, entity_type, entity_id, author_id, content, is_internal, metadata, created_at, updated_at
FROM note
WHERE workspace_id = ?
ORDER BY created_at DESC
LIMIT ?
OFFSET ?;

-- name: ListNotesByEntity :many
SELECT id, workspace_id, entity_type, entity_id, author_id, content, is_internal, metadata, created_at, updated_at
FROM note
WHERE workspace_id = ?
  AND entity_type = ?
  AND entity_id = ?
ORDER BY created_at DESC;

-- name: ListNotesByEntityPublic :many
SELECT id, workspace_id, entity_type, entity_id, author_id, content, is_internal, metadata, created_at, updated_at
FROM note
WHERE workspace_id = ?
  AND entity_type = ?
  AND entity_id = ?
  AND is_internal = 0
ORDER BY created_at DESC;

-- name: ListNotesByAuthor :many
SELECT id, workspace_id, entity_type, entity_id, author_id, content, is_internal, metadata, created_at, updated_at
FROM note
WHERE workspace_id = ?
  AND author_id = ?
ORDER BY created_at DESC;

-- name: UpdateNote :exec
UPDATE note
SET content = ?,
    is_internal = ?,
    metadata = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?;

-- name: DeleteNote :exec
DELETE FROM note
WHERE id = ?
  AND workspace_id = ?;

-- name: CountNotesByWorkspace :one
SELECT COUNT(*) FROM note
WHERE workspace_id = ?;

-- name: CountNotesByEntity :one
SELECT COUNT(*) FROM note
WHERE workspace_id = ?
  AND entity_type = ?
  AND entity_id = ?;
