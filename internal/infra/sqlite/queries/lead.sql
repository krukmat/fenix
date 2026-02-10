-- SQL queries for lead table
-- Task 1.5: Lead management queries

-- name: CreateLead :exec
INSERT INTO lead (id, workspace_id, contact_id, account_id, source, status, owner_id, score, metadata, created_at, updated_at, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetLeadByID :one
SELECT id, workspace_id, contact_id, account_id, source, status, owner_id, score, metadata, created_at, updated_at, deleted_at
FROM lead
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL
LIMIT 1;

-- name: ListLeadsByWorkspace :many
SELECT id, workspace_id, contact_id, account_id, source, status, owner_id, score, metadata, created_at, updated_at, deleted_at
FROM lead
WHERE workspace_id = ?
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT ?
OFFSET ?;

-- name: ListLeadsByOwner :many
SELECT id, workspace_id, contact_id, account_id, source, status, owner_id, score, metadata, created_at, updated_at, deleted_at
FROM lead
WHERE workspace_id = ?
  AND owner_id = ?
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListLeadsByStatus :many
SELECT id, workspace_id, contact_id, account_id, source, status, owner_id, score, metadata, created_at, updated_at, deleted_at
FROM lead
WHERE workspace_id = ?
  AND status = ?
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListLeadsByAccount :many
SELECT id, workspace_id, contact_id, account_id, source, status, owner_id, score, metadata, created_at, updated_at, deleted_at
FROM lead
WHERE workspace_id = ?
  AND account_id = ?
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateLead :exec
UPDATE lead
SET contact_id = ?,
    account_id = ?,
    source = ?,
    status = ?,
    owner_id = ?,
    score = ?,
    metadata = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL;

-- name: SoftDeleteLead :exec
UPDATE lead
SET deleted_at = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL;

-- name: CountLeadsByWorkspace :one
SELECT COUNT(*) FROM lead
WHERE workspace_id = ?
  AND deleted_at IS NULL;

-- name: CountLeadsByStatus :one
SELECT COUNT(*) FROM lead
WHERE workspace_id = ?
  AND status = ?
  AND deleted_at IS NULL;
