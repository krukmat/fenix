-- Migration 023 rollback: Remove append-only audit_event triggers

DROP TRIGGER IF EXISTS trg_audit_event_no_update;
DROP TRIGGER IF EXISTS trg_audit_event_no_delete;
