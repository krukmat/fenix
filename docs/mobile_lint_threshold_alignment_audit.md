---
doc_type: audit
title: "Mobile lint threshold alignment audit"
status: completed
created: 2026-06-28
updated: 2026-06-28
tags: [audit, mobile, qa, eslint, maintainability]
---

# Mobile lint threshold alignment audit

## Objective

Assess whether fenix mobile should tighten `max-lines-per-function` from `80` toward the stricter DubBridge-style threshold of `60`.

## Current baseline

- Fenix mobile currently enforces `max-lines-per-function = 80` in [mobile/eslint.config.js](/Users/matias/fenix/mobile/eslint.config.js:23).
- DubBridge mobile uses `max-lines-per-function = 60`.
- Current repo lint passes under the existing fenix threshold via `npm run lint` in `mobile/`.

## Measured impact

- Functions above `80` effective lines: **0**
- Functions above `60` effective lines: **26**

Top impacted functions under a `60`-line candidate threshold:

- `app/(auth)/register.tsx:27` — 78 lines
- `app/(tabs)/governance/audit.tsx:45` — 78 lines
- `src/hooks/useSSE.ts:44` — 78 lines
- `app/(tabs)/sales/deal-[id].tsx:167` — 77 lines
- `app/(auth)/login.tsx:29` — 76 lines
- `app/(tabs)/activity/insights.tsx:39` — 75 lines
- `src/components/governance/AuditEventCard.tsx:35` — 75 lines
- `src/components/inbox/InboxFeed.tsx:141` — 75 lines
- `app/(tabs)/workflows/index.tsx:46` — 73 lines
- `app/(tabs)/governance/usage.tsx:29` — 72 lines

Additional impacted files also include workflow screens, support/activity detail screens, CRM forms, approval/governance cards, and copilot/detail views.

## Interpretation

- The current `80` threshold is active and not obviously too loose in the sense of allowing runaway function size; the largest measured effective functions are still under `80`.
- Tightening directly to `60` would not be a small policy adjustment. It would create a remediation wave across **26** existing mobile functions.
- The likely remediation pattern would be extraction of screen-model hooks, subcomponents, field sections, action bars, and helper formatters/validators.

## Recommendation

Do **not** lower the threshold from `80` to `60` as a drive-by policy edit.

Recommended path:

1. Keep `80` for now as the enforced repo threshold.
2. If stronger decomposition pressure is desired, open a dedicated remediation wave for the 26 impacted functions.
3. Only lower the threshold after that remediation wave is either completed or intentionally staged by area.

## Verification

- `cd mobile && npm run lint`
- Local AST-based line audit over non-test `app/**` and `src/**` TypeScript/TSX files to count effective function lines under `>80` and `>60` candidate thresholds.
