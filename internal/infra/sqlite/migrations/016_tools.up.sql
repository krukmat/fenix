-- Migration 016: Tool registry foundation (Task 3.3)

CREATE TABLE IF NOT EXISTS tool_definition (
    id                    TEXT PRIMARY KEY,
    workspace_id          TEXT NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,
    description           TEXT,
    input_schema          JSON NOT NULL,
    required_permissions  JSON NOT NULL DEFAULT '[]',
    is_active             INTEGER NOT NULL DEFAULT 1,
    created_by            TEXT,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_tool_definition_workspace
    ON tool_definition(workspace_id);

CREATE INDEX IF NOT EXISTS idx_tool_definition_workspace_active
    ON tool_definition(workspace_id, is_active);
