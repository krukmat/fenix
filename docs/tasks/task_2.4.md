# Task 2.4 — Embedder & Indexing

**Status**: ✅ Complete
**Phase**: 2 — Knowledge & Retrieval
**Goal**: Async service that consumes `knowledge.ingested` events, calls `LLMProvider.Embed()` per batch, stores float32 vectors in `vec_embedding`, and marks chunks as `embedded`.

---

## Goals

1. `EmbedderService` subscribes to `TopicKnowledgeIngested` and processes pending chunks
2. Batch embedding via `LLMProvider.Embed()` (one call per knowledge_item, not per chunk)
3. Retry with exponential backoff (3 attempts: 100ms, 200ms, 400ms)
4. `vec_embedding` table stores vectors as JSON TEXT (pure-Go compatible, no sqlite-vec CGO)
5. Multi-tenant isolation: `workspace_id` on every vec_embedding row
6. Wiring: shared eventbus between IngestService and EmbedderService in routes.go

---

## Architecture

```
IngestService.Ingest()
  → embedding_document rows (status=pending)
  → bus.Publish("knowledge.ingested", {KnowledgeItemID, WorkspaceID, ChunkCount})
      ↓ (async goroutine)
EmbedderService.Start()
  → EmbedChunks(ctx, knowledgeItemID, workspaceID)
      → fetchPendingChunks()
      → callEmbedWithRetry() → LLMProvider.Embed([]texts)
      → storeVectors() → INSERT vec_embedding + UPDATE embedding_document status=embedded
```

---

## Design Decisions

1. **JSON TEXT vector storage** (not sqlite-vec): `modernc.org/sqlite` is pure-Go, no CGO extension support. Vectors stored as `"[0.1,0.2,...]"`. Task 2.5 will implement cosine similarity in Go or evaluate a CGO-capable driver.

2. **Batch embedding**: All chunks for a knowledge_item sent as a single `EmbedRequest.Texts` array. Reduces HTTP round-trips to Ollama (N chunks = 1 call, not N calls).

3. **Async non-blocking**: `go embedder.Start(ctx, bus)`. IngestService returns immediately after publishing event.

4. **Shared eventbus**: `routes.go` creates one `eventbus.Bus`, wired to both IngestService (publish) and EmbedderService (subscribe).

5. **TDD with stub LLMProvider**: `stubEmbedder` implements `llm.LLMProvider` — deterministic, no real Ollama required.

---

## Files Affected

| File | Action | Lines (after task) |
|------|--------|--------------------|
| `docs/tasks/task_2.4.md` | Create | this file |
| `internal/infra/sqlite/migrations/013_vec_embedding.up.sql` | Create | ~15 |
| `internal/infra/sqlite/migrations/013_vec_embedding.down.sql` | Create | ~3 |
| `internal/infra/sqlite/queries/knowledge.sql` | Modify | +10 lines |
| `internal/infra/sqlite/sqlcgen/knowledge.sql.go` | Regenerate | auto |
| `internal/infra/sqlite/sqlcgen/querier.go` | Regenerate | auto |
| `internal/domain/knowledge/embedder.go` | Create | ~150 |
| `internal/domain/knowledge/embedder_test.go` | Create | ~200 |
| `internal/api/routes.go` | Modify | wiring ~5 lines |

---

## Tasks Completed

- [x] T1: Create docs/tasks/task_2.4.md
- [x] T2: Migration 013 vec_embedding
- [x] T3: SQL queries + sqlc-generate
- [x] T4: TDD — embedder_test.go (tests first)
- [x] T5: Implement embedder.go
- [x] T6: Wiring in routes.go
- [x] T7: Gates passed (make test + make complexity)
- [x] T8: Commit
