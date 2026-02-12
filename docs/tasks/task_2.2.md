# Task 2.2: Ingestion Pipeline (Chunker + IngestService + HTTP Handler)

> **Status**: ✅ Complete
> **Priority**: P0 (Phase 2, Week 4)
> **Approach**: TDD (Test-First)
> **Depends on**: Task 2.1 (knowledge tables, migrations 011/012)
> **Based on**: `docs/implementation-plan.md` Section 4 — Phase 2, Week 4

---

## Overview

Implement the ingestion pipeline that transforms raw content (text, HTML, documents)
into indexed knowledge items with chunked embedding documents ready for Task 2.4.

**Pipeline flow**:
```
POST /api/v1/knowledge/ingest
  → IngestService.Ingest()
    → Normalize content (strip HTML)
    → Create knowledge_item (DB)
    → Chunker.Chunk() → []string
    → Create embedding_document per chunk (status=pending)
    → EventBus.Publish("knowledge.ingested", itemID)
  → 201 Created + KnowledgeItem JSON
```

---

## Goals

1. Chunker splits text into fixed-size token windows with overlap (512 tokens, 50 overlap)
2. IngestService creates `knowledge_item` + `embedding_document` records atomically
3. Idempotent: ingest of same entity (workspace+entity_type+entity_id) updates, not duplicates
4. Event bus (in-memory Go channels) notifies downstream consumers (Task 2.4 embedder)
5. HTTP handler exposes `POST /api/v1/knowledge/ingest` behind JWT auth middleware

---

## Files Affected

| File | Action |
|------|--------|
| `docs/tasks/task_2.2.md` | Created (this file) |
| `internal/domain/knowledge/chunker.go` | Created — whitespace tokenizer + chunking |
| `internal/domain/knowledge/chunker_test.go` | Created — unit tests for chunker |
| `internal/domain/knowledge/ingest.go` | Created — IngestService |
| `internal/domain/knowledge/ingest_test.go` | Created — integration tests for IngestService |
| `internal/infra/eventbus/eventbus.go` | Created — in-memory pub/sub |
| `internal/infra/eventbus/eventbus_test.go` | Created — unit tests for event bus |
| `internal/api/handlers/knowledge_ingest.go` | Created — HTTP handler |
| `internal/api/handlers/knowledge_ingest_test.go` | Created — handler tests |
| `internal/api/routes.go` | Modified — register knowledge route |

---

## Architecture Decisions

### Chunker (whitespace tokenizer)
- No external dependencies (MVP constraint)
- Token = whitespace-separated word
- Default: `chunkSize=512`, `overlap=50`
- Empty/whitespace-only input → returns empty slice (no chunks created)
- Short text (< chunkSize tokens) → returns 1 chunk

### IngestService transaction scope
- Uses `sql.Tx` to wrap `knowledge_item` creation + all `embedding_document` inserts
- If any insert fails → full rollback (no partial state)
- Idempotency: check `UNIQUE(workspace_id, entity_type, entity_id)` on knowledge_item
  - If exists: update `raw_content` + `normalized_content` + `updated_at`, delete old chunks, insert new chunks
  - If not exists: insert new item + chunks

### Event Bus
- Buffered Go channel per topic (buffer=100)
- `Publish` is non-blocking (drops event if buffer full, logs warning)
- `Subscribe` returns a read-only channel — caller owns consumption loop
- No persistence: events are fire-and-forget in MVP
- Interface-based for testability (`EventBus` interface)

### HTTP Handler
- Follows existing pattern: `KnowledgeIngestHandler` struct with `ingestService` field
- Request body: JSON with `source_type`, `title`, `raw_content`, optional `entity_type`/`entity_id`/`metadata`
- Response: 201 Created + `KnowledgeItem` JSON on success
- Workspace ID from JWT claims (same as all other handlers)

---

## Test Plan (TDD — write tests first)

### chunker_test.go (unit, no DB)
- `TestChunker_EmptyInput_ReturnsNoChunks`
- `TestChunker_ShortText_ReturnsSingleChunk`
- `TestChunker_LongText_ReturnsMultipleChunks`
- `TestChunker_OverlapPreservesTokens`
- `TestChunker_ExactChunkSize_ReturnsSingleChunk`

### ingest_test.go (integration, uses DB)
- `TestIngestService_CreateItem_And_Chunks` — creates knowledge_item + embedding_documents
- `TestIngestService_ChunksHaveStatusPending` — all chunks start as pending
- `TestIngestService_EmptyContent_StillIngests` — empty doc = 0 chunks, item created
- `TestIngestService_Idempotent_SameEntity_Updates` — second ingest same entity replaces chunks
- `TestIngestService_WorkspaceIsolation` — items from different workspaces are isolated
- `TestIngestService_PublishesEvent` — event bus receives `knowledge.ingested` after ingest

### eventbus_test.go (unit, no DB)
- `TestEventBus_PublishAndSubscribe`
- `TestEventBus_MultipleSubscribers`
- `TestEventBus_NonBlockingPublish`

### knowledge_ingest_test.go (handler, integration — real DB + real IngestService)
- `TestKnowledgeIngestHandler_Success_Returns201`
- `TestKnowledgeIngestHandler_MissingTitle_Returns400`
- `TestKnowledgeIngestHandler_MissingSourceType_Returns400`
- `TestKnowledgeIngestHandler_InvalidSourceType_Returns400`
- `TestKnowledgeIngestHandler_NoWorkspaceContext_Returns401`

> **Nota de implementación**: se optó por integración real (DB en memoria + IngestService real) en lugar de mock.
> Mayor confianza de integración entre capas. Tests de error difíciles de forzar (BeginTx, rollback) cubiertos en `ingest_test.go`.

---

## Task Log

| Date | Action | Status |
|------|--------|--------|
| 2026-02-10 | Task document created | ✅ |
| | Write chunker tests + implement chunker | ⏳ |
| | Write ingest tests + implement IngestService | ⏳ |
| | Implement event bus | ⏳ |
| | Implement HTTP handler + register route | ⏳ |
| | make test + make complexity pass | ⏳ |

---

**Next Action**: Write `chunker_test.go` (failing), then implement `chunker.go`.
