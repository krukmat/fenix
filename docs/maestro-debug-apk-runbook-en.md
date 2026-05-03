---
doc_type: summary
title: "Maestro debug APK runbook (EN)"
status: active
created: 2026-05-03
tags:
  - maestro
  - mobile
  - screenshots
  - android
  - runbook
task_refs:
  - SCR-FIX1
---

# Runbook: starting Maestro with the real debug APK

## Goal

Document the exact operational procedure that worked to run `Maestro` against the real `app-debug.apk` and diagnose the case where the app gets stuck on the splash screen.

Canonical command:

```bash
cd mobile && npm run screenshots
```

## Prerequisites

Before running Maestro, the local environment must have:

- an Android emulator or device connected through `adb`
- the Go backend running on `localhost:8080`
- the BFF running on `localhost:3000`
- a debug APK built and installed, or at least buildable from `mobile/android`
- Metro available on `localhost:8081`

## Main symptom observed

The app stayed stuck on the splash screen when launching the debug APK from the screenshots runner.

The problem was not login, seed, or BFF. The root cause was that the debug build was trying to load the JS bundle from Metro, and Metro was not running on `:8081`.

## Useful log evidence

When the splash issue is caused by Metro being down, `adb logcat` shows lines like:

```text
Couldn't connect to "ws://10.0.2.2:8081/message..."
The packager does not seem to be running
Unable to load script
Make sure you're running Metro...
```

You can also confirm that Android is still holding the splash window:

```bash
adb shell dumpsys activity activities | rg "com.fenixcrm.app|Splash"
adb shell dumpsys window windows | rg "Splash Screen com.fenixcrm.app|firstWindowDrawn"
```

## Full startup sequence that worked

### 1. Verify the Android device

```bash
adb devices
```

Expected: an emulator or device in `device` state.

### 2. Start the Go backend

From the repository root:

```bash
JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080
```

Quick check:

```bash
curl -fsS http://127.0.0.1:8080/health
```

### 3. Start the BFF

Start the BFF so it responds on `localhost:3000`.

Quick check:

```bash
curl -sS --max-time 3 http://127.0.0.1:3000
```

If the BFF does not respond, `seed-and-run.sh` exits before seeding.

### 4. Build the debug APK if needed

If the debug APK is missing or stale, rebuild it:

```bash
cd mobile
npx expo run:android
```

Direct Gradle alternative:

```bash
cd mobile/android
./gradlew assembleDebug
```

Expected APK path:

```text
mobile/android/app/build/outputs/apk/debug/app-debug.apk
```

### 5. Install or reinstall the debug APK

If you need to reinstall it manually:

```bash
adb install -r mobile/android/app/build/outputs/apk/debug/app-debug.apk
```

The runner also tries to install it automatically if the file exists.

### 6. Start Metro on `8081`

From `mobile/`:

```bash
cd mobile
npx expo start --port 8081 --host localhost
```

Note: `expo` reported that `--non-interactive` is not supported here and suggested using `CI=1` if non-interactive mode is needed.

### 7. Confirm that Metro is responding

```bash
curl -sS http://127.0.0.1:8081/status
```

Expected:

```text
packager-status:running
```

If this returns `Connection refused`, getting stuck on splash is expected for the debug APK.

### 8. Prepare Android networking

The runner already sets:

```bash
adb reverse tcp:3000 tcp:3000
adb reverse tcp:8080 tcp:8080
adb reverse tcp:8081 tcp:8081
```

Even though the emulator often resolves React Native through `10.0.2.2:8081`, keeping `adb reverse tcp:8081 tcp:8081` is still part of the correct setup and helps on physical devices.

### 9. Run the screenshots runner

```bash
cd mobile && npm run screenshots
```

In the successful run we observed:

- the runner detected Metro with `Using existing Metro server at http://127.0.0.1:8081/status`
- seeding completed correctly
- execution advanced to `Phase 1/2: capturing auth surface...`
- the splash/packager failure stopped happening

## Quick diagnosis if the app gets stuck on splash again

### Check 1. Local Metro

```bash
curl -sS --max-time 3 http://127.0.0.1:8081/status
```

If this fails, start Metro first.

### Check 1b. Backend and BFF

```bash
curl -fsS http://127.0.0.1:8080/health
curl -sS --max-time 3 http://127.0.0.1:3000
```

If either fails, the problem is still local infrastructure, not the Maestro flow itself.

### Check 2. Filtered app logcat

```bash
adb shell pidof com.fenixcrm.app
adb logcat --pid <PID> -d | rg "ReconnectingWebSocket|Unable to load script|packager|ReactNative|10.0.2.2:8081"
```

If you see `Unable to load script` or `The packager does not seem to be running`, the problem is still the bundler.

### Check 3. Android visual state

```bash
adb shell dumpsys activity activities | rg "com.fenixcrm.app|Splash"
adb shell dumpsys window windows | rg "Splash Screen com.fenixcrm.app|mTopFullscreenOpaqueWindowState"
```

If the splash window is still the top window, React Native did not draw the first screen.

## Runner adjustment applied

`mobile/maestro/seed-and-run.sh` was reinforced to:

- detect whether Metro is already alive at `http://127.0.0.1:8081/status`
- start Metro if it is missing
- wait actively for `packager-status:running`
- keep `adb reverse tcp:8081 tcp:8081`

This reduces the risk of repeating the failure when using the real debug APK.

## Follow-up observation after the fix

Once Metro was fixed, the next observed symptom was no longer splash but an intermittent Android ANR:

```text
Process system isn't responding
```

`auth-surface.yaml` already includes a block that tries to dismiss it with `Wait`. If it reappears, the problem is no longer the bundler but emulator stability or startup load.

## Reference commands

```bash
adb devices
curl -fsS http://127.0.0.1:8080/health
curl -sS --max-time 3 http://127.0.0.1:3000
curl -sS http://127.0.0.1:8081/status
adb shell pidof com.fenixcrm.app
adb logcat --pid <PID> -d | rg "Unable to load script|packager|10.0.2.2:8081"
adb shell dumpsys activity activities | rg "com.fenixcrm.app|Splash"
JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080
cd mobile && npx expo run:android
cd mobile/android && ./gradlew assembleDebug
cd mobile && npx expo start --port 8081 --host localhost
cd mobile && npm run screenshots
```

## Expected outcome

With Metro running and the device reachable:

- the app should leave the splash screen
- `auth-surface.yaml` should be able to wait for `login-screen`
- `npm run screenshots` should at least get through phase 1 without failing because of the packager
