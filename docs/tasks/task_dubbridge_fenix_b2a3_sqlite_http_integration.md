---
doc_type: task
id: DB-FENIX-B2.A3.T3
title: "B2.A3.T3 real Fenix SQLite and httptest S2S authorization verification"
status: ready_for_independent_readiness_review
phase: B2.A3
week: 2026-W46
tags: [dubbridge, fenix, s2s, security, sqlite, integration-test]
fr_refs: []
uc_refs: []
blocked_by: [DB-FENIX-B2.A3.T2]
blocks: [DB-FENIX-B2.A3.T4]
files_affected:
  - internal/api/middleware/dubbridge_service_auth_integration_test.go
  - docs/tasks/task_dubbridge_fenix_b2a3_sqlite_http_integration.md
criticality: critical
criticality_basis: "Tests certify a cross-tenant authentication/authorization boundary using the real persisted Fenix service store; a misleading green test could conceal unauthorized dispatch. Tests-only scope; no production policy changes or migration."
created: 2026-11-09
completed: null
---

# B2.A3.T3 real Fenix SQLite and httptest S2S authorization verification

**Plan**: [DubBridge to Fenix S2S security foundation](../plans/dubbridge_fenix_s2s_security_plan.md#current-direction).
**Coordination issue**: [B2.A3.T3 #7](https://github.com/krukmat/fenix/issues/7).
**Parent A3 issue**: [#5](https://github.com/krukmat/fenix/issues/5).
**Prior task**: [B2.A3.T2 mandatory source guard](task_dubbridge_fenix_b2a3_mandatory_scope_guard.md); implemented on the feature branch at `755fc79eabb7a02ef3208d8e9e59d41688bdd38a`, owner-accepted for its scoped functional behavior. Global QA debt remains separately tracked in [#4](https://github.com/krukmat/fenix/issues/4).
**Execution branch**: `feature/int-v2-dubbridge-consumes-fenix`; baseline `9c78515f556684741ab75f69ecb71a204f9a6a7e`.
**RRI**: N/A (cloud, DB-FENIX-INT). Independent task readiness, owner HITL for any new security logic, post-code review, and scoped Go QA are still mandatory.
**Peer readiness**: PENDING genuine independent reviewer PASS and explicit concurrence with `criticality: critical`.
**HITL**: Owner has requested autonomous delivery of T3/T4 within the documented testing/verification scope. No authorization to alter the S2S auth boundary, write a new migration, activate routes, provision credentials, or deploy. Escalate any such need separately.

## Scope and responsibilities

**Owner of task**: ChatGPT Cloud. Design, tests, security matrix, documentation, remediation proposal and GitHub issue coordination are cloud responsibilities. Local agent is restricted to running Go 1.25.13, peer-review CLI and posting evidence where cloud execution is unavailable.

Implement *tests only* for real Fenix `sqlite.NewDB` + `sqlite.MigrateUp` + `auth.ServiceCredentialStore` + `middleware.DubBridgeServiceAuth`, using local temporary SQLite with an isolated `httptest` downstream spy. No test-only mock service principal at this boundary; obtain authenticated identity from the real stored principal and opaque credential digest. The middleware must remain **unmounted** in `internal/api/routes.go`.

### Test cases (no production writes)

1. Fresh DB with migration 038 and two Fenix workspaces: register real service principals with distinct workspace/org/project; issue two random opaque service credentials by digest. Confirm valid POST source reaches the spy once and produces `AuthorizedDubBridgeScope` with the verified principal/workspace/org/project.
2. Cross-tenant POST: wrong organization, wrong project, wrong product, missing source, org-only/project-bound mismatch, unknown/inactive principal, caller JSON/header/query/context claims and human JWT cannot create authorized S2S context. On every denied request assert zero downstream/capability/operation/run calls.
3. Real SQLite lifecycle: future-issued credential, expired credential, explicit credential revoke, principal revoke, successful rotation invalidating old token, failed rotation preserving existing credential, unavailable/closed DB. Verify denial precedes dispatch with no bearer disclosure.
4. Duplicate/malformed/noncanonical Authorization; unsupported method and content type; malformed, duplicate-key and trailing JSON; exact 64 KiB and +1 boundaries. Responses use safe stable error projection and no tenant enumeration.
5. GET/status with **real Fenix service credential and nil scoped operation resolver**: deny before handler. Exercise mismatched operation-ownership via the T2 in-memory resolver contract only if needed; do not claim an actual persisted scoped-operation lookup, as B2.C does not yet supply it.
6. For every rejection, query the **real** SQLite `agent_run` table count before/after (if schema presents it) and check no operation table is newly created; never fabricate B2.C tables or test-only successful operations. Verify no persisted service secret in DB or returned error text.
7. Verify service token is rejected by actual human auth middleware and human JWT is rejected by S2S middleware without setting a human context key. If genuine signed human JWT creation requires unrelated auth setup, preserve the explicit integration limitation and do not invent a JWT test.
8. Confirm router is unchanged, no endpoint is exposed, no diagnostic capability is automatically registered, and test cleanup closes the temporary DB.

### System Context

```text
[httptest request: opaque bearer + UNTRUSTED source/status ID]
                        |
                        v
[DubBridgeServiceAuth -> Fenix credential verification + mandatory guard]
           |                             |
           v                             v
[real modernc SQLite DB]          [denial before handler]
[workspace/principal/credential]            |
           |                           [zero writes/run]
           v
[authorized test spy only] -- cannot invoke runtime or capabilities

[human AuthMiddleware] <--- separate JWT negative boundary
[B2.C scoped operation resolver] --- NOT IMPLEMENTED / no fake persisted lookup
```

Invariants: only Fenix SQLite owns principal/workspace/org/project bindings; downstream spy has no write authority; migration and DB lifecycle belong to test fixture; operations, runs, credentials and routes remain inactive outside temporary tests; test errors redact secrets and tenant details.

### High-Level Pseudocode

```text
open temporary modernc SQLite using Fenix NewDB; MigrateUp
seed two workspaces; register principal and credential via real store methods
construct DubBridgeServiceAuth(real_store, nil_resolver, inert_spy)
for valid POST:
    assert one handler invocation with exact authorized Fenix scope
for each invalid auth/scope/source/transport/lifecycle case:
    call S2S wrapper through httptest request
    assert denial, no leaked bearer or tenant data, no spy invocation
    compare SQLite agent_run counts before and after
rotate and revoke real stored credentials; assert prior bearer denied
send valid GET/status with real bearer and nil B2.C resolver:
    assert deny without persisted operation lookup or handler dispatch
verify human JWT middleware boundary independently
close temp DB; report test matrix and logs
```

## Execution and closure gates

- Before writing Go tests, run `scripts/peer-workflow-review.py task-readiness` on this **exact** task file plus S2S plan, with an English task-card preview. Record reviewer, actual PASS/FAIL/BLOCKED, artifact SHA, criticality concurrence in the T3 coordination issue. Never reuse the T2 peer-review artifact.
- Do not change any production auth logic, `routes.go`, `internal/infra/sqlite/migrations/`, operation store or capability registry. If a test reveals a real production flaw, document it as a separate security finding requiring its own authorization.
- Go 1.25.13: focused integration tests, `go test ./...`, `go test -race -count=3 ./internal/api/middleware ./internal/domain/auth ./internal/infra/sqlite`, `go vet ./internal/api/middleware`, formatting/complexity and `golangci-lint --new-from-rev`; report independent repo-wide QA debt #4 without claiming global PASS.
- Obtain actual independent `post-code-review` PASS of the exact test diff before code closure or publication under repository push policy. No self-review or fabricated evidence.
- Report all test commands, actual SHA, artifact path and hash. Run no real credential issuance, production migration, route activation, CI reactivation, deployment or main merge.
- T3 is complete only after its scoped tests and peer review are validated. T4 security handoff may use T3 evidence, but may not present the missing B2.C durable operation resolver as complete.

## Task Card (requires peer readiness PASS before presentation)

Task: DB-FENIX-B2.A3.T3
Task file: docs/tasks/task_dubbridge_fenix_b2a3_sqlite_http_integration.md
Plan file: docs/plans/dubbridge_fenix_s2s_security_plan.md
Summary: Prove the existing mandatory S2S guard with a genuine Fenix SQLite credential store and isolated HTTP handlers; verify revocation, rotation, tenant isolation and zero durable side effects on denial.
Code affected: internal/api/middleware/dubbridge_service_auth_integration_test.go only.
Criticality: critical
Criticality basis: Real-store security-boundary assertions protect against false cross-tenant acceptance; only tests are in scope.
Effort/reasoning: High - Real SQLite fixture, secret lifecycle and negative security matrix.
Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6
Estimated tokens: ~9000
Peer readiness review approval: reviewer=PENDING; artifact=PENDING; status=BLOCKED_UNTIL_ACTUAL_REVIEW
