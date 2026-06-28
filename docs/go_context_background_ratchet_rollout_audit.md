---
doc_type: audit
title: "Go context.Background ratchet rollout"
status: completed
created: 2026-06-28
updated: 2026-06-28
tags: [audit, go, qa, context, governance, maintainability]
---

# Go context.Background ratchet rollout

## Objective

Implement the first low-risk version of the `context.Background()` ratchet as a changed-lines-only maintainability check for Go runtime paths.

## Implemented behavior

The ratchet now runs inside [scripts/check-maintainability.py](/Users/matias/fenix/scripts/check-maintainability.py:1) and flags newly added `context.Background()` uses when all of the following are true:

- the added line is in a non-test Go file;
- the file is under `internal/` or `pkg/`;
- the line is not in `cmd/`;
- the line is not one of the explicitly approved owner-root patterns.

Approved patterns in this first rollout:

- `context.WithCancel(context.Background())`
- assignment to `BackgroundContext = context.Background()`

Anything else added in scoped runtime paths now fails the maintainability gate with guidance to derive from caller context or use an explicitly owned background-root pattern.

## Scope decisions

- **Changed-lines-only**: historical baseline remains grandfathered.
- **No new hook/CI wiring required**: the ratchet rides the existing `qa-maintainability` surface already enforced in pre-push and CI.
- **Conservative exception set**: this first version does not broadly allow `context.WithTimeout(context.Background(), ...)`, because that would hide exactly the cases the team wanted surfaced for review.

## Verification

- `python3 scripts/check_maintainability_test.py`
- `python3 -m py_compile scripts/check-maintainability.py scripts/check_maintainability_test.py`
- `make qa-maintainability FENIX_MAINTAINABILITY_BASE=HEAD`

## Notes

- The implementation surface is the maintainability checker rather than ruleguard, which keeps rollout risk low and limits enforcement to new changes.
- A future second phase could migrate this from a changed-lines-only ratchet to a broader lint rule if the team wants stricter repository-wide governance.
