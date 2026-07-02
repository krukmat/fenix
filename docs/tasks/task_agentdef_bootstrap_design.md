---
doc_type: task
id: AGENTDEF-BOOTSTRAP-DESIGN-001
title: "Design support-agent definition bootstrap contract"
status: completed
phase: remediation
week: "2026-W27"
tags: [adr, agent, bootstrap, onboarding, design, adr-032-followup]
fr_refs: [FR-230]
uc_refs: [UC-C1]
blocked_by: []
blocks: [AGENTDEF-BOOTSTRAP-IMPL-A-001, AGENTDEF-BOOTSTRAP-IMPL-B1-001]
files_affected:
  - docs/tasks/task_agentdef_bootstrap_design.md
  - docs/plans/agent-definition-bootstrap-remediation-plan.md
created: 2026-07-02
completed: 2026-07-02
---

# Task AGENTDEF-BOOTSTRAP-DESIGN-001

**Plan**: [Support-agent definition bootstrap remediation](../plans/agent-definition-bootstrap-remediation-plan.md)

## Task Card

Task: AGENTDEF-BOOTSTRAP-DESIGN-001

Task file: docs/tasks/task_agentdef_bootstrap_design.md

Plan file: docs/plans/agent-definition-bootstrap-remediation-plan.md

Summary: Decide the bootstrap shape (extend `Register`'s existing transaction vs. a dedicated provisioning endpoint) and lock the default `agent_definition` values for `support-agent` — id scheme, `allowed_tools` subset, `limits` — before any code is written. Closes the design ambiguity left in the plan's W1 workstream.

Code affected: Documentation only (this task file, and the plan file if the decision changes recorded options). No source files touched.

Effort/reasoning: Medium - no code changes, but the decision has downstream security/cost implications (agent tool reach, run-cost ceiling) and must reconcile with the ADR-032 `workspace_owner` tool grant precedent.

Recommended model: claude-sonnet-4-6

Estimated tokens: ~3000

Task type: design/decision record. No development-task pseudocode required — this task produces a decision, not code.

## Acceptance Criteria

1. Bootstrap shape decided and recorded: Option A (extend `Register`'s existing bootstrap transaction) or Option B (dedicated provisioning endpoint), with rationale.
2. Default `support-agent` `agent_definition` field values locked: `id` scheme, `name`, `agent_type`, `status`, `allowed_tools`, `limits` (explicit `max_runs_day`/`max_cost_day` or equivalent).
3. Explicit invariant recorded: `support-agent.allowed_tools` must be a subset of the ADR-032 `workspace_owner` bootstrap tool grant (`internal/domain/auth/service.go:33`), so the agent never has more reach than the operator who owns the workspace.
4. Decision on whether a new ADR is warranted, or whether this plan-level decision record is sufficient.
5. No product code, schema migration, or `agent_definition` row creation — decision record only.

## Scope

- In: locking the design contract described in plan workstream W1.
- Out: implementing `internal/domain/auth/service.go` changes (belongs to `AGENTDEF-BOOTSTRAP-IMPL-001`, which is RRI-gated separately since it touches the auth bootstrap transaction and a security-relevant permission surface).
- Out: any REST route implementation, even if Option B is selected.

## Risks

- Picking an `allowed_tools` list broader than the bootstrap role's own grant would let the support agent act beyond what the workspace owner who triggers it could do directly — must be explicitly checked against `defaultWorkspaceOwnerPermissions` (`internal/domain/auth/service.go:33`).
- Picking unlimited or missing `limits` would create an unbounded-cost agent at bootstrap time, contradicting the cost-governance direction in ADR-020.
- **Pre-existing collision risk, confirmed during design research (blocking, must resolve in this task, not deferred to impl):**
  `internal/domain/agent/agents/support.go:508` hardcodes
  `AgentID: "support-agent"` as a literal string on every
  `TriggerSupportAgent` call, for every workspace. `agent_definition.id` is a
  **global** primary key (`internal/infra/sqlite/migrations/018_agents.up.sql`,
  `id TEXT PRIMARY KEY` — no workspace-scoped uniqueness on `id` itself, only
  `UNIQUE(workspace_id, name)` is workspace-scoped). The orchestrator lookup
  (`internal/domain/agent/orchestrator.go:814-820`) is
  `WHERE id = ? AND workspace_id = ?`, which only works today because exactly
  one workspace has ever had this row (manually inserted for validation). The
  literal `"support-agent"` id is therefore load-bearing application
  behavior, not just a test-fixture convention — bootstrapping a second
  workspace with a UUID `id` would silently break `TriggerSupportAgent` for
  that workspace (lookup would find nothing, `ErrAgentNotFound`), while
  bootstrapping with the same literal `"support-agent"` id for a second
  workspace would hit the `PRIMARY KEY` constraint on insert.
  This task must resolve one of:
  (a) change `TriggerSupportAgent` to look up the definition by
  `(workspace_id, agent_type="support")` or `(workspace_id, name="Support Agent")`
  instead of a hardcoded global id, then bootstrap can safely use a
  per-workspace UUID `id`; or
  (b) keep the literal id convention but scope it per workspace some other way
  (e.g. compose `id` from workspace_id, which changes the hardcoded literal in
  `support.go:508` anyway).
  Either way, `internal/domain/agent/agents/support.go` is now in scope for
  `AGENTDEF-BOOTSTRAP-IMPL-001` — the plan's "expected code areas" list must be
  updated to include it, not just `internal/domain/auth/service.go`.

## Decision Record

1. **Bootstrap shape: Option A.** Extend `bootstrapWorkspaceDefaults`
   (`internal/domain/auth/service.go:193`) inside `Register`'s existing
   transaction, mirroring the ADR-032 role/pipeline pattern exactly. No new
   REST route. Rationale: closes the gap for every new workspace with zero
   operator action, matches an already-reviewed precedent, and avoids adding a
   provisioning endpoint for a single hardcoded agent type ahead of Agent
   Studio (P1), which is the correct place for general agent-definition CRUD.

2. **Default `support-agent` `agent_definition` values:**
   - `id`: bootstrap-generated `uuid.NewV7().String()` — **not** the literal
     `"support-agent"`. This resolves the collision risk below.
   - `name`: `"Support Agent"`
   - `agent_type`: `"support"`
   - `status`: `"active"`
   - `allowed_tools`: `["update_case", "send_reply", "create_task"]`.
     **Corrected 2026-07-02 during `AGENTDEF-BOOTSTRAP-IMPL-A-001` code
     cross-check.** The earlier draft of this line guessed
     `["update_case", "send_reply", "get_lead", "get_account", "get_deal"]`,
     which was wrong: `SupportAgent.AllowedTools()`
     (`internal/domain/agent/agents/support.go:114`) actually returns
     `["update_case", "send_reply", "create_task", "search_knowledge",
     "get_case", "get_contact"]`, and it does **not** use
     `get_lead`/`get_account`/`get_deal`. Of the agent's declared six, only
     `update_case`, `send_reply`, and `create_task` are (a) registered as
     enforced built-in executors (`internal/domain/tool/builtin.go:187-210`)
     and (b) present in the `workspace_owner` grant
     (`internal/domain/auth/service.go:33`). The other three
     (`search_knowledge`, `get_case`, `get_contact`) are **conceptual
     capabilities the agent declares but does not execute through the
     enforced tool surface** — they have no registered executor, matching the
     note already in this plan's confirmed-facts list. Decision (user-
     confirmed): bootstrap `allowed_tools` to the three enforced tools only.
     This preserves the subset-of-owner invariant with zero loss of
     executable capability. If `search_knowledge`/`get_case`/`get_contact`
     ever become real enforced tools, adding them to both the owner grant and
     this list is a deliberate future change, not an implicit one.
   - `limits`: `{"max_runs_day": 100, "max_cost_day_eur": 5}` as the initial
     conservative bootstrap ceiling. Not a final production tuning number —
     an explicit non-empty starting point per ADR-020's cost-governance
     direction, adjustable later via whatever admin surface Agent Studio (P1)
     eventually provides. No admin override mechanism exists yet, so this
     value must not be so low it blocks the P0 support flow in
     `EXTVAL-BATTERY-T3-001`-style validation (that run's real trigger volume
     was 1 run).
   - `objective`, `trigger_config`, `policy_set_id`,
     `active_prompt_version_id`: leave at column defaults / `NULL`, same as
     the existing manual test-fixture insert — no bootstrap value needed for
     P0.

3. **Tool-subset invariant:** confirmed — `allowed_tools` above is a strict
   subset of `defaultWorkspaceOwnerPermissions.tools`
   (`internal/domain/auth/service.go:33`). No tool appears in the agent grant
   that is absent from the owner grant.

4. **Id-collision resolution (from Risks, now closed):** Option (a). Change
   `triggerSupportRun` (`internal/domain/agent/agents/support.go:499-518`) to
   resolve the workspace's support-agent definition via the existing
   `ListAgentDefinitionsByType(workspace_id, "support")` sqlc query
   (`internal/infra/sqlite/sqlcgen/agent.sql.go:447`) instead of hardcoding
   `AgentID: "support-agent"`, then pass the resolved row's `id` into
   `TriggerAgentInput.AgentID`. This requires wiring a definition-lookup
   dependency into `SupportAgent` (currently it only holds the orchestrator);
   exact plumbing is an implementation decision for
   `AGENTDEF-BOOTSTRAP-IMPL-001`, not re-litigated here. Note for the
   implementer: `prospecting.go`, `kb.go`, `insights.go`, `deal_risk.go` have
   the identical hardcoded-`AgentID` pattern — out of scope for this P0 fix,
   but flag it so a future P1 Agent Studio task doesn't rediscover it from
   scratch.

5. **New ADR: not warranted.** This extends the already-accepted ADR-032
   bootstrap pattern (same transaction boundary, same atomicity invariant) to
   one more table; it does not introduce a new architectural stance. This
   plan-level decision record is sufficient. If a future change generalizes
   agent-definition bootstrap beyond `support-agent`, that decision (broader
   CRUD surface, multi-agent-type bootstrap policy) would warrant its own ADR
   at that time.

**Acceptance criteria status:** 1-4 satisfied above; 5 satisfied — no code,
schema, or `agent_definition` row was created by this task.
