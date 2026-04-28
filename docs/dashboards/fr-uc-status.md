---
title: FenixCRM — FR & UC Implementation Status
last_updated: 2026-04-27
tags: [dashboard, status]
---

# FenixCRM — FR & UC Implementation Status

> Last updated: 2026-04-27
> Source of truth for current wedge priority: `docs/architecture.md` + `docs/plans/fenixcrm_strategic_repositioning_spec.md`
> Source of truth for detailed requirement inventory: `docs/requirements.md` + BDD feature files in `features/`
>
> Strategic note:
> This dashboard is primarily an implementation-breadth view. It includes legacy broad-platform coverage that no longer defines the commercial wedge by itself.
> Current priority order is Support Copilot / Support Agent first, Sales Copilot second, with mobile breadth and broad Agent Studio surfaces no longer treated as universal P0 release gates.
> BDD audit for the current runner model: `docs/bdd_post_repositioning_audit.md`
>
> Commercial package model:
> `Support Copilot`, `Support Agent`, `Sales Copilot`

---

## 1. Functional Requirements (FR) — Implementation Status

Legend: ✅ Implemented | ⏳ Partial | ❌ Not implemented | 🔒 P1/P2 (deferred)

### CRM Core

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-001 | Core CRM entities (Account, Contact, Lead, Deal, Case) | P0 | ✅ | task_1.3, task_1.4, task_1.5 | @FR-001 in uc-s1, uc-c1 |
| FR-002 | Activity, Note, Attachment | P0 | ✅ | task_1.5 | — |
| FR-003 | Pipeline stages | P0 | ✅ | task_1.5 | — |
| FR-004 | Timeline / audit per record | P0 | ✅ | task_1.7 | — |
| FR-052 | Plugin marketplace | P2 | 🔒 | — | — |

### Knowledge & RAG

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-090 | Hybrid retrieval (BM25 + vector) | P0 | ✅ | task_2.5 | @FR-090 in uc-d1, uc-s1, uc-c1 |
| FR-091 | Multi-source ingestion (email, docs, calls) | P0/P1 | ⏳ | task_2.2 (base), W2-T4 boundary | @FR-091 in uc-d1 |
| FR-092 | Evidence pack (mandatory, grounded) | P0 | ✅ | task_2.6 | @FR-092 in uc-s1, uc-c1, uc-s2, uc-k1, uc-d1 |
| FR-093 | Incremental reindex via CDC (<60s SLA) | P0 | ✅ | task_2.7 | — |
| FR-094 | Knowledge item CRUD + workspace isolation | P0 | ✅ | task_2.1, task_2.2 | — |

### Copilot

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-200 | Copilot in-flow UI (SSE streaming) | P0 | ✅ | task_3.4, task_4.4 | @FR-200 in uc-s1 |
| FR-201 | Copilot context (retrieval + evidence) | P0 | ✅ | task_3.4 | @FR-200 in uc-s1 |
| FR-202 | Copilot actions (tool execution from UI) | P0 | ✅ | task_3.3, task_4.4 | @FR-202 in uc-s1, uc-a4 |
| FR-212 | Behavior contracts (Carta spec) | P1 | 🔒 | — | — |

### AI Behavior

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-210 | Mandatory abstention (insufficient evidence) | P0 | ✅ | task_3.4 | @FR-210 in uc-c1 (abstention scenario) |
| FR-211 | Safe tool routing (no dangerous tools) | P0 | ✅ | task_3.1, task_3.3 | @FR-211 in uc-b1 |
| FR-213 | Model-agnostic LLM adapter | P1 | ✅ | task_2.3 | — |

### Agent Runtime

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-230 | Agent definition + execution | P0 | ✅ | task_3.5 | multiple UCs |
| FR-231 | Agent catalog (prospecting, KB, insights) | P1 | ✅ | task_4.5b, task_4.5c, task_4.5d | uc-s2, uc-k1, uc-d1 |
| FR-232 | Human handoff with evidence | P0 | ✅ | task_3.7 | uc-c1 (handoff scenario) |
| FR-233 | Budget controls / quotas per agent | P1 | 🔒 | — | — |
| FR-243 | Replay / simulation | P1 | 🔒 | — | — |

### Agent Studio

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-240 | Prompt / policy versioning | P0 | ✅ | task_3.9 | uc-a2, uc-a3, uc-a8 |
| FR-241 | Workflow authoring (DSL) | P1 | ✅ | task_4.5a | uc-a2 |
| FR-242 | Skills builder | P1 | ✅ | task_4.5a | uc-a4 |
| FR-243 | Eval suite + gating | P1 | ✅ | task_4.7 | — |

### Integrations

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-050 | Email connector | P0 | ⏳ | task_2.2 (stub) | — |
| FR-051 | Document connector | P1 | 🔒 | — | — |
| FR-052 | Plugin SDK | P2 | 🔒 | — | — |

### Security & Audit

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-060 | RBAC / ABAC + policy engine | P0 | ✅ | task_1.6, task_3.1, BFF-ADMIN-50/51 (governance + policy sets web surface) | uc-g1 |
| FR-070 | Immutable audit trail | P0 | ✅ | task_1.7, task_4.6, BFF-ADMIN-40/41 (audit trail web surface) | uc-g1, uc-a7 |
| FR-071 | Approval chains | P0 | ✅ | task_3.2, BFF-ADMIN-30/31 (approvals queue + decision form web surface) | uc-c1 (approval scenario), uc-a7 |
| FR-072 | PII redaction (no-cloud policy) | P0 | ✅ | task_3.1 | — |

### Mobile & BFF

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-300 | Mobile app (React Native + Expo) | P0 | ✅ | task_4.2, task_4.3, task_4.4, task_4.5, ui-redesign-command-center | uc-s1 (mobile, partial); visual shell validated by Maestro screenshots |
| FR-301 | BFF Gateway (Express.js thin proxy) | P0 | ✅ | task_4.1 | — |
| FR-302 | Observability (/metrics, /health) | P0 | ✅ | task_4.9 | — |
| FR-303 | Mobile eval service | P1 | 🔒 | — | — |
| FR-304 | CRM List Centralized CRUD and Bulk Delete | P1 | ✅ | crm_list_centralized_crud_bulk_delete | uc-p2-crm-list-crud-mobile.feature (TST-067–076) |

---

## 2. Use Case Coverage Matrix

| UC | Title | Feature File | Go BDD | BFF BDD | Mobile BDD | Doorstop | Gaps |
|----|-------|-------------|--------|---------|------------|----------|------|
| UC-S1 | Sales Copilot | `uc-s1-sales-copilot.feature` + `uc-s1-sales-copilot-mobile-smoke.feature` | ✅ canonical backend | ⏳ | ⏳ smoke | ✅ UC_S1.yml | Mobile remains smoke-only, not canonical |
| UC-S2 | Prospecting Agent | `uc-s2-prospecting-agent.feature` + `uc-s2-prospecting-agent-mobile.feature` | ✅ | ⏳ | ⏳ defined (runner blocked) | ✅ UC_S2.yml | Mobile trigger implemented; mobile feature coverage added, runner still blocked |
| UC-S3 | Deal Risk Agent | `uc-s3-deal-risk-agent.feature` + `uc-s3-deal-risk-agent-mobile.feature` | ✅ canonical backend | ⏳ | ⏳ active trigger flow defined | ✅ UC_S3.yml | Backend runner and mobile trigger are active; Maestro flow now targets a stale seeded deal |
| UC-C1 | Support Agent | `uc-c1-support-agent.feature` | ✅ canonical backend | ⏳ | ❌ | ✅ UC_C1.yml | BFF admin shell (approvals, audit, agent runs, governance) provides web operator surface for governance flows in this UC |
| UC-K1 | KB Agent | `uc-k1-kb-agent.feature` + `uc-k1-kb-agent-mobile.feature` | ✅ | ⏳ | ⏳ defined (runner blocked) | ✅ UC_K1.yml | Mobile KB trigger implemented; mobile feature coverage added, runner still blocked |
| UC-D1 | Data Insights Agent | `uc-d1-data-insights-agent.feature` + `uc-d1-data-insights-agent-mobile.feature` | ✅ | ⏳ | ⏳ defined (runner blocked) | ✅ UC_D1.yml | Mobile Insights screen implemented; mobile feature coverage added, runner still blocked |
| UC-G1 | Governance | `uc-g1-governance.feature` | ✅ canonical backend | ⏳ | ❌ | ✅ UC_G1.yml | Replay/rollback scenarios deferred |
| UC-A1 | Agent Studio | `uc-a1-agent-studio.feature` | ⏳ baseline | ⏳ | ❌ | ✅ UC_A1.yml | Baseline/stub coverage only |
| UC-A2 | Workflow Authoring | `uc-a2-workflow-authoring.feature` | ✅ | ⏳ | ❌ | ✅ UC_A2.yml | — |
| UC-A3 | Workflow Verification | `uc-a3-workflow-verification-and-activation.feature` | ✅ | ⏳ | ❌ | ✅ UC_A3.yml | — |
| UC-A4 | Workflow Execution | `uc-a4-workflow-execution.feature` | ✅ | ⏳ | ❌ | ✅ UC_A4.yml | — (4 scenarios: happy, condition_false, tool_failure, approval) |
| UC-A5 | Signal Detection | `uc-a5-signal-detection-and-lifecycle.feature` | ✅ | ⏳ | ❌ | ✅ UC_A5.yml | — |
| UC-A6 | Deferred Actions | `uc-a6-deferred-actions.feature` | ✅ | ⏳ | ❌ | ✅ UC_A6.yml | — (3 scenarios: happy, archived, failure) |
| UC-A7 | Human Override & Approval | `uc-a7-human-override-and-approval.feature` | ✅ | ⏳ | ❌ | ✅ UC_A7.yml | — |
| UC-A8 | Workflow Versioning | `uc-a8-workflow-versioning-and-rollback.feature` | ✅ | ⏳ | ❌ | ✅ UC_A8.yml | — |
| UC-A9 | Agent Delegation | `uc-a9-agent-delegation.feature` | ✅ | ⏳ | ❌ | ✅ UC_A9.yml | — |
| UC-B1 | Safe Tool Routing | `uc-b1-safe-tool-routing.feature` | ✅ | ⏳ | ❌ | ✅ UC_B1.yml | — |
| UC-P1 | CRM Contacts Mobile | `uc-p1-crm-contacts-mobile.feature` | ❌ | ❌ | ✅ mobile canonical | — | Mobile-only UC — contacts list + detail screens (Task Mobile P1.4) |
| UC-P2 | CRM List CRUD and Bulk Delete | `uc-p2-crm-list-crud-mobile.feature` | ❌ | ❌ | ✅ mobile canonical | ✅ UC_P2.yml | Multi-select, row edit, bulk delete, read-only detail (FR-304) |

**Totals:** 19 UCs | Go canonical: 3 wedge UCs + runtime UCs | BFF: 0/19 ⏳ | Mobile: smoke UC-S1 + canonical UC-P1, UC-P2

---

## 3. UC Gap Closure — Pending Tasks

Source: `docs/tasks/task_uc_gap_closure.md`

### P0 Gaps (must close before release)

| Task | UC | Gap | Status |
|------|-----|-----|--------|
| Gap-1 | UC-S1 | Extend `suggest_actions.go` for Account/Deal (BE only responds to Case today) | ✅ done |
| Gap-2 | UC-S1, UC-S3 | Add `get_deal` and `update_deal` to tool registry | ✅ done |
| Gap-3 | UC-S1 | Copilot panel section on Account detail + Deal detail screens (Mobile) | ❌ pending |
| Gap-4 | UC-G1 | Audit Log screen in mobile app + `/audit/events` API methods | ✅ done |

### P1 Gaps (next release)

| Task | UC | Gap | Status |
|------|-----|-----|--------|
| Gap-5 | UC-S3 | Implement `deal_risk.go` agent | ✅ done |
| Gap-6 | UC-S3 | Wire `deal_risk` runner adapter + registry + handler endpoint | ✅ done |
| Gap-7 | UC-S3, UC-S2 | Agent trigger buttons in Deal detail + Leads screen (Mobile) | ✅ done for UC-S3; UC-S2 unchanged |
| Gap-8 | UC-K1, UC-D1 | KB search trigger + Data Insights screen (Mobile) | ✅ done |

---

## 4. BDD Pipeline Status

| Gate | CI Status | Command |
|------|-----------|---------|
| Doorstop integrity | ✅ Passing | `make doorstop-check` |
| UC→FR→TST traceability | ✅ Passing | `make bdd-trace-check` |
| Go BDD runner (`@stack-go and not @deferred`) | ✅ Passing | `make test-bdd-go` |
| BFF BDD runner | ⏳ Not implemented | `make test-bdd-bff` (reserved) |
| Mobile BDD runner | ❌ Blocked (Android SDK) | `make test-bdd-mobile` |

---

## 5. ADR Index

All architectural decisions are in `docs/decisions/`:

| ADR | Title | Area |
|-----|-------|------|
| ADR-001 | UUID full slug (no truncation) | Database / Testing |
| ADR-002 | sqlite-vec multi-tenant JOIN | Security / Vector Search |
| ADR-003 | TestMain + os.Setenv for JWT_SECRET | Testing / Auth |
| ADR-004 | t.Parallel + t.Setenv incompatibility | Testing / Go |
| ADR-005 | goconst + shadow in handlers | Go / Lint |
| ADR-006 | gocyclo complexity gate ≤7 | Quality / Go |
| ADR-007 | pkgauth alias for domain/auth | Go / Packages |
| ADR-008 | Route structure — public vs protected | Auth / Routing |
| ADR-009 | BFF as thin proxy | Architecture |
| ADR-010 | modernc.org/sqlite CGO_ENABLED=0 | Deploy / SQLite |
| ADR-011 | FTS5 sync via triggers | SQLite / Search |
| ADR-012 | Chunking strategy 512/50 | RAG / Knowledge |
| ADR-013 | Embedding async pipeline | RAG / Event Bus |
| ADR-014 | Hybrid search RRF k=60 | RAG / Search |
| ADR-015 | Evidence deduplication cosine >0.95 | RAG / Evidence |
| ADR-016 | CDC reindex SLA <60s | CDC / Search |
| ADR-017 | Quality gates gocognit/maintidx | Quality / Lint |
| ADR-018 | BDD pipeline strategy | Testing / BDD |
| ADR-019 | Product category: governed AI layer | Strategy |
| ADR-020 | Cost governance as runtime concern | Governance / Cost |
| ADR-021 | Integration-first context strategy | RAG / Context |
| ADR-022 | Mobile deprioritized for wedge | Mobile / Strategy |
| ADR-023 | APPROVE role validation — deferred to runtime, workspace-scoped, abstention on unknown | Security / DSL / Governance |
| ADR-024 | Defer TYPE, ENUM, ACTION, CONNECTOR — no implementation until runtime contracts exist | DSL / Language Design |
| ADR-025 | BFF as unified client gateway — web, mobile, and future clients all route through BFF | Architecture / Security |
| ADR-026 | Web builder stack: HTMX + Express (BFF) over separate React SPA — avoids 4th stack | Architecture / Web / Maintenance |
| ADR-027 | DESIGN.md visual contract — agent reads design token file before any mobile UI change | Mobile / Design / Agent |
| ADR-028 | Dual approval seed for snapshot runner approve/reject coverage — re-evaluate on runner parallelism or FSM reset | Testing / Snapshots / BFF |
| ADR-029 | BFF admin shell: HTMX read-only surface at `/bff/admin/*` over existing Go governance endpoints — no 4th stack, read-only constraint, governance-for-policy substitute | Architecture / Web / Admin |

---

## 6. BFF Admin Shell — Closeout Record (BFF-ADMIN-GAP handoff)

> Completed 2026-04-27. See `docs/plans/bff-admin-surface-gap-closure.md` for full task graph.

### Phase completion summary

| Phase | Tasks | Status |
|-------|-------|--------|
| A — Foundation | BFF-ADMIN-01, BFF-ADMIN-02, BFF-ADMIN-03 | ✅ Done |
| B — Workflows | BFF-ADMIN-10, BFF-ADMIN-11, BFF-ADMIN-12 | ✅ Done |
| C — Agent runs | BFF-ADMIN-20, BFF-ADMIN-21a–21d | ✅ Done |
| D — Approvals | BFF-ADMIN-30, BFF-ADMIN-31 | ✅ Done |
| E — Audit trail | BFF-ADMIN-40, BFF-ADMIN-41 | ✅ Done |
| F — Governance / Policy | BFF-ADMIN-50, BFF-ADMIN-51 | ✅ Done |
| G — Tools | BFF-ADMIN-60 | ✅ Done |
| H — Metrics | BFF-ADMIN-70 | ✅ Done |
| I — Closeout | BFF-ADMIN-90, BFF-ADMIN-91, BFF-ADMIN-92 | ✅ Done |
| J — Mobile regression | BFF-ADMIN-J4 | ✅ Done (2026-04-27) |

### BFF-ADMIN-J4 — Mobile screenshot regression sanity

**Status: COMPLETED 2026-04-27. Exit 0. 36/36 screenshots captured on Pixel_7_API_33.**

`bash mobile/maestro/seed-and-run.sh` passed both phases:

- **Phase 1** (`auth-surface.yaml`): `01_auth_login.png` ✅
- **Phase 2** (`authenticated-audit.yaml`): 35 screenshots ✅ — all COMPLETED

**Fix applied during run**: Android ANR dialog ("Process system isn't responding")
appeared on cold-start and at governance route transitions under emulator load.
Added `repeat: times: 8 / runFlow: when: visible / tapOn: Wait` dismiss loops in
`auth-surface.yaml` (cold start) and `authenticated-audit.yaml` (`governance/audit`
and `governance/usage` steps). Commit: `a09dd9e`.

**Seeder unchanged**: `scripts/e2e_seed_mobile_p2.go` — 0 lines diff from BFF admin
commits. No cross-track contamination confirmed.

**Screenshots archived**: `mobile/artifacts/screenshots/` (36 files, `01_auth_login`
through `32_crm_contacts_after_bulk_delete`).

### FR coverage added by admin shell

| FR | New web surface |
|----|----------------|
| FR-060 | Governance page (`/bff/admin/policy`) — quota states + policy sets |
| FR-070 | Audit trail page (`/bff/admin/audit`) — paginated list + record detail |
| FR-071 | Approvals queue + decision form (`/bff/admin/approvals`) |
| FR-200 | Agent runs review (`/bff/admin/agent-runs`) — list, detail, trace, evidence, tool calls, cost |
| FR-230 | Workflow administration (`/bff/admin/workflows`) — list, detail, activation |
| FR-233 | Governance quota monitoring (`/bff/admin/policy` → governance summary) |

### BFF-ADMIN-Task6 — Puppeteer admin screenshot suite

**Status: COMPLETED 2026-04-28. Exit 0. 13/13 screenshots captured.**

`cd bff && npm run admin-screenshots` passed all 7 phases:

- 13 PNGs generated in `bff/artifacts/admin-screenshots/`
- `report.html` (CSS image grid) + `index.md` (Markdown table) generated
- Fixes applied: `adminWorkflows` envelope crash, `adminAudit` wrong Go routes + field names, seeder NULL scan, DB path

Commit: `f6f303a`
Task doc: `docs/tasks/task_bff_admin_screenshots.md`

### Test suite state at closeout

| Suite | Tests | Status |
|-------|-------|--------|
| Full BFF Jest suite | 375 | ✅ Passing |
| Admin navigation smoke (`admin.e2e.test.ts`) | 16 | ✅ Passing |
| Per-route admin unit tests | ~200 (across admin*.test.ts) | ✅ Passing |
| Puppeteer admin screenshot suite | 12 routes | ✅ Passing (exit 0) |
