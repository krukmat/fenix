---
doc_type: task
id: DB-FENIX-B2.A3.T3
title: "B2.A3.T3 real Fenix SQLite and httptest S2S authorization verification"
status: branch_scoped_qa_pass_readiness_deferred
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
completed: 2026-11-09
---

# B2.A3.T3 real Fenix SQLite and httptest S2S authorization verification

**Plan**: [DubBridge to Fenix S2S security foundation](../plans/dubbridge_fenix_s2s_security_plan.md#current-direction).
**Coordination issue**: [B2.A3.T3 #7](https://github.com/krukmat/fenix/issues/7).
**Parent A3 issue**: [#5](https://github.com/krukmat/fenix/issues/5).
**Prior task**: [B2.A3.T2 mandatory source guard](task_dubbridge_fenix_b2a3_mandatory_scope_guard.md); implemented on the feature branch at `755fc79eabb7a02ef3208d8e9e59d41688bdd38a`, owner-accepted for its scoped functional behavior. Global QA debt remains separately tracked in [#4](https://github.com/krukmat/fenix/issues/4).
**Execution branch**: `feature/int-v2-dubbridge-consumes-fenix`; baseline `9c78515f556684741ab75f69ecb71a204f9a6a7e`.
**RRI**: N/A (cloud, DB-FENIX-INT). Independent task-readiness was owner-deferred *for this branch only*; the distinct code-review/Go gates were executed locally and reported PASS.
**Test source/QA**: Initial source `d938da5`, matrix expansion `4ea3be6`, format-only `367a70c` (final test SHA-256 `c0f75811b8a117cd166e131a424bc9928f8f392a2e9ce559f2bbf2f4b1bc723`). Local Go 1.25.13 tests/race/vet/new-code lint PASS and independent Claude post-code review PASS were reported with logs in [#7](https://github.com/krukmat/fenix/issues/7#issuecomment-6752968464). Published remote HEAD `acafa3d3b59f0539b7fbc5ec3114a43c332bce80` adds *separately scoped* repository QA cleanup, tracked in [#4](https://github.com/krukmat/fenix/issues/4). No route/migration/credential/deployment activation.
**Branch disposition**: T3 functional verification PASS with real temporary Fenix SQLite and isolated HTTP handlers; branch-scoped completion only. The independent task-readiness stage was owner-deferred, not PASS. No formal release/security certification.
**Peer readiness**: OWNER-DEFERRED for this feature branch (user instruction 2026-11-09). No reviewer executed, no PASS asserted, and no formal `WAIVER:` marker supplied. Must be restored for formal certification or a separately governed release. This is not an exception for credentials, production auth changes, migrations, or deployment.
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

## Verified execution record (local agent report, 2026-11-09)

- **SHA published**: `acafa3d3b59f0539b7fbc5ec3114a43c332bce80`. T3 test file was formatted and committed as `367a70c3f297a3d2ca5af6fd18e77e50dae98918`; the original test semantics were unchanged by formatting.
- **Go 1.25.13**: [#7 QA record](https://github.com/krukmat/fenix/issues/7#issuecomment-6752202211) reports T3 focused tests, `go test ./...`, scoped `-race -count=3`, vet and new-code lint all PASS.
- **Go pre-push**: [#7 final report](https://github.com/krukmat/fenix/issues/7#issuecomment-6752968464) reports RC 0, log SHA-256 `c91cd73ba86bece573dd292060b403842437321e5263afb15106b7d8ff5d6722`; traceability skipped (missing doorstop), govulncheck skipped (no module changes).
- **Independent post-code reviewer**: Claude PASS and local advisory PASS reported; artifact `/tmp/b2a3-t3-qa-ef69918/post-code-review.json` SHA-256 `814325e1805213cde44c2cce42b559014cf5ca9ada99f8920cd8304895b8b168`. ChatGPT Cloud has verified the GitHub source/commits, but not rerun these Mac-only tools.
- **Readiness**: owner-directed branch-only deferral, NOT a reviewer PASS or formal `WAIVER:`. No rollout authorization.
- **Extra work**: `43d935c` formatted 26 prior files; `acafa3d` refactored lint, error-wrapping and S2S registry complexity. These are not T3's original tests-only scope; maintain separate review/traceability in #4.

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
Peer readiness review approval: reviewer=OWNER_DEFERRED; artifact=NOT_RUN; status=DEFERRED_FEATURE_BRANCH_ONLY
