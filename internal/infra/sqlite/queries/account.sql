-- SQL queries for account table
-- Task 1.3.2: sqlc-annotated queries
-- IMPORTANT: All account queries filter by workspace_id for multi-tenancy isolation.

-- name: CreateAccount :exec
INSERT INTO account (id, workspace_id, name, domain, industry, size_segment, owner_id, address, metadata, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAccountByID :one
SELECT id, workspace_id, name, domain, industry, size_segment, owner_id, address, metadata, created_at, updated_at, deleted_at
FROM account
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL
LIMIT 1;

-- name: ListAccountsByWorkspace :many
SELECT id, workspace_id, name, domain, industry, size_segment, owner_id, address, metadata, created_at, updated_at, deleted_at
FROM account
WHERE workspace_id = ?
  AND deleted_at IS NULL
ORDER BY name ASC
LIMIT ?
OFFSET ?;

-- name: ListAccountsByOwner :many
SELECT id, workspace_id, name, domain, industry, size_segment, owner_id, address, metadata, created_at, updated_at, deleted_at
FROM account
WHERE workspace_id = ?
  AND owner_id = ?
  AND deleted_at IS NULL
ORDER BY name ASC;

-- name: UpdateAccount :exec
UPDATE account
SET name = ?,
    domain = ?,
    industry = ?,
    size_segment = ?,
    owner_id = ?,
    address = ?,
    metadata = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL;

-- name: SoftDeleteAccount :exec
UPDATE account
SET deleted_at = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL;

-- name: CountAccountsByWorkspace :one
SELECT COUNT(*) FROM account
WHERE workspace_id = ?
  AND deleted_at IS NULL;
