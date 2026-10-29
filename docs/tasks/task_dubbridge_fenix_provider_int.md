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
- [ ] **B1.T1** **SOURCE_REVIEW_PASS, LIVE_PENDING; LOCAL PROBE COMMITTED:** compared `agents/trigger` (creates/persists run without invoking runner) vs `workflows/{id}/execute` (runs DSL inline); inspected workspace-scoped run retrieval, human JWT and raw result exposure. [Assessment](../plans/dubbridge-fenix-b1-t1-api-runtime-assessment.md). Real runtime/agent test required for closure.
- [ ] **B1.T2** **ADR_EXPANDED, SECURITY_GATE_PENDING:** scoped Fenix-owned S2S identity, distinct human JWT trust, durable revocation/rotation and workflow allowlist. [Draft ADR](../decisions/ADR-DB-FENIX-S2S-PROPOSED.md). Independent security review, RRI and human gate pending.
- [ ] **B1.T3** **STRICT JSON SCHEMAS + SELFTEST COMMITTED; CONTRACT NOT FROZEN:** [DubBridge consumer v1 contract](https://github.com/krukmat/dubbridge/blob/feature/int-v2-dubbridge-consumes-fenix/docs/plan/dubbridge-fenix-b1-contract-v1-proposal.md) proposes async 202 + durable operation key, run status and safe result. Real RRI script + peer-readiness and contract approval pending.

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

## 2026-10-29 B1 evidence checkpoint

- **Finding:** `Orchestrator.TriggerAgent` persists run with `running` status but does not execute an agent; it is not an E2E success entrypoint on its own.
- **Finding:** `WorkflowHandler.Execute` calls `DSLRunner.Run` synchronously; the actual execution may finish or become pending/delegated. A 200 response is not a guaranteed terminal success.
- **Finding:** generic user JWT currently protects `/api/v1/*`; S2S credential and narrow service route do not yet exist.
- **Finding:** generic run DTO includes reasoning and tool calls. The service-facing result must omit them.
- **Decision for next phase:** reuse DSL execution engine behind a new strictly scoped S2S facade with durable deduplication, not a thin proxy to end-user endpoints.
- **Blocks:** current connection cannot boot provider or execute local Go app and RRI/peer scripts. The completed work is documentary source evidence and a **proposed** ADR/contract, **not B1 security acceptance**.

## 2026-10-29 additional deliverables

- [x] **B1.T1 evidence-preparation:** `scripts/int_b1_fenix_live_probe.py` committed; it requires disposable read-only workflow, local app and operator test JWT, returns sanitized run ID/status evidence. **Live probe not executed**.
- [x] **B1.T2 draft hardening:** ADR now recommends Fenix-issued opaque revocable service credential with isolated middleware, single Fenix workspace/immutable workflow version, and principal-bound **single DubBridge org/project pair**. This is a **proposal pending independent security review**, not accepted.
- [x] **B1.T3 schema drafting:** `docs/contracts/dubbridge-fenix-v1-{request,run}.schema.json` and `scripts/int_b1_contract_selftest.py` committed with positive/negative schema scenarios. **No full repo QA or real E2E**.
- [ ] **Required gates:** execute local probe on real disposable app; run `python3 scripts/int_b1_contract_selftest.py` in a checked-out Fenix branch with `jsonschema`; run required `scripts/rri.py` and peer-readiness/security review; approve ADR/contract. Do not record PASS until outputs are captured.

**B1 status: PARTIAL / NO-GO TO RUNTIME AUTH MERGE.** Old reverse-direction code remains absent from this branch; no Actions, main or production changes.

## 2026-10-29 executed offline test checkpoint (B1 continuation)

- **B1.T3 source + LOCAL_OFFLINE_PASS:** committed and locally ran `scripts/int_b1_contract_selftest.py` (**4 positive + 14 negative PASS**) and `scripts/int_b1_contract_status_matrix.py` (**60/60 PASS; 10 allowed + 50 rejected**). Tested **exact committed Git blobs**, verified for both test scripts and both JSON schemas. [Evidence](../audit/dubbridge-fenix-b1-contract-check-2026-10-29.md).
- Fixed an observed contract flaw: previously combinations such as `succeeded + error_code` or `failed + valid advisory result` were accepted by the candidate result schema. New schema explicitly separates success, in-progress and error states.
- `scripts/int_b1_fenix_live_probe.py` hardened to disable environment proxying of its disposable JWT and reject invalid response shapes. **Live run NOT EXECUTED**; syntax/behavior of updated probe not independently run in a full Fenix checkout.
- **B1.T2** security ADR remains proposed. Its scope includes principal-scoped DubBridge org/project, Fenix workspace + immutable workflow, separate human/service trust, revocation and dynamic tool permissions; **RRI, peer/security review and acceptance still pending**.
- **B1.T1** source reviewed but live workflow/model behavior unverified.
- **B1 verdict:** planning/contract preparation advanced, but **NOT CLOSED**, no S2S implementation or real E2E. Never treat 78 offline schema assertions as live agent validation.
