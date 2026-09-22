---
doc_type: task
id: W6-D
title: Product/demo handoff for the complete Fenix integration flow
status: ready
phase: integration
week: W6
tags: [fenix, w6, demo, handoff, blackboard, mermaid2sf, vel]
blocked_by: [W6-C]
blocks: []
created: 2026-09-22
completed:
---

# Task W6-D — Product/demo handoff

**Plan**: [W6 Functional Execution](../plans/fenix-integration-w6-functional-execution.md)

## Goal

Turn the W6 executable proofs into one concise end-to-end tour that explains what the integrated
system does, where authority lives, when M2SF and VEL participate, and how failures behave.

## Tasks

- **D1 — Happy-path tour**
  - Agent / Blackboard contribution;
  - collaborative proposal;
  - governed ToolRegistry execution;
  - M2SF semantic result;
  - optional VEL evidence.

- **D2 — Optionality matrix**
  - M2SF without VEL;
  - M2SF with VEL;
  - governed non-M2SF capability with VEL;
  - governed execution with neither provider.

- **D3 — Resilience tour**
  - M2SF retry/failure;
  - VEL indeterminate;
  - verification failure;
  - restart/reconciliation without business replay.

- **D4 — Authority/boundary diagram**
  - Fenix = governance/execution authority;
  - Blackboard = coordination/shared artifact state;
  - Mermaid2SF = Salesforce/FlowIR semantic authority;
  - VEL = cryptographic evidence authority.

- **D5 — Executable proof index**
  - link the W6-A/B/C tests and relevant W1-W5 contract docs;
  - distinguish executable proof from residual risks/QA waiver.

- **D6 — README / operator handoff**
  - add the concise integration tour to the canonical README or integration entry point;
  - close W6 with the next productization decision clearly identified.

## Constraints

- no new runtime layer;
- no new provider coupling;
- do not turn demo documentation into another architecture abstraction;
- retain the W5-C QA waiver and W6 no-global-QA decision accurately.
