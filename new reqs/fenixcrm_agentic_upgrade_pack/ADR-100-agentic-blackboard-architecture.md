# ADR-100 — Agentic Blackboard Architecture

Status: Proposed
Date: 2026-05-16

## Context

FenixCRM currently implements:
- governed execution
- retrieval grounding
- approvals
- auditability
- policy enforcement

However, agent execution is still relatively linear and workflow-centric.

The next strategic differentiation layer requires:
- multi-agent coordination
- shared operational cognition
- persistent reasoning context
- explainable distributed decision-making

## Decision

FenixCRM shall evolve toward a Blackboard-style multi-agent architecture inspired by:
- Hearsay-II Blackboard Systems
- Global Workspace Theory

Agents shall publish:
- hypotheses
- observations
- risks
- recommendations
- execution intents

into a shared cognitive workspace.

## Architectural Consequences

New domains:
- cognitive_workspace
- agent_memory
- signal_hypothesis
- reasoning_event

New runtime concepts:
- shared workspace
- collaborative reasoning
- hypothesis competition
- confidence aggregation

## Benefits

- stronger agentic differentiation
- explainable coordination
- easier replay/debugging
- future multi-agent orchestration
- richer operational intelligence

## Risks

- higher runtime complexity
- more difficult observability
- concurrency coordination concerns

## Implementation Direction

Phase 1:
- workspace event bus
- shared memory store
- reasoning timeline

Phase 2:
- specialized agents
- confidence arbitration
- collaborative planning
