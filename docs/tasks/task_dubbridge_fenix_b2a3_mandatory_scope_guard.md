---
doc_type: task
id: DB-FENIX-B2.A3.T2
title: "B2.A3.T2 mandatory Fenix-owned source authorization before S2S handlers"
status: ready_for_independent_readiness_review
phase: B2.A3
week: 2026-W45
tags: [dubbridge, fenix, s2s, security, authorization, middleware]
fr_refs: []
uc_refs: []
blocked_by: [DB-FENIX-B2.A3.T1]
blocks: [DB-FENIX-B2.A3.T3, DB-FENIX-B2.A3.T4]
files_affected:
  - internal/api/middleware/dubbridge_service_auth.go
  - internal/api/middleware/dubbridge_service_auth_test.go
  - internal/api/middleware/dubbridge_source_guard.go
  - internal/api/middleware/dubbridge_source_guard_test.go
  - docs/tasks/task_dubbridge_fenix_b2a3_mandatory_scope_guard.md
criticality: critical
criticality_basis: "An optional organization/project check can permit cross-tenant S2S reads or dispatch. This change modifies an authentication/authorization boundary; RRI N/A under DB-FENIX-INT cloud policy, with independent readiness, explicit owner HITL, independent post-code review and real Go QA still mandatory."
created: 2026-11-08
completed: null
---

# B2.A3.T2 mandatory Fenix-owned source authorization before S2S handlers

**Plan**: [DubBridge to Fenix S2S security foundation](../plans/dubbridge_fenix_s2s_security_plan.md#current-direction).
**Prior task**: [B2.A3.T1 readiness PASS](task_dubbridge_fenix_b2a3_middleware_readiness.md).
**Coordination**: [GitHub B2.A3 issue #5](https://github.com/krukmat/fenix/issues/5).
**Starting commit**: `69c464f0dea7fde4517199febe34ddf5177e2207` on `feature/int-v2-dubbridge-consumes-fenix`.
**RRI**: N/A — DB-FENIX-INT/cloud. Do not run the RRI calculator or require Ollama delegation.
**Readiness gate**: PENDING independent reviewer result for this exact T2 task; T1's reviewer PASS cannot be reused.
**Implementation HITL**: PENDING explicit owner authorization for T2 auth-boundary source changes. This task document does not grant it.

## Problem and bounded implementation scope

Current `DubBridgeServiceAuth` accepts a verified service bearer, places a Fenix-backed principal in private request context and invokes the supplied `next` handler. `AuthorizeDubBridgeSource` is a separate helper used only by a test handler; an application handler could omit it. Implement mandatory authorization **before `next.ServeHTTP`** for every supported S2S method, including GET/status, without mounting any route or enabling capabilities.

The only authoritative principal/workspace/DubBridge org/project binding is the Fenix-persisted identity returned by the real `AuthenticateServiceToken` store. Any JSON source, URL parameter, HTTP header or operation ID is untrusted input and may only be compared against Fenix-persisted authority; never promote caller values into trusted identity. Preserve strict, distinct human `/api/v1` JWT boundaries and default-deny. Do not make production route registration possible without an authorizing guard.

**Important dependency:** B2.C currently has only a SQL proposal; no real Go operation store or embedded production table exists. Therefore GET/status must **deny before the handler** when an authorized, persisted, principal+workspace+org/project-scoped operation lookup is unavailable. Do not invent owner data, fake a successful lookup, use operation-ID-only lookup, or create a fake operation merely to return success. A future B2.C integration requires its own task and review.

### In-scope

1. Define an explicit compulsory S2S guard/factory that cannot invoke the supplied handler with merely an authenticated but unscoped principal. Refactor the existing middleware so the exported S2S composition path always performs source authorization before dispatch; remove or internalize unsafe optional composition as needed. Existing tests that assume a bare authenticated GET reaches `next` must be updated intentionally.
2. For POST, parse the **bounded** source envelope as an untrusted assertion and check exact `product=dubbridge`, workspace (if present), organization and project against the identity loaded from Fenix; require complete source shape. For an org-only principal, only an explicitly org-scoped request is allowed. No fallback to a user JWT, caller headers, request context, or default tenant.
3. For GET/status, require a strongly typed scoped-operation resolver **before** dispatch. The resolver must return the persisted record constrained by principal ID, Fenix workspace ID, DubBridge organization ID and project ID; absent/unavailable resolver or mismatched/not-found record fails closed. Until B2.C is implemented, all GET/status checks deny. No ID-only acceptance.
4. Preserve strict Bearer syntax, duplicate-header rejection, expiry/revocation from live Fenix verification, typed-nil/error behavior, request body bound 64 KiB, safe JSON errors, service-only context key, and distinct human auth.
5. Keep cyclomatic complexity <=7 for modified code where required. Do not weaken audit/log redaction or fabricate workflow capabilities. Use isolated `httptest` with an inert downstream spy; T3 will add full modernc integration tests separately.
6. Define a stable private `AuthorizedS2SScope` only after both authentication and source guard PASS. The `next` handler must never be called for any denial. The user-visible error must not reveal existence of another org/project/operation.

### Explicitly excluded

- Changes to `internal/api/routes.go` or mounting a production S2S endpoint.
- Production migrations, real S2S tokens, installation of a capability, operation reservation, agent runs, model/tool invocation, CI reactivation, deployment or merge into `main`.
- B2.C durable operation-store implementation and B2.B capability registry enablement.
- Unrelated 26-file formatting, human rate-limit M2, toolchain upgrade, global QA debt (tracked by GitHub issue #4).

## System context

```text
 [DubBridge HTTP request] -- bearer, method, untrusted source / op_id -->
                  +----------------------------------------+
                  | ISOLATED S2S AUTH + MANDATORY GUARD    |
                  | - Fenix SQLite bearer verification    |
                  | - trusted stored principal/workspace  |
                  | - stored org/project binding          |
                  | - POST: compare untrusted source      |
                  | - GET: scoped persisted op lookup     |
                  | - no lookup available => DENY         |
                  +----------------------+-----------------+
                                         |
                         scoped, authorized context ONLY
                                         v
                [Inert S2S handler] -> [B2.B deny-by-default]
                                         |
                             [B2.C operation, NOT wired]

 [Human JWT /api/v1] --------------------> [Human middleware, separate]

 Invariants: no handler, capability lookup, operation write or agent_run on
 denial; Fenix owns principal identity; operation store (when added) owns
 scoped DB reads; caller owns DB lifecycle; guard never writes operations
 or spawns goroutines; all errors sanitize and fail closed.
```

## High-Level Pseudocode

```text
construct_guarded_s2s(verifier, source_authorizer, downstream):
    fail closed on missing/typed-nil dependencies; do not expose unguarded S2S constructor
    on request:
        reject duplicate/noncanonical bearer and unsupported method
        verify opaque service token against live Fenix SQLite
        require active dubbridge principal with stored workspace, org and project scope
        if POST:
            bound payload <= 64 KiB; parse strict source envelope
            compare untrusted product/org/project against stored identity; deny mismatch
        else if GET/status:
            require scoped persisted operation resolver
            authorize lookup using principal, workspace, org, project AND operation ID
            if resolver unavailable or record absent/mismatch: deny (no ID-only fallback)
        else: deny
        place authorized private service scope in request context
        invoke downstream handler once; never on denied path
```

## Test matrix / acceptance criteria

- POST exact persisted Fenix principal/workspace/org/project -> handler spy called exactly once (with test stub identity); wrong, missing or forged source -> 401/403, handler/registry/operation spies remain 0.
- Org-only principal rejects any project-scoped request; project-bound principal rejects absent/different project; no cross-tenant fallthrough.
- GET/status before B2.C real Go scoped lookup -> fail closed and no handler. With a test-only resolver, only an operation whose stored principal/workspace/org/project exactly matches is admitted; ID-only, mismatched or resolver-error lookup denied without leaking whether the ID exists.
- Forged header/query/context and human JWT cannot create an authorized service context; service token never populates human UserID/WorkspaceID.
- Missing/duplicate/malformed Bearer, expired/revoked token, nil/typed-nil verifier, nil handler, unrecognized method and unavailable DB -> safe error and zero downstream calls.
- Preserve request 64 KiB and 64 KiB+1 boundaries, bounded error response, one body parse (reject malformed/trailing JSON).
- All denials create zero operation rows and zero agent_run rows; use zero-call spies for T2 and confirm with real SQLite in T3. Do not claim an actual database invariant from spies alone.
- No production route mount, capability registration, real credential or migration; diff restricted to task `files_affected`.
- Run Go 1.25.13 local focused tests, `go test -race`, fmt/complexity/lint, relevant script QA with honest global FAIL attribution; no commit/push before required checks.
- Obtain **independent task-readiness PASS and explicit owner HITL approval before any auth code change**. After coding, independent post-code review PASS is required for closure, separate from QA and owner approval.

## Ready-for-review card contract

Task: DB-FENIX-B2.A3.T2
Task file: docs/tasks/task_dubbridge_fenix_b2a3_mandatory_scope_guard.md
Plan file: docs/plans/dubbridge_fenix_s2s_security_plan.md
Summary: Enforce Fenix-persisted organization/project authorization before every isolated S2S handler; fail closed for GET/status until a scoped operation resolver is available.
Code affected: internal/api/middleware/dubbridge_service_auth.go; internal/api/middleware/dubbridge_service_auth_test.go; internal/api/middleware/dubbridge_source_guard.go; internal/api/middleware/dubbridge_source_guard_test.go.
Criticality: critical
Criticality basis: Cross-tenant authorization is a security boundary; RRI N/A for DB-FENIX-INT/cloud; independent peer review and owner HITL mandatory.
Effort/reasoning: High - Requires safe composition for all methods without an existing B2.C operation store.
Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6
Estimated tokens: ~8000
Peer readiness review approval: reviewer=PENDING; artifact=PENDING; status=BLOCKED_UNTIL_ACTUAL_REVIEW