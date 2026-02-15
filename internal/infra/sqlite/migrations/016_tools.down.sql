-- Migration 016 rollback: Tool registry foundation

DROP INDEX IF EXISTS idx_tool_definition_workspace_active;
DROP INDEX IF EXISTS idx_tool_definition_workspace;
DROP TABLE IF EXISTS tool_definition;
