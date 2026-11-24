---
doc_type: task
id: FENIX-AI-W6-7
title: "Deterministic compatibility checks"
status: draft_pending_readiness
phase: W6
week: 0
tags: [ai-workflow, cloud-local]
fr_refs: []
uc_refs: []
blocked_by: [FENIX-AI-W6-3, FENIX-AI-W6-4, FENIX-AI-W6-5, FENIX-AI-W6-6]
blocks: [FENIX-AI-W6-8]
files_affected: ["scripts/workflow_exchange_test.py", "scripts/peer_workflow_review_test.py", "scripts/delegate_low_rri_test.py", "scripts/workflow_audit_aggregate_test.py"]
criticality: critical
criticality_basis: "May affect evidence integrity or governance boundary; provisional developer judgment."
created: 2026-11-24
completed: null
owner: "ChatGPT"
confidence_execution_pct: 92
---

# Task W6.7 — Deterministic compatibility checks

**Plan:** [W6 execution plan](../plans/ai_workflow_w6_cloud_local_execution_plan.md)
**Task:** W6.7
**Task file:** `docs/tasks/task_fenix_ai_workflow_w6_7.md`
**Plan file:** `docs/plans/ai_workflow_w6_cloud_local_execution_plan.md`
**Summary:** Validate legacy/new shadow paths and negative tests; collect actual QA and peer results before activation.
**Code affected:** scripts/workflow_exchange_test.py, scripts/peer_workflow_review_test.py, scripts/delegate_low_rri_test.py, scripts/workflow_audit_aggregate_test.py
**Criticality:** critical (provisional; RRI and reviewer concurrence pending)
**Effort/reasoning:** High — governance and evidence compatibility
**Recommended model:** resolve from current workflow policy and measured RRI; no new model assignment.
**Estimated tokens:** ~3000 (planning estimate, not provider usage).
**RRI:** NOT RUN.
**Peer readiness review approval:** NOT RUN; no artifact.
**Peer code review approval:** NOT RUN; no artifact.

## Acceptance criteria

Frozen fixtures; no model inference required for unit tests; no fake gate artifacts; fail closed on missing evidence.
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
run legacy and shadow fixtures; compare outcomes; report real gate artifacts
return evidence without changing active approval or routing defaults
```

## Draft evidence

See [W6 exchange draft](../ai-workflow/cloud_local_interaction_draft.md). No implementation or validation has been executed.
