---
doc_type: task
id: FENIX-AI-W6-4
title: "Provider-specific adapters"
status: draft_pending_readiness
phase: W6
week: 0
tags: [ai-workflow, cloud-local]
fr_refs: []
uc_refs: []
blocked_by: [FENIX-AI-W6-2, FENIX-AI-W6-3]
blocks: [FENIX-AI-W6-6, FENIX-AI-W6-7]
files_affected: ["scripts/peer-workflow-review.py", "scripts/delegate-low-rri.py", "scripts/peer_workflow_review_test.py", "scripts/delegate_low_rri_test.py"]
criticality: critical
criticality_basis: "May affect evidence integrity or governance boundary; provisional developer judgment."
created: 2026-11-24
completed: null
owner: "ChatGPT"
confidence_execution_pct: 90
---

# Task W6.4 — Provider-specific adapters

**Plan:** [W6 execution plan](../plans/ai_workflow_w6_cloud_local_execution_plan.md)
**Task:** W6.4
**Task file:** `docs/tasks/task_fenix_ai_workflow_w6_4.md`
**Plan file:** `docs/plans/ai_workflow_w6_cloud_local_execution_plan.md`
**Summary:** Add opt-in wrappers for existing CLI and Ollama; remote handoff uses approved redacted artifacts; bind actual input coverage to receipts.
**Code affected:** scripts/peer-workflow-review.py, scripts/delegate-low-rri.py, scripts/peer_workflow_review_test.py, scripts/delegate_low_rri_test.py
**Criticality:** critical (provisional; RRI and reviewer concurrence pending)
**Effort/reasoning:** High — cross-provider adapter compatibility and governance-sensitive integration
**Recommended model:** claude-opus-5-5. Non-default recommendation: currently active successor to the repository's pinned Opus tier; official model catalog checked 2026-11-24 (https://platform.claude.com/docs/en/models/overview). This is a task recommendation, not a change to reviewer configuration or routing defaults.
**Estimated tokens:** ~3000 (planning estimate, not provider usage).
**RRI:** 45 (Med-high); developer-scored local session, 2026-11-24; evidence: `logs/rri-w6/w6_4.json` (owner-reported; gitignored, not inspected in cloud). RRI criticality suggestion: false. Recompute if scope or scoring inputs change.
**Peer readiness review approval:** NOT RUN; blocked transitively by W6.2 (local session 2026-11-24).
**Peer code review approval:** NOT RUN; no artifact.

## Acceptance criteria

Codex native --base review cannot claim receipt coverage it did not receive; preserve independent gate and fallback rules, no new services or network exposure.
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
dispatch via existing adapter; record actual delivery coverage and outcome
return evidence without changing active approval or routing defaults
```

## Draft evidence

See [W6 exchange draft](../ai-workflow/cloud_local_interaction_draft.md). No implementation or validation has been executed.
