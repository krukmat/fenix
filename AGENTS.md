# AGENTS.md

## Push Policy

- `git push` is the final step, not the first validation step.
- Before any push, run all relevant local QA gates for the area touched by the change.
- If a required local gate cannot be executed due to environment limits, stop and report it before pushing.

## Mobile Rule

When a change touches `mobile/` or shared files that affect mobile CI, the minimum required local gates are:

- `bash scripts/check-no-inline-eslint-disable.sh`
- `cd mobile && npm run typecheck`
- `cd mobile && npm run lint`
- `cd mobile && npm run quality:arch`
- `cd mobile && npm run test:coverage`

Preferred shortcut:

- `bash scripts/qa-mobile-prepush.sh`

## Hooks

- Install repository hooks with `make install-hooks`.
- The `pre-push` hook runs the mobile QA script when the pending push includes `mobile/` or mobile-related CI files.
- There is no bypass. Fix the failing gate before pushing.
