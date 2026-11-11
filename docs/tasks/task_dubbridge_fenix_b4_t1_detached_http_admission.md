---
doc_type: task
id: DB-FENIX-B4.T1
title: "Detached, fail-closed S2S HTTP admission"
status: cloud_source_published_go_qa_peer_pending
phase: B4
week: 2026-W46
tags: [dubbridge, fenix, http, security, s2s]
fr_refs: []
uc_refs: []
traceability_scope: neutral_s2s_infrastructure
blocked_by: [DB-FENIX-B2.E, DB-FENIX-B2.B4]
blocks: [DB-FENIX-B4-REAL-E2E]
files_affected:
  - internal/api/s2s_admission_http.go
  - internal/api/s2s_admission_composition.go
  - internal/api/s2s_admission_http_test.go
  - internal/api/s2s_admission_sqlite_test.go
  - internal/domain/integration/s2s_admission.go
  - internal/domain/integration/s2s_admission_test.go
criticality: critical
criticality_basis: "An S2S transport boundary must not reuse human JWT, admit forged scope or mismatched idempotency keys, leak bearer/error details, or unintentionally mount a live endpoint."
created: 2026-11-11
completed: null
---

# DB-FENIX-B4.T1 — Detached HTTP admission

**Task owner:** ChatGPT Cloud. **RRI:** N/A, DB-FENIX-INT cloud. **Criticality:** critical. **Effort/reasoning:** High — security boundary, durable reservation, and request replay semantics. **Estimated tokens:** ~7000. **Recommended model:** Cloud author; independent reviewer only when an approved execution channel is available. **Peer readiness/code approvals:** NOT_RUN (no local agent tokens); no PASS or waiver claimed.

## System context — not mounted in production

```text
DubBridge Rust client (proposal)
       | POST /integrations/dubbridge/operations
       | Bearer fenix_s2s_v1_* + Idempotency-Key
       v
[ DETACHED Fenix HTTP handler ] ---- X NOT registered in api.NewRouter
       | max 64 KiB; exact path/method; canonical bearer; single headers
       v
Fenix S2SAdmission.AdmitWithIdempotencyKey
       | verified persisted principal before operation key validation
       | B1 validated envelope + header operation_id equality
       | active operator capability + REQUIRED versioned input validator
       v
C2 ServiceOperationStore.Reserve (atomic SQLite scope and replay guards)
       |
       +-- 202 new / 200 replay (operation/trace/replayed only)
       +-- 400 invalid / 403 opaque denied / 409 conflict / 503 unknown
       X NO agent_run / GET status / worker / human JWT / route activation
```

## Sequential tasks, ChatGPT owned

| Subtask | Source deliverable | Current state |
|---|---|---|
| **T1.1** | Explicit detached handler, strict POST/path, bounded body/header parsing, secure headers; default router unchanged | Source published |
| **T1.2** | Real persisted Fenix credential/store composition; authenticate before comparing header and validated B1 ID; redact and project allowlisted response | Source published |
| **T1.3** | Unit `httptest` and migrated SQLite HTTP tests for new/replay/conflict, invalid/missing/duplicate header, invalid body/encoding, forged scope, revoked principal, disabled capability, no agent dispatch; default router negative test | Tests authored, **NOT_RUN** |

### Pseudocode

```text
if method/path/header shape invalid: generic refusal
token <- parse canonical Fenix S2S bearer, never JWT
raw <- read bounded HTTP JSON body (no compression)
admission <- persisted Fenix principal; derive trusted org/project
validate B1 envelope, capability schema and equal Idempotency-Key
reserve in C2 SQLite atomically; never invoke handler/worker
return allowlisted 202/200 receipt or bounded 400/403/409/503 error
```

## Verification and gates

**Read-only Cloud source inspection:** 20/20 structural assertions PASS; this covers source presence and control flow, **not Go compilation**. Go 1.25.13 tests, race, vet, gofmt, project prepush, independent task readiness/code review, security T4, rate limits and operator provisioned B2.B4 configuration remain NOT_RUN/NOT_APPROVED.

**Activation prohibition:** `internal/api/routes.go`, `internal/server/server.go` and production configuration are unchanged. `NewDormantS2SAdmissionHandler` and `NewDormantS2SAdmissionFromStore` produce stand-alone handlers only. No registered URL, CI, new migration, credentials, deployment, merge or live E2E. **DO NOT mount until explicit owner/security approval and operator policy exist.** C4 GET/status remains deferred.

**Tracking:** [DubBridge B4 #13](https://github.com/krukmat/dubbridge/issues/13); [Fenix B2.E #10](https://github.com/krukmat/fenix/issues/10).
