---
doc_type: audit
id: audit_bdd_post_repositioning_2026_04_06
title: BDD Post-Repositioning Audit
status: completed
created: 2026-04-06
updated: 2026-04-06
tags: [bdd, governance, wedge, audit]
---

# BDD Post-Repositioning Audit

## Scope

Audit of the BDD layer after the wedge repositioning work that made backend contracts canonical for `UC-S1`, hardened `UC-C1`, and narrowed `UC-G1` to governance surfaces that are actually implemented.

## Decisions Closed

- `@stack-go` is now the canonical backend/contract runner.
- `@stack-mobile` remains smoke-only for UX entrypoints.
- `@deferred` scenarios are intentionally excluded from the default Go suite.
- `UC-S1` canonical coverage moved from mobile-entrypoint semantics to backend `sales-brief`.
- `UC-S3` is no longer presented as executable canonical coverage while the `deal_risk` agent is still missing.
- `UC-G1` replay and rollback moved out of the main passing suite and into deferred coverage.

## Implementation Result

- `tests/bdd/go/bdd_test.go` now runs `@stack-go && not @deferred`.
- `UC-S1`, `UC-C1`, and `UC-G1` run against a DB-backed harness with real handlers and deterministic LLM/evidence stubs.
- `workflow_runtime_bdd.go` remains the real harness base for `UC-A4` and `UC-A6`.
- Baseline/stub coverage for non-wedge UCs remains explicit in `tests/bdd/go/README.md`.

## Traceability Result

- `TST_047` through `TST_050` now cover backend canonical `UC-S1`.
- The legacy stray `TST047.yml` record was removed to eliminate duplicate and contradictory traceability.
- `UC`, `TST`, `features/README.md`, `tests/bdd/go/README.md`, and `docs/dashboards/fr-uc-status.md` were aligned to the same truth.

## Deferred Coverage

- `UC-S3` remains `@deferred`.
- Governance replay and rollback remain `@deferred` until there is a real FR-backed implementation path.

## Verification

- `go test ./tests/bdd/go/...`
- `make bdd-trace-check`
- `go test ./...`
