---
doc_type: audit
title: "Mobile wave 2 auth and workflow remediation audit"
status: completed
owner: "Workflow / QA Governance"
created: 2026-06-28
updated: 2026-06-28
tags: [mobile, qa, maintainability, auth, workflows]
---

# Mobile wave 2 auth and workflow remediation audit

## Scope executed

- Refactored auth route screens to consume shared input, error, submit, and route-link building blocks.
- Refactored workflow create and edit routes to consume a shared editor shell around `WorkflowForm`.

## Decomposition pattern established

- Keep route files responsible for data loading, mutation wiring, and navigation only.
- Move repeated visual form controls into shared UI primitives.
- Move repeated workflow editor composition into a reusable screen shell so route files only provide state and submit behavior.

## Files changed

- `mobile/app/(auth)/login.tsx`
- `mobile/app/(auth)/register.tsx`
- `mobile/app/(tabs)/workflows/new.tsx`
- `mobile/app/(tabs)/workflows/edit/[id].tsx`
- `mobile/src/components/ui/AuthControls.tsx`
- `mobile/src/components/workflows/WorkflowEditorScreen.tsx`
- `mobile/src/components/workflows/WorkflowForm.tsx`

## Verification

- `python3 scripts/check-maintainability.py --files 'mobile/app/(auth)/login.tsx' 'mobile/app/(auth)/register.tsx' 'mobile/app/(tabs)/workflows/new.tsx' 'mobile/app/(tabs)/workflows/edit/[id].tsx' 'mobile/src/components/ui/AuthControls.tsx' 'mobile/src/components/workflows/WorkflowEditorScreen.tsx'`
- `bash scripts/qa-mobile-prepush.sh`

## Outcome

- The touched auth and workflow route screens now rely on shared composition instead of duplicated inline shells.
- The maintainability ratchet remains green on the changed files.
- Mobile QA gates stayed green after the refactor, so the established extraction pattern is safe to reuse in later remediation waves.
