---
id: ADR-100
title: "Agentic Blackboard Architecture"
date: 2026-05-16
status: accepted
deciders: [matias]
tags: [adr, architecture, blackboard, multi-agent, agentic-upgrade]
related_tasks: []
related_frs: []
---

# ADR-100 — Agentic Blackboard Architecture

## Status

`accepted`

## Context

FenixCRM currently implements governed execution, retrieval grounding, approvals, auditability, and policy enforcement. However, agent execution is still relatively linear and workflow-centric.

The next strategic differentiation layer requires:
- multi-agent coordination
- shared operational cognition
- persistent reasoning context
- explainable distributed decision-making

## Decision

FenixCRM shall evolve toward a Blackboard-style multi-agent architecture inspired by Hearsay-II Blackboard Systems and Global Workspace Theory.

Agents shall publish hypotheses, observations, risks, recommendations, and execution intents into a shared cognitive workspace.

## Rationale

The blackboard pattern is uniquely suited for multi-agent coordination where agents have partial knowledge and must collaborate to converge on a plan. It provides natural explainability (every contribution is logged), replay support (reasoning timeline is append-only), and extensibility (new agent types plug in as subscribers).

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Direct agent-to-agent messaging | Tight coupling; hard to audit and replay |
| Centralized orchestrator only | Single point of bottleneck; limits parallelism |
| Event sourcing without shared workspace | Loses the shared cognition layer; agents cannot observe each other's reasoning |

## Consequences

**Positive:**
- Stronger agentic differentiation
- Explainable coordination via reasoning timeline
- Easier replay and debugging
- Future multi-agent orchestration foundation
- Richer operational intelligence

**Negative / tradeoffs:**
- Higher runtime complexity
- More difficult observability (partially mitigated by append-only timeline)
- Concurrency coordination concerns (addressed by R.11/R.12 workspace bus fixes)

## Implementation Direction

**Phase 1 (complete):**
- workspace event bus (`cognitive_workspace`, `reasoning_event`)
- shared memory store (`agent_memory`)
- reasoning timeline

**Phase 2 (complete):**
- specialized agents (Signal, Evidence, Policy)
- confidence arbitration (`signal_hypothesis`)
- collaborative planning (`ToolSequenceStep[]` → governed execution)

## References

- Remediation plan: `docs/plans/fenixcrm_agentic_upgrade_remediation_plan.md`
- `internal/domain/blackboard/` — implementation root
- R.11: WorkspaceBus race fix
- R.12: WorkspaceBus registry per cognitive_workspace_id

## Changelog

- 2026-05-16: Created as `Proposed` in `new reqs/fenixcrm_agentic_upgrade_pack/`
- 2026-05-18: Promoted to `docs/decisions/` with status `Accepted`
