---
doc_type: plan
title: "ADR-032 workspace bootstrap remediation"
status: completed
owner: "Auth / CRM / Governance"
created: 2026-07-02
updated: 2026-07-02
tags: [adr, auth, rbac, pipeline, onboarding, remediation]
---

# ADR-032 workspace bootstrap remediation

## Purpose

Turn ADR-032 into an implementation-ready workstream that makes a freshly registered workspace usable without manual SQL provisioning.

## Confirmed Problem Statement

`Register` currently commits only `workspace` and `user_account`. It does not provision:

- a default `role` plus `user_role` assignment for the first user
- a default `pipeline` plus at least one `pipeline_stage` for pipeline-gated entities

This leaves a brand-new workspace unable to perform baseline CRM operations without out-of-band setup.

## Decisions Already Fixed By ADR-032

- Bootstrap must happen at registration time, not as a manual operator step.
- Role assignment and pipeline creation belong in the same atomic bootstrap boundary as workspace/user creation.
- This work governs only newly registered workspaces going forward.
- Backfill for already-existing workspaces is explicitly out of scope.

## Out of Scope

- Reworking the overall RBAC model or inventing a new permissions framework.
- Backfilling historical workspaces.
- Agent definition seeding for support-agent execution. That remains a separate follow-up from EXTVAL-BATTERY-T3-001.
- Approval-routing behavior already covered by ADR-031 and its follow-up tasks.

## Workstreams

### W1. Define the bootstrap contract

Goal: remove ambiguity before code changes begin.

Deliverables:

- default role name, description, and minimum permission JSON for first-user ownership
- default pipeline/stage names for `deal` and `case`
- explicit invariant list:
  - a successful `Register` returns only after role assignment and pipeline seeding succeed
  - a failed bootstrap leaves no partial workspace behind
  - no legacy workspace backfill is triggered implicitly

Notes:

- Permission scope should satisfy basic CRM writes plus the tool-gated actions explicitly intended for the first workspace owner, without granting broad wildcard permissions by default.
- Pipeline defaults should be minimal and deterministic; this ADR does not require multi-stage templates.

### W1 Contract Outcome

Default bootstrap role:

- Name: `workspace_owner`
- Description: `Default first-user role created during workspace registration bootstrap.`

Minimum permission JSON:

```json
{
  "records": ["read_all"],
  "agents": ["execute"],
  "tools": [
    "create_task",
    "update_case",
    "update_deal",
    "send_reply",
    "get_lead",
    "get_account",
    "get_deal",
    "create_knowledge_item",
    "update_knowledge_item",
    "query_metrics"
  ]
}
```

Explicit exclusions:

- no `global:["admin"]`
- no `api:["admin"]`
- no wildcard keys or wildcard actions
- no implicit permission grants for future tools; new built-in tools must be added deliberately

Rationale:

- `records:["read_all"]` keeps retrieval and copilot filters workspace-wide for the first owner instead of silently degrading to owner-only reads.
- `agents:["execute"]` aligns the first-user role with the existing policy contract for agent execution without granting admin API powers.
- The `tools` list covers the currently enforced built-in tool surface used by first-party agents and operator workflows. It is intentionally explicit rather than wildcarded.

Default bootstrap pipelines:

1. `deal`
   - pipeline name: `Sales`
   - stage name: `Discovery`
   - stage position: `1`
   - optional fields left unset: `probability`, `sla_hours`, `required_fields`
2. `case`
   - pipeline name: `Support`
   - stage name: `Open`
   - stage position: `1`
   - optional fields left unset: `probability`, `sla_hours`, `required_fields`

Boundary rules:

- Only entities that are currently pipeline-gated at create time are seeded by default.
- `account`, `contact`, and `lead` receive no pipeline bootstrap because their create paths do not require one today.
- A future pipeline-gated entity must add its default bootstrap seed in the same change that introduces the requirement.
- `agent_definition` rows are not part of this bootstrap contract.

Confirmed implementation facts behind this contract:

- Standard CRM CRUD handlers are JWT/workspace-gated today and do not currently perform per-action RBAC checks in the handler path.
- The observed T3 blocker came from built-in tool authorization, not from `api.admin.*` checks.
- The current manual support-agent trigger path does not call `CheckAgentPermission`, but the policy engine already defines `agents: execute` as the non-admin agent-dispatch permission.
- Conceptual support-agent capabilities such as `search_knowledge`, `get_case`, and `get_contact` are not currently enforced through the built-in tool permission surface that caused the T3 failure.

### W2. Implement transactional bootstrap in auth

Goal: extend the current registration transaction so identity and baseline operability are created together.

Status: completed by `ADR032-BOOTSTRAP-IMPL-001` on 2026-07-02.

Expected code areas:

- `internal/domain/auth/service.go`
- `internal/domain/auth/service_test.go`
- `internal/infra/sqlite/sqlcgen/role.sql.go`
- `internal/infra/sqlite/sqlcgen/pipeline.sql.go`
- any helper needed to bind sqlc queries to the same `tx`

Implementation shape:

- keep `Register` as the single orchestration entrypoint
- add a bootstrap helper invoked inside the existing transaction
- create:
  - `role`
  - `user_role`
  - one `pipeline` + one `pipeline_stage` for `deal`
  - one `pipeline` + one `pipeline_stage` for `case`
- commit only after all inserts succeed

### W3. Add regression coverage for usable fresh workspaces

Goal: prove the defect is fixed from both the domain and API perspectives.

Status: completed by `ADR032-BOOTSTRAP-IMPL-001` on 2026-07-02.

Required verification targets:

- `Register` persists role assignment and default pipelines for a fresh workspace
- duplicate-email failure still leaves no partial records
- bootstrap failure after transaction start rolls back all created rows
- a fresh registration can create `deal` and `case` without manual pipeline seeding

Notes:

- Support-agent trigger validation is not the acceptance gate for ADR-032 because T3 also exposed a separate `agent_definition` provisioning gap.

### W4. Sync validation and governance artifacts

Goal: remove stale assumptions from operator-facing evidence.

Status: completed by `ADR032-BOOTSTRAP-DOCS-001` on 2026-07-02.

Required doc updates after code lands:

- update `docs/plans/external_validation_first_test_battery_plan.md` so T1 no longer treats manual pipeline creation as expected setup
- update the relevant external-validation task records that currently document manual SQL/bootstrap workarounds
- add a short follow-up note if agent-definition bootstrap is still unresolved, so future operators do not conflate the two defects

## Proposed Task Decomposition

1. `ADR032-BOOTSTRAP-DESIGN-001`
   Define the default role permission set and default pipeline/stage contract with explicit acceptance criteria.
2. `ADR032-BOOTSTRAP-IMPL-001`
   Implement the transactional bootstrap in `Register` and add domain/API regression coverage.
3. `ADR032-BOOTSTRAP-DOCS-001`
   Update external-validation and governance artifacts to reflect the new default behavior and remaining adjacent gaps.

## Verification Strategy

- Domain tests: `go test ./internal/domain/auth/...`
- API/integration tests: targeted `go test ./internal/api/...`
- If shared runtime helpers change, run the broader repo gate selected by the touched surface before any push.

## Exit Criteria

- A fresh workspace registration yields a first user with a default role assignment.
- A fresh workspace has default `deal` and `case` pipelines with at least one stage each.
- The bootstrap path is atomic: no partial workspace survives a failed registration.
- External-validation docs no longer instruct operators to provision pipelines manually for newly registered workspaces.
