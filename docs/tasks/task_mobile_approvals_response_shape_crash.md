---
doc_type: task
id: EXTVAL-BUG-MOBILE-APPROVALS-SHAPE-001
title: "Bug (P0): mobile Home tab hard-crashes on login — approvals response shape/casing mismatch"
status: ready
phase: external-validation-follow-up
week: "2026-W27"
tags: [bugfix, p0, mobile, approvals, home, api-contract, governance]
fr_refs: [FR-071, FR-232, FR-300]
uc_refs: [UC-A5, UC-A6]
blocked_by: []
blocks: [EXTVAL-BATTERY-T7-001]
files_affected:
  - mobile/src/services/api.secondary.ts
  - mobile/src/services/api.types.ts
  - mobile/src/components/approvals/ApprovalCard.tsx
  - mobile/app/(tabs)/home/index.tsx
  - mobile/__tests__/services/api.test.ts
  - mobile/__tests__/components/approvals/ApprovalCard.test.tsx
created: 2026-07-02
completed:
blocked_reason:
---

# Task EXTVAL-BUG-MOBILE-APPROVALS-SHAPE-001

**Plan**: [External Validation First Test Battery Plan](../plans/external_validation_first_test_battery_plan.md#t7-mobile-real-mode-navigation)

## Task Card

Task: EXTVAL-BUG-MOBILE-APPROVALS-SHAPE-001

Task file: docs/tasks/task_mobile_approvals_response_shape_crash.md

Plan file: docs/plans/external_validation_first_test_battery_plan.md

Summary: `approvalApi.getPendingApprovals` (`mobile/src/services/api.secondary.ts:61-66`) casts the raw BFF response body directly to `ApprovalRequest[]` via `return response.data as ApprovalRequest[]`, but the real backend/BFF response is `{"data": [...], "meta": {"total": N}}` (`internal/api/handlers/approval.go:70`), not a bare array. `mobile/app/(tabs)/home/index.tsx:20` (`const approvals = approvalsData ?? []`) only guards `null`/`undefined`, not a truthy non-array object, so the wrapper object reaches `HomeFeed` as `approvals` and crashes at `mobile/src/components/home/HomeFeed.tsx:33` (`approvals.map is not a function`) on every login for any workspace — reproduced live during `EXTVAL-BATTERY-T7-001` via real UI navigation (not a mock/fixture path). This blocks the Home tab, the app's post-login landing screen, for 100% of real users. A second, related contract mismatch exists in the same code path: the backend's `approvalResponse` struct emits camelCase (`resourceType`, `resourceId`, `workspaceId`, `requestedBy`, `approverId`), but the mobile `ApprovalRequest` type (`mobile/src/services/api.types.ts:103-115`) declares snake_case for most of those fields, and `ApprovalCard.tsx:101,104` reads `approval.resource_type`/`approval.resource_id` — which will always be `undefined` even after the array-unwrap fix, silently dropping that metadata line from the approval card UI.

Code affected: `mobile/src/services/api.secondary.ts` (fix `getPendingApprovals` unwrap, following the existing `normalizeSignalsResponse` pattern already used by `signalApi.getSignals` in the same file), `mobile/src/services/api.types.ts` (`ApprovalRequest` field casing), `mobile/src/components/approvals/ApprovalCard.tsx` (align field reads with corrected type), `mobile/app/(tabs)/home/index.tsx` (optional defense-in-depth `Array.isArray` guard, matching the pattern already used two lines below for `pendingApprovalCount`). Test files: `mobile/__tests__/services/api.test.ts` (existing `getPendingApprovals` test mocks `{ data: [] }` as a bare array response — the wrong contract shape, which is why this bug shipped undetected; must be corrected to the real `{data: {data: [...], meta: {...}}}` envelope), `mobile/__tests__/components/approvals/ApprovalCard.test.tsx` (add camelCase-field fixture coverage).

Effort/reasoning: RRI 35 (Moderate band — `python3 scripts/rri.py --auto-cc --touches mobile/src/services/api.secondary.ts --touches mobile/src/services/api.types.ts --touches mobile/src/components/approvals/ApprovalCard.tsx --touches mobile/__tests__/services/api.test.ts --touches mobile/__tests__/components/approvals/ApprovalCard.test.tsx --T 2 --A 1 --X 1 --D 2 --K 1 --P 3` → Final RRI 35). Localized to one API module, one type, one component, and their tests, but P (impact) is elevated because this is a crash on the primary post-login screen with a single call site that is easy to fix but currently blocks all real mobile usage. Per HITL_AUTONOMY_POLICY Moderate-band gate, this task card requires explicit approval before implementation.

Recommended model: claude-sonnet-4-6

Estimated tokens: ~6000

## System Context

This is a bugfix to existing client-side response-parsing code, not a new component, so a full System Context section (per CLAUDE.md's "new service or component" rule) is not strictly required — included in abbreviated form because the defect spans a client/server contract boundary.

```
Go backend (ApprovalHandler.ListPendingApprovals)
  -> encodes {"data": [...], "meta": {"total": N}} with camelCase fields
       |
       v
BFF (generic proxyRouter passthrough, bff/src/app.ts:90 `/bff/api/v1` -> proxyRouter)
  -> forwards backend body verbatim, no transformation
       |
       v
mobile apiClient.get('/bff/api/v1/approvals')  [api.secondary.ts:61-66]
  -> BUG: `response.data as ApprovalRequest[]` treats the whole envelope as the array
       |
       v
usePendingApprovals() hook [useAgentSpec.ts:108-120]  (React Query, no transform)
       |
       v
HomeScreen [(tabs)/home/index.tsx:20]
  -> `const approvals = approvalsData ?? []` (only catches null/undefined, not wrong-shape object)
       |
       v
HomeFeed.buildFeedItems [HomeFeed.tsx:33]
  -> approvals.map(...) THROWS — envelope object has no .map
```

Upstream trigger: any authenticated Home tab mount/refresh (`useSignals` + `usePendingApprovals` both fire on mount). Downstream consumer: none beyond `HomeFeed`/`ApprovalCard` — `mobile/app/(tabs)/inbox/index.tsx` sorts approvals from a separately-normalized `inboxApi.getInbox` response (`api.handoff.ts` already dual-key-normalizes that path) and does not share this bug. Key invariant for whoever picks this up: `normalizeSignalsResponse` (`api.secondary.ts:16-24`) is the established in-repo idiom for handling a list endpoint that may return either a bare array or a `{data: [...]}` envelope — reuse that pattern rather than inventing a new one, and prefer the dual-key `readString`-style helpers already in `api.handoff.ts` if a general snake/camel normalizer is warranted instead of a straight type-casing fix.

## High-Level Pseudocode

```text
# api.secondary.ts — getPendingApprovals
function normalizeApprovalsResponse(data):
  if data is an array: return data
  if data.data is an array: return data.data
  return []

getPendingApprovals(workspaceId):
  response = GET /bff/api/v1/approvals?workspace_id=workspaceId
  return normalizeApprovalsResponse(response.data)

# api.types.ts — ApprovalRequest
# change resource_type/resource_id/workspace_id/requested_by/approver_id
# to camelCase (resourceType/resourceId/workspaceId/requestedBy/approverId)
# to match the real backend contract (internal/api/handlers/approval.go:26-37)

# ApprovalCard.tsx
# update field reads: approval.resource_type -> approval.resourceType
#                      approval.resource_id  -> approval.resourceId

# home/index.tsx (defense in depth, optional but recommended)
const approvals = Array.isArray(approvalsData) ? approvalsData : []

# api.test.ts — getPendingApprovals test
# change mock from `{ data: [] }` (bare array masquerading as response.data)
# to `{ data: { data: [], meta: { total: 0 } } }` (the real envelope),
# and add a case with 1+ items using camelCase fields to prove unwrap + field mapping

# ApprovalCard.test.tsx
# add a fixture using camelCase resourceType/resourceId and assert they render
```

## Origin / Evidence

Found while executing `EXTVAL-BATTERY-T7-001` (Mobile Real-Mode Navigation). After a genuine `POST /bff/auth/login` through the real login UI (not E2E token injection — confirmed via `audit_event` row, `action=login`, `outcome=success`), the app navigated to the Home tab and immediately crashed with a fatal `TypeError: approvals.map is not a function (it is undefined)`. Correlated to a real, successful BFF round-trip via `audit_event` row `019f21ad-6cd9-70b6-e9db-9e552c78842c` (`action=get_request`, `path=/api/v1/approvals`, `status_code=200`, `duration_ms=4`), and confirmed the real response shape via direct `curl` against the same endpoint/token: `{"data":[],"meta":{"total":0}}`. Cross-referenced against `internal/api/handlers/approval.go:70` (`json.NewEncoder(w).Encode(map[string]any{"data": out, "meta": ...})`) to confirm this envelope is the deliberate, canonical backend contract — not a bug on the backend side. Full detail in `docs/tasks/task_extval_battery_t7_mobile_real_mode.md` (Finding 1).

## Risk

- The mobile app is currently unusable past login for any workspace with the typical (non-empty or empty, both reproduce) approvals state — this is not an edge case, it fires on the default landing screen for every real user.
- This was invisible to `EXTVAL-BATTERY-T3-001`/`T4-001`/`T5-001` because those validated the API layer directly (curl/API calls), never through the real mobile UI — `T7` is the only battery task that exercises this path, and it was blocked from completing its remaining acceptance criteria (Support/Inbox/Activity/Sales Brief/Governance navigation, support-agent trigger) as a direct result.
- The existing unit test for `getPendingApprovals` (`mobile/__tests__/services/api.test.ts:414-422`) mocks `apiClient.get` to resolve `{ data: [] }` — i.e., it hands the code a bare array, the wrong contract — so the test suite currently "proves" the buggy unwrap correct. Fixing the implementation without fixing this mock would either fail the existing test (revealing the bug, good) or, if the test is naively "fixed" to match new buggy behavior instead of the real contract, could mask a regression. The fix must anchor the mock to the real `{data: {data: [...], meta: {...}}}` shape confirmed by direct backend inspection, not to whatever the implementation currently returns.
- Field-casing mismatch (`resource_type`/`resource_id` vs. real `resourceType`/`resourceId`) is lower severity (silently missing UI text, not a crash, and guarded by an `if` so it doesn't throw) but is in the same file/type and should be fixed in the same pass to avoid re-touching this area twice for what is fundamentally one client/server contract drift.

## Scope Note

This task does not re-attempt the blocked portions of `EXTVAL-BATTERY-T7-001` (Support/Inbox/Activity/Sales Brief/Governance navigation, support-agent trigger, BFF-driven state refresh observation). Once this fix lands, `EXTVAL-BATTERY-T7-001`'s deferred acceptance criteria 4 and 5 should be re-validated in a follow-up battery re-run — the Android emulator (`fenix_t7`), native dev build toolchain, and a seeded high-priority case (`019f21ab-1784-73be-b2ef-bee868add036`, "ExtVal T7 - login not working after password reset") are already provisioned and available for that retest from this session.

## Acceptance Criteria

1. `getPendingApprovals` correctly unwraps `{data: [...], meta: {...}}` into a bare `ApprovalRequest[]`, following the `normalizeSignalsResponse` pattern.
2. `ApprovalRequest` type and `ApprovalCard` field reads use the real backend camelCase field names (`resourceType`, `resourceId`, and the other renamed fields), not snake_case.
3. `mobile/__tests__/services/api.test.ts`'s `getPendingApprovals` test mocks the real `{data: {data: [...], meta: {...}}}` envelope (not a bare array) and covers both empty and non-empty cases.
4. `mobile/__tests__/components/approvals/ApprovalCard.test.tsx` covers rendering with camelCase `resourceType`/`resourceId` fixtures.
5. `npm run typecheck`, `npm run lint`, and `npm run test:coverage` (or `test`) pass in `mobile/`.
6. Manual re-verification: real login on the `fenix_t7` emulator (or equivalent) reaches the Home tab without the `approvals.map is not a function` crash.
