---
doc_type: task
id: DB-FENIX-B2.C4
title: "B2.C4 scoped operation GET/status resolver and isolated HTTP integration"
status: planned_blocked_by_b2a3_t4_security_gate_and_c1_c3_qa
phase: B2.C
week: 2026-W46
tags: [dubbridge, fenix, s2s, sqlite, status, authorization]
fr_refs: []
uc_refs: []
blocked_by: [DB-FENIX-B2.A3.T4, DB-FENIX-B2.C1, DB-FENIX-B2.C2, DB-FENIX-B2.C3]
blocks: [DB-FENIX-B2.D]
files_affected:
  - internal/api/middleware/dubbridge_operation_resolver.go
  - internal/api/middleware/dubbridge_operation_resolver_test.go
criticality: critical
criticality_basis: "Cross-tenant status lookup can reveal private operation existence unless principal/workspace/org/project are independently verified."
created: 2026-11-10
completed: null
---

# B2.C4 — Fenix persisted S2S status lookup (not mounted)

**Plan**: [DubBridge → Fenix canonical integration](../plans/dubbridge-consumes-fenix-integration.md#blocktask-plan-two-levels-only).
**Parent**: [B2.C operation task](task_dubbridge_fenix_b2c_operation.md).
**RRI**: N/A cloud DB-FENIX-INT.
**Owner approval**: C1–C4 plan approved 2026-11-10; no public route, credential, production migration or deployment authorized.
**Precondition**: [B2.A3.T4 independent security assessment](task_dubbridge_fenix_b2a3_security_handoff.md) + real C1–C3 Go tests/review. Until then GET/status must remain default-deny.
**Owner**: ChatGPT Cloud; local agent only Go/race/reviewer CLI.

## System Context

```text
DubBridge GET status -> isolated DubBridgeServiceAuth (actual opaque bearer)
     -> VerifiedDubBridgeIdentity / AuthorizedDubBridgeScope
     -> B2.C4 resolver (no trusting operation_id or URL alone)
     -> B2.C2 ServiceOperationStore.Lookup
     -> active service_principal scoped to principal/workspace/org/project
     -> sanitized output [status, operation_id, allowed trace/run reference]
     X no raw input, digest, model prompt, tools or arbitrary run DTO
```

## High-Level Pseudocode

```text
receive only verified Fenix scope, validate bounded operation ID
load operation using all independent principal/workspace/org/project predicates
revalidate active persisted principal mapping and deny revoked/foreign lookup
return unavailable/forbidden without leaking operation existence
project stable allowlisted status fields, never raw request/config fingerprints
test with real SQLite + httptest two principals, revoked credential,
cross-workspace and same-ID collisions, invalid ID, unavailable store
do NOT mount any route before independent T4 security verdict and separate HITL
```

## Cloud-first implementation blueprint (design only, 2026-11-10)

Verified current source interfaces:
- `middleware.DubBridgeOperationResolver.LookupDubBridgeOperation(ctx, operationID, middleware.DubBridgeOperationScope)` currently provides the narrow guard contract in `internal/api/middleware/dubbridge_source_guard.go`.
- `integration.ServiceOperationStore.Lookup(ctx, integration.ServiceOperationScope, operationID)` already enforces an active persisted Fenix principal plus exact principal/workspace/org/project membership, returning a wider internal `ServiceOperationRecord`.
- `middleware.authorizeDubBridgeStatus` compares every returned scope field a second time and rejects nil resolver; no HTTP route is mounted.

**C4 adapter boundary** (no code until T4 independent security verdict and C1–C3 QA):
1. Convert ONLY `DubBridgeOperationScope` from verified middleware to `ServiceOperationScope`; never infer tenant membership from operation ID, path, body or raw JSON.
2. Delegate to `ServiceOperationStore.Lookup`; map storage-unavailable to `ErrDubBridgeOperationUnavailable`; map denied/missing/cross-tenant consistently to a generic denial, without operation existence leakage.
3. Return `DubBridgeStoredOperation` with exact operation ID and verified persisted scope for middleware's independent double-check.
4. If status projection is required in isolated httptest, make a distinct explicit allowlist of `operation_id`, `status` and approved trace/run references; do not serialize the internal record or expose SHA-256 digests, arbitrary inputs, tools, prompts or model outputs.
5. Preserve the nil resolver as the default, no production router registration, no public status endpoint, no credential issuance and no actual agent dispatch.

**Negative test matrix** for final C4 reviewer:
- Two independent real SQLite service principals sharing an operation ID cannot read one another's record.
- Exact workspace/org/project disagreement denies both repository lookup and middleware guard, even when IDs coincide.
- Revoked or inactive persisted principals cannot read a formerly accepted operation.
- Nil resolver, invalid operation ID, malformed/encoded path and unavailable SQLite fail closed without scope or existence disclosure.
- Valid scoped record passes `DubBridgeServiceAuth` to an isolated `httptest` next handler once; denied requests never invoke it.
- Public output allowlist omits request/config digests, internal records, raw input, model content and tools.
- No production router or workflow registration changed by tests.

**Stop conditions**: any C1–C3 Go QA failure or missing T4 independent verdict keeps code work at design-only. Any future C4 source must pass its own readiness and independent code review; B2.D owns full HTTP response/error contract and operational transport controls.

## Acceptance

- Implement interface `DubBridgeOperationResolver` from existing middleware with a thin adapter to the real C2 store; no synthetic persisted operation.
- Reject every mismatched principal, workspace, org and project including reused operation IDs across principals.
- No leaked request/config digest, raw prompt/tool args/outputs or tenant details.
- GET/status with nil resolver remains denied; route remains unmounted.
- Real Go 1.25.13 scoped tests/race, independent peer/security review and rollout HITL required.
- No production activation, real credential issuance, main merge or CI.

## Task Card

Task: DB-FENIX-B2.C4
Task file: docs/tasks/task_dubbridge_fenix_b2c4_status_resolver.md
Plan file: docs/plans/dubbridge-consumes-fenix-integration.md
Summary: Bind S2S GET/status exclusively to persisted Fenix principal/workspace/org/project, returning a safe allowlisted projection.
Code affected: internal/api/middleware/dubbridge_operation_resolver.go and isolated tests.
Criticality: critical
Criticality basis: Prevent operation-existence enumeration and cross-tenant status exposure.
Effort/reasoning: High — B2.A3 auth middleware and real SQLite integration.
Recommended model: GPT-6 Cloud with independent reviewer.
Estimated tokens: ~6000
Pseudocode: Verified scope -> scoped store lookup -> sanitized projection -> no route activation.
Peer readiness review approval: reviewer=PENDING; artifact=NOT_RUN; status=BLOCKED_BY_T4
