---
doc_type: task
id: DB-FENIX-INT
title: "Fenix provider tasks for DubBridge consumer"
status: planned
phase: integration
created: 2026-10-29
tags: [fenix, dubbridge, service-consumer, agent-runtime]
---

# Fenix provider task ledger — DubBridge → Fenix

[Canonical Fenix plan](../plans/dubbridge-consumes-fenix-integration.md). The older Fenix → DubBridge capability plan is **SUPERSEDED** and its code must not be merged as this feature.

## B0 — Direction and source inventory
- [x] **B0.T1** Product owner confirmed DubBridge initiates execution, Fenix provides agents/workflows.
- [x] **B0.T2** Reverse-direction `dubbridge.review_queue.list` client and startup wiring are explicitly quarantined in their feature branch.
- [x] **B0.T3** Source inventory: existing `agents/trigger`, `agents/runs/{id}`, `workflows/{id}/execute` are user-JWT-protected; S2S restrictions unverified.

## B1 — Contract and service identity
- [ ] **B1.T1** Run-level API semantics, workspace binding, authorization, accepted request and status/error inventory.
- [ ] **B1.T2** Fenix-owned S2S principal with strict operation/workspace/allowlisted workflow policy, token revocation and rotation; security ADR.
- [ ] **B1.T3** Versioned minimal input/output contract, external correlation and idempotency; actual RRI and independent peer readiness.

## B2 — Fenix runtime implementation
- [ ] **B2.T1** Service auth middleware/actor attribution isolated from general CRM JWTs.
- [ ] **B2.T2** Governed allowlisted run execution; no unrestricted agent/workflow selection.
- [ ] **B2.T3** Run-reference/terminal-state exposure, trace/audit correlation, deny matrix and local integration tests.

## B3 — Joint contract
- [ ] **B3.T1** Publish stable provider client contract and fixtures/schema for DubBridge implementer (fixtures are not E2E).
- [ ] **B3.T2** Validate timeouts, status polling, cancellation choice, request replay and recovery.

## B4 — Real E2E and closure
- [ ] **B4.T1** Authentic DubBridge event/DB → real Fenix authenticated agent/workflow run.
- [ ] **B4.T2** Observed terminal result, stable trace and persisted execution reference.
- [ ] **B4.T3** Unauthorized principals/workspaces/operations and replay denied; no DubBridge mutations.
- [ ] **B5.T1** Local full Go QA and independent code review.
- [ ] **B5.T2** Sanitized evidence and explicit GO/NO-GO.

**Current verdict:** Only architecture and source inventory are verified. Correct-direction runtime changes are not implemented; no real E2E executed. No Github Actions reactivation, main merge or secret sharing is authorized.
