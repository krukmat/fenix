---
doc_type: task
id: DB-FENIX-B2.C3
title: "B2.C3 lease fencing and indeterminate recovery without redispatch"
status: owner_approved_candidate_peer_readiness_pending
phase: B2.C
week: 2026-W46
tags: [dubbridge, fenix, sqlite, crash-recovery, idempotency]
fr_refs: []
uc_refs: []
traceability_scope: integration_infrastructure_no_product_fr_uc
blocked_by: [DB-FENIX-B2.C2]
blocks: [DB-FENIX-B2.C4]
files_affected:
  - internal/domain/integration/service_operation_recovery.go
  - internal/domain/integration/service_operation_recovery_test.go
criticality: critical
criticality_basis: "A lease lost during a crash must not silently repeat external side effects or allow a second worker to process the same operation."
created: 2026-11-10
completed: null
---

# B2.C3 — Lease and crash ambiguity without redispatch

**Plan**: [Canonical DubBridge → Fenix](../plans/dubbridge-consumes-fenix-integration.md#blocktask-plan-two-levels-only).
**Parent**: [B2.C operation task](task_dubbridge_fenix_b2c_operation.md).
**RRI**: N/A for cloud DB-FENIX-INT. Owner approved local C1–C4 design 2026-11-10.
**Peer readiness**: PENDING independent review; no T3 waiver or PASS applies here.
**Owner**: ChatGPT Cloud for code, tests, docs; local agent for true Go/race/reviewer gates.

## Requirements traceability and security dependencies (2026-11-10)

**FR/UC rationale:** `fr_refs: []` / `uc_refs: []` reflect an internal S2S lease and crash-ambiguity primitive, not a specific CRM agent workflow or product scenario. Trace instead to the [canonical B2.C integration plan](../plans/dubbridge-consumes-fenix-integration.md#blocktask-plan-two-levels-only), [parent B2.C task](task_dubbridge_fenix_b2c_operation.md), [C2 reservation](task_dubbridge_fenix_b2c2_operation_store.md) and this task's one-shot lease/no-redispatch acceptance. Do not attribute agent execution or `UC-G1` to this inert recovery code.

**Dependencies:** B2.A2 scoped principal source was accepted by the owner in [#3](https://github.com/krukmat/fenix/issues/3), not independently certified for release; C1/C2 are unverified Go candidates. No C3 readiness, race or independent post-code PASS has been supplied. **RRI: N/A — DB-FENIX-INT/cloud**.

## System Context

```text
Verified S2S + enabled scoped capability
   -> C2 durable accepted reservation (no run)
   -> C3 AcquireLease (only once, no retry on expired)
       |                 |
   no effects yet    worker interruption / unknown effect
       |                 v
       |            C3 ReconcileExpired -> indeterminate
       |                               -> NEVER redispatch
       v
     B6 future actual governed runtime (not part of this task)
        C4 scoped GET/status sees durable state
```

## High-Level Pseudocode

```text
for known accepted operation and live scoped Fenix principal:
  atomically claim only if no lease exists
  save bounded worker ID and lease expiry, never create a synthetic run
if lease expiry occurs:
  operator-only sweep marks accepted leased row indeterminate
  clear lease; never move it back to accepted automatically
for status lookup:
  match principal/workspace/org/project plus active persisted principal
  return metadata only, not raw inputs, prompts or model outputs
no background goroutine, worker launch, real execution or active route
```

**Additional capability-denial regression:** `TestB2CLeaseRejectsDisabledCapability` creates an accepted operation, disables its registry capability and confirms no worker can acquire a lease and no lease metadata changes. This is a source candidate with **Go 1.25.13 tests and independent peer review pending**; it does not add a runtime dispatcher or operator endpoint.

## C3 follow-up: revocation/update atomicity — 2026-11-10

**Implementation candidate:** `MarkIndeterminate` now repeats the persisted active Fenix principal/consumer/workspace/org/project check **inside the conditional `UPDATE`**, in addition to the existing `Lookup` check. This narrows the time-of-check/time-of-use window if a principal is revoked after lookup but before its leasing worker records ambiguity. An explicit Fenix-owned `ReconcileExpiredLeases` remains able to mark expired operations of revoked principals `indeterminate` for audit, without restoring caller access.

**Regression:** the revoked-principal/expired-lease test now first asserts the scoped worker API rejects `MarkIndeterminate`, then checks that operator reconciliation still succeeds and creates no `agent_run`. This is not a status resolver, dispatcher or route.

**Verification limits:** a standalone in-memory SQLite reproduction of the *conditional predicate* passed for active principal (row updated) and revoked principal (zero rows). It did **not** execute Fenix's real Go/modernc suite, full migration or independent review; those remain `NOT_RUN`.

## Cloud C3 bounded regression coverage — 2026-11-10

**Candidate change**: `TestB2CRevokedPrincipalExpiredLeaseReconciles` now covers an accepted operation whose one-shot lease has been acquired, whose owning principal is subsequently revoked and whose lease expires. The test expects a Fenix-owned reconciliation to mark the row `indeterminate`, clear lease metadata and **not** restore principal lookup permission or create a real run.

**Reason**: `ReconcileExpiredLeases` is an explicit audit/recovery primitive that may finalize an already-leased row after credential or principal revocation. Without this regression case the negative lookup test did not establish conservative crash cleanup after revocation. No production implementation change was required for C3.

**Verification**: test candidate committed; real Go 1.25.13 modernc/race and independent review **NOT RUN**. Status and code remain source-candidate, not certified.

## Acceptance

- Only one lease acquisition from a fresh accepted operation; concurrent attempts yield one winner.
- Expired lease is **not** newly acquired; reconciliation sets `indeterminate`.
- No automatic retry, fake `agent_run`/run_id, tool, provider or business workflow execution.
- Wrong workspace/org/project and revoked principal fail closed.
- Local Go 1.25.13 SQLite tests, race, security peer review required before closure.
- No production route, credentials, migrations or deployment activated.

## Task Card

Task: DB-FENIX-B2.C3
Task file: docs/tasks/task_dubbridge_fenix_b2c3_recovery.md
Plan file: docs/plans/dubbridge-consumes-fenix-integration.md
Summary: Provide single claim, scoped read and indeterminate crash recovery for durable S2S operations.
Code affected: internal/domain/integration/service_operation_recovery.go and tests only.
Criticality: critical
Criticality basis: Ambiguous external effects cannot be safely retried by default.
Effort/reasoning: High — atomic lease and recovery state machine.
Recommended model: Cloud author; independent peer review assigned by docs/playbooks/AGENT_WORKFLOW_GUIDE.md (no model-specific integration dependency).
Estimated tokens: ~5000
Pseudocode: Claim accepted once; do not regrant expired leases; mark ambiguous indeterminate without execution.
Peer readiness review approval: reviewer=PENDING; artifact=NOT_RUN; status=PENDING
