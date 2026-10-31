---
doc_type: task
id: DB-FENIX-INT
title: "Fenix provider for DubBridge — capability-neutral integration"
status: planned
phase: integration
rebaseline: 2026-10-31
---

# Canonical tasks — Fenix as DubBridge's agentic platform

**Binding decision:** user-defined real workflows are developed **later, as use cases arise**. None is a blocker for B1/B2/B3/B4/B5. Follow [corrected plan](../plans/dubbridge-consumes-fenix-integration.md). Legacy work on outgoing Fenix → DubBridge branches is quarantined.

## B0 — Architecture (DONE)

- [x] **B0.T1** DubBridge caller → Fenix provider, preserving DubBridge domain authority.
- [x] **B0.T2** Quarantine reverse-direction client and old proposed ADR.
- [x] **B0.T3** Source-audit existing Fenix agent/workflow and human JWT routes.

## B1 — Capability-neutral contract and security review

- [x] **B1.T1** Inventory existing APIs and record implications. **Source-only analysis satisfies B1.T1**; actual workflow behavior certification is **deferred to B6**.
- [ ] **B1.T2** Design dedicated, revocable Fenix-owned S2S principal, tenancy/workspace binding, scopes and default-deny capability policy. [Draft ADR](../decisions/ADR-DB-FENIX-S2S-PROPOSED.md) awaiting independent security acceptance.
- [ ] **B1.T3** Freeze generic envelope with `operation_id`, `capability_key`, bounded schema-versioned input, explicit capability-not-configured response, `run_id` nullable and safe status/result mapping. [Generic request schema](../contracts/dubbridge-fenix-v1-envelope-request.schema.json), [generic response schema](../contracts/dubbridge-fenix-v1-envelope-response.schema.json). Validation and review pending.
- [ ] **B1.T4** Task-level RRI/risk/security readiness and approval for scoped foundation; do **not** require running a product-specific workflow.

## B2 — Fenix provider foundation (incremental)

- [ ] **B2.A** Isolated S2S principal authentication/rotation/revocation, deny human JWT on S2S and deny S2S on CRM routes; SQLite storage and auth negative tests.
- [ ] **B2.B** Server-side capability registry, workspace and domain-scoping, versioned allowed capabilities, deny unknown/disabled ones before starting any agent; no business workflow.
- [ ] **B2.C** Durable SQLite `operation_id` registry and immutable request fingerprint, concurrency/lease/replay/crash recovery tests.
- [ ] **B2.D** Request/status/trace endpoint, safe error projection, bounded polling/reconciliation and no fabricated `run_id`.
- [ ] **B2.E** Real provider-side authentication/denial/status/idempotency tests **without** an enabled domain capability.

## B3 — Consumer interface support

- [ ] **B3.T1** Publish capability-neutral HTTP contract and operator configuration.
- [ ] **B3.T2** Verify errors, revocation, duplicate requests, timeouts and compatibility with DubBridge's Rust client.

## B4 — Real infrastructure E2E (NO real workflow prerequisite)

- [ ] **B4.T1** Two real services exchanging authenticated HTTP, observe Fenix security/operation logs.
- [ ] **B4.T2** Negative tenant/scopes/revocation and unconfigured capability rejection; no model invoked.
- [ ] **B4.T3** Idempotency, bounded errors and recovery; never interpret rejection as a successful agent run.

**B4 diagnostic condition:** Permit only a nonproduction, test-harness-enabled `integration.diagnostics` provider handler to exercise accepted-operation persistence/idempotency across both real apps. Never call it a workflow/agent or allow it in production; production unknown capabilities must remain denied.

## B5 — Foundation closure

- [ ] **B5.T1** Real Go/Rust local QA and independent security review.
- [ ] **B5.T2** Sanitized evidence, operating runbook and rollout GO/NO-GO.

## B6 — On-demand business capabilities (DEFERRED)

- [ ] **B6.T1** Discover/use-prioritize concrete user workflow.
- [ ] **B6.T2** Register capability input/output schemas, allowed tools, tenant policy and runtime mapping.
- [ ] **B6.T3** Implement actual workflow/agent, measure quality, and perform **functional** E2E with DubBridge.

## Historical evidence and scope correction (2026-10-31)

Prior review-backlog request/result JSON schemas, 18/18 and 60/60 offline assertions, later 120-case proposed tests, and guarded real-workflow probe belong to an **example-specific historical spike**, **NOT the normative foundation contract**. [Historical gate](../audit/dubbridge-fenix-b1c-gate-2026-10-31.md) incorrectly made real workflow testing a B1 blocker; superseded by owner decision. Keep source inspection findings, but do not reuse its NO-GO gate without accounting for this change.

A prior **RRI 122** was a forecast for an oversized, hypothetical S2S implementation, **not** a measured run of `scripts/rri.py`. B2 is decomposed by security and persistence boundary to keep per-leaf RRI/review auditable.

**Current state:** B0 DONE, B1.T1 SOURCE PASS, B1.T2–T4 OPEN; B2–B5 NOT IMPLEMENTED, B6 DEFERRED. **GO to scoped foundation design and risk review, NOT production rollout or unrestricted workflow execution.** GitHub Actions remain disabled; no main merge authorized.


## 2026-10-31 B1 next-block checkpoint — actual verification

- [x] **B1.T1 source inventory** accepted as the correct phase deliverable; real workflow functional run remains **B6**, not B1.
- [x] **B1.T3 draft schema validation** using *exact GitHub blobs*, verified by `git hash-object`: three neutral JSON Schema draft-2020-12 contracts plus `scripts/int_b1_envelope_selftest.py` (**5 positive / 12 negative PASS**) and `scripts/int_b1_envelope_matrix.py` (**120/120 PASS**, 14 allowed / 106 denied). Python parse/compile PASS. [Evidence and exact hashes](../audit/dubbridge-fenix-b1-neutral-checkpoint-2026-10-31.md). **Schema not yet frozen or independently approved.**
- [ ] **B1.T2 security approval**: dedicated Fenix S2S principal, SQLite trust, workspace/tenant authorization, opaque token lifecycle, capability default-deny proposal reviewed against actual middleware/JWT/tool registry code; [ADR](../decisions/ADR-DB-FENIX-S2S-PROPOSED.md) updated. **Independent security/peer approval NOT RUN**.
- [ ] **B1.T4 official RRI + reviewer receipt**: current repo policies inspected; prior monolithic RRI 122 is nonofficial forecast, not clearance. **`scripts/rri.py` and peer-workflow-review CLI not executed** on a full repo checkout. Actual credential/router/migration code must remain behind their task-specific gates.
- [x] **B2.A implementation prep** split into A1 pure secret/principal, A2 SQLite revocation, A3 isolated middleware, A4 operator QA. [Detailed task packet](dubbridge-fenix-b2a-s2s-identity.md). No application implementation, token issuance or S2S deployment.

**Gate:** GO for isolated B2.A1 readiness and RRI planning, **NO-GO** for active auth/router changes, migration, production/rollout or merge until independent review and explicit leaf approvals. Neither product needs a real workflow for the foundation.


## B2.A1 authorized scope and readiness — 2026-10-31

B2.A plan approved by owner. [B2.A1 task card](task_dubbridge_fenix_b2a1_credential_readiness.md) now specifies format, secret isolation, constant-time comparison, negative tests and **RRI 67 provisional (Complex)**. Independent peer readiness and official RRI **not run** (reviewer binaries and local full repo unavailable), so per repository governance **Go source implementation remains blocked**, and A2–A4 remain untouched. Approval of the block does not prove peer PASS or production security readiness. No real workflow requirement.
