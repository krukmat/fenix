> **DEPRECATED** — Promoted to [`docs/decisions/ADR-101-relationship-memory-engine.md`](../../docs/decisions/ADR-101-relationship-memory-engine.md) on 2026-05-18. This copy is retained for audit traceability only. Do not edit.

# ADR-101 — Relationship Memory Engine

Status: Proposed
Date: 2026-05-16

## Context

Traditional CRMs store records.
FenixCRM must evolve toward operational relationship cognition.

## Decision

FenixCRM shall implement a Relationship Memory Engine.

The system shall persist:
- stakeholder influence
- negotiation history
- trust evolution
- inferred intent
- communication tone
- relationship trajectory
- historical interaction embeddings

## Architectural Consequences

New domains:
- relationship_memory
- interaction_signal
- stakeholder_graph
- trust_score

Retrieval shall support:
- semantic relationship history
- temporal memory
- behavioral pattern recall

## Benefits

- stronger contextual intelligence
- harder-to-copy moat
- differentiated customer cognition
- richer autonomous workflows

## Risks

- memory drift
- privacy concerns
- higher storage complexity

## Implementation Direction

Start with:
- interaction summarization
- trust evolution scoring
- stakeholder graph extraction
