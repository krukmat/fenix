---
doc_type: audit
title: "Mobile threshold remediation wave"
status: completed
created: 2026-06-28
updated: 2026-06-28
tags: [audit, mobile, qa, eslint, refactor, maintainability]
---

# Mobile threshold remediation wave

## Objective

Turn the earlier `80 -> 60` mobile threshold assessment into a staged remediation program that can realistically tighten `max-lines-per-function` without a one-shot break.

## Current impact baseline

The earlier audit identified **26** non-test mobile functions above an effective `60`-line function threshold and **0** above the current `80` threshold.

## Recommended remediation batches

### Wave 1 — Shared form primitives and repeated screen-model patterns

Best payoff first because these refactors reduce multiple files at once and create reusable extraction patterns.

Target files:

- [mobile/src/components/crm/CRMAccountForm.tsx](/Users/matias/fenix/mobile/src/components/crm/CRMAccountForm.tsx:1)
- `mobile/src/components/crm/CRMCaseForm.tsx`
- `mobile/src/components/crm/CRMContactForm.tsx`
- `mobile/src/components/crm/CRMDealCreateForm.tsx`
- [mobile/src/components/workflows/WorkflowForm.tsx](/Users/matias/fenix/mobile/src/components/workflows/WorkflowForm.tsx:1)

Dominant extraction pattern:

- split field groups into section components;
- move payload/validation/reset logic into shared helpers or `use*FormModel` hooks;
- keep top-level form component as wiring + submit orchestration only.

Why first:

- this wave creates the reusable form decomposition style the rest of the mobile app can follow;
- it likely removes several of the 26 overages with one coherent pattern family.

### Wave 2 — Auth and workflow screen orchestration

These are route screens that mix loading, mutation, validation, navigation, and layout.

Target files:

- `mobile/app/(auth)/login.tsx`
- `mobile/app/(auth)/register.tsx`
- `mobile/app/(tabs)/workflows/index.tsx`
- `mobile/app/(tabs)/workflows/new.tsx`
- `mobile/app/(tabs)/workflows/[id].tsx`
- `mobile/app/(tabs)/workflows/edit/[id].tsx`

Dominant extraction pattern:

- extract `use*ScreenModel` hooks for data loading, submit/navigation decisions, and derived flags;
- extract list/detail sections, action bars, and empty/loading/error states into subcomponents.

Why second:

- workflow screens are a dense cluster and benefit from a consistent “route as coordinator” refactor style;
- auth screens are smaller scope and good early wins once the shared pattern is established.

### Wave 3 — Governance, activity, support, and inbox surfaces

These screens/components tend to be composition-heavy rather than form-heavy.

Target files:

- `mobile/app/(tabs)/governance/audit.tsx`
- `mobile/app/(tabs)/governance/usage.tsx`
- `mobile/src/components/governance/AuditEventCard.tsx`
- `mobile/src/components/governance/UsageDetailCard.tsx`
- `mobile/app/(tabs)/activity/insights.tsx`
- `mobile/app/(tabs)/activity/[id].tsx`
- `mobile/app/(tabs)/support/[id].tsx`
- [mobile/src/components/inbox/InboxFeed.tsx](/Users/matias/fenix/mobile/src/components/inbox/InboxFeed.tsx:1)
- `mobile/src/components/approvals/ApprovalCard.tsx`
- `mobile/src/components/signals/SignalDetailView.tsx`
- `mobile/src/components/copilot/CopilotPanel.tsx`

Dominant extraction pattern:

- split render branches into card/section fragments;
- extract formatting and view-model mapping helpers;
- move item-type switching and route mapping into smaller presentation helpers.

Why third:

- these are strong maintainability wins, but they are more varied than the form/workflow waves;
- the earlier waves establish the team’s preferred extraction idioms first.

### Wave 4 — Sales and detail screens

These are more domain-specific route screens and should come after the shared patterns exist.

Target files:

- `mobile/app/(tabs)/sales/deal-[id].tsx`
- `mobile/app/(tabs)/sales/[id]/brief.tsx`

Dominant extraction pattern:

- extract domain-specific action sections, summary panels, and mutation handlers;
- keep route files focused on data wiring and navigation.

Why fourth:

- fewer files than other waves;
- easier once shared detail-screen and card extraction patterns have already been used elsewhere.

### Wave 5 — Infrastructure hook special-case

Single-file special handling:

- [mobile/src/hooks/useSSE.ts](/Users/matias/fenix/mobile/src/hooks/useSSE.ts:1)

Dominant extraction pattern:

- split transport/event handling from hook state orchestration;
- extract message transformation helpers and stream lifecycle helpers;
- keep `useSSE` as a thin stateful facade over smaller pure helpers.

Why separate:

- this file is not a screen or component, so it wants a different refactor shape;
- bundling it into UI waves would dilute the plan.

## Suggested rollout order

1. Wave 1 — shared form primitives and form-model extraction
2. Wave 2 — auth + workflow route orchestration
3. Wave 3 — governance/activity/support/inbox composition surfaces
4. Wave 4 — sales/detail route screens
5. Wave 5 — `useSSE` hook refactor

## Policy recommendation

Do not drop `max-lines-per-function` from `80` to `60` until at least Waves 1 and 2 are complete.

Reason:

- Waves 1 and 2 create the shared decomposition idioms that make the remaining waves cheaper and more consistent.
- Lowering the threshold before those patterns exist would turn the policy change into scattered, opportunistic refactors instead of a coherent quality program.

## Best near-term governance payoff

If only one implementation wave is approved next, choose **Wave 1**.

It has the best combination of:

- reusable extraction patterns,
- multiple impacted files,
- low conceptual fragmentation,
- strong leverage for every later wave.
