-- W5-C G07-G10: restart-safe evidence delivery, reconciliation and verification lifecycle.
CREATE TABLE evidence_delivery (
    execution_id       TEXT PRIMARY KEY,
    workspace_id       TEXT NOT NULL,
    trace_id           TEXT NOT NULL,
    stream_id          TEXT NOT NULL,
    envelope_json      TEXT NOT NULL,
    proof_json         TEXT,
    delivery_state     TEXT NOT NULL,
    attempt_count      INTEGER NOT NULL DEFAULT 0,
    last_error         TEXT,
    next_attempt_at_ms INTEGER,
    created_at         TEXT NOT NULL,
    updated_at         TEXT NOT NULL
);

CREATE INDEX idx_evidence_delivery_due
    ON evidence_delivery(delivery_state, next_attempt_at_ms);

CREATE INDEX idx_evidence_delivery_stream_state
    ON evidence_delivery(stream_id, delivery_state);
