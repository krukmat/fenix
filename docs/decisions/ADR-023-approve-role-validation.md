---
id: ADR-023
title: "APPROVE role validation: deferred to runtime, workspace-scoped, with abstention on unknown role"
date: 2026-04-23
status: accepted
deciders: [matias]
tags: [adr, dsl, security, governance, approval, runtime]
related_tasks: [CLSF-53, CLSF-54]
related_frs: [FR-060, FR-070, FR-071]
---

# ADR-023 — APPROVE role validation: deferred to runtime, workspace-scoped, with abstention on unknown role

## Status

`accepted`

## Context

The DSL v1 `APPROVE` statement (CLSF-53) introduces a `role` clause:

```
APPROVE send_email role manager
CALL send_email WITH draft
```

The `role` value (`manager`) is captured by the parser as a plain `IdentifierExpr` string.
At parse time, no validation is performed against any role registry — the parser has no
database access and no workspace context.

This raises a security question: **what happens if the role string does not correspond to
any real role in the workspace?**

Three failure modes are possible:

1. **Silent pass** — the approval is granted automatically (dangerous: bypasses governance)
2. **Hard error** — the workflow run fails with an execution error (disruptive)
3. **Abstention** — the workflow abstains and escalates to a human (safe, consistent with
   the existing `GROUNDS` abstention model)

Additionally, the scope of role resolution needs to be defined: are role names global
across all workspaces, or resolved per-workspace?

## Decision

**1. Role validation is deferred to runtime — not parse time, not judge time.**

The parser captures the role name as a string. The judge validates DSL/Carta structure
and permissions (PERMIT/DELEGATE), but does not have access to the role registry. Role
resolution happens when the runtime processes an `ApproveStatement` and creates an
`ApprovalRequest`.

**2. Role names are workspace-scoped.**

The `Role` table in SQLite is partitioned by `workspace_id`. A role named `manager` in
workspace A is a different entity from `manager` in workspace B. When the runtime
resolves `APPROVE ... role manager`, it queries:

```sql
SELECT id FROM roles WHERE workspace_id = ? AND name = ?
```

**3. Unknown role → abstention, not error.**

If the role name does not match any role in the workspace, the workflow **abstains** —
it does not execute the guarded action and escalates to a human operator. This is
consistent with the `GROUNDS` abstention model (CLSF-14, ADR-014) and avoids silent
privilege bypass.

The abstention payload must include:
- `reason: "approve_role_not_found"`
- `role: "<the unresolved role string>"`
- `stage: "<the stage name>"`
- `workflow_id`
- `workspace_id`

This makes the failure observable and auditable without crashing the run.

**4. A pending `ApprovalRequest` with an unresolvable role must not auto-approve.**

If a role is deleted from the workspace after an `ApprovalRequest` is created, any
pending approval for that role must not be auto-approved. It must remain blocked until
an administrator either re-creates the role, reassigns the approval, or cancels the run.

**5. Static analysis warning (future).**

A future linting pass (post Wave 5) may warn when a role string in `APPROVE` does not
match any role currently registered in the workspace, surfacing this as a
`textDocument/publishDiagnostics` warning (severity 2) via the LSP server. This is
advisory only — it does not block workflow activation.

## Security implications

| Risk | Mitigation |
|---|---|
| `APPROVE` with nonexistent role silently passes | Abstention on unknown role — guarded action is never executed |
| Role deleted after `ApprovalRequest` created | Pending approval remains blocked — no auto-approve |
| Role name collision across workspaces | workspace_id scoping — roles are always resolved within the tenant boundary |
| `APPROVE` bypassed by removing the role | Abstention + audit event — the attempted bypass is logged |

## Consequences

- The runtime must query the `roles` table before creating an `ApprovalRequest`.
- Abstention on unknown role must emit an `AuditEvent` with `reason: approve_role_not_found`.
- The conformance evaluator (CLSF-54) marks workflows containing `APPROVE` as `extended`
  profile — they require runtime approval infrastructure to execute safely.
- A future LSP diagnostic pass can surface unknown roles as warnings at authoring time.
- Role validation is **not** added to the judge or parser — keeping those layers stateless
  and database-free.

## Alternatives considered

**A. Validate role at parse time (rejected)**
The parser is stateless and has no database access. Injecting a role registry into the
parser would couple the language layer to infrastructure and make it impossible to parse
DSL source offline or in the LSP server without a database connection.

**B. Validate role in the judge (rejected)**
The judge runs Carta checks and DSL structural validation. Adding role resolution would
require passing workspace context into the judge, breaking its pure-function contract and
making it harder to test.

**C. Hard error on unknown role (rejected)**
A hard error would crash the workflow run and produce no useful audit trail. Abstention
produces an observable, recoverable state that a human operator can act on.

**D. Auto-approve on unknown role (rejected)**
This would silently bypass the approval governance contract and is a security violation.
