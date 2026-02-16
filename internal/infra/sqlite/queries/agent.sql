-- Task 3.7: Agent queries

-- name: CreateAgentDefinition :one
INSERT INTO agent_definition (
    id, workspace_id, name, description, agent_type, objective,
    allowed_tools, limits, trigger_config, policy_set_id, active_prompt_version_id,
    status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', datetime('now'), datetime('now'))
RETURNING *;

-- name: GetAgentDefinitionByID :one
SELECT * FROM agent_definition
WHERE id = ? AND workspace_id = ?
LIMIT 1;

-- name: GetAgentDefinitionByName :one
SELECT * FROM agent_definition
WHERE workspace_id = ? AND name = ?
LIMIT 1;

-- name: ListAgentDefinitionsByWorkspace :many
SELECT * FROM agent_definition
WHERE workspace_id = ?
ORDER BY created_at DESC;

-- name: ListActiveAgentDefinitionsByWorkspace :many
SELECT * FROM agent_definition
WHERE workspace_id = ? AND status = 'active'
ORDER BY created_at DESC;

-- name: ListAgentDefinitionsByType :many
SELECT * FROM agent_definition
WHERE workspace_id = ? AND agent_type = ?
ORDER BY created_at DESC;

-- name: UpdateAgentDefinition :one
UPDATE agent_definition
SET name = ?, description = ?, agent_type = ?, objective = ?,
    allowed_tools = ?, limits = ?, trigger_config = ?, policy_set_id = ?,
    active_prompt_version_id = ?, status = ?, updated_at = datetime('now')
WHERE id = ? AND workspace_id = ?
RETURNING *;

-- name: DeleteAgentDefinition :exec
DELETE FROM agent_definition
WHERE id = ? AND workspace_id = ?;

-- name: CreateSkillDefinition :one
INSERT INTO skill_definition (
    id, workspace_id, name, description, steps, agent_definition_id,
    status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, 'draft', datetime('now'), datetime('now'))
RETURNING *;

-- name: GetSkillDefinitionByID :one
SELECT * FROM skill_definition
WHERE id = ? AND workspace_id = ?
LIMIT 1;

-- name: ListSkillDefinitionsByAgent :many
SELECT * FROM skill_definition
WHERE agent_definition_id = ?
ORDER BY created_at DESC;

-- name: ListSkillDefinitionsByWorkspace :many
SELECT * FROM skill_definition
WHERE workspace_id = ?
ORDER BY created_at DESC;

-- name: UpdateSkillDefinition :one
UPDATE skill_definition
SET name = ?, description = ?, steps = ?, agent_definition_id = ?,
    status = ?, updated_at = datetime('now')
WHERE id = ? AND workspace_id = ?
RETURNING *;

-- name: DeleteSkillDefinition :exec
DELETE FROM skill_definition
WHERE id = ? AND workspace_id = ?;

-- name: CreateAgentRun :one
INSERT INTO agent_run (
    id, workspace_id, agent_definition_id, triggered_by_user_id,
    trigger_type, trigger_context, status, inputs,
    retrieval_queries, retrieved_evidence_ids, reasoning_trace,
    tool_calls, output, abstention_reason,
    total_tokens, total_cost, latency_ms, trace_id,
    started_at, completed_at, created_at
) VALUES (?, ?, ?, ?, ?, ?, 'running', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), NULL, datetime('now'))
RETURNING *;

-- name: GetAgentRunByID :one
SELECT * FROM agent_run
WHERE id = ? AND workspace_id = ?
LIMIT 1;

-- name: ListAgentRunsByWorkspace :many
SELECT * FROM agent_run
WHERE workspace_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListAgentRunsByAgent :many
SELECT * FROM agent_run
WHERE workspace_id = ? AND agent_definition_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListAgentRunsByUser :many
SELECT * FROM agent_run
WHERE workspace_id = ? AND triggered_by_user_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListAgentRunsByStatus :many
SELECT * FROM agent_run
WHERE workspace_id = ? AND status = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateAgentRunStatus :one
UPDATE agent_run
SET status = ?, completed_at = datetime('now'), updated_at = datetime('now')
WHERE id = ? AND workspace_id = ?
RETURNING *;

-- name: UpdateAgentRun :one
UPDATE agent_run
SET status = ?, inputs = ?, retrieval_queries = ?, retrieved_evidence_ids = ?,
    reasoning_trace = ?, tool_calls = ?, output = ?, abstention_reason = ?,
    total_tokens = ?, total_cost = ?, latency_ms = ?,
    completed_at = CASE WHEN ? THEN datetime('now') ELSE completed_at END,
    updated_at = datetime('now')
WHERE id = ? AND workspace_id = ?
RETURNING *;

-- name: DeleteAgentRun :exec
DELETE FROM agent_run
WHERE id = ? AND workspace_id = ?;

-- name: CountAgentRunsByWorkspace :one
SELECT COUNT(*) as count FROM agent_run
WHERE workspace_id = ?;

-- name: CountAgentRunsByAgent :one
SELECT COUNT(*) as count FROM agent_run
WHERE workspace_id = ? AND agent_definition_id = ?;
