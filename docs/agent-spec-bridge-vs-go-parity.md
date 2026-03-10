# Bridge vs Go Parity Review

This document captures the `F3.9` comparison between:

- the Go baseline: support resolve flow in `SupportAgent`
- the bridge baseline: `resolve_support_case_parity`

## Flows Compared

### Go baseline

- agent: `support`
- path: high-confidence support resolution
- side effects:
  - `update_case`
  - `send_reply`

### Bridge baseline

- agent: `skill`
- workflow: `resolve_support_case_parity`
- steps:
  1. `SET case.status = resolved`
  2. `NOTIFY contact`
- side effects:
  - `update_case`
  - `send_reply`

## Parity Summary

| Dimension | Go support flow | Bridge workflow | Result |
|---|---|---|---|
| Run terminal status | `success` | `success` | parity |
| Case side effect | case becomes `resolved` | case becomes `resolved` | parity |
| Reply side effect | one reply note | one reply note | parity |
| Tool calls | `update_case`, `send_reply` | `update_case`, `send_reply` | parity |
| Policy enforcement | not explicit in support flow | explicit via `PolicyEngine` | intentional difference |
| Step trace shape | runtime steps only | runtime steps + `bridge_step` rows | intentional difference |
| Reasoning trace | Go-specific support reasoning | bridge output focused on steps | intentional difference |

## Interpretation

The bridge runner is good enough for Phase 4 because it already matches the Go flow on:

- functional outcome
- core side effects
- terminal run behavior

It still differs in how it models execution:

- policy is explicit in the bridge path
- trace granularity is higher in the bridge path
- reasoning payloads are not yet equivalent

Those differences are acceptable for the bridge stage because the goal is operational parity first, not identical internal representation.
