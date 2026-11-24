---
doc_type: task
id: FENIX-AI-W6-5
title: "Advisory delegation suitability"
status: draft_pending_readiness
phase: W6
week: 0
tags: [ai-workflow, cloud-local]
fr_refs: []
uc_refs: []
blocked_by: [FENIX-AI-W6-2]
blocks: [FENIX-AI-W6-7]
files_affected: ["scripts/workflow_model_router.py", "scripts/workflow_model_router_test.py"]
criticality: critical
criticality_basis: "May affect evidence integrity or governance boundary; provisional developer judgment."
created: 2026-11-24
completed: null
owner: "ChatGPT"
confidence_execution_pct: 90
---

# Task W6.5 — Advisory delegation suitability

**Plan:** [W6 execution plan](../plans/ai_workflow_w6_cloud_local_execution_plan.md)
**Task:** W6.5
**Task file:** `docs/tasks/task_fenix_ai_workflow_w6_5.md`
**Plan file:** `docs/plans/ai_workflow_w6_cloud_local_execution_plan.md`
**Summary:** Calculate an advisory suitability estimate only from measured evidence; default unknown when insufficient data.
**Code affected:** scripts/workflow_model_router.py, scripts/workflow_model_router_test.py
**Criticality:** critical (provisional; RRI and reviewer concurrence pending)
**Effort/reasoning:** Medium — bounded implementation with explicit negative tests and peer gates
**Recommended model:** claude-sonnet-5-5. Non-default recommendation: currently active successor to the repository's pinned Sonnet tier; official model catalog checked 2026-11-24 (https://platform.claude.com/docs/en/models/overview). This is a task recommendation, not a change to reviewer configuration or routing defaults.
**Estimated tokens:** ~3000 (planning estimate, not provider usage).
**RRI:** 33 (Moderate); developer-scored local session, 2026-11-24; evidence: `logs/rri-w6/w6_5.json` (owner-reported; gitignored, not inspected in cloud). RRI criticality suggestion: false. Recompute if scope or scoring inputs change.
**Peer readiness review approval:** NOT RUN; blocked transitively by W6.2 (local session 2026-11-24).
**Peer code review approval:** NOT RUN; no artifact.

## Acceptance criteria

RRI 26+ remains forbidden for local delegation; mandatory low-band attempt remains; no policy change; no autonomous acceptance.
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
derive advisory score from matched observations; preserve policy decision
return evidence without changing active approval or routing defaults
```

## Draft evidence

See [W6 exchange draft](../ai-workflow/cloud_local_interaction_draft.md). No implementation or validation has been executed.
