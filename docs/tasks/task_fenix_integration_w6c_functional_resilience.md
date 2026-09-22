---
doc_type: task
id: W6-C
title: Functional resilience across M2SF and VEL failures
status: in_progress
phase: integration
week: W6
tags: [fenix, w6, resilience, mermaid2sf, vel, blackboard]
blocked_by: [W6-B]
blocks: [W6-D]
created: 2026-09-22
completed:
---

# Task W6-C — Functional resilience

**Plan**: [W6 Functional Execution](../plans/fenix-integration-w6-functional-execution.md)

## Goal

Prove that provider/evidence failures remain explicit, preserve successful business results where
appropriate, maintain execution identity, and never replay a business capability merely to recover
evidence.

## Tasks

- **C1 — M2SF unavailable — IMPLEMENTED**
  - persistent 503;
  - exactly two bounded attempts;
  - same `execution_id` across attempts;
  - no invented provider result;
  - Blackboard outcome stops with `tool_failure`.

- **C2 — Retryable provider failure — IMPLEMENTED / REUSED**
  - covered by `TestW6B_CollaborativePlanExecutesDistinctGovernedStepsWithStableRetryIdentity`;
  - first export attempt returns 503;
  - second attempt succeeds;
  - retry retains the same `execution_id`.

- **C3 — VEL unavailable with optional evidence — IMPLEMENTED**
  - provider executes once and succeeds;
  - evidence recorder returns indeterminate/unavailable;
  - provider output is preserved;
  - Fenix exposes `evidence_indeterminate`;
  - provider is not replayed.

- **C4 — Verification failure — IMPLEMENTED**
  - provider executes once and succeeds;
  - evidence reports `verification_failed`;
  - provider output is preserved;
  - Fenix exposes `evidence_verification_failed`;
  - no verified-integrity claim is emitted.

- **C5 — Restart & reconciliation — NEXT**
  - resume evidence delivery/checkpoint/verify after restart;
  - prove the business capability is not repeated.

- **C6 — Blackboard failure projection — NEXT**
  - assert agent-safe failure state and retained result/reference;
  - no raw VEL bundle/internal state in Blackboard.

- **C7 — Resilience proof consolidation — NEXT**
  - consolidate C1-C6 into the W6-D demo/handoff proof.

## Proof

Primary proof file:

`internal/api/integration_runtime_w6_test.go`

First-block implementation commit:

`90ab2a88` — external failure resilience through Blackboard.

## Validation note

No full GitHub QA run was launched for this block, following the repository-owner decision made at
W5-C closure. A targeted local Go test could not be executed because the local runtime could not
resolve GitHub to clone the repository. The committed proof is therefore implementation-complete
but not represented as a globally green QA run.

## Non-goals

- no new retry engine;
- no direct agent-to-provider recovery logic;
- no replay of M2SF business actions to repair VEL state;
- no coupling that makes VEL mandatory for M2SF execution;
- no reopening of the W5-C coverage waiver.
