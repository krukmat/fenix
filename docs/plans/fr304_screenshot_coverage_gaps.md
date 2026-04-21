---
doc_type: task
id: FR304-SCREENSHOT-GAPS
title: "FR-304 Screenshot Coverage Gaps — Maestro Suite"
status: planned
phase: mobile-crm
week: ~
tags: [mobile, maestro, screenshots, crm, fr-304]
fr_refs: [FR-304]
uc_refs: [UC-P2]
blocked_by: []
blocks: []
files_affected:
  - mobile/maestro/authenticated-audit.yaml
created: 2026-04-21
completed: null
---

# FR-304 Screenshot Coverage Gaps — Maestro Suite

## Context

FR-304 (CRM List Centralized CRUD and Bulk Delete) was shipped with Maestro screenshot coverage
for accounts only (`20_crm_accounts_list`, `20b_crm_accounts_list_selected`). Three visual
behaviors remain unvalidated in the screenshot suite:

1. **List selection state for contacts, leads, deals, cases** — 4 entities missing
2. **Edit form opened from a list row edit button** — no form-from-list screenshot exists
3. **Bulk delete flow** — no screenshot captures the post-confirm state

Implementation changes are YAML-only in a single file. No mobile source code or Jest tests need to
change, but this still touches `mobile/`, so the mobile pre-push QA gate remains required before
push.

---

## File to Modify

`mobile/maestro/authenticated-audit.yaml`
**Insertion point:** after line 270 (`path: "20b_crm_accounts_list_selected"`), before line 272
(`- openLink: "fenixcrm:///crm/accounts/${SEED_ACCOUNT_ID}"`).

---

## Consistency Review

Status after review on 2026-04-21:

- Existing `authenticated-audit.yaml` already captures account default and selected states as
  `20_crm_accounts_list` and `20b_crm_accounts_list_selected`.
- Referenced list, selection, delete, edit, and account form testIDs exist in mobile source/tests.
- The insertion point is safe because account detail screenshot `21` uses a direct deep link with
  `${SEED_ACCOUNT_ID}` and does not depend on list row state.
- Contact deletion is safer than account deletion because account detail validation still follows
  this block.
- The only process gap in the original plan was verification: `npm run screenshots` is necessary
  for visual evidence, but `bash scripts/qa-mobile-prepush.sh` is the required local gate for any
  `mobile/` change under `AGENTS.md`.

---

## Execution Plan

### Task 1 - Add Missing Entity Selection Screenshots
**Effort:** Baja
**Reasoning sugerido al iniciar:** medium
**Files:** `mobile/maestro/authenticated-audit.yaml`

Add contacts, leads, deals, and cases selected-state coverage immediately after
`20b_crm_accounts_list_selected`.

Acceptance:
- `26_crm_contacts_list` and `26b_crm_contacts_list_selected` are captured.
- `27_crm_leads_list` and `27b_crm_leads_list_selected` are captured.
- `28_crm_deals_list` and `28b_crm_deals_list_selected` are captured.
- `29_crm_cases_list_selected` is captured.
- Each selected-state screenshot shows the selected checkbox and `Delete selected`.

### Task 2 - Add Row Edit Form Screenshot
**Effort:** Baja
**Reasoning sugerido al iniciar:** medium
**Files:** `mobile/maestro/authenticated-audit.yaml`

Open the account edit form from `crm-accounts-item-0-edit` and capture the populated form.

Acceptance:
- `30_crm_accounts_edit_form` is captured.
- The form is opened via the list-row edit button, not by direct edit deep link.
- The flow returns to `crm-accounts-list` before continuing.

### Task 3 - Add Bulk Delete Visual Flow
**Effort:** Baja
**Reasoning sugerido al iniciar:** medium
**Files:** `mobile/maestro/authenticated-audit.yaml`

Select the first contact, open the `Delete selected` confirmation, confirm deletion, and capture
the resulting contacts list.

Acceptance:
- Native dialog title `Delete selected` is visible before confirmation.
- `31_crm_contacts_bulk_delete_confirm` captures the native confirmation dialog before deletion.
- `32_crm_contacts_after_bulk_delete` is captured after `crm-contacts-empty` appears.
- Account screenshots `21-25` still run after the inserted block.

### Task 4 - Verify Locally
**Effort:** Media
**Reasoning sugerido al iniciar:** medium
**Files:** screenshot artifacts only

Run visual verification first, then the required mobile gate.

Commands:

```bash
npm --prefix mobile run screenshots
bash scripts/qa-mobile-prepush.sh
```

Acceptance:
- 10 new screenshots exist in `mobile/artifacts/screenshots/`.
- Existing screenshots `21_crm_account_detail` through `25_crm_cases_mutation_verified` still
  generate successfully.
- `bash scripts/qa-mobile-prepush.sh` passes before any push.

---

## Steps to Add

Insert the following YAML block after line 270:

```yaml
# ── 10b. CRM list selected states — contacts, leads, deals, cases ─────────────

- openLink: "fenixcrm:///crm/contacts"
- extendedWaitUntil:
    visible:
      id: "crm-contacts-list"
    timeout: 20000
- assertVisible:
    id: "crm-contacts-search"
- takeScreenshot:
    path: "26_crm_contacts_list"
- tapOn:
    id: "crm-contacts-item-0-select"
- assertVisible:
    id: "crm-contacts-delete-selected"
- takeScreenshot:
    path: "26b_crm_contacts_list_selected"

- openLink: "fenixcrm:///crm/leads"
- extendedWaitUntil:
    visible:
      id: "crm-leads-list"
    timeout: 20000
- assertVisible:
    id: "crm-leads-search"
- takeScreenshot:
    path: "27_crm_leads_list"
- tapOn:
    id: "crm-leads-item-0-select"
- assertVisible:
    id: "crm-leads-delete-selected"
- takeScreenshot:
    path: "27b_crm_leads_list_selected"

- openLink: "fenixcrm:///crm/deals"
- extendedWaitUntil:
    visible:
      id: "crm-deals-list"
    timeout: 20000
- assertVisible:
    id: "crm-deals-search"
- takeScreenshot:
    path: "28_crm_deals_list"
- tapOn:
    id: "crm-deals-item-0-select"
- assertVisible:
    id: "crm-deals-delete-selected"
- takeScreenshot:
    path: "28b_crm_deals_list_selected"

- openLink: "fenixcrm:///crm/cases"
- extendedWaitUntil:
    visible:
      id: "crm-cases-list"
    timeout: 20000
- tapOn:
    id: "crm-cases-item-0-select"
- assertVisible:
    id: "crm-cases-delete-selected"
- takeScreenshot:
    path: "29_crm_cases_list_selected"

# ── 10c. Edit form opened from list row edit button (accounts) ────────────────

- openLink: "fenixcrm:///crm/accounts"
- extendedWaitUntil:
    visible:
      id: "crm-accounts-list"
    timeout: 20000
- tapOn:
    id: "crm-accounts-item-0-edit"
- extendedWaitUntil:
    visible:
      id: "crm-account-form-screen"
    timeout: 20000
- takeScreenshot:
    path: "30_crm_accounts_edit_form"
- back
- extendedWaitUntil:
    visible:
      id: "crm-accounts-list"
    timeout: 10000

# ── 10d. Bulk delete flow (contacts) ──────────────────────────────────────────

- openLink: "fenixcrm:///crm/contacts"
- extendedWaitUntil:
    visible:
      id: "crm-contacts-list"
    timeout: 20000
- tapOn:
    id: "crm-contacts-item-0-select"
- assertVisible:
    id: "crm-contacts-delete-selected"
- tapOn:
    id: "crm-contacts-delete-selected"
- assertVisible:
    text: "Delete selected"
- takeScreenshot:
    path: "31_crm_contacts_bulk_delete_confirm"
- tapOn:
    text: "Delete"
- extendedWaitUntil:
    visible:
      id: "crm-contacts-empty"
    timeout: 30000
- takeScreenshot:
    path: "32_crm_contacts_after_bulk_delete"
```

---

## Key Design Decisions

- **Entity for bulk delete:** contacts (not accounts) — accounts detail screenshot `21` comes after
  this block and depends on the account record existing. Deleting a contact is safe.
- **Cases selection:** no `assertVisible: crm-cases-search` guard — cases list was already
  validated in `crm-mutation-case.yaml`; omitting it keeps parity with that flow's pattern.
- **Alert interaction:** `assertVisible: text: "Delete selected"` confirms the native dialog is
  open before tapping `text: "Delete"`. Alert title comes from `CoreCRMListViews.tsx` line 96.
- **Back navigation after edit form:** `back` + `extendedWaitUntil crm-accounts-list` guards
  against animation delay before next step.
- **Selection state resets on remount:** `useState<Set<string>>` in `EntityListFrame` clears
  when navigating away via `openLink`, so no state bleeds between flows.

---

## Screenshot Output (10 new files)

| File | Description |
|------|-------------|
| `26_crm_contacts_list.png` | Contacts list default state |
| `26b_crm_contacts_list_selected.png` | Contacts list with 1 selected + Delete selected visible |
| `27_crm_leads_list.png` | Leads list default state |
| `27b_crm_leads_list_selected.png` | Leads list with 1 selected + Delete selected visible |
| `28_crm_deals_list.png` | Deals list default state |
| `28b_crm_deals_list_selected.png` | Deals list with 1 selected + Delete selected visible |
| `29_crm_cases_list_selected.png` | Cases list with 1 selected + Delete selected visible |
| `30_crm_accounts_edit_form.png` | Account edit form opened from list row edit button |
| `31_crm_contacts_bulk_delete_confirm.png` | Native confirmation dialog for bulk delete |
| `32_crm_contacts_after_bulk_delete.png` | Contacts list after bulk delete confirmed (empty state) |

---

## Risk Notes

- **Android Alert API 33+:** Native `AlertDialog` buttons are resolved by text in Maestro's
  accessibility tree. If `tapOn: { text: "Delete" }` fails, check the OS view hierarchy dump
  for exact button label casing.
- **Post-delete empty state:** After deleting the 1 seeded contact, `32_crm_contacts_after_bulk_delete`
  will show the empty-list state. This is intentional.
- **No regression risk:** The accounts detail flow uses `${SEED_ACCOUNT_ID}`
  (direct deep link), not item-0 from the list, so the account record remains intact.

---

## Verification

1. Run `npm --prefix mobile run screenshots` and confirm 10 new `.png` files in `mobile/artifacts/screenshots/`
2. Inspect `*_selected` screenshots: filled ✓ checkbox + red "Delete selected" button visible
3. Inspect `30_crm_accounts_edit_form`: form fields populated (not empty), confirming edit mode loaded
4. Inspect `31_crm_contacts_bulk_delete_confirm`: native dialog shows the final `Delete` action
5. Inspect `32_crm_contacts_after_bulk_delete`: empty state or reduced list, confirming delete executed
6. Confirm `21_crm_account_detail` and `22–25` screenshots unchanged (regression check)
7. Run `bash scripts/qa-mobile-prepush.sh` from the repo root before any push
