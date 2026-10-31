---
doc_type: task
id: DB-FENIX-B2A
title: "B2.A — isolated S2S service identity, no workflow"
status: readiness_proposed
date: 2026-10-31
depends_on: DB-FENIX-B1
---

# B2.A — Fenix service-principal security foundation

**Boundary:** DubBridge consumes Fenix; **no actual agent/workflow is defined or executed here**. This task is a preparation packet, not an authorization to alter Fenix's active auth middleware, issue credentials, migrate production DB or merge.

## Source constraints confirmed

- `internal/api/middleware/auth.go`: existing `AuthMiddleware` delegates to human `pkg/auth.ParseJWT` and injects `ctxkeys.UserID`/`WorkspaceID`.
- `pkg/auth/auth.go`: human JWT is HS256 under `JWT_SECRET`, not a service-token issuer.
- `internal/api/routes.go`: human-protected `/api/v1` subtree; S2S route **must be isolated**, not a bypass inside that subtree.
- `internal/infra/sqlite/migrations/001_init_schema.up.sql`: real Fenix workspace/tenant root, user table and RBAC. Fenix principal state belongs to Fenix **SQLite**, not DubBridge PostgreSQL.
- `internal/domain/tool/registry.go`: ToolRegistry permission and capability governor must remain enforced once user-defined capabilities are onboarded. B2.A does not make any capability callable.

## Leaf tasks and acceptance (each needs own RRI + reviewer)

### B2.A1 — Pure credential verification

**Dependencies:** B1.T2 decision and security reviewer readiness. **Proposed files:** separate `internal/domain/auth/service_token*.go` plus focused tests; do not modify `pkg/auth/auth.go`.

- Define opaque, high-entropy credential format (>=256 random bits), strict upper bounds, reserved S2S namespace, one-time secret output, cryptographically safe digest and constant-time comparison.
- Distinguish service principal type from human actor; no synthetic `user_id`.
- Unit tests: malformed bearer; missing/empty/whitespace; human JWT string; insufficient entropy/length; compare failure; expiry; timing-safe check with no secret logging.
- **PASS** only with actual Go unit tests and peer review; documentation cannot PASS this task.

### B2.A2 — Fenix SQLite principal/revocation lifecycle

**Depends on:** A1 + exact schema review. **Proposed files:** new versioned reversible SQLite migration, repository methods, tests.

- One registered principal with `consumer=dubbridge`, authorized Fenix workspace, independently approved DubBridge org/project or installation scope, immutable active capability allowlist (initially empty), expiry, active/revoked state, credential version, audit linkage.
- Provisioning and rotation only through authenticated operator control; verify immediate revocation and failure after rotation.
- Unit/integration tests with real temporary SQLite: FK/tenant constraints, insert/lookup, concurrent rotation, atomic revoke/refresh, migration up/down, no plaintext persistence, secret redaction.
- No production migration or operational token creation until explicit security/migration approval.

### B2.A3 — Isolated HTTP guard and authorization matrix

**Depends on:** A1+A2 + approved ADR. **Proposed paths:** `internal/api/middleware/dubbridge_s2s*.go`, `internal/api/routes.go` (new restricted subtree), tests.

- Middleware creates `ServicePrincipal` context only after validated digest, expiry, revocation, consumer, workspace and source-tenant mapping.
- Test **human JWT rejected by S2S**, **S2S bearer rejected by human API**, wrong tenant denied, revoked token denied, capability absent denied. Rate-limit by principal. Bound request body to **candidate <=64 KiB**, response <=32 KiB; enforce strict JSON type/depth and capability-specific schema at later layer.
- Unknown/disabled capability uniform response, no tenant/registry enumeration; do not create a run, dispatch DSL, invoke tools or issue 202.
- Use real `httptest` and SQLite; assert no `agent_run` row created for denied calls.

### B2.A4 — Operator security and local QA packet

**Depends on:** A1–A3 + reviewer evidence.

- Rotation/revocation runbook, safe local env/key storage, audit events, dual-principal negative tests, Go test/race/vet as relevant, one clean source review from an independent reviewer.
- Explicit HITL approval for any new migration/active auth/router change. Keep feature disabled by default.
- No GitHub Actions reactivation; only local verification.

## RRI and stop conditions

The earlier **122 RRI** was a hypothetical score for an *oversized combined implementation*, not a signed script result. Each A1–A4 leaf must calculate fresh `python3 scripts/rri.py` with **actual affected paths**, objective CC/file count/test coverage and rubric floors, then request the repo's explicit human approval when RRI >25. `docs/policies/HITL_AUTONOMY_POLICY.md` and the peer reviewer are independent gates.

**B1 self-audit is not independent peer review.** [B1 neutral test/security audit](../audit/dubbridge-fenix-b1-neutral-checkpoint-2026-10-31.md) documents current evidence. Security/peer readiness: **PENDING**. B2.A application implementation: **NOT STARTED**.

## Allowed next action

Prepare the A1 task readiness review and exact RRI, then implement A1 **only after the applicable task-level approval and independent reviewer gates**. No customer data, cloud spend, active credentials, production API or live workflows are needed for this step.


## B2.A1 approval checkpoint — 2026-10-31

[Granular B2.A1 readiness card](task_dubbridge_fenix_b2a1_credential_readiness.md) is now committed. User accepted B2.A block, and the A1 scope is explicitly narrowed to pure token generation/parsing/digest verification and tests (two new Go files, no active auth integration).

- Forecast **RRI 67 (Complex)** using the Fenix policy formula with C=1 forecast, F=1, D=5, T=3, A=1, K=5, P=5, X=2, auth security +10. This is **not** an official `scripts/rri.py` receipt because the full repo/CLI is not present in this tool container.
- Mandatory **independent peer task-readiness** **BLOCKED/NOT RUN**: local environment has neither `codex` nor `claude` executable. The repository policy does not equate plan approval with a reviewer waiver.
- **Implementation A1–A4 NOT STARTED:** no service-token Go source or tests, SQLite migration, route, credential issuance, activation or deployment was changed. Next step is to execute the recorded RRI and peer review in an authorized full checkout.

**No workflow or agent definition is part of B2.A.**
