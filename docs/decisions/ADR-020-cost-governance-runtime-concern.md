---
doc_type: adr
id: ADR-020
title: "Cost governance is a runtime concern and requires first-class usage metering"
date: 2026-04-06
status: accepted
deciders: [matias]
tags: [adr, architecture, metering, quotas, cost]
related_tasks: []
related_frs: [FR-233]
---

# ADR-020 — Cost governance is a runtime concern and requires first-class usage metering

## Status

`accepted`

## Context

The runtime already records audit, policy, and some token/cost data at run level, but usage attribution is not yet formalized as its own domain. The repositioned product depends on being able to explain and govern AI cost per workspace, run, tool, and actor.

## Decision

Usage and quota concepts become part of the target architecture:

- `usage_event`
- `quota_policy`
- `quota_state`

Every governed run should emit usage attribution when data is available, and quota enforcement should be introduced without redesigning the runtime contract later.

## Consequences

### Positive

- cost visibility becomes compatible with the governance story
- workspace- and run-level reporting can be exposed through stable APIs
- future budget enforcement has a dedicated home in the architecture

### Tradeoffs

- runtime contracts need an additional target domain before implementation planning
- some current telemetry paths will need normalization instead of ad hoc fields

## References

- `docs/plans/fenixcrm_strategic_repositioning_spec.md`
- `docs/architecture.md`
- `docs/requirements.md`

