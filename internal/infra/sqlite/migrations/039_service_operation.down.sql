-- LOCAL TEST ROLLBACK ONLY. Production rollback is destructive and needs HITL.
DROP TRIGGER IF EXISTS service_operation_immutable_identity;
DROP TRIGGER IF EXISTS service_operation_scope_insert;
DROP INDEX IF EXISTS idx_service_operation_status_lease;
DROP INDEX IF EXISTS idx_service_operation_workspace_created;
DROP TABLE IF EXISTS service_operation;
