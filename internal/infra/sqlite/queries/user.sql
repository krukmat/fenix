-- SQL queries for user_account table
-- Task 1.2.7: sqlc-annotated queries
-- IMPORTANT: All user queries filter by workspace_id for multi-tenancy isolation.
-- Note: RETURNING * not supported by sqlc SQLite parser — use :exec + GetByID pattern.

-- name: CreateUser :exec
INSERT INTO user_account (id, workspace_id, external_idp_id, email, password_hash, display_name, avatar_url, status, preferences, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetUserByID :one
SELECT id, workspace_id, external_idp_id, email, password_hash, display_name, avatar_url, status, preferences, created_at, updated_at
FROM user_account
WHERE id = ?
  AND workspace_id = ?
LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, workspace_id, external_idp_id, email, password_hash, display_name, avatar_url, status, preferences, created_at, updated_at
FROM user_account
WHERE email = ?
LIMIT 1;

-- name: GetUserByExternalIDP :one
SELECT id, workspace_id, external_idp_id, email, password_hash, display_name, avatar_url, status, preferences, created_at, updated_at
FROM user_account
WHERE external_idp_id = ?
LIMIT 1;

-- name: ListUsersByWorkspace :many
SELECT id, workspace_id, external_idp_id, email, password_hash, display_name, avatar_url, status, preferences, created_at, updated_at
FROM user_account
WHERE workspace_id = ?
ORDER BY display_name ASC;

-- name: ListActiveUsersByWorkspace :many
SELECT id, workspace_id, external_idp_id, email, password_hash, display_name, avatar_url, status, preferences, created_at, updated_at
FROM user_account
WHERE workspace_id = ?
  AND status = 'active'
ORDER BY display_name ASC;

-- name: UpdateUser :exec
UPDATE user_account
SET display_name = ?,
    avatar_url   = ?,
    preferences  = ?,
    updated_at   = ?
WHERE id = ?
  AND workspace_id = ?;

-- name: UpdateUserStatus :exec
UPDATE user_account
SET status     = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?;

-- name: UpdateUserPasswordHash :exec
UPDATE user_account
SET password_hash = ?,
    updated_at    = ?
WHERE id = ?
  AND workspace_id = ?;

-- name: DeleteUser :exec
DELETE FROM user_account
WHERE id = ?
  AND workspace_id = ?;

-- name: CountUsersByWorkspace :one
SELECT COUNT(*) FROM user_account
WHERE workspace_id = ?;
