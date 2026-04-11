---
doc_type: handoff
id: maestro-screenshot-auth-bypass-handoff
title: Maestro Screenshot Auth Bypass — Handoff
status: in-progress
owner: mobile
created: 2026-04-11
plan: docs/plans/maestro-screenshot-auth-bypass-plan.md
---

# Maestro Screenshot Auth Bypass — Handoff

## Why this handoff exists

Implementation of `docs/plans/maestro-screenshot-auth-bypass-plan.md` was paused
after Task 5 (seed-and-run.sh) was written but before it was smoke-tested or
before the remaining tasks (6–11) were started. This document captures the
exact state of the work so a follow-up session can resume without re-deriving
context.

## User-locked decisions (immutable)

1. Landing route after bootstrap: `/inbox` (chosen to bypass the
   `/home → /inbox` redirect chain discovered in
   `mobile/app/(tabs)/home/index.tsx`, which is a pure `<Redirect href="/inbox" />`).
2. `mobile/maestro/visual-audit.yaml` must be deleted after the new flows land
   green — pre-authorized, no further confirmation needed.
3. E2E gate scope: runtime-only (`process.env.EXPO_PUBLIC_E2E_MODE !== '1'`
   early-return inside the `useEffect`). No babel plugin, no build-time
   exclusion.
4. User authorized working through the full task list without asking between
   tasks; report at the end.

## Completed tasks (verified)

### 1. Plan doc updated with locked decisions

- File: `docs/plans/maestro-screenshot-auth-bypass-plan.md`
- Current State section now describes the `/ → /home → /inbox` redirect chain
  and locks `/inbox` as the landing route.
- All `/home` and "configurable via env var" references removed.
- The flow description now says "launchApp + openLink: ${SEED_BOOTSTRAP_URL}"
  (see "Implementation deviation" section below).

### 2. Go seeder — Auth block exposed

- File: `scripts/e2e_seed_mobile_p2.go`
- Added `Auth { Token, UserID, WorkspaceID }` block to `seedOutput`
  (lines around 67–95).
- Populated from the existing `authResponse` in `main()`.
- File: `scripts/e2e_seed_mobile_p2_test.go`
- Added `TestSeedOutputExposesAuthBlock` — pure JSON-shape test, no DB.
- **Verified**: `go test ./scripts/... -run TestSeedOutputExposesAuthBlock -v` passes.
- **Verified**: `go vet ./scripts/...` clean.

### 3. e2e-bootstrap runtime gate

- File: `mobile/app/e2e-bootstrap.tsx`
- Added `EXPO_PUBLIC_E2E_MODE !== '1'` guard at the top of the `useEffect`,
  inside the `hasBootstrapped.current` check (so React Strict Mode double-run
  does not double-navigate).
- When gate is off: `router.replace('/login')` is called, `login()` is never
  invoked, no auth state is mutated.
- File: `mobile/__tests__/app/e2e-bootstrap.test.tsx` (NEW)
- 4 tests covering: gate off, gate unset, gate on success, gate on missing params.
- **Verified**: `npx jest __tests__/app/e2e-bootstrap.test.tsx` → 4/4 pass.
- Test file uses `globalThis.__bootstrapMocks` registry pattern to satisfy
  Jest's out-of-scope factory restriction.

### 4. auth-surface.yaml created

- File: `mobile/maestro/auth-surface.yaml` (NEW)
- Launches app, waits for `login-screen`, captures `01_auth_login.png`.
- No `inputText`, no `tapOn` on login fields.
- **Not yet verified on device.**

### 5. authenticated-audit.yaml created

- File: `mobile/maestro/authenticated-audit.yaml` (NEW)
- Starts with `launchApp: { clearState: false, stopApp: true }` then
  `openLink: ${SEED_BOOTSTRAP_URL}`.
- Waits for `inbox-screen`, captures `02_inbox.png`.
- Reuses the original visual-audit.yaml capture steps for:
  - `06_inbox_approval_inline` (conditional)
  - `07_inbox_handoff` (conditional)
  - `03_support_case_detail`
  - `04_sales_brief`
  - `08_activity_run_detail_denied` (via `openLink fenixcrm:///activity/...`)
  - `05_governance` (via `openLink fenixcrm:///governance`)
- **Not yet verified on device.**

## In-progress: Task 5 — seed-and-run.sh

### State

- File: `mobile/maestro/seed-and-run.sh` — **fully rewritten and saved**.
- `bash -n` syntax check: **PASS**.
- Functional smoke test of `url_encode` helper: **NOT RUN** (session paused
  before execution).

### Concrete changes in the rewrite

- Two flow constants: `AUTH_SURFACE_FLOW`, `AUTHED_AUDIT_FLOW`.
- New `REPORTS_DIR` at `mobile/artifacts/maestro-reports/`, separated from
  `OUTPUT_DIR` at `mobile/artifacts/screenshots/`.
- New helper `url_encode()` using Node `encodeURIComponent` via `process.argv`.
- New helper `compose_bootstrap_url()` — hard-codes landing route to `/inbox`,
  encodes token / userId / workspaceId / redirect, emits
  `fenixcrm:///e2e-bootstrap?token=...&userId=...&workspaceId=...&redirect=...`.
- `seed_to_env_lines` Node block:
  - Added `SEED_AUTH_TOKEN / SEED_USER_ID / SEED_WORKSPACE_ID`.
  - **Removed `SEED_PASSWORD`** — login UI no longer part of critical path.
- `print_seed_summary`:
  - Removed `SEED_PASSWORD` line entirely.
  - Added `SEED_AUTH_TOKEN=[redacted len=N]` redaction.
- New `run_maestro_flow()` wrapper — passes `SEED_BOOTSTRAP_URL` to Maestro as
  a single `-e` var. **Does NOT pass SEED_AUTH_TOKEN or SEED_PASSWORD to
  Maestro** (secrets stay inside the composed URL only).
- `main()` orchestration order:
  1. Seed fixtures.
  2. Parse seed to env.
  3. Hard-fail if `SEED_AUTH_TOKEN / SEED_USER_ID / SEED_WORKSPACE_ID` missing.
  4. Compose `SEED_BOOTSTRAP_URL` and export it.
  5. `print_seed_summary`.
  6. `adb reverse` networking.
  7. `pm clear` + `am start` + `wait_for_react_native_ready` (Phase 1 prep).
  8. `rm -rf OUTPUT_DIR REPORTS_DIR` and recreate.
  9. Run `auth-surface.yaml`.
  10. Run `authenticated-audit.yaml`.
  11. `copy_reports_screenshots` — walks `REPORTS_DIR/**.png` and copies to `OUTPUT_DIR`.

### Smoke tests pending on this file

- `url_encode` — verify JWT chars `.`, `+`, `/`, `=` encode correctly:
  ```
  source <(sed -n '/^url_encode()/,/^}/p' mobile/maestro/seed-and-run.sh)
  url_encode "eyJhbGciOi.JI/UzI1+NiIs="
  # expected: eyJhbGciOi.JI%2FUzI1%2BNiIs%3D
  ```
- `compose_bootstrap_url` against fake seed env vars — verify the full URL
  shape matches what `authenticated-audit.yaml` expects.
- `seed_to_env_lines` against a fixture JSON blob — verify `SEED_AUTH_TOKEN`,
  `SEED_USER_ID`, `SEED_WORKSPACE_ID` appear and `SEED_PASSWORD` does not.

## Implementation deviation worth flagging

The plan originally said Phase 2 would use `launchApp` with a `url:` field.
During Task 4 implementation I changed this to `launchApp` (no url) +
`openLink: ${SEED_BOOTSTRAP_URL}` as two separate Maestro commands.

**Why**: `launchApp.url` forces a cold re-launch, which would trigger a second
`wait_for_react_native_ready` race. `openLink` reuses the already-warm RN
runtime from Phase 1 and is the canonical Maestro deep-link command. The plan
doc was updated to reflect this.

**Risk**: `openLink` requires the Android activity to accept the deep link via
an intent filter. `mobile/app.json` declares `scheme: fenixcrm`, and the
existing `visual-audit.yaml` already used `openLink: fenixcrm:///...` for
Activity and Governance captures successfully, so this risk is low. But it is
unverified for the `e2e-bootstrap` route specifically.

## Pending tasks (not started)

6. **Delete `mobile/maestro/visual-audit.yaml`** — only after Task 11 end-to-end
   run passes. This is pre-authorized.
7. **Update screenshot runbook and migration docs**. Likely files to touch:
   - `docs/plans/maestro-screenshot-migration.md` (superseded by current plan).
   - Any `README.md` under `mobile/` that mentions `visual-audit.yaml`.
   - `docs/tasks/` entries referencing the old flow name.
   Search first with `Grep pattern="visual-audit"`.
8. **Governance verification** — build mobile without `EXPO_PUBLIC_E2E_MODE=1`,
   launch `fenixcrm:///e2e-bootstrap?token=x&userId=y&workspaceId=z` and
   confirm no auth state mutation and redirect to `/login`. The Jest test
   (Task 3) already covers the unit-level behavior; this is the full
   integration check.
9. **Secret leakage audit** — after running Task 11 end-to-end, grep
   `mobile/artifacts/maestro-reports/` for the seeded token and password to
   confirm nothing plaintext leaked. The expected token lives only inside the
   URL-encoded `SEED_BOOTSTRAP_URL` and should not appear in any Maestro
   report JSON/HTML.
10. **QA gates** — run:
    - `bash scripts/check-no-inline-eslint-disable.sh`
    - `cd mobile && npm run typecheck`
    - `cd mobile && npm run lint`
    - `cd mobile && npm run quality:arch`
    - `cd mobile && npm run test:coverage`
    - `bash scripts/qa-mobile-prepush.sh` (preferred wrapper)
    - `bash scripts/qa-go-prepush.sh` (because `scripts/e2e_seed_mobile_p2.go` changed)
11. **End-to-end `npm run screenshots`** — verify the 8 PNGs land
    deterministically:
    - `01_auth_login`, `02_inbox`, `03_support_case_detail`, `04_sales_brief`,
      `05_governance`, `06_inbox_approval_inline`, `07_inbox_handoff`,
      `08_activity_run_detail_denied`.
    Requires a booted Android emulator with the debug APK installed.

## Files changed so far (for the eventual commit)

- `docs/plans/maestro-screenshot-auth-bypass-plan.md` — rewritten and locked.
- `scripts/e2e_seed_mobile_p2.go` — Auth block added to seedOutput.
- `scripts/e2e_seed_mobile_p2_test.go` — TestSeedOutputExposesAuthBlock added.
- `mobile/app/e2e-bootstrap.tsx` — runtime E2E gate.
- `mobile/__tests__/app/e2e-bootstrap.test.tsx` — NEW, 4 tests.
- `mobile/maestro/auth-surface.yaml` — NEW.
- `mobile/maestro/authenticated-audit.yaml` — NEW.
- `mobile/maestro/seed-and-run.sh` — rewritten.
- `mobile/maestro/visual-audit.yaml` — **still present**, to be deleted in Task 6.

## Unrelated dirty files already in working tree (do NOT include in commit)

These were modified before this task started and are part of different work:

- `mobile/__tests__/navigation.test.tsx`
- `mobile/app/(tabs)/_layout.tsx`
- `mobile/app/(tabs)/activity/[id].tsx`
- `mobile/app/(tabs)/sales/[id].tsx`
- `mobile/app/(tabs)/sales/[id]/brief.tsx`
- `mobile/app/(tabs)/sales/deal-[id].tsx`
- `mobile/app/(tabs)/support/[id].tsx`

When committing this work, stage files explicitly by name — do not use
`git add -A`.

## How to resume

1. Read this file end-to-end.
2. Read the plan: `docs/plans/maestro-screenshot-auth-bypass-plan.md`.
3. Run the three pending `seed-and-run.sh` smoke tests from the "Smoke tests
   pending on this file" section.
4. Proceed with Tasks 6 → 11 in order.
5. Final commit: stage only the files in "Files changed so far" plus this
   handoff. Do not include the unrelated dirty files.
