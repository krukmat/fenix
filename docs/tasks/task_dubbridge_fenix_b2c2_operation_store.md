---
doc_type: task
id: DB-FENIX-B2.C2
title: "B2.C2 atomic Go SQLite service operation reservation and canonical request fingerprint"
status: owner_approved_code_candidate_readiness_pending
phase: B2.C
week: 2026-W46
tags: [dubbridge, fenix, sqlite, idempotency, go]
fr_refs: []
uc_refs: []
blocked_by: [DB-FENIX-B2.C1, DB-FENIX-B2.B]
blocks: [DB-FENIX-B2.C3]
files_affected:
  - internal/domain/integration/service_operation_store.go
  - internal/domain/integration/service_operation_digest.go
  - internal/domain/integration/service_operation_store_test.go
criticality: critical
criticality_basis: "Concurrent idempotency and strict tenant binding avoid duplicate operation acceptance and cross-workspace reads."
created: 2026-11-10
completed: null
---

# B2.C2 — Atomic operation reservation and canonical digest

**Plan**: [Canonical DubBridge → Fenix](../plans/dubbridge-consumes-fenix-integration.md#blocktask-plan-two-levels-only).
**Parent**: [B2.C](task_dubbridge_fenix_b2c_operation.md).
**RRI**: N/A (DB-FENIX-INT Cloud).
**Approval**: Owner approved C1–C4 proposal in conversation 2026-11-10. No production credentials, gateway/route, deployment, or main merge authorized.
**Readiness**: Independent task-readiness PENDING; source is an unverified branch candidate until local Go 1.25.13 and reviewer gates.
**Accountable**: ChatGPT Cloud; local agent Go and independent peer gate only.

## System Context

```text
Verified Fenix principal + resolved capability (not caller JSON)
   -> operation request normalized (strict JSON; depth/bytes bound)
   -> SHA-256 request + immutable capability config SHA-256
   -> [BEGIN IMMEDIATE Fenix SQLite transaction]
      SELECT exact active Fenix principal/workspace/org/project
      INSERT SELECT Fenix-owned authority ON CONFLICT DO NOTHING
      SELECT same scope + compare digest/version/config fingerprint
      COMMIT
   -> NEW accepted / SAME replay / DIFFERENT conflict
   -> C3 recovery, C4 status-only lookup (not active)
```

## High-Level Pseudocode

```text
parse one strict JSON envelope with unique nested keys and bounded depth
normalize maps into sorted JSON, hash bytes; never persist raw input
validate request id, scope, trace, version and nonzero fingerprints
resolve enabled exact capability in registry; unknown means deny
begin immediate serialized SQLite transaction
insert authority SELECT from live Fenix service_principal
lookup operation with principal/workspace/org/project predicates and live principal
same identity/capability/config/request hash => idempotent replay
different hash/config/contract => typed conflict, no overwrite
commit; no agent execution, HTTP response or capability auto-registration
```

## Acceptance and limitations

- Registry starts empty; no operation accepted until explicit test-only/operator registration.
- Principal source scope is fetched from the real DB in INSERT SELECT, not caller JSON; DB trigger checks it.
- Concurrent same-key inserts create one row; different payload/config yields `ErrServiceOperationConflict` and unchanged existing record.
- Replays after principal revoke denied; cross-principal operation IDs independent.
- Canonical JSON excludes formatting and key order variation, rejects duplicate nested keys/trailing JSON and >64KiB or >32 levels. This is **not RFC 8785 numeric canonicalization**, so e.g. `1` and `1.0` intentionally have different fingerprints.
- No production S2S gateway/route, credential, migration execution or real workflow.
- Go 1.25.13 local tests/race/vet/lint and independent post-code review PENDING; never infer PASS from Cloud code publication.

## Task Card

Task: DB-FENIX-B2.C2
Task file: docs/tasks/task_dubbridge_fenix_b2c2_operation_store.md
Plan file: docs/plans/dubbridge-consumes-fenix-integration.md
Summary: Atomic exact-scope durable SQLite operation reservation; deterministic request digest and replay/conflict semantics.
Code affected: internal/domain/integration/service_operation_store.go, service_operation_digest.go, tests.
Criticality: critical
Criticality basis: Durable idempotency and tenant authorization before any dispatcher.
Effort/reasoning: High — concurrent writer, canonical input and failure paths.
Recommended model: GPT-6 Cloud; Claude independent reviewer.
Estimated tokens: ~9000
Pseudocode: Verify capability, derive Fenix principal scope, insert under transaction, compare fingerprints, return record/replay/conflict without running any agent.
Peer readiness review approval: reviewer=PENDING; artifact=NOT_RUN; status=PENDING
