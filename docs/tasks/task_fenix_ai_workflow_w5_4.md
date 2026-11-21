---
doc_type: task
id: FENIX-AI-W5-4
status: local_last
updated: 2026-11-21
owner: Local
confidence_execution_pct: 90
---

# W5.4 — Measure local workflow performance

**Task:** W5.4 — Measure local workflow performance
**Task file:** `docs/tasks/task_fenix_ai_workflow_w5_4.md`
**Plan file:** `docs/plans/ai_workflow_w1_w5_execution_plan.md`
**Responsible:** Local
**Execution block:** B5
**Scope:** Profile Mac RAM, cold/warm starts, throughput, latency and retries
**Dependencies:** W5.3,W4.2
**Execution confidence:** 90% — Requires Mac and representative load.
**RRI:** Not yet computed for any future source-changing implementation. Compute using `scripts/rri.py` on exact modified paths.
**Peer readiness approval:** Not run / no artifact.
**Peer code review approval:** Not run / no artifact.
**Repository QA:** Not run (previous standalone staging tests are not repository certification).

## Acceptance criteria

- Redacted reproducible measurements for comparable tasks and review paths.
- Preserve existing RRI/HITL and peer-review gates; no `WAIVER` inferred.
- Evidence and any implementation are recorded in the repository; local model tests stay in the final local block.

## Status and evidence

Deferred by agreement: consolidate with the other final Mac-only task.
