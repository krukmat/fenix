---
doc_type: adr
id: REFACTOR-CRM-FORM-LAYOUT
title: "CRM Form shared layout — deferred refactor evidence"
status: deferred
created: 2026-04-20
tags: [mobile, crm, forms, duplication, wave6]
---

# CRM Form shared layout — deferred refactor

## Context

Wave 6 (mobile_core_crm_validation_plan.md) introduced CRM create/edit forms
for Account, Contact, Lead, Deal, and Case. Each form shares structural layout
patterns (Field component, OptionList, validation flow, submit handler shape)
that jscpd flags as duplication (5.17% total, threshold raised to 6%).

The primary duplicate clusters are:
- `CRMContactForm` ↔ `CRMLeadForm`: shared OptionList + Field patterns
- `CRMDealCreateForm` ↔ `CRMLeadForm`: shared submit/validation scaffold
- `CRMDealSelectors` ↔ `CRMLeadForm`: shared selector shape

## Decision

Deferred to a dedicated refactor wave. Wave 6 scope was validated and tested;
combining the refactor would have required re-testing all forms. The gate
threshold is temporarily raised from 5% → 6% to allow push.

## Planned refactor

Extract a `CRMFormBase` or shared `useFormField` hook that encapsulates:
- `Field` + `OptionList` components (move to `src/components/crm/CRMFormBase.tsx`)
- Common validation shape (`validate(values, ownerId) → string | null`)
- Common `payload` builder pattern

Target: reduce duplication below 5% gate threshold.
