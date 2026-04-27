---
id: ADR-029
title: "BFF admin shell: HTMX read-only surface at /bff/admin/* over existing Go governance endpoints"
date: 2026-04-27
status: accepted
deciders: [matias]
tags: [adr, web, architecture, bff, admin, governance, maintenance]
related_tasks: [BFF-ADMIN-01, BFF-ADMIN-02, BFF-ADMIN-03, BFF-ADMIN-10, BFF-ADMIN-11, BFF-ADMIN-12, BFF-ADMIN-20, BFF-ADMIN-21a, BFF-ADMIN-21b, BFF-ADMIN-21c, BFF-ADMIN-21d, BFF-ADMIN-30, BFF-ADMIN-31, BFF-ADMIN-40, BFF-ADMIN-41, BFF-ADMIN-50, BFF-ADMIN-51, BFF-ADMIN-60, BFF-ADMIN-70, BFF-ADMIN-90]
related_frs: [FR-060, FR-070, FR-071, FR-200, FR-230, FR-231, FR-233, FR-243]
related_ucs: [UC-C1]
supersedes: []
---

# ADR-029 — BFF admin shell: HTMX read-only surface at `/bff/admin/*`

## Status

`accepted`

## Context

After the Carta language-server flow (CLSF-62..78) shipped the authoring shell at
`/bff/builder`, the **web administration surface** for governed-AI operations remained
unbuilt. The underlying Go endpoints for workflows, agent runs, approvals, audit, policy,
tools, and metrics existed and were reachable only via raw API calls or mobile-oriented
JSON proxies. Operators lacked a browser-based surface for governance oversight.

The question was: where and how to build it?

ADR-026 already established that HTMX + Express (BFF) is the web stack for the repository.
The admin surface faces the same trade-offs: low interactivity (list/filter/read-only
detail), zero client-side state, and a heavy cost for adding a fourth stack.

### Path and method corrections discovered during implementation

The original spec assumed several endpoints that differed from reality in `internal/api/routes.go`:

| Spec (original) | Real endpoint (verified 2026-04-27) |
|---|---|
| `POST /api/v1/workflows/{id}/activate` | `PUT /api/v1/workflows/{id}/activate` |
| `GET /api/v1/agent-runs` | `GET /api/v1/agents/runs` |
| `POST /api/v1/approvals/{id}/decision` | `PUT /api/v1/approvals/{id}` |
| `GET /api/v1/audit` | `GET /api/v1/audit/events` |
| `GET /api/v1/tools` | `GET /api/v1/admin/tools` |
| `GET /api/v1/metrics` | `GET /metrics` (root, Prometheus text) |

All BFF admin handlers use the **real** endpoints above.

### Policy endpoints gap and governance substitute

`GET /api/v1/policy/sets` and `GET /api/v1/policy/versions/{id}` did not exist at the
start of this work. Phase F was redesigned to proxy `GET /api/v1/governance/summary`
instead, which returns active quota policy states and recent usage events — materially
equivalent for an operator dashboard.

`GO-POLICY-READ-01` landed on 2026-04-27, adding `GET /api/v1/policy/sets` and
`GET /api/v1/policy/sets/{id}/versions`. BFF-ADMIN-51 was then built on top of both
the governance summary and the new policy set endpoints, rendering them on the same
`/bff/admin/policy` page.

## Decision

**Extend the HTMX + Express BFF pattern (ADR-026) to a web admin shell at `/bff/admin/*`.**

### Route map

| BFF route | Go backend endpoint | Purpose |
|---|---|---|
| `GET /bff/admin` | — | Dashboard shell (static counts placeholder) |
| `GET /bff/admin/workflows` | `GET /api/v1/workflows` | Workflow list + filter |
| `GET /bff/admin/workflows/:id` | `GET /api/v1/workflows/{id}` | Workflow detail (read-only) |
| `POST /bff/admin/workflows/:id/activate` | `PUT /api/v1/workflows/{id}/activate` | Activation form |
| `GET /bff/admin/agent-runs` | `GET /api/v1/agents/runs` | Agent run list + filter |
| `GET /bff/admin/agent-runs/:id` | `GET /api/v1/agents/runs/{id}` | Run detail (header, trace, evidence, tool calls, cost) |
| `GET /bff/admin/approvals` | `GET /api/v1/approvals` | Pending approvals queue |
| `GET /bff/admin/approvals/:id` | `GET /api/v1/approvals/{id}` | Decision form |
| `POST /bff/admin/approvals/:id/decision` | `PUT /api/v1/approvals/{id}` | Approve / reject relay |
| `GET /bff/admin/audit` | `GET /api/v1/audit/events` | Paginated audit trail |
| `GET /bff/admin/audit/:id` | `GET /api/v1/audit/events/{id}` | Audit record detail |
| `GET /bff/admin/policy` | `GET /api/v1/governance/summary` + `GET /api/v1/policy/sets` | Governance + policy sets |
| `GET /bff/admin/policy/:id/versions` | `GET /api/v1/policy/sets/{id}/versions` | Policy version drill-down |
| `GET /bff/admin/tools` | `GET /api/v1/admin/tools` | Tool catalog (read-only) |
| `GET /bff/admin/metrics` | `GET /metrics` | Prometheus counters rendered as dashboard |

### Hard constraints preserved

1. **BFF stays thin**: zero business logic, zero DB access, zero cross-tenant aggregation
   the backend does not already provide (ADR-009).
2. **Read-only in this ADR**: write paths for policy version creation and tool registration
   are deferred. Any mutation the admin shell performs (activation, approval decision) is
   a direct relay of an existing Go endpoint.
3. **Multi-tenant safety**: workspace identity comes exclusively from the JWT validated by
   the Go backend. No client-supplied workspace identifier is trusted.
4. **No new Go endpoints**: admin pages are constrained to endpoints already registered
   in `internal/api/routes.go`. Gaps are documented and deferred.
5. **CORS**: the admin shell is served from the BFF on the same origin as its API proxies.
   HTMX requests are same-origin and carry no `Origin` header. No new CORS allowlist
   entry is needed (see ADR-026 CORS note).

### Shared layout contract

Every admin page uses `adminLayout(title, body)` from `bff/src/routes/adminLayout.ts`,
which renders:

- HTMX CDN `<script>` tag
- Admin chrome: `<header>` with workspace badge (`id="admin-workspace-badge"`), nav,
  sign-out affordance
- Bearer-token localStorage relay (`fenix.admin.bearerToken` + `htmx:configRequest`)
- A `<main>` slot for the page-specific body

The landmark `class="page-title"` on the first `<h2>` inside each page body is the
structural anchor used by the BFF-ADMIN-90 navigation smoke test.

### Test strategy

- **Per-route Jest/Supertest unit tests** (Phases B–H): verify mock call args, data
  rendering, error paths, pagination, and mutation relay for each route file.
- **Navigation smoke test** (BFF-ADMIN-90, `bff/tests/admin.e2e.test.ts`): walks all
  8 primary routes, asserts HTTP 200 and `class="page-title"` landmark presence. Catches
  layout regressions without duplicating per-route logic.
- **HTTP snapshot suite** (`npm run http-snapshots`): explicitly **not** extended to cover
  admin HTML routes. Per-route Jest/Supertest coverage is sufficient. See
  `docs/plans/bff-http-snapshots-plan.md` for the reopen condition.

## Rationale

| Criterion | HTMX + BFF admin shell | Separate admin SPA |
|---|---|---|
| Stack count | 3 (no change) | 4 (+1) |
| Interactivity need | Low (list, filter, read detail, one-step form) | Overkill |
| Reuse of existing pattern | Full — mirrors `/bff/builder` | None |
| CI gates added | 0 | 1 new gate |
| Multi-tenant safety surface | BFF thin relay, backend enforces | Must replicate workspace enforcement |

## Alternatives considered

| Option | Why rejected |
|---|---|
| Separate React admin SPA | Fourth stack; maintenance cost not justified for low-interactivity admin views |
| Extend mobile app with admin screens | Mobile is deprioritized for wedge (ADR-022); operators need a browser surface |
| Serve admin HTML from Go directly | Violates Go backend's zero-UI-concern boundary |
| Reuse existing JSON proxy routes for HTML | Conflates mobile API contract with web rendering; no shared layout |

## Consequences

**Positive:**
- Repository remains at 3 stacks. No new CI gate, no new `node_modules`.
- Admin operators get a browser surface without requiring the mobile client or raw API.
- Governance oversight (approvals, audit, quota states) is exposed in a reviewable,
  evidence-first UI consistent with the project's governed-AI principles.
- Navigation smoke test (`BFF-ADMIN-90`) acts as a regression guard for the shared layout.

**Negative / tradeoffs:**
- Write paths (policy CRUD, tool registration) require a separate ADR before implementation.
- Server-side HTML rendering is less ergonomic than React for complex interactive forms.
- The governance page substitutes for a missing policy-sets read endpoint; if
  `GO-POLICY-READ-01` had not landed, the policy panel would have shown only quota data.

## Deferred

- Write paths for policy version creation and tool registration (new ADR required).
- Dashboard counts and recent-runs aggregation (placeholder rendered; requires a
  `/api/v1/admin/summary` Go endpoint or client-side HTMX fragment composition).
- Headless PNG rendering of admin pages (no concrete use case yet).

## References

- `bff/src/routes/admin.ts` — admin router mount
- `bff/src/routes/adminLayout.ts` — shared HTMX chrome
- `bff/src/routes/adminAuth.ts` — bearer relay, 401 redirect helper
- `bff/src/routes/adminWorkflows.ts`, `adminAgentRuns.ts`, `adminAgentRunsFragments.ts`,
  `adminApprovals.ts`, `adminApprovalsFragments.ts`, `adminAudit.ts`, `adminPolicy.ts`,
  `adminTools.ts`, `adminMetrics.ts` — per-section route handlers
- `bff/tests/admin.e2e.test.ts` — navigation smoke test (BFF-ADMIN-90)
- `docs/decisions/ADR-026-web-builder-stack.md` — HTMX + BFF pattern origin
- `docs/decisions/ADR-009-bff-thin-proxy.md` — thin proxy constraint
- `docs/decisions/ADR-025-bff-unified-client-gateway.md` — BFF unified gateway
- `docs/plans/bff-admin-surface-gap-closure.md` — implementation handoff and task graph
