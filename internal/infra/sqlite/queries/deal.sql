-- SQL queries for deal table
-- Task 1.5: Deal management queries

-- name: CreateDeal :exec
INSERT INTO deal (id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, title, amount, currency, expected_close, status, metadata, created_at, updated_at, deleted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetDealByID :one
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, title, amount, currency, expected_close, status, metadata, created_at, updated_at, deleted_at
FROM deal
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL
LIMIT 1;

-- name: ListDealsByWorkspace :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, title, amount, currency, expected_close, status, metadata, created_at, updated_at, deleted_at
FROM deal
WHERE workspace_id = ?
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT ?
OFFSET ?;

-- name: ListDealsByOwner :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, title, amount, currency, expected_close, status, metadata, created_at, updated_at, deleted_at
FROM deal
WHERE workspace_id = ?
  AND owner_id = ?
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListDealsByAccount :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, title, amount, currency, expected_close, status, metadata, created_at, updated_at, deleted_at
FROM deal
WHERE workspace_id = ?
  AND account_id = ?
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListDealsByPipeline :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, title, amount, currency, expected_close, status, metadata, created_at, updated_at, deleted_at
FROM deal
WHERE workspace_id = ?
  AND pipeline_id = ?
  AND deleted_at IS NULL
ORDER BY stage_id, created_at DESC;

-- name: ListDealsByStage :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, title, amount, currency, expected_close, status, metadata, created_at, updated_at, deleted_at
FROM deal
WHERE workspace_id = ?
  AND stage_id = ?
  AND deleted_at IS NULL
  AND status = 'open'
ORDER BY created_at DESC;

-- name: ListDealsByStatus :many
SELECT id, workspace_id, account_id, contact_id, pipeline_id, stage_id, owner_id, title, amount, currency, expected_close, status, metadata, created_at, updated_at, deleted_at
FROM deal
WHERE workspace_id = ?
  AND status = ?
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateDeal :exec
UPDATE deal
SET account_id = ?,
    contact_id = ?,
    pipeline_id = ?,
    stage_id = ?,
    owner_id = ?,
    title = ?,
    amount = ?,
    currency = ?,
    expected_close = ?,
    status = ?,
    metadata = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL;

-- name: SoftDeleteDeal :exec
UPDATE deal
SET deleted_at = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?
  AND deleted_at IS NULL;

-- name: CountDealsByWorkspace :one
SELECT COUNT(*) FROM deal
WHERE workspace_id = ?
  AND deleted_at IS NULL;

-- name: CountDealsByPipeline :one
SELECT COUNT(*) FROM deal
WHERE workspace_id = ?
  AND pipeline_id = ?
  AND deleted_at IS NULL;

-- name: SumDealAmountByPipeline :one
SELECT COALESCE(SUM(amount), 0) FROM deal
WHERE workspace_id = ?
  AND pipeline_id = ?
  AND status = 'open'
  AND deleted_at IS NULL;
