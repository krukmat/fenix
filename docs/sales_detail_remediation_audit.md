---
doc_type: audit
title: "Mobile wave 4 sales detail remediation audit"
status: completed
owner: "Workflow / QA Governance"
created: 2026-06-28
updated: 2026-06-28
tags: [mobile, qa, maintainability, sales]
---

# Mobile wave 4 sales detail remediation audit

## Scope executed

- Refactored the sales deal detail route into a thinner coordinator backed by a dedicated sales detail composition component.
- Refactored the sales brief route into a thinner coordinator backed by a dedicated brief content component.

## Decomposition pattern established

- Keep route screens limited to params, data loading, header config, and mutation wiring.
- Move sales-specific summary cards, action groups, and detail sections into domain presentation components.
- Reuse the shared centered loading/error states introduced in earlier waves instead of repeating route-local state branches.

## Files changed

- `mobile/app/(tabs)/sales/deal-[id].tsx`
- `mobile/app/(tabs)/sales/[id]/brief.tsx`
- `mobile/src/components/sales/SalesDealDetailContent.tsx`
- `mobile/src/components/sales/SalesBriefContent.tsx`

## Verification

- `python3 scripts/check-maintainability.py --files 'mobile/app/(tabs)/sales/deal-[id].tsx' 'mobile/app/(tabs)/sales/[id]/brief.tsx' 'mobile/src/components/sales/SalesDealDetailContent.tsx' 'mobile/src/components/sales/SalesBriefContent.tsx'`
- `bash scripts/qa-mobile-prepush.sh`

## Outcome

- The remaining sales route screens now match the route-as-coordinator pattern used in the other mobile remediation waves.
- The sales cluster no longer depends on local oversized route render trees to stay under the current lint threshold.
- The only planned mobile maintainability wave left is the infrastructure-special-case `useSSE` refactor.
