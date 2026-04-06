# Go BDD

This directory contains the executable Go BDD runner and step definitions.

Current scope:

- executes `@stack-go and not @deferred` scenarios from `features/`
- integration-backed canonical coverage for:
  - `UC-S1`
  - `UC-C1`
  - `UC-G1`
  - `UC-A4`
  - `UC-A6`
- baseline or stub coverage for:
  - `UC-S2`
  - `UC-K1`
  - `UC-D1`
  - `UC-A1`
- AGENT_SPEC baseline coverage for:
  - `UC-A2` to `UC-A9`
- deferred from the default suite until product/runtime support exists:
  - `UC-S3`
  - replay/rollback scenarios under `UC-G1`

Entry point:

- `go test ./tests/bdd/go/...`
