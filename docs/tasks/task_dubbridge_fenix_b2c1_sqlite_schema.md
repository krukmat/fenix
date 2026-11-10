---
doc_type: task
id: DB-FENIX-B2.C1
title: "B2.C1 Fenix SQLite durable service operation schema"
status: readiness_pass_qa_pending
phase: B2.C
week: 2026-W46
tags: [dubbridge, fenix, sqlite, idempotency, security]
fr_refs: []
uc_refs: []
traceability_scope: integration_infrastructure_no_product_fr_uc
blocked_by: []
blocks: [DB-FENIX-B2.C2]
files_affected:
  - internal/infra/sqlite/migrations/039_service_operation.up.sql
  - internal/infra/sqlite/migrations/039_service_operation.down.sql
  - internal/infra/sqlite/service_operation_migration_test.go
criticality: critical
criticality_basis: "Migration 039 is C2's SQL authority boundary. A false-negative INSERT trigger can accept forged workspace/org/project values and inject an operation into another tenant scope; an incomplete UPDATE/DELETE trigger can rewrite or reuse a principal-plus-operation idempotency identity; and a weak CHECK can permit duplicate dispatch from invalid run or lease state."
created: 2026-11-10
completed: null
---

# B2.C1 — Embedded SQLite operation schema

Plan: [DubBridge consumes Fenix](../plans/dubbridge-consumes-fenix-integration.md).
Parent: [B2.C operation task](task_dubbridge_fenix_b2c_operation.md).
RRI: N/A under the DB-FENIX-INT cloud-agent scope.

## Start state and dependency

B2.A2 is satisfied for this branch-source task: migration 038 and its persisted
`service_principal` authority exist at owner-accepted commit
`87fd10d3e10658179d6f4ab626d33e2cf555539c`, with H1/H2/M1 Go/SQLite evidence
recorded in [issue #3](https://github.com/krukmat/fenix/issues/3). The exact
prerequisite file is
`internal/infra/sqlite/migrations/038_service_identity.up.sql`, SHA-256
`c7e63362dcfae6f6802db3801f68b30d477918ce4d60e1dbd250b3e9b9b60792`.

This clears only the technical start dependency for inert branch-local C1
implementation and QA. It is not an independent B2.A2 security or rollout
PASS. B2.A3.T4 remains separate, C4 is out of scope, and no waiver is implied.

Empty FR/UC references are intentional: this is capability-neutral S2S storage
infrastructure, not a CRM feature or configured agent workflow. Contractual
traceability is the canonical plan, parent B2.C task, neutral envelope, and the
acceptance criteria below.

## System context

```text
untrusted caller scope
  -> B2.C2 INSERT SELECT derives principal/workspace/org/project from Fenix DB
  -> B2.C1 migration 039 enforces the same persisted active principal in SQL
     and freezes operation identity plus request/config digests
  -> B2.C3 may lease once and reconcile ambiguity to indeterminate

No route, credential, capability activation, agent run, deployment or merge.
Caller owns transaction lifecycle; migration down is local/reversible only.
```

## Required implementation

- Create `service_operation` with composite primary key
  `(principal_id, operation_id)` and foreign keys to migration 038 authority.
- Store Fenix-derived workspace/org/project, capability/schema, request and
  configuration SHA-256, trace, timestamps, status, optional run and lease.
- Enforce status/run/lease consistency, including no fabricated accepted
  `run_id`, non-empty required run IDs, and no lease on any terminal status.
- `BEFORE INSERT`: require the persisted `service_principal` selected by
  `principal_id` to have `consumer='dubbridge'`, `status='active'`,
  `revoked_at_unix IS NULL`, and exact workspace/org/project equality. This
  binds every row to existing Fenix authority and rejects caller-forged scope.
- `BEFORE UPDATE`: abort any change to principal, operation ID, tenant scope,
  capability, schema, request/config digest, trace, or creation timestamp.
- Deny DELETE so a durable operation ID cannot be removed and reused until a
  separately designed retention protocol exists.
- Add scope/time and status/lease indexes. Down migration drops only 039-owned
  triggers, indexes and table in dependency order.

## Exact source candidates

- Up SQL SHA-256:
  `cd8bcc247f805d57b62420a3eb7b2cc6bb2f51a104a6f5060de30c53a0cd60b3`.
- Down SQL SHA-256:
  `a2b8a55c7ea34d9436e1dc14a44e0358ad1909fbf7118de4213ba141aaba1c1b`.
- Go migration test SHA-256:
  `0e68d9518b85c0ea4480be442bbcbe5e772793b43a072cf0635b7f7b8072bef4`.

The source already exists because this is an independent validation pass.
These hashes identify readiness inputs; they do not replace later Go execution
or exact-source code review.

## Acceptance

1. Fenix `embed.FS` loads migration 039 and down migration reverses it locally.
2. The same operation ID is unique per principal and independent across
   principals.
3. SQL rejects inactive/revoked principal or wrong workspace/org/project.
4. SQL rejects all immutable identity, scope, digest, config and trace changes.
5. SQL rejects invalid status/run/lease combinations, terminal mutation and
   operation deletion/reuse.
6. Project matching treats a persisted NULL project as the canonical empty
   string: NULL-to-empty is accepted, while any non-empty mismatch is rejected.
7. A valid active exact-scope principal can insert and perform permitted
   non-identity lifecycle updates; UPDATE and DELETE abort tests prove only the
   forbidden identity/terminal/deletion cases are blocked.
8. Down migration removes every 039 trigger, index and table without touching
   migration 038 authority; applying 039 after down succeeds again with no
   orphan object or stale constraint.
9. Real Go 1.25.13 focused migration tests, race, vet, full suite and repository
   prepush pass.
10. Independent exact-source post-code review passes.
11. Scope remains inert source only: no production migration, route, credential,
   capability, agent run, deployment, push or merge.

## High-Level Pseudocode

```text
create table keyed by persisted principal plus operation ID
bind scope and identity columns with foreign keys and CHECK constraints
before insert, select the exact active DubBridge principal and compare all scope
before update, compare every immutable old/new identity field and abort changes
before delete, abort to preserve durable idempotency identity
index scoped lookup and expired-lease reconciliation
on down, remove only migration-039 objects
test up/down/up, NULL-versus-empty project scope, valid lifecycle updates,
  cross-tenant negatives, identity/delete aborts and full state matrix
```

## Task Card

Task: DB-FENIX-B2.C1
Task file: docs/tasks/task_dubbridge_fenix_b2c1_sqlite_schema.md
Plan file: docs/plans/dubbridge-consumes-fenix-integration.md
Summary: Validate reversible embedded service_operation SQL with Fenix-owned tenant binding and immutable idempotency fields.
Code affected: internal/infra/sqlite/migrations/039_service_operation.up.sql, internal/infra/sqlite/migrations/039_service_operation.down.sql, and internal/infra/sqlite/service_operation_migration_test.go.
Criticality: critical
Criticality basis: This migration enforces tenant authority, durable request identity, and safe terminal lease state for every downstream C2 reservation.
Effort/reasoning: High - Validate triggers, foreign keys, state constraints, reversibility, and negative tests.
Recommended model: OpenAI: gpt-5.6-sol | Anthropic: claude-sonnet-4-6
Estimated tokens: ~7000
Pseudocode: Bind inserts to the exact active persisted principal, freeze identity and digests, deny deletion, enforce state/lease checks, and verify reversible migration behavior.
Peer readiness review approval: reviewer=peer-codex; artifact=logs/peer-workflow-review/task-readiness_codex_by_peer-codex_20261110T194523Z_c1.json; status=PASS
