---
id: ADR-016
title: "CDC reindex SLA: changes visible in search within 60 seconds"
date: 2026-01-30
status: accepted
deciders: [matias]
tags: [adr, cdc, search, reindex, sla, event-bus]
related_tasks: [task_2.7]
related_frs: [FR-090, FR-091]
---

# ADR-016 — CDC reindex SLA: changes visible in search within 60 seconds

## Status

`accepted`

## Context

CRM records (Account, Contact, Deal, Case) and Knowledge Items change frequently.
When a record is updated, the search index (FTS5 + vector) must reflect the new content
for the copilot and agents to provide accurate, grounded responses.

Without a Change Data Capture (CDC) mechanism, search results can remain stale
indefinitely — the LLM may cite outdated deal stages, resolved cases, or replaced contacts.

## Decision

Implement CDC using the existing Go channel-based event bus. The target SLA is:

**Changes must be visible in search within 60 seconds of the source record being written.**

**CDC flow:**

```
1. CRM handler writes to DB → publishes event:
   record.created | record.updated | record.deleted
   { entity_type, entity_id, workspace_id }

2. CDC subscriber receives event → determines if search index is affected:
   - Text fields changed? → trigger FTS5 reindex for this item
   - Attachments/notes changed? → trigger re-embedding for affected chunks

3. FTS5 reindex: UPDATE knowledge_item_fts (via trigger — automatic for knowledge_item)
4. Re-embedding: publish knowledge.ingested for affected embedding_document rows
   → Embedder picks up and regenerates vectors (see ADR-013)

5. Manual full reindex available:
   POST /api/v1/knowledge/reindex
   → Rebuilds FTS5 and re-queues all embedding_document rows as pending
```

**60-second SLA rationale:**

For CRM content, a 60-second delay between write and search visibility is acceptable:
- Sales reps do not expect real-time search during the same conversation where they edit
- Agents run on a per-trigger basis — they will not query within milliseconds of a write
- 60s allows the async embedding pipeline to complete for typical document sizes

## Rationale

- Event-driven CDC is already consistent with the event bus architecture used for
  knowledge ingestion (ADR-013) — no new infrastructure required
- The manual reindex endpoint provides an operator escape hatch when the SLA is missed
  (e.g., after a provider outage that left embeddings in `failed` state)
- FTS5 reindex via triggers (ADR-011) means keyword search is always up-to-date
  synchronously — only vector search has the 60s async window

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Polling loop (check for new records every N seconds) | Higher DB load; latency equals poll interval |
| Database CDC via SQLite WAL parsing | Complex; SQLite WAL is not designed for consumer parsing |
| Real-time reindex on every write (synchronous) | Blocks write path; unacceptable latency for bulk operations |
| No CDC (manual reindex only) | Stale search results by default; poor copilot quality |

## Consequences

**Positive:**
- Search results are fresh within 60s without manual intervention
- Manual reindex provides recovery from outages
- CDC uses existing event bus — no new infrastructure

**Negative / tradeoffs:**
- 60s window means a copilot query immediately after a deal update may return stale
  stage information — UI should surface a "last indexed" timestamp for transparency
- CDC subscriber adds a write amplification pattern — every CRM write generates an
  additional event + potential re-embedding cost

## References

- `internal/domain/cdc/` — CDC subscriber implementation
- `internal/infra/eventbus/` — Go channel-based event bus
- `docs/tasks/task_2.7.md` — CDC and reindex SLA design
- ADR-013 — embedding async pipeline (re-embedding triggered by CDC)
