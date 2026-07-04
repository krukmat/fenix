---
doc_type: task
id: EXTVAL-ANDROID-AVD-REPAIR-001
title: "Repair Android SDK/AVD state so T7 real-mode validation can run again"
status: complete
phase: external-validation-rerun
week: "2026-W27"
tags: [external-validation, android, avd, emulator, environment]
fr_refs: [FR-230, FR-232]
uc_refs: [UC-C1]
blocked_by: []
blocks: [EXTVAL-BATTERY-T7-001]
files_affected:
  - ~/.android/avd/fenix_t7.avd/config.ini
  - /Users/matias/Library/Android/sdk/system-images/android-34/google_apis/arm64-v8a/
  - ~/.zshrc
criticality: standard
criticality_basis: "Environment-repair task. No product-code changes, but it controls whether manual real-mode validation can resume."
created: 2026-07-04
completed: 2026-07-04
---

# Task EXTVAL-ANDROID-AVD-REPAIR-001

**Plan**: [External Validation Android Recovery and T7 Revalidation Plan](../plans/external_validation_android_recovery_plan.md)

## Task Card

Task: EXTVAL-ANDROID-AVD-REPAIR-001

Task file: docs/tasks/task_extval_android_avd_repair.md

Plan file: docs/plans/external_validation_android_recovery_plan.md

Summary: Repair the local Android SDK and `fenix_t7` AVD state so the emulator boots again, proving the environment can support the pending real-mode T7 rerun after the mobile approvals crash fix.

Code affected: Local Android environment only. Expected touch points are the SDK system-image installation and, if needed, recreation of the `fenix_t7` AVD. No product source files are expected to change.

Criticality: standard

Criticality basis: Environment-repair task. No product-code changes, but it controls whether manual real-mode validation can resume.

Effort/reasoning: Medium - requires reconciling SDK root, reinstalling or repairing the missing Android 34 arm64 Google APIs system image, and proving the emulator boots cleanly before handing back to T7.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Estimated tokens: ~5000

Peer readiness review approval: reviewer=local-gemma; artifact=logs/peer-workflow-review/task-readiness_codex_by_local-gemma_20260704T201536Z.json; status=PASS

Task type: environment repair. No dev-task pseudocode required.

## Acceptance Criteria

1. The Android 34 Google APIs arm64 system image exists under the active SDK root.
2. `fenix_t7` either boots successfully as-is or is recreated and then boots successfully.
3. `adb devices` shows the emulator as `device`.
4. `adb shell getprop sys.boot_completed` returns `1`.
5. The task records the exact environment state needed for the follow-up T7 rerun.

## Execution Update (2026-07-04)

Findings:

- The user SDK root at `/Users/matias/Library/Android/sdk` is incomplete for the
  `fenix_t7` AVD: it contains `system-images/android-36/...` but not the
  `android-34/google_apis/arm64-v8a` image referenced by
  `~/.android/avd/fenix_t7.avd/config.ini`.
- The Homebrew-managed SDK root at
  `/opt/homebrew/share/android-commandlinetools` does contain the required
  Android 34 arm64 Google APIs system image and matching emulator/platform
  tools.
- The AVD itself was not broken structurally; the active SDK root was.

Repair performed:

- Launched `fenix_t7` successfully by setting:
  - `ANDROID_SDK_ROOT=/opt/homebrew/share/android-commandlinetools`
  - `ANDROID_HOME=/opt/homebrew/share/android-commandlinetools`
- Persisted that SDK root in `~/.zshrc` so future interactive shells resolve:
  - `ANDROID_SDK_ROOT`
  - `ANDROID_HOME`
  - `adb`
  - `emulator`
  without manual overrides.

Verification:

- `find /opt/homebrew/share/android-commandlinetools/system-images -maxdepth 3 -type d`
  shows `android-34/google_apis/arm64-v8a`.
- `ANDROID_SDK_ROOT=/opt/homebrew/share/android-commandlinetools ... emulator -avd fenix_t7`
  starts successfully and resolves the system path.
- `adb devices -l` shows `emulator-5554 device`.
- `adb shell getprop sys.boot_completed` returns `1`.
- `zsh -ic 'which adb; which emulator; echo $ANDROID_SDK_ROOT; echo $ANDROID_HOME'`
  resolves to the Homebrew SDK root and binaries.

Outcome:

- Acceptance criteria 1-5 are satisfied.
- `fenix_t7` is bootable again on this host.
- No AVD recreation was required.
