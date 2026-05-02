-- F2-T1: Queries for ActualRunTrace enrichment.
-- Read-side only. Does not modify agent_run schema.

-- name: ListApprovalRequestsByIDs :many
-- F2-T1: fetch approval requests by a list of IDs for trace enrichment.
-- Caller extracts approval_id values from audit_event.details for a given trace_id.
SELECT * FROM approval_request
WHERE id IN (sqlc.slice(ids))
ORDER BY created_at ASC;
