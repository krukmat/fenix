-- Task 3.9: Prompt Versioning queries

-- name: CreatePromptVersion :one
INSERT INTO prompt_version (
    id, workspace_id, agent_definition_id, version_number,
    system_prompt, user_prompt_template, config, status, created_by
) VALUES (?, ?, ?, ?, ?, ?, ?, 'draft', ?)
RETURNING *;

-- name: GetActivePrompt :one
SELECT * FROM prompt_version
WHERE agent_definition_id = ?
  AND workspace_id = ?
  AND status = 'active'
LIMIT 1;

-- name: ListPromptVersionsByAgent :many
SELECT * FROM prompt_version
WHERE agent_definition_id = ?
  AND workspace_id = ?
ORDER BY version_number DESC;

-- name: GetPromptVersionByID :one
SELECT * FROM prompt_version
WHERE id = ? AND workspace_id = ?;

-- name: GetLatestPromptVersionNumber :one
SELECT COALESCE(MAX(version_number), 0) AS max_version
FROM prompt_version
WHERE agent_definition_id = ? AND workspace_id = ?;

-- name: SetPromptStatus :exec
UPDATE prompt_version
SET status = ?
WHERE id = ? AND workspace_id = ?;

-- name: ArchivePreviousActivePrompts :exec
UPDATE prompt_version
SET status = 'archived'
WHERE agent_definition_id = ?
  AND workspace_id = ?
  AND status = 'active'
  AND id != ?;

-- name: GetPreviousArchivedPrompt :one
SELECT * FROM prompt_version
WHERE agent_definition_id = ?
  AND workspace_id = ?
  AND status = 'archived'
ORDER BY version_number DESC
LIMIT 1;
