---
doc_type: plan
title: "Support-agent definition bootstrap remediation"
status: proposed
owner: "Auth / Agent Orchestration / Governance"
created: 2026-07-02
updated: 2026-07-02
tags: [agent, bootstrap, auth, onboarding, remediation, adr-032-followup]
---

# Support-agent definition bootstrap remediation

## Purpose

Close the adjacent gap left open by ADR-032: a freshly registered workspace has
a default role and default `deal`/`case` pipelines, but no `agent_definition`
row, so the support agent (the P0 wedge agent, UC-C1) cannot be triggered
without an out-of-band SQL insert. Give operators a real, governed path to
provision the support-agent definition — either automatically at registration
or through an authenticated API — without inventing the broader Agent Studio
CRUD surface (`FR-240/241/242`, P1) ahead of schedule.

## Confirmed problem statement

- `internal/domain/agent/orchestrator.go:187` (`TriggerAgent` →
  `getAgentDefinition`) fails immediately when `agent_definition` has 0 rows
  for the workspace.
- `Register` (`internal/domain/auth/service.go`) provisions `role`,
  `user_role`, and default `deal`/`case` pipelines (ADR-032,
  `ADR032-BOOTSTRAP-IMPL-001`), but does not touch `agent_definition`.
- There is no REST route to create an `agent_definition`. The only route
  surface under `/agents` (`internal/api/routes.go:489-501`) is triggers,
  run listing, run detail, cancel, and handoff — no create/provision.
- The only existing precedent for inserting a row is a raw SQL `INSERT` in a
  unit-test fixture (`internal/domain/agent/agents/support_test.go:69-78`),
  which is not something a real operator can use.
- This was documented as an unresolved finding in `EXTVAL-BATTERY-T3-001`
  (`docs/tasks/task_extval_battery_t3_support_agent.md`, Finding 1) and
  explicitly deferred out of `ADR-032` scope
  (`docs/decisions/ADR-032-workspace-bootstrap-defaults.md`, and
  `docs/plans/adr-032-workspace-bootstrap-remediation-plan.md`, "Out of
  Scope").

## Scope decision

Per `CLAUDE.md` P0 priorities, only the support agent (UC-C1, FR-230) is in
the current wedge. Prospecting, KB, insights, and deal-risk agents are P1
catalog agents (FR-231) gated behind Agent Studio (FR-240/241/242), which is
not yet built. Seeding definitions for those agent types now would grant
execution surface for agents the product has not committed to shipping yet.

Decision: bootstrap only the `support-agent` definition automatically. Do not
build a general-purpose agent-definition CRUD API in this remediation — that
belongs to Agent Studio (P1). If an operator needs a non-support agent
definition before Agent Studio ships, that remains an explicit manual/SQL
step, same as today, and is out of scope here.

## Out of scope

- Agent Studio CRUD surface for arbitrary agent definitions (FR-240/241/242).
- Seeding `prospecting`, `kb`, `insights`, or `deal-risk` definitions.
- Backfilling `agent_definition` for existing workspaces created before this
  change ships (same backfill exclusion precedent as ADR-032).
- Changing `TriggerAgent` orchestration behavior, tool permissions, or the
  policy engine.
- Reworking `skill_definition` or prompt-versioning.

## Decision (W1 — closed 2026-07-02)

Resolved by `AGENTDEF-BOOTSTRAP-DESIGN-001`
(`docs/tasks/task_agentdef_bootstrap_design.md`, Decision Record). Summary:
Option A (bootstrap inside `Register`'s existing transaction), bootstrap-
generated UUID `id` (not the literal `"support-agent"`), `allowed_tools`
scoped to what the support agent actually calls, `limits` set to
`{"max_runs_day": 100, "max_cost_day_eur": 5}`, and the pre-existing
hardcoded-`AgentID` lookup bug in `support.go:508` resolved by switching to
`ListAgentDefinitionsByType(workspace_id, "support")`. No new ADR. Full
rationale and exact values are recorded in the task file, not duplicated
here — that task file is now the source of truth for the W1 contract.

<details>
<summary>Original options considered (kept for context)</summary>

Two viable shapes were considered:

**Option A — Bootstrap at registration (same transaction as ADR-032).**
Add a `CreateAgentDefinition` call inside `bootstrapWorkspaceDefaults`
(`internal/domain/auth/service.go:193`), using the existing
`sqlcgen.CreateAgentDefinition` query (`internal/infra/sqlite/sqlcgen/agent.sql.go`).
Pros: zero new API surface, matches the ADR-032 precedent exactly, closes the
gap for every new workspace with no operator action. Cons: couples agent
provisioning further into the auth transaction; if `agent_definition` schema
gains required fields later (e.g. mandatory `policy_set_id`), `Register` must
track that.

**Option B — Dedicated authenticated provisioning endpoint.**
Add `POST /api/v1/agents/definitions` (workspace-owner permission only),
backed by the existing `CreateAgentDefinition` sqlc query, and call it once
from `Register` (or document it as a required first-login step). Pros:
reusable outside registration, smaller blast radius on the auth transaction.
Cons: more surface (handler, route, permission check) for a P0 gap that
Option A closes in ~15 lines.

Recommendation: **Option A**, for consistency with the ADR-032 precedent and
because it fully closes the gap without leaving a manual step. Final call
belongs to whoever approves the task card.

</details>

## Workstreams

### W1. Confirm bootstrap shape and default definition contract

Status: completed by `AGENTDEF-BOOTSTRAP-DESIGN-001` on 2026-07-02. See that
task file's Decision Record for the locked values (Option A, bootstrap-
generated UUID `id`, `allowed_tools` subset, `limits`, and the id-collision
fix requirement).

### W2 + W3. Implement bootstrap and fix the AgentID lookup, with regression coverage

Status: decomposed into three subtasks per the RRI ≥56 decomposition gate
(`docs/policies/RRI_POLICY.md`). The original single-task estimate scored
RRI=69 (Complex); `internal/domain/agent/agents/support.go` in particular
scored RRI=64 on its own because it touches the only P0 agent's trigger path
with no existing dedicated test coverage (T≥4 ∧ P≥4 trigger — characterization
tests required first).

1. **`AGENTDEF-BOOTSTRAP-IMPL-A-001`** (RRI≈52) — extend
   `bootstrapWorkspaceDefaults` (`internal/domain/auth/service.go:193`) to
   insert the `support-agent` `agent_definition` row inside the existing
   `Register` transaction, using the W1-locked defaults and the existing
   `sqlcgen.CreateAgentDefinition` query. Purely additive; does not change
   `support.go`. Commit only after all inserts (role, user_role, pipelines,
   agent definition) succeed, reusing the existing `defer tx.Rollback()` /
   `tx.Commit()` boundary in `insertWorkspaceAndUser`.
2. **`AGENTDEF-BOOTSTRAP-IMPL-B1-001`** (RRI≈32) — characterization tests
   pinning down `triggerSupportRun`'s current hardcoded-`AgentID` behavior
   (success path, missing-definition failure, and the cross-workspace
   collision symptom), before any fix is applied. Required by the T≥4∧P≥4
   decomposition trigger.
3. **`AGENTDEF-BOOTSTRAP-IMPL-B2-001`** (RRI≈45) — the actual fix:
   `internal/domain/agent/agents/support.go:508` currently hardcodes
   `AgentID: "support-agent"` as a literal global id.
   `agent_definition.id` is a global primary key, not workspace-scoped, so a
   bootstrap row created per workspace by IMPL-A cannot be found by that
   literal lookup for any workspace other than whichever one happened to get
   the id first. Fix: resolve the definition via
   `ListAgentDefinitionsByType(workspace_id, "support")`
   (`internal/infra/sqlite/sqlcgen/agent.sql.go:447`) and use the resolved
   row's `id`. Depends on B1's tests as the reviewable baseline.

Required verification targets across the three subtasks (moved from the
former W3):
- `Register` persists a `support-agent` `agent_definition` row for a fresh
  workspace (A).
- A freshly registered workspace can call `POST /agents/support/trigger`
  without a prior manual `agent_definition` insert — the actual acceptance
  bar, mirroring what `EXTVAL-BATTERY-T3-001` had to work around manually
  (A + B1 + B2 together).
- Duplicate-email failure still leaves no partial records (A).
- Bootstrap failure after transaction start still rolls back all created
  rows, including the new agent definition insert (A).
- Cross-workspace collision is demonstrated as a failing case before the fix,
  and as a passing case after (B1 then B2).

### W4. Sync validation and governance artifacts

Required doc updates after code lands:
- Update `docs/tasks/task_extval_battery_t3_support_agent.md` Finding 1 to
  record resolution (or partial resolution) for newly registered workspaces,
  the same way `ADR032-BOOTSTRAP-DOCS-001` updated Finding 2.
- Update `docs/plans/adr-032-workspace-bootstrap-remediation-plan.md` cross-
  reference note if it should point at this plan as the resolution of the
  previously "still unresolved" gap.
- If an ADR is warranted for this decision (agent-definition bootstrap scope,
  tool-list subset invariant), add one under `docs/decisions/` and update
  `docs/decisions/README.md`. Given this extends an already-decided ADR-032
  pattern rather than introducing a new architectural stance, a new ADR is
  optional — default to a plan-level decision unless the approver wants one.

## Proposed task decomposition

1. `AGENTDEF-BOOTSTRAP-DESIGN-001` — confirm W1 contract (Option A/B, default
   definition values, tool-list subset invariant, cost limits). **Completed
   2026-07-02.**
2. `AGENTDEF-BOOTSTRAP-IMPL-A-001` — bootstrap the `agent_definition` row in
   `Register` (RRI≈52). **Completed 2026-07-02.**
3. `AGENTDEF-BOOTSTRAP-IMPL-B1-001` — characterization tests for
   `triggerSupportRun`'s current lookup behavior (RRI≈32). **Completed
   2026-07-02.**
4. `AGENTDEF-BOOTSTRAP-IMPL-B2-001` — fix the hardcoded `AgentID` lookup,
   blocked on A and B1 (RRI≈45). **Completed 2026-07-02:**
   `triggerSupportRun` now resolves the definition id via
   `Orchestrator.ListAgentDefinitionsByType(workspace_id, "support")`
   instead of the literal `"support-agent"`; missing provisioning surfaces
   as `ErrSupportAgentNotProvisioned` (mapped to HTTP 404, not a generic
   500). B1's characterization tests were updated in place to assert the
   fixed behavior, plus a new end-to-end test proves a workspace
   bootstrapped purely by `Register` (no manual insert) can trigger
   successfully.
5. `AGENTDEF-BOOTSTRAP-DOCS-001` — implement W4, blocked on B2. **Completed
   2026-07-02:** updated `task_extval_battery_t3_support_agent.md` Finding 1
   and the `adr-032-workspace-bootstrap-remediation-plan.md` cross-reference
   to record resolution for newly registered workspaces.

(The original single `AGENTDEF-BOOTSTRAP-IMPL-001` task scored RRI=69 —
Complex band — which is a hard decomposition gate per
`docs/policies/RRI_POLICY.md`. Steps 2-4 above replace it.)

## Verification strategy

- Domain tests: `go test ./internal/domain/auth/...`
- Agent orchestration tests: `go test ./internal/domain/agent/...`
- If shared runtime helpers change, run the broader repo gate selected by the
  touched surface before any push (`scripts/qa-go-prepush.sh`).

## Exit criteria

- A fresh workspace registration yields a usable `support-agent`
  `agent_definition` row, with no manual SQL step required.
- The bootstrap remains atomic: no partial workspace survives a failed
  registration, including the new agent-definition insert.
- The support agent's `allowed_tools` is not broader than the bootstrap
  `workspace_owner` role's tool grant.
- `EXTVAL-BATTERY-T3-001` Finding 1 is updated to reflect resolution for new
  workspaces, without implying legacy validation workspaces were backfilled.
