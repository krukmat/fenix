---
doc_type: audit
id: DB-FENIX-B2.A3-T4-PRELIMINARY-AUDIT
title: "B2.A3 security evidence ledger and B2.A4 preliminary handoff"
status: branch_evidence_reconciled_t4_independent_security_pending
date: 2026-11-09
tags: [dubbridge, fenix, s2s, security, evidence, handoff]
---

# B2.A3 security verification — preliminary evidence inventory

**Plan**: [Fenix S2S security foundation](../plans/dubbridge_fenix_s2s_security_plan.md#current-direction).
**Tasks**: [T1](../tasks/task_dubbridge_fenix_b2a3_middleware_readiness.md), [T2](../tasks/task_dubbridge_fenix_b2a3_mandatory_scope_guard.md), [T3](../tasks/task_dubbridge_fenix_b2a3_sqlite_http_integration.md), [T4](../tasks/task_dubbridge_fenix_b2a3_security_handoff.md).
**Issues**: [#5](https://github.com/krukmat/fenix/issues/5), [#6](https://github.com/krukmat/fenix/issues/6), [#7](https://github.com/krukmat/fenix/issues/7), [#8](https://github.com/krukmat/fenix/issues/8), [QA debt #4](https://github.com/krukmat/fenix/issues/4).
**Scope**: feature branch `feature/int-v2-dubbridge-consumes-fenix`; **NOT a security certification, production acceptance, or deployment authorization**.
**RRI**: N/A (cloud DB-FENIX-INT). Independent security evaluation remains a separate gate.

## Evidence matrix

| Security claim | Source / evidence | Status | Gap / owner |
|---|---|---|---|
| Dedicated opaque S2S credential, not human JWT | Real `ServiceCredentialStore` and T3 integration test | **Go 1.25.13 PASS reported** | No production credential issued; rollout validation separate |
| Live persisted principal/workspace/org/project binding | T2 `755fc79e...`; real SQLite HTTP integration test `367a70c...` | **T3 test PASS reported** | Tenant identities only in disposable DB; formal security review still T4 |
| Every S2S POST passes mandatory guard before handler | `DubBridgeServiceAuth`; T2 exact-diff Claude PASS; T3 real-store HTTP spy | **PASS reported** | No production S2S endpoint mounted |
| GET/status cannot use an operation ID as proof of tenant scope | `dubbridge_source_guard.go`: missing B2.C resolver denies before handler | Source verified as default-deny | Actual B2.C persisted scoped lookup is **NOT IMPLEMENTED**; B2.C owns it |
| Cross-tenant mismatch and revoked/expired credentials deny | T3 tests at `367a70c` (base `d938da5`, extra cases `4ea3be6`); [QA log](https://github.com/krukmat/fenix/issues/7#issuecomment-6752202211) | **PASS reported**, Go 1.25.13, race | Specific test verdict for all four `TestB2A3T3*` suites |
| No `agent_run` write on rejected HTTP calls | T3 real SQLite `agent_run` count before/after and downstream spy | **PASS reported** | No B2.C operation table exists, so operation-row assertions still deferred |
| Human JWT and service token rejection across auth boundaries | T3 real-store test-only signed JWT versus opaque bearer | **PASS reported** | No deployed ingress tests; production routing not activated |
| Maximum POST size 64 KiB; sanitized errors | T2 and T3 64 KiB/64 KiB+1, malformed JSON/source and safe denial responses | **PASS reported** | Response 32 KiB, depth, and principal rate limits remain A4 |
| No S2S production mount, capability enablement or deployment | `internal/api/routes.go` unchanged by T2; no T3 production source touched | Static Git diff / route scope verified | Continue to preserve in T4/B5; no production inference |
| Full repository QA | [#7 final QA](https://github.com/krukmat/fenix/issues/7#issuecomment-6752968464), prepush log SHA-256 `c91cd73...` | **pre-push script RC 0 reported** after 26-file gofmt and cross-scope remediations | Traceability skipped (doorstop absent); govulncheck skipped (no module changes); security review of extra changes not independently inspected by Cloud |
| T3 independent task readiness | Owner instruction to omit temporarily **for this branch**, 2026-11-09; [#7](https://github.com/krukmat/fenix/issues/7) | **DEFERRED, NOT PASS** | Restore before formal certification or separately govern exception |
| T3 independent post-code review; T4 independent security review | [#7 final QA](https://github.com/krukmat/fenix/issues/7#issuecomment-6752968464) reports Claude PASS, advisor PASS; local artifact SHA-256 `814325e...` | T3 **PASS reported**; T4 independent assessment **NOT RUN** | Exact reviewer payload remains local; formal T4 security sign-off pending |

## Current disposition (2026-11-09, HEAD `acafa3d3b59f0539b7fbc5ec3114a43c332bce80`)

T1 readiness and T2's mandatory guard previously passed their independent gates; T2 is owner-accepted on the feature branch. T3 real Fenix SQLite + HTTP negative tests now have **local Go 1.25.13 PASS** for focused and full tests, race, vet, formatting and changed-code lint, plus a reported primary Claude `post-code-review` PASS. See [T3 local QA](https://github.com/krukmat/fenix/issues/7#issuecomment-6752202211) and [published SHA / prepush evidence](https://github.com/krukmat/fenix/issues/7#issuecomment-6752968464). The code was published after passing `scripts/qa-go-prepush.sh` (RC 0, log SHA-256 `c91cd73ba86bece573dd292060b403842437321e5263afb15106b7d8ff5d6722`). This is **reported evidence from the local runner**; Cloud independently confirmed the remote SHA and files, but did not rerun Go.

**Important scope expansion:** `367a70c` formats T3 tests only; `43d935c` mechanically formats 26 pre-existing Go files; `acafa3d` repairs cross-cutting complexity, lint and error wrapping in authentication, S2S registry, evidence and provider integrations, and records a separate QA remediation task. These extra changes are **outside the original T3 tests-only scope**. The original global QA blockers are no longer present in the executed prepush script, but the broader code change needs its own assurance before production security acceptance. [QA debt #4](https://github.com/krukmat/fenix/issues/4) preserves skipped traceability / vulnerability review and historical issues.

The owner instructed a temporary **deferral of T3 independent task-readiness for this feature branch**, NOT a PASS and NOT a formal `WAIVER:` statement. This does **not** extend to deployment, credential lifecycle, schema migrations, or a production route.

**T3 may be closed as feature-branch scoped QA verified. T4 evidence reconciliation and A4 planning can proceed, but T4 independent security certification remains pending**; neither the B2.C stored-operation resolver nor a live S2S route exists.

## B2.A4 handoff prerequisites

| Workstream | Needed evidence / action | Primary owner |
|---|---|---|
| Operator-controlled principal lifecycle | Register/rotate/revoke with audited trace, digest-only storage, no bearer in logs; explicit HITL on live credentials | ChatGPT design; owner approves; local operator executes |
| Persisted operation read/status | B2.C real Go store and principal + workspace + org + project scoped lookup; no ID-only GET | B2.C task owner, separate approval |
| Gateway and rate limiting | Principal-bound concurrency/rate limits, trusted-proxy model, 32 KiB response cap and bounded JSON depth | A4 design/implementation; separate security approval |
| SQLite evidence and recovery | T3 real modernc integration tests PASS reported; migration/restore review and failure injection still required before rollout | T3 verified on feature branch; A4 rollout |
| Independent security acceptance | T3 independent code-review PASS reported; T4 separate security reviewer and owner release disposition still required | Independent reviewer; owner |
| Repository quality | Prepush Go PASS after explicit cross-cutting fixes; traceability and govulncheck skipped as recorded; separate production impact review of non-T3 code | [#4](https://github.com/krukmat/fenix/issues/4) |
| Rollout | Disabled-by-default launch, no capability auto-registration; only future separately approved deployment | B5/owner |

## Invariants preserved

- Fenix, not caller JSON, owns S2S identity and scope.
- Denied requests must cause zero handler, capability, operation, and agent execution.
- A GET/status without a real scoped persisted B2.C resolver remains denied.
- No real credentials, production migrations, new routes, capability registration, GitHub Actions reactivation, deployment, or main merge are authorized by this audit.
