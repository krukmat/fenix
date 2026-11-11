---
doc_type: task
id: DB-FENIX-B2.E
title: "B2.E — Cloud-built inert authenticated admission and sanitized denial"
status: cloud_source_audit_complete_go_peer_pending
phase: B2.E
week: 2026-W46
tags: [dubbridge, fenix, s2s, admission, cloud]
fr_refs: []
uc_refs: []
traceability_scope: integration_infrastructure_no_product_fr_uc
blocked_by: [DB-FENIX-B1, DB-FENIX-B2.A2, DB-FENIX-B2.B, DB-FENIX-B2.C2, DB-FENIX-B2.D]
blocks: [DB-FENIX-B3]
files_affected:
  - internal/domain/integration/s2s_admission_envelope.go
  - internal/domain/integration/s2s_admission.go
  - internal/domain/integration/s2s_admission_projection.go
  - internal/domain/integration/s2s_admission_test.go
  - docs/plans/dubbridge-consumes-fenix-integration.md
criticality: critical
criticality_basis: "An untrusted S2S envelope can attempt to forge tenant identity, bypass a capability schema, cause a duplicate persisted operation or leak provider internals."
created: 2026-11-11
completed: null
---

# B2.E — Inert service admission

**Direction:** DubBridge consumes Fenix. **Accountable and implementation owner:** ChatGPT Cloud. **RRI:** N/A for DB-FENIX-INT cloud. No local agent tokens available. **Status:** source/test candidate, not certified.

## Context

```text
DubBridge (untrusted raw JSON + bearer)
         |
         v
Fenix auth.ServiceCredentialStore.AuthenticateServiceToken
         | current persisted principal and workspace/org/project
         v
B2.E1 strict bounded v1 envelope: match authorized source and hash normalized JSON
         |
         v
B2.E2 in-process admission: B2.B registry.Resolve + REQUIRED approved input validator
         |
         v
B2.C2 ServiceOperationStore.Reserve (atomic SQLite INSERT SELECT)
         |
         +--> bounded S2SAdmissionReceipt: operation_id, trace_id, replayed only
         |
         +--> B2.E3 static safe error projection (no HTTP status chosen)
         X    No route / no GET status / no agent_run / no workflow invocation
```

## Two-level block plan

| Task | Code or evidence | Dependencies | ChatGPT delivery |
|---|---|---|---|
| [B2.E1](task_dubbridge_fenix_b2e1_inert_envelope.md) | Strict B1 envelope, principal/tenant match, internal digest | B1, A2 | Authored |
| [B2.E2](task_dubbridge_fenix_b2e2_inert_reservation.md) | Credential + registry + capability-input validator + C2 reservation | E1, B2.B, C2 | Authored |
| [B2.E3](task_dubbridge_fenix_b2e3_inert_errors.md) | Allowlisted invalid/denied/conflict/unavailable public codes without HTTP | D1/D2 | Authored |
| [B2.E4](task_dubbridge_fenix_b2e4_inert_tests.md) | Persisted-credential SQLite acceptance/denial/replay tests | E1–E3 | Authored, not executed in project Go |
| [B2.E5](task_dubbridge_fenix_b2e5_inert_audit.md) | Source cross-check, hardening ledger and scoped release restrictions | E1–E4 | Cloud static review only |

**Latest Cloud source checkpoint:** [B2.E Cloud audit](../audit/dubbridge-fenix-b2e-cloud-source-review-2026-11-11.md) reports **21/21 static checks PASS** over E1–E4. Source snapshot `b9549ca6ad05f602a49457ae23292cc3f1a7ef2c`; this is *not* Go/race/vet/prepush PASS. Tracking [issue #10](https://github.com/krukmat/fenix/issues/10). No local work has been requested.

## Definition of done by boundary

- **Cloud implementation:** source and test code committed to this existing feature branch; no newly registered capability, public route, background agent, credential issuance, or workflow execution in production.
- **E1:** reject unknown/missing fields, non-UUID tenant source, malformed Unicode, duplicate JSON keys, oversized payload and mismatched authenticated tenant. Compute digest from the canonical full envelope inside Fenix.
- **E2:** require a persisted verified principal and explicitly enabled registry binding plus a mandatory operator-owned capability-specific input validator. Preserve C2's atomic reserve/replay semantics.
- **E3:** never return an internal database error, bearer, hashed request/config, tenant details, lease, internal status or run ID.
- **E4:** negative test matrix covers malformed input, forged scope, invalid/revoked tokens, disabled capability, no input schema, replay/conflict, no new `agent_run`.
- **E5:** document actual verifications versus unexecuted Go 1.25.13 / race / vet / project prepush / independent readiness/code review.

**Important security separation:** an injected test `S2SInputValidator` is not a production capability-schema registry. B2.B4 must provide audited operator registration/versioned schema management before any live route. A missing validator is explicitly denied. C4 GET/status remains deferred; independent T4 security is separate. **No runtime activation, CI, deployment, production migration, or merge.**

**Current test constraint:** Cloud environment has Go 1.23.2 but Fenix requires Go 1.25.13 and dependencies. Full project's Go QA and independent peer gates cannot be truthfully certified here; user has no local tokens, so no local-agent handoff is issued.
