---
doc_type: task
id: W6-B
title: Multi-agent Blackboard coordination over governed capabilities
status: ready
phase: integration
week: W6
tags: [fenix, w6, agents, blackboard, governance]
blocked_by: [W6-A]
blocks: [W6-C]
created: 2026-09-22
completed:
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

## Non-goals

- no new Blackboard implementation;
- no direct agent-to-M2SF or agent-to-VEL calls;
- no global event broker;
- no new policy authority;
- no requirement that every multi-agent execution produces VEL evidence.
