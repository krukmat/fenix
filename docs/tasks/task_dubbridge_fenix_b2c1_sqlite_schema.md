---
doc_type: task
id: DB-FENIX-B2.C1
title: "B2.C1 Fenix SQLite durable service operation schema"
status: readiness_needs_changes_doc_revision_peer_rerun_pending
phase: B2.C
week: 2026-W46
tags: [dubbridge, fenix, sqlite, idempotency, security]
fr_refs: []
uc_refs: []
traceability_scope: integration_infrastructure_no_product_fr_uc
blocked_by: [DB-FENIX-B2.A2]
blocks: [DB-FENIX-B2.C2]
files_affected:
  - internal/infra/sqlite/migrations/039_service_operation.up.sql
  - internal/infra/sqlite/migrations/039_service_operation.down.sql
  - internal/infra/sqlite/service_operation_migration_test.go
criticality: critical
criticality_basis: "Incorrect scope, leases or journal identity could compromise cross-tenant isolation and idempotency; critical independent verification is required, not yet achieved."
created: 2026-11-10
completed: null
---

# B2.C1 — Embedded SQLite operation schema

**Readiness dependency, visible without packet compaction:** B2.A2 persisted identity/migration 038 is **owner-accepted for branch-source QA**, with H1/H2/M1 Go evidence at `87fd10d` in [issue #3](https://github.com/krukmat/fenix/issues/3). This is **not** an independent B2.A2 security PASS or rollout approval. B2.C1 is an inert S2S infrastructure task; `fr_refs: []` and `uc_refs: []` are intentional because no product CRM use case is delivered. Technical requirements are the integration plan, S2S envelope and acceptance criteria below.

**Separate gates:** task-readiness evaluates whether C1 scope, dependencies and acceptance checks are sufficient to execute; Go 1.25.13 tests, vet, race, prepush and independent *post-code* review are required for **closure**, not evidence of a test execution already achieved at task-readiness. The earlier `needs_changes` verdict remains in force until a genuine new review. No waiver is implied.


**Plan**: [DubBridge consumes Fenix](../plans/dubbridge-consumes-fenix-integration.md#blocktask-plan-two-levels-only).
**Parent**: [B2.C operation task](task_dubbridge_fenix_b2c_operation.md).
**RRI**: N/A (DB-FENIX-INT cloud).
**Owner approval**: User approved C1–C4 plan in this conversation on 2026-11-10, bounded to local/feature-branch implementation; production migration, destructive rollback, credentials, public routes, CI and main merge remain forbidden.
**Peer task-readiness**: At source HEAD `6772905649d8390b4943273f4e52416696194087`, the owner reports a local `needs_changes` verdict citing B2.A2 dependency evidence and empty FR/UC without rationale; the local reviewer artifact was not supplied to Cloud. This document corrects those documentation gaps but **does not turn the verdict into PASS**. Re-run an independent readiness review on the new document SHA before any task closure. The earlier T3 readiness deferral is not a C1 waiver.
**Accountable**: ChatGPT Cloud. Local agent limited to Go 1.25.13 integration tests and independent reviewer.
**Source baseline**: feature/int-v2-dubbridge-consumes-fenix at b17162743ed8ee4f3e99be09e910e3b3d1c01fb6.

## Dependency and requirements traceability (readiness remediation, 2026-11-10)

**Dependency is not absent:** [B2.A2 remediation task](task_dubbridge_fenix_b2a2_security_remediation.md), [remediation audit](../audit/dubbridge-fenix-b2a2-remediation-2026-11-06.md), and [issue #3](https://github.com/krukmat/fenix/issues/3) show owner-accepted **scoped** B2.A2 source and Go/SQLite evidence on commit `87fd10d3e10658179d6f4ab626d33e2cf555539c`: H1 atomic credential rotation/rollback, H2 skew-safe revoke, M1 embedded SQLite **038** and locally tested down, focused race x3. Migration 039's foreign-key principal table therefore has an existing **feature-branch** predecessor. Historical [independent B2.A2 FAIL](../audit/dubbridge-fenix-b2a2-independent-security-review-2026-11-06.md) was remediated in source, but a fresh independent security acceptance is **not documented**; [B2.A3.T4 #8](https://github.com/krukmat/fenix/issues/8) also remains independently unreviewed. Thus `blocked_by: [DB-FENIX-B2.A2]` is **satisfied only for source prerequisite / branch-local QA**, not for formal security closure or rollout. Readiness reviewer must evaluate this precise scope and cannot infer global PASS or a policy waiver.

**Why `fr_refs: []` / `uc_refs: []` is intentional:** B2.C1 is a capability-neutral infrastructure migration providing a durable principal-scoped operation journal; it does **not** implement a CRM product requirement, business use case or actual agent workflow. The current `docs/requirements.md` FR/UC inventory is product-facing; assigning a convenient `UC-G1` or fabricated FR would falsely claim a delivered governance scenario. Direct contractual traceability is:
- [DB-FENIX-INT canonical block B2.C](../plans/dubbridge-consumes-fenix-integration.md#blocktask-plan-two-levels-only) and [parent B2.C acceptance](task_dubbridge_fenix_b2c_operation.md).
- [Neutral S2S operation envelope](../contracts/dubbridge-fenix-v1-envelope-request.schema.json), especially `operation_id`, `source`, `capability_key`, schema and trace identity.
- [B2.A2 persisted identity](task_dubbridge_fenix_b2a2_security_remediation.md), [B2.C2 reservation](task_dubbridge_fenix_b2c2_operation_store.md) and [B2.C3 recovery](task_dubbridge_fenix_b2c3_recovery.md).
- The C1 acceptance criteria below are the testable requirements. Product FR/UC links will be added only if a genuine product-level capability is later specified; they are **not mandatory for an inert S2S foundation task**.

**Readiness state**: local `needs_changes` reported at `67729056`; this is a document-remediation candidate awaiting an actual new independent verdict and Go 1.25.13 QA. No implementation activation or independent reviewer PASS is claimed.

## System Context

```text
DubBridge request (UNTRUSTED JSON org/project) -> Fenix S2S bearer guard
                                                     |
                                      verified Fenix persisted principal
                                                     |
                        +----------------------------v-------------------+
                        | B2.C2 ServiceOperationStore, writes with       |
                        | INSERT SELECT from active service_principal    |
                        | and cannot elevate caller scope                |
                        +----------------------------+-------------------+
                                                     |
                              B2.C1 embedded SQLite migration 039
                              service_operation (PK principal, operation)
                              immutable workspace/org/project + digest
                                                     |
                        B2.C3 lease/indeterminate; B2.C4 bounded status
  Never creates/executes agent_run, mounts routes, enables capabilities or
  allocates credentials. Caller owns DB lifecycle. Destructive down LOCAL ONLY.
```

## High-Level Pseudocode

```text
create service_operation with composite principal+operation uniqueness
foreign-key principal and Fenix workspace
persist Fenix-derived scope, request and configuration SHA-256 digests
enforce accepted state with NULL run_id and safe status/lease invariants
on INSERT require active stored service_principal matching exact scope
on UPDATE deny alteration of immutable identity/scope/digest
index scope/time and status/lease
on local rollback remove triggers/indexes/table, leaving principal/store intact
integration test migration version, constraints, wrong tenant, rollback
```

## Cloud C1 bounded source correction — 2026-11-10

**Candidate change**: Embedded migration 039 now enforces `status='accepted' => run_id IS NULL`, and a running/approval/handoff/success state requiring a run ID cannot use an empty string. `TestB2CServiceOperationSchemaConstraints` was extended to reject an accepted operation with a synthetic `run_id` and an empty running `run_id`.

**Reason**: the previous CHECK required a non-NULL run ID only after leaving `accepted` but allowed callers to fabricate one *while still accepted*, contrary to C1's inert/no-fake-run invariant. Only the existing feature-branch 039 up SQL and its test changed; no existing production migration was applied.

**Verification**: Static source change committed. An **isolated Python sqlite3 in-memory smoke** of the exact new CHECK-expression logic (not the embedded full 039 migration) exercised six state/run-id cases: **6/6 PASS** on ChatGPT's Linux environment. Real Go 1.25.13/modernc migration tests, independent readiness and peer review **NOT RUN** at this checkpoint. The reported prior C1 readiness `needs_changes` has not become a PASS.

## Acceptance

- Migration 039 is loaded by Fenix embed.FS, not a copied proposal under docs/contracts.
- Unique (principal_id, operation_id) permits same ID on another principal but no duplicate on one principal.
- SQL trigger rejects wrong workspace, org, project or inactive principal, regardless of Go caller.
- Immutable scope/digest/config fingerprint cannot be updated after insertion.
- Correct status/lease/run_id constraints, no fabricated run IDs.
- Real Go 1.25.13 migrated modernc SQLite tests, FK, rollback and peer review required for PASS.
- This creates an **inert local schema only**; no live credentials, production migration, route or capability.
- If peer/readiness or Go QA cannot run in Cloud, leave PENDING instead of asserting PASS.

## C1 terminal lease integrity hardening — 2026-11-10

**Source candidate:** Embedded migration 039 adds a CHECK rejecting worker leases on all terminal outcomes (`succeeded`, `completed_with_warnings`, `abstained`, `failed`, `denied`, `indeterminate`). This prevents a SQL write from leaving/reintroducing lease metadata after the outcome is final. Existing C3 `MarkIndeterminate` and `ReconcileExpiredLeases` already clear lease fields before recording ambiguity.

**Regression candidate:** the Go/modernc migration test now attempts to transition to every terminal state while retaining a worker lease, and checks that adding a lease to an already-indeterminate operation is denied. An **isolated, reduced Python sqlite3 reproduction** of the CHECK matrix covered 14 cases with 14 PASS (SQLite 3.46.1). This is *not* the full embedded 039 migration nor Go 1.25.13 execution; readiness, Go tests/race/vet/prepush, independent code review and production rollout remain PENDING.

## Task Card

Task: DB-FENIX-B2.C1
Task file: docs/tasks/task_dubbridge_fenix_b2c1_sqlite_schema.md
Plan file: docs/plans/dubbridge-consumes-fenix-integration.md
Summary: Create reversible embedded service_operation SQL with Fenix-owned tenant binding and immutable idempotency fields.
Code affected: migrations/039_service_operation.{up,down}.sql and service_operation_migration_test.go.
Criticality: critical
Criticality basis: Tenant isolation, immutable idempotency and recovery constraints.
Effort/reasoning: High — migration, triggers, FK, reversible rollback and tests.
Recommended model: Cloud author; independent peer review assigned by docs/playbooks/AGENT_WORKFLOW_GUIDE.md (no model-specific integration dependency).
Estimated tokens: ~7000
Pseudocode: Persist authority from active Fenix principal, restrict writes, reject cross-tenant mutations, verify local rollback.
Peer readiness review approval: reviewer=LOCAL_REVIEWER_UNSPECIFIED; artifact=NOT_PROVIDED_TO_CLOUD; status=NEEDS_CHANGES_AT_67729056_RECHECK_PENDING
