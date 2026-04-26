---
id: ADR-013
title: "Embedding generation is async: status=pending → event bus → embedder → retry"
date: 2026-01-24
status: accepted
deciders: [matias]
tags: [adr, rag, embeddings, async, event-bus]
related_tasks: [task_2.4]
related_frs: [FR-091, FR-092]
---

# ADR-013 — Embedding generation is async: status=pending → event bus → embedder → retry

## Status

`accepted`

## Context

Generating embeddings requires an LLM API call (or a local model inference). This is
an I/O-bound operation with variable latency (50ms–5s depending on provider). Doing it
synchronously during the ingestion HTTP request would:

1. Block the ingestion endpoint for seconds per document
2. Couple ingestion success to embedding provider availability
3. Make bulk ingestion (many documents at once) impractical

## Decision

Embedding generation is fully asynchronous, using a three-phase pipeline:

**Phase 1 — Ingestion (synchronous):**
```
POST /api/v1/knowledge/ingest
  → Create knowledge_item (status='active')
  → Create embedding_document rows per chunk (status='pending')
  → Publish event: knowledge.ingested { knowledge_item_id }
  → Return 201 immediately
```

**Phase 2 — Embedding (asynchronous):**
```
Embedder subscribes to knowledge.ingested
  → For each chunk in embedding_document WHERE status='pending':
      → Call LLM.Embed(chunk.text) → vector
      → UPDATE embedding_document SET status='embedded', vector=...
      → INSERT INTO vec_embedding (id, embedding) VALUES (chunk.id, vector)
```

**Phase 3 — Retry on failure:**
```
If LLM.Embed() fails:
  → Attempt 1: immediate retry
  → Attempt 2: backoff 1s
  → Attempt 3: backoff 4s
  → After 3 failures: UPDATE embedding_document SET status='failed'
  → Log to audit_event for visibility
```

Status transitions: `pending` → `embedded` (success) | `failed` (3 failures)

## Rationale

- Ingestion latency is decoupled from embedding provider latency — P95 ingestion stays fast
- If the embedding provider is temporarily unavailable, documents remain `pending` and
  can be retried via the manual reindex endpoint
- Exponential backoff (1s, 4s) avoids hammering a degraded provider
- The status field makes the pipeline observable — operators can query `pending` and
  `failed` counts to detect stuck embeddings
- Event-driven design matches the existing Go channel-based event bus architecture

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Synchronous embedding during ingestion | Blocks HTTP request for 1–5s; couples ingestion to provider availability |
| Background goroutine per document (no event bus) | Hard to track, no retry logic, goroutine leak risk |
| Polling loop (check pending every N seconds) | Higher latency than event-driven; unnecessary DB load |
| Batch embedding (accumulate then embed) | Adds batching complexity; higher latency for first document |

## Consequences

**Positive:**
- Ingestion endpoint responds in <100ms regardless of LLM provider latency
- Retry logic handles transient provider failures transparently
- `pending`/`failed` status is queryable — observable pipeline

**Negative / tradeoffs:**
- Newly ingested documents are not immediately searchable — there is a delay until
  embedding completes (typically <5s for a short document with a warm provider)
- `failed` embeddings require manual intervention (re-trigger via POST /knowledge/reindex)

## References

- `internal/domain/knowledge/embedder.go` — Embedder service
- `internal/infra/eventbus/` — Go channel-based event bus
- `docs/tasks/task_2.4.md` — embedding pipeline design
- `docs/tasks/task_2.7.md` — CDC and reindex (manual retry endpoint)
