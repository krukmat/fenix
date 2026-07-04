---
doc_type: task
id: EXTVAL-BATTERY-T7-RERUN-001
title: "Rerun T7 mobile real-mode validation after Android recovery and approvals crash fix"
status: complete
phase: external-validation-rerun
week: "2026-W27"
tags: [external-validation, battery, mobile, t7, rerun, android]
fr_refs: [FR-230, FR-232]
uc_refs: [UC-C1]
blocked_by: [EXTVAL-ANDROID-AVD-REPAIR-001]
blocks: []
files_affected: []
criticality: standard
criticality_basis: "Operational validation task against a repaired environment and a fix-bearing mobile build. No planned product-code changes."
created: 2026-07-04
completed: 2026-07-05
---

# Task EXTVAL-BATTERY-T7-RERUN-001

**Plan**: [External Validation Android Recovery and T7 Revalidation Plan](../plans/external_validation_android_recovery_plan.md)

## Task Card

Task: EXTVAL-BATTERY-T7-RERUN-001

Task file: docs/tasks/task_extval_t7_real_mode_rerun.md

Plan file: docs/plans/external_validation_android_recovery_plan.md

Summary: Rerun the real Android validation path for `T7` now that the mobile approvals crash fix is in place and the `fenix_t7` emulator can boot again, proving whether Home survives real login and whether the deferred mobile navigation criteria now pass.

Code affected: None expected. Runtime validation only across backend, BFF, emulator, and the existing mobile build.

Criticality: standard

Criticality basis: Operational validation task against a repaired environment and a fix-bearing mobile build. No planned product-code changes.

Effort/reasoning: Medium - requires coordinated restart of runtime services, real login through the visible UI, navigation across multiple mobile surfaces, and correlation with backend/BFF truth.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Estimated tokens: ~7000

Peer readiness review approval: reviewer=claude; artifact=logs/peer-workflow-review/task-readiness_codex_by_claude_20260704T202344Z.json; status=PASS

Task type: operational validation. No dev-task pseudocode required.

## Acceptance Criteria

1. The app boots to the real `/login` screen on `fenix_t7` with `EXPO_PUBLIC_E2E_MODE` unset or explicitly non-`1`.
2. Login succeeds through the visible UI.
3. Home no longer crashes after the real approvals fetch.
4. Support, Inbox, Activity, Sales Brief, and Governance are reachable from the logged-in app.
5. The rerun records whether `GET /api/v1/signals` for the workspace owner still fails with `403`.
6. If the UI path reaches a case detail screen, triggering the support agent produces a state refresh traceable to real backend/BFF activity.

## Rerun Result (2026-07-05)

Environment: backend and BFF restarted from `.env.external-validation` (provides
`JWT_SECRET`, `SCREENSHOT_MODE=false`, `ENABLE_SCREENSHOT_FIXTURES=false`).
Metro started with `EXPO_PUBLIC_E2E_MODE=0`. Emulator `fenix_t7` reused from
`EXTVAL-ANDROID-AVD-REPAIR-001` (already booted, `adb devices` showed `device`).

1. **PASS** — App booted to the real `/login` screen with `EXPO_PUBLIC_E2E_MODE=0`
   after Metro connected (initial install without Metro failed with the expected
   "Unable to load script" release-bundle error; resolved by running Metro and
   force-stopping/relaunching the app).
2. **PASS** — Registered a fresh workspace owner via `POST /bff/auth/register`
   (`extval.t7.rerun.20260705@fenixcrm.test`, workspace
   `019f2f2b-9971-7f81-7b13-1d5b5702fba2`), then typed the real credentials into
   the visible login form (`login-email-input`, `login-password-input`) and
   tapped `login-submit-button`. Backend log confirms
   `POST /auth/login ... 200` at `00:11:30`.
3. **PASS** — Home rendered "No items" empty state with no crash and no error
   boundary. Bottom nav (Inbox, Support, Sales, Activity, Governance) rendered
   correctly. Backend log shows the real approvals fetch succeeding:
   `GET /api/v1/approvals?...` → `200`.
4. **PASS** — Support, Inbox, Activity, Sales (Accounts/Deals/Leads/Contacts),
   and Governance all reached via bottom-nav taps with no crash on any surface.
5. **REPRODUCED** — `GET /api/v1/signals?workspace_id=...` still returns `403`
   for the freshly-registered workspace owner, matching the prior T7 run's
   Finding (tracked separately under `O7`). Confirmed at `00:11:30` and again at
   `00:16:42` post-navigation.
6. **PASS** — Created a case via `POST /api/v1/cases` (id
   `019f2f32-26be-7e95-022d-20b114d8204d`, since the fresh workspace had no
   seed data to reach case-detail through), reached it via real UI navigation
   (Support tab → case list item → case detail), and tapped the visible
   "Run Support Agent" button. Backend log:
   `POST /api/v1/agents/support/trigger ... 201` at `00:17:56`, immediately
   followed by an automatic UI refetch of
   `GET /api/v1/agents/runs?...status=awaiting_approval`. Verified the run via
   API: run `019f2f35-b2f0-70bc-52c6-71a6f1b5d7f6`,
   `status=handed_off`/`runtime_status=escalated`, referencing
   `CaseID=019f2f32-26be-7e95-022d-20b114d8204d`, with a populated reasoning
   trace and a `create_task` tool call — a real, traceable backend/BFF state
   change, not a UI-only optimistic update.

**Conclusion**: The mobile approvals crash fix is validated in real mode — Home
no longer crashes after a real approvals fetch. The `GET /api/v1/signals` 403
for the workspace owner is reproduced again and remains the next mobile rerun
blocker under `O7`, per the plan's decision tree ("If Home succeeds but
`signals` still 403s").
