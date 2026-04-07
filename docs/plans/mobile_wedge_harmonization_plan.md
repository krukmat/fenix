---
name: Mobile Wedge Harmonization Plan
status: active
owner: mobile
created: 2026-04-06
primary_refs:
  - docs/plans/fenixcrm_strategic_repositioning_spec.md
  - docs/plans/fenixcrm_strategic_repositioning_implementation_plan.md
  - docs/decisions/ADR-022-mobile-deprioritized-for-wedge.md
---

# Mobile Wedge Harmonization Plan

> **Status**: Active
> **Date**: 2026-04-06
> **Audience**: Product, Architecture, Mobile, BFF, API implementation agents
> **Primary references**: `docs/plans/fenixcrm_strategic_repositioning_spec.md`, `docs/plans/fenixcrm_strategic_repositioning_implementation_plan.md`, `docs/decisions/ADR-022-mobile-deprioritized-for-wedge.md`
> **Precedence rule**: this document is the canonical execution plan for mobile, BFF, and the minimum Go API changes required to harmonize the mobile surface with the approved wedge. If older mobile plans conflict with this document, this document takes precedence for `mobile/`, `bff/`, and the supporting mobile-facing API layer.

---

## 1. Purpose

This plan defines how to convert the current mobile experience from a broad CRM-oriented surface into a wedge-aligned product surface for:

1. **Support Copilot and Support Agent**
2. **Sales Copilot through Sales Brief**
3. **Governed execution with approvals, handoff, audit, and usage visibility**

This is not a greenfield redesign.

It is a constrained harmonization plan that:

- preserves the existing support and sales entities that are required by the wedge
- removes non-wedge mobile breadth from the visible product surface
- aligns mobile with the public contracts already exposed or implied by the current backend
- introduces only the minimum BFF and Go API additions necessary to keep mobile implementation clean and deterministic

---

## 2. Governing Decisions

The following decisions are fixed and shall not be re-opened during implementation.

### 2.1 Product Surface Decision

The mobile app shall no longer present itself as a broad CRM and workflow suite.

The mobile app shall present exactly five top-level surfaces:

1. `Inbox`
2. `Support`
3. `Sales`
4. `Activity Log`
5. `Governance`

No other top-level surface shall remain visible.

### 2.2 Scope Decision

This plan includes:

- `mobile/`
- `bff/`
- the minimum Go API additions required to make the approved mobile wedge implementable without client-side guesswork

This plan does not authorize unrelated backend expansion.

### 2.3 Removal Decision

The following visible surfaces shall be removed, not merely de-emphasized:

- `CRM` top-level navigation
- `Workflows` top-level navigation
- top-level `Copilot`
- top-level `Contacts`
- workflow authoring and workflow management screens
- object creation and edit flows that are not required by the support or sales wedge

### 2.4 Compatibility Decision

Compatibility shall be preserved only where the destination surface still exists in the wedge.

Allowed temporary hidden redirects:

- `/home` -> `/inbox`
- `/cases/*` and `/crm/cases/*` -> `/support/cases/*`
- `/accounts/*` and `/crm/accounts/*` -> `/sales/accounts/*`
- `/deals/*` and `/crm/deals/*` -> `/sales/deals/*`

The following route families shall be removed without public replacement:

- `/workflows/*`
- `/contacts/*`
- `/crm/contacts/*`

### 2.5 Support Agent Resolution Decision

The mobile client shall not offer arbitrary agent selection for the main support flow.

When the user triggers the support agent:

1. choose the first active agent definition with `agentType == "support"`
2. if multiple active support agents exist, prefer the one whose name is exactly `Support Agent`
3. if no active support agent exists, show a blocking configuration error and do not fall back to a different agent

---

## 3. Current State and Required Correction

The current mobile app already contains useful wedge-aligned capabilities:

- case, account, and deal detail screens
- contextual copilot entry points
- approvals UI
- signals UI
- handoff banner support
- activity log and run detail
- evidence and audit rendering

However, the current mobile surface still reflects the superseded product shape in four ways:

1. navigation still exposes broad CRM and workflow breadth as first-class product surfaces
2. client types and client copy still partially reflect legacy runtime and approval semantics
3. sales still enters through generic contextual copilot instead of the dedicated `sales-brief` contract
4. usage and quota visibility exist in the backend but are not yet surfaced through a mobile-ready contract

This plan corrects those four gaps without expanding scope.

---

## 4. Target Product Surfaces

### 4.1 Inbox

`Inbox` shall become the default post-login landing screen.

Visible route:

- `/(tabs)/inbox/index`

The inbox shall render a unified feed with four filter chips:

- `All`
- `Approvals`
- `Handoffs`
- `Signals`

The feed ordering shall be deterministic:

1. `approvals`, ordered by `expiresAt` ascending, then `createdAt` ascending
2. `handoffs`, ordered by `createdAt` descending
3. `signals`, ordered by `confidence` descending, then `createdAt` descending

Inbox item behavior:

- approval item -> opens approval detail affordance inline and allows `Approve` or `Reject`
- handoff item -> opens the target entity when entity context exists, otherwise opens activity detail
- signal item -> opens signal detail

Inbox shall not include:

- generic CRM object browsing
- workflow browsing
- top-level copilot chat

### 4.2 Support

`Support` shall be the mobile surface for the primary wedge.

Visible routes:

- `/(tabs)/support/index`
- `/(tabs)/support/cases/[id]`
- `/(tabs)/support/cases/[id]/copilot`

`Support` list behavior:

- list only cases
- default ordering: newest activity first when available, otherwise newest created item first
- no object creation button
- no object edit button

`Support` detail behavior:

- render the case header and current case metadata
- render signals related to the case
- render recent agent activity for the case
- render current handoff state when present
- expose exactly two primary CTAs:
  - `Run Support Agent`
  - `Open Support Copilot`

`Run Support Agent` behavior:

- triggers the resolved support agent with case context
- on success, navigates to the new run detail in `Activity Log`
- if the support agent is missing, shows a blocking configuration error and no trigger request is sent

`Open Support Copilot` behavior:

- opens the contextual copilot screen for the current case
- reuses the existing SSE chat infrastructure
- remains contextual only and is not visible as a top-level route

Support detail shall remain read-only for CRM data entry.

### 4.3 Sales

`Sales` shall be the mobile surface for the secondary wedge.

Visible routes:

- `/(tabs)/sales/index`
- `/(tabs)/sales/accounts/[id]`
- `/(tabs)/sales/deals/[id]`
- `/(tabs)/sales/brief/[entityType]/[id]`

`Sales` index behavior:

- show a segmented control with exactly two segments: `Accounts` and `Deals`
- default selected segment: `Accounts`
- list only the selected entity family

Account and deal detail behavior:

- remain available because they are required context for the sales wedge
- continue to show context sections such as timeline, related deals, related contacts, signals, and agent activity where already implemented
- do not expose generic edit entry points
- expose exactly one primary CTA: `View Sales Brief`

`View Sales Brief` behavior:

- navigates to `/(tabs)/sales/brief/[entityType]/[id]`
- calls the dedicated `sales-brief` contract
- does not fall back to generic summarize or generic chat

Sales Brief screen behavior:

- if the API returns a completed outcome, render:
  - `summary`
  - `risks`
  - `nextBestActions`
  - `confidence`
  - `evidencePack`
- if the API returns an abstention outcome, render:
  - `abstentionReason`
  - `confidence`
  - `evidencePack`
  - no action execution buttons

Generic copilot chat may remain available as a secondary follow-up action from the sales brief or entity detail, but it shall not be the primary sales entry point.

### 4.4 Activity Log

`Activity Log` shall be the operational trace surface.

Visible routes:

- `/(tabs)/activity/index`
- `/(tabs)/activity/[id]`

List behavior:

- show runs grouped or filtered by normalized public outcome, not by raw runtime status
- provide filter chips for:
  - `All`
  - `Completed`
  - `Warnings`
  - `Awaiting Approval`
  - `Handed Off`
  - `Denied`
  - `Abstained`
  - `Failed`

Detail behavior:

- show public `status` as the primary state
- show `runtime_status` only as secondary diagnostics
- show evidence
- show audit events
- show tool calls
- show output
- show per-run usage events fetched by `run_id`

### 4.5 Governance

`Governance` shall be a read-only surface for runtime cost and quota visibility.

Visible route:

- `/(tabs)/governance/index`

Governance behavior:

- render `recentUsage`
- render `quotaStates`
- remain functional if only one of the two sections is available
- show `No active quota policies` when quota policy metadata is absent or empty

Governance shall not become a general admin console.

---

## 5. Interface and Contract Alignment

### 5.1 Agent Run Contract

Mobile shall use two distinct fields:

- `status` for the normalized public outcome
- `runtime_status` for raw runtime diagnostics

The public outcome set shall be exactly:

- `completed`
- `completed_with_warnings`
- `abstained`
- `awaiting_approval`
- `handed_off`
- `denied_by_policy`
- `failed`

The runtime status set shall remain readable but secondary:

- `running`
- `success`
- `partial`
- `abstained`
- `failed`
- `escalated`
- `accepted`
- `rejected`
- `delegated`

Rendering rules:

- all user-facing lists and badges use `status`
- `runtime_status` appears only in run detail diagnostics
- `rejection_reason` is displayed only when `status == denied_by_policy`

### 5.2 Approval Contract

Mobile shall expose the following approval status set:

- `pending`
- `approved`
- `rejected`
- `expired`
- `cancelled`

Mobile shall expose the following decisions only:

- `approve`
- `reject`

The mobile client shall not send or render `deny` or `denied`.

### 5.3 Sales Brief Contract

The mobile client shall treat `POST /api/v1/copilot/sales-brief` as the primary sales intelligence contract.

Required response fields:

- `outcome`
- `entityType`
- `entityId`
- `summary`
- `risks`
- `nextBestActions`
- `confidence`
- `abstentionReason`
- `evidencePack`

The sales brief screen shall be implemented directly against this contract without client-side recomposition from multiple endpoints.

### 5.4 Usage and Quota Contract

The existing usage endpoints are not sufficient for a clean mobile screen because `quota-state` currently requires a `quota_policy_id` and the mobile client has no policy discovery contract.

Therefore the implementation shall add a single summary contract:

- `GET /api/v1/governance/summary`

The response shall contain:

- `recentUsage[]`
- `quotaStates[]`

Each `quotaStates[]` item shall contain both policy metadata and current state:

- `policyId`
- `policyType`
- `metricName`
- `limitValue`
- `resetPeriod`
- `enforcementMode`
- `currentValue`
- `periodStart`
- `periodEnd`
- `lastEventAt`
- `statePresent`

If an active quota policy exists but there is no persisted state row for the current period:

- the handler shall still return a `quotaStates[]` item
- `currentValue` shall be `0`
- `statePresent` shall be `false`
- `periodStart` and `periodEnd` shall be computed for the current policy period

This removes client-side ambiguity and avoids additional policy discovery calls.

---

## 6. Required BFF and Go API Changes

### 6.1 BFF Role

The BFF shall remain primarily a thin proxy.

It may introduce mobile-specific routes only where aggregation or aliasing materially simplifies the mobile client.

### 6.2 Required BFF Routes

The following new BFF routes shall be added:

- `GET /bff/api/v1/mobile/inbox`
- `POST /bff/api/v1/approvals/{id}/approve`
- `POST /bff/api/v1/approvals/{id}/reject`
- `GET /bff/api/v1/governance/summary`

### 6.3 Inbox Aggregation Route

`GET /bff/api/v1/mobile/inbox` shall aggregate:

- pending approvals
- active signals
- handed-off runs

Handoff aggregation rule:

1. list recent agent runs where `status = handed_off`
2. for each run, fetch the handoff package
3. return only successfully enriched handoff items in `handoffs[]`
4. if one handoff enrichment request fails, omit that item and continue assembling the response
5. do not fail the entire inbox response because one handoff item could not be enriched

### 6.4 Approval Alias Routes

The BFF approval alias routes shall translate to the existing backend decision handler.

Mapping:

- `POST /approvals/{id}/approve` -> backend decision `approve`
- `POST /approvals/{id}/reject` -> backend decision `reject`

The alias routes shall not expose legacy `deny`.

### 6.5 Governance Summary Route

The Go API shall add:

- `GET /api/v1/governance/summary`

The BFF shall proxy this route unchanged through:

- `GET /bff/api/v1/governance/summary`

No additional governance aggregation shall be done in mobile.

---

## 7. Mobile Cleanup and Migration Rules

### 7.1 Visible Surface Removal

The following screen families shall be removed from the visible product surface:

- workflow list
- workflow detail
- workflow create
- workflow edit
- CRM hub
- top-level contacts
- broad object create screens
- broad object edit screens

### 7.2 Allowed Contextual Context Retention

Read-only context that supports the wedge may remain inside wedge-aligned detail screens.

Examples allowed to remain:

- related contacts inside account detail
- related deals inside account detail
- timeline sections inside account or deal detail
- signals and agent activity inside case, account, or deal detail

These retained sections shall not reintroduce broad CRM navigation.

### 7.3 Query and Module Refactor

The mobile client shall be reorganized by capability, not by legacy product breadth.

Required capability groups:

- `inbox`
- `support`
- `sales`
- `activity`
- `governance`

Workflow-specific query keys, hooks, screens, and test suites shall be removed from active use.

### 7.4 Legacy Route Rules

Legacy routes shall be handled as follows:

- keep hidden redirects only for home, cases, accounts, and deals because those concepts remain within the wedge
- remove workflows and contacts route families entirely because they have no approved wedge destination

---

## 8. Execution Waves and Task Graph

Implementation shall proceed in ordered waves so the mobile wedge can be delivered without contract churn, navigation regressions, or test instability.

### 8.1 Dependency Rules

The following rules are mandatory:

- do not start visible navigation rewrites before the public mobile-facing contracts are fixed
- do not start the support or sales surface rewrites before the route shell is stable
- do not delete workflow or CRM breadth until the replacement wedge routes are implemented and test-covered
- do not finalize the functional E2E rewrite until the new seed data shape is stable
- do not push any mobile route removal before the required hidden redirects are in place, except for workflows and contacts which have no approved wedge destination

### 8.2 Wave 1 — Contract Lock and Mobile API Enablement

**Objective**: freeze the mobile-facing contracts so later UI work does not need rework.

| ID | Task | Hard dependencies | Unblocks |
|---|---|---|---|
| `W1-T1` | Freeze mobile public types for run outcomes, runtime status, approvals, sales brief, usage, and governance summary | none | all remaining waves |
| `W1-T2` | Add BFF approval alias routes: `approve`, `reject` | `W1-T1` | `W3-T5`, mobile approval hooks, approval tests |
| `W1-T3` | Add BFF inbox aggregation route for approvals, signals, and handoffs | `W1-T1` | `W3-T5`, inbox UI, inbox tests |
| `W1-T4` | Add Go `GET /api/v1/governance/summary` with quota policy metadata plus current state | `W1-T1` | `W1-T5`, `W5-T3`, governance tests |
| `W1-T5` | Add BFF proxy for `GET /bff/api/v1/governance/summary` | `W1-T4` | `W5-T3`, governance client hooks |
| `W1-T6` | Add mobile service and hook layer for inbox, approval aliases, sales brief, activity usage, and governance summary | `W1-T2`, `W1-T3`, `W1-T5` | `W3`, `W4`, `W5` |

Parallelization:

- `W1-T2`, `W1-T3`, and `W1-T4` may run in parallel after `W1-T1`
- `W1-T6` must wait for all client-facing routes it consumes to exist

### 8.3 Wave 2 — Navigation Shell and Route Migration

**Objective**: replace the top-level product shell before migrating wedge screens.

> Wave 2 status: ✅ COMPLETED (W2-T1, W2-T2, W2-T3, W2-T4)

| ID | Task | Hard dependencies | Unblocks | Status |
|---|---|---|---|---|
| `W2-T1` | Replace the drawer and top-level route tree with `Inbox`, `Support`, `Sales`, `Activity Log`, `Governance` | `W1-T1` | all surface rewrites | ✅ done |
| `W2-T2` | Add hidden redirects for `/home`, `/cases/*`, `/accounts/*`, and `/deals/*` into their wedge destinations | `W2-T1` | low-friction migration of existing entry points and tests | ✅ done |
| `W2-T3` | Remove visible `CRM`, `Workflows`, top-level `Copilot`, and top-level `Contacts` navigation | `W2-T1` | `W6-T1`, `W6-T2` | ✅ done |
| `W2-T4` | Remove visible create and edit entry points that are not part of the approved wedge | `W2-T1` | cleaner support and sales detail screens | ✅ done |

Parallelization:

- `W2-T2`, `W2-T3`, and `W2-T4` may run in parallel after `W2-T1`

### 8.4 Wave 3 — Support Wedge Surface

**Objective**: make support the primary mobile wedge.

| ID | Task | Hard dependencies | Unblocks |
|---|---|---|---|
| `W3-T1` | Create the `Support` case list route and move case browsing under it | `W2-T1`, `W2-T2` | `W3-T2`, support E2E |
| `W3-T2` | Create the `Support` case detail route and migrate case detail behavior into the new route family | `W3-T1` | `W3-T3`, `W3-T4`, `W3-T5` |
| `W3-T3` | Implement deterministic support-agent resolution and support-agent trigger flow | `W1-T6`, `W3-T2` | support run E2E, activity validation |
| `W3-T4` | Implement contextual support copilot route under support surfaces | `W3-T2` | support copilot E2E |
| `W3-T5` | Connect inbox approvals, handoffs, and signals into support navigation and support detail refresh paths | `W1-T2`, `W1-T3`, `W3-T2` | inbox E2E, support approval E2E, support handoff E2E |

Parallelization:

- `W3-T3` and `W3-T4` may run in parallel after `W3-T2`
- `W3-T5` may run in parallel with `W3-T3` after `W3-T2` if `W1-T2` and `W1-T3` are complete

### 8.5 Wave 4 — Sales Wedge Surface

**Objective**: make sales a dedicated wedge surface built around `sales-brief`, not generic chat-first entry.

| ID | Task | Hard dependencies | Unblocks |
|---|---|---|---|
| `W4-T1` | Create the `Sales` index with `Accounts` and `Deals` segmented browsing | `W2-T1`, `W2-T2` | `W4-T2`, sales E2E |
| `W4-T2` | Move account and deal detail screens under sales routes and remove edit-first CTAs | `W4-T1` | `W4-T3`, `W4-T4` |
| `W4-T3` | Implement the dedicated sales brief route and render completed plus abstained outcomes directly from the contract | `W1-T6`, `W4-T2` | sales brief E2E, sales abstention E2E |
| `W4-T4` | Keep generic copilot chat only as a secondary follow-up action from the sales route family | `W4-T3` | final sales surface completion |

Parallelization:

- `W4-T1` may run in parallel with Wave 3 once `W2-T1` and `W2-T2` are complete
- `W4-T3` must wait for both the service layer and the new sales route family

### 8.6 Wave 5 — Activity Log and Governance

**Objective**: expose the operational trace and governance surfaces against the normalized contracts.

| ID | Task | Hard dependencies | Unblocks |
|---|---|---|---|
| `W5-T1` | Rework activity list to use normalized public outcomes and wedge-aligned filters | `W1-T1`, `W2-T1` | `W5-T2`, activity E2E |
| `W5-T2` | Rework activity detail to show diagnostics, audit, evidence, tool calls, output, and per-run usage | `W1-T6`, `W5-T1` | run trace validation, support run validation |
| `W5-T3` | Implement governance screen against `governance/summary` | `W1-T5`, `W1-T6`, `W2-T1` | governance E2E |
| `W5-T4` | Route handoff fallback to activity detail when no entity context exists | `W5-T2` | robust handoff behavior across inbox and activity |

Parallelization:

- `W5-T1` and `W5-T3` may run in parallel after `W2-T1` and the Wave 1 service work is complete

### 8.7 Wave 6 — Cleanup, Seeds, Tests, and Documentation

**Objective**: remove superseded breadth only after the wedge surfaces and their tests are stable.

| ID | Task | Hard dependencies | Unblocks |
|---|---|---|---|
| `W6-T1` | Remove workflow screens, workflow hooks, workflow service calls, workflow query keys, and workflow test suites | `W2-T3`, `W3-T5`, `W4-T3`, `W5-T2` | final route cleanup |
| `W6-T2` | Remove CRM hub remnants, top-level contacts remnants, and obsolete legacy routes with no approved destination | `W2-T3`, `W3-T2`, `W4-T2` | final navigation cleanup |
| `W6-T3` | Update mobile seed data and helpers for approvals, handed-off runs, denied-by-policy runs, sales brief success and abstention, usage, and quota state | `W1-T4`, `W3-T5`, `W4-T3`, `W5-T3` | `W6-T4` |
| `W6-T4` | Rewrite functional E2E suites (Detox) to the new wedge-first route model. Note: Maestro visual-audit.yaml update is a separate post-harmonization task (see runner note below) | `W3-T5`, `W4-T3`, `W5-T3`, `W6-T3` | final QA sign-off |
| `W6-T5` | Replace or trim unit, hook, and BFF tests for removed surfaces and finalize green coverage | `W6-T1`, `W6-T2`, `W6-T4` | final QA sign-off |
| `W6-T6` | Update `mobile/README.md` and any remaining planning references, then run all required QA gates | `W6-T4`, `W6-T5` | completion |

### 8.8 Critical Path

The critical path has two co-critical branches that converge at `W6-T3`:

**Branch A (Support)**:

`W1-T1` -> `W2-T1` -> `W2-T2` -> `W3-T1` -> `W3-T2` -> `W3-T5` -> `W6-T3` -> `W6-T4` -> `W6-T5` -> `W6-T6`

**Branch B (Sales)**:

`W1-T1` -> `W2-T1` -> `W2-T2` -> `W4-T1` -> `W4-T2` -> `W4-T3` -> `W6-T3` -> `W6-T4` -> `W6-T5` -> `W6-T6`

Both branches share the same depth (10 steps). Either can delay the plan if it slips.

**Parallel feeder chain (Wave 1 API enablement)**:

`W1-T1` -> `W1-T4` -> `W1-T5` -> `W1-T6`

This chain feeds `W3-T3`, `W4-T3`, `W5-T2`, and `W5-T3` but is shorter than Branches A and B — it has float as long as it completes before the tasks that consume `W1-T6`.

`W1-T2` and `W1-T3` run in parallel with `W1-T4` (all depend only on `W1-T1`). They feed `W3-T5` directly but `W3-T5` also waits for `W3-T2` which is deeper, so they do not extend the critical path.

Interpretation:

- support (`W3-*`) and sales (`W4-*`) surface work proceed in parallel after `W2-T2`, but both must complete before `W6-T3` (seed data update)
- `W5-T1` (activity list rework) depends only on `W1-T1` + `W2-T1` and can start as early as Wave 2; `W5-T3` (governance) depends on `W1-T5` + `W1-T6` + `W2-T1` and can run in parallel with Waves 3-4 — neither is on the critical path
- `W6-T1` and `W6-T2` (cleanup) depend on Wave 3 and Wave 4 tasks respectively and must complete before `W6-T5`, but they run in parallel with `W6-T3` → `W6-T4` and do not extend the critical path unless they slip past `W6-T4`
- cleanup must never run ahead of replacement route coverage

Runner note:

- functional mobile E2E remains on the current Detox runner unless a separate canonical plan replaces it
- Maestro remains the approved runner for screenshot and visual audit flows, as specified in `docs/plans/maestro-screenshot-migration.md`

Post-harmonization coordination (Maestro screenshot suite):

- after Wave 6 is complete, the current `mobile/maestro/visual-audit.yaml` will be broken because it captures screens that are removed (CRM hub, Workflows, Contacts top-level, create/edit forms) and does not capture new screens (Inbox, Sales Brief, Governance)
- a follow-up task must update the Maestro visual audit flow to match the wedge-first navigation model:
  1. remove screenshots for deleted surfaces: CRM hub (05), Contacts list/detail (09-10), Workflow list/new/detail/edit (20-23), Copilot panel (19), Account new (08), Deal new/edit (13-14), Case new/edit (17-18)
  2. add screenshots for new surfaces: Inbox (with filter chips), Support case list, Support case detail (with agent/copilot CTAs), Sales index (segmented), Sales Brief (completed and abstained), Activity Log (with normalized filters), Governance (usage and quotas), Drawer with five wedge tabs
  3. update `mobile/maestro/seed-and-run.sh` to provide seed data for the new surfaces (approvals, handoffs, sales brief success/abstention, usage, quota state)
- this follow-up is explicitly scoped as a post-harmonization task and must not block Wave 6 delivery

### 8.9 Wave Summary

The waves are:

1. `Wave 1` — Contract Lock and Mobile API Enablement
2. `Wave 2` — Navigation Shell and Route Migration
3. `Wave 3` — Support Wedge Surface
4. `Wave 4` — Sales Wedge Surface
5. `Wave 5` — Activity Log and Governance
6. `Wave 6` — Cleanup, Seeds, Tests, and Documentation

---

## 9. Detailed Test Plan

### 9.1 Mobile Unit and UI Tests

Add or update tests for:

- drawer renders exactly five top-level items
- no visible `CRM`, `Workflows`, top-level `Copilot`, or top-level `Contacts`
- inbox filter chips, ordering, loading, empty, and error states
- support case list and support detail rendering
- support agent trigger success and missing-agent failure state
- support handoff banner routing
- sales segmented control
- sales brief completed state
- sales brief abstention state
- activity filters using normalized public outcomes
- activity detail usage section
- governance summary screen partial and complete states

Delete or replace tests for:

- workflow list
- workflow detail
- workflow create
- workflow edit
- CRM hub

### 9.2 Mobile Service and Hook Tests

Add or update tests for:

- inbox aggregation client calls
- approval alias routes
- sales brief route
- governance summary route
- normalized run outcome filtering
- query invalidation after approval decisions

Remove tests that assert workflow query keys or workflow API behavior inside the active mobile product path.

### 9.3 BFF Tests

Add BFF tests for:

- `mobile/inbox` returns approvals, signals, and enriched handoffs
- partial handoff enrichment failure does not fail the full inbox response
- approval alias routes translate to the correct backend decision
- governance summary route proxies correctly
- sales brief route continues to proxy correctly

### 9.4 Functional E2E Tests

Functional E2E shall continue to use the current Detox runner for this plan.

Maestro is out of scope for functional flow automation here and remains limited to screenshot and visual audit usage.

Replace the current E2E entry assumptions so login lands on `Inbox`.

Required E2E coverage:

1. `auth-inbox.e2e.ts`
   - login lands on inbox
   - inbox is visible

2. `support-flow.e2e.ts`
   - open support list
   - open seeded case
   - trigger support agent
   - navigate to resulting activity detail

3. `support-approval.e2e.ts`
   - pending approval appears in inbox
   - approve or reject action succeeds
   - refreshed status disappears from pending inbox section

4. `support-handoff.e2e.ts`
   - handed-off item appears in inbox
   - accept handoff navigates to the target case
   - evidence count and rationale remain visible

5. `sales-brief.e2e.ts`
   - open account or deal from sales surface
   - open sales brief
   - summary, risks, actions, confidence, and evidence pack are visible

6. `sales-brief-abstention.e2e.ts`
   - abstention reason is visible
   - no executable action buttons are rendered

7. `activity-log.e2e.ts`
   - filter by normalized public outcome
   - open run detail
   - audit and usage sections are visible

8. `governance.e2e.ts`
   - governance screen renders recent usage
   - governance screen renders quota states

Delete:

- `workflows.e2e.ts`

Rewrite existing account, deal, and case E2Es so they use the new wedge-first navigation model.

### 9.5 Seed Data Requirements

The mobile seed helper and its Go data source shall provide:

- one pending approval
- one handed-off run
- one denied-by-policy run
- one seeded support case with valid support-agent context
- one successful sales brief entity
- one abstaining sales brief entity
- recent usage events
- at least one active quota policy with current-period state

---

## 10. QA Gates and Exit Criteria

### 10.1 Required Local QA Gates

Before any push for this work, run:

```bash
cd bff && npm run test
bash scripts/check-no-inline-eslint-disable.sh
cd mobile && npm run typecheck
cd mobile && npm run lint
cd mobile && npm run quality:arch
cd mobile && npm run test:coverage
cd mobile && npm run e2e:test
```

### 10.2 Exit Criteria

This harmonization is complete only when all of the following are true:

- the visible mobile product surface contains exactly `Inbox`, `Support`, `Sales`, `Activity Log`, and `Governance`
- the support flow is demonstrable end-to-end on mobile
- the sales brief flow is demonstrable end-to-end on mobile
- approvals use only `approve` and `reject`
- handed-off runs are visible and actionable from inbox and activity
- activity detail shows audit and run-linked usage
- governance shows recent usage and quota state without extra client-side joins
- no remaining visible screen implies broad CRM parity or workflow parity

---

## 11. Non-Goals

The following work is explicitly out of scope for this plan:

- new mobile-first workflows
- plugin or marketplace work
- broad CRM expansion beyond cases, accounts, and deals already required by the wedge
- workflow authoring replacement
- general admin console behavior in governance
- backend feature work unrelated to mobile wedge harmonization

---

## 12. Default Assumptions

If implementation agents find missing details, the following defaults shall be used unless a newer canonical document overrides them:

- support remains the primary wedge
- sales remains the secondary wedge
- BFF stays thin unless aggregation or aliasing removes material mobile ambiguity
- Go remains the source of truth for normalized runtime outcomes, approval state semantics, sales brief output, audit behavior, and usage events
- hidden redirects are temporary compatibility mechanisms, not permanent product surfaces
