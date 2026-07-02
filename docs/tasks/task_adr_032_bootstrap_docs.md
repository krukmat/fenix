---
doc_type: task
id: ADR032-BOOTSTRAP-DOCS-001
title: "Sync ADR-032 external-validation documentation after bootstrap implementation"
status: completed
phase: remediation
week: "2026-W27"
tags: [adr, docs, external-validation, onboarding, bootstrap]
fr_refs: [FR-060, FR-070, FR-071]
uc_refs: []
blocked_by: [ADR032-BOOTSTRAP-IMPL-001]
blocks: []
files_affected:
  - docs/plans/external_validation_first_test_battery_plan.md
  - docs/tasks/task_extval_battery_t1_smoke_auth.md
  - docs/tasks/task_extval_battery_t3_support_agent.md
  - docs/plans/adr-032-workspace-bootstrap-remediation-plan.md
  - docs/tasks/task_adr_032_bootstrap_docs.md
created: 2026-07-02
completed: 2026-07-02
---

# Task ADR032-BOOTSTRAP-DOCS-001

**Plan**: [ADR-032 workspace bootstrap remediation](../plans/adr-032-workspace-bootstrap-remediation-plan.md)

## Task Card

Task: ADR032-BOOTSTRAP-DOCS-001

Task file: docs/tasks/task_adr_032_bootstrap_docs.md

Plan file: docs/plans/adr-032-workspace-bootstrap-remediation-plan.md

Summary: Update external-validation and governance-facing documentation so new workspaces no longer require manual role or pipeline SQL after ADR032-BOOTSTRAP-IMPL-001. Preserve the separate `agent_definition` provisioning gap as an unresolved adjacent follow-up instead of implying ADR-032 fixed it.

Code affected: Documentation only. Expected files are `docs/plans/external_validation_first_test_battery_plan.md`, `docs/tasks/task_extval_battery_t1_smoke_auth.md`, `docs/tasks/task_extval_battery_t3_support_agent.md`, this task file, and the ADR-032 remediation plan if status notes need sync.

Effort/reasoning: Medium - docs-only change, but it must distinguish resolved pipeline/RBAC bootstrap assumptions from the still-unresolved `agent_definition` provisioning gap across multiple validation artifacts.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Estimated tokens: ~6000

Peer readiness review approval: reviewer=Claude Code; artifact=logs/peer-workflow-review/task-readiness_codex_by_claude_20260702T070533Z.json; status=PASS

Task type: documentation and governance sync. No development-task pseudocode is required.

## Acceptance Criteria

1. `docs/plans/external_validation_first_test_battery_plan.md` no longer treats manual pipeline creation as expected setup for newly registered workspaces.
2. `docs/tasks/task_extval_battery_t1_smoke_auth.md` records that the manual pipeline workaround is historical and superseded for new workspaces by ADR032-BOOTSTRAP-IMPL-001.
3. `docs/tasks/task_extval_battery_t3_support_agent.md` records that first-user role assignment is resolved for new workspaces by ADR032-BOOTSTRAP-IMPL-001 while `agent_definition` seeding remains unresolved.
4. `docs/plans/adr-032-workspace-bootstrap-remediation-plan.md` marks the docs sync workstream complete or clearly records any remaining documentation gap.
5. No product code, schema migration, historical workspace backfill, or `agent_definition` seeding is introduced.

## Closure Notes

- Updated the external-validation battery plan so fresh workspaces use ADR-032 registration defaults instead of manual pipeline setup.
- Updated the T1 validation task to mark manual deal/case pipeline creation as a historical workaround superseded for newly registered workspaces.
- Updated the T3 validation task to mark first-user role assignment as resolved for new workspaces while keeping `agent_definition` provisioning as an unresolved separate gap.
- Marked the ADR-032 remediation plan completed.

## Scope

- In: documentation updates that remove stale manual role/pipeline bootstrap expectations for newly registered workspaces.
- In: explicit note that ADR-032 does not resolve legacy validation workspaces or support-agent `agent_definition` provisioning.
- Out: source code changes, migrations, operational validation reruns, commits, pushes, and any attempt to backfill existing databases.

## Risks

- Overstating ADR-032 could make operators believe old validation workspaces were backfilled automatically.
- Understating ADR-032 could leave stale instructions telling operators to keep manually creating pipelines for fresh workspaces.
- Conflating role/pipeline bootstrap with `agent_definition` provisioning would hide a separate support-agent onboarding gap.
