---
doc_type: adr
id: REFACTOR-CRM-FORM-LAYOUT
title: "CRM Form shared layout — deferred refactor evidence"
status: completed
created: 2026-04-20
tags: [mobile, crm, forms, duplication, wave6]
---

# CRM Form shared layout — completed refactor

## Problema previo

Wave 6 introduced CRM create/edit forms for Account, Contact, Lead, Deal, and Case.
jscpd reports 5.17% TypeScript duplication (threshold 5%). The primary duplicate clusters are:
- `CRMContactForm` ↔ `CRMLeadForm`: shared OptionList + Field patterns
- `CRMDealCreateForm` ↔ `CRMLeadForm`: shared submit/validation scaffold
- `CRMDealSelectors` ↔ `CRMLeadForm`: shared selector shape

## Motivación

Threshold temporarily raised 5% → 6% to allow push. Combining the refactor during Wave 6
would have required re-testing all five forms and broken the delivery cadence. A dedicated
refactor wave is safer and allows full regression coverage.

## Patrón aplicado

Deferred extraction pattern completed. The duplicated editable form controls and data-unwrapping
helpers were moved into a shared CRM form module, and the temporary jscpd threshold was restored
to 5%.

The refactor applied the **Extract Component** pattern:
- `CRMFormBase.tsx` — shared editable `Field`, `SubmitButton`, `FormErrorText`, `LoadingView`,
  `baseFormStyles`, `useCRMColors`, `record`, `unwrapDataArray`, and `listItems`.
- CRM create/edit forms now import the shared controls instead of defining local copies.
- `OptionList`, form validation, and payload builders remain local where behavior differs by
  workflow.

## Before

Five independent form files each contain:
```tsx
// CRMLeadForm, CRMContactForm, CRMDealCreateForm, CRMAccountForm, CRMCaseForm
const [errors, setErrors] = useState<Record<string, string>>({});
const validate = () => { ... }; // duplicated shape
return (
  <Field label="..." error={errors.x}>
    <TextInput ... />
  </Field>
);
```

## After

After the refactor:
```tsx
// CRMFormBase.tsx
export function Field(props: FieldProps) { ... }
export function SubmitButton(props: SubmitButtonProps) { ... }
export function LoadingView(props: LoadingViewProps) { ... }
export function listItems<T>(data, normalize): T[] { ... }

// CRMLeadForm.tsx (simplified)
return <Field label="Name" value={values.name} onChangeText={...} testID="crm-lead-form-name" />;
```

## Riesgos y rollback

- Risk: shared component changes break one or more forms simultaneously.
- Mitigation: targeted unit tests per migrated form plus the shared `CRMFormBase` test.
- Rollback: revert the `CRMFormBase.tsx` extraction; each form is self-contained.
- Gate threshold: `PATTERN_GATE_TS_DUP_THRESHOLD` restored to 5%.

## Tests

- Added `mobile/__tests__/components/crm/CRMFormBase.test.tsx`.
- Targeted Jest suites passed for Account, Contact, Lead, Deal, Case, DealSelectors, and
  EntityChildForms.
- `cd mobile && npm run typecheck` passed after each migrated slice.
- `make pattern-refactor-gate` passed with the restored 5% threshold.

## Métricas

| Metric | Before refactor | Final after refactor |
|---|---|---|
| jscpd duplication % | 5.17% | 2.17% |
| Clone count | 32 | 15 |
| Gate threshold | 6% (temporary) | 5% (restored) |
| Files affected | 5 form files | 7 migrated CRM files + 1 shared base + 1 shared test |
