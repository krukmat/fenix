---
doc_type: task
id: DB-FENIX-B2.C1
title: "B2.C1 Fenix SQLite durable service operation schema"
status: owner_approved_local_migration_candidate_peer_readiness_pending
phase: B2.C
week: 2026-W46
tags: [dubbridge, fenix, sqlite, idempotency, security]
fr_refs: []
uc_refs: []
blocked_by: [DB-FENIX-B2.A2]
blocks: [DB-FENIX-B2.C2]
files_affected:
  - internal/infra/sqlite/migrations/039_service_operation.up.sql
  - internal/infra/sqlite/migrations/039_service_operation.down.sql
  - internal/infra/sqlite/service_operation_migration_test.go
criticality: critical
criticality_basis: "Immutable tenant-scoped operation reservations and reversible schema constraints protect cross-tenant isolation and crash consistency."
created: 2026-11-10
completed: null
---

# B2.C1 — Embedded SQLite operation schema

**Plan**: [DubBridge consumes Fenix](../plans/dubbridge-consumes-fenix-integration.md#blocktask-plan-two-levels-only).
**Parent**: [B2.C operation task](task_dubbridge_fenix_b2c_operation.md).
**RRI**: N/A (DB-FENIX-INT cloud).
**Owner approval**: User approved C1–C4 plan in this conversation on 2026-11-10, bounded to local/feature-branch implementation; production migration, destructive rollback, credentials, public routes, CI and main merge remain forbidden.
**Peer task-readiness**: NOT RUN in Cloud; must be independently evaluated before formal closure/production certification. The earlier branch-specific T3 readiness deferral is not automatically a C1 waiver.
**Accountable**: ChatGPT Cloud. Local agent limited to Go 1.25.13 integration tests and independent reviewer.
**Source baseline**: feature/int-v2-dubbridge-consumes-fenix at b17162743ed8ee4f3e99be09e910e3b3d1c01fb6.

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

## Acceptance

- Migration 039 is loaded by Fenix embed.FS, not a copied proposal under docs/contracts.
- Unique (principal_id, operation_id) permits same ID on another principal but no duplicate on one principal.
- SQL trigger rejects wrong workspace, org, project or inactive principal, regardless of Go caller.
- Immutable scope/digest/config fingerprint cannot be updated after insertion.
- Correct status/lease/run_id constraints, no fabricated run IDs.
- Real Go 1.25.13 migrated modernc SQLite tests, FK, rollback and peer review required for PASS.
- This creates an **inert local schema only**; no live credentials, production migration, route or capability.
- If peer/readiness or Go QA cannot run in Cloud, leave PENDING instead of asserting PASS.

## Task Card

Task: DB-FENIX-B2.C1
Task file: docs/tasks/task_dubbridge_fenix_b2c1_sqlite_schema.md
Plan file: docs/plans/dubbridge-consumes-fenix-integration.md
Summary: Create reversible embedded service_operation SQL with Fenix-owned tenant binding and immutable idempotency fields.
Code affected: migrations/039_service_operation.{up,down}.sql and service_operation_migration_test.go.
Criticality: critical
Criticality basis: Tenant isolation, immutable idempotency and recovery constraints.
Effort/reasoning: High — migration, triggers, FK, reversible rollback and tests.
Recommended model: GPT-6 Cloud; independent Claude Code review.
Estimated tokens: ~7000
Pseudocode: Persist authority from active Fenix principal, restrict writes, reject cross-tenant mutations, verify local rollback.
Peer readiness review approval: reviewer=PENDING; artifact=NOT_RUN; status=PENDING
