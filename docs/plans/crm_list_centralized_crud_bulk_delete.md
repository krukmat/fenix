---
doc_type: task
id: CRM-LIST-CENTRALIZED-CRUD-BULK-DELETE
title: "CRM List Centralized CRUD and Bulk Delete"
status: completed
phase: mobile-crm
week: ~
tags: [mobile, crm, lists, crud, bulk-delete, ux]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - mobile/src/components/crm/CRMListScreen.tsx
  - mobile/src/components/crm/CRMListSelection.tsx
  - mobile/src/components/crm/CoreCRMListViews.tsx
  - mobile/src/components/crm/CoreCRMDetailViews.tsx
  - mobile/__tests__/app/(tabs)/crm/read-only.test.tsx
created: 2026-04-20
completed: 2026-04-20
---

# CRM List Centralized CRUD and Bulk Delete

## Context

Core CRM entity actions need to be centralized in the entity list screens for accounts,
contacts, leads, deals, and cases.

Current state:
- Create already exists from list screens through `New X` primary actions.
- Edit exists from detail screens and must move to list rows.
- Delete exists in API endpoints and mutation hooks, but has no visible UI.
- Detail screens should become read-only views once list actions own create, edit, and delete.

Desired behavior:
- Lists are the single operational surface for create, edit, and delete.
- Detail screens remain available for read-only inspection.
- Delete is only available as a bulk action through explicit multi-selection.

## UX Decisions

- Show a checkbox on every list row at all times.
- Tapping the row body keeps navigating to `/crm/<entity>/<id>`.
- Show an `Edit` action on every list row (pencil icon, no text label).
- Do not show individual row delete actions.
- Show `Delete selected` only when one or more rows are selected.
- Confirm destructive deletion with `Alert.alert`.

## Tasks

### Task 1 - Add Multi-Selection to CRM Lists
**Effort:** Media
**Status:** completed
**Reasoning sugerido al iniciar:** medium

Add always-visible checkboxes to each CRM entity list row. Selection state must be held at the
entity list level and keyed by entity id.

Expected controls:
- Per-row checkbox: `crm-<entity>-item-<index>-select`
- Select all visible rows: `crm-<entity>-select-all`
- Clear selection: `crm-<entity>-clear-selection`

`Select all` applies to the currently visible filtered rows, not all pages on the server.

Progress 2026-04-20:
- Extended `CRMListScreen` with reusable selection props, always-visible row checkboxes,
  `Select all`, `Clear`, and selected-count header controls.
- Added `CRMListSelection.tsx` to keep selection/header UI below lint complexity and file-size
  limits.
- Added entity-list selection state in `CoreCRMListViews`, keyed by CRM entity id.
- Preserved row-body navigation to `/crm/<entity>/<id>`.
- Added focused route test coverage for row checkbox toggling, selected count, clear, and
  select-all behavior.

Files touched:
- `mobile/src/components/crm/CRMListScreen.tsx` — added selection props to `CRMListScreenProps`
- `mobile/src/components/crm/CRMListSelection.tsx` — new file: `ListHeader`, `SelectableItem`
- `mobile/src/components/crm/CoreCRMListViews.tsx` — added `useSelectionState` hook, wired props
- `mobile/__tests__/app/(tabs)/crm/read-only.test.tsx` — added selection behavior tests

### Task 2 - Centralize Create and Edit in List Screens
**Effort:** Media
**Status:** completed
**Reasoning sugerido al iniciar:** medium

Keep the existing `New X` list header action for creation. Add an `Edit` action per row that
navigates to `/crm/<entity>/edit/<id>`.

The row body must continue navigating to the read-only detail route.

Expected edit testIDs:
- `crm-accounts-item-0-edit`
- `crm-contacts-item-0-edit`
- `crm-leads-item-0-edit`
- `crm-deals-item-0-edit`
- `crm-cases-item-0-edit`

Progress 2026-04-20:
- Added `onEdit?: (id: string) => void` to `SelectableItem` in `CRMListSelection.tsx`.
- Added `onRowEdit?: (id: string) => void` to `CRMListScreenProps` and threaded through
  `renderListItem` → `SelectableItem`.
- Each `CoreCRMXxxList` passes `onRowEdit={(id) => router.push('/crm/<entity>/edit/<id>')}`.
- 5 new TDD tests, all passing.

Files touched:
- `mobile/src/components/crm/CRMListSelection.tsx` — `SelectableItem`: added `onEdit` prop + Edit button
- `mobile/src/components/crm/CRMListScreen.tsx` — added `onRowEdit` to props + `renderListItem`
- `mobile/src/components/crm/CoreCRMListViews.tsx` — `onRowEdit` wired per entity
- `mobile/__tests__/app/(tabs)/crm/read-only.test.tsx` — 5 edit navigation tests

### Task 3 - Implement Bulk Delete with Existing Hooks
**Effort:** Media
**Status:** completed
**Reasoning sugerido al iniciar:** medium

Add `Delete selected` to each entity list when the selected id set is not empty.

Medium-scope boundary:
- Reuse the existing per-entity delete hooks only; do not design or add bulk-delete API endpoints.
- Keep deletion orchestration local to the list layer with a small reusable hook.
- Use `Promise.allSettled` only to preserve failed ids for retry; do not add retry queues,
  optimistic removal, background jobs, or cross-page selection.
- Validate behavior through focused route/component tests rather than expanding the visual
  screenshot suite in this task.

Use existing mutations:
- `useDeleteAccount`
- `useDeleteContact`
- `useDeleteLead`
- `useDeleteDeal`
- `useDeleteCase`

Deletion flow:
1. User selects one or more rows.
2. User taps `Delete selected`.
3. UI opens `Alert.alert` with the selected count.
4. Confirm runs deletes with `Promise.allSettled`.
5. If all deletes succeed, clear selection and rely on existing query invalidation.
6. If some deletes fail, show a failure alert and keep only failed ids selected.

Disable checkboxes, `Select all`, `Clear`, `Edit`, and `Delete selected` while bulk delete is
pending.

Progress 2026-04-20:
- Added `onBulkDelete` + `bulkDeletePending` props to `CRMListScreenProps` and `ListHeaderProps`.
- Added `Delete selected` button in `SelectionActions` (visible only when `selectedCount > 0`).
- Extracted `useSelectionState` and `useBulkDelete` hooks in `CoreCRMListViews` to satisfy
  `max-lines-per-function: 80` lint gate.
- `useBulkDelete` implements `Alert.alert` + `Promise.allSettled` + partial failure handling.
- Each `CoreCRMXxxList` calls its `useDeleteXxx` hook and passes `deleteFn` to `EntityListFrame`.
- 4 new TDD tests, all passing. 514/514 total.

Files touched:
- `mobile/src/components/crm/CRMListSelection.tsx` — `SelectionActions`: added Delete selected button + `bulkDeletePending` disable
- `mobile/src/components/crm/CRMListScreen.tsx` — added `onBulkDelete`/`bulkDeletePending` to props + `ListContentProps` + `ListHeaderComponent`
- `mobile/src/components/crm/CoreCRMListViews.tsx` — extracted `useSelectionState`, `useBulkDelete`; wired `deleteFn` per entity
- `mobile/__tests__/app/(tabs)/crm/read-only.test.tsx` — 4 bulk delete tests

### Task 4 - Extend Reusable CRMListScreen Support
**Effort:** Media
**Status:** completed
**Reasoning sugerido al iniciar:** medium

Extend `CRMListScreen` without coupling it to CRM entity types. The reusable list component should
support:
- item-level selection affordances
- item-level secondary actions
- bulk action controls in the list header

Existing list behavior must remain unchanged:
- loading
- error and retry
- empty state
- search
- refresh
- pagination
- primary header action

Progress 2026-04-20:
- No new code required. All affordances were built generically in Tasks 1–3.
- `CRMListItem` interface requires only `id: string`.
- All action callbacks are optional — the component degrades gracefully without them.
- Verified against spec checklist: all 10 behaviors confirmed present.

### Task 5 - Make Detail Screens Read-Only and Update Documentation
**Effort:** Baja
**Status:** completed
**Reasoning sugerido al iniciar:** low

Remove `Edit X` primary actions from core CRM detail screens while keeping edit routes intact for
list row navigation.

Progress 2026-04-20:
- Removed `primaryActionLabel` and `onPrimaryAction` from all 5 `CRMDetailShell` calls in
  `CoreCRMDetailViews.tsx` (account, contact, lead, deal, case).
- Removed orphaned `const router = useRouter()` declarations from the 5 affected components.
- Updated 5 existing tests to assert `queryByTestId('crm-*-detail-primary-action')` is null.
- 514/514 tests passing, QA gate verde.

Files touched:
- `mobile/src/components/crm/CoreCRMDetailViews.tsx` — removed Edit primary actions + orphaned router refs
- `mobile/__tests__/app/(tabs)/crm/read-only.test.tsx` — 5 detail read-only assertions

### Task 6 - Polish List Row UX (Edit icon + Delete selected visibility)
**Effort:** Baja
**Status:** completed
**Reasoning sugerido al iniciar:** medium

Post-screenshot review identified two UX issues:

1. **Edit button** — rendered as a large bordered text button. Replace with a vector pencil
   icon using a compact `TouchableOpacity`, no border, primary color only.
2. **Checkbox** — empty square looks broken. Use a `✓` checkmark with solid fill when selected,
   visible border always.
3. **Delete selected screenshot** — correct behavior (hidden with 0 selected) but not validated
   visually in the screenshot suite. In `mobile/maestro/authenticated-audit.yaml`, after
   `20_crm_accounts_list`, tap `crm-accounts-item-0-select`, assert
   `crm-accounts-delete-selected` is visible, and capture
   `20b_crm_accounts_list_selected`.

Expected testIDs unchanged — only visual treatment changes.

Visual evidence contract:
- `20_crm_accounts_list` shows the default list state: row checkbox and compact edit pencil.
- `20b_crm_accounts_list_selected` shows the selected-row state: solid checkbox with `✓`,
  `Delete selected`, and the compact edit pencil.
- No per-row delete icon is expected by design; destructive delete remains bulk-only.

Progress 2026-04-20:
- Replaced the bordered `Edit` text button with a compact `MaterialCommunityIcons`
  `pencil-outline` button while preserving `crm-<entity>-item-<index>-edit`.
- Replaced selected checkbox `x` with `✓`, solid selected fill, and a consistently visible border.
- Extended `mobile/maestro/authenticated-audit.yaml` to select the first account row, assert
  `crm-accounts-delete-selected`, and capture `20b_crm_accounts_list_selected`.

Files touched:
- `mobile/src/components/crm/CRMListSelection.tsx` — checkbox and edit-button visual polish
- `mobile/maestro/authenticated-audit.yaml` — selected-row screenshot coverage

## Test Plan

All tests implemented as TDD (red → green per task). Final suite: 514 passing.

Covered behaviors:
- `New X` navigates to `/crm/<entity>/new` ✅
- Row body navigates to `/crm/<entity>/<id>` ✅
- Row `Edit` navigates to `/crm/<entity>/edit/<id>` ✅
- Checkbox selects and deselects a row ✅
- `Select all` selects visible filtered rows ✅
- `Clear` clears selection ✅
- `Delete selected` hidden with no selection ✅
- `Delete selected` opens confirmation with selected count ✅
- Cancel does not call delete mutations ✅
- Confirm calls the correct delete mutation for every selected id ✅
- Detail screens no longer expose `crm-*-detail-primary-action` ✅

Required local gate for mobile changes:

```bash
bash scripts/qa-mobile-prepush.sh
```

## Assumptions

- No bulk delete endpoints will be added in this iteration.
- Bulk deletion reuses existing per-entity delete hooks.
- Partial failures keep failed ids selected for retry.
- This plan does not replace `docs/plans/crm_form_shared_layout_refactor.md`.
- `docs/plans/` is currently ignored by Git; this document is local unless ignore rules are changed.

## Open Items

- No open implementation items for this plan.
- Follow-up visual coverage is tracked separately in `docs/plans/fr304_screenshot_coverage_gaps.md`.
