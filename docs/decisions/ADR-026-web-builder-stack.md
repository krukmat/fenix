---
id: ADR-026
title: "Web builder surface uses HTMX + Express (BFF) over a separate React SPA"
date: 2026-04-23
status: accepted
deciders: [matias]
tags: [adr, web, architecture, bff, maintenance, stack]
related_tasks: [CLSF-62, CLSF-63, CLSF-64, CLSF-65, CLSF-66]
related_frs: [FR-241]
supersedes: []
---

# ADR-026 — Web builder stack: HTMX + Express over separate React SPA

## Status

`accepted`

## Context

CLSF-62 scaffolded a standalone React 19 + Vite SPA in `web/`. Before continuing with
CLSF-63–66, the question of stack maintenance cost was raised explicitly.

The repository already has three distinct technology stacks:

| Layer | Stack | Package manager / CI gate |
|---|---|---|
| Backend | Go 1.22+ | `go.mod`, `make test`, `make lint` |
| BFF | Express 5 + TypeScript | `bff/package.json`, `bff/jest` |
| Mobile | React Native 0.81 + Expo 54 | `mobile/package.json`, EAS Build |

Adding a standalone React SPA in `web/` would be a **fourth stack** with its own
`package.json`, `node_modules`, linter config, Vite config, and CI gate. The
maintenance cost accumulates across every dependency update, every Node version bump,
and every dev that needs to context-switch between four build systems.

### Interactivity profile of the web builder (Waves 6–7)

| Feature | Interactivity level |
|---|---|
| DSL text editor (CLSF-63) | Textarea + debounced API call + error list render |
| Read-only graph pane (CLSF-64) | Static SVG/Canvas render from JSON fixture |
| Inspector panel (CLSF-65) | Click node → show properties (no two-way binding) |
| Text-to-graph refresh loop (CLSF-66) | Debounce → fetch → re-render graph |
| Visual authoring (Wave 7, CLSF-76/77) | Drag nodes, draw edges, submit graph JSON |

Waves 6 features are **low interactivity** — debounced fetches, static renders,
click-to-inspect. Wave 7 adds drag-and-drop graph authoring, which is the first
feature that would genuinely benefit from a reactive component model.

### HTMX capability assessment

HTMX handles the Wave 6 interactivity profile without a client-side framework:

- `hx-post` with `hx-trigger="keyup changed delay:500ms"` covers the debounce+validate loop
- `hx-swap` replaces the diagnostics list and graph pane with server-rendered HTML
- Server-Sent Events via `hx-ext="sse"` covers CLSF-69 (SSE proxy validation)
- No build step, no bundler — HTMX is a single `<script>` tag

The Express BFF already renders JSON. Adding a thin HTML rendering layer (EJS or
`express-handlebars`, or plain `res.render`) is within the BFF's existing responsibility
boundary: it is still zero business logic, zero DB access.

## Decision

**Adopt HTMX + Express (BFF) as the web builder stack.**

- The `web/` directory and its React/Vite scaffold are **removed**.
- Web builder HTML is served from the BFF at `/bff/builder/*` routes.
- HTMX (CDN `<script>` tag, no npm dependency) handles partial updates.
- Server-side HTML fragments are rendered by Express route handlers in
  `bff/src/routes/builder.ts`.
- The BFF client contract from ADR-025 is preserved: auth via `/bff/auth/login`,
  API calls proxied through `/bff/api/v1/*`.

Wave 7 visual authoring (CLSF-76/77) requires drag-and-drop graph editing. That
feature will be re-evaluated when it becomes the next priority. If the interactivity
complexity at that point exceeds what HTMX can handle cleanly, a targeted React
component can be embedded **inside** the HTMX page for the graph canvas only —
without requiring a full SPA framework.

## Rationale

| Criterion | HTMX + BFF | React SPA (web/) |
|---|---|---|
| Stack count | 3 (no change) | 4 (+1) |
| Dependency surface | 0 new npm packages | Vite + React + type defs + linter |
| CI gates | 0 new gates | 1 new gate (web typecheck + build) |
| Wave 6 interactivity fit | Full — debounce, fetch, partial render | Full — but overkill |
| Wave 7 fit | Partial — graph drag/drop needs evaluation | Full |
| Dev context switch cost | Low — same BFF codebase | High — separate project |
| Build step | None (HTMX is a script tag) | Vite build required |

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Standalone React SPA in `web/` (initial CLSF-62 choice) | Fourth stack; maintenance cost not justified for Wave 6 interactivity level |
| React embedded in BFF package.json | Reduces stack count but adds bundler complexity to a server process; React SSR or client bundle in Express adds significant config |
| Next.js | SSR framework overkill for an internal tool; adds fifth runtime concern |
| Vue / Svelte | Same stack-count problem as React SPA; no existing precedent in repo |
| Plain HTML + vanilla JS served from Go | Mixes Go server with UI templates; violates Go backend's zero-UI-concern boundary |

## Consequences

**Positive:**
- Repository stays at 3 stacks — no new CI gate, no new `node_modules` to update.
- BFF developers can build builder routes without switching mental models.
- HTMX's `hx-ext="sse"` validates CLSF-69 (SSE proxy browser behavior) as a
  natural side effect of building CLSF-66 (refresh loop).
- Simpler deployment: BFF serves both API proxy and builder UI from one process.

**Negative / tradeoffs:**
- Wave 7 drag-and-drop authoring (CLSF-76/77) needs re-evaluation. If HTMX +
  a lightweight graph library (e.g. D3.js via CDN) is sufficient, no framework
  needed. If not, embed a targeted React component. This decision is deferred
  to an ADR-027 when CLSF-76 is the next task.
- Server-side HTML rendering in Express is less familiar than React components
  for frontend-oriented contributors.
- The `web/` scaffold created in CLSF-62 must be deleted and `.gitignore` entries
  reverted as part of this ADR's implementation.

## Follow-up actions

| Action | Task |
|---|---|
| Delete `web/` directory | CLSF-62 cleanup (immediate) |
| Add `bff/src/routes/builder.ts` + HTMX shell | CLSF-62 re-implementation |
| Re-evaluate Wave 7 graph authoring stack | ADR-027 before CLSF-76 |
| Admin surface at `/bff/admin/*` (HTMX, same pattern) | BFF-ADMIN-01..92 |

## CORS note — admin surface (BFF-ADMIN-03)

The admin web surface (`/bff/admin/*`) is served by the BFF itself on the same
origin as the API routes it proxies. Browser HTMX requests are therefore
**same-origin** and carry no `Origin` header. The existing CORS allowlist in
`bff/src/config.ts` (`parseAllowedOrigins`) requires no new entry for the admin
surface.

If the admin surface is ever served from a separate origin (e.g. a reverse-proxy
path rewrite that changes the effective origin), add that origin to
`BFF_CORS_ALLOWED_ORIGINS` and update this note. Until then, no allowlist change
is needed and none should be added to avoid spurious CORS grants.

## References

- `bff/src/` — Express BFF source (builder routes go here)
- `docs/decisions/ADR-009-bff-thin-proxy.md` — thin proxy constraint
- `docs/decisions/ADR-025-bff-unified-client-gateway.md` — BFF as unified gateway
