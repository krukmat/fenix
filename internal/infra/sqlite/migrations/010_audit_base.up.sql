-- Migration 010: Audit Logging Foundation
-- Creates the audit_event table for immutable audit trail
-- Related to: Task 1.7, FR-070, NFR-031

-- audit_event table: append-only, immutable audit trail
-- IMPORTANT: No updates or deletes should ever be performed on this table
CREATE TABLE audit_event (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    actor_type TEXT NOT NULL CHECK(actor_type IN ('user', 'agent', 'system')),
    action TEXT NOT NULL,                    -- e.g., 'login', 'create_account', 'delete_deal'
    entity_type TEXT,                        -- e.g., 'account', 'contact', 'case' (nullable for auth actions)
    entity_id TEXT,                          -- UUID of affected entity (nullable)
    details JSON,                            -- Flexible JSON: { old_value, new_value, changes, metadata }
    permissions_checked JSON,                -- Permissions verified: [{ permission, result }]
    outcome TEXT NOT NULL CHECK(outcome IN ('success', 'denied', 'error')),
    trace_id TEXT,                           -- For distributed tracing correlation
    ip_address TEXT,                         -- Client IP
    user_agent TEXT,                         -- Client user agent
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (workspace_id) REFERENCES workspace(id)
);

-- Indexes for common query patterns
-- Query by workspace (tenant isolation)
CREATE INDEX idx_audit_workspace ON audit_event(workspace_id);

-- Query by actor (user/agent activity)
CREATE INDEX idx_audit_actor ON audit_event(actor_id);

-- Query by entity (history of a specific record)
CREATE INDEX idx_audit_entity ON audit_event(entity_type, entity_id);

-- Query by date range (time-based analysis)
CREATE INDEX idx_audit_created ON audit_event(created_at);

-- Query by outcome (filter errors/denials)
CREATE INDEX idx_audit_outcome ON audit_event(outcome);

-- Query by action type (filter specific operations)
CREATE INDEX idx_audit_action ON audit_event(action);

-- Query by trace_id (distributed tracing)
CREATE INDEX idx_audit_trace ON audit_event(trace_id);
