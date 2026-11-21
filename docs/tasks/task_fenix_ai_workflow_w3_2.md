---
doc_type: task
id: FENIX-AI-W3-2
status: documented_repo
updated: 2026-11-21
owner: ChatGPT
confidence_execution_pct: 90
---

# W3.2 — Specify structured Nimble advisory precheck

**Task:** W3.2 — Specify structured Nimble advisory precheck
**Task file:** `docs/tasks/task_fenix_ai_workflow_w3_2.md`
**Plan file:** `docs/plans/ai_workflow_w1_w5_execution_plan.md`
**Responsible:** ChatGPT
**Execution block:** B4
**Scope:** Define PASS/FINDINGS/ESCALATE precheck as non-authoritative
**Dependencies:** W3.1,W4.2
**Execution confidence:** 90% — Value requires measurement before default rollout.
**RRI:** Not yet computed for any future source-changing implementation. Compute using `scripts/rri.py` on exact modified paths.
**Peer readiness approval:** Not run / no artifact.
**Peer code review approval:** Not run / no artifact.
**Repository QA:** Not run (previous standalone staging tests are not repository certification).

## Acceptance criteria

- Malformed output escalates; precheck never replaces tests or cloud review.
- Preserve existing RRI/HITL and peer-review gates; no `WAIVER` inferred.
- Evidence and any implementation are recorded in the repository; local model tests stay in the final local block.

## Status and evidence

The corpus, benchmark protocol or review contract is now versioned in the repository. Local replay, QA and independent gate artifacts remain outstanding.
