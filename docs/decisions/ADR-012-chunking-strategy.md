---
id: ADR-012
title: "Chunking strategy: 512-token chunks with 50-token overlap, whitespace tokenizer"
date: 2026-01-22
status: accepted
deciders: [matias]
tags: [adr, rag, knowledge, embeddings, chunking]
related_tasks: [task_2.2]
related_frs: [FR-091]
---

# ADR-012 — Chunking strategy: 512-token chunks with 50-token overlap, whitespace tokenizer

## Status

`accepted`

## Context

Knowledge items (emails, documents, call transcripts) must be split into chunks before
embedding. The chunk size directly affects:

1. **Embedding quality** — too large: semantic dilution; too small: lost context
2. **Vector search recall** — overlap between chunks prevents boundary artifacts
3. **LLM context usage** — chunks are the unit retrieved and injected into prompts
4. **Dependency complexity** — tokenization can require external libraries (tiktoken, HuggingFace)

## Decision

For the MVP, use a **simple whitespace tokenizer** with fixed parameters:

- **Chunk size**: 512 tokens
- **Overlap**: 50 tokens (≈10% of chunk size)
- **Tokenizer**: whitespace split (1 token ≈ 1 word)
- **Metadata per chunk**: `chunk_index`, `token_count`, `knowledge_item_id`

```go
type Chunk struct {
    Index      int
    Text       string
    TokenCount int
}

func chunkText(text string, size, overlap int) []Chunk {
    words := strings.Fields(text)
    // sliding window over words slice
}
```

These parameters are configurable via `internal/infra/config` for future per-workspace tuning.

## Rationale

- 512 tokens is the de-facto standard for RAG systems (OpenAI, LangChain defaults)
- 50-token overlap prevents losing context at chunk boundaries (e.g., a sentence split
  across two chunks)
- Whitespace tokenizer requires zero external dependencies — consistent with the
  model-agnostic, dependency-light design principle
- For most CRM content (short emails, deal notes, case descriptions) 512 tokens
  covers the full document in 1–3 chunks, making overlap less critical
- Config-driven parameters allow future tuning without code changes

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| tiktoken (OpenAI tokenizer) | External dependency; tied to OpenAI token counts |
| sentence-boundary chunking | Requires NLP library (spacy, nltk); adds build complexity |
| Fixed character length (e.g., 2000 chars) | Less predictable token counts across different content types |
| Semantic chunking (similarity-based) | Requires a running LLM for chunking — circular dependency |

## Consequences

**Positive:**
- Zero external dependencies for chunking
- Consistent chunk sizes across all content types
- Config-driven — tunable per workspace in P1

**Negative / tradeoffs:**
- Whitespace ≠ real tokenizer — actual token count may vary ±15% depending on
  punctuation and special characters
- For code snippets or structured data, whitespace chunking produces unnatural boundaries

## References

- `internal/domain/knowledge/chunker.go` — chunking implementation
- `internal/infra/config/` — chunk size and overlap configuration
- `docs/tasks/task_2.2.md` — ingestion pipeline design
