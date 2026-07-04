---
doc_type: task
id: CRIT-005
title: "Workflow and playbook integration for critical-task reviewer mix"
status: done
phase: implementation
week: 2026-W27
tags: [criticality, workflow, peer-review, governance, docs]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - docs/playbooks/AGENT_WORKFLOW_GUIDE.md
  - AGENTS.md
criticality: standard
criticality_basis: "Docs-only workflow integration; no runtime reviewer, auth, or migration change."
created: 2026-07-04
completed: 2026-07-04
---

# Task CRIT-005: Workflow and playbook integration for critical-task reviewer mix

**Plan**: [Critical-Task Classification and Local/Cross-Agent Reviewer Mix](../plans/critical_task_reviewer_mix_plan.md#rollout)

## Scope

Integrate the completed critical-task rollout into the portable workflow and
repo wrapper docs. This task closes the documentation loop after CRIT-001
through CRIT-004 are implemented.

Changes in scope:

- Update `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` so the workflow sequence
  reflects task criticality labeling, readiness-review concurrence, and the
  critical-only advisory local reviewer path at post-code-review time.
- Align `AGENTS.md` closure/reporting language with the final workflow, where
  needed, now that the rollout mechanics exist end to end.

## Out of scope

- Any further runtime changes to `scripts/peer-workflow-review.py`.
- Any change to RRI scoring or HITL approval thresholds.
- Any CI or hook wiring beyond the documentation already governed elsewhere.

## Task Card

Task: CRIT-005
Task file: docs/tasks/task_crit_005_workflow_playbook_integration.md
Plan file: docs/plans/critical_task_reviewer_mix_plan.md
Summary: Update the workflow playbook and repo wrapper docs so they describe the final critical-task classification flow, including labeling, peer concurrence, and the critical-only advisory local reviewer path. Keep the task documentation-only and do not change runtime behavior.
Code affected: docs/playbooks/AGENT_WORKFLOW_GUIDE.md, AGENTS.md
Criticality: standard
Criticality basis: Docs-only workflow integration; no runtime reviewer, auth, or migration change.
Effort/reasoning: Medium - the edits are small, but they must accurately reflect the now-implemented runtime and reporting contract without introducing policy drift.
Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6
Estimated tokens: ~2200
Peer readiness review approval: reviewer=local-gemma; artifact=logs/peer-workflow-review/task-readiness_codex_by_local-gemma_20260704T173631Z.json; status=PASS

## Acceptance criteria

- `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` documents the task criticality
  labeling step, readiness-review concurrence requirement, and critical-only
  advisory local reviewer path at post-code-review.
- `AGENTS.md` is aligned with the final closure/reporting expectations for the
  critical-task workflow where still needed after CRIT-002.
- No runtime code changes are made in this task.

## RRI

| Variable | Score | Evidence | Confidence |
|---|---|---|---|
| C cyclomatic | 0 | docs-only | High |
| F files | 1 | 2 docs | High |
| D domain | 0 | governance docs only | High |
| T coverage | 0 | no code path changes | High |
| A ambiguity | 1 | wording must match implemented behavior exactly | Medium |
| K coupling | 1 | integrates the prior rollout tasks into shared guidance | Medium |
| P impact | 2 | changes operator understanding of workflow/reporting | Medium |
| X context | 2 | plan + playbook + AGENTS | Medium |

Final RRI: 15 -> Low band (0-25) -> executable without full approval packet,
but still requires task-card presentation per repo reporting rules.
