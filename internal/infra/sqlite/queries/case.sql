-- SQL queries for case_ticket table
-- Task 1.5: Case/Support ticket management queries

-- name: CreateCase :exec
INSERT INTO case_ticket (id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, subject, description, priority, status, channel, sla_config, sla_deadline, metadata, created_at, updated_at, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetCaseByID :one
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, subject, description, priority, status, channel, sla_config, sla_deadline, metadata, created_at, updated_at, deleted_at
FROM case_ticket
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL
LIMIT 1;

-- name: ListCasesByWorkspace :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, subject, description, priority, status, channel, sla_config, sla_deadline, metadata, created_at, updated_at, deleted_at
FROM case_ticket
WHERE workspace_id = ?
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT ?
OFFSET ?;

-- name: ListCasesByOwner :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, subject, description, priority, status, channel, sla_config, sla_deadline, metadata, created_at, updated_at, deleted_at
FROM case_ticket
WHERE workspace_id = ?
  AND owner_id = ?
  AND deleted_at IS NULL
ORDER BY
  CASE priority
    WHEN 'critical' THEN 1
    WHEN 'high' THEN 2
    WHEN 'medium' THEN 3
    WHEN 'low' THEN 4
  END,
  created_at DESC;

-- name: ListCasesByAccount :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, subject, description, priority, status, channel, sla_config, sla_deadline, metadata, created_at, updated_at, deleted_at
FROM case_ticket
WHERE workspace_id = ?
  AND account_id = ?
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListCasesByStatus :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, subject, description, priority, status, channel, sla_config, sla_deadline, metadata, created_at, updated_at, deleted_at
FROM case_ticket
WHERE workspace_id = ?
  AND status = ?
  AND deleted_at IS NULL
ORDER BY
  CASE priority
    WHEN 'critical' THEN 1
    WHEN 'high' THEN 2
    WHEN 'medium' THEN 3
    WHEN 'low' THEN 4
  END,
  created_at DESC;

-- name: ListOpenCasesByPriority :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, subject, description, priority, status, channel, sla_config, sla_deadline, metadata, created_at, updated_at, deleted_at
FROM case_ticket
WHERE workspace_id = ?
  AND deleted_at IS NULL
  AND status IN ('open', 'in_progress', 'waiting')
ORDER BY
  CASE priority
    WHEN 'critical' THEN 1
    WHEN 'high' THEN 2
    WHEN 'medium' THEN 3
    WHEN 'low' THEN 4
  END,
  created_at DESC;

-- name: ListCasesBySLADeadline :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, subject, description, priority, status, channel, sla_config, sla_deadline, metadata, created_at, updated_at, deleted_at
FROM case_ticket
WHERE workspace_id = ?
  AND sla_deadline IS NOT NULL
  AND deleted_at IS NULL
  AND status IN ('open', 'in_progress', 'waiting')
ORDER BY sla_deadline ASC
LIMIT ?;

-- name: UpdateCase :exec
UPDATE case_ticket
SET account_id = ?,
    contact_id = ?,
    pipeline_id = ?,
    stage_id = ?,
    owner_id = ?,
    subject = ?,
    description = ?,
    priority = ?,
    status = ?,
    channel = ?,
    sla_config = ?,
    sla_deadline = ?,
    metadata = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL;

-- name: SoftDeleteCase :exec
UPDATE case_ticket
SET deleted_at = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL;

-- name: CountCasesByWorkspace :one
SELECT COUNT(*) FROM case_ticket
WHERE workspace_id = ?
  AND deleted_at IS NULL;

-- name: CountCasesByStatus :one
SELECT COUNT(*) FROM case_ticket
WHERE workspace_id = ?
  AND status = ?
  AND deleted_at IS NULL;

-- name: CountOverdueCases :one
SELECT COUNT(*) FROM case_ticket
WHERE workspace_id = ?
  AND sla_deadline < datetime('now')
  AND deleted_at IS NULL
  AND status IN ('open', 'in_progress', 'waiting');
