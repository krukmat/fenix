-- Roll back migration 034 by restoring the original stakeholder_graph CHECK constraints.
-- This rollback is only safe when no row still references the user entity type.

CREATE TEMP TABLE stakeholder_graph_user_guard (
    user_rows_absent INTEGER NOT NULL CHECK(user_rows_absent = 1)
);

INSERT INTO stakeholder_graph_user_guard (user_rows_absent)
SELECT CASE
    WHEN EXISTS (
        SELECT 1
        FROM stakeholder_graph
        WHERE from_entity_type = 'user' OR to_entity_type = 'user'
    ) THEN 0
    ELSE 1
END;

DROP TABLE stakeholder_graph_user_guard;

CREATE TABLE stakeholder_graph_old (
    id               TEXT     NOT NULL PRIMARY KEY,
    workspace_id     TEXT     NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    from_entity_type TEXT     NOT NULL
                              CHECK(from_entity_type IN ('account', 'contact', 'lead', 'deal', 'case')),
    from_entity_id   TEXT     NOT NULL,
    to_entity_type   TEXT     NOT NULL
                              CHECK(to_entity_type IN ('account', 'contact', 'lead', 'deal', 'case')),
    to_entity_id     TEXT     NOT NULL,
    influence_type   TEXT     NOT NULL
                              CHECK(influence_type IN ('reports_to', 'influences', 'blocks', 'collaborates', 'approves')),
    strength         REAL     NOT NULL DEFAULT 0.5
                              CHECK(strength >= 0.0 AND strength <= 1.0),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO stakeholder_graph_old (
    id,
    workspace_id,
    from_entity_type,
    from_entity_id,
    to_entity_type,
    to_entity_id,
    influence_type,
    strength,
    created_at,
    updated_at
)
SELECT
    id,
    workspace_id,
    from_entity_type,
    from_entity_id,
    to_entity_type,
    to_entity_id,
    influence_type,
    strength,
    created_at,
    updated_at
FROM stakeholder_graph;

DROP TABLE stakeholder_graph;

ALTER TABLE stakeholder_graph_old RENAME TO stakeholder_graph;

CREATE INDEX idx_stakeholder_graph_workspace
    ON stakeholder_graph(workspace_id);

CREATE INDEX idx_stakeholder_graph_from
    ON stakeholder_graph(workspace_id, from_entity_type, from_entity_id);

CREATE INDEX idx_stakeholder_graph_to
    ON stakeholder_graph(workspace_id, to_entity_type, to_entity_id);
