-- Rollback Migration 015: Approval Workflow foundation (Task 3.2)

DROP INDEX IF EXISTS idx_approval_request_decided_by_decided_at;
DROP INDEX IF EXISTS idx_approval_request_approver_by_status;
DROP INDEX IF EXISTS idx_approval_request_requested_by_status;
DROP INDEX IF EXISTS idx_approval_request_workspace_status_expiry;

DROP TABLE IF EXISTS approval_request;
