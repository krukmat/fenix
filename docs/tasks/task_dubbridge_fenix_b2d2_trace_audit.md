---
doc_type: task
id: DB-FENIX-B2.D2
title: "B2.D2 — non-status trace correlation, audit boundaries and redaction"
status: cloud_design_ready
phase: B2.D
week: 2026-W46
tags: [dubbridge, fenix, s2s, tracing, audit, redaction]
fr_refs: []
uc_refs: []
traceability_scope: neutral_s2s_infrastructure
blocked_by: [DB-FENIX-B1]
blocks: [DB-FENIX-B2.D3]
files_affected:
  - docs/tasks/task_dubbridge_fenix_b2d2_trace_audit.md
criticality: critical
criticality_basis: "Trace propagation and error logging can leak credentials, tenant identifiers or private request contents across S2S boundaries."
created: 2026-11-10
completed: null
---

# B2.D2 — Trace and audit on the inert provider boundary

**Owner:** ChatGPT Cloud. **RRI:** N/A. The contract is an integration-infrastructure concern without product FR/UC. No local-agent tokens are allocated.

## D2.1 — Correlation invariants

1. `trace_id` is a **caller-supplied correlation value**, validated and bounded by B1; it is **not** proof of caller identity or authorization.
2. The Fenix-side persisted principal and tenant scope come exclusively from authenticated Fenix authority, not from `trace_id` or arbitrary DubBridge `source` data.
3. On accepted reservations, the persisted `trace_id` is immutable (C1 trigger; C2 replay equality). On conflicts, do not return the stored trace of another request.
4. The operation ID is an idempotency key scoped to its authenticated principal, **not** a globally queryable reference. Public error correlation must not make it a lookup oracle.
5. Cross-service correlation for a denial is safe only where the authenticated boundary authorizes the same correlation context. Otherwise generate an opaque Fenix request ID and keep internal linkage private.

## D2.2 — Audit data-minimization table

| Event | Minimum protected audit metadata | Publicly exposed |
|---|---|---|
| Auth rejected | Timestamp, coarse reason category, internal audit correlation, bounded outcome | Generic denial only |
| Capability denied | Internal audit correlation, resolved policy decision category, sanitized outcome | Opaque unconfigured/denied classification by authorization policy |
| New reservation/replay | Fenix-scoped operation reference, event type, internal correlation, timestamp | Admission acknowledgment only when authorized later |
| Idempotency conflict | Internal reference, conflict category, time | Static conflict response; no comparison hashes |
| Lease accepted/reconciled | Lease transition type and bounded timestamp, internal operation key | Nothing in this phase |
| Storage unknown/failure | Internal failure class, sanitized storage-health signal, correlation | Generic unavailable/unknown-outcome policy |

Do not log raw tokens, payloads, request/config digests, full SQL parameters, unredacted principal or org IDs, stack traces or operator secrets as public/debug material. Existing Fenix internal audit storage is a separate authorization surface; the table above is a **target design**, not a claim that it is already emitted.

## D2.3 — Acceptance

- Trace propagation does not affect authorization, tenant lookup, replay or lease acquisition.
- Every future public field is allowlisted and source-backed; no generic `json.Marshal(internalRecord)`.
- Sensitive internal audit logs and public transport diagnostics are explicitly separated.
- Denial, replay/conflict, ambiguous expiry and revoked-principal cases have negative leakage test scenarios defined for B2.E.
- No GET/status resolver, broker, worker or actual log/transport activation in this source-only Cloud task.

**Release boundary:** T4 and operator policy B2.B4 are separate gates for live usage, not dependencies for this document.
