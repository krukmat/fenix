---
doc_type: task
id: FENIX-AI-W6-6
title: "End-to-end telemetry"
status: draft_pending_readiness
phase: W6
week: 0
tags: [ai-workflow, cloud-local]
fr_refs: []
uc_refs: []
blocked_by: [FENIX-AI-W6-2, FENIX-AI-W6-4]
blocks: [FENIX-AI-W6-7]
files_affected: ["scripts/workflow_audit_aggregate.py", "scripts/workflow_audit_aggregate_test.py"]
criticality: standard
criticality_basis: "Bounded planning, contract, measurement or reporting scope; provisional developer judgment."
created: 2026-11-24
completed: null
owner: "ChatGPT"
confidence_execution_pct: 93
---

# Task W6.6 — End-to-end telemetry

**Plan:** [W6 execution plan](../plans/ai_workflow_w6_cloud_local_execution_plan.md)
**Task:** W6.6
**Task file:** `docs/tasks/task_fenix_ai_workflow_w6_6.md`
**Plan file:** `docs/plans/ai_workflow_w6_cloud_local_execution_plan.md`
**Summary:** Correlate task/run/attempt, distinguish transport bytes, estimates and provider-measured tokens; retain errors.
**Code affected:** scripts/workflow_audit_aggregate.py, scripts/workflow_audit_aggregate_test.py
**Criticality:** standard (provisional; RRI and reviewer concurrence pending)
**Effort/reasoning:** Low — bounded telemetry changes; mandatory local attempt remains applicable if implementing
**Recommended model:** claude-sonnet-5-5. Non-default recommendation: currently active successor to the repository's pinned Sonnet tier; official model catalog checked 2026-11-24 (https://platform.claude.com/docs/en/models/overview). This is a task recommendation, not a change to reviewer configuration or routing defaults.
**Estimated tokens:** ~3000 (planning estimate, not provider usage).
**RRI:** 25 (Low); developer-scored local session, 2026-11-24; evidence: `logs/rri-w6/w6_6.json` (owner-reported; gitignored, not inspected in cloud). RRI criticality suggestion: false. Recompute if scope or scoring inputs change.
**Peer readiness review approval:** NOT RUN; blocked transitively by W6.2 (local session 2026-11-24).
**Peer code review approval:** NOT RUN; no artifact.

## Acceptance criteria

No double counts; unknown usage is null; failures and timeouts included; accepted task requires existing gates; no raw prompt export.
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
record terminal event; deduplicate and aggregate with unknown usage preserved
return evidence without changing active approval or routing defaults
```

## Draft evidence

See [W6 exchange draft](../ai-workflow/cloud_local_interaction_draft.md). No implementation or validation has been executed.
