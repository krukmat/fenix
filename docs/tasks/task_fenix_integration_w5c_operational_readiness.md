---
doc_type: task
id: W5-C
title: Durable evidence lifecycle and integration observability
status: in_progress
phase: integration
week: W5
tags: [fenix, w5, vel, outbox, reconciliation, checkpoint, observability]
fr_refs: []
uc_refs: []
blocked_by: [W5-B]
blocks: [W5-FINAL, W6]
files_affected:
  - internal/domain/evidence/
  - internal/infra/integration/vel/
  - internal/infra/integration/
  - internal/api/
  - internal/infra/sqlite/migrations/
  - docs/plans/
created: 2026-09-21
completed:
---

# Task W5-C — Durable evidence lifecycle and integration observability

**Plan**: [W5-C Operational Readiness](../plans/fenix-integration-w5c-operational-readiness.md#w5-c--operational-readiness)

## Scope

Implement G07-G10 as one coherent operational lifecycle so Fenix has one durable source of truth
for evidence delivery, checkpoint progression, partitioning and metrics.

## System Context

```text
ToolRegistry
   |
   | governed CapabilityEvidenceRequest
   v
Durable Runtime Recorder
   |
   +--> SQLite evidence_delivery
   |
   +--> VEL RuntimeSink ----> VEL event append / lookup
                |
                v
       Evidence Lifecycle Worker
                |
                +--> VEL checkpoint
                +--> VEL bundle + verify
                |
                v
        ProofReference state
                |
                +--> Fenix audit
                +--> integration metrics
```

Upstream triggers:
- successful/failed governed external capability execution when evidence was planned;
- server restart with unfinished durable evidence rows;
- periodic lifecycle reconciliation ticks.

Downstream consumers:
- VEL external-authority append/idempotency lookup;
- VEL checkpoint/bundle/verifier;
- Fenix audit metadata;
- agent-safe evidence status;
- Prometheus-compatible metrics.

Key invariants:
- never replay the governed business capability during evidence reconciliation;
- `execution_id` remains the VEL idempotency key;
- full VEL bundles are transient and never persisted by Fenix;
- background goroutines are owned by `RouterRuntime` and stop with its context;
- durable state is written before the first provider delivery attempt;
- workspace-v1 stream partitioning is deterministic and version-frozen.

## High-Level Pseudocode

```text
on governed capability evidence:
    envelope = build minimized envelope using workspace-v1 stream policy
    upsert durable row(envelope, pending_record)

    result = try append-or-idempotency-reconcile
    if recorded:
        persist compact proof + pending_checkpoint
        return pending_checkpoint without repeating business action
    if provider outcome is retryable/unknown:
        persist indeterminate + next retry
        return indeterminate as durable/accepted
    if deterministic contract failure:
        persist terminal diagnostic
        return error

background reconciliation loop:
    load due rows
    for each pending/indeterminate row:
        lookup existing event first when append outcome was uncertain
        if missing and safe:
            append same envelope with same execution_id
        persist recorded proof

    group recorded rows by stream
    for each stream:
        create checkpoint
        fetch verification bundle
        verify through VEL
        update all rows covered by checkpoint tree_size
        persist verified or verification_failed

metrics:
    observe provider calls + latency
    count reconciliation/checkpoint/verification outcomes
    expose pending durable-row gauge from worker/repository
```

## Acceptance Criteria

- durable row exists before VEL delivery;
- restart leaves enough data to reconcile later;
- unknown append outcome performs lookup before retry;
- repeated append uses the same `execution_id`;
- no reconciliation path calls the M2SF/business executor;
- verified checkpoint material is stored only as compact proof metadata;
- workspace-v1 partition is unit tested;
- provider/lifecycle metrics are exported;
- W5-C tests and final W5 validation pass.
