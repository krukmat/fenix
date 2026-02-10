-- SQL queries for activity table
-- Task 1.5: Activity (tasks, events, calls, emails) management queries

-- name: CreateActivity :exec
INSERT INTO activity (id, workspace_id, activity_type, entity_type, entity_id, owner_id, assigned_to, subject, body, status, due_at, completed_at, metadata, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetActivityByID :one
SELECT id, workspace_id, activity_type, entity_type, entity_id, owner_id, assigned_to, subject, body, status, due_at, completed_at, metadata, created_at, updated_at
FROM activity
WHERE id = ?
  AND workspace_id = ?
LIMIT 1;

-- name: ListActivitiesByWorkspace :many
SELECT id, workspace_id, activity_type, entity_type, entity_id, owner_id, assigned_to, subject, body, status, due_at, completed_at, metadata, created_at, updated_at
FROM activity
WHERE workspace_id = ?
ORDER BY created_at DESC
LIMIT ?
OFFSET ?;

-- name: ListActivitiesByEntity :many
SELECT id, workspace_id, activity_type, entity_type, entity_id, owner_id, assigned_to, subject, body, status, due_at, completed_at, metadata, created_at, updated_at
FROM activity
WHERE workspace_id = ?
  AND entity_type = ?
  AND entity_id = ?
ORDER BY created_at DESC;

-- name: ListActivitiesByOwner :many
SELECT id, workspace_id, activity_type, entity_type, entity_id, owner_id, assigned_to, subject, body, status, due_at, completed_at, metadata, created_at, updated_at
FROM activity
WHERE workspace_id = ?
  AND owner_id = ?
ORDER BY created_at DESC;

-- name: ListActivitiesByAssignee :many
SELECT id, workspace_id, activity_type, entity_type, entity_id, owner_id, assigned_to, subject, body, status, due_at, completed_at, metadata, created_at, updated_at
FROM activity
WHERE workspace_id = ?
  AND assigned_to = ?
ORDER BY
  CASE status
    WHEN 'pending' THEN 1
    WHEN 'completed' THEN 2
    WHEN 'cancelled' THEN 3
  END,
  due_at ASC NULLS LAST,
  created_at DESC;

-- name: ListPendingActivitiesByAssignee :many
SELECT id, workspace_id, activity_type, entity_type, entity_id, owner_id, assigned_to, subject, body, status, due_at, completed_at, metadata, created_at, updated_at
FROM activity
WHERE workspace_id = ?
  AND assigned_to = ?
  AND status = 'pending'
ORDER BY
  CASE WHEN due_at < datetime('now') THEN 0 ELSE 1 END,
  due_at ASC NULLS LAST;

-- name: ListActivitiesByType :many
SELECT id, workspace_id, activity_type, entity_type, entity_id, owner_id, assigned_to, subject, body, status, due_at, completed_at, metadata, created_at, updated_at
FROM activity
WHERE workspace_id = ?
  AND activity_type = ?
ORDER BY created_at DESC;

-- name: ListActivitiesByStatus :many
SELECT id, workspace_id, activity_type, entity_type, entity_id, owner_id, assigned_to, subject, body, status, due_at, completed_at, metadata, created_at, updated_at
FROM activity
WHERE workspace_id = ?
  AND status = ?
ORDER BY created_at DESC;

-- name: UpdateActivity :exec
UPDATE activity
SET activity_type = ?,
    entity_type = ?,
    entity_id = ?,
    owner_id = ?,
    assigned_to = ?,
    subject = ?,
    body = ?,
    status = ?,
    due_at = ?,
    completed_at = ?,
    metadata = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?;

-- name: DeleteActivity :exec
DELETE FROM activity
WHERE id = ?
  AND workspace_id = ?;

-- name: CountActivitiesByWorkspace :one
SELECT COUNT(*) FROM activity
WHERE workspace_id = ?;

-- name: CountActivitiesByEntity :one
SELECT COUNT(*) FROM activity
WHERE workspace_id = ?
  AND entity_type = ?
  AND entity_id = ?;

-- name: CountPendingActivitiesByAssignee :one
SELECT COUNT(*) FROM activity
WHERE workspace_id = ?
  AND assigned_to = ?
  AND status = 'pending';
