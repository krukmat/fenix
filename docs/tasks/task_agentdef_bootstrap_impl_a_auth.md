---
doc_type: task
id: AGENTDEF-BOOTSTRAP-IMPL-A-001
title: "Bootstrap support-agent agent_definition inside Register's transaction"
status: completed
phase: remediation
week: "2026-W27"
tags: [agent, bootstrap, auth, onboarding, adr-032-followup]
fr_refs: [FR-230]
uc_refs: [UC-C1]
blocked_by: [AGENTDEF-BOOTSTRAP-DESIGN-001]
blocks: [AGENTDEF-BOOTSTRAP-IMPL-B2-001]
files_affected:
  - internal/domain/auth/service.go
  - internal/domain/auth/service_test.go
  - internal/domain/auth/service_internal_test.go
  - scripts/qa-go-prepush.sh
  - scripts/pattern-refactor-gate.sh
created: 2026-07-02
completed: 2026-07-02
---

# Task AGENTDEF-BOOTSTRAP-IMPL-A-001

**Plan**: [Support-agent definition bootstrap remediation](../plans/agent-definition-bootstrap-remediation-plan.md)

## Task Card

Task: AGENTDEF-BOOTSTRAP-IMPL-A-001

Task file: docs/tasks/task_agentdef_bootstrap_impl_a_auth.md

Plan file: docs/plans/agent-definition-bootstrap-remediation-plan.md

Summary: Extend `bootstrapWorkspaceDefaults` (`internal/domain/auth/service.go:193`) to insert a `support-agent` `agent_definition` row, using the values locked in `AGENTDEF-BOOTSTRAP-DESIGN-001`, inside the same transaction as role/pipeline bootstrap. This is a pure additive insert — it does not change any existing bootstrap behavior, only adds one more row to the same atomic boundary.

Code affected: `internal/domain/auth/service.go`, `internal/domain/auth/service_test.go`.

Effort/reasoning: Medio-alto - toca la transacción de registro (invariante de atomicidad ya establecida por ADR-032), pero es un insert aditivo sobre un patrón ya probado, sin cambiar comportamiento existente. RRI=52 (Med-high band).

Recommended model: claude-opus-4-8

Estimated tokens: ~5000

## High-Level Pseudocode

```
// internal/domain/auth/service.go

func bootstrapWorkspaceDefaults(ctx, q, p, now) error:
    create default role (existing, unchanged)
    assign role to user (existing, unchanged)
    for each pipeline seed (deal, case):        // existing, unchanged
        create pipeline + stage

    // NEW:
    agentDefID := uuid.NewV7().String()
    if err := q.CreateAgentDefinition(ctx, {
        ID: agentDefID,
        WorkspaceID: p.workspaceID,
        Name: "Support Agent",
        AgentType: "support",
        AllowedTools: <locked subset from AGENTDEF-BOOTSTRAP-DESIGN-001>,
        Limits: {"max_runs_day": 100, "max_cost_day_eur": 5},
        Objective: nil, TriggerConfig: {}, PolicySetID: nil,
    }); err != nil:
        return fmt.Errorf("create default support agent definition: %w", err)
    return nil
```

## Acceptance Criteria

1. `Register` persists a `support-agent` `agent_definition` row (per `AGENTDEF-BOOTSTRAP-DESIGN-001` locked values: bootstrap-generated UUID `id`, `name="Support Agent"`, `agent_type="support"`, `status="active"`, the locked `allowed_tools` subset, `limits={"max_runs_day":100,"max_cost_day_eur":5}`) for every freshly created workspace, inside the same transaction as role/pipeline bootstrap.
2. Duplicate-email `Register` failure still leaves no partial records (existing test must still pass with the new insert added to the transaction).
3. A bootstrap failure after transaction start still rolls back the agent-definition insert along with role/pipeline rows — add an explicit test that forces failure *after* the new insert point, not just before it.
4. `support-agent.allowed_tools` is a subset of the ADR-032 `workspace_owner` bootstrap tool grant (`internal/domain/auth/service.go:33`) — assert this in a test, not just by inspection.
5. No schema migration. No backfill of existing/legacy workspaces. No change to `internal/domain/agent/agents/support.go` (that is `AGENTDEF-BOOTSTRAP-IMPL-B2-001`, blocked on separate characterization-test work).

## Scope

- In: `bootstrapWorkspaceDefaults` extension and its direct test coverage.
- Out: any change to `internal/domain/agent/agents/support.go` or how `TriggerAgent` resolves an agent id — the row created here is inert until `AGENTDEF-BOOTSTRAP-IMPL-B2-001` lands, since `triggerSupportRun` still uses the old hardcoded literal id until that fix ships.
- Out: commits, pushes.

## Risks

- If the `allowed_tools` subset doesn't match what `SupportAgent` actually calls, the row created here is either over-permissioned or unusable once B2 wires it in — cross-check against `support.go`'s real tool usage during implementation, not just the design record.
- This task alone does not fix the support-agent trigger path (that's B2) — a workspace bootstrapped after only this task still can't trigger the agent via the old literal-id lookup. Do not report this task's closure as "support agent now works end-to-end"; it only becomes true after B2 also lands.

## Verification Strategy

- `go test ./internal/domain/auth/...`
- Full local QA gate before any push: `scripts/qa-go-prepush.sh`.

## Closure Notes

- Extended `bootstrapWorkspaceDefaults` with `createDefaultSupportAgent`, which
  inserts one `support-agent` `agent_definition` row inside the existing
  `Register` transaction, using the design-locked values: bootstrap-generated
  UUID `id`, `name="Support Agent"`, `agent_type="support"`,
  `allowed_tools=["update_case","send_reply","create_task"]`,
  `limits={"max_runs_day":100,"max_cost_day_eur":5}`.
- **Design-record correction (documented in `AGENTDEF-BOOTSTRAP-DESIGN-001`):**
  the code cross-check revealed the design draft had guessed the wrong tool
  list. `SupportAgent.AllowedTools()` actually declares
  `[update_case, send_reply, create_task, search_knowledge, get_case,
  get_contact]`, and only the first three are (a) registered enforced
  executors and (b) in the `workspace_owner` grant. Bootstrap uses those three,
  keeping the subset-of-owner invariant. User-confirmed decision.
- **Unforeseen finding (sqlc NOT NULL / NULL-scan):** the `CreateAgentDefinition`
  sqlc query names `allowed_tools`, `limits`, `trigger_config` explicitly, so
  the schema `DEFAULT` values are bypassed and a nil `json.RawMessage` violates
  `NOT NULL` on `trigger_config`. Additionally its `RETURNING` clause scans
  `objective`, and `json.RawMessage` cannot Scan a SQL NULL. Both resolved by
  passing `{}` for `objective` and `trigger_config`. This is a latent sharp
  edge in the generated query, worth noting for `AGENTDEF-BOOTSTRAP-IMPL-B2`
  and any future caller.
- **QA tooling fixes (in scope as "run the local QA gates"):** the prepush and
  pattern-refactor gates invoked `deadcode` and `golangci-lint` by bare name,
  assuming `$(go env GOPATH)/bin` is on `PATH`. When it is not (e.g. from the
  pre-push hook), the tools reported "not found" and that shell error was
  miscounted as a real finding — a false gate failure. Both scripts now resolve
  the binary explicitly (PATH → `$(go env GOPATH)/bin`) and fail loudly with an
  install hint only if the tool is genuinely absent.
- Full `scripts/qa-go-prepush.sh` passes end-to-end (exit 0): fmt-check,
  complexity, lint, wrapcheck, test, coverage-gate (83.3%), coverage-tdd
  (85.0%), deadcode (0 findings), race-stability, pattern-refactor-gate
  (0 findings). Traceability and govulncheck auto-SKIPPED (no doorstop venv /
  no go.mod change).
- **Scope reminder honored:** `support.go` was NOT touched. The bootstrapped row
  is inert until `AGENTDEF-BOOTSTRAP-IMPL-B2-001` switches the trigger path off
  the hardcoded literal id. This task alone does not make the support agent
  triggerable end-to-end.
