-- Local rollback only. Production rollback requires separate operator approval.
DROP INDEX IF EXISTS idx_service_identity_audit_principal;
DROP TABLE IF EXISTS service_identity_audit;
DROP INDEX IF EXISTS idx_service_credential_one_active;
DROP INDEX IF EXISTS idx_service_credential_principal;
DROP TABLE IF EXISTS service_credential;
DROP INDEX IF EXISTS idx_service_principal_workspace_org;
DROP TABLE IF EXISTS service_principal;
