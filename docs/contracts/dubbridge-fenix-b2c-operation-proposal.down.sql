-- PROPOSAL ONLY. DESTRUCTIVE rollback requires explicit operator approval.
DROP INDEX IF EXISTS idx_service_operation_status;
DROP INDEX IF EXISTS idx_service_operation_workspace_created;
DROP TABLE IF EXISTS service_operation;
