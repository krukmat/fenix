---
doc_type: task
id: EXTVAL-BATTERY-T7-001
title: "Run battery T7 — Mobile real-mode navigation against live external validation runtime"
status: complete
phase: external-validation-battery
week: "2026-W27"
tags: [external-validation, battery, mobile, e2e-mode, governance, hitl]
fr_refs: [FR-230, FR-232]
uc_refs: [UC-C1]
blocked_by: [EXTVAL-BATTERY-T6-001]
blocks: []
files_affected: []
created: 2026-07-02
completed: 2026-07-02
blocked_reason:
---

# Task EXTVAL-BATTERY-T7-001

**Plan**: [External Validation First Test Battery Plan](../plans/external_validation_first_test_battery_plan.md#t7-mobile-real-mode-navigation)

## Task Card

Task: EXTVAL-BATTERY-T7-001

Task file: docs/tasks/task_extval_battery_t7_mobile_real_mode.md

Plan file: docs/plans/external_validation_first_test_battery_plan.md

Summary: Prove the mobile app is not relying on E2E auth-injection or query-idle behavior for validation by starting Expo without `EXPO_PUBLIC_E2E_MODE=1`, logging in through the visible auth UI on a real Android emulator, and navigating Support, Inbox, Activity, Sales Brief, and Governance while observing BFF-driven state refresh. This closes the mobile-UI gap explicitly deferred by T3, T4, and T5.

Code affected: None expected. Runtime/environment validation only (Android SDK/emulator installation, Expo runtime env override). No application source files touched.

Effort/reasoning: Medium - requires provisioning a full Android toolchain (Java, SDK platform-tools, emulator, system image) not previously present in this environment, then live UI navigation across five app surfaces.

Recommended model: claude-sonnet-4-6

Estimated tokens: ~9000

Task type: operational validation. No dev-task pseudocode required — no code is written.

## Pre-Existing Finding (discovered during setup, before navigation)

`mobile/.env` (committed, not gitignored, loaded by default by `expo start` /
`npm run android`) is byte-identical to `mobile/.env.e2e` and hardcodes
`EXPO_PUBLIC_E2E_MODE=1`, despite an inline comment stating
`# Set only in .env.e2e for Detox E2E builds`. Introduced in commit `6fb11a4`
("feat(ci): implement Detox E2E CI pipeline with Android emulator (Task 4.8)")
and never corrected since. This means a default `npm start`/`expo start` run
today boots straight into E2E auth-injection mode
(`mobile/app/e2e-bootstrap.tsx`), not the real login screen — the exact
condition this task exists to catch. Governance gate logic itself
(`process.env.EXPO_PUBLIC_E2E_MODE !== '1'` check in `e2e-bootstrap.tsx`) is
correct; the defect is purely in which env file ships as the default.

This task overrides the env var at process launch to validate the *real*
login path per plan intent, and reports the `.env` drift as a battery
finding rather than silently working around it.

## Environment Provisioning (performed this session)

No Android toolchain existed in this environment prior to this task (no
`adb`, no `simctl`/Xcode, no running Expo/emulator process for this repo).
Per explicit user authorization ("realiza tu el proceso. instala y configura
lo que necesites"), provisioned locally without `sudo`:

1. `brew install openjdk` (formula, not cask — avoids the `sudo`-gated
   `.pkg` installer that `temurin` cask requires and that failed
   non-interactively).
2. Used pre-existing `android-commandlinetools` cask
   (`/opt/homebrew/share/android-commandlinetools`) already installed.
3. `sdkmanager --licenses` (accepted).
4. `sdkmanager "platform-tools" "emulator" "platforms;android-34" "system-images;android-34;google_apis;arm64-v8a"`
   (arm64 image chosen to match host `uname -m` = arm64, native execution).
5. `avdmanager create avd -n fenix_t7 -k "system-images;android-34;google_apis;arm64-v8a" -d pixel_6`.
6. Booted headless: `emulator -avd fenix_t7 -no-window -no-audio -no-boot-anim -gpu swiftshader_indirect`.
7. Confirmed boot via `adb devices` (`device`, not `offline`) and
   `adb shell getprop sys.boot_completed` = `1`.

iOS Simulator was not provisioned (`app.json` declares `"platforms": ["android"]`
only — Android-only app, so this is sufficient scope).

## Runtime Provenance Check

Backend (`fenix serve --port 8080`) process start time: `Thu Jul 2 07:28:55 2026`.
Uncommitted T4 fix file mtimes: `evidence.go` Jul 1 21:02, `support.go` Jul 1
21:05 — both predate backend process start. Confirmed the running backend
binary was built from a checkout including the T4 fix. BFF (`node dist/server.js`,
started Jul 1 18:32) has no Go source dependency, so provenance is N/A for it;
`/bff/health` reports `backend: reachable`, confirming live connectivity to the
provenance-confirmed backend.

## Operational Procedure

1. Provision Android toolchain (done — see above).
2. Verify backend/BFF runtime provenance includes the T4 fix (done — see above).
3. Start Expo/Metro for `mobile/` with `EXPO_PUBLIC_E2E_MODE` explicitly unset/overridden to a non-`1` value at process launch (not relying on `mobile/.env`'s checked-in default).
4. Install/launch the app on `fenix_t7` emulator (`expo run:android` or `expo start` + `adb` install).
5. Confirm the app boots to `/login`, not `/e2e-bootstrap` or an authenticated route.
6. Log in through the visible auth UI using a retained or freshly created operator credential.
7. Navigate: Support, Inbox, Activity, Sales Brief, Governance.
8. Trigger the support agent from a support case detail screen.
9. Observe state refresh sourced from BFF (not stale/cached), cross-checked against backend logs/audit events for the same request window.
10. Capture environment dump showing `EXPO_PUBLIC_E2E_MODE` unset for the running process, plus mobile and BFF/backend logs.

## Acceptance Criteria

1. Expo is started with `EXPO_PUBLIC_E2E_MODE` unset (not `1`) for the validated run, verified by environment dump.
2. App boots to the real `/login` screen, not the E2E bootstrap route.
3. Login succeeds through the visible UI (not token injection).
4. Support, Inbox, Activity, Sales Brief, and Governance screens are reachable and render live data.
5. Triggering the support agent from a case detail screen produces a state refresh traceable to a real BFF/backend request (not idle/cached state).
6. `mobile/.env` default-E2E-mode finding is recorded regardless of navigation outcome.

## Findings Log

### Finding 1 (most significant — real product bug, not a fixture/mock issue): Home screen hard-crashes on login because `getPendingApprovals` unwraps the wrong response shape

Reproduced via real UI navigation (not curl, not a unit test): after a genuine
`POST /bff/auth/login` through the visible login form, the app navigates to
the Home tab and immediately renders a fatal `TypeError: approvals.map is not
a function (it is undefined)` in `mobile/src/components/home/HomeFeed.tsx:33`
(`buildFeedItems`), thrown from `mobile/app/(tabs)/home/index.tsx:49`. The dev
error overlay ("Render Error") is persistent and blocks all further
navigation — minimizing it leaves a blank white screen with no reachable tab
bar; the crash re-fires on every re-render attempt.

Root cause: `mobile/src/services/api.secondary.ts:61-66`
(`approvalApi.getPendingApprovals`) does
`const response = await apiClient.get('/bff/api/v1/approvals', ...); return response.data as ApprovalRequest[];`
— but the real BFF response body is `{"data": [...], "meta": {"total": N}}`
(confirmed via direct `curl` against the same endpoint/token used by the app:
`{"data":[],"meta":{"total":0}}`), not a bare array. The `as ApprovalRequest[]`
is an unchecked type assertion that hides the shape mismatch at compile time.
`mobile/app/(tabs)/home/index.tsx:20` then does
`const approvals = approvalsData ?? []`, which only guards against
`null`/`undefined` — it does not catch the case where `approvalsData` is a
truthy non-array object (`{data:[], meta:{...}}`), so `undefined` never
reaches the `?? []` fallback and `.map()` is called directly on the wrapper
object's `undefined` `.map` property... (technically `approvals` ends up
being the wrapper object itself, which has no `.map`).

**Evidence correlating the crash to a real BFF round-trip** (not stale/cached
state): `audit_event` row `019f21ad-6cd9-70b6-e9db-9e552c78842c`,
action=`get_request`, `path=/api/v1/approvals`, `status_code=200`,
`duration_ms=4`, timestamped `2026-07-02 09:14:01.561791`, immediately after
the `login` audit event (`019f21ad-69f2-...`, `09:14:00.818113`) which in turn
matches the UI login tap timestamp. A second identical successful
`/api/v1/approvals` call at `09:15:35.502431` matches the re-render/retry
after minimizing the error overlay — confirming this is a live, repeatable
BFF-driven failure, not a one-off.

**Risk:** This is a full navigation-blocking crash on the app's default
landing screen for any account with zero pending approvals (i.e., every fresh
workspace, and the common case for most operators most of the time) — not an
edge case. It directly blocks reaching Support, Inbox, Activity, Sales Brief,
and Governance from the Home entry point, and would be immediately visible to
any real external user on first login. This is a P0-severity finding for
external validation: the mobile app is currently unusable past login in
real-mode for a fresh/typical workspace.

**Recommendation:** File as a P0 bugfix, blocking for external demo/validation
readiness: (a) fix `approvalApi.getPendingApprovals` to unwrap
`response.data.data` (or fix the BFF/backend to return a bare array
consistently, whichever is the intended contract — check other list
endpoints for consistency), (b) remove/replace the unchecked
`as ApprovalRequest[]` cast with runtime validation, (c) add a defensive
`Array.isArray()` guard in `HomeScreen`/`HomeFeed` as defense-in-depth so a
future shape mismatch degrades gracefully instead of hard-crashing the whole
tab.

### Finding 2 (minor, RBAC/permissions gap on a fresh workspace): `GET /api/v1/signals` returns 403 for the workspace-owner operator

Same audit trail shows two `get_request|denied` events for
`GET /api/v1/signals`, `status_code=403`, for the same freshly-registered
operator who owns the workspace (the account/contact/case in this task were
all created by this same user via `ownerId`). A workspace owner/operator
being forbidden from their own workspace's signals feed on a completely fresh
workspace suggests either a missing default permission grant on
registration, or an RBAC scope that excludes the registering user by default.
Not confirmed as a blocker for T7's core acceptance criteria (Home screen
crashes on the `approvals` fetch regardless of this), but compounds the same
screen's brittleness — even if Finding 1 were fixed, this call would still
silently fail on the Home feed today.

**Recommendation:** P2 follow-up — verify default RBAC grants for a freshly
registered workspace-owning user include read access to
`/api/v1/signals` for their own workspace.

### Setup note: Expo Go is not viable for this project — requires a native dev build

`expo start --android` defaults to launching inside Expo Go, but `mobile/app.json`
declares `@config-plugins/detox` and other native config plugins (custom Android
`package: com.fenixcrm.app`, `expo-secure-store`, etc.) that Expo Go cannot host.
First attempt failed: `adb ... shell monkey -p host.exp.exponent ...` exited 251
(Expo Go could not resolve/launch the project). Pivoted to `expo run:android`,
which prebuilds a native `android/` project and installs a real dev-client APK —
consistent with how the project's own `npm run android` script
(`"android": "expo run:android"`) is defined. This required additionally
installing Android `build-tools;34.0.0`, `ndk;27.1.12297006`, and `cmake;3.22.1`
via `sdkmanager` (native modules require NDK/CMake to compile during prebuild).

### Setup note: `openjdk` (v26, latest) is too new for this project's Gradle version

First Gradle build attempt failed: `BUG! exception in phase 'semantic analysis' ...
Unsupported class file major version 70` (class file major version 70 = Java 26).
Gradle 8.14.3 (the version this project's wrapper pulls) does not yet support
JDK 26. Switched to `openjdk@17` (already present as a transitive brew
dependency, no extra install needed) via `JAVA_HOME`/`PATH`, which is the
standard JDK version for current React Native/Expo Android tooling.

## Closure Report

### Environment dump (acceptance criterion 1)

`expo run:android` process launch env (captured before Gradle build):
`EXPO_PUBLIC_E2E_MODE=0`, `EXPO_PUBLIC_BFF_URL=http://10.0.2.2:3000`,
`JAVA_HOME=/opt/homebrew/opt/openjdk@17`,
`ANDROID_HOME=ANDROID_SDK_ROOT=/opt/homebrew/share/android-commandlinetools`.
This overrides the checked-in `mobile/.env` value of `EXPO_PUBLIC_E2E_MODE=1`
(process env takes precedence over `@expo/env`'s dotenv loading). Confirmed
non-E2E boot behavior empirically: the app rendered the real `/login` form
(Email/Password fields, "Sign In" button), not the `/e2e-bootstrap` spinner
route — this is the actual governance-relevant proof, independent of the env
dump.

### Runtime provenance

Backend (`fenix serve --port 8080`, started 2026-07-02 07:28:55) confirmed
predates-fix-inclusive: uncommitted `evidence.go`/`support.go` (T4 fix)
mtimes are Jul 1, both before backend start. BFF (`node dist/server.js`,
started Jul 1 18:32:46) confirmed reachable and proxying to this backend via
`/bff/health` → `{"backend":"reachable"}`.

### Toolchain provisioned this session (no prior Android tooling existed)

- `brew install openjdk` (formula) — Java 26, later found incompatible with Gradle, superseded by `openjdk@17` (pre-existing transitive dependency).
- Pre-existing `android-commandlinetools` cask licensed via `sdkmanager --licenses`.
- `sdkmanager` install: `platform-tools`, `emulator`, `platforms;android-34`, `system-images;android-34;google_apis;arm64-v8a`, `build-tools;34.0.0`, `ndk;27.1.12297006`, `cmake;3.22.1`.
- AVD `fenix_t7` created (`pixel_6` device profile, arm64 system image) and booted headless (`-no-window -no-audio -no-boot-anim -gpu swiftshader_indirect`).
- `npx expo run:android` — native prebuild + Gradle assembleDebug + install, `BUILD SUCCESSFUL in 2m 9s`.

### Acceptance Criteria — Results

1. Expo started with `EXPO_PUBLIC_E2E_MODE` unset/non-`1` (`=0`), verified by process env dump above. **PASS.**
2. App boots to the real `/login` screen, not `/e2e-bootstrap`. **PASS** — confirmed via screenshot (`Email`/`Password` fields, "Sign In" button, no `e2e-bootstrap-screen` testID content).
3. Login succeeds through the visible UI (not token injection). **PASS** — typed credentials for a freshly `POST /bff/auth/register`-ed operator into the real form fields via `adb shell input text`, tapped "Sign In"; confirmed by `audit_event` row `019f21ad-69f2-...`, `action=login`, `outcome=success`, timestamp matching the UI tap.
4. Support, Inbox, Activity, Sales Brief, and Governance screens are reachable and render live data. **FAIL — blocked.** The Home tab (the post-login landing screen) hard-crashes on render (Finding 1) before any tab navigation is reachable; minimizing the dev error overlay leaves a blank screen with no navigable UI. This is a genuine product defect discovered by this task, not a test-setup gap — see Finding 1.
5. Triggering the support agent from a case detail screen produces a state refresh traceable to a real BFF/backend request. **NOT REACHED** — blocked by criterion 4's crash; case detail screen was never reached via UI navigation. (A case was created and is available via API — `019f21ab-1784-...`, "ExtVal T7 - login not working after password reset", `priority: high` — for a future retest once Finding 1 is fixed.)
6. `mobile/.env` default-E2E-mode finding is recorded regardless of navigation outcome. **PASS** — recorded above, independent of the Finding 1 blocker.

### Go/No-Go (per plan: "mobile is not relying on E2E auth or query-idle behavior for validation")

**Partial GO on the plan's core question, hard FAIL on usability.** The plan's
specific concern — is mobile secretly depending on E2E auth-injection or
idle/cached query state to appear to work — is answered **no**: real
`/login` UI, real credential entry, real `POST /bff/auth/login`, and a real,
freshly-fetched (`200`, `duration_ms: 4`) `/api/v1/approvals` call are all
confirmed via audit trail. The app is not faking real-mode. However, real-mode
navigation immediately surfaces a P0 crash (Finding 1) that was invisible to
T3/T4/T5's API-only validation and would be the first thing any real external
user or demo hits. Battery navigation criteria (Support/Inbox/Activity/Sales
Brief/Governance/agent-trigger) could not be completed as a direct consequence.
**Recommend this crash be fixed and T7 re-run for the deferred navigation
criteria (4 and 5) before treating mobile as externally demo-ready.**
