---
doc_type: task
id: FENIX-AI-W4-3
status: local_last
updated: 2026-11-21
owner: Local
confidence_execution_pct: 85
---

# W4.3 — Run local model benchmarks

**Task:** W4.3 — Run local model benchmarks
**Task file:** `docs/tasks/task_fenix_ai_workflow_w4_3.md`
**Plan file:** `docs/plans/ai_workflow_w1_w5_execution_plan.md`
**Responsible:** Local
**Execution block:** B5
**Scope:** Execute frozen comparison on Mac with actual installed model IDs
**Dependencies:** W1.4,W4.2
**Execution confidence:** 85% — Local model inventory and resources unverified.
**RRI:** Not yet computed for any future source-changing implementation. Compute using `scripts/rri.py` on exact modified paths.
**Peer readiness approval:** Not run / no artifact.
**Peer code review approval:** Not run / no artifact.
**Repository QA:** Not run (previous standalone staging tests are not repository certification).

## Acceptance criteria

- Raw timing/token/test evidence and model inventory captured; failures preserved.
- Preserve existing RRI/HITL and peer-review gates; no `WAIVER` inferred.
- Evidence and any implementation are recorded in the repository; local model tests stay in the final local block.

## Status and evidence

Deferred by agreement: consolidate with the other final Mac-only task.
