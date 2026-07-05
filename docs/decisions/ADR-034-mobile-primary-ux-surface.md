---
doc_type: adr
id: ADR-034
title: "Mobile is the primary UX surface for the Verifiable Trust wedge"
date: 2026-07-05
status: accepted
deciders: [matias]
tags: [adr, architecture, mobile, ux, governance, wedge, verifiable-trust]
related_tasks: [UIX-00]
related_frs: [FR-300]
---

# ADR-034 - Mobile is the primary UX surface for the Verifiable Trust wedge

## Status

`accepted`

## Context

The mobile app is already the only surface in the product that can deliver the wedge experience with the required density, inline trust cues, and thumb-first decision flow. The strategy plan for the governed console makes that position explicit: the operator should encounter the governed decision surface on mobile first, not as an afterthought behind web-first navigation.

ADR-022 still captures the broader commercial truth that mobile is not the system of record and does not become the backend moat. That stance remains valid. What changes here is the product-experience priority: mobile is now the primary UX surface for the Verifiable Trust wedge.

## Decision

1. Mobile is the primary UX surface for the Verifiable Trust wedge.
2. This ADR re-scopes experience priority only. It does not turn mobile into a hard release gate.
3. ADR-022 remains in force for the backend-moat stance and for the general rule that mobile is not a universal wedge gate.
4. Any mobile UX and navigation work that supports the wedge should assume the mobile app is the canonical place where operators inspect evidence, confidence, cost, policy state, approvals, and audit continuity.

## Consequences

- Mobile-first product investment is now governed as the default UX stance for the wedge.
- Documentation and task planning should treat mobile as the first-class surface for Verifiable Trust, while still preserving the backend/runtime as the system of truth.
- Release gating remains separate from UX priority. If a future decision needs mobile to become a commercial or release gate, that change must be recorded explicitly in a separate ADR.
- ADR-022 is narrowed by this decision only in the experience-priority sense; its broader backend-moat principle remains intact.

## References

- `docs/plans/ui_ux_governed_console_strategy_plan.md`
- `docs/plans/mobile_wedge_harmonization_plan.md`
- `docs/decisions/ADR-022-mobile-deprioritized-for-wedge.md`
