---
doc_type: plan
id: INT-W0
title: "Fenix to DubBridge: real integration analysis and contract"
status: superseded
created: 2026-10-28
tags: [fenix, dubbridge, integration, toolregistry, governance]
---

> **SUPERSEDED (2026-10-29):** owner clarified **DubBridge → Fenix**, with Fenix as the agent/workflow service provider. This document is a historical *reverse-direction* source audit, not an active implementation plan. Follow [canonical Fenix provider plan](dubbridge-consumes-fenix-integration.md).
# INT-W0 — Fenix ↔ DubBridge (Fenix-owned integration)

## Intent and actual status

Define a real, decoupled Fenix -> DubBridge workflow. This document is an analysis/design record, **not evidence of an implemented or passing cross-product integration**. Work is isolated on `feature/int-w0-dubbridge-integration-contract`; no runtime files or deployed services are changed by INT-W0.

The first production-representative vertical slice is **read-only review-queue inspection**. Fenix agents can retrieve and summarize DubBridge review tasks and propose human next actions; they must not record review decisions, grant rights, publish, trigger media processing, issue invitations, or alter P2P state.

**Explicit owner requirement, 2026-10-28:** no mocks, stubs, canned responses, or fixture-only integration tests may be presented as proof that Fenix and DubBridge actually interoperate. Unit/contract tests may support development, but the acceptance gate is a **real two-runtime end-to-end run** using an authenticated DubBridge principal, a real PostgreSQL-backed organization/project/review task, and actual Fenix ToolRegistry/Blackboard execution. No paid hosting or production rollout is authorized.

## Code and prior contracts inspected

- Fenix `internal/domain/tool/registry.go`: `RegisterCapabilityWithDefinition` and persisted permission/schema handling.
- Fenix `internal/domain/tool/capability.go`: `SideEffectRead`, `CapabilityRetryPolicy`, `CapabilityExecutionContext`, `execution_id`.
- Fenix `internal/domain/tool/capability_runtime_pipeline.go`: governance before provider invocation and bounded retry logic.
- Fenix `internal/infra/integration/m2sf/executor.go`: reusable HTTP/JSON integration pattern, trace and execution headers.
- Fenix `internal/api/integration_runtime.go` and `internal/api/integration_runtime_w6_test.go`: existing provider registration and functional Blackboard proof; **not** a DubBridge integration.
- Fenix `docs/integration-overview.md` and `docs/plans/fenix-integration-w6-functional-execution.md`: one authority per domain, tool-governed capabilities, optional independent evidence.
- DubBridge `apps/api/src/routes/review.rs`: existing GET queue and POST decision/publish routes.
- DubBridge `apps/api/src/dto/review.rs`: existing `ReviewQueueResponse` and `ReviewTaskResponse`.
- DubBridge `crates/auth/src/{service,issuer,principal}.rs` and `apps/api/src/middleware/org_scope.rs`: identity, scopes, org membership.
- DubBridge `docs/adr/ADR-008-rights-ledger-fail-closed-precondition.md`, `ADR-030-review-decision-ledger-and-fail-closed-publication-gate.md`, `ADR-031-mobile-jwt-credential-auth-fenix-parity.md`.

DubBridge-specific security/transport analysis and owner tasks:
[DubBridge integration boundary](https://github.com/krukmat/dubbridge/blob/feature/int-w0-fenix-integration-boundary/docs/plan/fenix-integration-int-w0-boundary.md).

## Ownership boundary

| Fenix owns | DubBridge owns |
|---|---|
| Agent planning, Blackboard, governed ToolRegistry invocation | Assets, review queue, actual review decisions, rights and publication |
| Fenix workspace/user/workflow/trace/execution IDs | DubBridge account/org/project/subject IDs and authorization |
| Input validation, least-data result projection, operational audit | Permission check on every API call, publication and media authority |
| Circuit timeout/retry classification for read-only calls | Current business state, verified JWT/membership, domain rejections |
| Optional independent Fenix evidence lifecycle (VEL) | Own immutable rights/review/audit records; never delegate gates |

Fenix's approval is **not** DubBridge's review approval. The same Fenix JWT must not be accepted as a DubBridge token merely because both projects currently use HS256. Never share the JWT signing secrets.

## Selected user-visible scenario (INT-W0.T1)

A human requests a summary of pending review work for an authorized DubBridge organization and project. A Fenix workflow calls the existing DubBridge queue API, receives real current tasks, stores a minimal agent-safe observation on Blackboard, and proposes priorities. An empty queue is a valid empty result, not a fabricated task. The user completes any actual approval/publication inside DubBridge.

Benefits measured in INT-W1: correct queue count/IDs/states, correlation to one Fenix execution, no cross-tenant leakage, and zero mutations.

## As-built API inventory (INT-W0.T2)

Existing DubBridge backend:
`GET /orgs/{org_id}/projects/{project_id}/review-tasks`.

Existing mobile gateway relay:
`GET /api/orgs/{org_id}/projects/{project_id}/review-tasks`.
The gateway forwards bearer credentials; it is not a new identity authority.

Existing response fields: `org_id`, `project_id`, `tasks[]`; each task contains `id`, `org_id`, `project_id`, `asset_id`, `target_language_id`, `assignee_subject_id`, `state`, `created_at`, `updated_at`, `assigned_at`. States are `pending | approved | rejected`.

DubBridge requires **its own verified bearer**, `workspaces:read`, membership of the named organization, and minimum `Reviewer` role. Backend checks that the project belongs to the organization.

Important current gap: the default DubBridge credential-login issuer grants scopes including `assets:ingest` and `workspaces:write`, so simply storing a normal user's login token in Fenix does **not** satisfy least privilege.

## Proposed Fenix-side v1 contract (INT-W0.T3; not implemented)

Semantic capability: `dubbridge.review_queue.list`, version `1`, operation `list`, `SideEffectRead`, max two attempts for retryable read errors, no mutation idempotency requirement.

Input to Fenix capability:
```json
{"contract_version":"1","org_id":"<uuid>","project_id":"<uuid>"}
```
`org_id` and `project_id` are validated for UUID syntax; Fenix authorization must select an explicitly authorized mapping to the destination tenant. Never derive destination authorization from caller-supplied UUIDs alone.

The HTTP adapter forwards only the permitted resource identifiers and a **DubBridge-issued limited credential**; it supplies `X-Fenix-Trace-ID` and `X-Fenix-Execution-ID` as correlation hints. Current DubBridge does not guarantee storage/echo of these headers: a backend/gateway correlation contract must be verified or added in INT-W1 before claiming end-to-end trace propagation.

The provider response is validated against DubBridge's *existing* wire DTO and normalized inside Fenix to a versioned **agent-safe** result containing only authorized task references/status. Never expose JWTs, email/passwords, consent evidence, P2P decryption details or private review comments to the Blackboard. The normalized response and enrichment are **proposed**, not extant endpoints.

Errors: 400/401/403/404 are non-retryable and preserve their semantics without leaking tenant existence; timeout/network/5xx are retryable for this GET with a bounded retry budget; malformed provider JSON or contract drift fail explicitly; no fallback synthetic success or guessed queue. A 2xx response does not by itself prove that the correct tenant was authorized.

## Candidate implementation (INT-W0.T4; design only)

```text
Fenix user / workflow
      -> Agentic Blackboard / PlannerExecutor
      -> ToolRegistry: dubbridge.review_queue.list
      -> Fenix permissions + governance
      -> Go DubBridgeToolExecutor (HTTP GET, bounded response, timeout, safe errors)
      -> DubBridge gateway /api/... or private apps/api /orgs/...
      -> DubBridge JWT + scope + org membership + project check
      -> PostgreSQL review queue
      -> validated minimal Fenix result -> Blackboard / audit
```

Proposed files for the next *separately gated* implementation: `internal/domain/dubbridgeinterop/`, `internal/infra/integration/dubbridge/`, and opt-in registration in `internal/api/integration_runtime.go`; actual locations must be rechecked on INT-W1 start. Do not alter DubBridge's rights, approval, or P2P engines.

## Real integration acceptance (no mocks)

1. Start actual Fenix and DubBridge API (and their required local DB/queue services) from pinned source revisions; record those revisions and launch commands.
2. Populate a **real**, authorized DubBridge org/project and a review task using existing supported paths or an auditable dev seed, not a mocked HTTP provider. Record task IDs and fixture provenance; no fabricated API responses.
3. Provision a DubBridge-issued restricted integration identity with `workspaces:read` only and verified role/membership. If unavailable, mark the real integration **BLOCKED**, not PASS.
4. Execute a real Fenix Blackboard -> ToolRegistry -> DubBridge GET; compare returned IDs/states with DubBridge PostgreSQL/source endpoint.
5. Prove denial for missing/expired token, missing membership, unauthorized project, and unknown task state. Verify Fenix doesn't invent results or retry a denied request.
6. Exercise a real temporary provider outage and restore; verify bounded retries, stable `execution_id` and recovery, with zero DubBridge mutation.
7. Inspect both runtimes' logs/audit to prove correlation without leaking credentials. If DubBridge lacks correlation instrumentation, record a follow-up and do not claim full traceability.
8. Preserve command output and pass/fail proof under each repository's own audit directory. Passing only local Go/Rust unit tests or a mock server is insufficient.

## INT-W0.T5 architecture decision

- **Conditional GO** to prepare real read-only INT-W1 once scoped S2S authentication, tenant mapping and transport/correlation semantics are settled.
- **NO-GO** to claiming operational integration before a real authorized two-runtime proof.
- HTTP/JSON first: it already fits both runtimes. MCP may be a later agent-tool facade; A2A is premature without a DubBridge autonomous agent role.
- No new paid hosting, P2P certification, shared database, common JWT secret, parallel governance engine, or forced VEL coupling.

## Checkpoints

See [Fenix INT-W0 task ledger](../tasks/task_fenix_dubbridge_int_w0.md). T1/T2 evidence is source inspection; T3/T4 remain proposed design; T5 remains conditional until independently reviewed. Run repository RRI, peer-readiness, docs checks and approval gates before executing INT-W1. Existing W6 functional tests were built for Mermaid2SF and cannot certify DubBridge.
