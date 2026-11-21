---
doc_type: task
id: FENIX-AI-W3-3
status: integration_pending
updated: 2026-11-21
owner: ChatGPT
confidence_execution_pct: 95
---

# W3.3 — Optimize review prompt and pass policy

**Task:** W3.3 — Optimize review prompt and pass policy
**Task file:** `docs/tasks/task_fenix_ai_workflow_w3_3.md`
**Plan file:** `docs/plans/ai_workflow_w1_w5_execution_plan.md`
**Responsible:** ChatGPT
**Execution block:** B2
**Scope:** Reuse existing local compaction; compare 1 vs 3 review passes
**Dependencies:** W3.1,W4.2
**Execution confidence:** 95% — Existing compaction available.
**RRI:** Not yet computed for any future source-changing implementation. Compute using `scripts/rri.py` on exact modified paths.
**Peer readiness approval:** Not run / no artifact.
**Peer code review approval:** Not run / no artifact.
**Repository QA:** Not run (previous standalone staging tests are not repository certification).

## Acceptance criteria

- Metrics and safe pilot rules defined; default remains 3 until evidence.
- Preserve existing RRI/HITL and peer-review gates; no `WAIVER` inferred.
- Evidence and any implementation are recorded in the repository; local model tests stay in the final local block.

## Status and evidence

Implementation candidate exists in staging only. Do not claim merged or verified.
