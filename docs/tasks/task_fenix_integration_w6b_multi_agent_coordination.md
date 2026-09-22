---
doc_type: task
id: W6-B
title: Multi-agent Blackboard coordination over governed capabilities
status: closed
phase: integration
week: W6
tags: [fenix, w6, agents, blackboard, governance]
blocked_by: [W6-A]
blocks: [W6-C]
created: 2026-09-22
completed: 2026-09-22
---

# Task W6-B — Multi-agent Blackboard coordination

**Plan**: [W6 Functional Execution](../plans/fenix-integration-w6-functional-execution.md)

## Goal

Prove that multiple agents can contribute to one Blackboard workspace and hand a selected plan to
the existing governed execution path without creating a second execution authority.

## Tasks

- **B1** Reuse the existing specialized Blackboard agents/artifact memory as contributors.
- **B2** Build/select one collaborative proposal containing a governed external capability step.
- **B3** Execute the selected proposal only through `PlannerExecutor -> ToolRegistry`.
- **B4** Verify one stable execution identity per external step and distinct identities for distinct steps.
- **B5** Verify contributor/artifact state survives the handoff but raw reasoning is not copied into VEL evidence.
- **B6** Exercise one stale/not-ready collaboration state and confirm execution defers instead of bypassing planning.
- **B7** Preserve a focused multi-agent functional proof for the W6 demo/handoff.

## Implementation result — 2026-09-22

**W6-B CLOSED.**

- **B1 — PASS by existing runtime proof**: specialized Signal, Evidence and Policy agents already
  persist independent Blackboard artifacts; `TestRuntime_StartPersistsSpecializedArtifacts`
  remains the canonical contributor proof.
- **B2 — IMPLEMENTED**: `PlanningConfig.ActionSteps` now binds a collaborative proposal to
  concrete executable `ToolSequenceStep` values. Empty configuration preserves the legacy
  generic planning markers, so the planner remains provider-neutral.
- **B3 — IMPLEMENTED**: the W6-B proof builds a collaborative plan with
  `salesforce.flow.export` + `salesforce.flow.validate` and executes it only through
  `PlannerExecutor -> ToolRegistry -> M2SF adapter`.
- **B4 — IMPLEMENTED**: the export step deliberately receives a transient 503. Its automatic
  retry keeps the same `execution_id`; the following validate step receives a different
  `execution_id`.
- **B5 — IMPLEMENTED**: evidence is enabled for the optional export step. The proof builds the
  actual minimized `EvidenceEnvelope`, checks input/output digests, preserves execution
  correlation, and asserts Blackboard-only collaboration content is absent from the envelope.
- **B6 — IMPLEMENTED**: the same action binding with missing evidence produces
  `awaiting_evidence`; `PlannerExecutor` returns a deferral with zero provider calls.
- **B7 — IMPLEMENTED**: the focused proof is preserved in
  `internal/api/integration_runtime_w6_test.go`, with planner binding coverage in
  `internal/domain/blackboard/planner_test.go`.

Implementation commits:

- `33d484b0` — collaborative-plan action binding + execution identity proof.
- `ba06e9ca` — Blackboard/evidence boundary + not-ready deferral proof.

Per the repository-owner QA decision made during W5-C closure, no additional GitHub QA run is
required for this W6-B handoff; the committed tests are the executable proof artifacts.

## Non-goals

- no new Blackboard implementation;
- no direct agent-to-M2SF or agent-to-VEL calls;
- no global event broker;
- no new policy authority;
- no requirement that every multi-agent execution produces VEL evidence.
