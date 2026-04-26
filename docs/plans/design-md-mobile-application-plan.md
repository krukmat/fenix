---
doc_type: summary
id: design-md-mobile-application-plan
title: "Apply DESIGN.md to Mobile UI Plan"
status: planned
phase: mobile-design-governance
week: ""
tags: [mobile, design-system, design-md, ui, accessibility, governance]
fr_refs: [FR-300]
uc_refs: []
adr_refs: [ADR-027]
created: 2026-04-26
completed: ""
---

# FenixCRM - Apply DESIGN.md to Mobile UI Plan

## Summary

This plan turns the completed `DESIGN.md` governance work into an executable
mobile implementation path. The goal is not to redesign FenixCRM, but to align
mobile UI code with the documented Command Center contract:

- Runtime source of truth: `mobile/src/theme/*`.
- Agent-facing visual contract: root `DESIGN.md`.
- Governance decision: `docs/decisions/ADR-027-design-md-agent-visual-context.md`.
- Current validation baseline: `npx @google/design.md lint DESIGN.md` reports
  `0 errors`, `26 warnings`, and `1 info`.

The plan is prepared for a coding agent. Execute tasks in order, report each task
before starting, close it with verification, and stop for explicit confirmation
before moving to the next task.

## Current Observations

The mobile app already uses the Command Center theme in many CRM and governance
surfaces, but the scan found remaining visual drift that should be handled in
small batches:

- Hard-coded Command Center values that should use `theme.colors` or shared
  token modules, for example `darkStackOptions`.
- Legacy light-theme or scaffold colors in `HomeFeed`, `CopilotPanel`,
  workflow graph nodes, signal confidence badges, and agent-detail surfaces.
- Repeated local chips, badges, cards, and code blocks that match concepts now
  documented in `DESIGN.md` but are not centralized.
- Semantic colors duplicated as hex literals instead of using
  `mobile/src/theme/semantic.ts` helpers or `semanticColors`.
- Existing `@google/design.md` warnings for unreferenced documented runtime tokens.
  These warnings are acceptable as governance baseline unless a task changes the
  runtime token set or component contract.

## Agent Execution Contract

- Read `DESIGN.md` and ADR-027 before each implementation task.
- Do not infer new visual direction from screenshots when a runtime token exists.
- Prefer existing `mobile/src/theme/*`, React Native Paper theme colors, and local
  component patterns over new ad hoc styling.
- Keep each task to one mobile area or one shared primitive.
- If a task changes `mobile/`, run the full mobile gate before push:
  `bash scripts/qa-mobile-prepush.sh`.
- If only documentation changes, run `npx @google/design.md lint DESIGN.md`.
- Any runtime token change must update `DESIGN.md` in the same task.
- No task in this plan should exceed Medium effort. Split broad UI migrations into
  the Medium or Low tasks below.

## Implementation Plan

| Task | Depends on | Effort / Reasoning | Output | Verification |
| --- | --- | --- | --- | --- |
| T1 - Establish mobile visual baseline | None | Low - read-only baseline capture from current contract and repo state. | Notes for current `DESIGN.md` lint result, current `mobile/src/theme/*` token values, and existing changed-file scope. | `npx @google/design.md lint DESIGN.md`; `git status --short --untracked-files=all`; no files edited. |
| T2 - Build visual drift inventory | T1 | Medium - requires classifying hard-coded styles by token, semantic helper, or intentional exception. | Inventory table grouping mobile drift by colors, typography, spacing/radius, cards, chips, code/data surfaces, and navigation. | `rg` evidence references each affected file and classifies every item as migrate, document exception, or defer. |
| T3 - Define shared mobile styling primitives | T2 | Medium - converts repeated design concepts into narrow reusable helpers without redesigning screens. | Proposal or implementation for small primitives/helpers only where duplication is proven. Candidate areas: cards, chips, data/code blocks, status/confidence colors, stack header options. | Typecheck passes if code changes; no new palette, radius scale, or typography scale is introduced. |
| T4 - Align navigation and shell surfaces | T3 | Low - limited migration of app shell values to existing theme tokens. | Token-backed shell/navigation styles, including `darkStackOptions` and any app-level loading shell that duplicates documented colors. | Visual constants match `DESIGN.md`; `cd mobile && npm run typecheck` passes. |
| T5 - Align CRM shared components | T4 | Medium - CRM has shared list/detail/form primitives that affect many screens. | CRM shared components use documented screen, card, button, status-chip, and data-code conventions. | Existing CRM tests pass; no screen-specific redesign is introduced. |
| T6 - Align governance and agent surfaces | T5 | Medium - governance and agent screens contain semantic status, evidence, code, and audit surfaces. | Governance and agent UI uses theme/semantic tokens for status chips, evidence cards, code blocks, and operational panels. | Relevant component tests pass; accessibility contrast for text-on-fill states is checked. |
| T7 - Align signals, inbox, support, and sales edge surfaces | T6 | Medium - these areas contain badges, confidence displays, and legacy color literals. | Remaining customer-facing wedge surfaces use documented tokens or recorded exceptions. | Relevant tests pass and `rg` confirms no migrated area retains avoidable one-off colors. |
| T8 - Align workflow and copilot visual surfaces | T7 | Medium - workflow/canvas and copilot surfaces may intentionally need specialized colors. | Workflow and copilot styles either map to documented tokens or record explicit exceptions for domain-specific node/message states. | Exceptions are documented in this plan or a follow-up ADR if they change the design contract. |
| T9 - Run accessibility and contrast pass | T8 | Medium - requires checking implemented text/background pairs after migrations. | Contrast notes for primary, error, semantic, chip, badge, and code/data states. | Failing implemented text/background pairs become separate runtime tasks; `DESIGN.md` warnings are updated if baseline changes. |
| T10 - Sync DESIGN.md and governance docs | T9 | Low - documentation update after implementation decisions are known. | `DESIGN.md`, this plan, and any affected governance docs reflect final token/component decisions. | `npx @google/design.md lint DESIGN.md` reports expected baseline and no errors. |
| T11 - Run full mobile QA | T10 | Medium - required gate for mobile runtime/UI changes. | QA result summary. | `bash scripts/qa-mobile-prepush.sh` passes, or the blocker is reported before push. |
| T12 - Commit and push scoped changes | T11 | Low - repository hygiene once QA is green. | Commit containing only this plan's scoped files. | Configure `git config fenix.ai-agent "chat-gpt5.4"`; verify staged diff excludes unrelated work; push after QA. |

## Medium Task Splits

Use these subtasks when a Medium task is too broad for one coding pass.

| Parent | Subtask | Effort / Reasoning | Output | Verification |
| --- | --- | --- | --- | --- |
| T2 | T2.1 - Color literal inventory | Low - mechanical search and classification. | List of hard-coded colors by token candidate. | `rg -n "#[0-9A-Fa-f]{3,8}" mobile/app mobile/src` evidence captured. |
| T2 | T2.2 - Shape and spacing inventory | Low - mechanical search for repeated layout primitives. | List of repeated radii, padding, and card/chip shapes. | Sampled files map to `DESIGN.md` spacing and rounded tokens. |
| T2 | T2.3 - Typography inventory | Low - identify local text styles that should use shared typography. | Candidate headings, labels, mono/data text, and compact metadata styles. | No changes yet; inventory only. |
| T3 | T3.1 - Choose helper boundaries | Low - decide only where reuse is proven. | Short decision note for helpers versus local styles. | Avoids creating primitives for one-off visuals. |
| T3 | T3.2 - Implement one helper family | Medium - code change limited to one primitive family. | One helper/component family, such as status chips or data blocks. | Targeted tests and typecheck pass. |
| T5 | T5.1 - CRM list primitives | Medium - shared list surfaces affect many screens. | List rows, selection controls, retry and primary actions aligned. | CRM list tests pass. |
| T5 | T5.2 - CRM detail primitives | Medium - detail cards and metadata have repeated patterns. | Detail header, sections, timeline, and empty cards aligned. | CRM detail tests pass. |
| T5 | T5.3 - CRM form primitives | Low - existing form base already centralizes much styling. | Form cards, labels, inputs, errors, and submit actions checked. | Form tests pass. |
| T6 | T6.1 - Governance cards and filters | Medium - cards and filters are repeated but bounded. | Usage/audit cards and filters use theme/semantic tokens. | Governance tests pass. |
| T6 | T6.2 - Agent detail and activity surfaces | Medium - many status colors require semantic classification. | Agent status, evidence, code, and audit blocks aligned. | Agent tests pass. |
| T7 | T7.1 - Signal confidence surfaces | Medium - confidence colors must preserve semantics and contrast. | Signal cards/detail confidence badges use semantic confidence tokens. | Signal tests pass. |
| T7 | T7.2 - Inbox/support/sales badges and chips | Medium - repeated wedge surfaces need token-backed fills/text. | Badges and chips use theme/semantic tokens or documented exceptions. | Wedge tests pass. |
| T8 | T8.1 - Workflow node color decision | Medium - node colors may be domain-specific and not all brand tokens. | Decision to map, keep, or document workflow node colors. | If kept, exception is documented with rationale. |
| T8 | T8.2 - Copilot message surface alignment | Low - message bubbles can map to existing surfaces. | Copilot panel surfaces use dark operational tokens where appropriate. | Copilot tests pass. |

## Verification Matrix

Run the narrowest relevant checks during each task, then the full mobile gate before
push.

| Change scope | Required local verification |
| --- | --- |
| Documentation only | `npx @google/design.md lint DESIGN.md` |
| `DESIGN.md` token changes | `npx @google/design.md lint DESIGN.md`; inspect matching `mobile/src/theme/*` values |
| Mobile component/screen changes | `cd mobile && npm run typecheck`; targeted tests for touched area |
| Shared mobile primitive or theme changes | `bash scripts/qa-mobile-prepush.sh` |
| Before push with any `mobile/` change | `bash scripts/qa-mobile-prepush.sh` |

## T9 Contrast Pass Results

Status: completed on 2026-04-26.

The runtime surface pairs checked from `mobile/src/theme/colors.ts` pass WCAG AA
for normal text:

| Pair | Ratio | Result |
| --- | ---: | --- |
| `onBackground` on `background` | 17.69:1 | Pass |
| `onSurface` on `surface` | 14.69:1 | Pass |
| `onSurfaceVariant` on `surface` | 6.20:1 | Pass |
| `onSurface` on `surfaceVariant` | 13.17:1 | Pass |
| `onSurfaceVariant` on `surfaceVariant` | 5.56:1 | Pass |
| `onPrimary` on `primary` | 5.29:1 | Pass |
| `primary` on `surface` | 4.92:1 | Pass |
| `error` on `surface` | 4.81:1 | Pass |
| `warning` on `surface` | 8.43:1 | Pass |
| `success` on `surface` | 7.14:1 | Pass |
| `onPrimaryContainer` on `primaryContainer` | 6.38:1 | Pass |
| `onSuccessContainer` on `successContainer` | 9.74:1 | Pass |
| `onWarningContainer` on `warningContainer` | 10.82:1 | Pass |
| `onErrorContainer` on `errorContainer` | 8.77:1 | Pass |

Implemented code/data surfaces also look safe where inspected:
`DSLViewer` uses `onSurface` on `surfaceVariant`, and agent input/tool code blocks
use `onSurfaceVariant` on `surface` or `background`.

Risks found:

| Implemented pair | Ratio | Affected examples | Follow-up |
| --- | ---: | --- | --- |
| `onError` / white text on `semanticColors.success` | 2.54:1 | Agent/activity status badges, signal confidence badges, workflow active status, deal won/qualified chips. | Replace text color with `brandColors.onPrimary` / near-black for direct green fills, or move to `successContainer` + `onSuccessContainer`. |
| `onError` / white text on `semanticColors.warning` | 2.15:1 | Agent warning/partial statuses, workflow or deal warning/open states. | Replace text color with `brandColors.onPrimary` / near-black for direct amber fills, or move to `warningContainer` + `onWarningContainer`. |
| `onError` / white text on `semanticColors.info` | 2.54:1 | Agent escalated/delegated info badges. | Use dark text on direct info fill, or introduce/document an info container pair before broad reuse. |
| `onError` / white text on `brandColors.primary` | 3.68:1 | Some active filter chips still use white text instead of `onPrimary`. | Use `brandColors.onPrimary`; documented primary button/chip pair passes at 5.29:1. |
| `onError` / white text on `brandColors.error` | 3.76:1 | Error/signal-count badges and denied/lost fills. | Current token pair only passes for large text. For compact labels, prefer `errorContainer` + `onErrorContainer` or darken the filled error color. |
| `onError` / white text on `#A78BFA` or `#8B5CF6` handed-off fills | 2.72:1 to 4.23:1 | Agent handed-off/public status chips. | Replace with a tokenized container/text pair or dark text when using light purple. |

Primary follow-up: introduce a small status-badge text-color helper or migrate
status badges to container tokens so `getAgentStatusColor`, `getConfidenceColor`,
workflow status colors, deal status colors, and audit outcome badges do not pair
compact white text with light semantic fills.

## T10 Documentation Sync Results

Status: completed on 2026-04-26.

`DESIGN.md` was updated to make the T9 contrast finding explicit: compact status
chip and badge text must use AA-safe text/fill pairs, and future work should not
pair white text with direct light semantic fills when a darker `on*` or container
alternative is available.

No runtime tokens or mobile components were changed in T10. The current
`@google/design.md` validation baseline remains `0 errors`, `26 warnings`, and
`1 info`.

## T11 Mobile QA Results

Status: completed on 2026-04-26.

Full mobile pre-push QA passed with:

- `bash scripts/qa-mobile-prepush.sh`
- `71` Jest test suites passed.
- `530` tests passed.
- Coverage summary: `77.7%` statements, `74.27%` branches, `75.07%` functions,
  and `77.7%` lines.

## Acceptance Criteria

- Mobile implementation tasks explicitly use `DESIGN.md` and ADR-027 as visual
  context.
- Remaining one-off colors, typography, spacing, and radius values are either
  migrated to runtime tokens/helpers or documented as intentional exceptions.
- Runtime token changes, if any, are reflected in `DESIGN.md` in the same task.
- `@google/design.md` validation has no errors and any warning baseline change is
  recorded.
- Full mobile QA passes before push for any mobile runtime/UI changes.
- The final commit excludes unrelated local work such as BFF snapshot files,
  generated binaries, or local tools.

## Out of Scope

- Broad visual redesign.
- New brand palette, type scale, spacing scale, or radius scale.
- Backend-only work and BFF snapshot implementation.
- Rewriting working screens solely to reduce local style declarations when no
  concrete drift exists.

## Assumptions

- FenixCRM mobile remains the primary runtime surface for the current
  `DESIGN.md` contract.
- Existing `@google/design.md` unreferenced-token warnings are acceptable unless a
  task changes token/component contracts.
- Some domain visuals, especially workflow node colors, may require documented
  exceptions instead of forced mapping to brand tokens.
