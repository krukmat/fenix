---
doc_type: adr
id: ADR-019
title: "Product category shift: governed AI layer for customer operations, not broad CRM replacement"
date: 2026-04-06
status: accepted
deciders: [matias]
tags: [adr, product, architecture, positioning, governance]
related_tasks: []
related_frs: [FR-200, FR-230, FR-232, FR-070]
---

# ADR-019 — Product category shift: governed AI layer for customer operations, not broad CRM replacement

## Status

`accepted`

## Context

The implemented strengths of FenixCRM are not generic CRM breadth. They are:

- evidence-grounded retrieval
- policy-governed execution
- approvals and handoff
- immutable auditability
- traceable agent and copilot behavior

Treating the project as a broad CRM replacement dilutes those strengths and creates unnecessary comparison against full-suite incumbents.

## Decision

FenixCRM is positioned as a **governed AI layer for customer operations**.

This means:

- the commercial wedge is Support Copilot / Support Agent first
- Sales Copilot is the next wedge
- CRM entities are treated as a **system of context**
- architecture and repo-facing documentation shall stop framing broad CRM parity as the main business axis

## Consequences

### Positive

- product messaging aligns with the strongest technical assets
- the roadmap can prioritize trust, governance, and integration boundaries
- support and sales workflows become the primary acceptance lens

### Tradeoffs

- broad CRM replacement language must be removed from primary docs
- generic CRM breadth work no longer wins priority by default
- some existing plan/requirements language becomes explicitly secondary until refreshed

## References

- `docs/plans/fenixcrm_strategic_repositioning_spec.md`
- `docs/architecture.md`
- `README.md`

