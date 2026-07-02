---
id: ADR-032
title: "Workspace bootstrap defaults: a freshly registered workspace has no usable role or pipeline"
date: 2026-07-02
status: proposed
deciders: [matias]
tags: [adr, auth, rbac, pipeline, workspace, onboarding, governance]
related_tasks: [EXTVAL-BATTERY-T1-001, EXTVAL-BATTERY-T3-001]
related_frs: [FR-060, FR-070, FR-071]
---

# ADR-032 — Workspace bootstrap defaults: a freshly registered workspace has no usable role or pipeline

## Status

`proposed`

## Context

Two independent external-validation battery tasks surfaced the same underlying gap from
different angles:

- **T1** (`docs/tasks/task_extval_battery_t1_smoke_auth.md`, Deviations From Plan Text):
  creating a `deal` or `case` requires a valid `pipelineId`/`stageId`
  (`internal/api/handlers/deal.go`, `case.go` — `CreateDealRequest`/`CreateCaseRequest`).
  A freshly registered workspace has zero pipelines. T1 had to create one pipeline+stage for
  `deal` and one for `case` by hand before either entity could be created at all.
- **T3** (`docs/tasks/task_extval_battery_t3_support_agent.md`, Finding 2): `Register`
  (`internal/domain/auth/service.go:93-122`) creates a `workspace` and `user_account` row but
  never creates or assigns any `role`/`user_role`. The first user of a new workspace has zero
  permissions, which blocked the support agent's `send_reply` tool call outright
  (`tool.denied: send_reply: permission_denied`) until a role was inserted directly via SQL,
  authorized ad hoc by the user.

Read together, these are one root cause, not two: **`Register` only bootstraps identity
(workspace + user), not the minimum operational state a workspace needs to be usable.**
Every external-validation task since T1 has had to work around this by hand
(SQL inserts, manually created pipelines) rather than through any documented or API-reachable
provisioning path. This is a real product gap, not a validation artifact — a real operator
onboarding through `/bff/auth/register` today would hit the same walls with no recourse.

This directly weakens the governance guarantee CLAUDE.md describes as the product's moat
("Governed: RBAC/ABAC... approval chains") — a user with zero role grants is not "governed
with least privilege," they are unable to do anything, including safe things, until an
operator with direct database access intervenes.

## Decision

**1. `Register` must leave a new workspace in a state where its first user can perform basic
CRM operations without further manual provisioning.**

At minimum, this means:

- A default `role` (e.g. `workspace_owner` or equivalent) is created and assigned to the
  registering user via `user_role`, granting the permission set needed for core CRM writes and
  the tool calls already documented as allowed in CLAUDE.md's Tool-Gated Actions model.
- A default `pipeline` + at least one `pipeline_stage` exists for each entity type that
  requires one to be created (`deal`, `case`, and any future pipeline-gated entity).

**2. This is bootstrap-time provisioning, not a permissions or pipeline-design change.**

This ADR does not redefine what roles exist, what permissions they carry, or how pipelines are
structured. It only decides that *some* default of each must exist immediately after
`Register`, so that the workspace is usable. The specific default role's permission set and the
specific default pipeline/stage names are implementation details for the follow-up task(s), not
decided here.

**3. Provisioning must happen inside the same transaction as workspace/user creation.**

`insertWorkspaceAndUser` (`internal/domain/auth/service.go:134+`) already creates workspace +
user atomically. Role and pipeline bootstrap rows must be added to that same transaction (or an
equivalently atomic follow-up step) so a workspace can never exist in a partially-provisioned
state — no workspace with a user but no role, and no workspace with entities enabled but no
pipeline.

**4. Existing workspaces created before this fix are out of scope for automatic migration.**

This ADR governs new workspace creation via `Register` going forward. Backfilling roles/
pipelines for already-existing workspaces (including the external-validation workspaces created
during T1-T5) is a separate, explicit decision if ever needed — not an implicit consequence of
this ADR.

## Consequences

- `Register` (or a bootstrap step called immediately after it) gains responsibility for
  creating baseline `role`/`user_role` and `pipeline`/`pipeline_stage` rows. This is new
  product code, to be scoped as its own implementation task(s) after this ADR is accepted.
- The external-validation plan's T1 section should be corrected to no longer describe manual
  pipeline creation as an expected step, once the bootstrap fix lands.
- Future workspace registration (via `/bff/auth/register` or any future onboarding flow) no
  longer requires SQL-level intervention to become usable.
- This does not by itself fix the approval self-approval defect from ADR-031 — that remains a
  separate, already-resolved concern (approver resolution).

## Alternatives considered

**A. Treat T1's pipeline gap and T3's role gap as two separate follow-ups (rejected)**

Both were originally logged as separate findings in their own task files. Splitting them again
into two unrelated implementation tasks would hide that they share one cause (`Register` doing
identity-only bootstrap) and risks two different ad hoc fixes with inconsistent transaction
boundaries. One ADR, with room for one or two scoped implementation tasks under it, keeps the
causal link visible.

**B. Fix this by documenting the manual steps instead of changing product code (rejected)**

This was T1's original own recommendation as a stopgap ("recommend the plan's T1 section be
updated to note this, or that a workspace-bootstrap default pipeline be considered as product
follow-up"). Documentation-only is insufficient because it does not fix the real-operator
onboarding gap — only validation-battery operators reading this ADR would benefit, not real
users registering a workspace in production.

**C. Auto-provision permissions broadly (e.g. grant all tool permissions to the first user)
(rejected)**

This would satisfy T3's immediate blocker but violates least-privilege and CLAUDE.md's RBAC/ABAC
governance principle. The default role's exact permission scope is left to the follow-up
implementation task, not decided here, but it must not be "all permissions" by default.
