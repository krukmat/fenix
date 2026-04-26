---
id: ADR-009
title: "BFF is a thin proxy — zero business logic, zero database access"
date: 2026-02-10
status: accepted
deciders: [matias]
tags: [adr, bff, architecture, express]
related_tasks: [task_4.1]
related_frs: [FR-301]
extended_by: [ADR-025]
---

# ADR-009 — BFF is a thin proxy: zero business logic, zero database access

## Status

`accepted`

## Context

Task 4.1 introduced the BFF (Backend For Frontend) layer — an Express.js 5 + TypeScript
service that sits between the React Native mobile app and the Go backend.

The BFF could be designed as a full API layer (with its own business logic, DB queries,
or data transformation), or as a thin proxy. Both patterns are common in the industry.

For FenixCRM, the Go backend already contains all business logic, domain models, and
persistence. The mobile app needs:

1. Auth header relay (JWT forwarding)
2. Request aggregation (combine N Go API calls into one BFF response)
3. SSE proxy (forward Server-Sent Events from Go to mobile)
4. Mobile-specific HTTP headers (content negotiation, platform hints)

## Decision

The BFF is a **thin proxy** with a hard constraint:

- **Zero business logic** — no domain rules, no validation beyond format checks
- **Zero database access** — no SQLite, no ORM, no direct data reads
- **Zero state** — stateless, horizontally scalable

Allowed BFF responsibilities:
```
Auth relay     → Forward Authorization: Bearer <jwt> from mobile to Go backend
Aggregation    → Fan-out N Go API calls, merge responses (no business rules)
SSE proxy      → Open SSE connection to Go, relay events to mobile client
Mobile headers → Set Accept, Content-Type, X-Platform headers before forwarding
```

Any feature that requires business logic goes into the Go backend, never the BFF.

## Rationale

- Go backend is the authoritative source of truth — duplicating logic in the BFF creates
  two places to maintain and two places to get out of sync
- BFF is deployed separately from the Go backend — keeping it stateless simplifies
  deployment (no migration, no DB connection pool management in BFF)
- SSE proxy is the only non-trivial BFF responsibility — all others are pure forwarding
- Mobile team can evolve BFF aggregation endpoints without touching Go domain logic

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Full BFF with own domain logic | Duplicates Go backend; two sources of truth |
| No BFF (mobile calls Go directly) | Mobile cannot handle SSE multiplexing; aggregation harder on device |
| GraphQL BFF | Overkill for MVP; adds schema maintenance overhead |
| BFF with own SQLite for caching | Adds state and persistence complexity; Go already handles caching concerns |

## Consequences

**Positive:**
- BFF is stateless — easy to scale horizontally, easy to restart without data loss
- Business logic lives in one place (Go) — single test surface
- BFF failures are recoverable — mobile can retry directly against Go in degraded mode

**Negative / tradeoffs:**
- Some mobile-specific response shaping must go into Go handlers (adding mobile-aware
  response fields) instead of the BFF
- Aggregation endpoints must be kept simple — complex fan-outs risk BFF timeout issues

## References

- `bff/src/` — Express.js BFF source
- `bff/Dockerfile.bff` — BFF container definition
- `docs/architecture.md` — Section 3 (System Architecture), BFF responsibilities
- `docs/decisions/ADR-025-bff-unified-client-gateway.md` — extends this decision to cover
  web and future clients (BFF as unified gateway, not mobile-only)
