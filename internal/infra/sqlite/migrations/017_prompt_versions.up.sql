-- Task 3.9: Prompt Versioning
-- Tabla para versionado de prompts de agentes
-- agent_definition_id NO tiene FK formal (tabla se crea en migration 018)

CREATE TABLE prompt_version (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspace(id),
    agent_definition_id TEXT NOT NULL,
    version_number INTEGER NOT NULL,
    system_prompt TEXT NOT NULL,
    user_prompt_template TEXT,
    config TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'draft',
    created_by TEXT REFERENCES user_account(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_definition_id, version_number)
);

CREATE INDEX idx_prompt_version_agent_status
    ON prompt_version(agent_definition_id, status);
