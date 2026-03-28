# Task BDD P7 - Add BDD-Capable CI and Runner Entry Points

**Status**: Completed
**Phase**: BDD Strategy and Traceability Consolidation
**Depends on**: `Makefile`, `bff/package.json`, `mobile/package.json`
**Required by**: BDD implementation waves

---

## Objective

Add explicit entry points for BDD validation and runner execution so the repository
can adopt BDD incrementally without replacing current quality gates.

---

## Scope

1. Add BDD targets to `Makefile`
2. Add BDD runner scripts to BFF and Mobile package manifests
3. Create placeholder runner directories for Go, BFF, and Mobile BDD work
4. Keep all current CI and quality gates intact

---

## Acceptance Criteria

- `Makefile` exposes `bdd-trace-check`, `test-bdd-go`, `test-bdd-bff`, `test-bdd-mobile`, and `test-bdd`
- BFF exposes an npm BDD test script
- Mobile exposes an npm BDD test script
- stack-specific BDD directories exist
- current non-BDD targets remain unchanged in role

---

## Implemented

- added BDD Make targets
- added BFF and Mobile package scripts for BDD entry points
- created placeholder BDD directories with README files
- preserved current gates and made BDD additive

---

## Implementation References

- `Makefile`
- `bff/package.json`
- `mobile/package.json`
- `tests/bdd/go/README.md`
- `bff/tests/bdd/README.md`
- `mobile/e2e/bdd/README.md`
- `docs/tasks/task_bdd_p7_ci_and_runner_entrypoints.md`

