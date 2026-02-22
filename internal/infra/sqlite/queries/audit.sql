-- Queries for audit_event table
-- Related to: Task 1.7, internal/domain/audit

-- name: CreateAuditEvent :exec
-- Creates a new audit event (append-only, immutable)
INSERT INTO audit_event (
    id, workspace_id, actor_id, actor_type, action,
    entity_type, entity_id, details, permissions_checked,
    outcome, trace_id, ip_address, user_agent, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAuditEventByID :one
-- Retrieves a single audit event by ID
SELECT * FROM audit_event WHERE id = ? LIMIT 1;

-- name: ListAuditEventsByWorkspace :many
-- Lists audit events for a workspace with pagination
-- Results ordered by created_at DESC (newest first)
SELECT * FROM audit_event
WHERE workspace_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountAuditEventsByWorkspace :one
-- Counts total audit events for a workspace
SELECT COUNT(*) FROM audit_event WHERE workspace_id = ?;

-- name: ListAuditEventsByActor :many
-- Lists audit events for a specific actor
SELECT * FROM audit_event
WHERE actor_id = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: ListAuditEventsByEntity :many
-- Lists audit events for a specific entity
SELECT * FROM audit_event
WHERE entity_type = ? AND entity_id = ?
ORDER BY created_at DESC
LIMIT ?;

-- name: ListAuditEventsByOutcome :many
-- Lists audit events filtered by outcome (success/denied/error)
SELECT * FROM audit_event
WHERE workspace_id = ? AND outcome = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListAuditEventsByAction :many
-- Lists audit events filtered by action type
SELECT * FROM audit_event
WHERE workspace_id = ? AND action = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListAuditEventsByTimeRange :many
-- Lists audit events within a time range
SELECT * FROM audit_event
WHERE workspace_id = sqlc.arg(workspace_id)
  AND datetime(created_at) BETWEEN datetime(sqlc.arg(date_from)) AND datetime(sqlc.arg(date_to))
ORDER BY created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: QueryAuditEvents :many
-- Lists audit events filtered by optional compound criteria
SELECT * FROM audit_event
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(actor_id) = '' OR actor_id = sqlc.arg(actor_id))
  AND (sqlc.arg(entity_type) = '' OR entity_type = sqlc.arg(entity_type))
  AND (sqlc.arg(action) = '' OR action = sqlc.arg(action))
  AND (sqlc.arg(outcome) = '' OR outcome = sqlc.arg(outcome))
  AND (sqlc.arg(date_from) = '' OR datetime(created_at) >= datetime(sqlc.arg(date_from)))
  AND (sqlc.arg(date_to) = '' OR datetime(created_at) <= datetime(sqlc.arg(date_to)))
ORDER BY created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);
