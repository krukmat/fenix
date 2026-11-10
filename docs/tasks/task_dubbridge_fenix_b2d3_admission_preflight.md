---
doc_type: task
id: DB-FENIX-B2.D3
title: "B2.D3 — safe admission preflight and error/trace negative test plan"
status: cloud_design_ready
phase: B2.D
week: 2026-W46
tags: [dubbridge, fenix, s2s, input-validation, idempotency, negative-tests]
fr_refs: []
uc_refs: []
traceability_scope: neutral_s2s_infrastructure
blocked_by: [DB-FENIX-B2.D1, DB-FENIX-B2.D2, DB-FENIX-B2.C2]
blocks: [DB-FENIX-B2.E]
files_affected:
  - docs/tasks/task_dubbridge_fenix_b2d3_admission_preflight.md
criticality: critical
criticality_basis: "A caller-supplied operation digest or identity field could bypass durable replay, tenant isolation or sanitized denial if the future ingress fails to bind verified inputs."
created: 2026-11-10
completed: null
---

# B2.D3 — Non-status preflight, fully offline design

**Owner:** ChatGPT Cloud. **RRI:** N/A. This task defines future B2.E admission invariants but does not activate an HTTP endpoint or a live capability.

## D3.1 — Required order at a future admission boundary

```text
bounded transport request
  -> authenticate Fenix S2S principal (persisted live authority)
  -> derive workspace/org/project scope from Fenix identity
  -> validate versioned B1 request schema, size, JSON/UTF-8/Unicode
  -> require caller-declared source to match Fenix-owned scope; never trust it to grant scope
  -> resolve enabled Fenix capability/schema policy (default deny)
  -> recompute canonical request digest in Fenix from approved envelope
  -> call existing C2 Reserve with Fenix-derived scope and digest
  -> acknowledge only a genuinely reserved/replayed operation, no execution
```

**Design boundary:** raw `RequestSHA256` accepted by current C2 Go API is *not* proof of correct envelope binding. Future ingress must recompute it rather than trust a client-provided SHA. C2 `ConfigSHA256` is registry-derived today and cannot be injected by the caller. `trace_id` remains immutable and must match on replay.

## D3.2 — Cloud-authored negative scenario matrix

| Input / state | Expected admission outcome | Forbidden effect |
|---|---|---|
| No, invalid, expired or revoked S2S credential | Opaque denial | New operation or `agent_run` |
| Mismatched workspace, org or project from body | Denial | Cross-tenant record |
| Unknown/disabled capability or wrong schema version | Default denial | New accepted row |
| Missing or malformed operation/trace ID | Bounded invalid input | Persisted identity |
| >64 KiB JSON, duplicate nested key, invalid raw UTF-8 or lone surrogate | Bounded invalid input | Canonical digest alias |
| Same authenticated scope/ID, identical envelope/config/trace | Replay of existing record | Duplicate row or side effect |
| Same ID, changed input/config/trace | Conflict | Silent overwrite |
| Revoke between authorization and reservation | Persisted-principal insert check denies | Accepted row after revoke |
| Capacity/storage/timeout uncertainty | Unknown/unavailable classification | Guessed `accepted` or automatic retry |
| Revoked principal with previously expired lease | Operator reconciliation only | Restored caller lookup or redispatch |

Tests for HTTP status/body, request throttling and dynamic admission are allocated to future B2.E after explicit implementation approval. Their exact expected status codes remain undecided until an authenticated API contract is ratified.

## D3.3 — Acceptance and stopping point

1. Matrix reconciles B1 schema and B2.C live SQL/Go contract; no invented registered capability or run.
2. All authority comes from persisted Fenix identity and registry; DubBridge source is untrusted input.
3. B2.C1–C3 remains an inert source-only foundation. B2.E may implement transport after separate source/security planning; actual production use needs T4, B2.B4 and explicit owner approval.
4. This task is **Cloud documentation and test specification**, not executed code or a local QA requirement. No further local agent usage is requested.
