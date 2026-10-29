---
doc_type: adr
id: ADR-DB-FENIX-S2S
title: "Proposed — restricted DubBridge caller identity at Fenix"
status: proposed
created: 2026-10-29
---

# Proposed ADR — DubBridge as a restricted Fenix agentic service caller

**Status: PROPOSED — NOT ACCEPTED.** Subject to RRI, independent security review and owner task approval. This is a draft design only, not an authorization to deploy or issue credentials.

## Context

DubBridge is the product/client that needs advisory agent/workflow execution. Fenix owns the runtime. Existing `/api/v1` Fenix bearer auth uses human `JWT_SECRET`, `user_id` and `workspace_id`. Attaching a human token to DubBridge would grant a broad user authority rather than one integration capability.

## Decision proposal

- Fenix exclusively owns issuing/recognizing **service credentials**. Use a dedicated verification domain (separate key/audience/issuer/type from human JWTs), bounded TTL, service principal identifier, credential version/JTI, and exact `agentic:workflow:invoke` plus `agentic:run:read` permissions. Choose concrete signed token vs opaque reference after operational rotation review; never share keys with DubBridge.
- Persist a revocable service principal bound to **one Fenix workspace**, one consumer `dubbridge`, and **allowlisted workflow IDs** with active state. An operator provisions/rotates/revokes through an authenticated, audited nonpublic management flow; no public self-issuance.
- Provide **isolated S2S routes**, e.g. `POST /integrations/dubbridge/runs` and `GET /integrations/dubbridge/runs/{id}`; do not mount under generic user-token `/api/v1` without a separate correctly ordered authenticator. Caller-supplied workspace/workflow IDs are advisory at best, never authority.
- Create run with an **atomic unique** composite key `(service_principal_id, external_operation_id)` and immutable request digest. Identical replay returns existing run/status; same key with changed payload → 409. Handle crash after reservation and before outcome with durable state/reconciler; no automatic duplicate execution.
- Validate DubBridge user/org/project at its own API before sending minimized facts. Fenix verifies its own service principal/allowed workspace/operation, **not** another product's JWT. DubBridge claim fields are untrusted metadata and correlation, not impersonation.
- Wrap Fenix DSL workflow runtime through a narrowly allowed execution service with run persistence, result redaction and bounded concurrency/timeouts. Do not permit arbitrary tools, workflow IDs, prompt injection via raw DSL or direct CRM APIs.
- Outbound responses include only `operation_id`, Fenix `run_id`, `status`, `trace_id`, `result` (allowlisted advisory schema) or an error code. Never return raw `reasoningTrace`, `toolCalls`, human JWT, secret or arbitrary internal input.
- Preserve Fenix's native policy/governance/evidence gates; approvals for eventual DubBridge writes continue to be enforced by DubBridge. No direct DB sharing, no second agent engine, no forced paid hosting or MCP/A2A in v1.

## Preferred implementation option for B2 review

**Prefer a high-entropy opaque bearer credential, NOT a copied human JWT, for the first single-consumer S2S slice.** This is a security architecture **recommendation**, not an accepted decision. Fenix stores only a keyed identifier plus one-way credential digest, expiration, revocation and principal/workspace/workflow/version bindings; the plaintext token is revealed once through an operator-controlled provisioning process and injected to DubBridge via its secret configuration. Verify digest using constant-time comparison. Use distinct token namespace and middleware so this bearer never authenticates to ordinary `/api/v1` user routes. Benefits: immediate DB-backed revocation, no cross-product signing key exchange, simple rotation and no claim/audience parser ambiguity. Costs: a DB lookup per request; operational credential distribution is still required. A short-lived signed JWT with separate verifier and durable revocation is a legitimate alternative if independent security review shows an operational need.

**Fail-closed server-side authorization order (required):**
1. Identify the S2S route, parse strict bounded bearer; reject any user JWT or missing token.
2. Load active service principal by bounded public credential key, validate its digest, expiry, revocation/rotation state.
3. Verify `consumer=dubbridge`, allowed action (`invoke` or `read_run`), **one Fenix workspace** and **one immutable workflow version/allowlist**. Never derive those values from body fields or HTTP headers.
4. For a submit, validate strict schema and compare `Idempotency-Key` to body `operation_id` before durable transactional reservation. Bind request fingerprint to both service principal and selected workflow version. Reject same key/different input.
5. Resolve configured workflow and enforce its **effective ToolRegistry capability allowlist**, side-effect classification and current Fenix policy. Merely allowlisting a workflow ID is **insufficient** if workflow permissions or versions drift.
6. Reserve, persist and dispatch exactly one logical run with a durable reconciliation path. Separate service-facing result projection from generic user-facing run DTO; never leak raw model reasoning or tool call parameters.
7. Audit principal, workspace, workflow version, external operation key hash, trace and outcome, not bearer or raw task data.

**Open design decision:** Fenix's existing persistence uses a local application DB for agent runs; choose transaction/lease schema against its actual supported DB before code. Do not use DubBridge PostgreSQL as Fenix's credential/operation store. Require explicit reviewer acceptance of token storage/TTL/rotation, tool-effect limits, idempotency crash safety and service output redaction.

## Security deny matrix (must become real negative tests)

| Attempt | Expected |
|---|---|
| Human user JWT on S2S path | 401/403 |
| S2S credential on ordinary user CRM routes | 401/403 |
| Missing/expired/revoked/rotated service token | 401 |
| Wrong Fenix workspace, non-allowlisted workflow | 403/404, no tenant enumeration |
| Service token with no invoke scope | 403 |
| Service tries to create/modify Fenix tools/agents/CRM | 403 |
| Duplicate identical external request | Same run; no re-execution |
| Duplicate key with changed payload | 409 |
| Provider unavailable or crashed | Explicit pending/failed/indeterminate with reconciliation |
| Malformed payload/untrusted tool or prompt selection | 400/422, no run |

## Required reviewer questions

1. Should service credentials be opaque rotating references or JWTs, given revocation durability and minimal ops cost?
2. What recovery/lease mechanism ensures the unique request key never launches duplicate work after process failure?
3. How to bind Fenix workspace to a DubBridge installation/project without trusting caller-supplied IDs?
4. What explicit Fenix policy/tool allowlist restricts an agent invoked from a service principal?
5. What compact audit record proves caller identity and run correlation without leaking source data?

**Exit:** approve only after code-path threat analysis, signed RRI receipt, peer security review, negative tests and real service run. The owner approved execution of block B1, **not** a waiver of security reviews.
