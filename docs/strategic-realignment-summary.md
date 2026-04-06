---
doc_type: summary
id: summary-strategic-realignment-2026-04-06
title: Strategic Realignment Summary
status: active
date: 2026-04-06
tags: [strategy, architecture, obsidian, governance, planning]
related_docs:
  - architecture
  - requirements
  - plans/fenixcrm_strategic_repositioning_spec
  - plans/fenixcrm_strategic_repositioning_implementation_plan
  - dashboards/status
---

# Strategic Realignment Summary

## Purpose

This note captures the repository-level strategic realignment applied on 2026-04-06 so the Obsidian vault reflects the current product direction without requiring a separate manual reconstruction step.

## Direction Locked

FenixCRM is positioned as a **governed AI layer for customer operations**, not as a broad CRM replacement.

Current wedge order:

1. Support Copilot and Support Agent
2. Sales Copilot
3. Evidence-grounded execution with approvals, auditability, and policy enforcement

## Documentation Updated

- `docs/architecture.md`
- `docs/requirements.md`
- `README.md`
- `CLAUDE.md`
- `docs/plans/fenixcrm_strategic_repositioning_implementation_plan.md`
- `docs/dashboards/status.md`
- `docs/dashboards/fr-uc-status.md`

## ADRs Created

- `docs/decisions/ADR-019-product-category-governed-ai-layer.md`
- `docs/decisions/ADR-020-cost-governance-runtime-concern.md`
- `docs/decisions/ADR-021-integration-first-context-strategy.md`
- `docs/decisions/ADR-022-mobile-deprioritized-for-wedge.md`

## Architectural Effects

- CRM breadth is reframed as a **context layer**, not the main moat.
- Retrieval and evidence are first-class product boundaries.
- Policy, approval, audit, and metering are runtime-critical concerns.
- Usage and quota capabilities are now explicit target domains.
- The minimum connector-ingest boundary is now frozen in `knowledge_item` and `POST /api/v1/knowledge/ingest`.
- Mobile and BFF remain supported interfaces, but they do not define wedge completion.
- The package model is now explicit in top-level docs: `Support Copilot`, `Support Agent`, and `Sales Copilot`.

## Planning Inputs Now Required

The next planning iteration should prioritize:

1. Commercial validation material around the two wedges
2. Deferred connector expansion on top of the frozen ingest boundary
3. Marketplace and non-wedge follow-up outside the main delivery path

## Implementation Plan Added

The canonical implementation plan for the strategic direction now lives in:

- `docs/plans/fenixcrm_strategic_repositioning_implementation_plan.md`
- `docs/wedge-demo-uat-summary.md`

That plan does three things:

1. reorders execution around the support wedge first and the sales wedge second
2. pulls forward usage attribution, approval determinism, and evidence contract hardening
3. downgrades mobile breadth, marketplace scope, and broad CRM expansion to non-blocking work

## Vault Maintenance Rule

From this point onward, any change that alters architecture, scope, roadmap, operating rules, or project status should update both the source document and the relevant Obsidian tracking artifact in the same working turn.
