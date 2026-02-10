-- Migration 010: Rollback - Remove audit_event table

DROP INDEX IF EXISTS idx_audit_trace;
DROP INDEX IF EXISTS idx_audit_action;
DROP INDEX IF EXISTS idx_audit_outcome;
DROP INDEX IF EXISTS idx_audit_created;
DROP INDEX IF EXISTS idx_audit_entity;
DROP INDEX IF EXISTS idx_audit_actor;
DROP INDEX IF EXISTS idx_audit_workspace;
DROP TABLE IF EXISTS audit_event;
