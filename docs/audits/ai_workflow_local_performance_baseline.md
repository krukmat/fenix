---
doc_type: audit
title: "AI Development Workflow Performance — baseline and measurement contract"
scope: local-development-workflow
status: baseline_instrumented_not_measured
updated: 2026-11-21
---

# W5 — Local AI workflow performance baseline

This report measures the **development workflow**, NOT Fénix product APIs or application runtime.

## Prior evidence inspected

- `docs/plans/local_peer_review_prompt_compaction_plan.md`: PAW-F17 compacted local fallback/advisory packets; compaction exists and must be benchmarked, not reimplemented.
- `scripts/gemma-code-review.py`: `--passes` defaults to 3. Reducing passes is a hypothesis until finding recall has been measured.
- `scripts/peer-workflow-review.py`: task-readiness and post-code primary independent reviews, plus critical advisory. A zero-output Codex review was incorrectly mapped to PASS; a fail-closed patch is committed but not independently certified.
- `scripts/gemma_local.py`: Ollama usage counters and `logs/gemma-audit/*.jsonl` provide model-level token and timing evidence when present.
- `scripts/gemma-audit-report.py`: existing report tool; reuse its metrics rather than duplicating benchmark assumptions.

## New reporting surface

Run from the repository root, read-only:

```sh
python3 scripts/workflow_audit_aggregate.py --gemma-log-dir logs/gemma-audit --peer-log-dir logs/peer-workflow-review --output out/workflow-audit.json
```

The output includes record counts, measured token coverage, local durations, stages and attempts. If logs are absent, it reports `no_records`; no fabricated p95 or token totals. It never exports raw prompts or reviewers' findings.

## Performance hypotheses, NOT conclusions

1. Three local review passes may add avoidable cost to simple diffs, but may catch distinct defects. Compare recall and cost before modifying defaults.
2. Cold model load/eviction under limited Mac RAM may dominate response latency. Measure prefill, generation, cold/warm calls and memory pressure.
3. Broad task packets can increase model input tokens; PAW-F17 already mitigates local fallback and critical advisory packet size. Compare preserved acceptance coverage.
4. Repeated peer fallback or unavailable review CLIs can cause delays. Count retries and primary/fallback attempts using artifacts.
5. Role-specific routing may help task completion, but extra classifiers may cost more than they save. Nimble stays optional and advisory.

## Measurement protocol (W4.3 + W5.4, Mac-only last block)

Freeze task inputs and revision; capture actual `ollama list`, model tag, prompt/eval tokens, wall time, Ollama cold/warm load time, p50/p95 across repetitions, RAM pressure, first-token time when available, timeouts, fallback count, deterministic test pass rate and cloud tokens per accepted task. Use anonymized/fake task packets and do not upload model payloads.

## Decision rules

- Optimize **end-to-end accepted task cost and latency** subject to no regression in quality/safety, not tokens/s alone.
- Report measured, unmeasured and unmeasurable metrics separately.
- No model becomes stable by default and no review gate is weakened without reproducible benchmark evidence.
- The present report is a baseline. W5.5 remains conditional on W5.4.

## Token attribution patch

The developer and local reviewer audit writers now include the selected model and Ollama-measured input tokens when returned by the stream. The aggregator reports `prompt_tokens` (measured) and `packet_tokens_est` (estimated) separately, with coverage ratios. Existing history without fields remains missing; no zero-fill or fabricated savings.
