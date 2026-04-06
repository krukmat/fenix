---
doc_type: adr
id: ADR-022
title: "Mobile is supported but not a universal wedge gate"
date: 2026-04-06
status: accepted
deciders: [matias]
tags: [adr, architecture, mobile, bff, prioritization]
related_tasks: []
related_frs: [FR-300, FR-301]
---

# ADR-022 — Mobile is supported but not a universal wedge gate

## Status

`accepted`

## Context

The repository contains a mobile app and BFF, but the repositioned wedge is governed AI execution for support and sales workflows. Requiring mobile parity before validating the wedge would make interface symmetry more important than product proof.

## Decision

Mobile and BFF remain supported interfaces, but:

- they are optional delivery surfaces for the wedge
- they do not block core support workflow completion
- architecture and requirements shall not treat mobile-first parity as a universal P0 release gate

## Consequences

### Positive

- support and sales workflow value can be validated through API and core runtime first
- BFF/mobile work can be justified by concrete workflow evidence
- architectural priority remains on retrieval, governance, approvals, audit, and metering

### Tradeoffs

- some existing documentation and roadmap language must be downgraded from P0 expectations
- mobile-specific NFRs move behind wedge validation unless a workflow proves they are critical

## References

- `docs/plans/fenixcrm_strategic_repositioning_spec.md`
- `docs/architecture.md`
- `docs/requirements.md`
