-- SQL queries for pipeline and pipeline_stage tables
-- Task 1.5: Pipeline management queries

-- === PIPELINE QUERIES ===

-- name: CreatePipeline :exec
INSERT INTO pipeline (id, workspace_id, name, entity_type, settings, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetPipelineByID :one
SELECT id, workspace_id, name, entity_type, settings, created_at, updated_at
FROM pipeline
WHERE id = ?
  AND workspace_id = ?
LIMIT 1;

-- name: ListPipelinesByWorkspace :many
SELECT id, workspace_id, name, entity_type, settings, created_at, updated_at
FROM pipeline
WHERE workspace_id = ?
ORDER BY name ASC
LIMIT ?
OFFSET ?;

-- name: ListPipelinesByEntityType :many
SELECT id, workspace_id, name, entity_type, settings, created_at, updated_at
FROM pipeline
WHERE workspace_id = ?
  AND entity_type = ?
ORDER BY name ASC;

-- name: UpdatePipeline :exec
UPDATE pipeline
SET name = ?,
    entity_type = ?,
    settings = ?,
    updated_at = ?
WHERE id = ?
  AND workspace_id = ?;

-- name: DeletePipeline :exec
DELETE FROM pipeline
WHERE id = ?
  AND workspace_id = ?;

-- name: CountPipelinesByWorkspace :one
SELECT COUNT(*) FROM pipeline
WHERE workspace_id = ?;

-- === PIPELINE STAGE QUERIES ===

-- name: CreatePipelineStage :exec
INSERT INTO pipeline_stage (id, pipeline_id, name, position, probability, sla_hours, required_fields, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPipelineStageByID :one
SELECT id, pipeline_id, name, position, probability, sla_hours, required_fields, created_at, updated_at
FROM pipeline_stage
WHERE id = ?
LIMIT 1;

-- name: ListPipelineStagesByPipeline :many
SELECT id, pipeline_id, name, position, probability, sla_hours, required_fields, created_at, updated_at
FROM pipeline_stage
WHERE pipeline_id = ?
ORDER BY position ASC;

-- name: UpdatePipelineStage :exec
UPDATE pipeline_stage
SET name = ?,
    position = ?,
    probability = ?,
    sla_hours = ?,
    required_fields = ?,
    updated_at = ?
WHERE id = ?;

-- name: DeletePipelineStage :exec
DELETE FROM pipeline_stage
WHERE id = ?;

-- name: DeletePipelineStagesByPipeline :exec
DELETE FROM pipeline_stage
WHERE pipeline_id = ?;

-- name: CountPipelineStagesByPipeline :one
SELECT COUNT(*) FROM pipeline_stage
WHERE pipeline_id = ?;
