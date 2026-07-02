---
doc_type: task
id: ADR032-BOOTSTRAP-IMPL-001
title: "Implement ADR-032 transactional workspace bootstrap"
status: completed
phase: remediation
week: "2026-W27"
tags: [adr, auth, rbac, pipeline, onboarding, transaction]
fr_refs: [FR-060, FR-070, FR-071]
uc_refs: []
blocked_by: [ADR032-BOOTSTRAP-DESIGN-001]
blocks: [ADR032-BOOTSTRAP-DOCS-001]
files_affected:
  - internal/domain/auth/service.go
  - internal/domain/auth/service_test.go
  - internal/domain/auth/service_internal_test.go
  - internal/api/integration_crm_test.go
  - docs/plans/adr-032-workspace-bootstrap-remediation-plan.md
  - docs/tasks/task_adr_032_bootstrap_impl.md
created: 2026-07-02
completed: 2026-07-02
---

# Task ADR032-BOOTSTRAP-IMPL-001

**Plan**: [ADR-032 workspace bootstrap remediation](../plans/adr-032-workspace-bootstrap-remediation-plan.md)

## Task Card

Task: ADR032-BOOTSTRAP-IMPL-001

Task file: docs/tasks/task_adr_032_bootstrap_impl.md

Plan file: docs/plans/adr-032-workspace-bootstrap-remediation-plan.md

Summary: Extend `Register` so a new workspace is created with its first-user role assignment and default `deal`/`case` pipelines inside the same transaction. The implementation must follow the design contract from ADR032-BOOTSTRAP-DESIGN-001 exactly, preserve duplicate-email behavior, and prove rollback safety with focused regression coverage.

Code affected: Product code changes are expected in `internal/domain/auth/`, transaction-bound query usage for role/pipeline bootstrap, and focused regression tests in auth and API integration paths. No schema migration or historical backfill is in scope.

Effort/reasoning: High - auth-boundary change, cross-table transaction semantics, rollback correctness, and regression coverage across domain plus API behavior. Measured RRI=64 (Complex band), so this task stays decomposed and requires explicit approval before implementation.

Recommended model: OpenAI: gpt-5.5 | Anthropic: claude-opus-4-8

Estimated tokens: ~14000

Peer readiness review approval: reviewer=Claude Code; artifact=logs/peer-workflow-review/task-readiness_codex_by_local-gemma_20260702T063739Z.json; status=PASS

Pseudocode:

```text
hash password
begin registration transaction
insert workspace
insert user_account
insert workspace_owner role with fixed permission JSON
insert user_role linking first user to workspace_owner
insert Sales pipeline for deal and Discovery stage
insert Support pipeline for case and Open stage
commit transaction

in tests:
  register fresh workspace and assert role/user_role/pipelines/stages exist
  register duplicate email and assert no extra bootstrap rows are committed
  force bootstrap failure after transaction start and assert full rollback
  register through API path and verify fresh workspace can create deal and case without manual pipeline creation
```

## Acceptance Criteria

1. `Register` creates `workspace_owner` and assigns it to the registering user in the same transaction as workspace/user creation.
2. `Register` creates default `deal` and `case` pipelines with one stage each in that same transaction.
3. A failure in any bootstrap insert leaves no partial workspace, user, role, or pipeline rows committed.
4. Fresh registration no longer requires manual pipeline creation before `deal` or `case` creation in focused regression coverage.
5. No migration or automatic backfill for pre-existing workspaces is introduced.

## Closure Notes

- Implemented the transactional bootstrap inside `Register`'s existing workspace/user transaction.
- Added auth-domain coverage for default role assignment, default pipeline/stage rows, duplicate-email rollback safety, and injected bootstrap failure rollback.
- Added API integration coverage proving a freshly registered workspace can create a `deal` and `case` using seeded pipelines without manual pipeline creation.
- No schema migration, sqlc regeneration, historical backfill, or `agent_definition` seeding was introduced.

## Scope

- In: transactional bootstrap implementation, focused regression tests, and plan/task sync required by the implementation.
- Out: schema migrations, historical workspace backfill, `agent_definition` seeding, and broader RBAC redesign.

## Risks

- A helper that silently escapes the registration transaction would recreate the defect under partial-failure conditions.
- Regression tests can miss rollback behavior unless failure is injected after some bootstrap rows are already written.
- Overreaching into agent-definition bootstrap would mix two separate defects and make verification ambiguous.
