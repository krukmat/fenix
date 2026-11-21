---
doc_type: task
id: FENIX-AI-W4-2
status: documented
updated: 2026-11-21
owner: ChatGPT
confidence_execution_pct: 98
---

# W4.2 — Define model benchmark KPIs

**Task:** W4.2 — Define model benchmark KPIs
**Task file:** `docs/tasks/task_fenix_ai_workflow_w4_2.md`
**Plan file:** `docs/plans/ai_workflow_w1_w5_execution_plan.md`
**Responsible:** ChatGPT
**Execution block:** B1
**Scope:** Specify task correctness, time, cost, failure, retry and escalation metrics
**Dependencies:** W4.1
**Execution confidence:** 98% — Metrics and corpus can be defined off-Mac.
**RRI:** Not yet computed for any future source-changing implementation. Compute using `scripts/rri.py` on exact modified paths.
**Peer readiness approval:** Not run / no artifact.
**Peer code review approval:** Not run / no artifact.
**Repository QA:** Not run (previous standalone staging tests are not repository certification).

## Acceptance criteria

- Baseline protocol includes token coverage and comparable workloads.
- Preserve existing RRI/HITL and peer-review gates; no `WAIVER` inferred.
- Evidence and any implementation are recorded in the repository; local model tests stay in the final local block.

## Status and evidence

Source-backed analysis exists in [reconciliation](../audits/ai_workflow_w1_w5_reconciliation.md) and prior staging. This is NOT a certified source integration.
