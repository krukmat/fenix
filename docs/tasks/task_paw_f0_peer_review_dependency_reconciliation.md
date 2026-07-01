---
doc_type: task
id: PAW-F0
title: "Reconcile PAW dependency drift before Phase F peer-review gates"
status: done
phase: F
week: ""
tags: [paw, devex, workflow, peer-review, docs, governance]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: [PAW-F1]
files_affected:
  - docs/plans/portable_agent_workflow_port_plan.md
  - docs/tasks/task_paw_f0_peer_review_dependency_reconciliation.md
  - docs/tasks/.gitignore
  - .gitignore
created: 2026-07-01
completed: "2026-07-01"
rri: 18
rri_band: Low
hp: "PAW plan records Phase F dependency order and the task ledger can track PAW-F0 as a canonical coordination task"
ec: "Ignored canonical plan/task artifacts are detected and given narrow Git-trackable exceptions instead of broad ignore-rule changes"
coverage_cert: ""
---

# Task PAW-F0

**Plan**: [Portable Agent Workflow Port Plan](../plans/portable_agent_workflow_port_plan.md#8-task-decomposition)

## Task Card

Task: PAW-F0
Task file: docs/tasks/task_paw_f0_peer_review_dependency_reconciliation.md
Plan file: docs/plans/portable_agent_workflow_port_plan.md
Summary: Reconcile the Portable Agent Workflow plan so Phase F is documented with provider-aware peer-review dependencies before implementation tasks are created. Keep this task documentation-only and do not implement the peer-review script.
Code affected: No product code. Expected files are docs/plans/portable_agent_workflow_port_plan.md, docs/tasks/task_paw_f0_peer_review_dependency_reconciliation.md, docs/tasks/.gitignore, and .gitignore.
Effort/reasoning: Low - Documentation and dependency alignment only.
Recommended model: claude-sonnet-4-6
Estimated tokens: ~2500

## Summary

Document Phase F in the Portable Agent Workflow plan and make the new canonical plan/task artifacts trackable. Phase F adds provider-aware peer review gates: Claude Code callers are reviewed by Codex, Codex callers are reviewed by Claude, and any other local or remote provider defaults to Claude review.

## Acceptance Criteria

1. `docs/plans/portable_agent_workflow_port_plan.md` includes Phase F scope, task decomposition, dependency ordering, risks, and verification strategy.
2. `PAW-F0` exists as a task record with required `doc_type: task` frontmatter.
3. The PAW plan is made Git-trackable with a narrow `.gitignore` exception because it is canonical.
4. The `PAW-F0` task record is made Git-trackable with a narrow `docs/tasks/.gitignore` exception.
5. No peer-review script, Makefile target, hook, or product code is implemented in this task.

## Scope

- **In**: Documentation of Phase F and dependency order; ignore-rule exceptions required for canonical tracking.
- **Out**: No `scripts/peer-workflow-review.py`, no `make qa-peer-workflow-review`, no hook enforcement, no product code.

## Risks

- Existing Phase E task ledgers and the PAW plan may disagree on status and blockers. This task records Phase F without silently rewriting completed task history.
- Broad ignore-rule changes could accidentally promote operational task records. Use narrow exceptions only.
