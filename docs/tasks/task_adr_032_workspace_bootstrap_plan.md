---
doc_type: task
id: ADR032-PLAN-001
title: "Analyze ADR-032 and produce the implementation plan"
status: done
phase: governance
week: "2026-W27"
tags: [adr, planning, auth, rbac, pipeline, onboarding]
fr_refs: [FR-060, FR-070, FR-071]
uc_refs: []
blocked_by: []
blocks: [ADR032-BOOTSTRAP-DESIGN-001]
files_affected:
  - .gitignore
  - docs/decisions/ADR-032-workspace-bootstrap-defaults.md
  - docs/tasks/.gitignore
  - docs/plans/adr-032-workspace-bootstrap-remediation-plan.md
  - docs/tasks/task_adr_032_workspace_bootstrap_plan.md
created: 2026-07-02
completed: 2026-07-02
---

# Task ADR032-PLAN-001

**Plan**: [ADR-032 workspace bootstrap remediation](../plans/adr-032-workspace-bootstrap-remediation-plan.md)

## Task Card

Task: ADR032-PLAN-001

Task file: docs/tasks/task_adr_032_workspace_bootstrap_plan.md

Plan file: docs/plans/adr-032-workspace-bootstrap-remediation-plan.md

Summary: Analyze ADR-032 against the current auth, RBAC, pipeline, and external-validation evidence, then produce the implementation plan that decomposes the work into design, code, and documentation follow-ups. The task also promotes the resulting plan/task artifacts out of ignore rules so the ADR workstream is shareable. The plan must preserve the ADR boundaries: atomic bootstrap for new workspaces only, no historical backfill, and no silent scope creep into unrelated agent-definition provisioning.

Code affected: No product code changes are planned in this task. Expected analysis and documentation areas are docs/decisions/, docs/plans/, docs/tasks/, .gitignore, internal/domain/auth/, internal/domain/crm/, and internal/domain/policy/.

Effort/reasoning: Low - planning only, with a bounded docs surface and confirmed code paths. RRI is expected to stay in the low band.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Estimated tokens: ~8000

Peer readiness review approval: reviewer=Claude Code; artifact=logs/peer-workflow-review/task-readiness_codex_by_claude_20260702T061857Z.json; status=PASS

## Acceptance Criteria

1. A dedicated ADR-032 plan file exists and is linked from this task.
2. The plan separates bootstrap-contract design, transactional implementation, regression coverage, and doc sync into discrete follow-up tasks.
3. The plan states explicit non-goals, including no historical backfill and no implicit agent-definition provisioning.
4. The resulting canonical plan/task artifacts are not silently blocked by ignore rules.
5. The final report distinguishes confirmed code-path facts from planning recommendations.

## Scope

- In: ADR analysis, code-path inspection, plan creation, task-ledger updates, and minimal ignore-rule promotion for the new canonical artifacts.
- Out: product code changes, migrations, commits, pushes, or executing the bootstrap fix itself.

## Risks

- The default role can easily become over-broad if the permission contract is not defined before coding.
- A bootstrap helper that is not truly transaction-bound would recreate the same defect in a subtler form.
- T3's separate `agent_definition` gap could be accidentally folded into this ADR unless the scope stays explicit.
