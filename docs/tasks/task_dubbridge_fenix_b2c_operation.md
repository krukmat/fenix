---
doc_type: task
id: DB-FENIX-B2C
title: "B2.C — durable S2S operation idempotency proposal"
status: c1_readiness_needs_changes_recheck_c1_c3_qa_pending
phase: integration
week: "2026-W45"
tags: [dubbridge, fenix, s2s, sqlite, idempotency]
fr_refs: []
uc_refs: []
traceability_scope: integration_infrastructure_no_product_fr_uc
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

### C2/C3 follow-up source work

C2 added a test that caller-forged configuration hashes do not influence registry-owned replay identity. C3 tightened the `MarkIndeterminate` UPDATE to atomically validate an active persisted principal against the exact operation scope and expanded revoked-principal regression coverage. These are source/test candidates only. Current change set includes seven Go/SQL/test paths; Go 1.25.13 QA/readiness/independent code review remain pending. C4 is still deferred.

**Regression completeness:** C2 now includes direct persisted-principal revoke tests against existing replay and new reservations; C3 adds a disabled-capability lease denial test. Neither has been executed with Go 1.25.13, and both still require independent peer review.\n\n**Additional C1/C2 source integrity:** migration 039 now forbids live lease metadata on terminal outcomes (14/14 isolated SQLite invariant cases PASS; real Go pending); canonical digest rejects malformed UTF-8 instead of silently replacing bytes. Go negative assertions were added. All changes remain source candidates, not integration PASS.\n\n## C1–C3 Cloud correction checkpoint — 2026-11-10

- **C1:** Migration 039 `run_id` invariants tightened (`accepted` must have NULL; required states cannot have an empty run ID); negative SQLite integration assertions authored.
- **C2:** Trace ID added to immutable reservation replay comparison; mismatch conflict test authored.
- **C3:** Real SQLite test authored for revoked principal + expired lease reconciling into `indeterminate` without restored access.
- Five source/test files changed; **no code was executed with Go 1.25.13 by Cloud**. Treat all as source candidates pending real fmt/Go/modernc/race/vet/prepush and independent readiness/code review; do not mark as PASS/COMPLETE. C4 remains deferred and non-blocking. Independent T4 security remains a separate issue.

## Active task plan — B2.C1–C3 (2026-11-10)

The current approved implementation scope is **C1 embedded SQLite 039**, **C2 atomic scoped reservation/idempotency**, and **C3 one-shot lease/ambiguous recovery**. These are the sole bounded deliverables of the current B2.C task. Dependency sequence: B2.A2 scoped persisted identity → C1 → C2 (plus B2.B default-deny registry) → C3 → independent readiness + Go 1.25.13 QA + exact-source code review → Cloud remediation and source-only closure record. See [canonical plan](../plans/dubbridge-consumes-fenix-integration.md#current-execution-baseline--b2c1c3-only-2026-11-10).

**Status:** C1 previous readiness `needs_changes` (documentary findings corrected, new verdict pending); C1–C3 Go and independent code review have not been completed. No reviewer model constitutes a dependency: independent-review procedures belong to [AGENT_WORKFLOW_GUIDE](../playbooks/AGENT_WORKFLOW_GUIDE.md), not this task plan.

**Out of scope of this source-only exit:** deferred C4, status GET/polling, B2.A3.T4 independent security/production approval, B2.B4 production operator policy, B2.D transport/rate/response activation, credentials, CI, rollout or merge. They remain separate workstreams, **not** requirements to certify the inert C1–C3 code. No historical QA debt is silently marked PASS.

## Current branch checkpoint — 2026-11-10

**Coordination**: [#9](https://github.com/krukmat/fenix/issues/9). **Branch**: `feature/int-v2-dubbridge-consumes-fenix`. Owner approved C1–C4 task plan; no production migration, credential, route, CI reactivation, agent/model execution or main merge.

**Current operator scope (2026-11-10): C1–C3 ONLY.** C4 status adapter is explicitly **DEFERRED**, not a blocker for verifying/closing C1–C3 or starting independently scoped follow-on planning. Keep GET/status default-deny. Do not assert that deferral authorizes live status, B4 status E2E or production rollout; these need a later C4 or separately approved equivalent.

| Task | Accountable | Implementation | State |
|---|---|---|---|
| [C1 schema](task_dubbridge_fenix_b2c1_sqlite_schema.md) | ChatGPT Cloud | Embedded reversible migration **039** with active-Fenix-scope trigger, immutable fields and real SQLite Go test authored. Synthetic Python sqlite schema/negative/rollback verification PASS, not real `go test` | BRANCH_CANDIDATE / Go 1.25.13 QA + peer PENDING |
| [C2 store](task_dubbridge_fenix_b2c2_operation_store.md) | ChatGPT Cloud | Atomic `BEGIN IMMEDIATE`/INSERT SELECT from active principal, exact scope, typed replay/conflict, Fenix registry-owned config fingerprint, bounded duplicate-safe canonical JSON, concurrent tests authored | BRANCH_CANDIDATE / Go 1.25.13 QA + peer PENDING |
| [C3 recovery](task_dubbridge_fenix_b2c3_recovery.md) | ChatGPT Cloud | One-shot lease, explicit indeterminate on ambiguity/expiry, scoped lookup and no auto-dispatch; real SQLite concurrency/revocation tests authored | BRANCH_CANDIDATE / Go 1.25.13 QA + peer PENDING |
| [C4 status](task_dubbridge_fenix_b2c4_status_resolver.md) | ChatGPT Cloud, later | Documentation only; GET/status remains default-deny | **DEFERRED / NON-BLOCKING FOR C1–C3** |

**Security and evidence limits**: Published B2.C code is unvalidated by local Go/reviewer; branch migrations are not production migrations, and task-readiness has not been independently executed or waived for B2.C. Do not mark C1–C3 PASS or closed. The old docs/contracts SQLite schema remains historical proposal-only: source of truth for **candidate implementation** is embedded `039`. Any later production status-enabled integration must still satisfy B2.B4 operator capability policy, C4 (or a separately accepted equivalent), B2.D transport/rate/response controls and rollout/security HITL. **These are not exit gates for the inert C1–C3 source-only block.**

## Readiness reconciliation and evidence handoff — 2026-11-10

At `6772905649d8390b4943273f4e52416696194087`, the owner reported C1 readiness `needs_changes` (B2.A2 prerequisite evidence not supplied and empty FR/UC lacking a rationale). No independent review artifact is available to Cloud. The [C1 task](task_dubbridge_fenix_b2c1_sqlite_schema.md) now carries [B2.A2 issue #3](https://github.com/krukmat/fenix/issues/3) owner-accepted scoped evidence: H1/H2/M1 Go/modernc/race x3 PASS **reported at `87fd10d`**, while independent B2.A2/T4 security acceptance is outstanding. The technical principal/migration 038 prerequisite exists on this feature branch; formal C1 readiness and security closure remain blocked on authentic reviewer outcomes. No result is transferred from B2.A2 into C1's separate Go tests.

**FR/UC note:** Empty `fr_refs` and `uc_refs` are intentional for this **capability-neutral integration foundation**. B2.C does not deliver a CRM business UC, configured agent or workflow. Requirements trace via [canonical B2.C plan](../plans/dubbridge-consumes-fenix-integration.md#blocktask-plan-two-levels-only), [S2S envelope](../contracts/dubbridge-fenix-v1-envelope-request.schema.json) and each C1–C4 task's acceptance criteria; no invented FR/UC IDs. Reviewers should evaluate this explicit scope rather than infer missing product traceability.

**Follow-up:** rerun C1 independent task-readiness on the revised task before certification; run actual B2.C Go 1.25.13 QA and independent post-code review only on frozen source; separately review T4 exact `43d935c..acafa3d`. See [local validation handoff](../handoffs/dubbridge-fenix-b2c-readiness-validation-2026-11-10.md). RRI: N/A — DB-FENIX-INT/cloud. No Go implementation, router, CI, migration activation, deployment or main merge.

## Execution order — Cloud first, local gates last (owner direction 2026-11-10)

| Order | Accountable | Work | Output / gate |
|---|---|---|---|
| CF1 | ChatGPT Cloud | Review C1–C3 source, schema transitions, scope, immutable request identity, lease/revoke and tests; classify findings without claiming Go PASS | Source-backed static audit and bounded patch list |
| CF2 | ChatGPT Cloud | Correct source/test issues identified by CF1 when authorized by task-readiness governance; extend negative fixtures; do not enable routes or runtime | Published feature-branch code candidates and exact changed-file inventory |
| CF3 | ChatGPT Cloud | Freeze C1–C3 candidate SHA and reconcile task/plan/#9 for local verification | Minimal local handoff; C4 excluded |
| LG1 | Local agent | Run C1–C3 Go 1.25.13 fmt/tests/race/vet/prepush, independent readiness and code review | Genuine verdict/logs; no C4 gate |
| LG2 | Independent security reviewer | Separately review B2.A3.T4 exact diff in #8 (remains pending) | Security evidence, not a prerequisite for inert C1–C3 source-only closure |
| CF4 | ChatGPT Cloud | Remedy actual C1–C3 findings and document bounded closure when their gates PASS | C1–C3 complete or still pending; C4 stays deferred |
| NEXT | ChatGPT Cloud | Prepare only non-status B2.D follow-on tasks, subject to independent gates | No GET/status handler or production integration |

**Gate policy**: Sequence is about minimizing local agent implementation, not bypassing independent peer task-readiness, post-code review or owner HITL. If a mandatory readiness gate is not available, CF2 may only prepare a patch/design; it cannot claim a completed task or bypass the gate. Existing Go source is a candidate, not a validated baseline. RRI remains N/A for this cloud integration.

**Cloud static review notes, unexecuted in Go**:
- The service store receives a precomputed request digest; HTTP B2.D must enforce digest-to-approved-envelope binding before calling Reserve, rather than trusting caller-supplied digests.
- Current config fingerprint covers the registry binding tuple/handler, not future B2.B4 approved schema/policy/budget version. No dispatch until B2.B4.
- Migration 039's local rollback test executes down SQL directly and does not establish an operational migrate-down/reapply procedure or production recovery. No destructive rollout.
- C4 is deferred by owner; its previously outlined adapter remains only future design reference. Do not spend current QA/review cycles on it.

**Next operational step**: rerun independent C1 readiness, then C1–C3 Go QA and exact-scope independent code review. T4 security review is tracked separately. C4 must not appear as a dependency in this local validation packet.

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
- B2.C4: **deferred at operator request**; it is not part of the present C1–C3 closure. If revived, separate security/QA/HITL are mandatory before any status functionality.

At the historical 2026-11-05 checkpoint the schema lived only under docs/contracts. As of 2026-11-10, embedded migration 039 and Go source candidates exist on the feature branch, still without a live route or certified production reconciler.
Do not claim exact-once side effects or a real integrated E2E test.

**RRI: N/A** under scoped cloud-agent owner policy. No real workflows in B2.C.
