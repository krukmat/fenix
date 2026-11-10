---
doc_type: task
id: DB-FENIX-B2.D1
title: "B2.D1 — non-status denial and error projection contract"
status: cloud_design_ready
phase: B2.D
week: 2026-W46
tags: [dubbridge, fenix, s2s, contract, errors, security]
fr_refs: []
uc_refs: []
traceability_scope: neutral_s2s_infrastructure
blocked_by: [DB-FENIX-B1]
blocks: [DB-FENIX-B2.D3]
files_affected:
  - docs/contracts/dubbridge-fenix-v1-envelope-response.schema.json
  - docs/tasks/task_dubbridge_fenix_b2d1_denial_contract.md
criticality: critical
criticality_basis: "An incorrect public error projection can disclose tenant identity, existence of capabilities, internal run state or credentials."
created: 2026-11-10
completed: null
---

# B2.D1 — failure contract before any status interface

**Owner:** ChatGPT Cloud. **Direction:** DubBridge consumes Fenix. **RRI:** N/A (cloud integration). This is a **Cloud-owned source-contract design** task; no local agent allocation.

## Existing normative facts

- The [B1 neutral response envelope](../contracts/dubbridge-fenix-v1-envelope-response.schema.json) is a **proposal** for actual persisted operation lifecycle outcomes; its `error_code` enum contains `POLICY_DENIED`, `EXECUTION_FAILED`, `PROVIDER_UNAVAILABLE`, `TIMEOUT`, `UNKNOWN_STATE`.
- The same envelope **requires an `operation_id`**. It cannot truthfully represent pre-reservation denial when no operation was created; no false `accepted`, `run_id`, or durable journal row may be fabricated.
- [B2.C2](task_dubbridge_fenix_b2c2_operation_store.md) exposes internal `ErrServiceOperationInvalid`, `ErrServiceOperationDenied`, `ErrServiceOperationConflict`, `ErrServiceOperationUnavailable`. Raw error text is **not** an approved public transport contract.
- A capability that is absent/disabled is default-denied. The B1 plan mentions `CAPABILITY_NOT_CONFIGURED` for the **future** admission boundary, but it is **not** in the lifecycle response schema's enum.
- C4 GET/status is deferred; no status projection or polling contract is implemented by this task.

## D1.1 — Define the separation between admission and lifecycle failures

**Decision:** retain two distinct semantic envelopes, with no implementation claim:

| Failure boundary | Public result policy (design candidate) | Durable operation |
|---|---|---|
| Unauthenticated, expired or revoked principal | Opaque authorization denial; no principal/capability existence oracle | None |
| Malformed/oversized envelope, unsupported version | Bounded invalid-request response, without leaking accepted parser structure | None |
| Unknown or disabled capability | Sanitized **pre-reservation** rejection; distinguish `CAPABILITY_NOT_CONFIGURED` only if authenticated and policy authorizes disclosure | None |
| Existing operation ID but different immutable input/config/trace | Conflict response with no request hash, registered handler, or tenant fields | Existing row unchanged |
| SQLite storage outage/timeout | Service-unavailable/unknown-outcome policy; never claim a new reservation or re-run a side effect | None or unknown; no fabricated status |
| Existing admitted operation later becomes indeterminate | Future lifecycle contract may use `UNKNOWN_STATE` only on an explicitly authorized projection | Remains `indeterminate`, no automatic replay |

HTTP status codes remain a **future B2.E boundary decision** pending actual route/auth context. Do not hard-code 401 vs 403 vs 404 or 409 vs 422 from design assumptions. Keep opaque denial for unauthenticated and unauthorized actors and prevent information leakage through status-code differentiation.

## D1.2 — Define public-safe fields

A candidate pre-reservation denial can expose only an allowlisted static code, a generic stable message and a safely generated correlation ID. Never expose raw `error.Error()`, bearer/JWT, principal ID, workspace/org/project identifiers, capability registration details, config/request hashes, SQL errors, stack traces, `lease_owner`, or internal `run_id`. A correlation ID is a reference, not an authorization token.

## D1.3 — Acceptance

1. Every known B2.A/B2.B/B2.C denial category maps to **exactly one** generic public classification; unknown failures fail closed.
2. Unauthorized access to another principal's operation is indistinguishable from a missing operation at the public boundary.
3. Pre-reservation denials never manufacture a persisted operation or lifecycle response.
4. Terminal `indeterminate` is not retried automatically and is not converted to `failed` or `accepted` for convenience.
5. Public error schema must be separately versioned/reviewed before B2.E implementation, without rewriting the approved B1 neutral envelope by assumption.
6. Deliverable for this Cloud block is this design and a negative-case mapping matrix. **No route, status GET, production deployment or independently certified runtime behavior.**

**Dependency policy:** B2.A3.T4 independent security review is tracked separately for any later live security boundary. It does not block this Cloud contract design.
