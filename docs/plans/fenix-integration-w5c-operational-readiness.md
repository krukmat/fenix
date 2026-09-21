---
doc_type: plan
id: W5-C
title: Fenix Integration W5-C Operational Readiness
status: in_progress
phase: integration
tags: [fenix, integration, vel, reliability, observability]
created: 2026-09-21
---

# W5-C — Operational Readiness

## Goal

Close the remaining W5 operational-readiness gaps after W5-B without introducing a broker,
duplicating VEL cryptography, or coupling Mermaid2SF and VEL.

W5-C owns four gaps:

- **G07** durable evidence reconciliation/outbox
- **G08** checkpoint scheduling + verification progression
- **G09** final stream partition policy
- **G10** integration observability/metrics

## Constraints

- Fenix remains the governance/execution authority.
- Mermaid2SF remains the Salesforce/FlowIR semantic authority.
- VEL remains the cryptographic evidence/checkpoint/verification authority.
- Evidence recovery MUST NOT repeat the business capability.
- The default architecture remains Go + SQLite + in-process workers; no Kafka/broker is added.
- VEL evidence remains independently optional from M2SF execution.
- Fenix stores compact proof/checkpoint references, never full VEL verification bundles.

## W5-C system context

```text
Governed capability completion
        |
        v
EvidenceEnvelope
        |
        v
SQLite evidence_delivery outbox  <---- process restart / retry
        |
        +--> VEL append / idempotency lookup
        |        |
        |        v
        |    recorded event
        |
        +--> lifecycle worker
                 |
                 +--> checkpoint stream
                 +--> fetch portable bundle
                 +--> VEL independent verify
                 |
                 v
        compact ProofReference + lifecycle state
                 |
                 +--> audit projection
                 +--> /metrics integration telemetry
```

## G07 — Durable reconciliation/outbox

Persist the minimized `EvidenceEnvelope` before crossing the VEL boundary.

Recovery policy:

```text
pending_record
   -> append
   -> recorded

append outcome unknown
   -> indeterminate
   -> lookup(stream_id, execution_id)
   -> found: recorded
   -> not found: retry append with SAME execution_id
```

Only evidence delivery is retried. The business capability is never replayed.

## G08 — Checkpoint + verification lifecycle

Use the existing VEL provider lifecycle:

```text
POST /v1/streams/{stream_id}/checkpoints
GET  /v1/streams/{stream_id}/bundle
POST /v1/verify
```

The Fenix lifecycle worker advances durable rows:

```text
recorded -> pending_checkpoint -> verified
                              \-> verification_failed
```

One checkpoint/verification pass may advance multiple rows from the same stream when the
checkpoint tree size covers their event sequence.

## G09 — Stream partition policy

Freeze **workspace partitioning v1**:

```text
stream_id = "workspace/" + workspace_id
```

Rationale:

- preserves deterministic ordering for one workspace;
- avoids trace/run stream explosion;
- keeps replay/idempotency lookup stable across process restarts;
- aligns with the current single-node Fenix deployment model;
- leaves future sharding as an explicit versioned policy change instead of an implicit string change.

## G10 — Observability

Expose provider and lifecycle telemetry through the existing Prometheus-compatible `/metrics`
surface:

- provider call count by provider/operation/outcome;
- provider cumulative latency;
- durable reconciliation attempts;
- checkpoint attempts;
- verification outcomes;
- current due/pending durable evidence count.

Correlation remains based on `trace_id` + `execution_id`, propagated through the existing
Fenix HTTP headers and persisted in the outbox envelope.

## Acceptance gate

W5-C is complete when:

1. an indeterminate VEL append survives process restart and can reconcile without replaying M2SF;
2. recorded evidence progresses to checkpointed/verified from the worker;
3. stream IDs are produced only by the frozen workspace-v1 policy;
4. integration telemetry is visible in `/metrics`;
5. unit/integration tests cover restart-safe recovery, idempotent retry, verification progression,
   stream policy and metrics;
6. W5 documentation reflects the final runtime behavior.
