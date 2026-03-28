# Go BDD

This directory contains the executable Go BDD runner and step definitions.

Current scope:

- executes `@stack-go` scenarios from `features/`
- business baseline coverage for:
  - `UC-C1`
  - `UC-G1`
  - `UC-S2`
  - `UC-S3`
  - `UC-K1`
  - `UC-D1`
  - `UC-A1`
- AGENT_SPEC baseline coverage for:
  - `UC-A2` to `UC-A9`
- partial domain-backed hardening for:
  - `UC-A2`
  - `UC-A3`
  - `UC-A8`

Entry point:

- `go test ./tests/bdd/go/...`
