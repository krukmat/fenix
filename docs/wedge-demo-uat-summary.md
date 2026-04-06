---
doc_type: summary
title: "Wedge Demo and UAT Bundle"
status: active
created: 2026-04-06
updated: 2026-04-06
tags: [summary, demo, uat, wedge]
---

# Wedge Demo and UAT Bundle

This note defines the canonical demo and UAT path for the repositioned wedge on 2026-04-06.

## Objective

Demonstrate that FenixCRM is commercially credible as a governed AI layer for customer operations through:

1. one support flow with approval
2. one support flow with abstention or handoff
3. one sales copilot flow with grounded account/deal context
4. visible audit and usage evidence for each flow

## Scope

This bundle validates the current wedge only:

- `Support Copilot`
- `Support Agent`
- `Sales Copilot`

It does not require:

- mobile parity
- marketplace or plugin packaging
- broad CRM expansion beyond the support and sales slices already implemented

## Demo Flows

### 1. Support flow with approval

Use `POST /api/v1/agents/support/trigger` with a real `case_id` and `customer_query` that leads to a governed action requiring approval.

Verify:

- the run is created and visible through `GET /api/v1/agents/runs/{id}`
- the public outcome becomes `awaiting_approval` before the protected action executes
- pending work is visible through `GET /api/v1/approvals`
- an operator can resolve the approval through `PUT /api/v1/approvals/{id}`
- the final run state is auditable and usage-attributed

### 2. Support flow with abstention or handoff

Run `POST /api/v1/agents/support/trigger` against a case that either lacks enough evidence or must be escalated.

Verify one of these canonical outcomes:

- `abstained` with evidence-backed insufficiency
- `handed_off` with retrievable continuity package through `GET /api/v1/agents/runs/{id}/handoff`

In both cases, verify:

- evidence pack is present
- audit trail captures the run outcome
- usage is visible through `GET /api/v1/usage`

### 3. Sales Copilot grounded brief

Use `POST /api/v1/copilot/sales-brief` with `entityType=account|deal` and a real `entityId`.

Verify:

- response includes `summary`, `risks`, `nextBestActions`, and `evidencePack`
- outcome is `completed` when grounding is sufficient
- outcome is `abstained` when grounding is insufficient
- usage is visible through `GET /api/v1/usage`

## Visibility Checks

For each scenario above, capture:

- run or request identifier
- public outcome
- evidence pack presence
- one audit record from `GET /api/v1/audit/events`
- one usage event from `GET /api/v1/usage`

When quota state exists for the workspace, also verify `GET /api/v1/quota-state`.

## Acceptance Checklist

- Support scenario with approval is reproducible.
- Support scenario with abstention or handoff is reproducible.
- Sales Copilot brief is reproducible.
- Audit and usage visibility exist for every scenario.
- No acceptance step depends on mobile or BFF parity.

## Deferred From This Bundle

- mobile-specific walkthroughs
- marketplace packaging
- broad CRM workflows outside `case`, `account`, and `deal`
