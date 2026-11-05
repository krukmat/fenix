---
doc_type: task
id: DB-FENIX-B2C
title: "B2.C — durable S2S operation idempotency proposal"
status: sqlite_proposal_only
phase: integration
week: "2026-W45"
tags: [dubbridge, fenix, s2s, sqlite, idempotency]
fr_refs: []
uc_refs: []
blocked_by: [DB-FENIX-B2A, DB-FENIX-B2B]
blocks: [DB-FENIX-B2D]
files_affected:
  - docs/contracts/dubbridge-fenix-b2c-operation-proposal.up.sql
  - docs/contracts/dubbridge-fenix-b2c-operation-proposal.down.sql
  - scripts/int_b2c_operation_sqlite_smoke.py
criticality: critical
criticality_basis: "Cross-service idempotency, crash recovery and multi-tenant consistency"
created: 2026-11-05
completed:
---
# Task DB-FENIX-B2C

**Plan**: [Canonical DubBridge → Fenix plan](../plans/dubbridge-consumes-fenix-integration.md)

## System Context

```text
Authorized DubBridge S2S principal -> authorized capability (B2.B) -> B2.C operation
                                 -> unique (principal_id, operation_id) + digest
                                 -> status + immutable true agent_run_id (future B6)
```

## Tasks and acceptance

- B2.C1: principal-scoped idempotency key, unique SQLite operation row, exact digest equality on replay; same key/different request gives conflict.
- B2.C2: run ID nullable during accepted state, required when running and on success. No fake workflow or agent runs.
- B2.C3: crash/lease uncertainty must enter indeterminate; no automatic duplication of side effects.
- B2.C4: complete real Go integration + testable deployment migration **only after** A2/A3 security, full QA and independent peer review.

Proposed schema lives **only under docs/contracts**; it is NOT in embedded
Fenix migrations and does not activate any route. The isolated SQLite selftest
tests the schema and a candidate SQL sequence, not a production reconciler.
Do not claim exact-once side effects or a real integrated E2E test.

**RRI: N/A** under scoped cloud-agent owner policy. No real workflows in B2.C.
