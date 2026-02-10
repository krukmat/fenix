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
WHERE workspace_id = ? AND created_at BETWEEN ? AND ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
