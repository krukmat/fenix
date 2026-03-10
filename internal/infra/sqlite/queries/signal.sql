-- AGENT_SPEC Phase 2: Signal foundation queries

-- name: CreateSignal :one
INSERT INTO signal (
    id, workspace_id, entity_type, entity_id, signal_type, confidence,
    evidence_ids, source_type, source_id, metadata, status, dismissed_by,
    dismissed_at, expires_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSignalByID :one
SELECT * FROM signal
WHERE id = ? AND workspace_id = ?
LIMIT 1;

-- name: ListSignalsByWorkspace :many
SELECT * FROM signal
WHERE workspace_id = ?
ORDER BY created_at DESC;

-- name: ListSignalsByEntity :many
SELECT * FROM signal
WHERE workspace_id = ? AND entity_type = ? AND entity_id = ?
ORDER BY created_at DESC;

-- name: ListSignalsByStatus :many
SELECT * FROM signal
WHERE workspace_id = ? AND status = ?
ORDER BY created_at DESC;

-- name: DismissSignal :one
UPDATE signal
SET status = 'dismissed', dismissed_by = ?, dismissed_at = ?, updated_at = ?
WHERE id = ? AND workspace_id = ?
RETURNING *;
