---
doc_type: plan
title: "External Validation Open Points Rerun Plan"
status: active
created: 2026-07-04
task: EXTVAL-RERUN-PLAN-001
depends_on:
  - docs/plans/external_validation_first_test_battery_plan.md
  - docs/tasks/task_extval_battery_t2_knowledge_evidence.md
  - docs/tasks/task_extval_battery_t3_support_agent.md
  - docs/tasks/task_extval_battery_t5_sales_brief_copilot.md
  - docs/tasks/task_extval_battery_t7_mobile_real_mode.md
---

# External Validation Open Points Rerun Plan

## Purpose

Collect the unresolved product and validation gaps left open by the first
external-validation battery, then define the minimum fix-and-rerun sequence
needed to close those gaps without redoing already-green checks.

## Scope

This plan covers only the items that remained open after the first battery.
It does not reopen T1, T4, or T6 as standalone workstreams unless a dependent
fix changes the validated behavior enough to require a targeted rerun.

## Closed Baseline

These battery areas are already green enough to treat as baseline evidence:

- T1 Backend/BFF smoke and auth: passed end-to-end against real backend/BFF/DB.
- T4 Approval and handoff: passed after the threshold-normalization fix.
- T6 Deterministic eval regression: passed at
  `FENIX_EVAL_POLICY_COMPLIANCE_MIN=1.0`.

## Unresolved Points Inventory

### O1. Weak-query evidence does not abstain cleanly

Source: `EXTVAL-BATTERY-T2-001`

- Status: open
- Current evidence: irrelevant queries still return real but low-relevance
  items with `confidence: medium` instead of an empty result or explicit
  abstain signal.
- Why it matters: downstream copilot and agent flows cannot reliably distinguish
  "some retrieval happened" from "relevant grounding exists".
- Blocking effect on rerun: rerun T2 must wait for either a product fix or an
  explicit product decision that the current confidence contract is acceptable.

### O2. Support Agent validation does not match the documented LLM expectation

Source: `EXTVAL-BATTERY-T3-001`

- Status: open
- Current evidence: the support-agent path reaches terminal states, but it does
  not call the configured chat model; it uses rule-based thresholds and
  template strings, with `total_tokens: 0`.
- Why it matters: the plan and product documentation frame UC-C1 as grounded,
  evidence-based agent reasoning. The current implementation does not validate
  that claim.
- Blocking effect on rerun: rerun T3 requires a product decision first:
  either wire real LLM-backed reasoning into the support path, or downgrade the
  documented expectation and rerun against the accepted non-LLM contract.

### O3. Support Agent abstain path still sends a reply

Source: `EXTVAL-BATTERY-T3-001`

- Status: open
- Current evidence: `executeAbstainedAction` still invokes `send_reply`.
- Why it matters: the observed behavior conflicts with the documented
  "abstain + escalate to human" semantics.
- Blocking effect on rerun: if O2 keeps the support path active, rerun T3
  should include an explicit assertion for abstain behavior.

### O4. Sales Brief hard-fails on malformed suggested-actions output

Source: `EXTVAL-BATTERY-T5-001`

- Status: open
- Current evidence: `POST /bff/api/v1/copilot/sales-brief` failed 2/2 with
  `500` after real 20-26 second LLM calls because the suggested-actions JSON
  did not parse.
- Why it matters: a real LLM formatting deviation takes down the entire brief
  instead of degrading gracefully.
- Blocking effect on rerun: rerun T5 must wait for a code fix that keeps the
  brief functional when action parsing fails.

### O5. LLM usage accounting is incomplete or incorrect

Source: `EXTVAL-BATTERY-T3-001`, `EXTVAL-BATTERY-T5-001`

- Status: open
- Current evidence:
  - failed sales-brief generations do not emit `usage_event`
  - successful copilot chat records the embedding model instead of the chat
    model, with zero units
  - support-agent runs expose zero tokens because the current path does not use
    the chat model
- Why it matters: validation evidence for cost, latency, and model provenance is
  incomplete.
- Blocking effect on rerun: T5 rerun should verify this directly; T3 rerun
  depends on the O2 architecture decision.

### O6. Mobile real-mode login succeeds but the home screen crashes immediately

Source: `EXTVAL-BATTERY-T7-001`

- Status: fixed and verified (`task_mobile_approvals_response_shape_crash.md`,
  `EXTVAL-BATTERY-T7-RERUN-001`, 2026-07-05)
- Fix: the mobile approvals client (`mobile/src/services/api.secondary.ts`)
  now normalizes the BFF's `{data, meta}` payload instead of treating it as a
  bare array (`mobile/app/(tabs)/home/index.tsx`,
  `mobile/src/components/approvals/ApprovalCard.tsx` field alignment).
- Verified in real mode: fresh owner registered and logged in through the
  visible `/login` UI, Home rendered its empty state with no crash after the
  real `GET /api/v1/approvals` fetch (`200`), and all five bottom-nav surfaces
  (Support, Inbox, Activity, Sales, Governance) were reached without error.

### O7. Workspace owner `GET /api/v1/signals` returns 403 on fresh workspace

Source: `EXTVAL-BATTERY-T7-001`

- Status: fixed and verified (`task_extval_o7_signals_403_workspace_owner.md`,
  2026-07-05)
- Root cause: `defaultWorkspaceOwnerPermissions`
  (`internal/domain/auth/service.go:34`) — the bootstrap grant for the
  `workspace_owner` role — only had `records`/`agents`/`tools` keys, none of
  which satisfy the policy engine's `roleAllowsAction` fallback for
  `resource="api"`. Every handler that calls `checkActionAuthorization` with
  that resource (`signals`, `blackboard`, `eval`, `prompt`, `tool`,
  `workflow`) denied the freshly registered owner, while ungated handlers
  (approvals, cases, accounts, governance) worked fine — masking the defect
  until the real-mode T7 rerun hit `signals` specifically.
- Fix: added `"global":["admin"]` to the grant, satisfying
  `hasGlobalAdminPermission` for any `resource="api"` action, consistent with
  "owner" semantics.
- Verified: unit tests (`internal/domain/policy/evaluator_unit_test.go`,
  `internal/domain/auth/service_test.go`) plus a live check against the
  restarted backend — a freshly registered owner now receives `200` (not
  `403`) from `GET /api/v1/signals`.

### O8. Mobile Sales Brief and mobile Copilot remain unvalidated in real mode

Source: `EXTVAL-BATTERY-T5-001`, `EXTVAL-BATTERY-T7-001`

- Status: open
- Current evidence: API-level copilot chat passed, but the mobile Sales Brief
  screen and mobile Copilot route were deferred or blocked before real-mode UI
  completion.
- Why it matters: the first battery never completed the live mobile surfaces
  for these features.
- Blocking effect on rerun: rerun T7 must absorb these deferred mobile checks
  after O4 and O6 are fixed.

## Rerun Strategy

### Wave A. Fix blocking product gaps before rerunning the battery

1. Fix O6 and O7 together: stabilize the mobile Home screen against real BFF
   response shapes and confirm fresh-workspace owner access for `signals`.
2. Fix O4 and O5 for the copilot/sales-brief path: degrade gracefully on
   malformed suggested-actions output and record usage on both success and
   failure with the correct model attribution.
3. Resolve O2 and O3 by explicit product decision plus implementation:
   either make Support Agent genuinely LLM-backed, or update the documented
   contract and the validation criteria to the accepted non-LLM design.
4. Fix or formally accept O1: add an abstain-ready evidence-floor contract, or
   explicitly document why the current confidence behavior is sufficient.

### Wave B. Rerun only the battery points invalidated by those gaps

1. Rerun T2 with the weak-query scenario.
   Success condition: irrelevant queries produce an explicit abstain-safe
   outcome under the accepted evidence contract.
2. Rerun T3 using the decided Support Agent contract.
   Success condition: the support path matches the documented design, emits
   consistent reasoning/usage evidence, and no longer leaves ambiguity about
   whether the chat model participated.
3. Rerun T5 API surfaces.
   Success condition: `sales-brief` returns a non-fixture live response,
   copilot chat still returns grounded output, and usage attribution is correct.
4. Rerun T7 mobile real-mode navigation.
   Success condition: real login reaches a stable Home screen, Support/Inbox/
   Activity/Sales Brief/Governance are navigable, mobile support trigger is
   reachable, and the deferred mobile Sales Brief/Copilot checks complete.
5. Conditionally rerun T4 if the T2/T3 fixes materially change approval-entry
   behavior.
   Trigger: evidence-threshold or support-action logic changes that could alter
   whether approval requests are created for the same scenario.

## Secondary Operational Follow-Ups

These items remain open from the original plan, but they are not primary
functional blockers for the focused rerun unless the team explicitly wants a
more automated second pass:

- Align `.env.example` and any Compose model defaults with the locally validated
  Ollama model set.
- Add a dedicated external-validation Maestro flow that avoids
  `seed-and-run.sh`, `e2e-bootstrap`, and screenshot mode.
- Add a readiness script that checks Go, Java, Android SDK, Ollama models,
  backend/BFF health, and fixture-disabled state before a rerun starts.

Treat these as operational hardening tasks, not prerequisites for proving the
remaining product behavior gaps.

## Execution Preconditions For The Rerun

- Start backend and BFF from a checkout that includes every fix selected for
  Wave A; record `git rev-parse HEAD` and process start times.
- Use a fresh validation workspace for rerun evidence unless a specific task
  requires continuity with an existing seeded workspace.
- Keep `SCREENSHOT_MODE=false`, `ENABLE_SCREENSHOT_FIXTURES=false`, and
  `EXPO_PUBLIC_E2E_MODE` unset for every rerun claim.
- Re-run the relevant local QA gates for each fix wave before any externally
  observed rerun.
- Preserve API transcripts, audit rows, usage rows, and mobile evidence under a
  single rerun evidence packet so the second pass can be compared directly to
  the first battery.

## Suggested Discrete Tasks

1. Task: fix and retest mobile Home real-mode crash plus signals access.
2. Task: fix and retest sales-brief graceful degradation and usage tracking.
3. Task: resolve Support Agent LLM-vs-rule-based contract and retest T3.
4. Task: tighten evidence abstain semantics and rerun the weak-query scenario.
5. Task: execute the focused rerun battery for T2/T3/T5/T7 and conditional T4.

## Exit Criteria

This rerun plan is complete when:

- every open point O1-O8 is either fixed and rerun, or explicitly accepted as a
  product decision with updated documentation
- T2, T3, T5, and T7 each have a final rerun result recorded against the real
  runtime
- any conditional T4 rerun is either executed or explicitly waived with a
  documented reason
- no external-validation claim for mobile or copilot behavior depends on the
  first battery's blocked or partial outcomes
