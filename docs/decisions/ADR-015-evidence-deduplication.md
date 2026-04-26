---
id: ADR-015
title: "Evidence pack deduplication using cosine similarity threshold (>0.95) and confidence tiers"
date: 2026-01-28
status: accepted
deciders: [matias]
tags: [adr, rag, evidence, deduplication, confidence]
related_tasks: [task_2.6]
related_frs: [FR-092]
---

# ADR-015 — Evidence pack deduplication using cosine similarity threshold (>0.95) and confidence tiers

## Status

`accepted`

## Context

The hybrid search (BM25 + vector, RRF-merged) can return near-duplicate chunks — for
example, two slightly different versions of the same email paragraph, or the same
knowledge item chunked with overlap. Injecting all of them into the LLM prompt:

1. Wastes context tokens (cost)
2. Gives the LLM the impression a single piece of evidence appears from multiple sources
   (false confidence boost)
3. May cause the LLM to weight that evidence more heavily than warranted

Additionally, the copilot and agent layer needs a structured confidence signal per
evidence pack to decide whether to respond or abstain.

## Decision

**Deduplication:**

Remove near-duplicates from the candidate set using cosine similarity between chunk vectors:

```
If cosine_similarity(chunk_A.vector, chunk_B.vector) > 0.95:
    Keep only the chunk with the higher RRF score
    Discard the other
```

Threshold of 0.95 was chosen because:
- Overlap chunks (50-token overlap) typically score 0.85–0.92 — below the threshold
- True near-duplicates (same text, minor edits) score 0.97–1.00 — above the threshold

**Confidence tiers:**

After deduplication, assign confidence based on the top-ranked evidence score:

| Tier | Condition | Behavior |
|------|-----------|----------|
| `high` | top_score > 0.80 | Respond with evidence |
| `medium` | 0.50 < top_score ≤ 0.80 | Respond with caveat |
| `low` | top_score ≤ 0.50 | Abstain + escalate to human |

**Warnings added to evidence pack:**
- `freshness_warning`: if top evidence is older than 30 days (staleness signal)
- `filtered_count`: number of candidates removed by permission/sensitivity filters
- `dedup_count`: number of near-duplicates removed

## Rationale

- Cosine similarity deduplication is O(n²) on the candidate set, but n is bounded at
  retrieval time (top-20 candidates) — negligible latency impact
- Confidence tiers map directly to the abstention policy (FR-210): low confidence → abstain
- Warnings give the LLM and the UI transparent signals about evidence quality
- 0.95 threshold preserves intentional near-duplicate sources while removing true duplicates

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Dedup by exact text match | Misses near-duplicates with minor edits (common in email threads) |
| No deduplication | Token waste; false confidence amplification |
| Threshold of 0.99 | Too strict — overlapping chunks would not be deduplicated |
| Threshold of 0.90 | Too aggressive — would remove distinct chunks that happen to be topically similar |

## Consequences

**Positive:**
- Evidence packs are compact and non-redundant
- Confidence tiers drive abstention policy automatically
- Warnings are surfaced in the UI (copilot panel) for user awareness

**Negative / tradeoffs:**
- Deduplication requires loading all candidate vectors into memory for pairwise comparison
  (bounded by retrieval top-K, acceptable for MVP)
- The 0.95 threshold may need calibration based on real-world content — evals will surface this

## References

- `internal/domain/evidence/pack.go` — evidence pack assembly and deduplication
- `docs/tasks/task_2.6.md` — evidence pack design
- `docs/architecture.md` — Evidence Pack schema (sources, confidence, abstain_reason)
