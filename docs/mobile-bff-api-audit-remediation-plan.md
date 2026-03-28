---
title: "Mobile/BFF/API Audit And Remediation Plan"
version: "1.0"
date: "2026-03-28"
timezone: "Europe/Madrid"
language: "en"
status: "proposed"
audience: ["engineering", "mobile", "platform", "product"]
tags: ["mobile", "bff", "api", "audit", "remediation", "deployment"]
canonical_id: "fenix-mobile-bff-api-audit-v1"
source_of_truth:
  - "docs/mobile-agent-spec-transition-gap-closure-plan.md"
  - "docs/deployment-plan-digitalocean.md"
  - "docs/requirements.md"
  - "docs/architecture.md"
  - "mobile/src/services/api.ts"
  - "mobile/src/services/api.agents.ts"
  - "mobile/src/services/api.secondary.ts"
  - "bff/src/routes/*.ts"
  - "internal/api/routes.go"
---

# Mobile/BFF/API Audit And Remediation Plan

> Purpose: audit the mobile-facing API contract affected by the current BFF and deployment shape, then define a remediation path that reduces long-term maintenance cost.

> Executive summary: the current production shape is documented as `Caddy -> BFF -> Go backend`, while the actual mobile contract is split between BFF-only routes and Go pass-through routes. The target state is a minimal BFF that keeps auth, Copilot SSE, health, and generic relay behavior, while moving mobile-required domain fields into the Go API.

## Context

The repository currently documents and implements a mobile path that goes
through the BFF, not directly to the Go backend.

Observed deployment intent:

- public traffic enters through Caddy
- `/bff/*` is routed to the BFF
- all other backend API traffic is routed to the Go service

Observed mobile client behavior:

- mobile calls a mix of `/bff/*` and `/bff/api/v1/*`
- mobile depends on BFF-specific aggregation for CRM list/detail flows
- mobile depends on BFF for auth relay and Copilot SSE

Decision carried into this audit:

- keep a single public entrypoint via Caddy
- reduce custom BFF surface over time
- keep the BFF as a minimal mobile gateway, not as the main domain contract

## Current Observed Contract

### Mobile-facing routes in active use

Direct BFF routes:

- `/bff/auth/login`
- `/bff/auth/register`
- `/bff/copilot/chat`
- `/bff/accounts`
- `/bff/deals`
- `/bff/cases`
- `/bff/accounts/:id/full`
- `/bff/deals/:id/full`
- `/bff/cases/:id/full`
- `/bff/health`

Go pass-through routes used through the BFF:

- `/bff/api/v1/accounts`
- `/bff/api/v1/contacts`
- `/bff/api/v1/deals`
- `/bff/api/v1/cases`
- `/bff/api/v1/signals`
- `/bff/api/v1/workflows`
- `/bff/api/v1/agents/runs`
- `/bff/api/v1/agents/runs/:id`
- `/bff/api/v1/agents/runs/:id/handoff`
- `/bff/api/v1/approvals`
- other standard CRUD and workflow endpoints under `/bff/api/v1/*`

### Where behavior lives today

BFF-owned behavior:

- auth relay
- Copilot SSE proxy
- health check
- CRM list enrichment with `active_signal_count`
- CRM detail aggregation for account, deal, and case screens

Go-owned behavior:

- workflows contract
- signals contract
- agent runs contract including `accepted`, `rejected`, `delegated`
- `rejection_reason`
- readiness endpoint `/readyz`
- underlying CRM CRUD and related list/detail endpoints

## Audit Findings

### 1. Documentation drift exists between intended and actual contracts

The local documentation describes the BFF as a thin gateway, but the active
implementation still owns mobile-specific domain shaping for CRM list/detail
responses. This makes the BFF a material contract layer, not only a transport
layer.

### 2. `docs/openapi.yaml` does not reflect the mobile-relevant API surface

The OpenAPI file does not currently cover key routes that matter to the mobile
contract and the deployment/readiness story, including:

- `/readyz`
- signals endpoints used by mobile
- workflow endpoints used by mobile
- agent run endpoints used by mobile

This means the formal API description is not sufficient for mobile contract
verification.

### 3. BFF health checks backend reachability, not backend readiness

Current BFF health behavior calls Go `/health`, while deployment and
architecture now rely on `/readyz` to validate database plus critical AI
providers. This creates a gap where BFF may report healthy while Copilot or
provider-backed flows are not actually ready.

### 4. `active_signal_count` is currently a BFF-owned concern

The current BFF aggregated and list routes add `active_signal_count` by querying
signals per entity. This avoids mobile-side N+1 requests, but it keeps mobile
data shaping in the BFF and increases BFF/backend chattiness.

### 5. Agent run status compatibility depends on the Go contract, not OpenAPI

Mobile relies on `accepted`, `rejected`, `delegated`, and `rejection_reason`.
Those fields are exposed by Go handlers and mobile tests, but they are not
represented as the formal public contract in `docs/openapi.yaml`.

### 6. BFF aggregation currently tolerates partial success with `200`

The aggregated CRM detail routes can return `200` responses with partial `null`
subsections when some backend calls fail. This behavior is not clearly
documented as a supported contract and should either be formalized or removed.

## Target Contract

### Keep in the BFF

- `/bff/auth/*`
- `/bff/copilot/chat`
- `/bff/health`
- `/bff/api/v1/*` as transparent relay

### Retire gradually from the BFF

- `/bff/accounts`
- `/bff/deals`
- `/bff/cases`
- `/bff/accounts/:id/full`
- `/bff/deals/:id/full`
- `/bff/cases/:id/full`

### Move to the Go API as first-class contract behavior

- mobile-required `active_signal_count` on account, deal, and case payloads
- all workflow, signal, and agent run fields already consumed by mobile
- readiness semantics centered on `/readyz`

### Resulting architecture

- public mobile traffic still enters through Caddy
- BFF remains in the path for auth, SSE, health, and generic proxying
- domain-specific response shaping moves out of the BFF
- mobile composes detail screens explicitly or consumes Go-enriched responses

## Remediation Plan

### Phase 1 - Route matrix and drift report

- build a matrix of `mobile call -> BFF route -> Go route -> documented source`
- mark each route as `keep`, `migrate`, `move to Go`, or `remove`
- capture doc drift against deployment, architecture, mobile spec, and OpenAPI

### Phase 2 - Expand Go contracts for mobile parity

- add `active_signal_count` to Go list/detail responses where mobile still needs
  it
- keep agent run status fields and `rejection_reason` stable
- ensure workflows and signals remain explicitly available through Go routes

### Phase 3 - Move mobile off BFF custom aggregation

- migrate CRM list/detail data access to `/bff/api/v1/*`
- let mobile compose detail queries explicitly where needed
- keep Copilot and auth through the BFF

### Phase 4 - Minimize or remove BFF custom domain routes

- deprecate the custom list/detail aggregation routes
- keep only minimal gateway responsibilities in the BFF
- align health behavior with `/readyz`

### Phase 5 - Clean up docs and tests

- update documentation so the described contract matches the deployed contract
- remove or rewrite tests that assume BFF-owned domain aggregation
- keep coverage for auth relay, SSE proxy, health, and pass-through behavior

## Validation

Required validation areas:

- BFF tests for auth, proxy, Copilot SSE, and health
- mobile API and hook tests
- backend route and handler tests for signals, workflows, and agent runs
- smoke coverage for Copilot, workflow flows, rejected runs, delegated runs,
  and signal badges

Minimum acceptance signals:

- mobile no longer depends on BFF custom aggregation to function
- the public contract is describable from docs without reading implementation
- `/bff/health` reflects backend readiness relevant to mobile-facing traffic

## Task Breakdown

- `docs/tasks/task_mobile_bff_remediation_1.md` - route matrix and drift report
- `docs/tasks/task_mobile_bff_remediation_2.md` - Go contract enrichment for
  mobile parity
- `docs/tasks/task_mobile_bff_remediation_3.md` - mobile migration to
  pass-through and explicit composition
- `docs/tasks/task_mobile_bff_remediation_4.md` - BFF minimization and contract
  cleanup
- `docs/tasks/task_mobile_bff_remediation_5.md` - documentation and validation
  alignment

Execution dependency chain:

1. task 1 defines route ownership and compatibility windows
2. task 2 moves mobile-required domain fields into Go
3. task 3 migrates mobile away from BFF custom CRM routes
4. task 4 removes BFF custom domain behavior and aligns health with `/readyz`
5. task 5 updates architecture, OpenAPI, deployment docs, and final validation

## Follow-up Documentation Updates

These updates are intentionally tracked as follow-up work to the remediation,
not completed by this narrative document itself:

- update `docs/architecture.md` so the BFF is described as a minimal mobile
  gateway instead of an aggregation-oriented gateway
- update `docs/openapi.yaml` to reflect the current Go API surface that mobile
  depends on, including `/readyz`, signals, workflows, and agent runs
- avoid documenting BFF custom routes in the future if they are on the removal
  path

## Assumptions

- production uses a single public domain fronted by Caddy
- the primary goal is lower maintenance cost, not preserving every existing BFF
  convenience route
- the BFF remains necessary for auth relay, Copilot SSE, and health, but should
  stop being the main domain contract for CRM data
