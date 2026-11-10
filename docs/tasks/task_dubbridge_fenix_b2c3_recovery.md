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
Recommended model: GPT-6 Cloud with independent reviewer.
Estimated tokens: ~5000
Pseudocode: Claim accepted once; do not regrant expired leases; mark ambiguous indeterminate without execution.
Peer readiness review approval: reviewer=PENDING; artifact=NOT_RUN; status=PENDING
