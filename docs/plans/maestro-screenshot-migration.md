---
name: Maestro Screenshot Migration
status: superseded
owner: mobile
supersedes: docs/plans/mobile-screenshot-audit.md, docs/plans/mobile-detox-activity-launch-timeout-fix.md
created: 2026-04-06
---

# Migrate Screenshot Suite from Detox to Maestro

## Status Update

This migration is complete, but the original single-flow design is no longer
the current runner. The canonical screenshot flow is now:

- `mobile/maestro/auth-surface.yaml`
- `mobile/maestro/authenticated-audit.yaml`
- `mobile/maestro/seed-and-run.sh`

`mobile/maestro/visual-audit.yaml` has been retired. Final PNGs are written to
`mobile/artifacts/screenshots/` and Maestro reports to
`mobile/artifacts/maestro-reports/`.

## Problem

The Detox-based screenshot suite (`mobile/e2e/screenshots.e2e.ts`) is blocked by a
hardcoded 45-second timeout in AndroidX `MonitoringInstrumentation.startActivitySync()`.
Multiple workarounds were attempted (pre-warm globalSetup, `android.attached` config,
`behavior.launchApp: 'manual'`, Kotlin patching). None produced a reliable, unattended
screenshot run. The root cause is structural: Detox instruments via AndroidX test
infrastructure, which imposes a non-configurable activity launch timeout.

## Solution

Replace the Detox screenshot suite with **Maestro**, a mobile UI testing framework that
drives the app via ADB + accessibility services — no AndroidX instrumentation, no 45s
timeout, no test APK required.

### Why Maestro

| Concern | Detox (current) | Maestro (proposed) |
|---------|-----------------|-------------------|
| Activity launch timeout | 45s hardcoded, cannot configure | N/A — no instrumentation |
| Test APK required | Yes (assembleAndroidTest) | No — uses debug APK directly |
| testID support | Native | Via `resource-id` (RN testID maps automatically) |
| iOS reuse | Separate Detox config | Same YAML flow, zero changes |
| CI headless | Requires workarounds | Native `maestro test` on emulator |
| Screenshot command | `device.takeScreenshot(name)` | `- takeScreenshot: name` |
| Lines of test code | 365 (TS) + 134 (helpers) + 50 (global-setup) | ~160 (YAML) |
| Dependencies added | 0 (already installed but broken) | 1 CLI tool (`maestro`) |

### What stays the same

- All 26 screens covered (same test IDs)
- Seed data via `go run ./scripts/e2e_seed_mobile_p2.go` (called from a shell pre-step)
- Backend + BFF must be running on :8080 / :3000
- Output: PNG files under `mobile/artifacts/screenshots/`
- Existing Detox E2E tests (`e2e:test`, `test:bdd`) are NOT touched

### What changes

- Screenshot suite moves from `screenshots.e2e.ts` (Detox/Jest) to `maestro/visual-audit.yaml`
- `npm run screenshots` calls `maestro test` instead of `detox test`
- Detox screenshot-specific config (`android.emu.debug.screenshots`) becomes unused
- Several Detox workaround files are deleted

### Current repo state to preserve while migrating

- `mobile/package.json` still points `screenshots` to
  `detox test --configuration android.emu.debug.screenshots`
- `mobile/.detoxrc.js` still defines `android.emu.debug.screenshots` with
  Detox artifacts rooted at `mobile/artifacts/screenshots/`
- The current Detox suite relies on:
  - seeded credentials from `scripts/e2e_seed_mobile_p2.go`
  - list search for Contacts and Cases
  - scroll-to-visible for Deals and Workflows
  - BFF lookup of an active signal ID for screenshot `04_home_signal_detail`
- Earlier handoff notes mention `mobile/scripts/run-screenshots.sh` and
  `mobile/scripts/run-screenshots.mjs`, but those files are not present in the
  repo as of 2026-04-06. The Maestro migration must not depend on them.

---

## Prerequisites

Before any task, verify:

```bash
# 1. Maestro installed
maestro --version
# If missing: curl -Ls "https://get.maestro.mobile.dev" | bash

# 2. Emulator running
adb devices  # must show emulator-XXXXX device

# 3. App installed
adb shell pm list packages | grep fenixcrm  # must show com.fenixcrm.app

# 4. Backend + BFF running
curl -s http://localhost:8080/health  # 200
curl -s http://localhost:3000/bff/health  # 200

# 5. Seed data
cd /Users/matiasleandrokruk/Documents/FenixCRM
go run ./scripts/e2e_seed_mobile_p2.go
```

---

## Task Breakdown

### Task 1 — Install Maestro CLI

**Action:** Install Maestro on the dev machine.

**Commands:**
```bash
curl -Ls "https://get.maestro.mobile.dev" | bash
maestro --version
```

**Verification:** `maestro --version` returns a version number.

**Files affected:** None (system-level install).

---

### Task 2 — Create seed runner script

**Action:** Create a shell script that runs the Go seeder and exports the seed data
as environment variables for Maestro.

**File to create:** `mobile/maestro/seed-and-run.sh`

**Logic:**
1. Be callable from `cd mobile && npm run screenshots`, but internally resolve the
   repo root before running any Go command
2. Run `go run ./scripts/e2e_seed_mobile_p2.go` from repo root
3. Parse the JSON output to extract IDs (credentials, account, contact, deal, case,
   workflows, agentRuns). Prefer Node for parsing because it already exists in the
   repo toolchain; use `jq` only if already available.
4. Export at least: `SEED_EMAIL`, `SEED_PASSWORD`, `SEED_ACCOUNT_ID`,
   `SEED_CONTACT_ID`, `SEED_CONTACT_EMAIL`, `SEED_DEAL_ID`, `SEED_CASE_ID`,
   `SEED_CASE_SUBJECT`, `SEED_WORKFLOW_ACTIVE_ID`, `SEED_AGENT_RUN_REJECTED_ID`
5. Resolve an optional active signal for screenshot `04` by logging into the BFF
   with the seeded credentials and querying active signals. Export `SEED_SIGNAL_ID`
   when found.
6. Set up adb reverse ports: `adb reverse tcp:3000 tcp:3000 && adb reverse tcp:8080 tcp:8080`
7. Fail fast with a clear message if the debug app is not installed on the emulator
8. Launch the app: `adb shell am start -n com.fenixcrm.app/.MainActivity`
9. Wait for app-ready UI markers such as `login-screen` or `register-screen`
   instead of relying only on logcat text
10. Run: `maestro test mobile/maestro/visual-audit.yaml`
11. Normalize the generated PNGs into `mobile/artifacts/screenshots/`. Do not
    assume Maestro stores screenshots on-device; use the local output directory
    confirmed during implementation.

**Verification:** Script runs without errors, prints the exported seed values
(redacting secrets if logged) and reports the final screenshot output directory.

**Notes:**
- The script should preserve the single-entry-point UX:
  `cd mobile && npm run screenshots`
- The script handles cold-boot waits that Detox could not (no 45s instrumentation
  limit here).

---

### Task 3 — Create Maestro flow: Auth screens (screenshots 01-02)

**File to create:** `mobile/maestro/visual-audit.yaml`

**Flow outline:**

```yaml
appId: com.fenixcrm.app
name: Visual Audit — FenixCRM Mobile
tags:
  - screenshots
  - visual-audit

---

# Auth screens — app launches to login screen
- assertVisible:
    id: "login-screen"
    timeout: 60000  # cold boot can be slow
- takeScreenshot: "01_auth_login"

- tapOn:
    id: "go-to-register-link"
- assertVisible:
    id: "register-screen"
    timeout: 10000
- takeScreenshot: "02_auth_register"
```

**testIDs used:** `login-screen`, `go-to-register-link`, `register-screen`

**Verification:** `maestro test mobile/maestro/visual-audit.yaml` produces
`01_auth_login.png` and `02_auth_register.png`.

---

### Task 4 — Add login flow + Home screens (screenshots 03-04)

**Append to:** `mobile/maestro/visual-audit.yaml`

**Flow outline:**

```yaml
# Login with seeded credentials
- tapOn:
    id: "go-to-login-link"  # back to login from register
- assertVisible:
    id: "login-screen"
    timeout: 10000
- clearText:
    id: "login-email-input"
- inputText:
    id: "login-email-input"
    text: "${SEED_EMAIL}"
- clearText:
    id: "login-password-input"
- inputText:
    id: "login-password-input"
    text: "${SEED_PASSWORD}"
- tapOn:
    id: "login-submit-button"
- assertVisible:
    id: "home-feed"
    timeout: 60000

# 03 — Home feed
- takeScreenshot: "03_home_feed"

# 04 — Signal detail
- tapOn:
    id: "home-feed-chip-signals"
- runFlow:
    when:
      visible:
        id: "home-feed-signal-${SEED_SIGNAL_ID}"
    commands:
      - tapOn:
          id: "home-feed-signal-${SEED_SIGNAL_ID}"
      - assertVisible:
          id: "signal-detail"
          timeout: 10000
      - takeScreenshot: "04_home_signal_detail"
      - pressKey: "back"
- runFlow:
    when:
      visible:
        id: "home-feed-empty"
    commands:
      - takeScreenshot: "04_home_signal_detail_EMPTY"
```

**testIDs used:** `login-email-input`, `login-password-input`, `login-submit-button`,
`home-feed`, `home-feed-chip-signals`, `home-feed-signal-${SEED_SIGNAL_ID}`,
`home-feed-empty`, `signal-detail`

**Verification:** Screenshots 01-04 generated. If no active signal exists after
seeding, the implementation must either:
- export `SEED_SIGNAL_ID` via the BFF lookup in Task 2, or
- emit the explicit fallback screenshot `04_home_signal_detail_EMPTY`

---

### Task 5 — CRM Hub + Accounts (screenshots 05-08)

**Append to:** `mobile/maestro/visual-audit.yaml`

**Flow outline:**

```yaml
# 05 — CRM Hub (drawer submenu)
- tapOn:
    id: "drawer-open-button"
- assertVisible:
    id: "drawer-content"
    timeout: 5000
- tapOn:
    id: "drawer-crm-tab"
- assertVisible:
    id: "drawer-crm-submenu"
    timeout: 5000
- takeScreenshot: "05_crm_hub"

# 06 — Accounts list
- tapOn:
    id: "drawer-crm-accounts"
- assertVisible:
    id: "accounts-list"
    timeout: 10000
- takeScreenshot: "06_crm_accounts_list"

# 07 — Account detail
- tapOn:
    id: "accounts-list-item-0"
- assertVisible:
    id: "account-detail-screen"
    timeout: 10000
- takeScreenshot: "07_crm_account_detail"
- pressKey: "back"

# 08 — Account new
- tapOn:
    id: "create-account-fab"
- assertVisible:
    id: "account-form-screen"
    timeout: 5000
- takeScreenshot: "08_crm_account_new"
- pressKey: "back"
```

**testIDs used:** `drawer-open-button`, `drawer-content`, `drawer-crm-tab`,
`drawer-crm-submenu`, `drawer-crm-accounts`, `accounts-list`, `accounts-list-item-0`,
`account-detail-screen`, `create-account-fab`, `account-form-screen`

**Verification:** Screenshots 01-08 generated.

---

### Task 6 — Contacts, Deals, Cases (screenshots 09-18)

**Append to:** `mobile/maestro/visual-audit.yaml`

**Flow outline covers:**

| # | Screen | Key testIDs |
|---|--------|-------------|
| 09 | Contacts list | `drawer-crm-contacts`, `contacts-list` |
| 10 | Contact detail | `contacts-search`, `contact-item-${SEED_CONTACT_ID}`, `contact-detail-header` |
| 11 | Deals list | `drawer-crm-deals`, `deals-list` |
| 12 | Deal detail | `deals-flatlist`, `deal-item-${SEED_DEAL_ID}`, `deal-detail-screen` |
| 13 | Deal new | `create-deal-fab`, `deal-new-screen` |
| 14 | Deal edit | `deal-edit-button`, `deal-edit-screen` |
| 15 | Cases list | `drawer-crm-cases`, `cases-list` |
| 16 | Case detail | `cases-search`, `cases-list-item-0`, `case-detail-screen` |
| 17 | Case new | `create-case-fab`, `case-new-screen` |
| 18 | Case edit | `case-edit-button`, `case-edit-screen` |

**Pattern for each section:**
1. Open drawer → tap CRM tab → tap section item
2. Wait for list visible
3. Screenshot list
4. Use the same targeting strategy as the current Detox suite:
   - Contacts: filter via `contacts-search`, then tap `contact-item-${SEED_CONTACT_ID}`
   - Deals: `scrollUntilVisible` / equivalent on `deals-flatlist` until
     `deal-item-${SEED_DEAL_ID}` is visible
   - Cases: filter via `cases-search`, then tap `cases-list-item-0`
5. Screenshot detail
6. Back
7. Tap FAB → screenshot new form → back
8. (If edit exists) tap item → tap edit → screenshot edit → back × 2

**Verification:** Screenshots 01-18 generated.

---

### Task 7 — Copilot, Workflows, Agents, Drawer (screenshots 19-26)

**Append to:** `mobile/maestro/visual-audit.yaml`

**Flow outline covers:**

| # | Screen | Key testIDs |
|---|--------|-------------|
| 19 | Copilot panel | `drawer-copilot-tab`, `copilot-panel` |
| 20 | Workflows list | `drawer-workflows-tab`, `workflows-list` |
| 21 | Workflow new | `workflows-new-btn`, `workflow-new-screen` |
| 22 | Workflow detail | `workflows-flatlist`, `workflow-${SEED_WORKFLOW_ACTIVE_ID}`, `workflow-detail` |
| 23 | Workflow edit | `workflow-edit-btn`, `workflow-edit-screen` |
| 24 | Agents list | `drawer-activity-tab`, `agent-runs-list-screen` |
| 25 | Agent run detail | `agent-run-item-${SEED_AGENT_RUN_REJECTED_ID}`, `agent-run-detail-screen` |
| 26 | Drawer open | `drawer-open-button`, `drawer-content` |

**Verification:** All 26 screenshots generated. Workflow detail/edit should mirror
the current Detox logic by scrolling `workflows-flatlist` until
`workflow-${SEED_WORKFLOW_ACTIVE_ID}` is visible before tapping.

---

### Task 8 — Update npm scripts + detox config cleanup

**File:** `mobile/package.json`

**Changes:**
```diff
- "screenshots": "detox test --configuration android.emu.debug.screenshots",
+ "screenshots": "bash maestro/seed-and-run.sh",
```

**File:** `mobile/.detoxrc.js`

**Changes:**
- Remove `android.emu.debug.screenshots` configuration block (no longer used)
- Keep `android.emu.debug` configuration intact (used by `e2e:test`)
- Do not remove unrelated Detox settings used by `e2e:test`

**Verification:** `npm run screenshots` triggers the Maestro flow, not Detox.

---

### Task 9 — Delete obsolete Detox screenshot files

**Files to delete (with user confirmation):**
- `mobile/e2e/screenshots.e2e.ts` — replaced by `maestro/visual-audit.yaml`
- `mobile/e2e/jest.screenshots.config.ts` — no longer needed
- `mobile/e2e/global-setup-screenshots.ts` — obsolete pre-warm script
- `mobile/e2e/global-setup-screenshots.js` — obsolete compiled pre-warm
- `mobile/e2e/helpers/screenshots.helper.ts` — BFF helper only used by screenshot suite

**Files to keep:**
- `mobile/e2e/helpers/auth.helper.ts` — used by other E2E tests
- `mobile/e2e/helpers/seed.helper.ts` — used by other E2E tests
- `mobile/.detoxrc.js` — still used by `e2e:test`

**Verification:** `npm run e2e:test` still works after deletion. `npm run screenshots`
still works after deletion.

---

### Task 10 — End-to-end validation + commit

**Validation checklist:**
1. `maestro --version` returns valid version
2. `npm run screenshots` completes without errors
3. Expected PNG files exist under `mobile/artifacts/screenshots/`
4. Each PNG shows the expected screen (visual spot-check)
5. `npm run e2e:test` still passes (Detox tests unaffected)
6. Required local mobile QA gates pass:
   - `bash scripts/check-no-inline-eslint-disable.sh`
   - `cd mobile && npm run typecheck`
   - `cd mobile && npm run lint`
   - `cd mobile && npm run quality:arch`
   - `cd mobile && npm run test:coverage`
7. `npm run quality` passes if kept as the convenience aggregate gate

**Commit (single commit per user preference):**
```
feat(mobile): migrate screenshot suite from Detox to Maestro

Detox screenshot suite was blocked by a hardcoded 45s activity launch
timeout in AndroidX MonitoringInstrumentation. Multiple workarounds
failed (pre-warm, android.attached, manual mode, Kotlin patching).

Replaces with Maestro, which drives the app via ADB without
instrumentation. Same 26 screens, same testIDs, same seed data.
Zero changes to the app code.

Supersedes: docs/plans/mobile-screenshot-audit.md
Supersedes: docs/plans/mobile-detox-activity-launch-timeout-fix.md
```

**Files in commit:**
- `mobile/maestro/seed-and-run.sh` (new)
- `mobile/maestro/visual-audit.yaml` (new)
- `mobile/package.json` (updated screenshots script)
- `mobile/.detoxrc.js` (removed screenshots config)
- `docs/plans/maestro-screenshot-migration.md` (this plan)
- Deleted: `screenshots.e2e.ts`, `jest.screenshots.config.ts`,
  `global-setup-screenshots.ts`, `global-setup-screenshots.js`,
  `screenshots.helper.ts`

---

## Confirmed testID → Maestro Mapping

React Native `testID` props are exposed as `resource-id` on Android.
Maestro accesses them via `id: "testID-value"`. No app code changes needed.

Full testID inventory (from `docs/plans/mobile-screenshot-audit.md`):

| testID | Maestro selector | Screen |
|--------|-----------------|--------|
| `login-screen` | `id: "login-screen"` | Login |
| `go-to-register-link` | `id: "go-to-register-link"` | Login → Register |
| `register-screen` | `id: "register-screen"` | Register |
| `login-email-input` | `id: "login-email-input"` | Login form |
| `login-password-input` | `id: "login-password-input"` | Login form |
| `login-submit-button` | `id: "login-submit-button"` | Login form |
| `drawer-open-button` | `id: "drawer-open-button"` | All authenticated screens |
| `drawer-content` | `id: "drawer-content"` | Drawer |
| `drawer-home-tab` | `id: "drawer-home-tab"` | Drawer → Home |
| `drawer-crm-tab` | `id: "drawer-crm-tab"` | Drawer → CRM toggle |
| `drawer-crm-submenu` | `id: "drawer-crm-submenu"` | Drawer → CRM expanded |
| `drawer-crm-accounts` | `id: "drawer-crm-accounts"` | Drawer → Accounts |
| `drawer-crm-contacts` | `id: "drawer-crm-contacts"` | Drawer → Contacts |
| `drawer-crm-deals` | `id: "drawer-crm-deals"` | Drawer → Deals |
| `drawer-crm-cases` | `id: "drawer-crm-cases"` | Drawer → Cases |
| `drawer-copilot-tab` | `id: "drawer-copilot-tab"` | Drawer → Copilot |
| `drawer-workflows-tab` | `id: "drawer-workflows-tab"` | Drawer → Workflows |
| `drawer-activity-tab` | `id: "drawer-activity-tab"` | Drawer → Activity |
| `home-feed` | `id: "home-feed"` | Home screen |
| `home-feed-flatlist` | `id: "home-feed-flatlist"` | Home feed list |
| `home-feed-chip-signals` | `id: "home-feed-chip-signals"` | Home signals filter |
| `home-feed-signal-{id}` | `id: "home-feed-signal-${SEED_SIGNAL_ID}"` | Home signal item |
| `signal-detail` | `id: "signal-detail"` | Signal detail |
| `accounts-list` | `id: "accounts-list"` | Accounts list |
| `accounts-list-item-0` | `id: "accounts-list-item-0"` | First account |
| `account-detail-screen` | `id: "account-detail-screen"` | Account detail |
| `account-form-screen` | `id: "account-form-screen"` | Account form |
| `create-account-fab` | `id: "create-account-fab"` | Accounts FAB |
| `contacts-list` | `id: "contacts-list"` | Contacts list |
| `contact-item-{id}` | `id: "contact-item-${SEED_CONTACT_ID}"` | Contact item |
| `contact-detail-header` | `id: "contact-detail-header"` | Contact detail (`CRMDetailHeader` with `testIDPrefix="contact-detail"`) |
| `contacts-search` | `id: "contacts-search"` | Contacts search input |
| `deals-list` | `id: "deals-list"` | Deals list |
| `deal-item-{id}` | `id: "deal-item-${SEED_DEAL_ID}"` | Deal item |
| `deal-detail-screen` | `id: "deal-detail-screen"` | Deal detail |
| `deal-new-screen` | `id: "deal-new-screen"` | Deal new form |
| `deal-edit-screen` | `id: "deal-edit-screen"` | Deal edit form |
| `deal-edit-button` | `id: "deal-edit-button"` | Deal edit button |
| `create-deal-fab` | `id: "create-deal-fab"` | Deals FAB |
| `cases-list` | `id: "cases-list"` | Cases list |
| `cases-list-item-0` | `id: "cases-list-item-0"` | First case |
| `case-detail-screen` | `id: "case-detail-screen"` | Case detail |
| `case-new-screen` | `id: "case-new-screen"` | Case new form |
| `case-edit-screen` | `id: "case-edit-screen"` | Case edit form |
| `case-edit-button` | `id: "case-edit-button"` | Case edit button |
| `create-case-fab` | `id: "create-case-fab"` | Cases FAB |
| `cases-search` | `id: "cases-search"` | Cases search input |
| `copilot-panel` | `id: "copilot-panel"` | Copilot panel |
| `workflows-list` | `id: "workflows-list"` | Workflows list |
| `workflows-new-btn` | `id: "workflows-new-btn"` | New workflow button |
| `workflow-new-screen` | `id: "workflow-new-screen"` | Workflow new form |
| `workflow-{id}` | `id: "workflow-${SEED_WORKFLOW_ACTIVE_ID}"` | Workflow card |
| `workflow-detail` | `id: "workflow-detail"` | Workflow detail |
| `workflow-edit-btn` | `id: "workflow-edit-btn"` | Workflow edit button |
| `workflow-edit-screen` | `id: "workflow-edit-screen"` | Workflow edit form |
| `agent-runs-list-screen` | `id: "agent-runs-list-screen"` | Agent runs list |
| `agent-run-item-{id}` | `id: "agent-run-item-${SEED_AGENT_RUN_REJECTED_ID}"` | Agent run item |
| `agent-run-detail-screen` | `id: "agent-run-detail-screen"` | Agent run detail |

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Maestro cannot find `resource-id` for a testID | Low | Medium | React Native always maps testID to resource-id on Android. Verify with `maestro hierarchy` on first test. |
| No active signal exists for screenshot `04` | Medium | Medium | Resolve `SEED_SIGNAL_ID` via BFF during Task 2. If none exists, emit `04_home_signal_detail_EMPTY` and document it explicitly. |
| Scroll-to-find for seeded items fails | Medium | Low | Use `scrollUntilVisible` with `id` selector. Maestro supports this natively. |
| Maestro env var interpolation syntax differs | Low | Low | Test with one var first in Task 3. Maestro uses `${VAR}` syntax. |
| Maestro screenshot output path differs from expected | Low | Low | Confirm the local output directory during Task 2 and normalize/copy artifacts into `mobile/artifacts/screenshots/`. |
| Existing Detox E2E tests break during cleanup | Low | High | Task 9 only deletes screenshot-specific files. Run `npm run e2e:test` before commit. |

---

## Rollback Plan

If Maestro proves unsuitable:
1. Revert the commit (all Detox screenshot files are restored)
2. Uninstall Maestro (`rm -rf ~/.maestro`)
3. Resume with Strategy A or B from `docs/plans/mobile-detox-activity-launch-timeout-fix.md`

---

## Success Criteria

- [ ] `maestro --version` works
- [ ] `npm run screenshots` runs unattended (no manual keypress)
- [ ] Expected PNG files are generated under `mobile/artifacts/screenshots/`
- [ ] Each PNG shows the correct screen (visual spot-check)
- [ ] `npm run e2e:test` still passes (Detox tests unaffected)
- [ ] `npm run quality` passes
- [ ] Required mobile QA gates pass before any push
- [ ] Zero changes to app source code (only test infrastructure)
