---
id: ADR-025
title: "BFF is the unified client gateway — web, mobile, and future clients all route through BFF"
date: 2026-04-23
status: accepted
deciders: [matias]
tags: [adr, bff, architecture, security, web, gateway]
related_tasks: [task_4.1, CLSF-62]
related_frs: [FR-301]
supersedes: []
---

# ADR-025 — BFF as unified client gateway

## Status

`accepted`

## Context

ADR-009 defined the BFF as a thin proxy for the React Native mobile app. Wave 6 (CLSF-62)
introduces a web builder surface — the first non-mobile client. This raised two questions:

1. Should the web builder call the Go backend directly, or route through the BFF?
2. Is authentication harmonized across clients, or does each client implement its own
   auth flow against Go?

Inspection of the current BFF reveals that:

- `POST /bff/auth/login` and `POST /bff/auth/register` relay to Go and return a JWT.
  These endpoints have no mobile-specific logic — any HTTP client can use them.
- `GET|POST /bff/api/v1/*` transparently proxies all requests to Go `/api/v1/*`,
  forwarding the `Authorization: Bearer <jwt>` header unchanged.
- `app.use(cors())` is configured without origin restrictions — browsers can already call
  the BFF without CORS errors.
- The Go backend validates the JWT in `AuthMiddleware`. The BFF does not validate it —
  it only relays it. JWT validation is the Go backend's responsibility in all cases.

Technically, the web builder can already use the BFF today without code changes.
The only missing piece was an explicit architectural decision.

## Decision

The BFF is the **unified client gateway** for all FenixCRM clients.

- **Web builder** (Wave 6+): calls `POST /bff/auth/login` to obtain a JWT, then sends
  `Authorization: Bearer <jwt>` on all requests to `/bff/api/v1/*`.
- **Mobile app**: unchanged — continues through BFF as today.
- **No client calls Go directly** in production. The Go backend CORS policy should be
  restricted to internal origins only (BFF + localhost for local dev).
- The BFF thin-proxy constraint from ADR-009 is preserved: zero business logic, zero
  DB access, zero state. Harmonization does not add BFF complexity — it only formalizes
  the existing behavior as the intended pattern.

Auth flow for all clients:

```
Client (mobile / web / future)
  │
  ├─ POST /bff/auth/login { email, password }
  │      ↓ relay to Go /auth/login
  │      ← { token: "<jwt>" }
  │
  └─ GET|POST /bff/api/v1/* { Authorization: Bearer <jwt> }
         ↓ transparent proxy to Go /api/v1/*
         ← Go response (JWT validated by Go AuthMiddleware)
```

## Rationale

- The BFF already implements auth relay and transparent proxy. Harmonization requires
  no new code — only policy alignment.
- A single public surface (BFF) simplifies CORS management, rate limiting, and future
  API gateway concerns (logging, circuit breaking) without duplicating logic per client.
- Restricting Go's CORS to internal origins reduces the attack surface: a leaked JWT
  cannot be used against the Go backend directly from a browser without going through the
  BFF first (which can add rate limiting and logging).
- ADR-009's thin-proxy constraint is not violated: the BFF still does zero business logic.
  The web client uses the same auth + proxy contracts as mobile.

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Web builder calls Go directly | Requires CORS open on Go backend; splits auth flows across clients; inconsistent API surface |
| Web builder has its own BFF | Two BFF processes with duplicate auth relay logic; no benefit for a thin proxy pattern |
| Introduce a dedicated API gateway (nginx, Kong) | Overkill for MVP; adds operational complexity; the BFF already plays this role adequately |
| OAuth2 / session cookies for web, JWT for mobile | Two auth models to maintain; BFF already handles JWT relay cleanly for both |

## Consequences

**Positive:**
- One auth flow for all clients — lower cognitive overhead for new contributors.
- Go backend CORS can be locked down to internal origins — improved security posture.
- Future clients (CLI tool, third-party integrations) have a clear entry point.
- Web builder (CLSF-62) can be implemented without auth infrastructure work — it uses
  existing BFF endpoints.

**Negative / tradeoffs:**
- BFF becomes a single point of failure for all clients. Mitigation: Go backend remains
  accessible on internal network for health checks and ops tooling.
- CORS lockdown on Go is a follow-up task (not a Wave 6 blocker) — must be tracked
  explicitly to avoid being forgotten.
- SSE (copilot streaming) for the web client must route through the BFF SSE proxy, which
  was designed for mobile. Verify SSE proxy behavior with browser `EventSource` API before
  Wave 6 SSE work begins.

## Follow-up tasks

| Task | Priority | Description |
|------|----------|-------------|
| Go CORS lockdown | P1 | Restrict `Access-Control-Allow-Origin` on Go backend to BFF + localhost |
| BFF CORS tightening | P1 | Replace `cors()` with explicit origin allowlist in `bff/src/app.ts` |
| SSE proxy browser validation | P1 before SSE work | Verify `EventSource` from browser works through BFF SSE proxy |

## References

- `bff/src/app.ts` — BFF entry point, CORS and middleware registration
- `bff/src/middleware/authRelay.ts` — Bearer token relay
- `bff/src/routes/auth.ts` — Login/register relay
- `bff/src/routes/proxy.ts` — Transparent API proxy
- `docs/decisions/ADR-009-bff-thin-proxy.md` — Thin proxy constraint (unchanged)
