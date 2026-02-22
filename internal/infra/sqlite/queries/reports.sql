-- Task 4.5e - Reporting base queries (FR-003)

-- name: SalesFunnelByWorkspace :many
SELECT
    ps.name,
    ps.position AS stage_order,
    COUNT(d.id) AS deal_count,
    COALESCE(SUM(d.amount), 0) AS total_value,
    COALESCE(ps.probability, 0) AS probability
FROM pipeline_stage ps
JOIN pipeline p ON p.id = ps.pipeline_id
LEFT JOIN deal d ON d.stage_id = ps.id
    AND d.workspace_id = p.workspace_id
    AND d.deleted_at IS NULL
WHERE p.workspace_id = ?
  AND p.entity_type = 'deal'
GROUP BY ps.id
ORDER BY ps.position;

-- name: DealAgingByWorkspace :many
SELECT
    ps.name,
    AVG(julianday('now') - julianday(d.updated_at)) AS avg_days
FROM deal d
JOIN pipeline_stage ps ON d.stage_id = ps.id
WHERE d.workspace_id = ?
  AND d.status = 'open'
  AND d.deleted_at IS NULL
GROUP BY ps.id;

-- name: CaseVolumeByWorkspace :many
SELECT
    priority,
    status,
    COUNT(*) AS count
FROM case_ticket
WHERE workspace_id = ?
  AND deleted_at IS NULL
GROUP BY priority, status;

-- name: CaseBacklogByWorkspace :many
SELECT
    id,
    priority,
    status,
    created_at,
    CAST(julianday('now') - julianday(created_at) AS INTEGER) AS aging_days
FROM case_ticket
WHERE workspace_id = ?
  AND status IN ('open', 'in_progress', 'waiting', 'escalated')
  AND deleted_at IS NULL
  AND (julianday('now') - julianday(created_at)) > (sqlc.arg(aging_days) + 0)
ORDER BY created_at ASC;

-- name: CaseMTTRByWorkspace :many
SELECT
    priority,
    AVG(julianday(updated_at) - julianday(created_at)) AS avg_resolution_days
FROM case_ticket
WHERE workspace_id = ?
  AND status IN ('closed', 'resolved')
  AND deleted_at IS NULL
GROUP BY priority;