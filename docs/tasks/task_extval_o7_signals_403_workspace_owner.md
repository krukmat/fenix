---
doc_type: task
id: EXTVAL-O7-SIGNALS-403-001
title: "Fix workspace_owner bootstrap permissions missing api/global scope, causing GET /api/v1/signals 403 for fresh workspace owners"
status: complete
phase: external-validation-rerun
week: "2026-W27"
tags: [external-validation, bugfix, auth, permissions, signals, o7]
fr_refs: [FR-060, FR-070, FR-071]
uc_refs: [UC-C1]
blocked_by: []
blocks: []
files_affected:
  - internal/domain/auth/service.go
  - internal/domain/auth/service_test.go
  - internal/domain/policy/evaluator_unit_test.go
criticality: critical
criticality_basis: "Anchor rubric P floor >= 4 for internal/domain/auth (ADR-017); this is a real permission-grant defect affecting the default first-user role."
created: 2026-07-05
completed: 2026-07-05
---

# Task EXTVAL-O7-SIGNALS-403-001

**Plan**: [External Validation Open Points Rerun Plan](../plans/external_validation_open_points_rerun_plan.md#o7-workspace-owner-get-apiv1signals-returns-403-on-fresh-workspace)

## Task Card

Task: EXTVAL-O7-SIGNALS-403-001

Task file: docs/tasks/task_extval_o7_signals_403_workspace_owner.md

Plan file: docs/plans/external_validation_open_points_rerun_plan.md

Summary: The `workspace_owner` role granted to the first user during `POST /auth/register` bootstrap omits the `api`/`global` permission namespace entirely. The policy engine's RBAC fallback (`roleAllowsAction`) requires one of `global:["*"|"admin"]`, `api:["signals.list"|"*"]`, `*:["signals.list"|"*"]`, or `api:["admin"]` (admin.* actions only) to authorize `GET /api/v1/signals`. None of these are present in `defaultWorkspaceOwnerPermissions`, so every freshly registered workspace owner is denied access to `signals`, `blackboard`, `eval`, `prompt`, `tool`, and `workflow` endpoints — the only handlers that call `checkActionAuthorization` with `resource="api"`. Other endpoints (approvals, cases, accounts, agents/runs, governance/summary) never enforce this check, which is why the defect was invisible until the T7 real-mode rerun hit `signals` specifically.

Root cause evidence (confirmed via code read, not speculation):
- `internal/domain/auth/service.go:29` — `defaultRoleName = "workspace_owner"`.
- `internal/domain/auth/service.go:34` — `defaultWorkspaceOwnerPermissions` JSON only has keys `records`, `agents`, `tools`. No `api` or `global` key.
- `internal/api/handlers/signal.go:54-57,83-86` — `List`/`Dismiss` call `checkActionAuthorization(w, r, h.authz, resourceAPI, "signals.list"|"signals.dismiss")`.
- `internal/domain/policy/evaluator.go:202-221` — `CheckActionPermission` falls back to `roleAllowsAction` when no policy set is active (the case for a fresh workspace).
- `internal/domain/policy/evaluator.go:491-496` — `roleAllowsAction` requires `hasGlobalAdminPermission` (checks `global:["*"|"admin"]`) OR `hasDirectResourcePermission` (`api:["signals.list"|"*"]`) OR `hasWildcardPermission` (`*:["signals.list"|"*"]`) OR `hasAPIAdminPermission` (`api:["admin"]`, but only for actions prefixed `admin.` — does not cover `signals.list`).
- Reproduced live in `EXTVAL-BATTERY-T7-RERUN-001`: fresh owner registered via `/auth/register`, `GET /api/v1/signals` → `403` at `2026-07-04T22:11:30Z` and again at `22:16:42Z` after full app navigation, while `approvals`/`cases`/`accounts`/`governance/summary` all returned `200` for the same token.

Code affected: `internal/domain/auth/service.go` (the `defaultWorkspaceOwnerPermissions` grant), plus its two existing unit test files if they assert on the exact permissions JSON.

Criticality: critical

Criticality basis: RRI anchor rubric floors `internal/domain/auth/**` at P=4 (ADR-017, auth/authorization-adjacent), and the auth_security penalty applies. This is a genuine access-control defect (a legitimate first-party owner is wrongly denied), not a hardening/edge-case gap.

Effort/reasoning: Medium — single-file grant change plus verifying it doesn't silently over-grant, but the anchor rubric forces the P/K/D floors up because the touched file is `internal/domain/auth/**`.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Estimated tokens: ~6000

## RRI

| Variable | Score | Evidence | Confidence |
|---|---|---|---|
| C cyclomatic | 0 | raw CC 1 (single map literal edit) -> score 0 | High |
| F files | 0 | 1 file touched (`internal/domain/auth/service.go`); test files only if existing assertions break | High |
| D domain | 3 | anchor rubric floor: `internal/domain/auth/**` (ADR-017) | High |
| T coverage | 1 | `service_test.go` and `service_internal_test.go` exist and likely assert on default role/permissions | Medium |
| A ambiguity | 0 | root cause fully diagnosed with file:line evidence; fix target is unambiguous | High |
| K coupling | 3 | anchor rubric floor: `internal/domain/auth/**` (ADR-017) | High |
| P impact | 4 | anchor rubric floor: `internal/domain/auth/**` (ADR-017) — permission/authorization data | High |
| X context | 2 | must read `policy/evaluator.go` RBAC fallback + `auth/service.go` bootstrap to pick correct grant shape | Medium |

**Base value:** 100 x (weighted / 5) = 30
**Penalties applied:** `auth_security` (+10 — anchor-rubric P floor >= 4, auth/audit/rights/secrets)
**Final RRI:** 40 -> band Moderate (26-40) -> Effort M / claude-sonnet-4-6 / thinking Off
**Gates for this band:** Present task card and wait for explicit approval. Confirm tests exist in the affected area (confirmed: yes).
**Criticality suggested:** yes — anchor-rubric P floor >= 4 (auth/audit/rights/secrets)

## System Context

```
POST /auth/register
      |
      v
AuthService.Register (internal/domain/auth/service.go)
      |
      v
bootstrapWorkspaceDefaults()
      |  creates Role{Name: "workspace_owner",
      |          Permissions: defaultWorkspaceOwnerPermissions}  <-- BUG: missing "api"/"global" key
      |  AssignRole(user, role)
      v
[later] GET /api/v1/signals  (any authenticated request)
      |
      v
SignalHandler.List (internal/api/handlers/signal.go)
      |
      v
checkActionAuthorization(w, r, h.authz, "api", "signals.list")
      |
      v
PolicyEngine.CheckActionPermission (internal/domain/policy/evaluator.go)
      |  no active policy set for fresh workspace -> falls back to:
      v
roleAllowsAction(rolePerms, "api", "signals.list")
      |  checks (in order): global:["*"|"admin"], api:["signals.list"|"*"],
      |                      *:["signals.list"|"*"], api:["admin"] (admin.* only)
      v
   all false -> 403 Forbidden
```

Upstream trigger: any `POST /auth/register` call (workspace bootstrap).
Downstream consumers: every handler that calls `checkActionAuthorization` with `resourceAPI` — currently `signal.go`, `blackboard_handlers.go`, `eval.go`, `prompt.go`, `tool.go`, `workflow.go`. Approvals/cases/accounts/agents/governance handlers do NOT call this check and are unaffected either way.

Key invariants for whoever picks this up:
- `roleAllowsAction` is pure/read-only over the permissions map — the grant format must match one of its four recognized shapes (`global:*|admin`, `api:<action>|*`, `*:<action>|*`, `api:admin` for `admin.*` actions only). Inventing a fifth shape here will silently do nothing.
- `hasAPIAdminPermission` only matches actions with an `admin.` prefix — granting `api:["admin"]` alone will NOT fix `signals.list`/`signals.dismiss` (they don't have that prefix). This is a easy trap: adding `"api":["admin"]` looks plausible but doesn't work for signals per the current code.
- The safest general fix — consistent with `workspace_owner` being described as "Default first-user role" (`service.go:32`) — is granting `"global": ["admin"]`, which satisfies `hasGlobalAdminPermission` and covers ALL `resource="api"` actions checked via `checkActionAuthorization`, not just `signals.*`. This avoids having to enumerate every current and future `api`-gated action individually.
- Do not remove or narrow the existing `records`/`agents`/`tools` grants — they are used elsewhere (verify via grep before changing shape).
- Existing tests (`service_test.go`, `service_internal_test.go`) may assert the literal JSON string or parsed map of `defaultWorkspaceOwnerPermissions` — must update assertions to match the new shape, not just make the fix compile.

## High-Level Pseudocode

```
# internal/domain/auth/service.go

const defaultWorkspaceOwnerPermissions = JSON({
  records: ["read_all"],
  agents:  ["execute"],
  tools:   [ ...existing tool list... ],
  global:  ["admin"],   # NEW: grants full api-resource authorization
                         # so workspace_owner passes checkActionAuthorization
                         # for signals.list, signals.dismiss, and any future
                         # resource="api" gated action, without needing a
                         # per-action enumeration.
})

# No control-flow changes needed elsewhere:
# roleAllowsAction() already checks hasGlobalAdminPermission() first,
# which reads perms["global"] for "*" or "admin".

# Test updates:
# - service_test.go / service_internal_test.go: update any assertion
#   on defaultWorkspaceOwnerPermissions (literal string or parsed map)
#   to include the "global":["admin"] key.
# - Add/extend a test that registers a fresh workspace owner and asserts
#   PolicyEngine.CheckActionPermission(ctx, userID, "api", "signals.list", nil)
#   now returns true (regression guard for this exact defect).
```

## Acceptance Criteria

1. `internal/domain/auth/service.go`'s `defaultWorkspaceOwnerPermissions` grants a scope that satisfies `roleAllowsAction` for `resource="api"`, any `action` (not just `admin.*`-prefixed ones).
2. A freshly registered workspace owner (via `POST /auth/register`) can call `GET /api/v1/signals?workspace_id=<own>` and receive `200`, not `403`.
3. The existing `records`/`agents`/`tools` grants for `workspace_owner` are preserved unchanged in content.
4. Existing unit tests in `internal/domain/auth/service_test.go` and `service_internal_test.go` pass, updated if they assert the exact permissions shape.
5. A new or extended test exists that specifically regresses the `signals.list` authorization path for a freshly bootstrapped `workspace_owner` role (via `PolicyEngine.CheckActionPermission` or an equivalent handler-level test), so this defect cannot silently reappear.
6. No other endpoint's authorization behavior changes as a side effect (approvals/cases/accounts/agents/governance were already ungated and remain so; `admin.*`-gated endpoints like blackboard/eval/prompt/tool/workflow should also now be reachable by `workspace_owner`, which is consistent with "owner" semantics — confirm this is acceptable, not a scope regression).

## Result (2026-07-05)

Fix applied: added `"global":["admin"]` to `defaultWorkspaceOwnerPermissions`
(`internal/domain/auth/service.go:34`), satisfying `hasGlobalAdminPermission`
in the policy engine's RBAC fallback for any `resource="api"` action.

1. **PASS** — `roleAllowsAction(perms, "api", "signals.list")` now returns
   `true` for the `workspace_owner` grant shape (verified with a unit test
   using the exact grant fixture).
2. **PASS** — Live verification against the real backend (restarted with the
   fix): registered a fresh owner via `POST /bff/auth/register`
   (`o7fix.verify.20260705@fenixcrm.test`), then called
   `GET /api/v1/signals?workspace_id=...` directly — backend log confirms
   `200 12B` (previously `403`).
3. **PASS** — `records`/`agents`/`tools` grant content unchanged; only the new
   `global` key was added.
4. **PASS** — `go test ./internal/domain/auth/... ./internal/domain/policy/...`
   — both packages pass.
5. **PASS** — Added `TestRoleAllowsAction_WorkspaceOwnerGrantsSignalsList`,
   `TestRoleAllowsAction_WorkspaceOwnerGrantsSignalsDismiss`, and
   `TestRoleAllowsAction_WithoutGlobalAdmin_DeniesSignalsList` in
   `internal/domain/policy/evaluator_unit_test.go` (pure function tests,
   mirroring the exact `defaultWorkspaceOwnerPermissions` shape) as a
   regression guard. Also extended `service_test.go`'s existing bootstrap
   assertion to check for the new `global:["admin"]` key.
6. **PASS** — No other handler's authorization path was touched; the fix only
   adds a permission key. `admin.*`-gated endpoints (blackboard/eval/prompt/
   tool/workflow) becoming reachable by `workspace_owner` is an intended
   consequence of "owner" semantics, not a scope regression.
