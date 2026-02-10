-- SQL queries for role and user_role tables
-- Task 1.2.7: sqlc-annotated queries
-- IMPORTANT: All role queries filter by workspace_id for RBAC isolation.

-- ========================
-- ROLE queries
-- ========================

-- name: CreateRole :exec
INSERT INTO role (id, workspace_id, name, description, permissions, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetRoleByID :one
SELECT id, workspace_id, name, description, permissions, created_at, updated_at
FROM role
WHERE id = ?
  AND workspace_id = ?
LIMIT 1;

-- name: GetRoleByName :one
SELECT id, workspace_id, name, description, permissions, created_at, updated_at
FROM role
WHERE workspace_id = ?
  AND name = ?
LIMIT 1;

-- name: ListRolesByWorkspace :many
SELECT id, workspace_id, name, description, permissions, created_at, updated_at
FROM role
WHERE workspace_id = ?
ORDER BY name ASC;

-- name: UpdateRole :exec
UPDATE role
SET name        = ?,
    description = ?,
    permissions = ?,
    updated_at  = ?
WHERE id = ?
  AND workspace_id = ?;

-- name: DeleteRole :exec
DELETE FROM role
WHERE id = ?
  AND workspace_id = ?;

-- ========================
-- USER ROLE (assignment) queries
-- ========================

-- name: AssignRole :exec
INSERT INTO user_role (id, user_id, role_id, created_at)
VALUES (?, ?, ?, ?);

-- name: GetUserRole :one
SELECT id, user_id, role_id, created_at
FROM user_role
WHERE user_id = ?
  AND role_id = ?
LIMIT 1;

-- name: ListRolesByUser :many
SELECT r.id, r.workspace_id, r.name, r.description, r.permissions, r.created_at, r.updated_at
FROM role r
JOIN user_role ur ON ur.role_id = r.id
WHERE ur.user_id = ?
  AND r.workspace_id = ?
ORDER BY r.name ASC;

-- name: ListUsersByRole :many
SELECT ua.id, ua.workspace_id, ua.external_idp_id, ua.email, ua.password_hash, ua.display_name, ua.avatar_url, ua.status, ua.preferences, ua.created_at, ua.updated_at
FROM user_account ua
JOIN user_role ur ON ur.user_id = ua.id
WHERE ur.role_id = ?
  AND ua.workspace_id = ?
ORDER BY ua.display_name ASC;

-- name: RevokeRole :exec
DELETE FROM user_role
WHERE user_id = ?
  AND role_id = ?;

-- name: RevokeAllRoles :exec
DELETE FROM user_role
WHERE user_id = ?;
