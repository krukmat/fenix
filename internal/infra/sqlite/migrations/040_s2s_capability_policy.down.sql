-- LOCAL TEST ROLLBACK ONLY. Production policy/audit deletion requires HITL.
DROP TRIGGER IF EXISTS service_s2s_capability_audit_no_delete;
DROP TRIGGER IF EXISTS service_s2s_capability_audit_no_update;
DROP TRIGGER IF EXISTS service_s2s_capability_no_delete;
DROP TRIGGER IF EXISTS service_s2s_capability_audit_update;
DROP TRIGGER IF EXISTS service_s2s_capability_audit_insert;
DROP TRIGGER IF EXISTS service_s2s_capability_enable_guard;
DROP TRIGGER IF EXISTS service_s2s_capability_immutable_fields;
DROP TRIGGER IF EXISTS service_s2s_capability_insert_guard;
DROP INDEX IF EXISTS idx_service_s2s_capability_audit_scope;
DROP TABLE IF EXISTS service_s2s_capability_audit;
DROP INDEX IF EXISTS idx_service_s2s_capability_active;
DROP TABLE IF EXISTS service_s2s_capability_policy;
