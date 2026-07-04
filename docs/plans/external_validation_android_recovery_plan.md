---
doc_type: plan
title: "External Validation Android Recovery and T7 Revalidation Plan"
status: active
created: 2026-07-04
task: EXTVAL-ANDROID-RECOVERY-001
depends_on:
  - docs/plans/external_validation_open_points_rerun_plan.md
  - docs/tasks/task_extval_battery_t7_mobile_real_mode.md
  - docs/tasks/task_mobile_approvals_response_shape_crash.md
---

# External Validation Android Recovery and T7 Revalidation Plan

## Purpose

Recover the broken local Android emulator path that now blocks manual
real-mode validation, then define the shortest safe rerun sequence needed to
 close the pending `T7` acceptance criteria after the mobile approvals crash fix.

## Problem Statement

The mobile approvals crash fix is implemented and local mobile QA is green, but
the final manual proof remains blocked by the local Android environment:

- backend and BFF can be restarted and respond healthy
- the `fenix_t7` AVD still exists
- the AVD config points to `system-images/android-34/google_apis/arm64-v8a/`
- that system-image directory is missing on the host
- the emulator exits with `Broken AVD system path`

This is now an environment-recovery problem, not a product-code problem.

## Current Evidence

### Product-side evidence already green

- Mobile approvals response-shape fix implemented in code.
- Required mobile QA gates passed:
  - `bash scripts/check-no-inline-eslint-disable.sh`
  - `cd mobile && npm run typecheck`
  - `cd mobile && npm run lint`
  - `cd mobile && npm run quality:arch`
  - `cd mobile && npm run test:coverage`
- Peer code review passed for the crash fix.

### Environment-side evidence still blocking

- `curl http://localhost:8080/health` and `curl http://localhost:3000/bff/health`
  can pass when services are restarted.
- `~/.android/avd/fenix_t7.avd/config.ini` still references
  `image.sysdir.1=system-images/android-34/google_apis/arm64-v8a/`.
- `find /Users/matias/Library/Android -path '*system-images/android-34/google_apis/arm64-v8a'`
  returns no directory.
- Direct emulator launch fails with:
  - `Cannot find AVD system path`
  - `Broken AVD system path`

## Root-Cause Analysis

### Primary blocker

The AVD definition survived, but the Android system image it depends on no
longer exists under the SDK root. The emulator therefore fails before Android
boots, so the task cannot reach app install or login.

### Likely causes

1. The system image was removed or relocated after the original T7 run.
2. The host now has partial Android SDK contents split across different roots.
3. The `fenix_t7` AVD may have been created against an SDK layout that no
   longer matches the current shell environment.

### Non-causes

- Not caused by the mobile approvals code fix.
- Not caused by backend/BFF health.
- Not caused by Expo route logic.

## Recovery Strategy

### Phase A. Repair the Android runtime substrate

1. Confirm the active SDK root that the emulator should use.
   Evidence:
   - `ANDROID_SDK_ROOT`
   - `ANDROID_HOME`
   - presence of `emulator/`, `platform-tools/`, and `system-images/`
2. Reinstall the missing system image if absent:
   - `platforms;android-34`
   - `system-images;android-34;google_apis;arm64-v8a`
3. Recreate `fenix_t7` if the AVD still points to stale or temp-backed paths.
4. Boot the emulator headless and verify:
   - `adb devices` shows `device`
   - `adb shell getprop sys.boot_completed` returns `1`

### Phase B. Restore the mobile launch path

1. Restart Ollama if needed.
2. Restart backend from the current fix-bearing checkout.
3. Restart BFF with production validation env and fixtures disabled.
4. Verify:
   - `/health`
   - `/readyz`
   - `/bff/health`
5. Launch the Android app using the shortest valid path:
   - reuse installed dev build if present, or
   - `cd mobile && EXPO_PUBLIC_E2E_MODE=0 npx expo run:android`

### Phase C. Execute the focused T7 rerun

1. Confirm the app boots to the real `/login` UI.
2. Log in through visible auth UI.
3. Confirm Home no longer crashes after the approvals fetch.
4. Re-check whether `GET /api/v1/signals` still returns `403` for the workspace
   owner.
5. Navigate:
   - Support
   - Inbox
   - Activity
   - Sales Brief
   - Governance
6. If Home is stable, complete the previously blocked agent-trigger path from a
   case detail screen.
7. Record updated evidence packet for the rerun.

## Decision Tree

### If Phase A succeeds

Proceed directly to T7 rerun and treat the mobile crash fix as ready for final
closure pending observed UI behavior.

### If Phase A fails because the SDK is incomplete

Open a bounded environment-repair task and stop before claiming manual product
revalidation.

### If Phase A succeeds but Home still fails

Reopen the product bug because the local QA suite was insufficient to capture
the remaining runtime issue.

### If Home succeeds but `signals` still 403s

Treat the crash fix as validated and keep the signals issue as the next mobile
rerun blocker under `O7`.

## Suggested Discrete Tasks

1. Task: repair Android SDK/AVD state and prove emulator boot readiness.
2. Task: rerun `T7` real-mode login/navigation against the fixed mobile build.
3. Task: if needed, isolate the `signals` 403 workspace-owner defect.

## Exit Criteria

This recovery plan is complete when:

- the emulator boots again on this host
- backend and BFF are healthy from the current checkout
- `T7` login is rerun through the visible UI
- the Home approvals crash is either disproven in real mode or re-reproduced
  with new evidence
