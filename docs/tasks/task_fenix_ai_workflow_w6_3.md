---
doc_type: task
id: FENIX-AI-W6-3
title: "Incremental context and receipts"
status: draft_pending_readiness
phase: W6
week: 0
tags: [ai-workflow, cloud-local]
fr_refs: []
uc_refs: []
blocked_by: [FENIX-AI-W6-2]
blocks: [FENIX-AI-W6-4, FENIX-AI-W6-7]
files_affected: ["scripts/workflow_exchange.py", "scripts/workflow_exchange_test.py"]
criticality: critical
criticality_basis: "May affect evidence integrity or governance boundary; provisional developer judgment."
created: 2026-11-24
completed: null
owner: "ChatGPT"
confidence_execution_pct: 92
---

# Task W6.3 — Incremental context and receipts

**Plan:** [W6 execution plan](../plans/ai_workflow_w6_cloud_local_execution_plan.md)
**Task:** W6.3
**Task file:** `docs/tasks/task_fenix_ai_workflow_w6_3.md`
**Plan file:** `docs/plans/ai_workflow_w6_cloud_local_execution_plan.md`
**Summary:** Resolve immutable content references and compute canonical receipt digests; use full capsules if receivers cannot resolve the baseline.
**Code affected:** scripts/workflow_exchange.py, scripts/workflow_exchange_test.py
**Criticality:** critical (provisional; RRI and reviewer concurrence pending)
**Effort/reasoning:** High — governance and evidence compatibility
**Recommended model:** resolve from current workflow policy and measured RRI; no new model assignment.
**Estimated tokens:** ~3000 (planning estimate, not provider usage).
**RRI:** NOT RUN.
**Peer readiness review approval:** NOT RUN; no artifact.
**Peer code review approval:** NOT RUN; no artifact.

## Acceptance criteria

Reject stale/dirty mismatches, path traversal, symlinks, digest mismatch and missing required context; no truncation of acceptance criteria or protected evidence.
Preserve current RRI/HITL and independent peer review requirements. Do not declare DONE until the applicable checks are complete.

## System context

```text
Approved task + RRI -> existing supervisor -> opt-in exchange adapter
                                      |                |
                                      |                v
                                      |        existing CLI / Ollama
                                      |                |
                                      v                v
                           existing QA and peer gates <- receipt/result
                                      |
                                      v
                              evidence and telemetry
```

No new service or process lifecycle; no direct database writes or new privileges. Errors return a blocked/invalid result; existing supervisor owns acceptance.

## High-level pseudocode

```text
load approved task and pinned source
validate applicable authority, context and scope
resolve allowed immutable content; verify exact digests; else full capsule or block
return evidence without changing active approval or routing defaults
```

## Draft evidence

See [W6 exchange draft](../ai-workflow/cloud_local_interaction_draft.md). No implementation or validation has been executed.
