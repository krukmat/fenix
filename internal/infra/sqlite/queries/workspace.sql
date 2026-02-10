-- SQL queries for workspace table
-- Task 1.2.7: sqlc-annotated queries
-- Note: RETURNING * not supported by sqlc SQLite parser — use :exec + GetByID pattern.

-- name: CreateWorkspace :exec
INSERT INTO workspace (id, name, slug, settings, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetWorkspaceByID :one
SELECT id, name, slug, settings, created_at, updated_at
FROM workspace
WHERE id = ?
LIMIT 1;

-- name: GetWorkspaceBySlug :one
SELECT id, name, slug, settings, created_at, updated_at
FROM workspace
WHERE slug = ?
LIMIT 1;

-- name: ListWorkspaces :many
SELECT id, name, slug, settings, created_at, updated_at
FROM workspace
ORDER BY name ASC;

-- name: UpdateWorkspace :exec
UPDATE workspace
SET name       = ?,
    slug       = ?,
    settings   = ?,
    updated_at = ?
WHERE id = ?;

-- name: DeleteWorkspace :exec
DELETE FROM workspace
WHERE id = ?;
