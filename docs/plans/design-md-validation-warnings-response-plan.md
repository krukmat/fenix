---
doc_type: summary
id: design-md-validation-warnings-response-plan
title: "DESIGN.md Validation Warnings Response Plan"
status: planned
phase: design-governance
week: ""
tags: [design-system, agents, ui, mobile, documentation, governance, accessibility]
fr_refs: [FR-300]
uc_refs: []
adr_refs: [ADR-027]
created: 2026-04-26
completed: ""
---

# FenixCRM — DESIGN.md Validation Warnings Response Plan

## Summary

This plan governs how FenixCRM responds to `@google/design.md` warnings after the
initial `DESIGN.md` adoption.

The policy is governance-first. It does not change `mobile/` runtime tokens. It
classifies linter warnings, defines when a warning is acceptable documentation noise,
and defines when a warning must become a separate runtime/design task with mobile QA.

This document is intended for handoff to a future Codex agent. It explains how the
warning baseline should guide later mobile adaptation work without authorizing that
agent to change runtime UI as part of this governance task.

## Current Baseline

Baseline command:

```bash
npx @google/design.md lint DESIGN.md
```

Current result from the completed adoption:

- `0 errors`
- `26 warnings`
- `1 info`

Known warning classes:

- Missing canonical `colors.primary`; the current contract uses runtime-aligned names
  such as `brandPrimary` and documents their mapping to `mobile/src/theme/colors.ts`.
- Documented runtime tokens not referenced by the minimal `components` block.

Resolved warning class:

- Primary button contrast was resolved by changing `brandOnPrimary` /
  `brandColors.onPrimary` from `#FFFFFF` to `#0A0D12`. The pair
  `#0A0D12` on `#3B82F6` reports contrast ratio `5.29:1`, above WCAG AA
  `4.5:1`.

## Warning Policy

`error` findings are blocking. They must be fixed before merge unless the command
cannot run because of an explicit environment, dependency, or network blocker.

Schema or naming warnings should be fixed in `DESIGN.md` when the fix improves
`@google/design.md` compatibility without diverging from runtime tokens.

Unreferenced-token warnings are acceptable when the token exists in
`mobile/src/theme/*` and is intentionally documented for future UI work. Do not remove
runtime-aligned tokens only to silence the linter.

Contrast and accessibility warnings are not silently accepted for future runtime work.
If the warning reflects implemented UI, open a separate design/runtime task before
changing colors or component behavior.

## Escalation Rules

If only `DESIGN.md` or documentation changes, run:

```bash
npx @google/design.md lint DESIGN.md
```

No mobile gate is required for documentation-only changes.

If a change touches `mobile/src/theme/*`, mobile screens, navigation, or reusable
mobile UI components, run:

```bash
bash scripts/qa-mobile-prepush.sh
```

Runtime token changes must update `DESIGN.md` in the same turn and cite ADR-027 as the
governance source for the visual-context contract.

## Task Breakdown

| Task | Summary | Expected files or areas | Effort/reasoning | Verification |
| --- | --- | --- | --- | --- |
| T1 — Re-run DESIGN.md validation | Run the linter and compare the result with the documented warning baseline. | `DESIGN.md` | Low — command-only validation against an existing baseline. | `npx @google/design.md lint DESIGN.md` reports `0 errors`, or the blocker is reported verbatim. |
| T2 — Classify validation findings | Map each finding to blocking, acceptable documentation noise, or escalation-required accessibility/runtime concern. | `docs/plans/design-md-validation-warnings-response-plan.md`, `DESIGN.md` | Medium — requires judgment about whether a warning reflects schema compatibility, documented runtime tokens, or implemented UI risk. | Findings are grouped by outcome and no `error` finding is left unresolved. |
| T3 — Preserve runtime-aligned documentation | Keep intentionally documented mobile runtime tokens in `DESIGN.md` when they still exist in `mobile/src/theme/*`. | `DESIGN.md`, `mobile/src/theme/*` read-only context | Low — confirm token existence without changing runtime files. | Unreferenced-token warnings are either accepted with rationale or converted into a documentation cleanup task. |
| T4 — Escalate contrast/accessibility risks | For contrast or accessibility warnings, decide whether the warning maps to implemented UI and create a separate runtime/design task if it does. | `DESIGN.md`, mobile component and screen layer read-only context, future task record if needed | Medium — requires tracing token usage before deciding whether a runtime change is justified. | A follow-up task exists for implemented UI risk, or the plan records why the warning is not an implemented state. |
| T5 — Guard governance scope | Ensure this governance task does not change mobile runtime tokens, screens, navigation, or reusable UI components. | `docs/plans/design-md-validation-warnings-response-plan.md`; no `mobile/` writes | Low — scope check before closing the task. | `git diff -- mobile` is empty for this task. |

## Mobile Adaptation Handoff

This plan is an input to future mobile adaptation, not the adaptation itself. The next
agent should treat it as a decision filter:

1. Run `npx @google/design.md lint DESIGN.md` and compare the result to the baseline
   above.
2. Ignore unreferenced-token warnings when the token still exists in
   `mobile/src/theme/*` and is documented intentionally.
3. Convert contrast/accessibility warnings into a separate mobile design task when
   the warning maps to an implemented component, screen, or runtime token.
4. Do not change `mobile/src/theme/*` inside this governance plan.

For the known primary-button contrast warning, the follow-up mobile task must inspect
actual usage before choosing a fix. The agent should check primary action styling in
the mobile component and screen layer, confirm whether `brandPrimary` /
`brandOnPrimary` is rendered as text-on-background in production UI, and then choose
one explicit runtime approach:

- Change `brandPrimary`.
- Change `brandOnPrimary`.
- Add or document a component-specific primary-button token.
- Keep runtime tokens unchanged and record why the linter warning does not reflect an
  implemented UI state.

If the follow-up changes runtime tokens or mobile UI, it must update `DESIGN.md`,
reference ADR-027, and run `bash scripts/qa-mobile-prepush.sh`.

## Follow-Up Recommendation

Use `docs/plans/mobile-primary-button-contrast-response-plan.md` as the separate
runtime/design plan to evaluate the `brandPrimary` / `brandOnPrimary` contrast
warning before changing any token.

That follow-up should inspect actual primary-button usage in mobile, choose whether
to modify `brandPrimary`, `brandOnPrimary`, component-specific button colors, or the
documented component token, and then run the required mobile QA gate if runtime files
change.

## Acceptance Criteria

- This plan exists as an Obsidian artifact with YAML front matter and
  `doc_type: summary`.
- The current `@google/design.md` warning baseline is documented.
- Warning classes are classified into blocking, acceptable, or escalation-required
  outcomes.
- Escalation rules distinguish documentation-only changes from `mobile/` runtime or
  UI changes.
- A future Codex agent can tell whether a warning should be ignored, documented, or
  promoted to a separate mobile/runtime task.
- No `mobile/` files are changed by this governance plan.

## Assumptions

- The goal is governance, not immediate visual redesign.
- Existing `DESIGN.md` warnings are accepted as the baseline for the completed
  adoption.
- The contrast warning is important, but any fix belongs in a separate runtime/mobile
  task because changing colors would alter the implemented design system.
