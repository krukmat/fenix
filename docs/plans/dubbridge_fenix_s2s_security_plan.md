---
doc_type: plan
id: DB-FENIX-S2S-SECURITY
title: "DubBridge to Fenix S2S security foundation"
status: active
date: 2026-11-06
tags: [dubbridge, fenix, s2s, security]
---

# DubBridge to Fenix S2S security foundation

## Current direction

B2.A3.T2 now has a mandatory Fenix-owned source guard on the feature branch at `755fc79eabb7a02ef3208d8e9e59d41688bdd38a`. The guard authorizes bounded POST source assertions against persisted Fenix identity before dispatch and denies GET/status until B2.C supplies a persisted principal/workspace/organization/project-scoped operation resolver. The S2S route remains unmounted. T2 code passed Go 1.25.13 scoped tests, race, vet, changed-code lint/complexity and independent post-code review. Repository-wide fmt, complexity and lint remain failing under [QA debt issue #4](https://github.com/krukmat/fenix/issues/4); no global QA PASS or rollout readiness is claimed. See [the T2 task](../tasks/task_dubbridge_fenix_b2a3_mandatory_scope_guard.md) and [issue #6](https://github.com/krukmat/fenix/issues/6) for the executable record.

B2.A2 repairs the existing Fenix credential store on the feature branch before a new independent security review. The SQLite schema is for local and future deployment review only; no production migration, live credential, route activation, GitHub Actions change, or merge to main is authorized. RRI is N/A under the DB-FENIX-INT cloud-agent exception.

B2.A3.T1 prepares the isolated middleware authorization contract and independent readiness review. The current middleware authenticates a service bearer but leaves organization/project authorization to a separate optional helper. Future implementation must place mandatory source authorization before every S2S handler and operation lookup, preserve default-deny, and obtain task-scoped auth HITL before code changes. This planning step does not activate a route.

## B2.A3 ownership and follow-up (2026-11-09)

**Owner decision:** ChatGPT Cloud is the primary author and coordinator for B2.A3 tasks. The local agent performs only independent reviewer commands, Go 1.25.13 QA and environmental checks unavailable to Cloud. The owner is consulted only for task-scoped security/migration/deployment HITL, not routine Go test authoring. GitHub remains the sole canonical artifact location.

| Task | Accountable | Execution split | State | Evidence |
|---|---|---|---|---|
| B2.A3.T1 readiness | ChatGPT Cloud | Cloud contract; local independent readiness | COMPLETE | [T1 record](../tasks/task_dubbridge_fenix_b2a3_middleware_readiness.md), [#5](https://github.com/krukmat/fenix/issues/5) |
| B2.A3.T2 compulsory source guard | ChatGPT Cloud | Cloud source design; local Go QA and independent post-code review | OWNER_ACCEPTED_SCOPED; repo-wide QA debt #4 | [T2 record](../tasks/task_dubbridge_fenix_b2a3_mandatory_scope_guard.md), [#6](https://github.com/krukmat/fenix/issues/6) |
| B2.A3.T3 real SQLite/httptest verification | ChatGPT Cloud | Cloud published real-store HTTP tests; local Go 1.25.13 QA deferred; no credential/route activation | TESTS_COMMITTED / OWNER_DEFERRED_READINESS / QA_PENDING | [T3 record](../tasks/task_dubbridge_fenix_b2a3_sqlite_http_integration.md), [#7](https://github.com/krukmat/fenix/issues/7) |
| B2.A3.T4 security evidence and B2.A4 handoff | ChatGPT Cloud | Cloud preliminary evidence inventory published; independent security review still needed | PRELIMINARY_AUDIT / BLOCKED_BY_T3_QA | [T4 record](../tasks/task_dubbridge_fenix_b2a3_security_handoff.md), [#8](https://github.com/krukmat/fenix/issues/8) |

**Sequence:** The owner temporarily deferred T3 independent readiness for this feature branch on 2026-11-09, enabling test authoring without a PASS claim. T3 real-store test source is committed at `d938da521eb9a429ffba62b7c12dc52163deda0c`, with additional negative matrix cases committed at `4ea3be66ecfd36f7c896214f2ac9710e8455447f`; Go QA and reviewer evidence are pending. [Preliminary T4 audit](../audit/dubbridge-fenix-b2a3-security-verification-2026-11-09.md) inventories gaps; T4 final security certification must still wait for T3 actual real-store evidence and independent review. B2.C owns durable operation lookup, so B2.A3 may certify GET/status **fail-closed with no resolver**, not production operation status. Neither T3 nor T4 authorizes a production route, credential, migration, capability, CI reactivation, deployment or main merge. Global QA debt remains [#4](https://github.com/krukmat/fenix/issues/4) without being mislabeled as PASS.

## Executable task

- [B2.A2 security remediation](../tasks/task_dubbridge_fenix_b2a2_security_remediation.md): fix H1 rotation, H2 revocation, M1 runtime SQLite integration and tests, and H3 issues attributable to B2.A2. Run Go 1.25.13 and repository QA, then request independent post-code review.
- [B2.A3.T1 middleware readiness](../tasks/task_dubbridge_fenix_b2a3_middleware_readiness.md): record the mandatory organization/project guard, negative authorization matrix, dependencies, and independent readiness verdict before any B2.A3 implementation.
- [B2.A3.T2 mandatory scope guard](../tasks/task_dubbridge_fenix_b2a3_mandatory_scope_guard.md): owner-accepted scoped implementation; independent code review and changed-code QA PASS, repo-wide debt separately tracked in #4. Do not imply rollout readiness.
- [B2.A3.T3 real SQLite/httptest](../tasks/task_dubbridge_fenix_b2a3_sqlite_http_integration.md): independent readiness and local Go QA required; isolated real credential lifecycle and negative S2S matrix; no production route.
- [B2.A3.T4 security audit/handoff](../tasks/task_dubbridge_fenix_b2a3_security_handoff.md): blocked by T3; reconcile exact evidence and prepare B2.A4, leaving B2.C and #4 independent.

## Acceptance

1. A revoked principal or credential is denied even when the revocation clock precedes creation or issue time.
2. Concurrent rotation leaves exactly one live credential per principal; failures leave the prior credential usable, and a successful rotation makes it unusable.
3. Embedded Fenix migrations create the constrained S2S schema on a fresh local database and support an isolated down/rollback check. Version, audit linkage, and empty default capability allowlist are represented without enabling a capability.
4. Tests use Fenix `NewDB` with modernc SQLite, exercise foreign keys, tenant scope, concurrent rotation, revocation, rollback, and secret redaction.
5. B2.A2 code passes its relevant format, complexity, lint, and tests. The full `scripts/qa-go-prepush.sh` result and unrelated baseline failures are reported precisely.
6. Independent readiness and code review gates are recorded. Deployment and activation remain separate decisions.
