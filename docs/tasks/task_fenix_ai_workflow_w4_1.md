---
doc_type: task
id: FENIX-AI-W4-1
status: documented_repo
updated: 2026-11-21
owner: ChatGPT
confidence_execution_pct: 98
---

# W4.1 — Prepare historical Fenix task corpus

**Task:** W4.1 — Prepare historical Fenix task corpus
**Task file:** `docs/tasks/task_fenix_ai_workflow_w4_1.md`
**Plan file:** `docs/plans/ai_workflow_w1_w5_execution_plan.md`
**Responsible:** ChatGPT
**Execution block:** B1
**Scope:** Select 20 source-backed task scenarios and freeze comparison inputs
**Dependencies:** None
**Execution confidence:** 98% — Historical paths verified; replay needs Mac.
**RRI:** Not yet computed for any future source-changing implementation. Compute using `scripts/rri.py` on exact modified paths.
**Peer readiness approval:** Not run / no artifact.
**Peer code review approval:** Not run / no artifact.
**Repository QA:** Not run (previous standalone staging tests are not repository certification).

## Acceptance criteria

- All source task paths exist; frozen prompts and expected outcomes ready for replay.
- Preserve existing RRI/HITL and peer-review gates; no `WAIVER` inferred.
- Evidence and any implementation are recorded in the repository; local model tests stay in the final local block.

## Status and evidence

The corpus, benchmark protocol or review contract is now versioned in the repository. Local replay, QA and independent gate artifacts remain outstanding.
