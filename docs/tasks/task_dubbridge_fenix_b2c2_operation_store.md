---
doc_type: task
id: DB-FENIX-B2.C2
title: "B2.C2 atomic Go SQLite service operation reservation and canonical request fingerprint"
status: readiness_pass_qa_pending
phase: B2.C
week: 2026-W46
tags: [dubbridge, fenix, sqlite, idempotency, go]
fr_refs: []
uc_refs: []
traceability_scope: integration_infrastructure_no_product_fr_uc
blocked_by: [DB-FENIX-B2.C1, DB-FENIX-B2.B]
blocks: [DB-FENIX-B2.C3]
files_affected:
  - internal/domain/integration/service_operation_store.go
  - internal/domain/integration/service_operation_digest.go
  - internal/domain/integration/service_operation_store_test.go
criticality: critical
criticality_basis: "Concurrent idempotency and strict tenant binding avoid duplicate operation acceptance and cross-workspace reads."
created: 2026-11-10
completed: null
---

# B2.C2 — Atomic operation reservation and canonical digest

**Plan**: [Canonical DubBridge → Fenix](../plans/dubbridge-consumes-fenix-integration.md#blocktask-plan-two-levels-only).
**Parent**: [B2.C](task_dubbridge_fenix_b2c_operation.md).
**RRI**: N/A (DB-FENIX-INT Cloud).
**Approval**: Owner approved C1–C4 proposal in conversation 2026-11-10. No production credentials, gateway/route, deployment, or main merge authorized.
**Readiness**: Independent task-readiness PASS at the current validation checkpoint; source remains a branch candidate until local Go 1.25.13 and exact-source reviewer gates complete.
**Accountable**: ChatGPT Cloud; local agent Go and independent peer gate only.

## Requirements traceability and security dependencies (2026-11-10)

**FR/UC rationale:** `fr_refs: []` and `uc_refs: []` intentionally do not claim any CRM product use case. B2.C2 is a capability-neutral S2S storage primitive, not a configured agent, workflow or business operation. Its executable contract is the [DB-FENIX-INT B2.C plan](../plans/dubbridge-consumes-fenix-integration.md#blocktask-plan-two-levels-only), the [neutral S2S envelope](../contracts/dubbridge-fenix-v1-envelope-request.schema.json), the [B2.C parent task](task_dubbridge_fenix_b2c_operation.md) and the acceptance matrix below. Do not map `UC-G1` simply because it mentions replay/audit: no agent-run business use case is executed here.

**Dependencies:** C1's migration 039 exists as the branch source prerequisite; B2.B's registry is default-deny and its durable operator policy B2.B4 remains a separate rollout concern. [B2.A2 issue #3](https://github.com/krukmat/fenix/issues/3) confirms the owner-accepted feature-branch persisted principal baseline at `87fd10d`; independent security certification remains pending. C2 readiness is independently PASS, while QA and exact-source review remain separate. **RRI: N/A — DB-FENIX-INT/cloud**.

## System Context

```text
Verified Fenix principal + resolved capability (not caller JSON)
   -> operation request normalized (strict JSON; depth/bytes bound)
   -> SHA-256 request + immutable capability config SHA-256
   -> [BEGIN IMMEDIATE Fenix SQLite transaction]
      SELECT exact active Fenix principal/workspace/org/project
      INSERT SELECT Fenix-owned authority ON CONFLICT DO NOTHING
      SELECT same scope + compare digest/version/config fingerprint
      COMMIT
   -> NEW accepted / SAME replay / DIFFERENT conflict
   -> C3 recovery, C4 status-only lookup (not active)
```

## High-Level Pseudocode

```text
parse one strict JSON envelope with unique nested keys and bounded depth
normalize maps into sorted JSON, hash bytes; never persist raw input
validate request id, scope, trace, version and nonzero fingerprints
resolve enabled exact capability in registry; unknown means deny
begin immediate serialized SQLite transaction
insert authority SELECT from live Fenix service_principal
lookup operation with principal/workspace/org/project predicates and live principal
same identity/capability/config/request hash => idempotent replay
different hash/config/contract => typed conflict, no overwrite
commit; no agent execution, HTTP response or capability auto-registration
```

## C2 follow-up: caller-supplied configuration isolation — 2026-11-10

**Additional regression candidate:** `TestB2CReserveNewReplayConflictAndScope` now provides a forged caller-side `ConfigSHA256` and demands an unchanged idempotent replay with the stored Fenix-derived fingerprint. Existing `Reserve` code always recomputes the fingerprint from an enabled registry binding and does not require a production code change for this case.

**Verification limits:** published test source only. Real Go 1.25.13, race and peer review remain `NOT_RUN`.

## Cloud C2 bounded source correction — 2026-11-10

**Candidate change**: `sameOperationRequest` now compares immutable `trace_id` in addition to scope, operation/capability/schema, request SHA and capability configuration SHA. `TestB2CReserveNewReplayConflictAndScope` now asserts conflict when the same operation ID/request digest is replayed with a different trace ID.

**Reason**: the stored trace ID is declared immutable by migration 039. The prior replay comparator omitted it, so a caller passing an unchanged digest with a different trace could receive a replay success without detecting the mismatch. This is defense in depth; the future ingress still must bind digest to the actual approved envelope (B2.D), and no ingress is active today.

**Verification**: code/test candidate committed; Go 1.25.13 focused/full/race QA and independent task-readiness/post-code review **NOT RUN**. The change does not claim real agent dispatch, an enabled production registry or rollout.

## Acceptance and limitations

- Registry starts empty; no operation accepted until explicit test-only/operator registration.
- Principal source scope is fetched from the real DB in INSERT SELECT, not caller JSON; DB trigger checks it.
- Concurrent same-key inserts create one row; different payload/config yields `ErrServiceOperationConflict` and unchanged existing record.
- Replays after principal revoke denied; cross-principal operation IDs independent.
- Canonical JSON excludes formatting and key order variation, rejects duplicate nested keys/trailing JSON and >64KiB or >32 levels. This is **not RFC 8785 numeric canonicalization**, so e.g. `1` and `1.0` intentionally have different fingerprints.
- No production S2S gateway/route, credential, migration execution or real workflow.
- Go 1.25.13 QA and independent post-code review are separate closure gates; never infer their result from Cloud code publication or readiness.

**Additional security regression:** `TestB2CReserveRejectsRevokedPrincipal` covers both replay of an existing operation and reservation of a new operation ID after the persisted Fenix principal is revoked. Both must return the same generic denial, and the operation journal row count must not change. The prior C2 test only covered a disabled registry binding, so principal revocation was not tested directly at the reservation boundary. This test is **authored, NOT_RUN with Go**.

## C2 strict canonical digest input — 2026-11-10

**Source candidate:** `CanonicalOperationDigest` now rejects byte sequences that are not valid UTF-8 before tokenizing JSON. `encoding/json` otherwise replaces invalid bytes with the Unicode replacement character, so distinct malformed byte streams can collapse onto an identical canonical digest. The existing normalized/duplicate-key/depth/node/size limits remain unchanged; this is not an RFC 8785 conversion.

**Regression candidate:** `TestB2CCanonicalDigestLimitsAndDuplicateKeys` now feeds a malformed UTF-8 JSON string and requires `ErrServiceOperationInvalid`. No live ingress, capability registration or execution was added. Actual Go 1.25.13 and independent peer verdict are **NOT_RUN**.

## Task Card

Task: DB-FENIX-B2.C2
Task file: docs/tasks/task_dubbridge_fenix_b2c2_operation_store.md
Plan file: docs/plans/dubbridge-consumes-fenix-integration.md
Summary: Atomic exact-scope durable SQLite operation reservation; deterministic request digest and replay/conflict semantics.
Code affected: internal/domain/integration/service_operation_store.go, service_operation_digest.go, tests.
Criticality: critical
Criticality basis: Durable idempotency and tenant authorization before any dispatcher.
Effort/reasoning: High — concurrent writer, canonical input and failure paths.
Recommended model: Cloud author; independent peer review assigned by docs/playbooks/AGENT_WORKFLOW_GUIDE.md (no model-specific integration dependency).
Estimated tokens: ~9000
Pseudocode: Verify capability, derive Fenix principal scope, insert under transaction, compare fingerprints, return record/replay/conflict without running any agent.
Peer readiness review approval: reviewer=peer-codex; artifact=logs/peer-workflow-review/task-readiness_codex_by_peer-codex_20261110T194523Z_c2.json; status=PASS
