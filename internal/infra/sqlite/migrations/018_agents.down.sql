-- Migration 018: Rollback agent definitions, skills, and runs (Task 3.7)

DROP INDEX IF EXISTS idx_agent_run_created_at;
DROP INDEX IF EXISTS idx_agent_run_triggered_by;
DROP INDEX IF EXISTS idx_agent_run_status;
DROP INDEX IF EXISTS idx_agent_run_agent;
DROP INDEX IF EXISTS idx_agent_run_workspace;
DROP TABLE IF EXISTS agent_run;

DROP INDEX IF EXISTS idx_skill_definition_status;
DROP INDEX IF EXISTS idx_skill_definition_agent;
DROP INDEX IF EXISTS idx_skill_definition_workspace;
DROP TABLE IF EXISTS skill_definition;

DROP INDEX IF EXISTS idx_agent_definition_type;
DROP INDEX IF EXISTS idx_agent_definition_workspace_status;
DROP INDEX IF EXISTS idx_agent_definition_workspace;
DROP TABLE IF EXISTS agent_definition;
