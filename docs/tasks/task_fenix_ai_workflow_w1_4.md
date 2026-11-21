---
doc_type: task
id: FENIX-AI-W1-4
status: code_committed_peer_qa_pending
updated: 2026-11-21
owner: ChatGPT
confidence_execution_pct: 90
---

# W1.4 — Integrate configurable model routing

**Task:** W1.4 — Integrate configurable model routing
**Task file:** `docs/tasks/task_fenix_ai_workflow_w1_4.md`
**Plan file:** `docs/plans/ai_workflow_w1_w5_execution_plan.md`
**Responsible:** ChatGPT
**Execution block:** B3
**Scope:** Wire a deterministic role resolver to eligible low-RRI delegation
**Dependencies:** W1.3,W3.4
**Execution confidence:** 90% — Cannot certify existing repo gates here.
**RRI:** Not yet computed for any future source-changing implementation. Compute using `scripts/rri.py` on exact modified paths.
**Peer readiness approval:** Not run / no artifact.
**Peer code review approval:** Not run / no artifact.
**Repository QA:** Not run (previous standalone staging tests are not repository certification).

## Acceptance criteria

- Tests cover unsafe scope, absent model, existing fallback and no implicit moderate delegation.
- Preserve existing RRI/HITL and peer-review gates; no `WAIVER` inferred.
- Evidence and any implementation are recorded in the repository; local model tests stay in the final local block.

## Status and evidence

Opt-in routing adapter and unit tests were committed to main, with the active default preserved. No runtime model check, full QA or peer gate has run.
