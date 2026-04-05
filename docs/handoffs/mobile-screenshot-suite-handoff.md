# Mobile Screenshot Suite Handoff

Use this handoff when continuing the Detox mobile screenshot work for FenixCRM.

## Mission

You are taking over the mobile screenshot stabilization effort.

Primary goal:

- make `npm run screenshots` complete successfully
- generate the expected screenshot set under `mobile/artifacts/screenshots/`
- do not expand scope into general mobile hardening unless it directly unblocks the screenshot suite

Secondary constraints:

- keep the user informed with a concise TODO list while working
- prioritize the primary goal over style-only cleanup
- preserve existing user changes in the worktree

## Source Of Truth

Read these before changing behavior:

1. `docs/plans/mobile-screenshot-audit.md`
2. `mobile/e2e/screenshots.e2e.ts`
3. `mobile/e2e/helpers/auth.helper.ts`
4. `mobile/e2e/helpers/seed.helper.ts`
5. `mobile/e2e/helpers/screenshots.helper.ts`
6. `scripts/e2e_seed_mobile_p2.go`
7. `mobile/.detoxrc.js`
8. `mobile/package.json`
9. `mobile/app/_layout.tsx`
10. `mobile/app/(tabs)/_layout.tsx`

## Current Objective State

What is already in place:

- dedicated Jest config for screenshots
- dedicated Detox screenshots configuration
- `npm run screenshots` script
- Detox screenshot suite in `mobile/e2e/screenshots.e2e.ts`
- E2E bootstrap changes in `mobile/app/_layout.tsx`
- root route support through `mobile/app/index.tsx`
- auth route fixes in the auth and tabs layouts
- seeded mobile P2 fixtures now include stable contact and case metadata

What was already proven in earlier runs:

- `01_auth_login`
- `02_auth_register`
- `05_crm_hub`
- `06_crm_accounts_list`
- `07_crm_account_detail`

That means the effort is past setup. The remaining work is runtime stabilization and full-run completion.

## Changes Already Applied

### 1. Drawer navigation fix

`mobile/app/(tabs)/_layout.tsx` was patched because logs showed invalid navigator targets like:

`The action 'NAVIGATE' with payload {"name":"crm/accounts/index"} was not handled by any navigator.`

Current drawer behavior uses `router.push(...)` for:

- `/home`
- `/crm/accounts`
- `/crm/contacts`
- `/crm/deals`
- `/crm/cases`
- `/copilot`
- `/workflows`
- `/activity`

This fix should be kept unless runtime evidence proves a different route contract.

### 2. Seed stabilization

`scripts/e2e_seed_mobile_p2.go` and `mobile/e2e/helpers/seed.helper.ts` were extended so the seed now exposes:

- `contact.id`
- `contact.email`
- `case.id`
- `case.subject`

This was done to remove runtime dependency on BFF contact creation.

### 3. Screenshot suite stabilization

`mobile/e2e/screenshots.e2e.ts` was refactored to:

- use explicit section navigation helpers
- relaunch authenticated sessions more predictably
- use seeded contact/case data instead of creating contacts through BFF
- reopen workflow detail before edit where needed

### 4. BFF runtime contact creation was removed from the critical path

`mobile/e2e/helpers/screenshots.helper.ts` no longer needs `POST /bff/api/v1/contacts` for the screenshot suite.

Reason:

- that route returned `502 socket hang up` during E2E runs

## Root Cause (Resolved)

AndroidX `MonitoringInstrumentation.startActivitySync()` has a **hardcoded 45s timeout** that cannot be configured via Detox or Jest. The FenixCRM app takes >45s on cold boot (JIT/ART odex cache verification of React Native bytecode). All attempts to pre-warm the app before `launchApp` failed because Detox always issues a new `am start` intent which resets the 45s clock.

**Decision: use `android.attached` configuration.** This bypasses the timeout entirely because Detox attaches to an already-running app process instead of launching it.

## How To Run Screenshots

**Prerequisites** (must be done before `npm run screenshots`):

1. Ensure the emulator is running with the app already loaded and warm:
   ```sh
   adb shell am start -n com.fenixcrm.app/.MainActivity
   # Wait until the login screen appears in the emulator
   ```

2. Ensure port forwarding is active (BFF + backend):
   ```sh
   adb reverse tcp:3000 tcp:3000
   adb reverse tcp:8080 tcp:8080
   adb reverse tcp:8099 tcp:8099
   ```

3. Verify backend, BFF, and Metro are running:
   - Backend: `http://localhost:8080/health`
   - BFF: `http://localhost:3000/bff/health`
   - Metro: `http://localhost:8081`

4. Run the suite:
   ```sh
   cd mobile && npm run screenshots
   ```

**Active Detox configuration:** `android.att.debug.screenshots` (uses `android.attached`, matches any connected emulator via `adbName: '.*'`).

**Note:** `adbName: '.*'` is a regex that matches any connected device. If multiple devices/emulators are connected, Detox will fail. Ensure only one emulator is connected when running screenshots.

## Known Good Operational Assumptions

- backend API should be healthy on `http://localhost:8080/health`
- BFF should be checked on `http://localhost:3000/bff/health`
- Metro should be available on `8081`
- mobile E2E mode depends on `EXPO_PUBLIC_E2E_MODE=1`
- the seed script should remain the source of deterministic CRM data for the screenshot suite

## Guardrails

- do not revert unrelated user changes
- do not broaden scope into general Detox cleanup unless startup requires it
- do not optimize for lint/style cleanup ahead of restoring a successful screenshot run
- keep the drawer route fix and seed stabilization unless proven wrong by new evidence

## Expected Output

- complete screenshot run generating 26 PNGs under `mobile/artifacts/screenshots/`
- final PNG count verification
- list of remaining failing screens, if any

