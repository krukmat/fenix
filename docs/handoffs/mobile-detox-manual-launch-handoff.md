# Mobile Detox Manual Launch Handoff

Use this handoff when taking over the Android mobile screenshot stabilization work after the `launchApp: 'manual'` migration attempt.

## Mission Context

The original attached-device workaround is no longer the active analysis path in this branch state.

The current codebase is in a transitional state where:

- Detox screenshots are configured for Android emulator manual launch
- a local wrapper script orchestrates emulator boot, `adb reverse`, and instrumentation bootstrap
- the screenshot suite still does not complete
- the remaining blocker appears to be in the Android instrumentation / Detox runtime handshake, not in the screenshot navigation logic itself

This handoff intentionally does **not** recommend a fix. It documents what was changed, what was observed, and what confidence level applies to each inference.

## Files Touched In This Attempt

Relevant modified or added files:

- `mobile/.detoxrc.js`
- `mobile/e2e/screenshots.e2e.ts`
- `mobile/e2e/helpers/screenshots.helper.ts`
- `mobile/app/e2e-bootstrap.tsx`
- `mobile/scripts/run-screenshots.sh`
- `mobile/scripts/run-screenshots.mjs`
- `mobile/patches/detox+20.47.0.patch`

Patch content already persisted in `mobile/patches/detox+20.47.0.patch` includes:

- `ActivityLaunchHelper.kt` launch change
- `EmulatorTelnet.js` timeout increase
- `pressAnyKey.js` TTY fallback

## Branch-State Summary

Current behavioral intent in the branch:

- `npm run screenshots` should remain the single user entry point
- screenshots use Detox config `android.emu.debug.screenshots`
- that config uses `behavior.launchApp: 'manual'`
- the wrapper script is responsible for the manual launch flow around Detox
- `01_auth_login` and `02_auth_register` remain unauthenticated UI captures
- `03+` use `fenixcrm:///e2e-bootstrap?...` for deterministic authenticated entry

## Current Detox Configuration State

`mobile/.detoxrc.js` currently defines:

- `android.emu.debug.screenshots`
- `device: 'simulator'`
- `app: 'android.debug'`
- `behavior.launchApp: 'manual'`
- `behavior.init.reinstallApp: false`

The global behavior remains `launchApp: 'auto'`, but the screenshots configuration overrides it to manual launch.

## Screenshot Suite State

`mobile/e2e/screenshots.e2e.ts` currently:

- still calls `device.launchApp({ newInstance: true, launchArgs: { detoxEnableSynchronization: '0' } })`
- then explicitly calls:
  - `runtimeDevice.deviceDriver?.waitUntilReady?.()`
  - `runtimeDevice.deviceDriver?.waitForActive?.()`
  - `device.disableSynchronization()`
- uses `e2e-bootstrap` for authenticated screenshots after the auth screens

This explicit readiness wait was added during the manual-launch investigation to avoid issuing the first Espresso call before the runtime was actually ready.

## Wrapper State

`mobile/scripts/run-screenshots.sh` currently:

- restarts ADB
- ensures an emulator for `Pixel_7_API_33`
- clears `~/Library/Detox/device.registry.json`
- applies `adb reverse` for `3000` and `8080`
- exports `FENIX_SCREENSHOTS_DEVICE_SERIAL`
- invokes `node ./scripts/run-screenshots.mjs`

Notable hardening applied during investigation:

- stale AVD `.lock` files are removed before launching the emulator
- duplicate AVD launch is avoided by checking for an already-running emulator process
- the prior zsh bug where `boot_completed=''` polluted stdout has been removed

`mobile/scripts/run-screenshots.mjs` currently:

- spawns `detox test --configuration android.emu.debug.screenshots`
- parses Detox manual-launch prompt output
- extracts:
  - instrumentation class
  - launch args table
- detects the actual dynamic `detoxServer` port from the prompt
- waits for the host Detox server port to accept TCP connections before starting instrumentation
- applies dynamic `adb reverse tcp:<detoxPort> tcp:<detoxPort>`
- launches:
  - `adb -s <serial> shell am instrument -w -r ...`
- sends newline back to Detox so the manual prompt can continue

The wrapper no longer relies on a hardcoded `8099` for the Detox server path.

## E2E Bootstrap State

`mobile/app/e2e-bootstrap.tsx` exists and is used by the screenshot suite.

The intended role of this route is:

- avoid depending on login UI for `03+`
- inject or reconstruct authenticated state deterministically
- redirect to `/home`

This route was introduced to isolate screenshot navigation problems from runtime-launch problems.

## Observed Timeline Of Investigation

### Phase 1: attached-mode and auth/navigation cleanup

Earlier work in this branch fixed real suite issues unrelated to launch:

- auth landing was redirected toward `/home`
- drawer navigation was normalized
- `accounts-list` was no longer treated as the only valid post-login success marker
- `03_home_feed` was made less dependent on drawer state
- authenticated screenshots gained deterministic deep-link bootstrap

Those changes were valid and reduced one class of flakiness, but they did not resolve the app-launch/runtime disconnect.

### Phase 2: move away from attached mode

The codebase was moved away from `android.attached` screenshots and back to manual-launch semantics.

Reason observed in runtime:

- attached mode without `device.launchApp()` left Detox disconnected from the app
- attached mode with `device.launchApp()` tended to reintroduce the Android instrumentation launch bottleneck

### Phase 3: wrapper automation

The wrapper evolved through several iterations:

1. initial manual prompt automation
2. TTY patch to avoid `process.stdin.setRawMode` crash in Detox manual mode
3. dynamic Detox port extraction instead of assuming `8099`
4. AVD lock cleanup
5. duplicate-emulator prevention
6. explicit host-port listening check before instrumentation launch

By the end of this sequence, the wrapper itself was materially more correct than at the start.

## Strongest Runtime Evidence

### 1. Manual prompt automation is active

In clean runs, Detox prints:

- assignment of `screenshots.e2e.ts` to `emulator-5554`
- manual launch block with:
  - instrumentation class
  - `detoxServer`
  - `detoxSessionId`
  - `detoxEnableSynchronization`

The wrapper then logs:

- `[screenshots] app start requested`
- `[screenshots] detox reverse ready: <dynamicPort>`
- `[screenshots] instrumentation starting: ...`
- `[screenshots] instrumentation started`

### 2. Detox can see the app process PID

In the cleanest failing run, Detox reported:

- `Found the app (com.fenixcrm.app) with process ID = 3115. Proceeding...`

This means the app process exists long enough for PID discovery.

### 3. Failure occurs at readiness, not at PID discovery

The same run then failed with:

- `An error occurred while waiting for the app to become ready. Waiting for disconnection...`
- `Failed to run application on the device`
- `The app disconnected.`

This placed the failure after process discovery and before a successful app-ready handshake.

### 4. Android instrumentation reports a crash, not a clean disconnect

The same failing run emitted:

- `INSTRUMENTATION_RESULT: shortMsg=Process crashed.`

Android logcat also showed:

- `Crash of app com.fenixcrm.app running instrumentation ComponentInfo{com.fenixcrm.app.test/androidx.test.runner.AndroidJUnitRunner}`

### 5. Dynamic reverse is present on-device

During the clean failing manual-launch run, `adb reverse --list` included:

- `tcp:3000`
- `tcp:8080`
- `tcp:<dynamicDetoxPort>`

This means the dynamic reverse entry was created, at least from ADB’s perspective.

## Independent Instrumentation Probe

A separate direct command was run outside Detox:

```sh
adb -s emulator-5554 shell am instrument -w -r -e debug false com.fenixcrm.app.test/androidx.test.runner.AndroidJUnitRunner
```

Observed behavior:

- it starts `com.fenixcrm.app.DetoxTest`
- it logs `runDetoxTests`
- logcat then shows Detox trying to connect to a server

Relevant log lines:

- `I/TestRunner(...): started: runDetoxTests(com.fenixcrm.app.DetoxTest)`
- `I/Detox(...): Detox server connection details: url=ws://localhost:8099, sessionId=com.fenixcrm.app`
- `I/Detox(...): Connecting to server...`
- repeated `DetoxWSClient ... Retrying...`

This probe was not a valid screenshot run by itself, but it showed that the Android test APK can enter `DetoxTest` and repeatedly attempt server connection.

## Evidence About Failure Modes

### Failure mode A: instrumentation/runtime readiness crash

Symptoms:

- manual prompt appears
- wrapper starts instrumentation
- Detox sees app PID
- `waitUntilReady()` fails
- app disconnects
- instrumentation reports `Process crashed`

This is currently the cleanest reproducible failure mode.

### Failure mode B: emulator/ADB instability

Separate failing runs also showed:

- `adb: device 'emulator-5554' not found`
- `Cannot connect` during emulator telnet handshake in earlier stages
- `offline` device state in earlier manual debugging
- emulator-launch contamination due to stale `multiinstance.lock`

This means not every failing run is equivalent. Some failures are higher-level runtime failures, while others are lower-level emulator state failures.

## Assumptions And Confidence Levels

The following assumptions are ordered by confidence.

### High confidence

#### A1. The screenshot suite’s primary remaining blocker is no longer drawer/auth navigation logic.

Evidence:

- the cleanest current failures happen before `01_auth_login` logic begins
- failing stack traces point to launch/readiness, not route navigation
- Detox fails inside `beforeAll`

Confidence: high

#### A2. The manual-launch wrapper is now materially closer to correct behavior than the earlier attached-mode wrapper state.

Evidence:

- dynamic Detox port is extracted from Detox output
- dynamic port reverse is actually listed in `adb reverse --list`
- instrumentation launch is automated
- stale AVD lock and duplicate-launch bugs were found and corrected

Confidence: high

#### A3. The app process exists briefly enough for Detox to discover a PID, but not long enough to complete a successful ready handshake.

Evidence:

- Detox logs a concrete PID
- readiness fails immediately afterward
- instrumentation reports process crash/disconnect

Confidence: high

### Medium confidence

#### B1. The current failure is inside Android instrumentation / Detox runtime connection semantics rather than inside the screenshot test code.

Evidence:

- failure occurs in `beforeAll`
- no test-specific navigation has run yet
- instrumentation crash is reported by Android

Why not high confidence:

- the runtime path is still being driven through `device.launchApp()` in manual mode, so the boundary between test API usage and runtime driver behavior is not perfectly isolated

Confidence: medium

#### B2. The Android test APK itself is not obviously broken at class-loading time.

Evidence:

- direct `am instrument` starts `com.fenixcrm.app.DetoxTest`
- `runDetoxTests` begins
- Detox logs appear from the Android side

Why not high confidence:

- Android still reports instrumentation crash later
- the crash may still originate inside test APK runtime after startup

Confidence: medium

#### B3. A race condition between Detox server availability and Android instrumentation startup was at least one legitimate problem during the investigation.

Evidence:

- earlier wrapper versions started instrumentation immediately after seeing the prompt
- later versions added explicit waiting for the host Detox port to listen

Why not high confidence:

- after adding the host-port wait, the core crash still remained
- therefore the race may have been a real bug, but not the final blocker

Confidence: medium

### Low confidence

#### C1. ABI mismatch warnings are the direct cause of the runtime crash.

Evidence:

- logcat repeatedly shows:
  - `Package uses different ABI(s) than its instrumentation: package[com.fenixcrm.app]: arm64-v8a, null instrumentation[com.fenixcrm.app.test]: null, null`

Why low confidence:

- the warning predates some runs that progressed further
- no direct stack trace tied the crash to ABI loading failure

Confidence: low

#### C2. Missing `.dm` files in app/test package paths are causally responsible for the crash.

Evidence:

- logcat shows `Unable to open .../base.dm`

Why low confidence:

- this is common noise in many Android environments
- no direct crash stack pinned the failure to `.dm` absence

Confidence: low

#### C3. The remaining blocker is a single root cause rather than an interaction between multiple unstable layers.

Evidence against certainty:

- there are at least two independent failure families:
  - emulator/ADB instability
  - instrumentation/app-ready crash

Confidence: low

## Things That Are Easy To Misread

### 1. “All 26 tests failed” does not mean 26 distinct application bugs.

In the cleanest recent runs, a single failure in `beforeAll` caused:

- `01_auth_login` to fail
- the rest to skip or fail as follow-on fallout

### 2. A direct `adb reverse --list` entry does not prove successful WebSocket connection.

It only proves the reverse mapping exists in ADB.

### 3. A discovered PID does not imply successful Detox readiness.

The process can exist long enough for PID detection and still fail before sending the expected ready message.

### 4. The presence of `e2e-bootstrap` does not mean authenticated screenshots are currently reachable.

The suite is not reliably reaching screenshot logic yet; current failures happen earlier.

## Known Operational Contaminants During This Investigation

These are not final conclusions about the product. They are environmental contaminants that affected observations:

- too many local exec sessions remained open during debugging
- direct instrumentation probes were at one point run in parallel with wrapper-based runs
- stale `multiinstance.lock` existed for the AVD
- emulator state occasionally disappeared from ADB during a run

Any future analysis should account for the fact that not every historical failure in this branch came from the same runtime state quality.

## Current Repro Snapshot

The cleanest current repro is:

```sh
cd mobile
npm run screenshots -- --testNamePattern '01_auth_login'
```

Observed output pattern in that clean repro:

1. wrapper logs device ready
2. wrapper logs reverse ready
3. Detox prints manual-launch prompt
4. wrapper logs dynamic Detox reverse
5. wrapper launches instrumentation
6. Detox finds app PID
7. readiness fails
8. Android instrumentation reports `Process crashed`

## Current Analytical Position

The branch is no longer in the “missing route / broken drawer / bad auth landing” phase.

The analysis position at handoff time is:

- the manual-launch pipeline is implemented
- the wrapper has had several real bugs corrected
- the remaining blocker is observed at the instrumentation / app-ready boundary
- emulator instability still exists as a separate contaminant and should not be conflated with the clean readiness-crash repro

## No-Resolution Constraint

This handoff intentionally avoids prescribing a fix path.

It is meant to transfer:

- branch-state intent
- exact evidence already gathered
- runtime symptoms
- certainty-ranked assumptions

without steering the next agent toward a specific remediation strategy.
