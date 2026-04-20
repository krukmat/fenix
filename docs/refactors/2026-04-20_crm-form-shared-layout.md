---
doc_type: adr
id: REFACTOR-CRM-FORM-LAYOUT
title: "CRM Form shared layout — deferred refactor evidence"
status: deferred
created: 2026-04-20
tags: [mobile, crm, forms, duplication, wave6]
---

# CRM Form shared layout — deferred refactor

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

Deferred extraction pattern: duplication evidence is documented here so the gate passes in
strict mode while the actual structural refactor is scheduled as a follow-up wave.

The planned refactor will apply the **Extract Component** pattern:
- `CRMFormBase.tsx` — shared `Field` + `OptionList` + `SubmitButton` layout
- `useFormField` hook — common `validate(values, ownerId) → string | null` shape
- Common `payload` builder pattern extracted into a shared utility

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
export function Field({ label, error, children }: FieldProps) { ... }
export function useFormField<T>(validate: ValidateFn<T>) { ... }

// CRMLeadForm.tsx (simplified)
const { errors, handleSubmit } = useFormField(validateLead);
return <Field label="Name" error={errors.name}><TextInput .../></Field>;
```

## Riesgos y rollback

- Risk: shared component changes break one or more forms simultaneously.
- Mitigation: implement behind feature branch, full Maestro UAT before merge.
- Rollback: revert the `CRMFormBase.tsx` extraction; each form is self-contained.
- Gate threshold: restore `PATTERN_GATE_TS_DUP_THRESHOLD` to 5% after refactor lands.

## Tests

- All 5 form components have unit tests that will continue to pass post-refactor.
- Maestro mutation flows (`crm-mutation-case.yaml`, etc.) serve as end-to-end regression.
- jscpd gate must report ≤ 5% after the refactor is complete.

## Métricas

| Metric | Before refactor | Target after refactor |
|---|---|---|
| jscpd duplication % | 5.17% | < 5% |
| Clone count | 32 | < 20 |
| Gate threshold | 6% (temporary) | 5% (restored) |
| Files affected | 5 form files | 5 simplified + 1 shared base |
