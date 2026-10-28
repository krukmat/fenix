---
doc_type: task
id: INT-W1
title: "Real DubBridge read-only capability in Fenix"
status: in_progress
phase: integration
week: "2026-W44"
created: 2026-10-28
tags: [fenix, dubbridge, no-mocks, e2e]
blocked_by: [INT-W0]
blocks: [INT-W2]
---

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

- [x] **B3.T1** Local staging client: fixed project, HTTPS/loopback, no redirects, response limit, unknown-state/duplicate rejection, minimal Blackboard projection. Go unit tests and vet PASS **in staging**.
- [ ] **B3.T2** Register `dubbridge.review_queue.list` in Fenix ToolRegistry with read class, permission enforcement and governed retries. Code drafted, not built within full Fenix app.
- [ ] **B3.T3** Load opt-in credential/config and execute a real Blackboard -> ToolRegistry path; no mocked provider acceptance.

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

Staged code is **not a GitHub commit** and the standalone test package is **not a full application test**. No Rust security changes or database migrations have been implemented yet. Full integration E2E has not run. Prior written approval permits continuation within this frozen scope but does not waive formal RRI, independent review and local QA.
