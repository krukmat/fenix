-- Migration 023: Enforce append-only audit_event
-- Related to: FR-070

CREATE TRIGGER trg_audit_event_no_update
BEFORE UPDATE ON audit_event
BEGIN
    SELECT RAISE(ABORT, 'audit_event is append-only');
END;

CREATE TRIGGER trg_audit_event_no_delete
BEFORE DELETE ON audit_event
BEGIN
    SELECT RAISE(ABORT, 'audit_event is append-only');
END;
