---
doc_type: task
id: remove-e2e-sales-brief-mock
title: Remove E2E Sales Brief Mock from Production Code
status: pending
phase: mobile
tags: [mobile, e2e, mock, cleanup, backend-connectivity]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - mobile/src/services/api.agents.ts
  - mobile/.env
  - mobile/__tests__/services/api.test.ts
  - mobile/.env.example
  - mobile/e2e/wedge-followup.e2e.ts
created: 2026-04-12
completed:
---

## Context

The mobile app's `api.agents.ts` contains a runtime conditional that bypasses the real BFF API for the `getSalesBrief` function when `EXPO_PUBLIC_E2E_MODE === '1'` is set. This violates the project rule that production code must never contain mocks.

Additionally, `mobile/.env` (the default development environment file) incorrectly sets `EXPO_PUBLIC_E2E_MODE=1`, meaning **all local development builds** silently return fake sales brief data instead of hitting the real backend.

The full API chain already works end-to-end: Mobile → `POST /bff/api/v1/copilot/sales-brief` → Go backend. Nothing in the backend needs to change.

---

## Files Affected

| File | Change |
|------|--------|
| `mobile/src/services/api.agents.ts` | Delete lines 7, 9-84, 177-179 (isE2E flag + mock function + if branch) |
| `mobile/.env` | Remove `EXPO_PUBLIC_E2E_MODE=1` from default dev env |
| `mobile/__tests__/services/api.test.ts` | Add new test asserting mock does not intercept when env var is set |
| `mobile/.env.example` | Add documentation comment (optional) |
| `mobile/e2e/wedge-followup.e2e.ts` | Post-deploy validation only — assertions may need update to match real backend outcomes |

**Must NOT be changed:**
- `mobile/app/e2e-bootstrap.tsx` — legitimate E2E auth injection gate
- `mobile/app/_layout.tsx` — uses `isE2E` to disable React Query auto-fetching in Detox (infrastructure, not mock)
- `mobile/.env.e2e` — correctly sets `EXPO_PUBLIC_E2E_MODE=1` for Detox builds

---

## Tasks (ordered by dependency)

### Task 1 — Fix `mobile/.env` (prerequisite)
**File:** `mobile/.env` line 4
**Status:** pending

Remove `EXPO_PUBLIC_E2E_MODE=1`. This flag must not be in the default env. The variable only belongs in `.env.e2e`.

**Before:**
```
# Task 4.8 — E2E test environment
# This file is loaded during Detox E2E builds to disable React Query auto-fetching
EXPO_PUBLIC_BFF_URL=http://10.0.2.2:3000
EXPO_PUBLIC_E2E_MODE=1
```

**After:**
```
EXPO_PUBLIC_BFF_URL=http://10.0.2.2:3000
```

---

### Task 2 — Write failing test (TDD red phase)
**File:** `mobile/__tests__/services/api.test.ts` — append inside `describe('salesBriefApi')` block after line 505
**Status:** pending

Add a test that sets `EXPO_PUBLIC_E2E_MODE=1` at runtime and verifies `apiClient.post` is still called. This test **must fail** before Task 3 is done, because `isE2E` is evaluated at module load time and the cached `true` value causes the mock to return without calling the API.

```typescript
it('always calls POST regardless of EXPO_PUBLIC_E2E_MODE env var', async () => {
  const saved = process.env.EXPO_PUBLIC_E2E_MODE;
  process.env.EXPO_PUBLIC_E2E_MODE = '1';

  jest.resetModules();
  const { salesBriefApi: fresh } =
    require('../../src/services/api.agents') as typeof import('../../src/services/api.agents');

  const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValueOnce({
    data: { outcome: 'completed', summary: 'test' },
  } as never);

  await fresh.getSalesBrief('deal', 'deal-1');

  expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/copilot/sales-brief', {
    entityType: 'deal',
    entityId: 'deal-1',
  });

  process.env.EXPO_PUBLIC_E2E_MODE = saved;
  jest.resetModules();
});
```

---

### Task 3 — Remove mock from `api.agents.ts` (TDD green phase)
**File:** `mobile/src/services/api.agents.ts`
**Status:** pending

1. **Delete line 7:** `const isE2E = process.env.EXPO_PUBLIC_E2E_MODE === '1';`
2. **Delete lines 8-84:** blank line + entire `buildE2ESalesBriefMock` function
3. **Collapse `salesBriefApi`** (lines 174-187) to:

```typescript
// W1-T1: Sales Brief API — dedicated contract for POST /api/v1/copilot/sales-brief
export const salesBriefApi = {
  getSalesBrief: async (entityType: string, entityId: string) => {
    const response = await apiClient.post('/bff/api/v1/copilot/sales-brief', {
      entityType,
      entityId,
    });
    return response.data as SalesBrief;
  },
};
```

The `SalesBrief` import (line 5) is still used — no import changes needed.

---

### Task 4 — Document `.env.example` (optional, no functional impact)
**File:** `mobile/.env.example`
**Status:** pending

Add a commented line documenting the E2E variable so developers know it exists:
```
EXPO_PUBLIC_BFF_URL=http://10.0.2.2:3000
# EXPO_PUBLIC_E2E_MODE=1  # Set only in .env.e2e for Detox E2E builds
```

---

### Task 5 — Post-deploy Detox validation (manual, after commit)
**File:** `mobile/e2e/wedge-followup.e2e.ts` lines 51-70
**Status:** pending

The two Sales Brief E2E tests assert:
- Line 55: `by.text('abstained')` for the seeded account
- Line 64: `by.text('completed')` for the seeded deal

After removing the mock, these assertions must match what the **real Go backend** returns for the seeded entities. Run the Detox suite against a live backend. If outcomes differ from the mock values, update `by.text(...)` assertions accordingly.

No code change needed unless the real backend returns different outcomes.

---

## Dependency Order

```
Task 1 (.env fix)
  └── Task 2 (write failing test — red)
        └── Task 3 (remove mock — green)
              └── Task 4 (.env.example doc — optional)
                    └── Task 5 (Detox E2E validation — post-deploy)
```

---

## Verification

1. `cd mobile && npx jest --testPathPattern=api.test.ts` — all salesBriefApi tests pass
2. `cd mobile && npx jest` — full unit test suite passes
3. Start backend + BFF locally, run app in dev mode, navigate to Sales Brief screen → data loads from real Go backend
4. `cd mobile && npx detox test` against live backend — Task 5 validation
