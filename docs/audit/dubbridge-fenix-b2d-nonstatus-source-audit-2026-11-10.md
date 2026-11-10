---
doc_type: audit
id: DB-FENIX-B2D-NONSTATUS-SOURCE-AUDIT-2026-11-10
title: "B2.D non-status Cloud contract consistency and gap review"
status: cloud_static_audit_completed_no_runtime_claim
date: 2026-11-10
tags: [dubbridge, fenix, s2s, contracts, security, cloud]
---

# B2.D non-status contract audit (Cloud source review)

**Reviewed implementation baseline:** `feature/int-v2-dubbridge-consumes-fenix` source at `e53fad7f54fba46af00d9f2b7e6e287b217479ad`; the plan/task documentation added after that SHA does not purport to change Go behavior. This is a **source/contract audit**, not Go execution, a security-signoff or deployment approval.

**Reviewed:** B1 neutral request/response JSON schemas, `internal/domain/integration/service_operation_store.go` and `service_operation_recovery.go`, and the new Cloud-owned [B2.D parent](../tasks/task_dubbridge_fenix_b2d_nonstatus.md) plus [D1](../tasks/task_dubbridge_fenix_b2d1_denial_contract.md), [D2](../tasks/task_dubbridge_fenix_b2d2_trace_audit.md), [D3](../tasks/task_dubbridge_fenix_b2d3_admission_preflight.md).

## Static source checks

A read-only source check evaluated **12/12 PASS** assertions: the neutral request required fields, UUID-format org ID, response mandatory operation ID and its error enumeration, presence of four typed C2 errors, persisted-principal reservation authority, trace ID replay equality, request digest input, conservative no-redispatch guarantee, D1–D3 task references, ingress-side digest recomputation specification and GET/status exclusion. **PASS means those source facts were found; it is not a test of their runtime enforcement.**

## Gap ledger, Cloud ownership

| ID | Source fact / mismatch | Disposition and next owner | Blocking scope |
|---|---|---|---|
| D-G1 | B1 lifecycle response requires `operation_id` and does **not** include `CAPABILITY_NOT_CONFIGURED` in `error_code`; no operation may exist for a rejected admission | **Addressed in D1 design:** keep pre-reservation denial distinct from operation lifecycle response; Cloud owns versioned proposal before future B2.E transport | Future ingress spec, not current B2.D design |
| D-G2 | C2 `Reserve` accepts a caller-supplied `RequestSHA256`; it does not parse or verify a full B1 envelope itself | **Addressed in D3 design:** future Fenix ingress validates B1 object and computes canonical digest locally from the approved envelope | Future B2.E admission implementation |
| D-G3 | B1 `source.org_id` is `format: uuid`; current SQLite/C2 storage validates persisted scope/identity but does not independently enforce a public JSON UUID-format requirement | **Addressed in D3 design:** enforce B1 schema validation at ingress and exact match against authenticated persisted authority; never infer safety from internal tests using opaque `org1` fixture values | Future B2.E schema validation |
| D-G4 | B1 proposed response may include `run_id`, `result`, terminal/error projection; C1–C3 are inert and cannot provide a production status/result implementation | **Deferred intentionally:** no status GET, no external status serialization or invented `run_id` in B2.D. Keep the response schema proposed and inactive | Later separately approved status scope, not B2.D |
| D-G5 | C3 `ReconcileExpiredLeases` can mark revoked principals' rows `indeterminate` for operator audit and denies caller lookup after revocation | **Addressed in D2 design:** audit reconciliation and public denied access are different surfaces. No automatic replay or state oracle | Later transport security tests |
| D-G6 | Caller-provided `trace_id` is validated and stored but cannot establish caller authorization or safe public exposure | **Addressed in D2 design:** trace is correlation only; approved public fields need redaction/authorization and internal correlation isolation | Future B2.E diagnostics |
| D-G7 | The latest reported C1–C3 QA PASS applied to an effective local tree; changes were later published at `e53fad7`, while the reviewer provider-route provenance was not established in the final issue comment | **Cloud reconciliation only:** distinguish QA PASS report from formal gate-policy closure; no additional local token consumption or fabricated reviewer acceptance | Formal C1–C3 closure only, not B2.D design |

## Scope and acceptance

**Completed Cloud responsibilities:** created B2.D three-task design; defined denial/error boundary, redaction/correlation invariants and admission/negative-case matrix; reconciled each with concrete source and schema facts in the ledger above. No unresolved contract mismatch stops continued Cloud planning.

**Not performed:** B2.D Go/HTTP implementation, full JSON schema runtime validation, security testing, independent reviewer execution, local Go/CI checks, credentials, deploy, production migration, GET/status, active capability or worker dispatch. No local agent execution is being requested; user token budget for local agents is exhausted.

C4 remains deferred. Security T4 [issue #8](https://github.com/krukmat/fenix/issues/8) remains a separate prerequisite for a live security boundary. Production rollout, route activation and human authorization are not implied.
