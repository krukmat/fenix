---
doc_type: audit
id: DB-FENIX-B2.A3-T4-PRELIMINARY-AUDIT
title: "B2.A3 security evidence ledger and B2.A4 preliminary handoff"
status: preliminary_blocked_by_t3_qa
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
| Dedicated opaque S2S credential, not human JWT | `internal/domain/auth/service_token.go`, real `ServiceCredentialStore`; B2.A2 tests | Implemented earlier; T3 HTTP link not executed | T3 full real-DB QA pending |
| Live persisted principal/workspace/org/project binding | `internal/domain/auth/service_principal_store.go`; T2 guard source `755fc79eabb7a02ef3208d8e9e59d41688bdd38a` | Source inspected; T2 scoped QA/reviewer PASS reported | T3 real SQLite/HTTP tests unexecuted |
| Every S2S POST passes mandatory guard before handler | `internal/api/middleware/dubbridge_service_auth.go`; T2 Claude exact-diff review PASS | Implemented on feature branch | T3 real-store test verdict pending |
| GET/status cannot use an operation ID as proof of tenant scope | `dubbridge_source_guard.go`: missing B2.C resolver denies before handler | Source verified as default-deny | Actual B2.C persisted scoped lookup is **NOT IMPLEMENTED**; B2.C owns it |
| Cross-tenant mismatch and revoked/expired credentials deny | T2 in-memory negative tests; T3 new real SQLite/HTTP test candidate `d938da521eb9a429ffba62b7c12dc52163deda0c` | Test source committed; **not run** | T3 Go 1.25.13 test/race/vet/lint and reviewer pending |
| No `agent_run` write on rejected HTTP calls | T3 candidate counts real `agent_run` rows before/after each assertion | Test authored, not executed | T3 actual SQLite result required; no B2.C operation table is created |
| Human JWT and service token rejection across auth boundaries | Separate `AuthMiddleware` and `DubBridgeServiceAuth`; T3 candidate creates test-only signed JWT | Test authored, not executed | T3 actual verdict pending |
| Maximum POST size 64 KiB; sanitized errors | T2 scoped tests reported PASS; T3 candidate includes size and projection assertions | Test authored, not executed | T3 execution pending; response 32 KiB/depth/rate-limit concerns remain A4 |
| No S2S production mount, capability enablement or deployment | `internal/api/routes.go` unchanged by T2; no T3 production source touched | Static Git diff / route scope verified | Continue to preserve in T4/B5; no production inference |
| Full repository QA | [#4](https://github.com/krukmat/fenix/issues/4), T2 issue #6 | Known global FAILs | Separate workstream; do not mislabel PASS |
| T3 independent task readiness | Owner instruction to omit temporarily **for this branch**, 2026-11-09; [#7](https://github.com/krukmat/fenix/issues/7) | **DEFERRED, NOT PASS** | Restore before formal certification or separately govern exception |
| T3 independent post-code review; T4 independent security review | No completed artifacts for T3 or T4 | **NOT RUN** | Genuine independent reviewers still required for formal closure |

## Current disposition

T1 readiness was independently reviewed. T2 was **owner-accepted for its bounded feature-branch implementation** after a genuine Claude post-code review and scoped Go QA; unrelated global QA debt remains #4. T3 source is published to GitHub as tests-only, but this environment has Go 1.23.2 and no full Fenix checkout; no Go 1.25.13 or actual SQLite execution occurred here. The owner directed temporary **deferral of independent task readiness for this branch**. Do not call the absent gate PASS, imply a formal `WAIVER:` statement, or treat test-source existence as a passing integration test.

**B2.A3.T4 remains BLOCKED** for final security sign-off. The present record is an evidence inventory, not a T4 independent reviewer finding or signed report.

## B2.A4 handoff prerequisites

| Workstream | Needed evidence / action | Primary owner |
|---|---|---|
| Operator-controlled principal lifecycle | Register/rotate/revoke with audited trace, digest-only storage, no bearer in logs; explicit HITL on live credentials | ChatGPT design; owner approves; local operator executes |
| Persisted operation read/status | B2.C real Go store and principal + workspace + org + project scoped lookup; no ID-only GET | B2.C task owner, separate approval |
| Gateway and rate limiting | Principal-bound concurrency/rate limits, trusted-proxy model, 32 KiB response cap and bounded JSON depth | A4 design/implementation; separate security approval |
| SQLite evidence and recovery | Real ephemeral modernc integration test PASS, migration/restore review, failure-injection and no side effects on denial | T3/local Go validation; A4 rollout |
| Independent security acceptance | True T3 peer code review; T4 security reviewer and owner disposition | Independent reviewer; owner |
| Repository quality | Existing global fmt/complexity/lint/wrapcheck/coverage debt tracked without misattribution | [#4](https://github.com/krukmat/fenix/issues/4) |
| Rollout | Disabled-by-default launch, no capability auto-registration; only future separately approved deployment | B5/owner |

## Invariants preserved

- Fenix, not caller JSON, owns S2S identity and scope.
- Denied requests must cause zero handler, capability, operation, and agent execution.
- A GET/status without a real scoped persisted B2.C resolver remains denied.
- No real credentials, production migrations, new routes, capability registration, GitHub Actions reactivation, deployment, or main merge are authorized by this audit.
