-- B2.A2 PROPOSAL ONLY. DESTRUCTIVE rollback needs explicit operator approval.
DROP INDEX IF EXISTS idx_service_credential_principal;
DROP TABLE IF EXISTS service_credential;
DROP INDEX IF EXISTS idx_service_principal_workspace_org;
DROP TABLE IF EXISTS service_principal;
