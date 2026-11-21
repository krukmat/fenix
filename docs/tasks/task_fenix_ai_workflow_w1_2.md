---
doc_type: task
id: FENIX-AI-W1-2
status: documented
updated: 2026-11-21
owner: ChatGPT
confidence_execution_pct: 98
---

# W1.2 — Define model role matrix

**Task:** W1.2 — Define model role matrix
**Task file:** `docs/tasks/task_fenix_ai_workflow_w1_2.md`
**Plan file:** `docs/plans/ai_workflow_w1_w5_execution_plan.md`
**Responsible:** ChatGPT
**Execution block:** B3
**Scope:** Specify bounded roles for Qwen, GPT-OSS, Gemma and optional Nimble
**Dependencies:** W1.1
**Execution confidence:** 98% — Model suitability still requires Mac verification.
**RRI:** Not yet computed for any future source-changing implementation. Compute using `scripts/rri.py` on exact modified paths.
**Peer readiness approval:** Not run / no artifact.
**Peer code review approval:** Not run / no artifact.
**Repository QA:** Not run (previous standalone staging tests are not repository certification).

## Acceptance criteria

- Roles, constraints and exact-tag discovery documented; no unmeasured winner claimed.
- Preserve existing RRI/HITL and peer-review gates; no `WAIVER` inferred.
- Evidence and any implementation are recorded in the repository; local model tests stay in the final local block.

## Status and evidence

Source-backed analysis exists in [reconciliation](../audits/ai_workflow_w1_w5_reconciliation.md) and prior staging. This is NOT a certified source integration.
