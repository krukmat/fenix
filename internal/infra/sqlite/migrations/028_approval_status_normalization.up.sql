-- Migration 028: normalize approval statuses to rejected/cancelled

ALTER TABLE approval_request RENAME TO approval_request_old;

CREATE TABLE approval_request (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    requested_by   TEXT NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
    approver_id    TEXT NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
    decided_by     TEXT REFERENCES user_account(id) ON DELETE SET NULL,

    action         TEXT NOT NULL,
    resource_type  TEXT,
    resource_id    TEXT,
    payload        JSON NOT NULL DEFAULT '{}',
    reason         TEXT,

    status         TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'rejected', 'expired', 'cancelled')),
    expires_at     DATETIME NOT NULL,
    decided_at     DATETIME,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO approval_request (
    id, workspace_id, requested_by, approver_id, decided_by,
    action, resource_type, resource_id, payload, reason,
    status, expires_at, decided_at, created_at, updated_at
)
SELECT
    id, workspace_id, requested_by, approver_id, decided_by,
    action, resource_type, resource_id, payload, reason,
    CASE status
        WHEN 'denied' THEN 'rejected'
        ELSE status
    END,
    expires_at, decided_at, created_at, updated_at
FROM approval_request_old;

DROP TABLE approval_request_old;

CREATE INDEX idx_approval_request_workspace_status_expiry
    ON approval_request(workspace_id, status, expires_at);

CREATE INDEX idx_approval_request_requested_by_status
    ON approval_request(requested_by, status);

CREATE INDEX idx_approval_request_approver_by_status
    ON approval_request(approver_id, status);

CREATE INDEX idx_approval_request_decided_by_decided_at
    ON approval_request(decided_by, decided_at);
