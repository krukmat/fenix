---
doc_type: task
id: DB-FENIX-B2.D
title: "B2.D — Cloud-owned non-status S2S admission, denial and trace contracts"
status: cloud_design_delivered
phase: B2.D
week: 2026-W46
tags: [dubbridge, fenix, s2s, contract, trace, admission]
fr_refs: []
uc_refs: []
traceability_scope: neutral_s2s_infrastructure
blocked_by: [DB-FENIX-B1]
blocks: [DB-FENIX-B2.E]
files_affected:
  - docs/tasks/task_dubbridge_fenix_b2d_nonstatus.md
  - docs/tasks/task_dubbridge_fenix_b2d1_denial_contract.md
  - docs/tasks/task_dubbridge_fenix_b2d2_trace_audit.md
  - docs/tasks/task_dubbridge_fenix_b2d3_admission_preflight.md
criticality: critical
criticality_basis: "A future transport adapter can leak tenant data, create false operation evidence or bypass idempotency if authority, error and correlation invariants are not precisely specified."
created: 2026-11-10
completed: null
---

# B2.D — Non-status Cloud block

**Direction:** DubBridge consumes Fenix; B2.D specifies Fenix's neutral S2S provider boundary. **Task owner: ChatGPT Cloud.** RRI N/A. No local agent tokens are budgeted for this phase.

The operator has deferred C4 GET/status. Therefore the current B2.D block excludes GET/status polling, source-backed status projections, result payloads, live HTTP routing, runtime dispatch and any production rollout. Its input dependencies are the **existing** neutral B1 schemas and current inert B2.C1–C3 source; the independent T4 release-security issue remains separate.

## Two-level task structure

| Task | Cloud deliverable | Dependencies | Current state |
|---|---|---|---|
| [B2.D1](task_dubbridge_fenix_b2d1_denial_contract.md) | Decide semantic distinction between pre-reservation denial and lifecycle errors, with a bounded leakage-safe error matrix | B1 proposed neutral envelope, B2.C2 error classes | **Design delivered** |
| [B2.D2](task_dubbridge_fenix_b2d2_trace_audit.md) | Define trace-as-correlation-only, audit metadata minimization and public redaction invariants | B1 trace schema and Fenix-owned S2S principal | **Design delivered** |
| [B2.D3](task_dubbridge_fenix_b2d3_admission_preflight.md) | Define future admission order and negative scenario matrix binding Fenix-computed canonical digest to authenticated envelope | D1, D2, B2.C1–C3 source | **Design delivered** |

## Source-backed Cloud design decisions

1. **Two error boundaries:** Pre-reservation rejection must not masquerade as a versioned lifecycle response requiring an `operation_id`. B1's `CAPABILITY_NOT_CONFIGURED` *admission* concept is not already a lifecycle `error_code` enum member. Avoid mutating the normative response schema by assumption.
2. **Default denial:** unknown/disabled capabilities, invalid credentials, mismatched scopes and invalid JSON must not create a durable operation or agent run.
3. **Source of truth:** principal, workspace, org and project come from active persisted Fenix identity, never DubBridge JSON fields. Registry-derived capability fingerprint remains authoritative.
4. **Digest boundary:** B2.C2 receives a precomputed SHA; a future B2.E ingress must recompute it from its validated approved envelope. The present source is inert and does **not** claim this is currently enforced by an HTTP adapter.
5. **Trace and diagnostics:** correlation is not authorization. Public response fields must be allowlisted. Internal DB/agent/run/error details are not exposed by default.
6. **Terminal ambiguity:** expired or revoked leased records may be marked `indeterminate` only through the explicitly documented conservative reconciliation. No automatic side-effect replay.
7. **Future boundary:** B2.E needs separately approved transport/security implementation planning; operator-policy B2.B4 and independent T4 security acceptance are release-stage gates, not a reason to stop this Cloud design.

## Acceptance for this Cloud block

- D1–D3 cover denied/invalid/replay/conflict/unavailable/ambiguous cases with negative leakage expectations.
- The contract cites actual B1 schemas and C1–C3 capabilities without inventing a registered agent or security clearance.
- Scope is only contract/test planning and safe architectural decisions, not a compiled implementation or local QA PASS.
- **No additional local reviewer or test execution requested** while local token budget is exhausted. Any future implementation can use these task cards when an execution channel is available; this design does not assert a security-gate waiver.
- C4 remains deferred, with `blocks: []` in its own task; no new B2.D dependency on C4.

## Next Cloud-owned work

After delivering D1–D3, continue with a **Cloud-only source-backed gap review** across the neutral B1 envelopes and existing C1–C3 source, correcting the task documents when a factual mismatch is found. Only after the contract is settled should an actual transport-facing implementation be proposed; avoid gratuitous HTTP status mappings or a new endpoint.
