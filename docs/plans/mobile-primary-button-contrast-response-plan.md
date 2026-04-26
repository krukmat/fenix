---
doc_type: summary
id: mobile-primary-button-contrast-response-plan
title: "Mobile Primary Button Contrast Response Plan"
status: completed
phase: design-governance
week: ""
tags: [mobile, design-system, accessibility, contrast, documentation, governance]
fr_refs: [FR-300]
uc_refs: []
adr_refs: [ADR-027]
created: 2026-04-26
completed: 2026-04-26
---

# FenixCRM - Mobile Primary Button Contrast Response Plan

## Summary

This plan defines the follow-up work for the `@google/design.md` contrast warning
on the documented primary button token pair:

- `brandPrimary`: `#3B82F6`
- `brandOnPrimary`: `#0A0D12`
- Original reported contrast: `3.68:1` with `#FFFFFF` on `#3B82F6`
- Current selected contrast: `5.29:1` with `#0A0D12` on `#3B82F6`
- WCAG AA target for normal text: `4.5:1`

The warning maps to implemented mobile UI. It must not be dismissed as only
documentation noise, but this plan does not authorize changing runtime colors in
the same governance step. Any runtime fix must be a separate mobile task with the
required mobile QA gate.

## Evidence

Runtime source of truth:

- `mobile/src/theme/colors.ts` defines `brandColors.primary` as `#3B82F6`.
- `mobile/src/theme/colors.ts` defines `brandColors.onPrimary` as `#0A0D12`.
- `DESIGN.md` documents those same values as `brandPrimary` and
  `brandOnPrimary`.

Implemented text-on-primary usages found during inspection:

| Area | Evidence | Classification | Why it matters |
| --- | --- | --- | --- |
| CRM forms | `mobile/src/components/crm/CRMFormBase.tsx` renders `SubmitButton` with `backgroundColor: colors.primary` and text `color: colors.onPrimary`. | Direct text-on-primary | Form submit actions render the warned pair directly. |
| CRM list retry action | `mobile/src/components/crm/CRMListScreen.tsx` renders retry button background `colors.primary` and text `colors.onPrimary`. | Direct text-on-primary | Error recovery action renders the warned pair directly. |
| CRM detail primary action | `mobile/src/components/crm/CoreCRMReadOnly.tsx` renders primary action background `colors.primary` and text `colors.onPrimary`. | Direct text-on-primary | Detail CTA renders the warned pair directly. |
| CRM list primary action | `mobile/src/components/crm/CRMListSelection.tsx` renders list-level primary action background `colors.primary` and text `colors.onPrimary`. | Direct text-on-primary | List CTAs render the warned pair directly. |
| CRM status filter chips | `mobile/src/components/crm/CRMListSelection.tsx` renders active status chips with `backgroundColor: colors.primary` and text `colors.onPrimary`. | Direct text-on-primary | Active filters render the warned pair in compact text. |
| CRM selected checkbox | `mobile/src/components/crm/CRMListSelection.tsx` renders selected checkboxes with background `colors.primary` and text `colors.onPrimary`. | Direct mark-on-primary | The selected mark uses the same contrast pair, though it is a symbol rather than normal text. |
| Governance audit filter chips | `mobile/src/components/governance/AuditFilterBar.tsx` renders active chips with `backgroundColor: colors.primary` and text `colors.onPrimary`. | Direct text-on-primary | Active governance filters render the selected accessible pair. |
| Support inbox badge | `mobile/app/(tabs)/support/index.tsx` renders `styles.inboxBadgeText` with `colors.onPrimary` over `colors.primary`. | Direct text-on-primary | Badge text renders the selected accessible pair. |
| Agent trigger button | `mobile/src/components/agents/TriggerAgentButton.tsx` sets React Native Paper `Button` background to `colors.primary`. | Needs component-style confirmation | Text color is controlled by Paper/theme styling rather than a visible `colors.onPrimary` expression. |

Decorative or non-text `colors.primary` usages, such as `ActivityIndicator`,
`RefreshControl`, header tint, borders, icons, timeline indicators, selected tab
labels, and status colors, are lower priority for this contrast warning because
they do not render normal text on the primary background.

## Decision Boundary

This follow-up must choose one explicit runtime approach:

- Change `brandColors.primary` to a darker accessible blue.
- Change `brandColors.onPrimary` to a text color that reaches contrast on the
  current blue.
- Add a component-specific primary-button token and update `DESIGN.md` to document
  that component contract.
- Keep runtime tokens unchanged only if implementation evidence changes and the
  warning no longer maps to a rendered text-on-background state.

Because implemented mobile UI currently uses the pair directly, "ignore the
warning" is not an acceptable final outcome without code evidence that those
rendered states no longer exist.

## Selected Strategy

T3 selects changing `brandColors.onPrimary` from `#FFFFFF` to `#0A0D12` and
updating equivalent hard-coded white-on-primary usages to the token during T4.
This keeps `brandColors.primary` at `#3B82F6`, preserving its current use as
operator-blue text, icons, indicators, active navigation, borders, and status
accents on dark surfaces.

Contrast checks for the selected strategy:

| Pair | Ratio | Outcome |
| --- | --- | --- |
| Current `#FFFFFF` on `#3B82F6` | `3.68:1` | Fails WCAG AA for normal text. |
| Selected `#0A0D12` on `#3B82F6` | `5.29:1` | Passes WCAG AA for normal text. |
| Existing `#3B82F6` on `#0A0D12` | `5.29:1` | Preserves current primary text/link contrast on the app background. |
| Existing `#3B82F6` on `#111620` | `4.92:1` | Preserves current primary text/link contrast on surface panels. |
| Existing `#EF4444` on selected `#0A0D12` | `5.17:1` | Keeps delete/error filled actions readable if they currently use `onPrimary`. |

Rejected alternatives:

- Darken `brandColors.primary`: rejected for this task because an accessible dark
  blue for white text, such as `#2563EB`, drops primary-blue text on
  `brandBackground` below AA (`3.76:1`) and would affect many decorative and link
  uses beyond primary buttons.
- Add only a component-specific primary-button token: rejected as the first move
  because implemented CRM chips, selected checkboxes, badges, and actions already
  use the shared `onPrimary` concept or hard-coded white over the same primary
  background.
- Keep runtime tokens unchanged: rejected because T2 confirmed rendered
  text-on-primary states.

T4 should update `mobile/src/theme/colors.ts`, root `DESIGN.md`, and equivalent
hard-coded white-on-primary usages identified in T2. It should not change
`brandColors.primary` unless new evidence invalidates this decision.

T4 implementation note: `brandColors.onPrimary` and `DESIGN.md` `brandOnPrimary`
were changed to `#0A0D12`. Equivalent hard-coded white-on-primary usages in
`AuditFilterBar` and the support inbox badge were migrated to `colors.onPrimary`.

## Implementation Plan

Execute these tasks in order. Do not start a runtime edit until the inspection and
decision task is complete.

| Task | Depends on | Effort / Reasoning | Output | Verification |
| --- | --- | --- | --- | --- |
| T1 - Confirm current contrast baseline | None | Low - deterministic token and linter check. | Current `@google/design.md` warning plus current runtime token values. | `npx @google/design.md lint DESIGN.md` reports the primary-button contrast warning, or the blocker is reported verbatim. |
| T2 - Trace implemented text-on-primary usages | T1 | Medium - requires separating actual text-on-background uses from decorative primary color uses. | Usage inventory for `colors.primary` with `colors.onPrimary` in mobile components and screens. | `rg -n "backgroundColor: colors\\.primary|color: colors\\.onPrimary" mobile/src mobile/app` evidence is reviewed and categorized. |
| T3 - Choose the runtime strategy | T2 | Medium - requires balancing accessibility, brand continuity, component scope, and design-system governance. | One selected approach from the decision boundary with rejected alternatives recorded. | Chosen approach reaches WCAG AA for implemented button text or documents why a component-specific exception is valid. |
| T4 - Implement runtime/design change | T3 | Medium - token or component changes affect shared mobile UI. | Updated runtime token or component-specific button styling, plus aligned `DESIGN.md`. | Changed runtime values and `DESIGN.md` stay in sync; no unrelated UI redesign is introduced. |
| T5 - Run inline-disable guard | T4 | Low - single repository script for lint hygiene. | Inline ESLint disable check result. | `bash scripts/check-no-inline-eslint-disable.sh` passes, or the blocker is reported verbatim. |
| T6 - Run mobile typecheck | T5 | Medium - validates TypeScript impact of shared token/component changes. | Mobile typecheck result. | `cd mobile && npm run typecheck` passes, or the blocker is reported verbatim. |
| T7 - Run mobile lint | T6 | Medium - validates static quality for mobile runtime edits. | Mobile lint result. | `cd mobile && npm run lint` passes, or the blocker is reported verbatim. |
| T8 - Run mobile architecture quality gate | T7 | Medium - validates mobile architecture constraints after shared UI changes. | Architecture quality result. | `cd mobile && npm run quality:arch` passes, or the blocker is reported verbatim. |
| T9 - Run mobile coverage gate | T8 | Medium - validates test coverage after runtime UI/theme changes. | Coverage result. | `cd mobile && npm run test:coverage` passes, or the blocker is reported verbatim. |
| T10 - Close documentation loop | T9 | Low - update governance state after the runtime decision is validated. | This plan or a linked completion note records the selected fix and validation outcome. | The `@google/design.md` warning count is updated if the fix changes the baseline. |

The preferred shortcut for T5 through T9 is `bash scripts/qa-mobile-prepush.sh`.
If the shortcut fails, execute and report the individual subtasks above so the
blocker remains isolated to a medium-or-lower unit of work.

## Runtime Guardrails

- Do not change `mobile/src/theme/*` before T3 selects a strategy.
- Do not change only `DESIGN.md` to silence this warning while mobile still
  renders `colors.primary` with `colors.onPrimary`.
- Do not choose a new blue from visual preference alone; the selected value must
  satisfy contrast and preserve the Command Center direction unless a separate
  design decision changes the brand.
- If runtime tokens change, update `DESIGN.md` in the same turn and reference
  ADR-027 as the governance source for the visual-context contract.

## Acceptance Criteria

- The implemented text-on-primary usages are inventoried.
- The selected strategy is explicit and records why alternatives were not chosen.
- Runtime token or component changes, if any, are reflected in `DESIGN.md`.
- `npx @google/design.md lint DESIGN.md` is re-run after documentation changes.
- If any `mobile/` file changes, `bash scripts/qa-mobile-prepush.sh` is run before
  push.

## Completion Notes - 2026-04-26

- `brandColors.onPrimary` changed from `#FFFFFF` to `#0A0D12` while
  `brandColors.primary` stayed `#3B82F6`.
- Root `DESIGN.md` now documents `brandOnPrimary: "#0A0D12"`.
- The primary button contrast warning was removed from
  `npx @google/design.md lint DESIGN.md`; the current result is `0 errors`,
  `26 warnings`, and `1 info`.
- Equivalent hard-coded white-on-primary usages in
  `mobile/src/components/governance/AuditFilterBar.tsx` and
  `mobile/app/(tabs)/support/index.tsx` now use `colors.onPrimary`.
- Required mobile gates passed:
  `bash scripts/check-no-inline-eslint-disable.sh`,
  `cd mobile && npm run typecheck`,
  `cd mobile && npm run lint`,
  `cd mobile && npm run quality:arch`, and
  `cd mobile && npm run test:coverage`.
- Coverage gate result: `71` test suites passed, `530` tests passed, global
  coverage `77.7%` statements, `74.27%` branches, `75.07%` functions, and
  `77.7%` lines.

## Assumptions

- FenixCRM mobile is the relevant runtime surface for this warning.
- The current contrast warning is real for CRM submit, retry, detail primary, and
  selection actions.
- This document is a planning and handoff artifact; it does not itself change
  mobile UI behavior.
