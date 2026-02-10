-- SQL queries for timeline_event table
-- Task 1.5: Timeline/audit trail queries

-- name: CreateTimelineEvent :exec
INSERT INTO timeline_event (id, workspace_id, entity_type, entity_id, actor_id, event_type, old_value, new_value, context, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTimelineEventByID :one
SELECT id, workspace_id, entity_type, entity_id, actor_id, event_type, old_value, new_value, context, created_at
FROM timeline_event
WHERE id = ?
  AND workspace_id = ?
LIMIT 1;

-- name: ListTimelineEventsByWorkspace :many
SELECT id, workspace_id, entity_type, entity_id, actor_id, event_type, old_value, new_value, context, created_at
FROM timeline_event
WHERE workspace_id = ?
ORDER BY created_at DESC
LIMIT ?
OFFSET ?;

-- name: ListTimelineEventsByEntity :many
SELECT id, workspace_id, entity_type, entity_id, actor_id, event_type, old_value, new_value, context, created_at
FROM timeline_event
WHERE workspace_id = ?
  AND entity_type = ?
  AND entity_id = ?
ORDER BY created_at DESC
LIMIT ?
OFFSET ?;

-- name: ListTimelineEventsByActor :many
SELECT id, workspace_id, entity_type, entity_id, actor_id, event_type, old_value, new_value, context, created_at
FROM timeline_event
WHERE workspace_id = ?
  AND actor_id = ?
ORDER BY created_at DESC
LIMIT ?
OFFSET ?;

-- name: ListTimelineEventsByEventType :many
SELECT id, workspace_id, entity_type, entity_id, actor_id, event_type, old_value, new_value, context, created_at
FROM timeline_event
WHERE workspace_id = ?
  AND event_type = ?
ORDER BY created_at DESC
LIMIT ?
OFFSET ?;

-- name: CountTimelineEventsByWorkspace :one
SELECT COUNT(*) FROM timeline_event
WHERE workspace_id = ?;

-- name: CountTimelineEventsByEntity :one
SELECT COUNT(*) FROM timeline_event
WHERE workspace_id = ?
  AND entity_type = ?
  AND entity_id = ?;

-- name: GetLatestTimelineEventByEntity :one
SELECT id, workspace_id, entity_type, entity_id, actor_id, event_type, old_value, new_value, context, created_at
FROM timeline_event
WHERE workspace_id = ?
  AND entity_type = ?
  AND entity_id = ?
ORDER BY created_at DESC
LIMIT 1;
