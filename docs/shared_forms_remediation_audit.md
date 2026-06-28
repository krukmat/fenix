---
doc_type: audit
title: "Mobile Wave 1 shared forms remediation"
status: completed
created: 2026-06-28
updated: 2026-06-28
tags: [audit, mobile, qa, eslint, refactor, maintainability, forms]
---

# Mobile Wave 1 shared forms remediation

## Objective

Execute the first remediation wave for a future `max-lines-per-function` tightening by reducing the shared CRM and workflow form surfaces that were relying on the current `80` threshold headroom.

## Refactor pattern introduced

This wave established a reusable decomposition pattern for mobile forms:

1. keep the top-level form component focused on query/mutation wiring, local state, and submit orchestration;
2. extract repeated field sections into private `*Fields` components;
3. move shared visual structure into reusable form primitives;
4. centralize repeated choice-list presentation in shared helpers rather than per-form inline JSX.

## Shared primitives added

Added reusable form-level helpers in [CRMFormBase.tsx](/Users/matias/fenix/mobile/src/components/crm/CRMFormBase.tsx:1):

- `FormScreen`
- `FormSectionLabel`
- `OptionButtonList`
- `OptionButtonItem`

These now provide a common shell for CRM form screens plus a reusable selection-list pattern used across multiple forms.

## Files remediated

- [CRMAccountForm.tsx](/Users/matias/fenix/mobile/src/components/crm/CRMAccountForm.tsx:1)
- [CRMCaseForm.tsx](/Users/matias/fenix/mobile/src/components/crm/CRMCaseForm.tsx:1)
- [CRMContactForm.tsx](/Users/matias/fenix/mobile/src/components/crm/CRMContactForm.tsx:1)
- [CRMDealCreateForm.tsx](/Users/matias/fenix/mobile/src/components/crm/CRMDealCreateForm.tsx:1)
- [WorkflowForm.tsx](/Users/matias/fenix/mobile/src/components/workflows/WorkflowForm.tsx:1)

## Post-wave effective line check

After the refactor, the targeted over-threshold functions measured as:

- `CRMCaseForm` — **60**
- `CRMDealCreateForm` — **59**
- `CRMContactForm` — **56**

The other targeted forms dropped below the reporting threshold used in the post-check (`>55` effective lines), which confirms they no longer depend on the `80` gate headroom.

## Outcome

- The wave achieved the immediate goal: the targeted shared forms are materially closer to a future `60` threshold.
- The CRM form layer now has a clearer pattern for later waves: screen shell + extracted field section + shared selector list primitive.
- `CRMCaseForm` is still exactly on the candidate threshold, so it should be treated as “compliant but with no headroom” if the repo later tightens to `60`.

## Verification

- `cd mobile && npm run typecheck`
- `cd mobile && npm run lint`
- `cd mobile && npm run quality:arch`
- `cd mobile && npm run test:coverage`
- post-change AST-based effective line audit of the five targeted files
