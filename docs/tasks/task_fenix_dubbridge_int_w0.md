---
doc_type: task
id: INT-W0
title: "Fenix and DubBridge integration analysis and documentation"
status: in_progress
phase: integration
week: "2026-W44"
created: 2026-10-28
tags: [fenix, dubbridge, integration, contracts, no-mocks]
blocked_by: []
blocks: [INT-W1]
---

# INT-W0 — Fenix-owned task ledger

Plan: [Fenix ↔ DubBridge INT-W0](../plans/fenix-dubbridge-int-w0-integration.md).
DubBridge scope owner: [INT-W0 boundary](https://github.com/krukmat/dubbridge/blob/feature/int-w0-fenix-integration-boundary/docs/plan/fenix-integration-int-w0-boundary.md).

## Authorized scope

Analysis and documentation on branch `feature/int-w0-dubbridge-integration-contract`. No runtime change, no pretend test pass. **Real integration only:** a later development slice must execute against actual Fenix/DubBridge services, authenticated org/project state, and persistent data; mocks are not acceptance evidence.

## Tasks and dependencies

| Task | Deliverable | Depends on | Status at 2026-10-28 |
|---|---|---|---|
| INT-W0.T1 | Use case and acceptance | — | ANALYZED; selected read-only review queue |
| INT-W0.T2 | Source-based integration / auth inventory | T1 | ANALYZED; existing APIs and auth boundary identified |
| INT-W0.T3 | Semantic contract v1, error/identity rules | T2 | PROPOSED; must be independently reviewed |
| INT-W0.T4 | Adapter design and real E2E certification protocol | T3 | PROPOSED; no implementation |
| INT-W0.T5 | ADR/GO-NO-GO decision | T4 | CONDITIONAL GO for planning; real integration BLOCKED |

### T1 acceptance

- [x] Define an interaction with observable user value: inspect a DubBridge review queue.
- [x] Preserve DubBridge approval, rights, publishing and P2P authority.
- [x] Exclude writes from the first functional slice.

### T2 acceptance

- [x] Identify Fenix `ToolRegistry`, `PlannerExecutor` and existing M2SF HTTP pattern.
- [x] Inspect DubBridge GET review task DTO, scope and org/project membership checks.
- [x] Identify excessive privileges in default password-login token and missing dedicated S2S authorization as blockers.

### T3 acceptance (pending)

- [ ] Accept the versioned semantic input/output contract through peer review.
- [ ] Freeze Fenix/DubBridge tenant and subject mapping.
- [ ] Agree transport identity, token lifecycle, trace header contract, and safe error model with the DubBridge owner.

### T4 acceptance (pending)

- [ ] Approve exact source paths and RRI for real implementation.
- [ ] Define real local PostgreSQL/Redis and two-app E2E setup with reproducible data.
- [ ] Define negative auth/tenant tests, provider outage/retry tests and evidence capture without mock HTTP calls.

### T5 acceptance (pending)

- [ ] Review risks and record one accepted architecture decision (an accepted ADR is **not** claimed yet).
- [ ] Require evidence of scoped service identity before any live provider execution.
- [ ] Decide GO/REWORK/NO-GO for real INT-W1 implementation based on gates, not a mocked response.

## Proposed INT-W1 critical path

1. DubBridge owned authorization: create/verify a genuine least-privileged service identity, approved project mapping and bounded token lifecycle.
2. DubBridge owned trace propagation where absent; preserve actual account/org/project authorization.
3. Fenix implement the governed `dubbridge.review_queue.list` HTTP executor using existing ToolRegistry and Blackboard.
4. Run actual cross-runtime E2E against persistent DubBridge state. Capture denials, retries, trace IDs and absence of writes.
5. Close only after QA, reviewer evidence, required owner authorization and real E2E PASS.

## Gate record

Scope changes to runtime/authentication/security must be **separate tasks** with individual RRI and reviewer checks. Peer readiness review, local QA, and integrated runtime certification are **not yet run**; no development PASS is claimed. Do not commit/push to `main` or merge the feature branch without those gates.

Esfuerzo/razonamiento: High — cross-repository policy and identity design. Actual model tokens/telemetry: not measured.
