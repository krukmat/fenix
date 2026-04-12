---
name: Maestro Screenshot Auth Bypass Plan
status: review-draft
owner: mobile
supersedes: docs/plans/maestro-screenshot-migration.md
created: 2026-04-11
---

# Make `npm run screenshots` deterministic by removing login UI from the critical path

## Status Update

- Green emulator run completed with the expected 8 screenshots.
- The inbox screenshot seed now produces an interleaved mixed queue with 2 approvals,
  2 handoffs, 2 active signals, and 2 rejected runs so `02_inbox` shows visible
  variety without relying on scrolling.
- `mobile/maestro/visual-audit.yaml` has been retired in favor of the two-phase
  flow: `auth-surface.yaml` + `authenticated-audit.yaml`.
- `mobile/maestro/seed-and-run.sh` now launches the app via ADB for phase 1 and
  sanitizes `mobile/artifacts/maestro-reports/` so the bootstrap JWT is not
  retained in report artifacts.

## Summary

`npm run screenshots` currently hangs on Maestro `inputText` against the password
field of `login-screen` and times out with `DEADLINE_EXCEEDED`. The root cause is
a soft-keyboard / secure-entry race on the emulator build, not `inputText` itself.
The fix is to stop driving authentication through the login UI in the screenshot
critical path, and instead inject authenticated session state via an ADB deep link
into the existing E2E bootstrap route.

The run will be split into two orchestrated phases:

- Phase 1 — Auth surface capture: cold launch the app, wait for `login-screen`,
  capture `01_auth_login`. No text input, no submit.
- Phase 2 — Authenticated audit: authenticate outside Maestro by launching the
  `e2e-bootstrap` deep link, then run only the authenticated screenshot flow
  starting from the configured landing screen.

Login UI functional validation is intentionally removed from this command and
remains covered by the existing Jest + mobile unit suites.

## Current State (verified)

- Seed producer: `scripts/e2e_seed_mobile_p2.go` (already performs
  login-or-register and holds an internal `authResponse` struct with
  `Token / UserID / WorkspaceID`, but does not expose them in `seedOutput`).
- Seed consumer: `mobile/maestro/seed-and-run.sh`, `seed_to_env_lines` Node block
  (lines 97–122).
- Maestro flows:
  - `mobile/maestro/auth-surface.yaml`
  - `mobile/maestro/authenticated-audit.yaml`
- Bootstrap route: `mobile/app/e2e-bootstrap.tsx` — accepts
  `token / userId / workspaceId / redirect`, defaults redirect to `/home`, and is
  gated by `EXPO_PUBLIC_E2E_MODE`.
- Deep-link scheme: `fenixcrm` (confirmed in `mobile/app.json`).
- Authenticated landing route chain (verified in code):
  `/` → `/home` → `/inbox`. `mobile/app/index.tsx` redirects authenticated
  users to `/home`, and `mobile/app/(tabs)/home/index.tsx` is itself a
  `Redirect href="/inbox"` component (see comment "W2-T2: /home → /inbox
  redirect, Home feed migrated to the Inbox wedge tab"). The actual landing
  testID for an authenticated session is `inbox-screen`, defined in
  `mobile/src/components/inbox/InboxFeed.tsx`.
- Screenshot runner strategy: pass `redirect=/inbox` directly to skip the
  `/home` hop. Chain-equivalent but one fewer redirect bounce.

## Implementation Changes

### Flow file layout (rename, not just add)

- Delete `mobile/maestro/visual-audit.yaml` after its two successors land green
  on the target emulator. Deletion was pre-authorized by the user on the
  decision-lock step of this plan.
- Create `mobile/maestro/auth-surface.yaml`:
  - No `launchApp`; the runner foregrounds the app via ADB first.
  - `extendedWaitUntil: id: login-screen`.
  - `assertVisible: id: login-screen`.
  - `takeScreenshot: 01_auth_login`.
  - No `inputText`, no `tapOn` on any login field.
- Create `mobile/maestro/authenticated-audit.yaml`:
  - First step: `openLink: ${SEED_BOOTSTRAP_URL}`. The shell script composes the
    full URL-encoded deep link and passes it as a single Maestro env var.
    `launchApp` was removed after proving unstable in the dev-client path; the
    RN runtime from Phase 1 is reused directly.
  - Then: `extendedWaitUntil` on `inbox-screen` (the real landing testID after
    the authenticated redirect chain resolves).
  - Then: `takeScreenshot: 02_inbox`.
  - Then: the existing Support / Sales Brief / Activity / Governance capture
    steps from `visual-audit.yaml`, unchanged.

### Runner orchestration

Update `mobile/maestro/seed-and-run.sh`:

- Replace the single `FLOW_FILE` constant with two paths:
  - `AUTH_SURFACE_FLOW="${SCRIPT_DIR}/auth-surface.yaml"`
  - `AUTHED_AUDIT_FLOW="${SCRIPT_DIR}/authenticated-audit.yaml"`
- Orchestration order:
  1. Seed deterministic fixtures (unchanged).
  2. Parse the seeder JSON, including the new `auth` block (see next section).
  3. `adb reverse` (unchanged).
  4. `adb shell pm clear ${APP_ID}` → ADB foreground launch for phase 1.
  5. Run `auth-surface.yaml` via Maestro, capturing only `01_auth_login`.
  6. Run `authenticated-audit.yaml` via Maestro. `openLink` dispatches the
     bootstrap deep link into the already-running app. No extra `pm clear`
     between phases — we want the same RN runtime, just a new authenticated
     session.
  7. Normalize PNG output into `mobile/artifacts/screenshots/` (unchanged).
- Do **not** pass `SEED_PASSWORD` to Maestro. Remove the `-e SEED_PASSWORD=...`
  line entirely.
- Do **not** log `SEED_AUTH_TOKEN` in `print_seed_summary`. Log a `[redacted]`
  placeholder with the token length only.
- Maestro test reports must be written to a directory distinct from the final
  reviewable screenshot set so report artifacts can be excluded from any
  reviewer-facing bundle.

### Auth bootstrap contract

Producer side — `scripts/e2e_seed_mobile_p2.go`:

- Extend `seedOutput` with a new `Auth` block:
  ```
  Auth struct {
      Token       string `json:"token"`
      UserID      string `json:"userId"`
      WorkspaceID string `json:"workspaceId"`
  } `json:"auth"`
  ```
- Populate it from the existing `authResponse` returned by `loginOrRegister`.
- Update `scripts/e2e_seed_mobile_p2_test.go` to assert the new block is present
  and non-empty, and to assert `credentials.password` is still emitted for
  non-screenshot consumers of the seeder.

Consumer side — `mobile/maestro/seed-and-run.sh` `seed_to_env_lines`:

- Add three new env mappings:
  - `SEED_AUTH_TOKEN      = seed.auth?.token`
  - `SEED_USER_ID         = seed.auth?.userId`
  - `SEED_WORKSPACE_ID    = seed.auth?.workspaceId`
- Add a derived mapping:
  - `SEED_LANDING_ROUTE   = "/inbox"` — hard-coded. Bypasses the `/home → /inbox`
    redirect hop and lands directly on the real authenticated content screen.
    No env var override; intentional simplicity.
- URL-encode `SEED_AUTH_TOKEN`, `SEED_USER_ID`, `SEED_WORKSPACE_ID`, and
  `SEED_LANDING_ROUTE` before composing the deep link. Implementation: a Node
  one-liner (`node -e 'process.stdout.write(encodeURIComponent(process.argv[1]))' "$v"`)
  wrapped in a shell helper `url_encode`. JWTs contain `.`, `+`, `/`, `=`, so
  unencoded values will break the Intent parser.
- The fully composed deep link is exported as `SEED_BOOTSTRAP_URL` and passed to
  Maestro as a single `-e SEED_BOOTSTRAP_URL=...` variable, which the
  `authenticated-audit.yaml` flow references via `${SEED_BOOTSTRAP_URL}` in its
  `launchApp.url`. This keeps URL-encoding logic out of Maestro YAML.

### App-side hardening

- Update `mobile/app/e2e-bootstrap.tsx`:
  - Add a runtime gate: if `process.env.EXPO_PUBLIC_E2E_MODE !== '1'`, the route
    must render nothing and redirect immediately to `/login` without calling
    `login()` or mutating any auth state. This prevents a production build from
    shipping an unconditional auth-injection surface — a direct requirement of
    the governance principles in `CLAUDE.md` ("Tools, not mutations").
  - Keep the existing accepted params: `token`, `userId`, `workspaceId`,
    `redirect`. Default `redirect` remains `/home`.
  - If any of `token`, `userId`, `workspaceId` are missing, redirect to `/login`
    without mutating auth state (already true today).
- No other app-side changes. Screenshot orchestration stays in the shell script.

### Authenticated screenshot flow content

`authenticated-audit.yaml` must:

- Begin with `launchApp.url: ${SEED_BOOTSTRAP_URL}`.
- Then `extendedWaitUntil` on `home-screen` (the authenticated landing route
  `/home`).
- Navigate to Inbox via `tab-inbox` for the `02_inbox` capture.
- Not call any `inputText` against the login form. `inputText` remains allowed
  elsewhere (e.g., notes, filters) if a future step legitimately needs it — the
  rule is "no login UI interaction in the screenshot critical path", not "no
  `inputText` anywhere".
- Reuse the existing stable `testID` contract for navigation and capture:
  - `inbox-screen` (landing verification after bootstrap)
  - `tab-support`
  - `support-cases-list-item-0`
  - `tab-sales`
  - `sales-tab-deals`
  - `sales-deal-item-0`
  - `sales-deal-brief-button`
  - `activity-run-detail-screen` (via `openLink` on
    `fenixcrm:///activity/${SEED_RUN_DENIED_ID}`)
  - `governance-screen` (via `openLink` on `fenixcrm:///governance`)
- Preserve the existing conditional captures for inbox inline surfaces when
  present:
  - `06_inbox_approval_inline` (when `inbox-approval-${SEED_APPROVAL_ID}` visible)
  - `07_inbox_handoff` (when `inbox-handoff-${SEED_RUN_HANDOFF_ID}` visible)

### Artifacts and documentation

- Keep final PNGs in `mobile/artifacts/screenshots/` (unchanged public path).
- Write Maestro JSON/HTML reports to `mobile/artifacts/maestro-reports/` so they
  can be excluded from reviewer-facing bundles and CI screenshot uploads.
- Tokens and passwords must not appear in:
  - `seed-and-run.sh` stdout (`print_seed_summary` must redact).
  - Maestro reports — achieved by passing secrets only via
    `SEED_BOOTSTRAP_URL` (which is itself a URL-encoded, scoped E2E-only value)
    and never via raw `SEED_PASSWORD` / `SEED_AUTH_TOKEN` env vars.
- Update the screenshot runbook and migration docs to state explicitly:
  - Login UI is no longer part of the screenshot critical path.
  - Login UI behavior is covered by mobile unit tests and manual QA, not by
    `npm run screenshots`.
  - `e2e-bootstrap` is E2E-mode-only and inert in production builds.

## Interfaces and Public Contracts

- **Seeder output**: gains an `auth` object with `token`, `userId`,
  `workspaceId`. Existing fields unchanged. Consumers other than the screenshot
  runner remain compatible.
- **`e2e-bootstrap` route**: same query params, now gated by
  `EXPO_PUBLIC_E2E_MODE=1`. Non-E2E behavior becomes "redirect to `/login`
  without mutation".
- **`package.json`**: `screenshots` script still maps to
  `bash maestro/seed-and-run.sh`. No new npm script.
- **`npm run screenshots`**: becomes a visual capture command only. It is not a
  login interaction validation command.

## Alternatives Considered

- **Alt A — two-phase orchestration with `adb shell am start`** (original
  draft): keeps auth injection outside Maestro entirely. Rejected because (1)
  Maestro's `launchApp.url` is a first-class feature that does the same thing
  with less shell glue, (2) a bare `am start` creates a second RN-ready race
  that the runner would need to handle with another `wait_for_react_native_ready`
  loop, and (3) Maestro's test recording does not see the `am start` event, so
  the report timeline becomes harder to read.
- **Alt B — a dedicated E2E-only backend endpoint that returns a one-shot
  session token**: cleaner governance boundary, but requires a new HTTP handler
  and new audit surface. Out of scope for fixing a flaky screenshot runner; the
  seeder already produces the token we need.
- **Alt C — fix the `inputText` race itself**: possible, but fragile — the
  failure mode depends on emulator build, soft-keyboard timing, and secure-entry
  flags. Even if fixed, the login UI remains a slow, brittle critical path for
  visual capture work that does not benefit from it.

## Test Plan

Functional verification:

1. `go test ./scripts/...` — new assertions on `seedOutput.auth` presence.
2. `node -e` dry run of `seed_to_env_lines` against a fixture seed JSON,
   asserting `SEED_AUTH_TOKEN`, `SEED_USER_ID`, `SEED_WORKSPACE_ID` appear and
   that URL-encoded characters are preserved in the composed
   `SEED_BOOTSTRAP_URL`.
3. `auth-surface.yaml` against a clean app state produces only
   `01_auth_login.png` and does not touch login inputs.
4. `authenticated-audit.yaml` against the same device lands deterministically
   on `inbox-screen` after `launchApp.url` and completes without `inputText`
   against login fields.
5. `npm run screenshots` end-to-end produces this final set:
   - `01_auth_login`
   - `02_inbox`
   - `03_support_case_detail`
   - `04_sales_brief`
   - `05_governance`
   - `06_inbox_approval_inline` (when fixture present)
   - `07_inbox_handoff` (when fixture present)
   - `08_activity_run_detail_denied`

Governance / safety verification:

6. Build the app with `EXPO_PUBLIC_E2E_MODE` unset. Launch
   `fenixcrm:///e2e-bootstrap?token=x&userId=y&workspaceId=z` and confirm the
   app redirects to `/login` **without** calling `login()` or persisting any
   token. This protects the production build from an auth-injection surface.
7. Inspect `mobile/artifacts/maestro-reports/` JSON/HTML after a full run and
   confirm no plaintext token, password, or bearer value appears.
8. `print_seed_summary` stdout contains no token material — only lengths and
   ids safe for logs.

Functional coverage of the login UI (not regressed):

9. Confirm the existing mobile unit suite still covers login form submission,
   error states, and success path. If coverage gaps exist after removing login
   UI from the screenshot flow, open a follow-up task — do not silently ship
   reduced coverage.

Pre-push gates (mandatory per CLAUDE.md):

- `bash scripts/check-no-inline-eslint-disable.sh`
- `cd mobile && npm run typecheck`
- `cd mobile && npm run lint`
- `cd mobile && npm run quality:arch`
- `cd mobile && npm run test:coverage`
- preferred: `bash scripts/qa-mobile-prepush.sh`
- On the Go side (because the seeder changes): `bash scripts/qa-go-prepush.sh`.

## Assumptions and Defaults

- Screenshot execution will use full login bypass after capturing the login
  surface.
- Login UI validation is handled outside `npm run screenshots`, via existing
  mobile unit tests and manual QA.
- `/inbox` is the authenticated landing route for screenshots. This matches
  where the app's own redirect chain (`/` → `/home` → `/inbox`) lands in
  practice. Bypassing the `/home` hop is an intentional simplification.
- The seeder is the source of truth for bootstrap auth data. The script will
  not perform a second login request.
- `EXPO_PUBLIC_E2E_MODE=1` is already set by the screenshot runner environment
  (confirmed present in `mobile/.env` and `mobile/.env.e2e`). Production builds
  must never ship with this flag enabled.
- The priority is deterministic screenshot generation, not end-to-end
  validation of the login form inside the screenshot command.

## Rollback

- Removing `visual-audit.yaml` is the only destructive change. It will be
  deleted only after `auth-surface.yaml` + `authenticated-audit.yaml` land and
  a full green run is observed on the target emulator.
- If the new flow regresses, revert the three edits to `seed-and-run.sh`,
  `e2e-bootstrap.tsx`, and `e2e_seed_mobile_p2.go`, and restore
  `visual-audit.yaml` from Git history. The old flow remains a valid fallback.
