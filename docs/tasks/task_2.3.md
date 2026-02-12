# Task 2.3 — LLM Provider Interface

**Status**: ✅ Complete
**Phase**: 2 — Knowledge & Retrieval
**Goal**: Define a model-agnostic LLM provider abstraction and implement the Ollama HTTP adapter + MVP router.

---

## Goals

1. Shared types for LLM requests/responses (`types.go`)
2. `LLMProvider` interface covering `ChatCompletion`, `Embed`, `ModelInfo`, `HealthCheck`
3. `OllamaProvider` HTTP adapter (calls `/api/embeddings` + `/api/chat`, no stream in MVP)
4. MVP `Router` — pass-through to default provider (extensible for policy/budget/fallback in Phase 3)
5. Config via env vars: `LLM_PROVIDER`, `OLLAMA_BASE_URL`, `OLLAMA_MODEL`

---

## Architecture References

- `docs/architecture.md` Section 8 — LLM Adapter Layer
- `docs/implementation-plan.md` Week 5 — Task 2.3
- Feeds into: Task 2.4 (Embedder), Task 3.x (Agent orchestration)

---

## TDD Strategy

- `provider_test.go`: compile-time check that `*OllamaProvider` satisfies `LLMProvider`
- `ollama_test.go`: unit tests using `httptest.NewServer` — no real Ollama needed
- `router_test.go`: unit tests using mock `LLMProvider` (function struct)
- **No mocks in handler tests** (not applicable here — this is infra layer)

---

## Files Affected

| File | Action | Lines (after task) |
|------|--------|--------------------|
| `internal/infra/llm/types.go` | Create | ~55 |
| `internal/infra/llm/provider.go` | Create | ~25 |
| `internal/infra/llm/provider_test.go` | Create | ~15 |
| `internal/infra/llm/ollama.go` | Create | ~130 |
| `internal/infra/llm/ollama_test.go` | Create | ~180 |
| `internal/infra/llm/router.go` | Create | ~45 |
| `internal/infra/llm/router_test.go` | Create | ~60 |
| `internal/infra/config/config.go` | Create | ~50 |
| `docs/tasks/task_2.3.md` | Create | this file |

**No changes to `go.mod`** — Ollama adapter uses stdlib `net/http`.

---

## Decisions

1. `ChatCompletionStream` excluded from MVP — adds goroutine complexity not needed for Task 2.4 (embeddings are synchronous).
2. Tests do NOT require real Ollama — `httptest.NewServer` returns fixture JSON.
3. Default embed model: `nomic-embed-text` (768 dims, float32, sqlite-vec compatible).
4. Default chat model: `llama3.2:3b`.
5. `Router.Route()` in MVP is a pass-through — policy/no-cloud/budget logic deferred to Task 3.x.
6. `[]float32` vector type — matches sqlite-vec BLOB format.

---

## Tasks Completed

- [x] T1: Create docs/tasks/task_2.3.md
- [x] T2: Create internal/infra/llm/types.go
- [x] T3: Create provider.go + provider_test.go (TDD — interface defined first)
- [x] T4: Create ollama.go + ollama_test.go (httptest mock)
- [x] T5: Create router.go + router_test.go
- [x] T6: Create internal/infra/config/config.go
- [x] T7: Gates passed (make test + make complexity)
- [x] T8: Commit
