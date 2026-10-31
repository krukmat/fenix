---
doc_type: adr
id: ADR-DB-FENIX-S2S
title: "Proposed — capability-neutral Fenix service principal for DubBridge"
status: proposed
created: 2026-10-29
rebaseline: 2026-10-31
---

# ADR proposal — S2S boundary without a predefined workflow

**Status: PROPOSED. Independent security review pending.** Owner clarification (2026-10-31): no actual workflow, agent, prompt, business operation or model provider needs to be chosen during B1–B5. Those are B6 use-driven capability onboarding. This ADR **replaces earlier assumptions** of binding DubBridge's first S2S credential to a hardcoded review backlog workflow and fixed workflow version.

## Context

DubBridge calls Fenix to use its agentic platform. Fenix's existing `/api/v1` routes consume a **human JWT**; `workflows/{id}/execute` invokes its DSL runtime inline; generic run responses contain internal inputs, tool calls and reasoning. This is **not** a valid S2S integration surface.

## Proposed foundation decision

1. **Authentication:** Fenix owns a dedicated nonhuman S2S principal and restricted bearer. For a low-scale first consumer prefer a high-entropy opaque credential stored by digest only, with server-side active/revoked/expiry/rotation status in Fenix SQLite. Operator-issued via audited internal process; no shared `JWT_SECRET`, no Fenix user JWT, no credential embedded in DubBridge client code or responses. Human JWTs never authenticate S2S routes; S2S credentials never authenticate CRM routes.
2. **Scopes:** `integration:capabilities:invoke` and `integration:operations:read` as separate permissions. Principal references one configured Fenix workspace and an explicitly authorized DubBridge organization/install identity; project-level access is constrained by verified, centrally provisioned allowlists as needed. Source org/project in JSON is **not authority**. Fail closed for missing mappings; do not assume workspace ID alone isolates Blackboard/memory.
3. **Capability registry:** key/version is a **logical contract** distinct from workflow/agent ID. The registry starts empty, with all capabilities disabled until operator registration. A principal may invoke only a registered, enabled, specifically allowed capability for its mapped workspace/tenant. Unknown/disabled capability returns a stable typed error **before durable acceptance or model execution**. Fenix must never auto-create or select an arbitrary workflow from the caller's text.
4. **Per-capability onboarding (B6):** define versioned input/output schemas, approved tool effects/budget, immutable workflow/agent mapping and independent approval when a real use case arises. No concrete workflow ID or read-only review analysis is required to build the S2S foundation.
5. **Transport envelope:** versioned bounded JSON includes `operation_id`, `capability_key`, `input_schema_version`, `trace_id`, source context and `input`. Generic schema enforces structural limits; at invocation Fenix additionally enforces the registered capability-specific input schema, policy and budget. No user-supplied DSL/prompt/tool list. Responses must use sanitized typed per-capability output, never the generic internal `agentRunToResponse`.
6. **Operation safety:** unique `(principal_id,operation_id)` plus canonical request digest and immutable capability configuration fingerprint in Fenix SQLite. `operation_id` is the stable polling reference, `run_id` is **nullable** until an actual runtime run exists. Replayed identical commands never create a second logical operation; changed payload with same key yields 409. Unknown capability yields rejection before reserving an accepted operation. Crash/lease ambiguity yields `indeterminate` rather than unsafe automatic redispatch.
7. **Audit:** principal, mapped workspace/source scope, capability key/version, operation ID hash, timestamp, trace, deny/accept status, safe error and audit event. Never log raw bearer, model prompt, personal media or unrestricted input.
8. **Rollout:** first deploy **disabled-by-default** with no registered domain capability. Prove real DubBridge↔Fenix transport, auth/revocation/cross-tenant denial, capability-not-configured rejection and idempotency/status behavior with real apps and stores. Actual agent execution belongs to B6, independently approved.

## Threats and verification matrix (foundation only)

| Risk | Required control / negative evidence |
|---|---|
| Human JWT accepted as service identity | 401/403 on S2S, tests against real auth middleware |
| Service bearer accepted by ordinary user/CRM API | 401/403, no user context injection |
| Expired/revoked/rotated credential | fail closed and auditable |
| Cross-organization or wrong Fenix workspace | deny before any capability/runtime lookup, no tenant enumeration |
| Unregistered/disabled capability | explicit `CAPABILITY_NOT_CONFIGURED` / `CAPABILITY_DISABLED`, no run created |
| Caller supplies arbitrary workflow ID, prompts or tool definitions | request schema rejects, no execution |
| Replay and network ambiguity | same logical operation for matching digest, 409 on mismatch; no unsafe retry |
| Leaked model/tool reasoning | no generic internal run DTO on S2S boundary |
| Request flood or costly execution | principal-bound rate/size/compute budgets, disabled capability cannot consume model resources |
| Compromised per-project integration data | centrally configured tenant scopes, isolation tests for workspace/Blackboard/logs |

## Sequencing, risk and review gate

- **B1:** architecture, machine-readable **generic** envelope, RRI assessment and security review. No live workflow required.
- **B2:** **decompose** into S2S auth, capability registry, SQLite operation registry, read/status projection, negative foundation tests. No business workflow configured.
- **B3/B4:** real DubBridge consumer and infrastructure E2E with capability absent; this is valid proof of **foundation**, not functional model execution.
- **B5:** independent security review, local Go/Rust QA and operational GO/NO-GO.
- **B6:** define/version actual workflow from real use; certify full agent/model business E2E there.

Prior monolithic RRI forecast 122 (Excessive) was hypothetical, **not official or measured**. Recalculate actual RRI per B2 leaf before implementation, following Fenix policy. Reviewer still must verify token entropy/digest/rotation, tenant isolation, concurrency/lease/idempotency, tool policy and output redaction before deployment. No approvals are silently granted by this design rewrite.

**Decision status: draft only. No service credential issued; no S2S runtime implemented; no changes to main or GitHub Actions.** Historical threat review remains in [older B1.C audit](../audit/dubbridge-fenix-b1c-gate-2026-10-31.md); its business-workflow gate is superseded.
