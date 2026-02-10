-- SQL queries for contact table
-- Task 1.4: sqlc-annotated queries
-- IMPORTANT: All contact queries filter by workspace_id for multi-tenancy isolation.

-- name: CreateContact :exec
INSERT INTO contact (
    id,
    workspace_id,
    account_id,
    first_name,
    last_name,
    email,
    phone,
    title,
    status,
    owner_id,
    metadata,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetContactByID :one
SELECT id, workspace_id, account_id, first_name, last_name, email, phone, title, status, owner_id, metadata, created_at, updated_at, deleted_at
FROM contact
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL
LIMIT 1;

-- name: ListContactsByWorkspace :many
SELECT id, workspace_id, account_id, first_name, last_name, email, phone, title, status, owner_id, metadata, created_at, updated_at, deleted_at
FROM contact
WHERE workspace_id = ?
  AND deleted_at IS NULL
ORDER BY first_name ASC, last_name ASC
LIMIT ?
OFFSET ?;

-- name: ListContactsByAccount :many
SELECT id, workspace_id, account_id, first_name, last_name, email, phone, title, status, owner_id, metadata, created_at, updated_at, deleted_at
FROM contact
WHERE workspace_id = ?
  AND account_id = ?
  AND deleted_at IS NULL
ORDER BY first_name ASC, last_name ASC;

-- name: UpdateContact :exec
UPDATE contact
SET account_id = ?,
    first_name = ?,
    last_name = ?,
    email = ?,
    phone = ?,
    title = ?,
    status = ?,
    owner_id = ?,
    metadata = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL;

-- name: SoftDeleteContact :exec
UPDATE contact
SET deleted_at = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL;

-- name: CountContactsByWorkspace :one
SELECT COUNT(*)
FROM contact
WHERE workspace_id = ?
  AND deleted_at IS NULL;
