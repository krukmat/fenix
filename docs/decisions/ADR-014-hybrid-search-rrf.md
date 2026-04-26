---
id: ADR-014
title: "Hybrid search uses Reciprocal Rank Fusion (RRF, k=60) over BM25 + vector results"
date: 2026-01-26
status: accepted
deciders: [matias]
tags: [adr, rag, search, hybrid, rrf, bm25, vector]
related_tasks: [task_2.5]
related_frs: [FR-090, FR-092]
---

# ADR-014 — Hybrid search uses Reciprocal Rank Fusion (RRF, k=60) over BM25 + vector results

## Status

`accepted`

## Context

FenixCRM's retrieval layer must support both keyword queries (exact terms like deal names,
contact emails) and semantic queries (intent-based like "what were the main objections
in this deal?"). No single retrieval method handles both well:

- **BM25** (FTS5): excellent for exact keyword matching, poor for paraphrase/synonym recall
- **Vector similarity** (sqlite-vec ANN): excellent for semantic meaning, poor for rare
  exact terms not seen during embedding training

The challenge is combining the two ranked lists into a single, coherent ranking without
knowing the relative quality of each in advance.

## Decision

Use **Reciprocal Rank Fusion (RRF)** to merge BM25 and vector search results:

```
rrf_score(d) = Σ 1 / (k + rank_i(d))
```

Where:
- `k = 60` (standard RRF constant — reduces the impact of high-rank outliers)
- `rank_i(d)` = position of document `d` in result list `i` (1-indexed)
- Sum is over both result lists (BM25 ranks + vector ranks)

Documents appearing in only one list get a score contribution from that list only.
Documents appearing in both lists get a boosted combined score.

**Fallback handling:**
- If BM25 fails → use vector results only (no RRF)
- If vector search fails → use BM25 results only (no RRF)
- If both fail → return empty evidence pack + abstain signal

**Performance target:** < 500ms p95 with warm SQLite cache

## Rationale

- RRF is order-based, not score-based — no normalization of BM25 scores vs. cosine
  distances needed (they are not directly comparable)
- k=60 is the empirically validated default from the original RRF paper (Cormack et al.)
- Fallback to single-method search ensures retrieval never fully fails due to one
  subsystem being unavailable
- Equal weighting between BM25 and vector is the correct default — domain-specific
  reweighting can be added in P1 if evals show imbalance

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Score normalization + linear combination | BM25 and cosine scores are not comparable without calibration; prone to one dominating |
| Vector search only | Poor recall for exact keyword queries (deal IDs, contact names) |
| BM25 only | Poor recall for semantic queries and paraphrase variants |
| Learned reranker (cross-encoder) | Requires GPU inference; too slow for <500ms target; P1 candidate |

## Consequences

**Positive:**
- Handles both keyword and semantic queries without tuning
- Gracefully degrades when one subsystem fails
- No model inference required for ranking — pure rank arithmetic

**Negative / tradeoffs:**
- RRF does not account for confidence differences between retrievers — a low-confidence
  vector match gets the same treatment as a high-confidence BM25 match if they share the rank
- k=60 is a good default but may need tuning per domain in P1 (evals will surface this)

## References

- `internal/domain/retrieval/hybrid.go` — RRF implementation
- `docs/tasks/task_2.5.md` — hybrid search design
- Cormack, Clarke, Buettcher (2009): "Reciprocal Rank Fusion outperforms Condorcet and individual rank learning methods"
- sqlite-vec: https://github.com/asg017/sqlite-vec
