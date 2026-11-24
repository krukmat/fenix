---
doc_type: task
id: FENIX-AI-W6-2
title: "Version exchange contracts"
status: draft_pending_readiness
phase: W6
week: 0
tags: [ai-workflow, cloud-local]
fr_refs: []
uc_refs: []
blocked_by: [FENIX-AI-W6-1]
blocks: [FENIX-AI-W6-3, FENIX-AI-W6-4, FENIX-AI-W6-5, FENIX-AI-W6-6]
files_affected: ["scripts/workflow_exchange.py", "scripts/workflow_exchange_test.py"]
criticality: standard
criticality_basis: "Bounded planning, contract, measurement or reporting scope; provisional developer judgment."
created: 2026-11-24
completed: null
owner: "ChatGPT"
confidence_execution_pct: 95
---

# Task W6.2 — Version exchange contracts

**Plan:** [W6 execution plan](../plans/ai_workflow_w6_cloud_local_execution_plan.md)
**Task:** W6.2
**Task file:** `docs/tasks/task_fenix_ai_workflow_w6_2.md`
**Plan file:** `docs/plans/ai_workflow_w6_cloud_local_execution_plan.md`
**Summary:** Define strictly validated TaskPacket, ContextReceipt and ResultEnvelope; distinguish proposals from acceptance.
**Code affected:** scripts/workflow_exchange.py, scripts/workflow_exchange_test.py
**Criticality:** standard (provisional; RRI and reviewer concurrence pending)
**Effort/reasoning:** Medium — bounded scope
**Recommended model:** resolve from current workflow policy and measured RRI; no new model assignment.
**Estimated tokens:** ~3000 (planning estimate, not provider usage).
**RRI:** NOT RUN.
**Peer readiness review approval:** NOT RUN; no artifact.
**Peer code review approval:** NOT RUN; no artifact.

## Acceptance criteria

Reject unsupported versions, malformed input, wrong identities, missing required evidence; legacy adapter behavior unchanged.
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
validate and canonicalize versioned envelope; bind identities
return evidence without changing active approval or routing defaults
```

## Draft evidence

See [W6 exchange draft](../ai-workflow/cloud_local_interaction_draft.md). No implementation or validation has been executed.
