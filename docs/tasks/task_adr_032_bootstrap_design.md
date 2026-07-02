---
doc_type: task
id: ADR032-BOOTSTRAP-DESIGN-001
title: "Define the ADR-032 bootstrap contract for default role and pipeline seeds"
status: done
phase: governance
week: "2026-W27"
tags: [adr, design, auth, rbac, pipeline, onboarding]
fr_refs: [FR-060, FR-070, FR-071]
uc_refs: []
blocked_by: []
blocks: [ADR032-BOOTSTRAP-IMPL-001]
files_affected:
  - docs/decisions/ADR-032-workspace-bootstrap-defaults.md
  - docs/plans/adr-032-workspace-bootstrap-remediation-plan.md
  - docs/tasks/task_adr_032_bootstrap_design.md
created: 2026-07-02
completed: 2026-07-02
---

# Task ADR032-BOOTSTRAP-DESIGN-001

**Plan**: [ADR-032 workspace bootstrap remediation](../plans/adr-032-workspace-bootstrap-remediation-plan.md)

## Task Card

Task: ADR032-BOOTSTRAP-DESIGN-001

Task file: docs/tasks/task_adr_032_bootstrap_design.md

Plan file: docs/plans/adr-032-workspace-bootstrap-remediation-plan.md

Summary: Define the exact bootstrap contract that ADR-032 leaves intentionally open: the first-user default role, its minimum permission JSON, and the default `deal` and `case` pipeline/stage seeds. The output must give the implementation task an unambiguous contract without expanding scope into legacy backfill or unrelated agent-definition provisioning.

Code affected: No product code changes are planned in this task. Expected analysis and documentation areas are docs/decisions/, docs/plans/, docs/tasks/, internal/domain/policy/, internal/domain/tool/, internal/domain/auth/, and the external-validation task evidence that motivated ADR-032.

Effort/reasoning: Medium - docs-only work, but it sets the permission and onboarding contract for an auth-boundary change and must stay precise enough to prevent over-broad grants.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Estimated tokens: ~9000

Peer readiness review approval: reviewer=Claude Code; artifact=logs/peer-workflow-review/task-readiness_codex_by_local-gemma_20260702T062057Z.json; status=PASS

## Acceptance Criteria

1. The default role name, description, and minimum permission JSON are stated explicitly.
2. The default pipeline and stage names for `deal` and `case` are stated explicitly.
3. The design records invariants for atomicity, no implicit backfill, and no dependency on manual SQL after `Register`.
4. The design states what remains out of scope, including `agent_definition` provisioning and broader RBAC redesign.
5. The final report distinguishes confirmed implementation facts from contract choices that remain recommendations until accepted.

## Scope

- In: contract definition, invariant capture, dependency notes, and documentation updates needed to make the implementation task unambiguous.
- Out: product code changes, migrations, commits, pushes, or executing the bootstrap implementation itself.

## Risks

- Over-scoping the role permissions would turn a usability fix into a silent privilege expansion.
- Under-scoping the permissions would preserve the “fresh workspace is unusable” failure mode in a narrower form.
- Mixing support-agent provisioning into this task would blur two different root causes and make verification ambiguous.

## Confirmed Code-Path Facts

- The role permission payload is a JSON object keyed by resource, for example `{"tools":["send_reply"]}`, not a flat string list.
- Standard CRM CRUD handlers for `accounts`, `contacts`, `deals`, `cases`, and `pipelines` are currently protected by JWT plus workspace isolation; they do not perform per-action RBAC checks in the handler path.
- The built-in tool permission surface currently enforced through `PolicyEngine` includes `create_task`, `update_case`, `update_deal`, `send_reply`, `get_lead`, `get_account`, `get_deal`, `create_knowledge_item`, `update_knowledge_item`, and `query_metrics`.
- The observed T3 failure was caused by built-in tool authorization (`send_reply`) and not by missing `api.admin.*` permission.
- The policy engine already defines `agents: execute` as the non-admin agent-dispatch permission, even though the current manual support trigger path does not yet enforce it directly.
- `deal` creation requires a valid `pipeline_id` and `stage_id`; `case` creation accepts optional pipeline/stage fields but the external-validation evidence and ADR scope treat `case` as part of the minimum seeded operational state.

## Design Contract

Default role:

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

- No `global:["admin"]`.
- No `api:["admin"]`.
- No wildcard permissions.
- No automatic grants for future tools or admin surfaces.

Default pipeline seeds:

1. `deal` bootstrap seed:
   - pipeline name: `Sales`
   - first stage name: `Discovery`
   - first stage position: `1`
2. `case` bootstrap seed:
   - pipeline name: `Support`
   - first stage name: `Open`
   - first stage position: `1`

For both seeds, `probability`, `sla_hours`, and `required_fields` stay unset by default.

## Invariants

- `Register` must commit `workspace`, `user_account`, `role`, `user_role`, and both pipeline seeds in one atomic transaction.
- Any failure after transaction start must roll back the entire bootstrap.
- No historical workspace backfill is implied by this contract.
- No `agent_definition` bootstrap is implied by this contract.
- A future entity may only join the bootstrap set when its create path becomes pipeline-gated and the same change adds an explicit seed contract.
