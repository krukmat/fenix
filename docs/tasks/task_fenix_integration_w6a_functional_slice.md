---
doc_type: task
id: W6-A
title: First agent-driven functional integration slice
status: implemented
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

## Implementation — 2026-09-22

Implemented in `internal/api/integration_runtime_w6_test.go` (commit `17f8fef3`).

The proof exercises the real composition seam:

```text
Blackboard PlannerExecutor
        -> ToolRegistry.Execute
        -> W4 governance planner
        -> concrete M2SF HTTP adapter
        -> normalized Flow result
        -> Blackboard observation/memory
        -> optional evidence recorder
        -> correlated tool audit
```

Two variants use the same `salesforce.flow.export` functional scenario:

- evidence OFF: M2SF executes and no evidence recorder call occurs;
- evidence ON: the same governed execution calls the evidence port and preserves the same
  `trace_id` / `execution_id` seen by M2SF.

The proof also asserts that Blackboard memory contains the provider result/reference while not
persisting verification bundles or a reasoning-trace field.

No new Blackboard-to-ToolRegistry bridge was required: `PlannerExecutor` already executes
planned steps through `ToolRegistry.Execute()`.

Per the owner-approved QA policy for this continuation, no additional GitHub QA run is required
to proceed to W6-B.

## Non-goals

- no new provider transport;
- no new Blackboard implementation unless an actual functional gap is found;
- no new evidence engine;
- no mandatory M2SF+VEL coupling;
- no W5-C coverage-only remediation.
