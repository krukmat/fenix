---
doc_type: task
id: W6-A
title: First agent-driven functional integration slice
status: ready
phase: integration
week: W6
tags: [fenix, w6, agents, blackboard, mermaid2sf, vel]
blocked_by: [W5-FINAL]
blocks: [W6-B]
created: 2026-09-22
completed:
---

# Task W6-A — First agent-driven functional integration slice

**Plan**: [W6 Functional Execution](../plans/fenix-integration-w6-functional-execution.md)

## Tasks

- **A1** Define one representative Salesforce Flow agent scenario.
- **A2** Route it through the existing Agentic Blackboard/orchestration path.
- **A3** Execute M2SF only through the governed ToolRegistry capability.
- **A4** Cover evidence OFF and evidence ON variants.
- **A5** Verify Blackboard stores coordination/artifact state, not raw reasoning or VEL bundles.
- **A6** Verify trace/execution/audit/proof correlation.
- **A7** Preserve an executable fixture/demo proof.

## Non-goals

- no new provider transport;
- no new Blackboard implementation unless an actual functional gap is found;
- no new evidence engine;
- no mandatory M2SF+VEL coupling;
- no W5-C coverage-only remediation.
