-- eval.sql
-- Task 4.7: FR-242 Eval Service Basic

-- name: CreateEvalSuite :one
INSERT INTO eval_suite (id, workspace_id, name, domain, test_cases, thresholds)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetEvalSuiteByID :one
SELECT * FROM eval_suite
WHERE id = ? AND workspace_id = ?;

-- name: ListEvalSuites :many
SELECT * FROM eval_suite
WHERE workspace_id = ?
ORDER BY created_at DESC;

-- name: UpdateEvalSuite :exec
UPDATE eval_suite
SET name = ?, domain = ?, test_cases = ?, thresholds = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND workspace_id = ?;

-- name: DeleteEvalSuite :exec
DELETE FROM eval_suite
WHERE id = ? AND workspace_id = ?;

-- name: CreateEvalRun :one
INSERT INTO eval_run (id, workspace_id, eval_suite_id, prompt_version_id, status, scores, details, triggered_by)
VALUES (?, ?, ?, ?, 'running', '{}', '[]', ?)
RETURNING *;

-- name: GetEvalRunByID :one
SELECT * FROM eval_run
WHERE id = ? AND workspace_id = ?;

-- name: ListEvalRuns :many
SELECT * FROM eval_run
WHERE workspace_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListEvalRunsBySuite :many
SELECT * FROM eval_run
WHERE workspace_id = ? AND eval_suite_id = ?
ORDER BY created_at DESC;

-- name: UpdateEvalRunResult :exec
UPDATE eval_run
SET status = ?, scores = ?, details = ?, completed_at = CURRENT_TIMESTAMP
WHERE id = ? AND workspace_id = ?;