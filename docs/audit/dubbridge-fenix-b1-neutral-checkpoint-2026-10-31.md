---
doc_type: audit
id: DB-FENIX-B1-NEUTRAL-2026-10-31
title: "B1 neutral contract local tests and security readiness"
status: schema_pass_security_review_pending
date: 2026-10-31
---

# B1 · Capability-neutral contract — evidence and decision

**Product architecture:** DubBridge is the client; Fenix is a service provider. No business agent, LLM, prompt, workflow mapping, test workflow or domain output is required to close the *design* of B1. Actual capabilities are separately onboarded in B6.

## Test execution

The following actual GitHub blobs were reconstructed locally and compared using `git hash-object`; **all five hashes matched exactly**:

| Source | Exact Git blob SHA |
|---|---|
| `docs/contracts/dubbridge-fenix-v1-envelope-request.schema.json` | `3ccbfc7e42c728e1cfcf4c15d86be00fcfcaa745` |
| `docs/contracts/dubbridge-fenix-v1-envelope-response.schema.json` | `68cf5a85ab3691dc954ea1095d241afe2f715f2b` |
| `docs/contracts/dubbridge-fenix-v1-envelope-error.schema.json` | `7a556185bc1a385ff5ebbc5148278f05281896e9` |
| `scripts/int_b1_envelope_selftest.py` | `5b7156be83a46a5c10071f65246a71077a5091c0` |
| `scripts/int_b1_envelope_matrix.py` | `9bbdc18f29ef5d08d84329779250e5f9457dda8f` |

In an isolated Python 3 environment with `jsonschema==4.26.0`, run:

```bash
python3 scripts/int_b1_envelope_selftest.py
python3 scripts/int_b1_envelope_matrix.py
python3 -m py_compile scripts/int_b1_envelope_selftest.py scripts/int_b1_envelope_matrix.py
```

Recorded result:

```text
DUBBRIDGE_FENIX_NEUTRAL_ENVELOPE_SELFTEST=PASS positives=5 negatives=12
DUBBRIDGE_FENIX_NEUTRAL_ENVELOPE_MATRIX_V1=PASS cases=120 accepted=14 rejected=106
PY_COMPILE=PASS
```

**Result:** OFFLINE CONTRACT PASS (17/17 directed and 120/120 matrix cases). **NOT** Fenix S2S authentication, secret revocation, SQLite operation integrity, policy enforcement, Rust integration, runtime E2E or actual agent certification.

## B1.T2 security source review

Read actual Fenix `internal/api/middleware/auth.go`, `pkg/auth/auth.go`, `internal/api/routes.go`, `internal/domain/tool/registry.go` and `internal/infra/sqlite/migrations/001_init_schema.up.sql` on this branch.

- The existing `/api/v1` middleware accepts human JWTs via `pkg/auth.ParseJWT`, injecting `UserID` and `WorkspaceID`. No service identity may enter this middleware as an authorization shortcut.
- User JWTs use `JWT_SECRET`, HS256 and human/workspace claims. Service credentials must use **distinct trust** (recommended: opaque random 256-bit token with digest in Fenix SQLite), and a separate context principal type. Do not mint surrogate users.
- Fenix SQLite `workspace` is the tenant root; S2S bindings must be independently provisioned and checked against trusted workspace + DubBridge org/project policy, not claims from the request body.
- The existing `ToolRegistry` models tool permissions and a capability governor; the external S2S capability registry **must not** bypass those controls or confuse logical integration capabilities with tool executor names.
- The service capability registry starts **empty and default-deny**. Unknown/disabled capabilities are denied before allocating a run. No arbitrary workflow IDs, model prompts, DSL or tool lists from the HTTP caller.
- The generic `input` schema limits the number of top-level properties but **does not** limit JSON depth or total bytes. B2 must enforce a hard limit (proposed initial **64 KiB per request**, **32 KiB per response**, subject to explicit product review), strict content type, bounded nesting/decode, per-capability schema with `additionalProperties:false` and explicit resource quotas; the generic JSON Schema alone cannot secure opaque arbitrary inputs.
- `trace_id`, `source.org_id`, `source.project_id` and `Idempotency-Key` are untrusted wire values. Normalize and bind to an authenticated principal and persistent operation fingerprint. Header `Idempotency-Key` must match body `operation_id`.
- For unauthorized / unregistered capabilities, avoid revealing another tenant's registry inventory; status/error semantics require a uniform nonenumerating policy and a negative test matrix.

These are **self-audit findings**, not a peer/security-review PASS. [Current draft ADR](../decisions/ADR-DB-FENIX-S2S-PROPOSED.md) contains the proposed decision. No token/route/SQLite migration has been implemented.

## B1.T4 risk and RRI / gates

Fenix source-of-truth policies: `docs/policies/RRI_POLICY.md`, `docs/policies/HITL_AUTONOMY_POLICY.md`, `docs/playbooks/AGENT_WORKFLOW_GUIDE.md`. A previously reported **RRI=122** for a monolithic future S2S implementation is only a **hypothetical forecast**, not a real `scripts/rri.py` receipt. **Do not use that value as an all-phase block**.

- B1 present work is a *design, schema and offline-test* task; runtime code was not changed. No code-deployment approval is inferred.
- **B2.A must be decomposed into independently testable leaves:** A1 pure secret parsing/verification and principal representation; A2 SQLite principal/revocation lifecycle; A3 isolated middleware/router guard with deny-matrix; A4 operator rotation/config/QA. Separate readiness RRI measurements, task-specific explicit HITL approval where policy requires, and provider-aware independent peer review. Do not combine migration, middleware and business workflow execution in a single high-risk patch.
- No live primary peer reviewer or complete Fenix checkout is available in this tool execution context. **Peer readiness gate: NOT RUN**. **Official project RRI CLI: NOT RUN**. Both are required before closing high-risk implementation or merging.
- No requirement to run an actual business workflow for B1/B2–B5. In B4, a strictly test-only deterministic diagnostic capability may exercise real authenticated HTTP/SQLite operation acceptance and idempotency, explicitly as *infrastructure E2E*, never as an agent E2E.

## B1 gate and next executable work

**B1.T1: SOURCE REVIEW PASS. B1.T3: OFFLINE SCHEMA PASS (candidate contract). B1.T2/T4: SECURITY/PEER GATE PENDING.**

**GO**: scoped preparation, RRI/readiness and design of **B2.A1 pure credential verification**. **NO-GO**: issuance of real service credentials, modifying live auth/router, applying migrations, production enablement, main merge or claiming a full S2S integration, before explicit task-level approvals and security/peer gates.

Disabled GitHub Actions remain unchanged; no remote workflow runs were triggered by this work.
