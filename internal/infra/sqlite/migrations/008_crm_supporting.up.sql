-- Migration 008: Supporting CRM Entities
-- Task 1.5: activity, note, attachment, timeline_event tables
-- Supporting entities for CRM operations and audit trail

-- Activity: Tasks, events, calls, emails linked to any entity
CREATE TABLE IF NOT EXISTS activity (
    id            TEXT    NOT NULL PRIMARY KEY,          -- UUID v7
    workspace_id  TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    activity_type TEXT    NOT NULL
                         CHECK (activity_type IN ('task', 'event', 'call', 'email')),
                                                         -- Type of activity
    entity_type   TEXT    NOT NULL
                         CHECK (entity_type IN ('account', 'contact', 'deal', 'case')),
                                                         -- Polymorphic: what entity this activity is for
    entity_id     TEXT    NOT NULL,                      -- FK to the entity (polymorphic)
    owner_id      TEXT    NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
                                                         -- Who created/owns this activity
    assigned_to   TEXT    REFERENCES user_account(id) ON DELETE SET NULL,
                                                         -- Who should complete it (nullable)
    subject       TEXT    NOT NULL,                      -- Activity title/subject
    body          TEXT,                                  -- Description/details
    status        TEXT    NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending', 'completed', 'cancelled')),
                                                         -- Activity status
    due_at        TEXT,                                  -- When it's due (ISO 8601)
    completed_at  TEXT,                                  -- When it was completed
    metadata      TEXT,                                  -- JSON: recurrence, reminders, etc.
    created_at    TEXT    NOT NULL,                      -- ISO 8601 UTC
    updated_at    TEXT    NOT NULL
);

-- Note: Internal notes/comments on any entity
CREATE TABLE IF NOT EXISTS note (
    id            TEXT    NOT NULL PRIMARY KEY,          -- UUID v7
    workspace_id  TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    entity_type   TEXT    NOT NULL
                         CHECK (entity_type IN ('account', 'contact', 'deal', 'case')),
                                                         -- Polymorphic: what entity this note is for
    entity_id     TEXT    NOT NULL,                      -- FK to the entity (polymorphic)
    author_id     TEXT    NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
                                                         -- Who wrote the note
    content       TEXT    NOT NULL,                      -- Note content (markdown supported)
    is_internal   INTEGER NOT NULL DEFAULT 0
                         CHECK (is_internal IN (0, 1)),
                                                         -- 1 = internal only (not visible to customer)
    metadata      TEXT,                                  -- JSON: mentions, attachments refs, etc.
    created_at    TEXT    NOT NULL,                      -- ISO 8601 UTC
    updated_at    TEXT    NOT NULL
);

-- Attachment: File metadata for uploads linked to any entity
CREATE TABLE IF NOT EXISTS attachment (
    id            TEXT    NOT NULL PRIMARY KEY,          -- UUID v7
    workspace_id  TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    entity_type   TEXT    NOT NULL,                      -- Polymorphic: account, contact, deal, case, etc.
    entity_id     TEXT    NOT NULL,                      -- FK to the entity (polymorphic)
    uploader_id   TEXT    NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
                                                         -- Who uploaded the file
    filename      TEXT    NOT NULL,                      -- Original filename
    content_type  TEXT,                                  -- MIME type (e.g., "application/pdf")
    size_bytes    INTEGER,                               -- File size in bytes
    storage_path  TEXT    NOT NULL,                      -- Path in storage (e.g., "./data/attachments/...")
    sensitivity   TEXT    DEFAULT 'internal'
                         CHECK (sensitivity IN ('public', 'internal', 'confidential', 'pii')),
                                                         -- Data sensitivity level
    metadata      TEXT,                                  -- JSON: checksum, encryption info, etc.
    created_at    TEXT    NOT NULL                       -- ISO 8601 UTC
);

-- Timeline Event: Immutable audit trail of entity changes
CREATE TABLE IF NOT EXISTS timeline_event (
    id            TEXT    NOT NULL PRIMARY KEY,          -- UUID v7
    workspace_id  TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    entity_type   TEXT    NOT NULL,                      -- Type of entity affected
    entity_id     TEXT    NOT NULL,                      -- ID of entity affected
    actor_id      TEXT    REFERENCES user_account(id) ON DELETE SET NULL,
                                                         -- Who made the change (NULL = system/agent)
    event_type    TEXT    NOT NULL
                         CHECK (event_type IN ('created', 'updated', 'deleted', 'stage_changed', 'note_added', 'activity_completed', 'agent_action')),
                                                         -- Type of event
    old_value     TEXT,                                  -- JSON: previous state (for updates)
    new_value     TEXT,                                  -- JSON: new state
    context       TEXT,                                  -- JSON: agent_run_id, tool_call_id, etc.
    created_at    TEXT    NOT NULL                       -- ISO 8601 UTC
);

-- === INDEXES FOR ACTIVITY ===
CREATE INDEX IF NOT EXISTS idx_activity_workspace      ON activity (workspace_id);
CREATE INDEX IF NOT EXISTS idx_activity_entity         ON activity (workspace_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_activity_owner          ON activity (workspace_id, owner_id);
CREATE INDEX IF NOT EXISTS idx_activity_assigned       ON activity (assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_activity_type           ON activity (workspace_id, activity_type);
CREATE INDEX IF NOT EXISTS idx_activity_status         ON activity (workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_activity_due            ON activity (workspace_id, due_at) WHERE due_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_activity_created        ON activity (workspace_id, created_at DESC);

-- Composite index for "my open tasks" view
CREATE INDEX IF NOT EXISTS idx_activity_my_tasks
    ON activity (assigned_to, status, due_at)
    WHERE assigned_to IS NOT NULL AND status = 'pending';

-- === INDEXES FOR NOTE ===
CREATE INDEX IF NOT EXISTS idx_note_workspace          ON note (workspace_id);
CREATE INDEX IF NOT EXISTS idx_note_entity             ON note (workspace_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_note_author             ON note (workspace_id, author_id);
CREATE INDEX IF NOT EXISTS idx_note_internal           ON note (workspace_id, is_internal) WHERE is_internal = 1;
CREATE INDEX IF NOT EXISTS idx_note_created            ON note (workspace_id, created_at DESC);

-- === INDEXES FOR ATTACHMENT ===
CREATE INDEX IF NOT EXISTS idx_attachment_workspace    ON attachment (workspace_id);
CREATE INDEX IF NOT EXISTS idx_attachment_entity       ON attachment (workspace_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_attachment_uploader     ON attachment (workspace_id, uploader_id);
CREATE INDEX IF NOT EXISTS idx_attachment_sensitivity  ON attachment (workspace_id, sensitivity);
CREATE INDEX IF NOT EXISTS idx_attachment_created      ON attachment (workspace_id, created_at DESC);

-- === INDEXES FOR TIMELINE EVENT ===
CREATE INDEX IF NOT EXISTS idx_timeline_workspace      ON timeline_event (workspace_id);
CREATE INDEX IF NOT EXISTS idx_timeline_entity         ON timeline_event (workspace_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_timeline_actor          ON timeline_event (actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_timeline_event_type     ON timeline_event (workspace_id, event_type);
CREATE INDEX IF NOT EXISTS idx_timeline_created        ON timeline_event (workspace_id, created_at DESC);

-- Composite index for entity history queries
CREATE INDEX IF NOT EXISTS idx_timeline_entity_history
    ON timeline_event (entity_type, entity_id, created_at DESC);
