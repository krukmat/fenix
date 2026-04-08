---
doc_type: task
id: F4-T3
title: "Update functional mobile E2E coverage for follow-up wedge flows"
status: blocked
phase: F4
week: 8
tags: [task, mobile, wedge, e2e, followup]
fr_refs: [FR-071, FR-092, FR-230, FR-232]
uc_refs: [UC-A7, UC-C1, UC-S1, UC-G1]
blocked_by: [F2-T5, F3-T3, F4-T1]
blocks: [F4-T4]
files_affected:
  - mobile/.detoxrc.js
  - mobile/e2e
  - mobile/package.json
  - scripts/e2e_seed_mobile_p2.go
created: 2026-04-08
completed:
parent_plan: docs/plans/mobile_wedge_harmonization_followup_plan.md
---

# F4-T3 — Update functional mobile E2E coverage for follow-up wedge flows

## Goal

Bring the functional mobile E2E layer into line with the corrected wedge runtime so completion is validated by real flows, not only by unit and UI tests.

## Context

The parent plan already defines the target E2E surface, but the follow-up runtime changes require the Detox flow assumptions to be corrected after Inbox and Sales Brief parity are restored.

## Spec

### Requirements

- Validate login to Inbox.
- Validate support approval and support handoff flows against the corrected runtime.
- Validate Sales Brief flow against completed and abstained outcomes.
- Validate activity and governance flows as the operational trace surfaces.

### Out of scope

- replacing Detox with a new functional runner
- broad mobile hardening outside follow-up wedge parity

## TDD Plan

### Tests first

- Audit the current E2E set against the follow-up runtime behavior.
- Rewrite or replace scenarios whose navigation assumptions are now outdated.

### Implementation steps

1. Update seeded data assumptions as needed.
2. Rewrite E2E entry and navigation flows for Inbox-first behavior.
3. Add or adjust scenarios for approval, handoff, Sales Brief, and activity.
4. Run the functional E2E suite for the touched scenarios.

## Files to Create / Modify

| # | File | Action | Lines affected |
|---|------|--------|----------------|
| 1 | `mobile/e2e` | Modify multiple files | — |
| 2 | `mobile/package.json` | Verify only | — |
| 3 | `scripts/e2e_seed_mobile_p2.go` | Modify if needed | — |

## Acceptance Criteria

- [ ] Functional E2E reflects the corrected wedge runtime.
- [ ] Inbox-first navigation is covered.
- [ ] Support approval, support handoff, Sales Brief, and activity scenarios are updated.
- [ ] Touched E2E flows pass locally or blockers are documented explicitly.

## Notes / Decisions

- If Detox instability remains unrelated to the follow-up code, document the blocker instead of masking it.

## Summary (complete when done)

Updated the Detox layer to a wedge-first baseline:
- added a dedicated `wedge-followup.e2e.ts` smoke suite for Inbox approval, Activity handoff, Sales Brief, Activity denied detail, and Governance;
- added bottom-tab Detox IDs in the tabs layout;
- rewrote E2E auth/navigation helpers for Inbox-first wedge navigation;
- restored compile-time compatibility for legacy skipped suites via seed helper aliases and a dedicated E2E tsconfig override;
- explicitly skipped the obsolete pre-wedge Detox suites so the default functional runner no longer validates removed drawer-era flows.

Validation status:
- `npx tsc --noEmit -p e2e/tsconfig.json` passes.
- Mobile runtime is now aligned for `Sales Brief` and handoff routing at the contract level:
  - handoff payloads are normalized in the mobile services layer so `Activity -> Accept Handoff` resolves case context correctly even when the backend only provides `caseId` / `triggerContext`;
  - `SalesBrief.nextBestActions` remains structured in runtime code, but `salesBriefApi.getSalesBrief()` now short-circuits to an E2E-only mock whenever `EXPO_PUBLIC_E2E_MODE=1` so Detox no longer waits on the local LLM path.
- Targeted mobile validation passes after those changes:
  - `npm run test -- --runTestsByPath '__tests__/services/api.test.ts' '__tests__/app/(tabs)/sales/brief.test.tsx' '__tests__/components/agents/HandoffBanner.test.tsx'`
  - `npx tsc --noEmit -p tsconfig.app.json`
- Detox no longer blocks on post-login visibility. It now reaches the suite and can pass `Inbox`, handoff, and both `Sales Brief` scenarios in some runs, but the runner remains flaky across emulator restarts and repeated executions.

Current blocker:
- The remaining blocker is Detox/Android-emulator instability rather than backend runtime:
  - repeated `npx detox test --configuration android.emu.debug e2e/wedge-followup.e2e.ts --cleanup` runs oscillate between mostly-green results and full-suite visibility timeouts after the emulator is recreated;
  - the temporary `Sales Brief` E2E mock is diagnostic only and does not close the task, because `F4-T3` still needs a stable runner result before `F4-T4`.
