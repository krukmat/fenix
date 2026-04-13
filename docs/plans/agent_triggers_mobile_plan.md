---
title: "Agent Triggers — Mobile Implementation Plan"
doc_type: task
status: review
phase: post-mvp
tags: [gap-7, gap-8, mobile, agents, UC-S2, UC-S3, UC-K1, UC-D1]
fr_refs: [FR-230, FR-231, FR-300]
uc_refs: [UC-S2, UC-S3, UC-K1, UC-D1]
created: 2026-04-13
last_updated: 2026-04-13
reviewed: 2026-04-13
---

# Agent Triggers — Mobile Implementation Plan

## Context

FenixCRM has 4 operational agents in the Go backend (Support, Prospecting, KB, Insights) with stable API contracts, BDD coverage, and BFF proxy pass-through. Only the Support Agent has a mobile trigger button today. The remaining 3 agents are fully functional but invisible to mobile users — implemented backend capability that delivers zero value because there is no entry point.

This plan closes **Gap-7** (UC-S2 Prospecting, UC-S3 Deal Risk mobile triggers) and **Gap-8** (UC-K1 KB, UC-D1 Insights mobile triggers) from `docs/tasks/task_uc_gap_closure.md`. It moves 3 UCs from red to green in the mobile column of the coverage matrix.

Deal Risk Agent (UC-S3) backend is not yet implemented (Gap-5+6 pending). The plan includes a disabled placeholder button to be activated when the backend ships.

**Critical execution notes discovered during plan review:**
- The queued trigger endpoints for Prospecting / KB / Insights return `{ run_id, status, agent }` in snake_case. This is not the same contract as `triggerSupportRun`, so Wave 1 must normalize those responses before UI code depends on them for navigation.
- Sales already uses an inline segmented screen (`accounts | deals`) in `sales/index.tsx`. Leads should follow that same pattern as a third inline segment; the plan must not duplicate the leads list in both `sales/index.tsx` and `sales/leads/index.tsx`.
- `AgentActivitySection` currently only accepts `account | deal | case`. If lead detail reuses it, Wave 3 must widen that prop contract to include `lead`.
- `mobile/src/services/api.ts` is already close to the 300-line architecture gate. Wave 1 may add lead methods there only if the limit remains green; otherwise the same wave must split CRM methods into a dedicated extracted file and re-export them.

**BDD / vault contrast notes:**
- `features/uc-s2-prospecting-agent.feature` covers grounded prospect research and outreach drafting. The mobile task therefore should expose a generic **Prospecting Agent** trigger and route to run detail, not hard-code a UI promise for a specific final artifact such as "draft email only".
- `features/uc-k1-kb-agent.feature` covers knowledge draft generation from a resolved support outcome with grounded evidence. Mobile can gate by `case.status === 'resolved'`, but the grounded-evidence precondition remains a backend responsibility and should not be duplicated as brittle client logic.
- `features/uc-d1-data-insights-agent.feature` explicitly includes both successful analytical answers and safe rejection of unsupported conclusions. The mobile Insights screen must therefore navigate to the run detail for **all** outcomes, not assume a success-only happy path.
- `features/uc-s3-deal-risk-agent.feature` remains tagged `@deferred`, which confirms the placeholder approach for Deal Risk is the correct integration point until the backend runner ships.
- Mobile feature coverage now exists for `UC-S2`, `UC-K1`, `UC-D1`, and the deferred `UC-S3` placeholder. The remaining BDD gap is runner execution in CI/local Android, not absence of mobile feature files.

---

## Progress Checkpoint

**2026-04-13**

- Wave 1 completed in code:
  - `agentApi` now exports typed `triggerProspectingRun`, `triggerKBRun`, and `triggerInsightsRun`
  - queued trigger responses are normalized to `{ runId, status, agent }`
  - `useWedge` now exposes specialized mutation hooks for all three triggers
  - `useCRM` now exposes `queryKeys.leads`, `queryKeys.lead`, `useLeads()`, and `useLead(id)`
- Wave 2 completed in code:
  - support case detail now renders a conditional `Generate KB Article` trigger when `case.status === 'resolved'`
  - legacy case detail mirrors the same KB trigger behavior for parity while that route still exists
- Wave 3 completed in code:
  - Sales now includes a third inline `Leads` tab backed by `useLeads()`
  - lead detail now exists at `sales/leads/[id]` with a Prospecting Agent trigger that routes to activity run detail
  - `AgentActivitySection` now accepts `lead`, and deal detail includes the disabled Deal Risk placeholder
- Wave 4 completed in code:
  - Activity now exposes an `Insights Agent` entry point that routes to a dedicated insights form screen
  - the new insights screen submits `query`, `date_from`, and `date_to` through `useTriggerInsightsAgent()`
  - date inputs are normalized to RFC3339 before the trigger request and successful submissions route to activity run detail
- Wave 5 completed in code:
  - integration coverage now exercises KB visibility, prospecting navigation, insights RFC3339 serialization, and shared pending-state behavior
  - `npm run screenshots` now seeds `lead` + `resolvedCase` fixtures and captures the new mobile surfaces through Maestro
  - dashboard/vault status now reflects Gap-7 as partial and Gap-8 as done
- Wave 1 verification completed:
  - focused Jest coverage for API normalization, trigger hooks, and CRM lead hooks
  - `mobile` `typecheck`, `lint`, and `quality:arch` passing locally
- Wave 2 verification completed:
  - dedicated KB trigger tests added for support detail and legacy case detail render coverage
  - full `bash scripts/qa-mobile-prepush.sh` passing after the UI integration
- Wave 3 verification completed:
  - dedicated Sales tests added for the Leads tab, Prospecting trigger, lead activity rendering, and Deal Risk placeholder
  - full `bash scripts/qa-mobile-prepush.sh` passing after the Sales navigation + lead detail integration
- Wave 4 verification completed:
  - dedicated Activity tests added for the insights entry point, form validation, RFC3339 serialization, and success navigation
  - full `bash scripts/qa-mobile-prepush.sh` passing after the Activity + Insights integration
- Wave 5 verification completed:
  - `mobile/__tests__/integration/agent-triggers.test.tsx` passing
  - `go test ./scripts/...` passing after seeder expansion
  - `npm run screenshots` passing with new captures: KB trigger, lead prospecting, deal-risk placeholder, and insights entry screen

**2026-04-13 — Post-implementation review (orchestrator audit)**

Independent verification of coder agent deliverables. Full QA gate + regression suite executed.

- **Test results**: 60/60 suites passing, 451/451 tests green, 0 regressions
- **QA gate**: `bash scripts/qa-mobile-prepush.sh` — PASSED (typecheck, lint, arch, coverage)
- **Architecture gate**: `api.ts` at 297 lines (under 300-line limit)
- **Dashboard**: `docs/dashboards/fr-uc-status.md` correctly updated — Gap-7 partial, Gap-8 done, UC-S2/UC-K1/UC-D1 mobile columns reflect implemented state

Wave-by-wave code audit results:

| Wave | Status | Notes |
|------|--------|-------|
| W1: API + Hooks | ✅ verified | 3 trigger methods with snake→camel normalization, 3 mutation hooks, lead query hooks |
| W2: KB Trigger | ✅ verified | Conditional render on `resolved` status in both support and legacy case detail |
| W3: Leads + Prospecting | ✅ verified | 3rd Sales tab, lead detail, AgentActivitySection widened, Deal Risk placeholder |
| W4: Insights Screen | ✅ verified | Query + date range form, RFC3339 serialization, navigation to run detail |
| W5: Integration Tests | ✅ verified | 22 dedicated tests covering cross-screen interactions |
| W5: Maestro E2E | ✅ verified | Implemented by extending `mobile/maestro/authenticated-audit.yaml` and `mobile/maestro/seed-and-run.sh` instead of introducing a separate `agent-triggers.yaml`; `npm run screenshots` now covers KB trigger, lead prospecting, deal-risk placeholder, and insights entry. |

**Audit clarification:**
- The original review expected a dedicated `mobile/maestro/agent-triggers.yaml` file, but the implementation intentionally reused the existing authenticated visual-audit flow.
- The canonical Maestro deliverable for Wave 5 is therefore the extended `mobile/maestro/authenticated-audit.yaml` plus the seed export additions in `mobile/maestro/seed-and-run.sh`, not a new standalone YAML.
- No Wave 5 artifact remains open on that point; the remaining follow-up is visual polish of individual screens, not missing Maestro plumbing.

---

## Wave 1: API Layer + Hooks (Foundation)

> **Dependency**: None. All subsequent waves depend on this.
> **Goal**: Wire the data plumbing — API methods, mutation hooks, leads queries — so UI tasks can consume them directly.

### W1-T1: Agent-specific trigger methods in API layer

**Files to modify:**
- `mobile/src/services/api.agents.ts` — add `triggerProspectingRun`, `triggerKBRun`, `triggerInsightsRun`
- `mobile/src/services/api.types.ts` — add `QueuedAgentTriggerResponse`

**Pattern to follow:** endpoint-specific helpers in `api.agents.ts`, but do not copy the `triggerSupportRun` response shape directly because Support returns a different payload envelope.

**Request schemas (from Go handlers):**
```typescript
triggerProspectingRun(ctx: { lead_id: string; language?: string })
  → POST /bff/api/v1/agents/prospecting/trigger

triggerKBRun(ctx: { case_id: string; language?: string })
  → POST /bff/api/v1/agents/kb/trigger

triggerInsightsRun(ctx: { query: string; date_from?: string; date_to?: string; language?: string })
  → POST /bff/api/v1/agents/insights/trigger
```

**Normalized response contract for all 3 methods:**
```typescript
interface QueuedAgentTriggerResponse {
  runId: string;
  status: string;
  agent: string;
}
```

Each method should normalize backend `{ run_id, status, agent }` to the shared camelCase contract above so screens can navigate with `runId` without duplicating snake_case handling.

**Test first:**
- `mobile/__tests__/services/api.agents.test.ts` — verify correct URLs, payload shapes, and snake_case → camelCase response normalization

---

### W1-T2: Mutation hooks for each agent trigger

**Files to modify:**
- `mobile/src/hooks/useWedge.ts` — add `useTriggerProspectingAgent`, `useTriggerKBAgent`, `useTriggerInsightsAgent`

**Pattern to follow:** `useTriggerSupportAgent()` (line 138-151 of useWedge.ts)

Each hook:
- Calls its respective `agentApi.triggerXxxRun` method
- `onSuccess` invalidates `wedgeQueryKeys.agentRuns(workspaceId)` for cache refresh
- Returns the normalized `QueuedAgentTriggerResponse` so UI code can use `mutateAsync()` and route to `/activity/{runId}` when needed

**Test first:**
- `mobile/__tests__/hooks/useWedge.triggers.test.ts` — verify mutate calls correct API, `mutateAsync()` resolves normalized `runId`, onSuccess invalidates cache, and `isPending` reflects loading

---

### W1-T3: Leads API + hooks for Prospecting context

**Files to modify:**
- `mobile/src/services/api.ts` — add `getLeads(workspaceId, pagination)` and `getLead(id)` to `crmApi` if line-count stays under the architecture gate; otherwise extract CRM methods to a dedicated file in the same wave and re-export from `api.ts`
- `mobile/src/hooks/useCRM.ts` — add `queryKeys.leads`, `queryKeys.lead`, `useLeads()` (infinite query), and `useLead(id)` (single query)

**Pattern to follow:** `useDeals()` / `useDeal(id)` in useCRM.ts

**Lead typing note:** mobile does not currently expose a shared `Lead` type. Wave 1 can keep lead list/detail interfaces screen-local, but the hooks and query keys must exist before Wave 3 starts.

**Verification (Wave 1 exit):**
- `npx jest --testPathPattern="api.agents|useWedge.triggers"` — all pass
- Three trigger API methods exported from `agentApi`
- Three mutation hooks exported from `useWedge`
- `useLeads` and `useLead` hooks functional
- Lead query keys added to `useCRM.ts`

---

## Wave 2: KB Agent Trigger in Case Detail (Gap-8 partial)

> **Dependency**: W1-T1, W1-T2
> **Goal**: Add a "Generate KB Article" button to support case detail, visible only when `case.status === 'resolved'` (backend precondition).

### W2-T1: KB trigger tests (TDD)

**File to create:**
- `mobile/__tests__/app/(tabs)/support/kb-trigger.test.tsx`

**Also modify:**
- `mobile/__tests__/app/(tabs)/support/[id].test.tsx` — update screen mocks to include `useTriggerKBAgent` without regressing the existing support detail tests

**Test cases:**
1. KB trigger button NOT rendered when `case.status !== 'resolved'` (open, in_progress)
2. KB trigger button IS rendered when `case.status === 'resolved'` (testID: `kb-trigger-button`)
3. Press calls `useTriggerKBAgent().mutate({ case_id: caseData.id })`
4. Button disabled while `isPending === true`
5. Button text shows "Running..." during pending state

---

### W2-T2: KB trigger button in support case detail

**File to modify:**
- `mobile/app/(tabs)/support/[id].tsx` — add KB trigger section after the Support Agent trigger (line 225)

**Placement:** Between the existing Support Agent trigger and `AgentActivitySection` (line 227). Conditional render:
```tsx
{caseData.status === 'resolved' && (
  <View style={styles.section}>
    <Button mode="outlined" testID="kb-trigger-button"
      disabled={triggerKB.isPending}
      onPress={() => triggerKB.mutate({ case_id: caseData.id })}>
      {triggerKB.isPending ? 'Running...' : 'Generate KB Article'}
    </Button>
  </View>
)}
```

**Also modify:**
- `mobile/app/(tabs)/cases/[id].tsx` — same KB trigger in legacy case detail, placed after `AgentActivitySection` in `renderCaseContent` (line 98)

**Verification (Wave 2 exit):**
- KB button only visible for resolved cases
- Trigger sends `{ case_id }` payload to `/agents/kb/trigger`
- `npx jest --testPathPattern="kb-trigger"` — all pass

---

## Wave 3: Leads Navigation + Prospecting Trigger (Gap-7)

> **Dependency**: W1-T3 (leads hooks), W1-T2 (prospecting hook)
> **Goal**: Add Leads sub-tab inside Sales, with lead detail screen containing a Prospecting Agent trigger.

### W3-T1: Prospecting trigger tests (TDD)

**File to create:**
- `mobile/__tests__/app/(tabs)/sales/prospecting-trigger.test.tsx`

**Also modify:**
- `mobile/__tests__/app/(tabs)/sales/index.test.tsx` — extend the existing segmented-sales tests for the third `Leads` tab
- `mobile/__tests__/components/agents/AgentActivitySection.test.tsx` — add a `lead` entity case if Wave 3 widens the shared section

**Test cases:**
1. Prospecting trigger button rendered on lead detail (testID: `prospecting-trigger-button`)
2. Press calls `useTriggerProspectingAgent().mutate({ lead_id })`
3. Button disabled while `isPending`
4. Successful trigger request navigates to agent run detail via `router.push`

---

### W3-T2: Lead detail screen

**Files to create:**
- `mobile/app/(tabs)/sales/leads/[id].tsx` — Lead detail with:
  - Lead metadata (name, email, source, status)
  - Prospecting trigger button (testID: `prospecting-trigger-button`)
  - `AgentActivitySection` for entity type `lead`

**Files to modify:**
- `mobile/app/(tabs)/sales/_layout.tsx` — register `leads/[id]` in the Sales stack
- `mobile/src/components/agents/AgentActivitySection.tsx` — widen `entityType` to include `lead`

**Pattern to follow:** Sales Account detail (`mobile/app/(tabs)/sales/[id].tsx`) for layout structure. Support case detail for trigger button integration.

**Why only detail gets its own route:** the leads list lives inside the segmented Sales screen (`sales/index.tsx`). Creating a separate `sales/leads/index.tsx` would duplicate the same surface and introduce conflicting navigation responsibilities.

---

### W3-T3: Add "Leads" tab to Sales screen

**File to modify:**
- `mobile/app/(tabs)/sales/index.tsx`

**Changes:**
- Extend `type Tab = 'accounts' | 'deals' | 'leads'`
- Add third tab button in `TabBar` (testID: `sales-tab-leads`)
- Add `LeadsTab` component following `AccountsTab` pattern, using `useLeads()` hook
- Each lead row navigates to `/sales/leads/{id}`

---

### W3-T4: Deal Risk trigger placeholder (Gap-7 blocked portion)

**File to modify:**
- `mobile/app/(tabs)/sales/deal-[id].tsx` — add disabled "Analyze Deal Risk" button to the actual wedge deal-detail implementation used by `/sales/deals/{id}`

**Placement:** After the "Open Copilot" button (line 93). Permanently disabled with "Coming Soon" label:
```tsx
<View style={styles.section}>
  <Button mode="outlined" testID="deal-risk-trigger-button" disabled={true}>
    Analyze Deal Risk (Coming Soon)
  </Button>
</View>
```

No API wiring. Activated when Gap-5+6 backend ships.

**Verification (Wave 3 exit):**
- Sales screen has 3 tabs: Accounts, Deals, Leads
- Leads list renders and navigates to lead detail
- Prospecting trigger sends `{ lead_id }` to `/agents/prospecting/trigger`
- Deal Risk button visible but disabled
- `npx jest --testPathPattern="prospecting-trigger"` — all pass

---

## Wave 4: Insights Agent Screen (Gap-8 completion)

> **Dependency**: W1-T1, W1-T2 (independent of W2 and W3 — can run in parallel)
> **Goal**: New standalone screen for ad-hoc analytical queries via the Insights Agent.

### W4-T1: Insights screen tests (TDD)

**File to create:**
- `mobile/__tests__/app/(tabs)/activity/insights.test.tsx`

**Also modify:**
- `mobile/__tests__/app/(tabs)/activity/index.test.tsx` — extend the Activity landing screen tests for the new navigation affordance

**Test cases:**
1. Query input rendered (testID: `insights-query-input`)
2. Run button rendered (testID: `insights-run-button`)
3. Run button disabled when query empty
4. Run button enabled when query has content
5. Press calls `useTriggerInsightsAgent().mutate({ query, date_from?, date_to? })`
6. Loading state disables button and shows spinner
7. Date inputs present (testID: `insights-date-from`, `insights-date-to`) and selected values are serialized to RFC3339 before trigger

---

### W4-T2: Create Insights screen

**File to create:**
- `mobile/app/(tabs)/activity/insights.tsx`

**Structure:**
- `TextInput` multiline for query (testID: `insights-query-input`)
- Optional date range inputs — `date_from` / `date_to` (testID: `insights-date-from`, `insights-date-to`), serialized to RFC3339 because the backend rejects non-RFC3339 values
- "Run Insights" button (testID: `insights-run-button`), disabled when query empty
- Uses `useTriggerInsightsAgent()` hook
- On successful trigger request: navigate to `/activity/{runId}` (agent run detail), where both grounded answers and safe rejections are inspected
- V1 scope ends at form submission + navigation to run detail. Embedded run history stays out of scope for this wave to keep W4 independent of entity-bound activity components.

**Pattern to follow:** Sales Brief screen (`mobile/app/(tabs)/sales/[id]/brief.tsx`) for the hook+render pattern. Support case detail for button states.

---

### W4-T3: Navigation wiring for Insights

**Files to modify:**
- `mobile/app/(tabs)/activity/_layout.tsx` — add `insights` as a Stack.Screen route
- `mobile/app/(tabs)/activity/index.tsx` — add "Insights" navigation card/button (testID: `activity-insights-nav`) that navigates to `/activity/insights`

**Verification (Wave 4 exit):**
- Insights screen renders query input, date inputs, run button
- Empty query disables run button
- Trigger sends `{ query, date_from?, date_to? }` to `/agents/insights/trigger` with RFC3339 date strings
- Navigation: Activity tab → Insights card → Insights screen
- `npx jest --testPathPattern="insights"` — all pass

---

## Wave 5: Integration Tests + E2E (Hardening)

> **Dependency**: All previous waves
> **Goal**: Cross-screen integration validation and Maestro E2E flow.

### W5-T1: Integration tests

**File to create:**
- `mobile/__tests__/integration/agent-triggers.test.tsx`

**Coverage:**
- KB trigger conditional rendering across case statuses
- Prospecting trigger cache invalidation propagation
- Insights trigger with date range sends RFC3339 strings
- All triggers follow disabled-while-pending pattern

---

### W5-T2: Maestro E2E flow

**Files to modify:**
- `mobile/maestro/authenticated-audit.yaml`
- `mobile/maestro/seed-and-run.sh`

**Flow:**
1. Reuse the authenticated visual-audit bootstrap path already used by `npm run screenshots`
2. Navigate to Support → open resolved case → verify KB trigger button visible → screenshot
3. Navigate to Sales → Leads tab → open lead → verify Prospecting trigger → screenshot
4. Navigate to Sales → open deal detail → verify Deal Risk placeholder → screenshot
5. Navigate to Activity → Insights → verify entry screen renders correctly → screenshot

---

### W5-T3: Seed data for E2E

**Ensure seed data includes:**
- At least one resolved case (for KB trigger)
- At least one lead (for Prospecting trigger)
- Add `SEED_LEAD_ID` and `SEED_RESOLVED_CASE_ID` to seed variables

**Verification (Wave 5 exit):**
- `npx jest --coverage` — no regressions, all new tests pass
- `npm run screenshots` completes without failures using the extended authenticated Maestro flow
- Coverage delta: positive across components, hooks, services
- `docs/dashboards/fr-uc-status.md` updated so UC-S2 / UC-K1 / UC-D1 mobile status and Gap-7 / Gap-8 rows reflect the implemented state

---

## Files Summary

### New Files (8 planned)

| File | Wave | Purpose |
|------|------|---------|
| `mobile/__tests__/services/api.agents.test.ts` | W1 | API method unit tests |
| `mobile/__tests__/hooks/useWedge.triggers.test.ts` | W1 | Hook mutation tests |
| `mobile/__tests__/app/(tabs)/support/kb-trigger.test.tsx` | W2 | KB trigger tests |
| `mobile/__tests__/app/(tabs)/sales/prospecting-trigger.test.tsx` | W3 | Prospecting tests |
| `mobile/app/(tabs)/sales/leads/[id].tsx` | W3 | Lead detail + trigger |
| `mobile/__tests__/app/(tabs)/activity/insights.test.tsx` | W4 | Insights screen tests |
| `mobile/app/(tabs)/activity/insights.tsx` | W4 | Insights query screen |
| `mobile/__tests__/integration/agent-triggers.test.tsx` | W5 | Integration tests |

### Modified Files (core set; additional test updates expected)

| File | Wave | Change |
|------|------|--------|
| `mobile/src/services/api.agents.ts` | W1 | +3 trigger methods with normalized queued response |
| `mobile/src/services/api.types.ts` | W1 | +`QueuedAgentTriggerResponse` |
| `mobile/src/services/api.ts` | W1 | +lead CRM methods or re-export after CRM split |
| `mobile/src/hooks/useWedge.ts` | W1 | +3 mutation hooks returning normalized `runId` |
| `mobile/src/hooks/useCRM.ts` | W1 | +lead query keys, `useLeads`, `useLead` |
| `mobile/maestro/authenticated-audit.yaml` | W5 | Extend existing screenshot flow for KB, Prospecting, Deal Risk, and Insights surfaces |
| `mobile/maestro/seed-and-run.sh` | W5 | Export lead/resolved-case seed vars into Maestro runtime |
| `mobile/app/(tabs)/support/[id].tsx` | W2 | +KB trigger button (conditional) |
| `mobile/app/(tabs)/cases/[id].tsx` | W2 | +KB trigger button (legacy) |
| `mobile/__tests__/app/(tabs)/support/[id].test.tsx` | W2 | +mock updates for new KB hook |
| `mobile/app/(tabs)/sales/_layout.tsx` | W3 | +`leads/[id]` route registration |
| `mobile/app/(tabs)/sales/index.tsx` | W3 | +Leads tab |
| `mobile/__tests__/app/(tabs)/sales/index.test.tsx` | W3 | +third tab assertions |
| `mobile/src/components/agents/AgentActivitySection.tsx` | W3 | +`lead` entity support |
| `mobile/__tests__/components/agents/AgentActivitySection.test.tsx` | W3 | +lead entity coverage |
| `mobile/app/(tabs)/sales/deal-[id].tsx` | W3 | +Deal Risk placeholder |
| `mobile/app/(tabs)/activity/_layout.tsx` | W4 | +insights route |
| `mobile/app/(tabs)/activity/index.tsx` | W4 | +Insights nav card |
| `mobile/__tests__/app/(tabs)/activity/index.test.tsx` | W4 | +Insights nav assertions |
| `docs/dashboards/fr-uc-status.md` | W5 | +UC / gap status sync after implementation |

---

## Design Decisions

1. **Specialized API methods over generic `triggerRun`**: Each agent has a distinct request schema (`lead_id` vs `case_id` vs `query`). Specialized methods provide type safety. Their return values must also be normalized because the queued trigger endpoints return `run_id` in snake_case while `triggerSupportRun` follows a different contract.

2. **Hooks in `useWedge.ts`**: The existing support trigger lives here. All mutation hooks with cache invalidation belong in `useWedge.ts`. Read queries stay in `useAgentSpec.ts`.

3. **Leads under Sales tab**: Leads are a sales concept. Adding a third tab (`accounts | deals | leads`) keeps navigation consistent with the wedge model instead of creating a new top-level tab. The leads list stays inline inside `sales/index.tsx`; only the lead detail gets its own route.

4. **KB trigger conditional render (not disabled)**: A resolved case shows the button; other statuses hide it. This is clearer UX than a disabled button with a tooltip — the user doesn't need to wonder why it's disabled.

5. **Deal Risk as disabled placeholder**: The button exists in the UI (testID wired for E2E) but does nothing until Gap-5+6 backend ships. Only `disabled` and `onPress` need updating later.

6. **Lead detail reuses shared activity UI**: Instead of inventing a second run-history widget for leads, Wave 3 widens `AgentActivitySection` to accept `lead`.

7. **Insights under Activity tab**: The Insights agent queries data across the CRM, not tied to a specific entity. Activity tab already shows agent run history — natural home for an analytics query screen.

8. **No BFF changes required**: The transparent proxy (`/bff/api/v1/* → /api/v1/*`) already forwards all agent endpoints. No new BFF routes needed.

---

## Dependency Graph

```
W1-T1 (API methods) ─────────┬──→ W2 (KB Trigger)
W1-T2 (Mutation hooks) ──────┤
                              ├──→ W4 (Insights Screen)  [parallel with W2, W3]
W1-T3 (Leads API + hooks) ───┴──→ W3 (Leads + Prospecting)

W2 + W3 + W4 ──────────────────→ W5 (Integration + E2E)
```

W2, W3, and W4 can run in parallel once W1 is complete.

---

## Gap Closure Impact

| Gap | UC | Status After | Dashboard Change |
|-----|-----|-------------|------------------|
| Gap-7 (partial) | UC-S2 | Prospecting trigger in mobile | Mobile: ❌ → ✅ |
| Gap-7 (partial) | UC-S3 | Deal Risk placeholder (backend pending) | Mobile: ❌ → ⏳ |
| Gap-8 (partial) | UC-K1 | KB trigger in case detail | Mobile: ❌ → ✅ |
| Gap-8 (partial) | UC-D1 | Insights screen | Mobile: ❌ → ✅ |
