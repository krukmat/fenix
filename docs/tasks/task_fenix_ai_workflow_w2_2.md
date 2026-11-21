---
doc_type: task
id: FENIX-AI-W2-2
status: documented
updated: 2026-11-21
owner: ChatGPT
confidence_execution_pct: 95
---

# W2.2 — Define risk and capability eligibility

**Task:** W2.2 — Define risk and capability eligibility
**Task file:** `docs/tasks/task_fenix_ai_workflow_w2_2.md`
**Plan file:** `docs/plans/ai_workflow_w1_w5_execution_plan.md`
**Responsible:** ChatGPT
**Execution block:** B4
**Scope:** Combine RRI decision with role competence, path sensitivity and test evidence
**Dependencies:** W2.1,W1.3
**Execution confidence:** 95% — Risk scores need per-task computation.
**RRI:** Not yet computed for any future source-changing implementation. Compute using `scripts/rri.py` on exact modified paths.
**Peer readiness approval:** Not run / no artifact.
**Peer code review approval:** Not run / no artifact.
**Repository QA:** Not run (previous standalone staging tests are not repository certification).

## Acceptance criteria

- Decision table rejects critical/sensitive tasks and preserves approval.
- Preserve existing RRI/HITL and peer-review gates; no `WAIVER` inferred.
- Evidence and any implementation are recorded in the repository; local model tests stay in the final local block.

## Status and evidence

Source-backed analysis exists in [reconciliation](../audits/ai_workflow_w1_w5_reconciliation.md) and prior staging. This is NOT a certified source integration.
