---
doc_type: audit
title: "Mobile useSSE remediation audit"
status: completed
owner: "Workflow / QA Governance"
created: 2026-06-28
updated: 2026-06-28
tags: [mobile, qa, maintainability, hooks, sse]
---

# Mobile useSSE remediation audit

## Scope executed

- Refactored `mobile/src/hooks/useSSE.ts` into a thinner hook facade.
- Extracted request-body building, initial-message creation, assistant-message updates, stream-message handling, and client closing into `mobile/src/hooks/useSSE.helpers.ts`.
- Added direct helper coverage in `mobile/__tests__/hooks/useSSE.helpers.test.ts`.

## Decomposition pattern established

- Keep the hook focused on state ownership, auth lookup, and client wiring.
- Move pure stream-event and message-transformation logic into helper functions that are easy to unit test.
- Keep teardown and client lifecycle operations behind tiny, explicit helpers instead of repeating inline close logic.

## Files changed

- `mobile/src/hooks/useSSE.ts`
- `mobile/src/hooks/useSSE.helpers.ts`
- `mobile/__tests__/hooks/useSSE.helpers.test.ts`

## Verification

- `python3 scripts/check-maintainability.py --files 'mobile/src/hooks/useSSE.ts' 'mobile/src/hooks/useSSE.helpers.ts' 'mobile/__tests__/hooks/useSSE.helpers.test.ts'`
- `cd mobile && CI=1 npx jest --config jest.logic.config.ts --runInBand --watchman=false --forceExit --testTimeout=30000 --runTestsByPath '__tests__/hooks/useSSE.test.ts' '__tests__/hooks/useSSE.context.test.ts' '__tests__/hooks/useSSE.helpers.test.ts'`
- `bash scripts/qa-mobile-prepush.sh`

## Outcome

- The last planned mobile maintainability special-case file now follows the same extraction principles as the route/component remediation waves.
- Mobile QA remained green after the hook split, including the newly added helper tests.
- The staged remediation program is complete enough to support a focused readiness reassessment for a future `80 -> 60` threshold move.
