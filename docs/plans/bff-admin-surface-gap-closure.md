---
doc_type: handoff
id: BFF-ADMIN-GAP
title: BFF web administration surface — agent handoff
status: completed
phase: post-CLSF
week:
tags: [handoff, bff, admin, governance, web]
fr_refs: [FR-060, FR-070, FR-071, FR-200, FR-230, FR-231, FR-233, FR-243]
uc_refs: [UC-C1]
blocked_by: []
blocks: []
files_affected: []
created: 2026-04-26
completed:
---

# Handoff: BFF web administration surface

## You are picking up

A gap detected after the Carta language-server flow shipped: Waves 6 and 7
(`docs/plans/carta-language-server-flow.md`, tasks `CLSF-62..CLSF-78`)
delivered the **authoring** shell at `/bff/builder` only. The **web
administration surface** for governed-AI operations was never built. The
underlying Go endpoints exist; they are reachable today only via raw API
or thin JSON proxies meant for the mobile client.

Your job: build the missing HTMX admin pages inside the BFF that surface
the existing Go governance endpoints (workflows, agent runs, approvals,
audit, policy, tools, metrics).

Follow-up note `2026-04-28`: workflow authoring inside the BFF admin is now
tracked separately in `docs/plans/bff-admin-workflow-authoring-plan.md`. This
handoff remains the source of truth for the original admin read/operate shell,
while create/edit builder integration moved into that follow-up plan.

## Read these first (non-negotiable)

1. `CLAUDE.md` (project root) — architecture rules, reporting cadence,
   handoff scope discipline, push discipline.
2. `docs/architecture.md` — 3-tier rule. The BFF is a thin
   proxy/aggregator with **zero business logic and zero DB access**.
3. `docs/plans/carta-language-server-flow.md` — Wave 6/7 builder pattern
   (HTMX CDN, bearer-token localStorage relay, two-pane layout).
4. `docs/decisions/ADR-026-*` — BFF web shell pattern. Mirror it.
5. `docs/decisions/ADR-008-*` — protected `/api/v1/*` consumes JWT with
   `workspace_id`. Never trust client-supplied workspace identifiers.
6. `docs/decisions/ADR-025-*` — CORS allowlist mechanism.
7. `bff/src/routes/builder.ts` — reference implementation for HTMX
   fragments and bearer-token relay. Match its structure.
8. `docs/plans/bff-http-snapshots-plan.md` and
   `bff/scripts/snapshots/README.md` — existing `npm run http-snapshots`
   suite for BFF endpoints (JSON + HTML report). **Decision: do not
   extend this suite to cover the admin surface.** Per-route Jest /
   Supertest coverage (Phases B–H) is sufficient. The suite stays as-is;
   if a future audit or client requirement demands black-box HTML
   captures of admin pages, reopen this decision in a separate task.
9. `docs/plans/maestro-screenshot-auth-bypass-plan.md`,
   `docs/plans/maestro-screenshot-migration.md`,
   `docs/plans/mobile_screenshots_runner_fix_plan.md`,
   `docs/plans/fr304_screenshot_coverage_gaps.md`,
   `mobile/maestro/seed-and-run.sh`, and the Maestro flows
   (`auth-surface.yaml`, `authenticated-audit.yaml`, `visual-audit.yaml`,
   `crm-mutation-case.yaml`) — existing mobile screenshot pipeline. Do
   not duplicate seeding or auth-bypass logic; Phase J reuses these
   primitives where applicable.

## Hard constraints

- **No new Go endpoints** in this handoff. If a page needs a backend
  endpoint that does not exist, stop, document the gap as a separate
  task, and do not widen the Go surface.
- **BFF stays thin**: no business logic, no database access, no
  cross-tenant aggregation that the backend does not already provide.
- **HTMX only**: no SPA, no React on the web admin. Reuse the CDN
  approach from the builder shell.
- **Multi-tenant safety**: rely on backend `workspace_id` enforcement via
  JWT; never accept a workspace identifier from the client.
- **TDD per task**: write the Jest/Supertest test first, then the
  handler. Project rule: no commit if tests are broken.
- **One task at a time**: present a task card, wait for explicit
  approval, implement, run focused tests, close with the standard
  outcome report, then propose the next task. Never batch.
- **Scope discipline (handoff rule)**: do not pull in files, services,
  routes, or constants that are not listed in this document. Exploration
  context informs your work; it does not extend the spec.
- **Pre-push hook is mandatory**: run `make install-hooks` once after
  clone. Do not push until BFF QA passes locally
  (`cd bff && npm run lint && npm run build && npm test`).

## Out of scope

- New Go endpoints. Mobile parity. Agent Studio (FR-240/241/242). Eval
  suites UI. Marketplace. Any authoring feature already shipped under
  `CLSF-62..78`. Write paths for policy versions and tool registry (read
  only in this handoff; writes need a separate ADR).
- Extending `npm run http-snapshots` to cover admin HTML routes. The
  suite stays as-is; admin coverage is provided by per-route Jest /
  Supertest tests in Phases B–H. No admin-only capture pipeline.
- Headless PNG rendering of admin pages. Deferred until a concrete
  use case appears.

## Inventory of the gap

### Original spec vs. reality (verified 2026-04-27 against `internal/api/routes.go`)

The table below maps every capability the spec assumed to the **actual**
Go endpoint. Discrepancies are flagged and resolved inline.

| Capability                     | Spec endpoint (original)                     | Real endpoint (verified)                              | Delta / note |
|-------------------------------|----------------------------------------------|-------------------------------------------------------|--------------|
| List workflows                | `GET /api/v1/workflows`                      | `GET /api/v1/workflows` ✅                            | — |
| Workflow detail               | `GET /api/v1/workflows/{id}`                 | `GET /api/v1/workflows/{id}` ✅                       | — |
| Activate / deactivate         | `POST /api/v1/workflows/{id}/activate`       | `PUT /api/v1/workflows/{id}/activate` ✅              | Method is PUT not POST |
| List agent runs               | `GET /api/v1/agent-runs`                     | `GET /api/v1/agents/runs` ✅                          | Path is `/agents/runs` |
| Agent run detail              | `GET /api/v1/agent-runs/{id}`                | `GET /api/v1/agents/runs/{id}` ✅                     | Path is `/agents/runs/{id}` |
| List approvals                | `GET /api/v1/approvals`                      | `GET /api/v1/approvals` ✅                            | — |
| Decide approval               | `POST /api/v1/approvals/{id}/decision`       | `PUT /api/v1/approvals/{id}` ✅                       | Method is PUT, no `/decision` suffix |
| Audit trail (list)            | `GET /api/v1/audit`                          | `GET /api/v1/audit/events` ✅                         | Path suffix `/events` required |
| Audit record detail           | —                                            | `GET /api/v1/audit/events/{id}` ✅                    | — |
| Policy sets / versions        | `GET /api/v1/policy/sets`, `.../versions/`   | **DOES NOT EXIST** ❌                                 | No policy read endpoints in Go — see gap below |
| Governance / quota summary    | (not in spec)                                | `GET /api/v1/governance/summary` ✅                   | Returns `recentUsage` + `quotaStates`; use for Phase F substitute |
| Tool registry                 | `GET /api/v1/tools`                          | `GET /api/v1/admin/tools` ✅                          | Path is `/admin/tools` |
| Metrics                       | `GET /api/v1/metrics`                        | `GET /metrics` (root, not under `/api/v1`) ✅         | Prometheus text format; already relayed by `bff/src/routes/metrics.ts` |

### Gap: Policy endpoints do not exist in Go

`GET /api/v1/policy/sets` and `GET /api/v1/policy/versions/{id}` are
**not registered** in `internal/api/routes.go`. No handler file for policy
read exists. The `PolicyEngine` and `PolicySet` domain types exist in
`internal/domain/policy/` but are not exposed via HTTP.

**Resolution (decided 2026-04-27)**:
- `BFF-ADMIN-50` and `BFF-ADMIN-51` as originally designed **cannot be
  built** without new Go endpoints. Creating those is out of scope for
  this handoff (constraint: no new Go endpoints).
- **Substitute**: Phase F is redesigned as a **Governance page** using
  `GET /api/v1/governance/summary`, which returns active quota policies
  (with limit, enforcement mode, current consumption) and recent usage
  events. This is materially equivalent for an admin operator: it shows
  the active policy constraints and their real-time state without
  requiring the missing policy CRUD endpoints.
- The missing policy read endpoints are tracked as a deferred gap in the
  `decisions` section at the bottom of this document.

### Path and method corrections applied to task specs

Tasks already shipped (BFF-ADMIN-20 through BFF-ADMIN-31) used the
correct real paths found at implementation time. The table above
documents the deltas for tasks not yet built.

Tasks **BFF-ADMIN-40 / BFF-ADMIN-41** (audit) were already built using
the correct `/api/v1/audit/events` path. ✅

Remaining tasks use the corrected real paths documented here.

## Target shape

A single new BFF mount `/bff/admin/*` rendering HTMX pages that share an
admin chrome (header with workspace badge, nav, sign-out). Each page is
a thin proxy aggregator over existing Go endpoints. Routes:

```
/bff/admin/                          dashboard (counts + recent runs)
/bff/admin/workflows                 list + filter  → GET /api/v1/workflows
/bff/admin/workflows/:id             detail         → GET /api/v1/workflows/{id}
/bff/admin/agent-runs                list + filter  → GET /api/v1/agents/runs
/bff/admin/agent-runs/:id            detail         → GET /api/v1/agents/runs/{id}
/bff/admin/approvals                 pending list   → GET /api/v1/approvals
/bff/admin/approvals/:id             decision form  → PUT /api/v1/approvals/{id}
/bff/admin/audit                     paginated list → GET /api/v1/audit/events
/bff/admin/audit/:id                 record detail  → GET /api/v1/audit/events/{id}
/bff/admin/policy                    governance     → GET /api/v1/governance/summary
/bff/admin/tools                     tool catalog   → GET /api/v1/admin/tools
/bff/admin/metrics                   infra counters → GET /metrics (Prometheus text)
```

> **Note**: `/bff/admin/policy` shows governance summary (quota states +
> recent usage) as a substitute for the missing policy read endpoints.
> When `GO-POLICY-READ-01` lands, this route can be extended to show
> policy set detail alongside the governance data.

## Effort scale

- **Low**: <1h, single file or thin handler, no new patterns.
- **Medium-Low**: 1–3h, one route file with multiple fragments, follows
  existing pattern.
- **Medium**: 3–6h, multiple coordinated fragments or non-trivial
  rendering. Original High-effort items have been split below into
  Medium-or-smaller pieces.

## Task graph (dependency-ordered)

Original High-effort items have been split. Every task has Effort and
Reasoning. Sequence below is a topological order; tasks at the same
indentation level can be parallelized once their predecessors are done.

### Phase A — Foundation (must land first)

#### `BFF-ADMIN-01` — Mount `/bff/admin` router and shared layout

- **Effort**: Medium-Low
- **Reasoning**: Establishes the entry point and shared HTMX chrome
  every later page consumes. Mirror `bff/src/routes/builder.ts` for
  CDN script tags, layout, and bearer-token localStorage relay. No
  proxy calls yet — render an empty shell with nav placeholders.
- **Depends on**: none.
- **Files (expected)**: `bff/src/routes/admin.ts`,
  `bff/src/routes/adminLayout.ts` (helper), `bff/src/app.ts` (mount),
  `bff/__tests__/admin.test.ts`.
- **QA gate**: `cd bff && npm run lint && npm run build && npm test --
  admin.test.ts`.

#### `BFF-ADMIN-02` — Bearer-token relay and 401 handling

- **Effort**: Low
- **Reasoning**: Reuse the bearer relay pattern from CLSF-62. On 401
  from upstream, redirect to the existing auth flow rather than leaking
  raw status. Pure middleware glue.
- **Depends on**: `BFF-ADMIN-01`.
- **Files**: `bff/src/routes/admin.ts` (middleware wiring), tests.
- **QA gate**: focused jest run on the new test file.

#### `BFF-ADMIN-03` — CORS allowlist update for admin origin

- **Effort**: Low
- **Reasoning**: ADR-025 requires the admin origin to be explicit.
  Update the allowlist default and document the new origin in a
  follow-up note to ADR-026.
- **Depends on**: `BFF-ADMIN-01`.
- **Files**: `bff/src/app.ts`, ADR follow-up note, tests
  (`cors.test.ts`).
- **QA gate**: focused jest run on `cors.test.ts`.

### Phase B — Workflow administration

Originally one Medium task; split into three so each task ships an
independent HTMX fragment.

#### `BFF-ADMIN-10` — Workflows list page

- **Effort**: Medium-Low
- **Reasoning**: HTMX table fragment over `GET /api/v1/workflows`.
  Pagination and filter (status, name) are query-param relays. No
  client-side state.
- **Depends on**: `BFF-ADMIN-02`.
- **Files**: `bff/src/routes/adminWorkflows.ts`,
  `bff/__tests__/adminWorkflows.test.ts`.
- **QA gate**: focused jest run.

#### `BFF-ADMIN-11` — Workflow detail page (read-only)

- **Effort**: Medium-Low
- **Reasoning**: Renders source, conformance, activation status, and a
  link into the existing builder. Read-only here; activation is
  separated into `BFF-ADMIN-12` to keep blast radius small.
- **Depends on**: `BFF-ADMIN-10`.
- **Files**: same router, tests.
- **QA gate**: focused jest run.

#### `BFF-ADMIN-12` — Activation / deactivation form

- **Effort**: Medium-Low
- **Reasoning**: Single HTMX form posting through the BFF to the Go
  activation endpoint. Surfaces backend diagnostics on failure without
  retrying. State change is mutating, so test both happy path and
  upstream-error path.
- **Depends on**: `BFF-ADMIN-11`.
- **Files**: same router, tests.
- **QA gate**: focused jest run.

### Phase C — Agent run review

Original `BFF-ADMIN-21` was High; split into four Medium-or-smaller
fragments rendered independently.

#### `BFF-ADMIN-20` — Agent runs list with filters

- **Effort**: Medium-Low
- **Reasoning**: HTMX list over `GET /api/v1/agent-runs` with filter
  controls (status, agent, date range). Filters are query-param
  relays.
- **Depends on**: `BFF-ADMIN-02`.
- **Files**: `bff/src/routes/adminAgentRuns.ts`, tests.
- **QA gate**: focused jest run.

#### `BFF-ADMIN-21a` — Agent run detail: header and outcome ✅ DONE

- **Effort**: Low
- **Reasoning**: Top section of the detail page: run id, agent, status,
  outcome, abstention reason, timestamps. Pure formatting of upstream
  payload.
- **Depends on**: `BFF-ADMIN-20`.
- **Files**: `bff/src/routes/adminAgentRuns.ts` (handler GET /:id, `AgentRunDetail` interface, `buildDetailHeader`), `bff/tests/adminAgentRuns.test.ts` (11 new tests — BFF-ADMIN-21a suite).
- **QA gate**: 25/25 tests pass, full suite 226/226, lint clean, build clean.

#### `BFF-ADMIN-21b` — Agent run detail: reasoning trace fragment ✅ DONE

- **Effort**: Medium-Low
- **Reasoning**: Renders the trace array as a readable list. Long
  payloads must paginate by step index, not by client truncation.
- **Depends on**: `BFF-ADMIN-21a`.
- **Files**: `bff/src/routes/adminAgentRuns.ts` (`buildTraceFragment`, `TRACE_PAGE_SIZE`, `TraceStep` interface, updated `buildDetailHeader` and `GET /:id` handler), `bff/tests/adminAgentRuns.test.ts` (9 new tests — BFF-ADMIN-21b suite).
- **QA gate**: 235/235 tests pass, lint clean, build clean.

#### `BFF-ADMIN-21c` — Agent run detail: evidence pack fragment ✅ DONE

- **Effort**: Medium-Low
- **Reasoning**: Renders evidence sources, snippets, scores,
  timestamps, confidence tier. Evidence dedupe is already enforced in
  the backend; the BFF only displays.
- **Depends on**: `BFF-ADMIN-21a`.
- **Files**: `bff/src/routes/adminAgentRuns.ts` (`EvidenceItem` interface, `CONFIDENCE_COLORS`, `BADGE_NEUTRAL`, `PANEL_CARD` constants, `buildEvidenceFragment`), `bff/tests/adminAgentRuns.test.ts` (8 new tests — BFF-ADMIN-21c suite).
- **QA gate**: 243/243 tests pass, lint clean, build clean.

#### `BFF-ADMIN-21d` — Agent run detail: tool calls and cost panel ✅ DONE

- **Effort**: Medium-Low
- **Reasoning**: Renders tool call list with status, latency, idempotency
  key, plus cost panel (tokens, euros). All values come from the
  backend payload — do not compute aggregates in the BFF.
- **Depends on**: `BFF-ADMIN-21a`.
- **Files**: `bff/src/routes/adminAgentRunsFragments.ts` (new — extracted `buildToolCallsFragment`, `buildCostPanel`, `buildTraceFragment`, `buildEvidenceFragment`, `buildDetailHeader`, shared interfaces and constants; triggered by max-lines:200 gate), `bff/src/routes/adminAgentRuns.ts` (refactored to import from fragments module), `bff/tests/adminAgentRuns.test.ts` (12 new tests — BFF-ADMIN-21d suite).
- **QA gate**: 54/54 focused tests pass, 255/255 full suite, lint clean, build clean.

### Phase D — Approvals (web reviewer surface)

#### `BFF-ADMIN-30` — Approvals queue list ✅ DONE

- **Effort**: Low
- **Reasoning**: HTMX list over `GET /api/v1/approvals`. Filter to
  pending by default. Existing JSON proxy under `/bff/approvals`
  remains for mobile and is untouched.
- **Depends on**: `BFF-ADMIN-02`.
- **Files**: `bff/src/routes/adminApprovals.ts` (new router, `extractParams` extracted to satisfy complexity gate), `bff/src/routes/admin.ts` (mount), `bff/tests/adminApprovals.test.ts` (12 tests).
- **QA gate**: 12/12 focused tests pass, 267/267 full suite, lint clean, build clean.

#### `BFF-ADMIN-31` — Decision form (approve / reject + reason) ✅ DONE

- **Effort**: Medium-Low
- **Reasoning**: HTMX form posts to the Go decision endpoint. Mutating
  call: cover happy path, validation error, and upstream error. Surface
  the backend reason field verbatim.
- **Depends on**: `BFF-ADMIN-30`.
- **Files**: `bff/src/routes/adminApprovals.ts` (GET /:id detail + form, POST /:id/decision relay; `req.body ?? {}` guard for missing body on no-send requests), `bff/tests/adminApprovals.test.ts` (13 new tests — BFF-ADMIN-31 suite).
- **QA gate**: 25/25 focused tests pass, 280/280 full suite, lint clean, build clean.

### Phase E — Audit trail

#### `BFF-ADMIN-40` — Audit list with filters ✅ DONE

- **Effort**: Medium-Low
- **Reasoning**: Paginated list over `GET /api/v1/audit/events` (real
  path, corrected from spec) with filter controls (actor, resource_type,
  date_from, date_to). Pagination by offset (backend uses
  `parsePaginationParams`, not cursor; nextCursor field supported if
  present). Relay all filters as query params.
- **Depends on**: `BFF-ADMIN-02`.
- **Files**: `bff/src/routes/adminAudit.ts` (GET / handler, filter form,
  pagination link, `extractAuditParams`), `bff/tests/adminAudit.test.ts`
  (17 tests — BFF-ADMIN-40 suite).
- **QA gate**: 17/17 focused tests pass, 307/307 full suite, lint clean,
  build clean.

#### `BFF-ADMIN-41` — Audit record detail fragment ✅ DONE

- **Effort**: Low
- **Reasoning**: Read-only formatted view of one immutable audit
  record. Real path: `GET /api/v1/audit/events/{id}`. No POST form,
  no mutation affordance.
- **Depends on**: `BFF-ADMIN-40`.
- **Files**: `bff/src/routes/adminAudit.ts` (GET /:id handler,
  `buildDetailBody`, policy checks section), `bff/tests/adminAudit.test.ts`
  (10 tests — BFF-ADMIN-41 suite).
- **QA gate**: 27/27 focused tests pass (BFF-ADMIN-40 + 41 combined),
  307/307 full suite, lint clean, build clean.

### Phase F — Governance page (replaces original Policy phase)

> **Redesign note (2026-04-27)**: The original `BFF-ADMIN-50/51` assumed
> `GET /api/v1/policy/sets` and `GET /api/v1/policy/versions/{id}`, which
> do not exist in the Go backend. Phase F is redesigned to use
> `GET /api/v1/governance/summary` (available, verified), which returns
> active quota policies with live consumption state and recent usage events.
> The deferred gap (missing policy read endpoints) is tracked in the
> Decisions section. BFF-ADMIN-50/51 IDs are preserved for backlog
> continuity but their spec is replaced below.

#### `BFF-ADMIN-50` — Governance page (quota policies + recent usage)

- **Effort**: Low
- **Reasoning**: Proxy of `GET /api/v1/governance/summary`. Renders two
  sections: active quota policies (policy type, metric, limit,
  enforcement mode, current consumption vs. limit, period) and recent
  usage events (last 20). Read-only. No policy write paths in this
  handoff.
- **Real endpoint**: `GET /api/v1/governance/summary` — returns
  `{ recentUsage: UsageEvent[], quotaStates: QuotaStateItem[] }`.
  `QuotaStateItem` shape: `{ policyId, policyType, metricName,
  limitValue, resetPeriod, enforcementMode, currentValue, periodStart,
  periodEnd, lastEventAt?, statePresent }`.
- **Depends on**: `BFF-ADMIN-02`.
- **Files**: `bff/src/routes/adminPolicy.ts` (renamed from original
  plan; renders governance data), `bff/tests/adminPolicy.test.ts`.
- **QA gate**: focused jest run.

#### `BFF-ADMIN-51` — Policy sets and versions detail ✅ DONE

- **Status**: COMPLETED 2026-04-27.
- **Real endpoints**:
  - `GET /api/v1/policy/sets` — list with optional `?is_active=true|false` filter
  - `GET /api/v1/policy/sets/{id}/versions` — versions for a set
- **Effort**: Low
- **Reasoning**: HTMX page proxying the two new Go endpoints. Read-only; same pattern as
  BFF-ADMIN-50 (governance page). No new decisions needed.
- **Depends on**: `BFF-ADMIN-02`, `GO-POLICY-READ-01` ✅.
- **Files**: `bff/src/routes/adminPolicy.ts` (routes `/bff/admin/policy` and
  `/bff/admin/policy/:id/versions` — implemented alongside BFF-ADMIN-50),
  `bff/tests/adminPolicy.test.ts` (22 tests — 11 BFF-ADMIN-50 + 11 BFF-ADMIN-51).
- **QA gate**: 22/22 focused tests pass (`npm test -- --testPathPattern=adminPolicy`).

### Phase G — Tools registry (read-only)

#### `BFF-ADMIN-60` — Tools list and schema/permissions detail

- **Effort**: Low
- **Reasoning**: Read-only catalog of registered tools. Schema and
  permission snapshot come straight from the backend.
- **Real endpoint**: `GET /api/v1/admin/tools` (not `/api/v1/tools` as
  the original spec assumed — verified in `routes.go` line 316).
- **Depends on**: `BFF-ADMIN-02`.
- **Files**: `bff/src/routes/adminTools.ts`, `bff/tests/adminTools.test.ts`.
- **QA gate**: focused jest run.

### Phase H — Metrics dashboard

**Backend reality check (verified 2026-04-27)**: `GET /metrics` (root
path, not under `/api/v1`) returns Prometheus text format
(`text/plain; version=0.0.4`) with exactly three counters:
`fenixcrm_requests_total`, `fenixcrm_request_errors_total`,
`fenixcrm_uptime_seconds`. There is no JSON payload, no per-agent
breakdown, no token/cost/abstention data, and no `sort_by` support.
See `internal/api/handlers/metrics.go`.

The BFF metrics proxy in `bff/src/routes/metrics.ts` already relays this
endpoint for mobile. Do not invent fields that do not exist in the backend.

Additionally, `GET /api/v1/governance/summary` (used in Phase F /
BFF-ADMIN-50) provides `recentUsage` and `quotaStates` which include
per-policy cost and consumption data — this is the closest available
substitute for "token/cost/runs" metrics. If BFF-ADMIN-70 is implemented
after BFF-ADMIN-50, consider linking to the governance page rather than
duplicating data.

#### `BFF-ADMIN-70` — Metrics page (proxy of available backend data)

- **Effort**: Low
- **Reasoning**: Render the three Prometheus counters (request total,
  error total, uptime) as a simple dashboard page. Pure proxy of
  `GET /metrics` (root path) Prometheus text payload — parse the text
  format and display as labeled cards. Do not aggregate, derive, or
  fabricate metrics not present in the response. Link to the Governance
  page (`/bff/admin/policy`) for per-policy quota consumption.
- **Real endpoint**: `GET /metrics` (root, Prometheus text).
- **Depends on**: `BFF-ADMIN-02`.
- **Files**: `bff/src/routes/adminMetrics.ts`, `bff/tests/adminMetrics.test.ts`.
- **QA gate**: focused jest run.

### Phase J — Mobile screenshot regression sanity

The existing capture pipelines stay untouched in this handoff:

- **BFF HTTP snapshots** (`cd bff && npm run http-snapshots`) — out of
  scope. Admin HTML coverage is provided by per-route Jest / Supertest
  tests in Phases B–H. The decision to keep the suite as-is is
  documented in this handoff's "Read these first" section, item 8, and
  must not be re-litigated here. If a future audit or client
  requirement demands black-box HTML captures, open a separate task
  rather than amending this handoff.
- **Mobile Maestro screenshots** (`mobile/maestro/seed-and-run.sh`
  with `auth-surface.yaml` then `authenticated-audit.yaml` /
  `visual-audit.yaml` / `crm-mutation-case.yaml`) — not modified.

Phase J keeps a single regression-sanity task because the **seeder is
shared** between BFF backend work and the mobile screenshot pipeline
(`scripts/e2e_seed_mobile_p2.go`). Any inadvertent change there during
Phases A–H must be caught before closing the handoff.

#### `BFF-ADMIN-J4` — Mobile screenshot regression sanity ✅ DONE

- **Effort**: Low
- **Reasoning**: Confirm `mobile/maestro/seed-and-run.sh` and
  `visual-audit.yaml` still pass after Phases A–H land. Admin work
  should not touch mobile, but the shared seeder
  (`scripts/e2e_seed_mobile_p2.go`) is a known cross-track surface.
- **Completed**: 2026-04-27. All 36 screenshots captured. Both phases
  passed 100% on `Pixel_7_API_33` (Pixel 7, API 33).
- **Fix applied**: Android ANR dialog ("Process system isn't responding")
  appeared on cold-start and at governance route transitions due to
  emulator load. Added `repeat: times: 8 / runFlow: when: / tapOn: Wait`
  dismiss loops in `auth-surface.yaml` (Phase 1) and
  `authenticated-audit.yaml` (governance/audit + governance/usage steps).
  No regressions in existing flow logic.
- **Seeder unchanged**: `scripts/e2e_seed_mobile_p2.go` — 0 lines of diff
  from BFF admin commits. Confirmed no cross-track contamination.
- **Files changed**: `mobile/maestro/auth-surface.yaml`,
  `mobile/maestro/authenticated-audit.yaml`.
- **QA gate**: `bash mobile/maestro/seed-and-run.sh` — exit 0, 36/36
  screenshots in `mobile/artifacts/screenshots/`.

### Phase I — Closeout

#### `BFF-ADMIN-90` — End-to-end navigation smoke test ✅ DONE

- **Effort**: Medium-Low
- **Reasoning**: Supertest suite that walks dashboard → workflows →
  agent runs → approvals → audit → policy → tools → metrics, asserting
  200 status and presence of each landmark element. Catches regressions
  in the shared layout.
- **Depends on**: every Phase B–H task.
- **Files**: `bff/tests/admin.e2e.test.ts` (16 tests — BFF-ADMIN-90 suite).
- **QA gate**: 16/16 focused tests pass, 375/375 full suite, lint clean, build clean.

#### `BFF-ADMIN-91` — Architecture and ADR documentation ✅ DONE

- **Effort**: Low
- **Reasoning**: Update `docs/architecture.md` to record the admin
  shell. Add a sibling ADR to ADR-026 documenting the admin pattern,
  routes, and read-only constraint. Add a short note in
  `docs/plans/bff-http-snapshots-plan.md` recording the explicit
  decision to **not** extend the snapshot suite to admin HTML routes,
  with the reopen condition (future audit or client requirement).
- **Depends on**: `BFF-ADMIN-01`.
- **Files**: `docs/architecture.md` (new "BFF web shell routes" subsection under Section 10),
  `docs/decisions/ADR-029-bff-admin-shell.md` (new — full route map, constraints, rationale),
  `docs/plans/bff-http-snapshots-plan.md` (new "Decision: admin HTML routes are NOT covered" section).
- **QA gate**: documentation review ✅.

#### `BFF-ADMIN-92` — FR/UC dashboard update + mobile regression record ✅ DONE

- **Effort**: Low
- **Reasoning**: Update `docs/dashboards/fr-uc-status.md` so FR-060,
  FR-070, FR-071 reflect the new web surface coverage. Adjust UC-C1
  coverage row if applicable. Record the result of the mobile
  regression run (`BFF-ADMIN-J4`); if it was deferred, note the
  reason.
- **Depends on**: `BFF-ADMIN-91`, `BFF-ADMIN-J4`.
- **Files**: `docs/dashboards/fr-uc-status.md` (FR-060/070/071 updated, UC-C1 updated, ADR-029 added to index, new Section 6 closeout record with BFF-ADMIN-J4 deferred note and test suite state).
- **QA gate**: documentation review ✅.

## Verification gates (project-wide rules apply)

- Each task: `cd bff && npm run lint && npm run build && npm test --
  <focused-test-file>`.
- Before any commit, the whole BFF suite must be green
  (`cd bff && npm test`). Project rule: do not commit with broken
  tests.
- Pre-push hook auto-runs the appropriate QA gate. Do not bypass.
- Agent attribution: before commit run
  `export AI_AGENT="<your-agent-id>"` and
  `git config fenix.ai-agent "<your-agent-id>"`.

## Reporting cadence (mandatory per project CLAUDE.md)

Open every task with a task card:

```
Tarea: <BFF-ADMIN-NN>
Resumen: <one or two sentences>
Código afectado: <expected files>
Esfuerzo/razonamiento: <Low | Medium-Low | Medium> — <brief reason>
Tokens estimado: ~N
```

Close every task with an outcome report:

```
Resultado: <what changed>
Verificación: <commands run>
Archivos afectados: <files changed>
Complejidad: <Baja | Media | Alta | Muy alta>
Tokens: ~N
```

After closing, propose the next task card and **wait for explicit
approval** before starting.

## Decisions (resolved before Phase A)

1. **Admin surface at `/bff/admin/*`** — Approved. Reuse BFF per
   ADR-026; no separate web app.
2. **Policy and tools read-only in this handoff** — Approved. Write
   paths deferred to a separate ADR.
3. **Metrics as pure proxy of `GET /metrics`** — Approved. BFF
   stays thin; no new aggregation logic. Real path is root `/metrics`,
   not `/api/v1/metrics` (verified 2026-04-27).
4. **`BFF-ADMIN-J4` emulator access** — Pre-authorized. Run
   `bash mobile/maestro/seed-and-run.sh` without requesting
   per-action approval. If the emulator is unavailable at task time,
   defer and note it in `BFF-ADMIN-92`; do not block the handoff.

## Decisions added during implementation (2026-04-27)

5. **Phase F redesigned: governance page replaces policy sets** —
   `GET /api/v1/policy/sets` and `GET /api/v1/policy/versions/{id}`
   do not exist in Go. Phase F (BFF-ADMIN-50) is redesigned to proxy
   `GET /api/v1/governance/summary` instead, which provides active
   quota policy states and recent usage. BFF-ADMIN-51 is deferred
   pending Go endpoint creation.

6. **Audit path correction** — Real path is `GET /api/v1/audit/events`
   (not `/api/v1/audit`). BFF-ADMIN-40/41 were implemented with the
   correct path. The adminAudit.ts router proxies
   `/api/v1/audit/events` for list and `/api/v1/audit/events/{id}` for
   detail.

7. **Tools path correction** — Real path is `GET /api/v1/admin/tools`
   (not `/api/v1/tools`). BFF-ADMIN-60 spec updated accordingly.

8. **Agent runs path correction** — Real path is
   `GET /api/v1/agents/runs` (not `/api/v1/agent-runs`).
   BFF-ADMIN-20/21a-21d were implemented with the correct path.

9. **Approval decision method correction** — Real endpoint is
   `PUT /api/v1/approvals/{id}` (not
   `POST /api/v1/approvals/{id}/decision`). BFF-ADMIN-31 was
   implemented with the correct method.

## Deferred gap: policy read endpoints — RESOLVED ✅

- **GO-POLICY-READ-01 completed 2026-04-27.**
- `GET /api/v1/policy/sets` and `GET /api/v1/policy/sets/{id}/versions` now exist.
- Handler: `internal/api/handlers/policy.go`. Tests: `internal/api/handlers/policy_test.go` (10 tests).
- Routes registered in `internal/api/routes.go` under `/api/v1/policy`.
- BFF-ADMIN-51 is now unblocked and ready for implementation.
