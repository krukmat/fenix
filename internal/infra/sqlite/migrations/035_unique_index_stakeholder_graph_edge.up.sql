-- Migration 035: enforce one stakeholder edge per logical relationship.

CREATE UNIQUE INDEX IF NOT EXISTS uq_stakeholder_graph_edge
    ON stakeholder_graph(
        workspace_id,
        from_entity_type,
        from_entity_id,
        to_entity_type,
        to_entity_id,
        influence_type
    );
