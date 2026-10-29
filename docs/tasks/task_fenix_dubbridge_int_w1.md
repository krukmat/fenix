---
doc_type: task
id: INT-W1
title: "Real DubBridge read-only capability in Fenix"
status: superseded
phase: integration
week: "2026-W44"
created: 2026-10-28
tags: [fenix, dubbridge, no-mocks, e2e]
blocked_by: [INT-W0]
blocks: [INT-W2]
---

> **SUPERSEDED / CLOSED AS WRONG DIRECTION (2026-10-29).** Previous checkboxes describe work on Fenix → DubBridge only; they are **not progress against the corrected project**. New tasks: [DubBridge → Fenix provider ledger](task_dubbridge_fenix_provider_int.md). Keep the previous Go paths quarantined and do not merge them as product integration.
# INT-W1 — Fenix task ledger

**Plan:** [INT-W1](../plans/fenix-dubbridge-int-w1-real-integration.md). DubBridge security authority is owned by the companion repository, not duplicated in Fenix.

## Block B0 — Cost controls

- [x] **B0.T1** User confirmed GitHub Actions disabled in both repos.
- [x] **B0.T2** Run counts stayed unchanged across new DubBridge documentation commits. The connector cannot query Actions configuration, so administrative verification remains operator-owned.
- [ ] Previous queued DubBridge push-review jobs must be cancelled or shown terminal before interpreting the queue as clean.

## Block B1 — DubBridge identity

- [x] **B1.T1** Dedicated-boundary security ADR proposed on DubBridge branch.
- [ ] **B1.T2** PostgreSQL principal + revocation and explicit project binding implemented and locally verified.
- [ ] **B1.T3** Real operator issuance and dedicated read route validated, with scope checks independent of mobile JWT.

## Block B2 — Traceability

- [x] **B2.T1** Local Go transport validates bounded correlation IDs and adds trace/execution headers; package-level checks performed.
- [ ] **B2.T2** DubBridge validates, persists and exposes correlated trace/audit evidence from actual API requests.

## Block B3 — Fenix read-only capability

- [x] **B3.T1** Feature-branch client (locally unit-tested): fixed project, HTTPS/loopback, no redirects, response limit, unknown-state/duplicate rejection, minimal Blackboard projection. Go unit tests and vet PASS **in staging**.
- [x] **B3.T2** Capability code versioned and **wired to `configureCrossPlatformRuntimeWithServices`** on feature branch (`b4c0fc0`); read-only descriptor, required tool permission, bounded retries and fail-closed configuration. **Source integration done, runtime/full-app QA still pending**.
- [ ] **B3.T3** Real Blackboard -> ToolRegistry call still pending actual DubBridge S2S API. Startup now loads opt-in config and rejects partial/unsafe credentials. Added a required `FENIX_DUBBRIDGE_WORKSPACE_ID` actor-workspace binding, checked before every provider request (`6cb7ed6`, `9bd66d2`).

## Block B4 — E2E

- [ ] **B4.T1** Real DubBridge DB seeded through auditable path, restricted service identity provisioned.
- [ ] **B4.T2** Compare real endpoint output with persisted review task.
- [ ] **B4.T3** Deny cross-org/project, expired/revoked/wrong-audience/human token and mutation attempts.
- [ ] **B4.T4** Verify outage recovery, bounded retries, stable trace IDs and both audit streams.

## Block B5 — Verification and closure

- [ ] **B5.T1** Required RRI/peer readiness, peer code review and full Fenix local QA PASS.
- [ ] **B5.T2** Attach sanitized exact commands, commits, counts and artifacts.
- [ ] **B5.T3** Mark GO only if genuine two-runtime E2E passes; otherwise NO-GO with explicit blockers.

## Current status

Client/executor/tests/CLI, opt-in registration and startup wiring are **committed on the feature branch** (`6bd73b8`, `b4c0fc0`, `5620f4a`, `6cb7ed6`, `9bd66d2`). Full Fenix application QA cannot run in the current isolated Go 1.23 environment; the standalone client Go tests and vet PASS but **do not validate the registered application**. No Rust security changes or database migrations have been implemented yet. Full integration E2E has not run. Prior written approval permits continuation within this frozen scope but does not waive formal RRI, independent review and local QA.

## 2026-10-29 checkpoint — Fenix wiring and workspace isolation

- `B3.T2` source status: **WIRED IN FEATURE BRANCH**, pending integrated compile/test and peer code review.
- `B3.T3` cannot pass until DubBridge has a real service issuer/verifier and protected route.
- Required operator-controlled config: `FENIX_DUBBRIDGE_URL`, `FENIX_DUBBRIDGE_TOKEN`, `FENIX_DUBBRIDGE_ORG_ID`, `FENIX_DUBBRIDGE_PROJECT_ID`, **`FENIX_DUBBRIDGE_WORKSPACE_ID`**. All five absent = integration off; partial = startup failure. Provider HTTP is HTTPS-only except loopback.
- `restrictedDubBridgeExecutor.Execute` denies a missing/mismatched `ctxkeys.WorkspaceID` before any network call. This prevents cross-workspace use even when another Fenix workspace grants `tools:dubbridge.review_queue.list` by mistake.
- Tests for opt-in registration and workspace denial were committed; their execution in the **full application** is still unverified.
- Latest reviewed Actions run totals (2026-10-29): Fenix 1200; DubBridge 3599, unchanged by these commits. Queued historic DubBridge jobs remain operator-owned.
