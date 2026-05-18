---
id: ADR-101
title: "Relationship Memory Engine"
date: 2026-05-16
status: accepted
deciders: [matias]
tags: [adr, architecture, relationship-memory, trust, stakeholder-graph, agentic-upgrade]
related_tasks: []
related_frs: []
---

# ADR-101 — Relationship Memory Engine

## Status

`accepted`

## Context

Traditional CRMs store records. FenixCRM must evolve toward operational relationship cognition — understanding not just what happened but the evolving trust, influence, and behavioral patterns behind each stakeholder relationship.

## Decision

FenixCRM shall implement a Relationship Memory Engine that persists:
- stakeholder influence and trust evolution
- negotiation history
- inferred intent and communication tone
- relationship trajectory
- historical interaction embeddings

## Rationale

Relationship cognition is the hardest-to-copy moat in AI CRM. Structured records are commoditized; understanding the evolution of trust and intent across time creates differentiated, contextual intelligence that improves with every interaction.

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Store only raw activity records | Loses the derived cognition layer; no trust or intent signals |
| External graph database | Adds infra complexity; SQLite + sqlite-vec covers the MVP scope |
| LLM-only relationship summary on demand | Not persistent; loses historical trajectory; expensive per query |

## Consequences

**Positive:**
- Stronger contextual intelligence per customer
- Harder-to-copy competitive moat
- Differentiated customer cognition
- Richer autonomous workflows (agents can reason about trust, not just activity)

**Negative / tradeoffs:**
- Memory drift risk (mitigated by immutable interaction signals + trust score versioning)
- Privacy concerns (PII policy enforced via policy engine before storage)
- Higher storage complexity (sqlite-vec for embeddings, per ADR-013)

## New Domains

- `relationship_memory` — derived narrative per account/contact pair
- `interaction_signal` — raw typed signal per CRM event
- `stakeholder_graph` — graph edges between entities (trust, influence)
- `trust_score` — versioned trust metric per entity pair

## Implementation Direction

**Phase B (complete):**
- Interaction summarization (`Summarizer`)
- Trust evolution scoring (`TrustEngine`)
- Stakeholder graph extraction (`GraphExtractor`)
- Relationship memory embedding (`MemoryEmbedder`)
- All four engines wired to shared event bus in `main`

## References

- Remediation plan: `docs/plans/fenixcrm_agentic_upgrade_remediation_plan.md`
- `internal/domain/relationship/` — implementation root
- R.1–R.6: Phase B operability remediation tasks

## Changelog

- 2026-05-16: Created as `Proposed` in `new reqs/fenixcrm_agentic_upgrade_pack/`
- 2026-05-18: Promoted to `docs/decisions/` with status `Accepted`
