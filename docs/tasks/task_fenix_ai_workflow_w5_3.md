---
doc_type: task
id: FENIX-AI-W5-3
status: documented_repo_with_instrumentation
updated: 2026-11-21
owner: ChatGPT
confidence_execution_pct: 95
---

# W5.3 — Audit prompts, context and local memory policy

**Task:** W5.3 — Audit prompts, context and local memory policy
**Task file:** `docs/tasks/task_fenix_ai_workflow_w5_3.md`
**Plan file:** `docs/plans/ai_workflow_w1_w5_execution_plan.md`
**Responsible:** ChatGPT
**Execution block:** B1
**Scope:** Assess prompt budgets, keep-alive, token telemetry and model lifecycle
**Dependencies:** W5.1
**Execution confidence:** 95% — Code and policy inspectable.
**RRI:** Not yet computed for any future source-changing implementation. Compute using `scripts/rri.py` on exact modified paths.
**Peer readiness approval:** Not run / no artifact.
**Peer code review approval:** Not run / no artifact.
**Repository QA:** Not run (previous standalone staging tests are not repository certification).

## Acceptance criteria

- Baseline instrumentation opportunities documented; no invented measurements.
- Preserve existing RRI/HITL and peer-review gates; no `WAIVER` inferred.
- Evidence and any implementation are recorded in the repository; local model tests stay in the final local block.

## Status and evidence

Source-backed audit and a read-only aggregation tool are now in main. No local observations or repository QA artifacts exist yet.
