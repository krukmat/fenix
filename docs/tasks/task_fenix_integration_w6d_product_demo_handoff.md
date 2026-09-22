---
doc_type: task
id: W6-D
title: Product/demo handoff for the complete Fenix integration flow
status: in_progress
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

- **D1 — Happy-path tour — IMPLEMENTED**
  - Agent / Blackboard contribution;
  - collaborative proposal;
  - governed ToolRegistry execution;
  - M2SF semantic result;
  - optional VEL evidence.

- **D2 — Optionality matrix — IMPLEMENTED**
  - M2SF without VEL;
  - M2SF with VEL;
  - governed non-M2SF capability with VEL;
  - governed execution with neither provider.

- **D3 — Resilience tour**
  - M2SF retry/failure;
  - VEL indeterminate;
  - verification failure;
  - restart/reconciliation without business replay.

- **D4 — Authority/boundary diagram — IMPLEMENTED**
  - Fenix = governance/execution authority;
  - Blackboard = coordination/shared artifact state;
  - Mermaid2SF = Salesforce/FlowIR semantic authority;
  - Evidence Runtime / Outbox = Fenix evidence-delivery boundary;
  - VEL = cryptographic evidence authority, explicitly **not a ToolRegistry tool**.

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

## D1/D2/D4 result — 2026-09-22

The canonical README now separates the two runtime branches:

```text
business capability path:
Agent -> Blackboard -> PlannerExecutor -> ToolRegistry -> Governance -> M2SF/other capability

evidence path:
governed outcome -> Evidence Runtime / Outbox -> VEL
```

VEL is not registered as an agent tool and cannot be directly invoked through ToolRegistry.

Remaining work:

- **D3** resilience tour;
- **D5** executable proof index;
- **D6** final README/operator handoff and W6 closure.
