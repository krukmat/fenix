---
doc_type: task
id: DB-FENIX-B2C
title: "B2.C — durable S2S operation idempotency proposal"
status: c1_c3_branch_candidates_go_qa_pending
phase: integration
week: "2026-W45"
tags: [dubbridge, fenix, s2s, sqlite, idempotency]
fr_refs: []
uc_refs: []
blocked_by: [DB-FENIX-B2A, DB-FENIX-B2B]
blocks: [DB-FENIX-B2D]
files_affected:
  - docs/contracts/dubbridge-fenix-b2c-operation-proposal.up.sql
  - docs/contracts/dubbridge-fenix-b2c-operation-proposal.down.sql
  - scripts/int_b2c_operation_sqlite_smoke.py
  - internal/infra/sqlite/migrations/039_service_operation.up.sql
  - internal/infra/sqlite/migrations/039_service_operation.down.sql
  - internal/infra/sqlite/service_operation_migration_test.go
  - internal/domain/integration/service_operation_store.go
  - internal/domain/integration/service_operation_digest.go
  - internal/domain/integration/service_operation_recovery.go
criticality: critical
criticality_basis: "Cross-service idempotency, crash recovery and multi-tenant consistency"
created: 2026-11-05
completed:
---
# Task DB-FENIX-B2C

## Current branch checkpoint — 2026-11-10

**Coordination**: [#9](https://github.com/krukmat/fenix/issues/9). **Branch**: `feature/int-v2-dubbridge-consumes-fenix`. Owner approved C1–C4 task plan; no production migration, credential, route, CI reactivation, agent/model execution or main merge.

| Task | Accountable | Implementation | State |
|---|---|---|---|
| [C1 schema](task_dubbridge_fenix_b2c1_sqlite_schema.md) | ChatGPT Cloud | Embedded reversible migration **039** with active-Fenix-scope trigger, immutable fields and real SQLite Go test authored. Synthetic Python sqlite schema/negative/rollback verification PASS, not real `go test` | BRANCH_CANDIDATE / Go 1.25.13 QA + peer PENDING |
| [C2 store](task_dubbridge_fenix_b2c2_operation_store.md) | ChatGPT Cloud | Atomic `BEGIN IMMEDIATE`/INSERT SELECT from active principal, exact scope, typed replay/conflict, Fenix registry-owned config fingerprint, bounded duplicate-safe canonical JSON, concurrent tests authored | BRANCH_CANDIDATE / Go 1.25.13 QA + peer PENDING |
| [C3 recovery](task_dubbridge_fenix_b2c3_recovery.md) | ChatGPT Cloud | One-shot lease, explicit indeterminate on ambiguity/expiry, scoped lookup and no auto-dispatch; real SQLite concurrency/revocation tests authored | BRANCH_CANDIDATE / Go 1.25.13 QA + peer PENDING |
| [C4 status](task_dubbridge_fenix_b2c4_status_resolver.md) | ChatGPT Cloud | Adapter/HTTP test contract only. Real GET/status remains default-deny until B2.A3.T4 security verdict and C1–C3 QA | BLOCKED / NOT IMPLEMENTED |

**Security and evidence limits**: Published B2.C code is unvalidated by local Go/reviewer; branch migrations are not production migrations, and task-readiness has not been independently executed or waived for B2.C. Do not mark C1–C3 PASS or closed. The old docs/contracts SQLite schema remains historical proposal-only: source of truth for **candidate implementation** is embedded `039`. Production operation acceptance must also await B2.B4 operator capability policy, B2.C4 scoped status and B2.D transport/rate/response controls.

## Execution order — Cloud first, local gates last (owner direction 2026-11-10)

| Order | Accountable | Work | Output / gate |
|---|---|---|---|
| CF1 | ChatGPT Cloud | Review C1–C3 source, schema transitions, scope, immutable request identity, lease/revoke and tests; classify findings without claiming Go PASS | Source-backed static audit and bounded patch list |
| CF2 | ChatGPT Cloud | Correct source/test issues identified by CF1 when authorized by task-readiness governance; extend negative fixtures; do not enable routes or runtime | Published feature-branch code candidates and exact changed-file inventory |
| CF3 | ChatGPT Cloud | Finish C4 design, interface mapping and real-SQLite test matrix only; do not implement or wire C4 while security/QA dependency is open | Ready-to-execute C4 task, default-deny retained |
| CF4 | ChatGPT Cloud | Freeze candidate SHA, reconcile task/plan/#9 and produce a single exact validation packet for the local agent | Minimal local handoff |
| LG1 | Local agent | After CF1–CF4, run actual Go 1.25.13 fmt, focused tests, race, vet, full prepush and independent C1–C3 task-readiness/post-code-review | Command logs, SHA, real reviewer verdict and path list |
| LG2 | Independent local security reviewer | Verify B2.A3.T4 exact security diff in #8, independently of C1–C3 review | Real T4 security verdict/artifact |
| CF5 | ChatGPT Cloud | Remedy LG1/LG2 findings; repeat local gates only for affected scope | Updated candidate SHA and evidence reconciliation |
| CF6 | ChatGPT Cloud, conditionally | Implement unmounted C4 adapter/status projection **only after** C1–C3 QA and T4 security acceptance | C4 source candidate; no production route |
| LG3 | Local agent + reviewer | Run focused C4 tests/review on CF6 final SHA, with one necessary rerun after any patch | C4 certification packet; B2.C closure decision |

**Gate policy**: Sequence is about minimizing local agent implementation, not bypassing independent peer task-readiness, post-code review or owner HITL. If a mandatory readiness gate is not available, CF2 may only prepare a patch/design; it cannot claim a completed task or bypass the gate. Existing Go source is a candidate, not a validated baseline. RRI remains N/A for this cloud integration.

**Cloud static review notes, unexecuted in Go**:
- The service store receives a precomputed request digest; HTTP B2.D must enforce digest-to-approved-envelope binding before calling Reserve, rather than trusting caller-supplied digests.
- Current config fingerprint covers the registry binding tuple/handler, not future B2.B4 approved schema/policy/budget version. No dispatch until B2.B4.
- Migration 039's local rollback test executes down SQL directly and does not establish an operational migrate-down/reapply procedure or production recovery. No destructive rollout.
- C4 needs a thin adapter from `DubBridgeOperationResolver.LookupDubBridgeOperation` to scoped `ServiceOperationStore.Lookup` and a separate allowlisted status projection. It must not expose internal digests or mount a route.

**Next operational step**: complete CF1–CF4 and only then send LG1/LG2 with frozen SHAs and actual remaining findings.

## Historical checkpoint — 2026-11-05 (superseded by 2026-11-10 above)

PROPOSAL-ONLY operation table, rollback and offline SQLite idempotency selftest committed under docs/contracts/scripts. Replay, per-principal unique operation key, digest conflict, indeterminate crash state, negative constraints and rollback PASS in-memory. **No Go operation store, no embedded migration, no dispatcher or live HTTP endpoint has been implemented**. Independent security review and real Go/SQLite QA PENDING. Evidence: [B2 cloud audit](../audit/dubbridge-fenix-b2-cloud-checkpoint-2026-11-05.md).


**Plan**: [Canonical DubBridge → Fenix plan](../plans/dubbridge-consumes-fenix-integration.md)

## System Context

```text
Authorized DubBridge S2S principal -> authorized capability (B2.B) -> B2.C operation
                                 -> unique (principal_id, operation_id) + digest
                                 -> status + immutable true agent_run_id (future B6)
```

## Tasks and acceptance

- B2.C1: principal-scoped idempotency key, unique SQLite operation row, exact digest equality on replay; same key/different request gives conflict.
- B2.C2: run ID nullable during accepted state, required when running and on success. No fake workflow or agent runs.
- B2.C3: crash/lease uncertainty must enter indeterminate; no automatic duplication of side effects.
- B2.C4: complete real Go integration + testable deployment migration **only after** A2/A3 security, full QA and independent peer review.

At the historical 2026-11-05 checkpoint the schema lived only under docs/contracts. As of 2026-11-10, embedded migration 039 and Go source candidates exist on the feature branch, still without a live route or certified production reconciler.
Do not claim exact-once side effects or a real integrated E2E test.

**RRI: N/A** under scoped cloud-agent owner policy. No real workflows in B2.C.
