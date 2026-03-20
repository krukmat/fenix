-- AGENT_SPEC Phase 2: Workflow foundation queries

-- name: CreateWorkflow :one
INSERT INTO workflow (
    id, workspace_id, agent_definition_id, parent_version_id, name, description,
    dsl_source, spec_source, version, status, created_by_user_id, archived_at,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetWorkflowByID :one
SELECT * FROM workflow
WHERE id = ? AND workspace_id = ?
LIMIT 1;

-- name: GetWorkflowByNameAndVersion :one
SELECT * FROM workflow
WHERE workspace_id = ? AND name = ? AND version = ?
LIMIT 1;

-- name: GetActiveWorkflowByName :one
SELECT * FROM workflow
WHERE workspace_id = ? AND name = ? AND status = 'active'
LIMIT 1;

-- name: ListWorkflowsByWorkspace :many
SELECT * FROM workflow
WHERE workspace_id = ?
ORDER BY created_at DESC;

-- name: ListWorkflowsByStatus :many
SELECT * FROM workflow
WHERE workspace_id = ? AND status = ?
ORDER BY created_at DESC;

-- name: ListWorkflowVersionsByName :many
SELECT * FROM workflow
WHERE workspace_id = ? AND name = ?
ORDER BY version DESC, created_at DESC;

-- name: UpdateWorkflow :one
UPDATE workflow
SET agent_definition_id = ?, description = ?, dsl_source = ?, spec_source = ?,
    status = ?, archived_at = ?, updated_at = ?
WHERE id = ? AND workspace_id = ?
RETURNING *;

-- name: DeleteWorkflow :exec
DELETE FROM workflow
WHERE id = ? AND workspace_id = ?;
