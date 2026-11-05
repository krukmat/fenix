---
doc_type: task
id: DB-FENIX-B2B-REGISTRY
title: "B2.B — capability-neutral default-deny S2S registry"
status: candidate_no_runtime
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
