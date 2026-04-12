---
doc_type: task
id: governance-mobile-enhancement
title: Mobile Governance Layer Enhancement
status: completed
phase: post-mvp
week: ~
tags: [mobile, governance, audit, usage, react-native]
fr_refs: [FR-060, FR-070, FR-071]
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - mobile/app/(tabs)/governance/index.tsx
  - mobile/app/(tabs)/governance/audit.tsx
  - mobile/app/(tabs)/governance/usage.tsx
  - mobile/app/(tabs)/activity/[id].tsx
  - mobile/src/services/api.types.ts
  - mobile/src/services/api.secondary.ts
  - mobile/src/hooks/useWedge.ts
created: 2026-04-12
completed: 2026-04-12
---

# Mobile Governance Layer Enhancement

## Context

The Go backend has a complete governance system (audit trail, usage/quota metering, approval workflows, policy engine) but the mobile app only surfaces a minimal summary screen: 20 usage events shown as `metric_name + value` rows, and quota progress bars. This plan closes the gap in three waves, prioritized by user value for a governed AI CRM.

**No BFF changes needed**: `bff/src/routes/proxy.ts` already transparently proxies all `/bff/api/v1/*` → `/api/v1/*`. All new mobile API calls work through the existing proxy.

**Critical discovery**: `GET /api/v1/governance/summary` and `GET /api/v1/usage` return full usage fields in camelCase (`actorType`, `toolName`, `modelName`, `estimatedCost`, `latencyMs`, `createdAt`). The mobile app still typed and rendered `UsageEvent` in snake_case. The implementation corrected the mobile contract first, then built Waves 2 and 3 on top of that canonical shape.

**Second discovery**: `ListUsage` uses `page.Limit` but ignores `page.Offset` — it does not support true pagination, only page-size control. The usage drilldown therefore loads more by increasing `limit`, not by accumulating offset pages.

**Third discovery**: `/api/v1/audit/events` supports server-side filters for `actor_id`, `entity_type`, `action`, `outcome`, and date range, but not `actor_type`. Wave 2 narrowed the initial mobile filter bar to supported outcome filters instead of introducing a misleading client-side-only filter.

---

## Wave 1 — Governance Screen Enrichment (Quick Win)

**Status**: [x] VERIFIED 2026-04-12 — implemented in repo and contract-corrected against real backend payloads

### Files created/modified
- CREATED: `mobile/app/(tabs)/governance/_layout.tsx`
- CREATED: `mobile/src/components/governance/UsageDetailCard.tsx`
- CREATED: `mobile/__tests__/components/governance/UsageDetailCard.test.tsx` (12 tests)
- MODIFIED: `mobile/app/(tabs)/governance/index.tsx` (local types removed, UsageDetailCard + nav links)
- MODIFIED: `mobile/__tests__/app/(tabs)/governance/index.test.tsx` (14 tests, 4 new Wave 1 assertions)

**Goal**: Replace barebones `metric_name + value` rows with a rich `UsageDetailCard`. Add navigation entry points to Audit (Wave 2) and Usage list (Wave 3). Create the `_layout.tsx` required for sub-screens.

### Files to CREATE

| File | Purpose |
|------|---------|
| `mobile/app/(tabs)/governance/_layout.tsx` | Stack navigator — prerequisite for Waves 2 & 3 sub-screens |
| `mobile/src/components/governance/UsageDetailCard.tsx` | Rich card: actorType badge, toolName, modelName, cost (€X.XXXX), latencyMs, createdAt |
| `mobile/__tests__/components/governance/UsageDetailCard.test.tsx` | TDD tests (write first) |

### Files to MODIFY

| File | What changes |
|------|-------------|
| `mobile/app/(tabs)/governance/index.tsx` | Remove local `UsageEvent` type; import canonical from `api.types.ts`; replace bare rows with `UsageDetailCard`; add "View All" stub + "Audit Trail →" nav link; add `sectionHeaderRow` style |
| `mobile/__tests__/app/(tabs)/governance/index.test.tsx` | Extend (keep existing tests); add: rich usage card renders, "View All" testID, audit trail link testID |

### _layout.tsx pattern (follows `activity/_layout.tsx` exactly)

```tsx
import React from 'react';
import { Stack } from 'expo-router';

export default function GovernanceLayout() {
  return (
    <Stack screenOptions={{ headerShown: false, animation: 'slide_from_right' }}>
      <Stack.Screen name="index" />
      <Stack.Screen name="audit" options={{ title: 'Audit Trail', headerShown: true }} />
      <Stack.Screen name="usage" options={{ title: 'Usage Events', headerShown: true }} />
    </Stack>
  );
}
```

### UsageDetailCard props

```typescript
interface UsageDetailCardProps {
  event: UsageEvent;           // canonical type from api.types.ts (camelCase fields)
  testIDPrefix?: string;
  onPress?: () => void;
}
// Fields: actorType badge | toolName (primary) | modelName (secondary) | estimatedCost (€) | latencyMs | createdAt
// Fallback for optional fields: '—'
// Card style: marginBottom: 8, marginHorizontal: 16 (matches ApprovalCard/SignalCard)
```

### Type fix: canonical UsageEvent

The canonical mobile `UsageEvent` now uses camelCase (matching Go JSON). Wave 1 verification showed the screen and tests had been implemented against an outdated snake_case shape; this task corrected the client contract and reused it across governance and activity surfaces.

### TDD tests to write FIRST for UsageDetailCard

- Renders actorType badge
- Renders toolName as primary label; renders `'—'` when null
- Renders modelName with secondary style; renders `'—'` when null
- Renders estimatedCost formatted as `€X.XXXX`
- Renders latencyMs; renders `'—'` when null
- Renders createdAt as localized date string
- Calls onPress when tapped

### index.tsx navigation additions

```tsx
// "View All" for usage — stub until Wave 3:
<TouchableOpacity testID="governance-view-all-usage" onPress={() => { /* Wave 3 */ }}>
  <Text>View All</Text>
</TouchableOpacity>

// Audit Trail link:
<TouchableOpacity testID="governance-audit-trail-link" onPress={() => router.push(wedgeHref('/governance/audit'))}>
  <Text>Audit Trail →</Text>
</TouchableOpacity>
```

---

## Wave 2 — Audit Trail Screen

**Status**: [x] COMPLETED 2026-04-12 — audit screen, API methods, hooks, cards, and tests added

**Goal**: New `governance/audit` screen with filterable, paginated list of audit events. Read-only compliance view. Tap to expand inline (no push, avoids deep stack).

### New Types (append to `mobile/src/services/api.types.ts`)

```typescript
export type AuditOutcome = 'success' | 'denied' | 'error';
export type AuditActorType = 'user' | 'agent' | 'system';

export interface AuditEvent {
  id: string;
  workspace_id: string;
  actor_id: string;
  actor_type: AuditActorType;
  action: string;
  entity_type?: string;
  entity_id?: string;
  details?: Record<string, unknown>;
  outcome: AuditOutcome;
  trace_id?: string;
  ip_address?: string;
  created_at: string;
}

export interface AuditFilters {
  actor_id?: string;
  entity_type?: string;
  action?: string;
  outcome?: AuditOutcome;
  date_from?: string;   // ISO string — date range reserved for future
  date_to?: string;
}

// Generic paginated wrapper — matches Go writePaginatedOr500 shape
export interface PaginatedResponse<T> {
  data: T[];
  meta: { total: number; limit: number; offset: number; };
}
```

Add re-exports to `mobile/src/services/api.ts`:
```typescript
export type { AuditEvent, AuditFilters, AuditOutcome, AuditActorType, PaginatedResponse } from './api.types';
```

### New API Client Methods (extend `governanceApi` in `api.secondary.ts`)

```typescript
getAuditEvents: async (
  workspaceId: string,
  filters?: AuditFilters,
  pagination?: { page?: number; limit?: number }
) => {
  const limit = pagination?.limit ?? 20;
  const offset = ((pagination?.page ?? 1) - 1) * limit;
  // Audit endpoint uses offset (not page) — see handlers/audit.go:53
  const response = await apiClient.get('/bff/api/v1/audit/events', {
    params: { workspace_id: workspaceId, limit, offset, ...filters },
  });
  return response.data as PaginatedResponse<AuditEvent>;
},

getAuditEventById: async (workspaceId: string, id: string) => {
  const response = await apiClient.get(`/bff/api/v1/audit/events/${id}`, {
    params: { workspace_id: workspaceId },
  });
  return response.data as AuditEvent;
},
```

> **Pagination note**: Audit endpoint (`handlers/audit.go:53`) uses `offset`. Usage endpoint (`handlers/usage.go:64`) uses `page.Limit` only (no offset). The client must translate correctly per endpoint.

### New Hooks (extend `useWedge.ts`)

```typescript
// Add to wedgeQueryKeys:
auditEvents: (workspaceId: string, filters?: AuditFilters, page?: number) =>
  ['audit-events', workspaceId, filters ?? {}, page ?? 1] as const,

// New hook:
export function useAuditEvents(filters?: AuditFilters, page = 1) {
  const workspaceId = useWorkspaceId();
  return useQuery({
    queryKey: wedgeQueryKeys.auditEvents(workspaceId ?? '', filters, page),
    queryFn: () => governanceApi.getAuditEvents(workspaceId!, filters, { page, limit: 20 }),
    staleTime: 30_000,   // shorter than summary — compliance data should feel fresh
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}
```

### Files to CREATE

| File | Purpose |
|------|---------|
| `mobile/app/(tabs)/governance/audit.tsx` | Audit trail screen (filter bar + FlatList + load-more) |
| `mobile/src/components/governance/AuditEventCard.tsx` | Read-only card: action, actor, outcome badge, expandable detail |
| `mobile/src/components/governance/AuditFilterBar.tsx` | Outcome chip row (All/Success/Denied/Error) aligned with server-supported filters |
| `mobile/__tests__/app/(tabs)/governance/audit.test.tsx` | TDD screen tests |
| `mobile/__tests__/components/governance/AuditEventCard.test.tsx` | TDD card tests |
| `mobile/__tests__/components/governance/AuditFilterBar.test.tsx` | TDD filter bar tests |

### Files to MODIFY

| File | What changes |
|------|-------------|
| `mobile/src/services/api.types.ts` | Append W2 types |
| `mobile/src/services/api.secondary.ts` | Extend `governanceApi` with `getAuditEvents`, `getAuditEventById` |
| `mobile/src/services/api.ts` | Add W2 type re-exports |
| `mobile/src/hooks/useWedge.ts` | Add `auditEvents` query key + `useAuditEvents` hook |
| `mobile/__tests__/hooks/useWedge.test.ts` | Extend: audit hook tests |
| `mobile/__tests__/services/api.test.ts` | Extend: audit API method tests |

### AuditEventCard props & behavior

```typescript
interface AuditEventCardProps {
  event: AuditEvent;
  expanded?: boolean;
  onPress: () => void;
  testIDPrefix?: string;
}
// Collapsed: action (title) | actor_type + actor_id (subtitle) | outcome badge | created_at
// Expanded: adds entity_type · entity_id | trace_id | details as JSON string (monospace)
// Outcome badge colors: success → #10B981 | denied → #EF4444 | error → #DC2626
```

### AuditFilterBar props

```typescript
interface AuditFilterBarProps {
  filters: AuditFilters;
  onChange: (f: AuditFilters) => void;
}
// Chip style: follows exact pattern from activity/index.tsx FilterChips
// Initial mobile scope uses outcome chips only because actor_type is not supported server-side
// Date range: deferred (too complex for this wave)
```

### audit.tsx screen structure

```
1. AuditFilterBar (top)
2. FlatList of AuditEventCard — one expanded ID tracked in local state
3. Load-more: onEndReached → setPage(p => p + 1)
4. allEvents accumulated: useEffect on data → setAllEvents(prev => page === 1 ? data.data : [...prev, ...data.data])
5. Filter change resets: useEffect on filters → setPage(1); setAllEvents([])
6. States: loading (ActivityIndicator) | empty ("No audit events found") | error (colors.error)
```

> **No `useInfiniteQuery`**: consistent with the rest of the codebase (activity screen, inbox). Maintains the `useQuery` + local state accumulation pattern.

### TDD tests to write FIRST

**AuditEventCard**: renders action as title; renders outcome badge in correct color; collapsed state hides entity info; expanded shows entity_type + entity_id + trace_id; calls onPress when tapped.

**AuditFilterBar**: renders all outcome chips; calls onChange with correct outcome filter; active chip shows primary background.

**audit.tsx screen**: loading indicator while fetching; cards rendered on data; empty state; error state; filter change resets to page 1.

---

## Wave 3 — Usage Drilldown Screen

**Status**: [x] COMPLETED 2026-04-12 — usage drilldown, summary card, deep link activation, and activity cross-link added

**Goal**: Full usage event list at `governance/usage` with rich `UsageDetailCard`, cost summary card at top, and optional `run_id` filter. Activates the "View All" stub from Wave 1. Enables cross-tab deep link from `activity/[id].tsx`.

### New Types (append to `mobile/src/services/api.types.ts`)

```typescript
export interface UsageFilters {
  run_id?: string;
}

// Computed client-side from first-page events (no dedicated Go endpoint)
export interface UsageCostSummary {
  totalCost: number;
  totalInputUnits: number;
  totalOutputUnits: number;
  eventCount: number;
}
```

Add re-exports to `mobile/src/services/api.ts`:
```typescript
export type { UsageFilters, UsageCostSummary } from './api.types';
```

### New API Client Methods (extend `governanceApi`)

```typescript
getUsageEvents: async (
  workspaceId: string,
  filters?: UsageFilters,
  pagination?: { page?: number; limit?: number }
) => {
  // Usage endpoint ignores offset, so "page" increases the requested limit.
  const limit = (pagination?.page ?? 1) * (pagination?.limit ?? 20);
  const response = await apiClient.get('/bff/api/v1/usage', {
    params: { workspace_id: workspaceId, limit, ...filters },
  });
  return response.data as PaginatedResponse<UsageEvent>;
},
```

### New Hooks (extend `useWedge.ts`)

```typescript
usageEvents: (workspaceId: string, filters?: UsageFilters, page?: number) =>
  ['usage-events', workspaceId, filters ?? {}, page ?? 1] as const,

export function useUsageEvents(filters?: UsageFilters, page = 1) {
  const workspaceId = useWorkspaceId();
  return useQuery({
    queryKey: wedgeQueryKeys.usageEvents(workspaceId ?? '', filters, page),
    queryFn: () => governanceApi.getUsageEvents(workspaceId!, filters, { page, limit: 20 }),
    staleTime: 60_000,
    gcTime: 5 * 60_000,
    retry: 1,
    refetchOnWindowFocus: false,
    enabled: !!workspaceId,
  });
}
```

### Files to CREATE

| File | Purpose |
|------|---------|
| `mobile/app/(tabs)/governance/usage.tsx` | Usage list screen (cost summary + FlatList + load-more + run_id deep link) |
| `mobile/src/components/governance/UsageCostSummaryCard.tsx` | 2×2 stat grid: total cost / event count / input units / output units |
| `mobile/__tests__/app/(tabs)/governance/usage.test.tsx` | TDD screen tests |
| `mobile/__tests__/components/governance/UsageCostSummaryCard.test.tsx` | TDD card tests |

### Files to MODIFY

| File | What changes |
|------|-------------|
| `mobile/src/services/api.types.ts` | Append W3 types |
| `mobile/src/services/api.secondary.ts` | Add `getUsageEvents` |
| `mobile/src/services/api.ts` | Add W3 type re-exports |
| `mobile/src/hooks/useWedge.ts` | Add `usageEvents` key + `useUsageEvents` |
| `mobile/app/(tabs)/governance/index.tsx` | Activate "View All" link → `router.push(wedgeHref('/governance/usage'))` |
| `mobile/__tests__/hooks/useWedge.test.ts` | Extend: usage hook tests |

### UsageCostSummaryCard props

```typescript
interface UsageCostSummaryCardProps {
  summary: UsageCostSummary;
  testIDPrefix?: string;
}
// Layout: 2x2 stat grid (RN-Paper Card)
// Top row: totalCost (€X.XXXX) | eventCount
// Bottom row: totalInputUnits | totalOutputUnits
// testIDs: ${prefix}-total-cost | -event-count | -input-units | -output-units
```

### usage.tsx screen structure

```
1. useLocalSearchParams<{ run_id?: string }>() — pre-fill run_id from deep link
2. UsageCostSummaryCard (computed from first-page events)
3. FlatList of UsageDetailCard (reuse Wave 1 component)
4. Load-more increases requested limit (`page * 20`) because `/usage` does not support offset
5. States: loading | empty ("No usage events found") | error
```

**Deep link from activity**: In `mobile/app/(tabs)/activity/[id].tsx`, the run detail UsageSection adds a "View Full Usage" button:
```tsx
router.push(wedgeHref(`/governance/usage?run_id=${run.id}`))
```

### TDD tests to write FIRST

**UsageCostSummaryCard**: renders totalCost as €X.XXXX; renders eventCount; renders inputUnits and outputUnits; handles zero values without crash.

**usage.tsx**: loading state; cost summary card renders with events; usage event cards render; empty state; pre-fills run_id from route params; load-more increases requested limit.

---

## Dependencies Between Waves

```
Wave 1 must complete before Wave 2:
  - _layout.tsx (required for sub-screen routing)
  - api.types.ts canonical UsageEvent (W2 AuditEvent builds on this file)

Wave 1 must complete before Wave 3:
  - "View All" stub activation
  - _layout.tsx (same as above)

Wave 2 and Wave 3 are INDEPENDENT after the UsageEvent contract correction from Wave 1.
Wave 3 also updates `activity/[id].tsx` because that screen consumed the same outdated usage shape and is the canonical deep-link origin for `run_id`.
```

---

## Verification

### Per-wave verification

**Wave 1**
```bash
cd mobile && npx jest __tests__/components/governance/UsageDetailCard.test.tsx --no-coverage
npx jest __tests__/app/\(tabs\)/governance/index.test.tsx --no-coverage
npx tsc --noEmit
```

**Wave 2**
```bash
npx jest __tests__/components/governance/AuditEventCard.test.tsx --no-coverage
npx jest __tests__/components/governance/AuditFilterBar.test.tsx --no-coverage
npx jest __tests__/app/\(tabs\)/governance/audit.test.tsx --no-coverage
npx jest __tests__/hooks/useWedge.test.ts --no-coverage
npx tsc --noEmit
```

**Wave 3**
```bash
npx jest __tests__/components/governance/UsageCostSummaryCard.test.tsx --no-coverage
npx jest __tests__/app/\(tabs\)/governance/usage.test.tsx --no-coverage
npx tsc --noEmit
```

### Full suite before commit
```bash
bash scripts/qa-mobile-prepush.sh
```

### Manual E2E verification
1. Start Go backend: `./fenixcrm serve --port 8080`
2. Start BFF: `node bff/dist/index.js --port 3000`
3. Run Expo: `cd mobile && npx expo start`
4. Navigate to Governance tab → verify rich usage cards display actorType/toolName/cost
5. Tap "Audit Trail →" → verify audit screen loads with filter chips
6. Apply outcome filter → verify list updates
7. Tap "View All" on governance screen → verify usage list screen opens
8. Verify cross-tab link from Activity → Run Detail → "View Full Usage" navigates to usage screen pre-filtered by run_id

---

## Files Affected (summary)

### Wave 1
- CREATE: `mobile/app/(tabs)/governance/_layout.tsx`
- CREATE: `mobile/src/components/governance/UsageDetailCard.tsx`
- CREATE: `mobile/__tests__/components/governance/UsageDetailCard.test.tsx`
- MODIFY: `mobile/app/(tabs)/governance/index.tsx`
- MODIFY: `mobile/__tests__/app/(tabs)/governance/index.test.tsx`

### Wave 2
- CREATE: `mobile/app/(tabs)/governance/audit.tsx`
- CREATE: `mobile/src/components/governance/AuditEventCard.tsx`
- CREATE: `mobile/src/components/governance/AuditFilterBar.tsx`
- CREATE: `mobile/__tests__/app/(tabs)/governance/audit.test.tsx`
- CREATE: `mobile/__tests__/components/governance/AuditEventCard.test.tsx`
- CREATE: `mobile/__tests__/components/governance/AuditFilterBar.test.tsx`
- MODIFY: `mobile/src/services/api.types.ts`
- MODIFY: `mobile/src/services/api.secondary.ts`
- MODIFY: `mobile/src/services/api.ts`
- MODIFY: `mobile/src/hooks/useWedge.ts`
- MODIFY: `mobile/__tests__/hooks/useWedge.test.ts`
- MODIFY: `mobile/__tests__/services/api.test.ts`

### Wave 3
- CREATE: `mobile/app/(tabs)/governance/usage.tsx`
- CREATE: `mobile/src/components/governance/UsageCostSummaryCard.tsx`
- CREATE: `mobile/__tests__/app/(tabs)/governance/usage.test.tsx`
- CREATE: `mobile/__tests__/components/governance/UsageCostSummaryCard.test.tsx`
- MODIFY: `mobile/src/services/api.types.ts`
- MODIFY: `mobile/src/services/api.secondary.ts`
- MODIFY: `mobile/src/services/api.ts`
- MODIFY: `mobile/src/hooks/useWedge.ts`
- MODIFY: `mobile/app/(tabs)/governance/index.tsx`
- MODIFY: `mobile/__tests__/hooks/useWedge.test.ts`
- MODIFY: `mobile/app/(tabs)/activity/[id].tsx`

## Implementation Result

Completed on 2026-04-12 with local mobile QA green:

- `bash scripts/check-no-inline-eslint-disable.sh`
- `cd mobile && npm run typecheck`
- `cd mobile && npm run lint`
- `cd mobile && npm run quality:arch`
- `cd mobile && npm run test:coverage`
- `bash scripts/qa-mobile-prepush.sh`

Result: 52 test suites passed, 414 tests passed, mobile coverage threshold satisfied.
