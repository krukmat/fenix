---
doc_type: task
id: DB-FENIX-B2B-REGISTRY
title: "B2.B — capability-neutral default-deny S2S registry"
status: b2b4_cloud_source_candidate_qa_peer_pending
phase: integration
week: "2026-W45"
tags: [dubbridge, fenix, s2s, capability, security]
fr_refs: []
uc_refs: []
blocked_by: [DB-FENIX-B1]
blocks: [DB-FENIX-B2C]
files_affected:
  - internal/domain/integration/s2s_capability_registry.go
  - internal/domain/integration/s2s_capability_registry_test.go
criticality: critical
criticality_basis: "Authorization scope boundaries; independent peer review required"
created: 2026-11-05
completed:
---
# Task DB-FENIX-B2B-REGISTRY

## B2.B4 — ChatGPT Cloud implementation checkpoint (2026-11-11)

**Owner:** ChatGPT Cloud, **RRI:** N/A. Existing branch `feature/int-v2-dubbridge-consumes-fenix`. **Result:** B2.B4.1–B2.B4.5 authored and published as an **inert source/test candidate**, not Go QA or security PASS. The feature-branch migration 040 is **not deployed to production** and can only be rolled back with separate HITL approval outside local test use.

| Subtask | Actual Cloud code | Status |
|---|---|---|
| **B2.B4.1 — Fenix governance boundary** | Explicit `S2SPolicyOperatorAuthorizer` checks operator's active Fenix `user_account` workspace and existing `PolicyEngine.CheckActionPermission("s2s_capabilities","manage")`. No role grant or endpoint is added. ToolRegistry's execution/runtime governor remains independent; admission grants **never** bypass tool/execution approvals | Authored |
| **B2.B4.2 — Durable policy** | Migration `040_s2s_capability_policy`: versioned disabled-by-default capability+schema, active-principal constraints, at most one enabled version, immutable identity, revision increments, SQLite-triggered append-only operator audit, no deletion. Read-only policy store is default; `Provision/SetEnabled` require explicit authorized operator | Authored |
| **B2.B4.3 — Capability input schema** | Strict bounded JSON-schema subset, reject unknown keywords, unknown fields, duplicate keys, bad Unicode, invalid type/depth/length; schema hash checked against SQLite. Managed S2S registry reads fresh SQLite policy, no direct `Register/Disable` bypass. C2 hash binds schema and policy; **revalidates current enabled policy under the SAME `BEGIN IMMEDIATE` SQLite transaction** before new reservations **or replays** | Authored |
| **B2.B4.4 — Negative QA source** | Migration constraints/audit/version tests; persisted policy scope, roles, revoked principal, disabled capability, version change, schema rejection, forged binding, nil managed-store and mid-request policy disable; detached HTTP + real migrated SQLite tests | Tests authored; **NOT_RUN** |
| **B2.B4.5 — Cloud source consistency** | **46/46 read-only structural source assertions PASS** (schema, migration, auditable policy, Fenix RBAC/ABAC bridge, pre-insert policy guard, HTTP default-deny, test presence). No compilation/SQL execution evidence is implied | Source audit complete |

**API composition:** `NewDormantS2SAdmissionFromPolicies(db)` is the safe default detached handler backed by the read-only durable store; an empty policy table denies everything. No registry seeding, JWT/credential creation, HTTP route mount, worker/model, C4 GET/status, CI or deployment.

**Important gates remaining:** Fenix Go 1.25.13 compile, gofmt, focused/full/race/vet/prepush, SQLite execution, independent code/security review **NOT_RUN** (Cloud lacks full toolchain/private checkout; no local tokens). Operator authorization and schema management have no live management route; policy grants are **not** provisioned. T4 security #8 and explicit activation/real B4 E2E remain separate.

## Current authoritative cloud status — 2026-11-05

Pure Go registry and tests **COMMITTED** in assigned feature branch; empty by default, scoped, fail-closed and concurrency-tested (50 repeats; 8 race repeats; 93.3% isolated coverage). Full project QA and independent review PENDING. Registry is in-memory and not connected to route dispatch or durable operator configuration. **B2.B4 OPEN**. Evidence: [B2 cloud audit](../audit/dubbridge-fenix-b2-cloud-checkpoint-2026-11-05.md).


**Plan**: [DubBridge consumes Fenix — canonical plan](../plans/dubbridge-consumes-fenix-integration.md)

## System Context

```text
DubBridge S2S client -> Fenix service principal (B2.A) -> capability gate (B2.B)
                                                          |
                                                          +-- unconfigured/disabled -> DENY
                                                          +-- registered/authorized -> future B2.C/B6
```

The registry is a capability-neutral policy *precondition*, not Fenix ToolRegistry,
not an agent executor and not a privileged API. The initial state is empty;
registry state is not wired into Fenix's HTTP routes. No workflows or models
need to be defined at this phase.

## Tasks

- **B2.B1** Implement isolated logical capability metadata and workspace/service-principal binding, including schema version and explicit active flag.
- **B2.B2** Deny by default for unknown/disabled capability; fail closed on principal, workspace, version or identifier mismatch.
- **B2.B3** Test registration validation, permitted/denied lookups, defensive copy and concurrent reads; no runtime execution.
- **B2.B4** Review effective ToolRegistry/tool-policy boundary, and decide durable capability settings before production.

## High-Level Pseudocode

```text
new_registry() -> empty, all capabilities denied
register_from_operator(config):
   validate canonical key, schema version, principal, workspace, handler ID
   atomically store scoped immutable copy
lookup(principal, workspace, capability, schema_version):
   lookup exact tuple under read lock
   return only if enabled and exact scope/schema matches, otherwise unavailable
```

## Acceptance

Go unit tests and race detector PASS against isolated module; no registered
capabilities by default; no arbitrary workflow ID selection or model execution.
Full Fenix Go QA and independent peer/security review are not available through
this cloud-only connector and remain **PENDING**. No merge to main, no credentials,
no production registry and no GitHub Actions. **RRI: N/A** by owner decision.
