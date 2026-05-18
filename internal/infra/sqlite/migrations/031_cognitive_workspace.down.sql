-- Migration 031 rollback: drop Blackboard cognitive workspace domain (Task R.13)
-- Reverses 031_cognitive_workspace.up.sql in dependency order (children before parents)

-- 1. Remove the opt-in column added to agent_run
ALTER TABLE agent_run DROP COLUMN cognitive_workspace_id;

-- 2. Drop agent_memory indexes and table
DROP INDEX IF EXISTS idx_agent_memory_workspace_scope;
DROP INDEX IF EXISTS idx_agent_memory_workspace;
DROP TABLE IF EXISTS agent_memory;

-- 3. Drop signal_hypothesis indexes and table
DROP INDEX IF EXISTS idx_signal_hypothesis_workspace_status;
DROP INDEX IF EXISTS idx_signal_hypothesis_workspace;
DROP TABLE IF EXISTS signal_hypothesis;

-- 4. Drop reasoning_event indexes and table
DROP INDEX IF EXISTS idx_reasoning_event_workspace_created;
DROP INDEX IF EXISTS idx_reasoning_event_workspace_type;
DROP INDEX IF EXISTS idx_reasoning_event_workspace;
DROP TABLE IF EXISTS reasoning_event;

-- 5. Drop cognitive_workspace indexes and table (parent last)
DROP INDEX IF EXISTS idx_cognitive_workspace_status;
DROP INDEX IF EXISTS idx_cognitive_workspace_agent_run;
DROP INDEX IF EXISTS idx_cognitive_workspace_workspace;
DROP TABLE IF EXISTS cognitive_workspace;
