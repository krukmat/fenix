# Task BDD P4 - Extend Traceability Tooling for the UC Layer

**Status**: Completed
**Phase**: BDD Strategy and Traceability Consolidation
**Depends on**: `reqs/UC/*.yml`, `cmd/frtrace/main.go`
**Required by**: P5, P6, P7, BDD rollout

---

## Objective

Extend the traceability scanner so the new UC Doorstop layer is validated alongside
the existing FR and TST layers.

---

## Scope

1. Load UC items from `reqs/UC`
2. Report UC counts in the traceability output
3. Validate that required top-level UCs exist in Doorstop
4. Validate that active UC items link only to FR items that exist in Doorstop
5. Preserve existing FR/TST validation behavior

---

## Out of Scope

- BDD feature parsing
- TST BDD metadata validation
- behavior-family validation
- `reqs/README.md` updates

---

## Acceptance Criteria

- `cmd/frtrace` loads UC items from `reqs/UC`
- the report includes UC counts
- the tool fails when a required top-level UC is missing
- the tool fails when an active UC item links to an FR missing from Doorstop
- FR/TST checks continue to work

---

## Implemented

- added UC loading to `cmd/frtrace/main.go`
- added validation for required consolidated top-level UC IDs
- added validation for active `UC -> FR` links
- added UC counts to the traceability report
- kept current FR/TST traceability checks intact

---

## Sources of Truth

- `docs/bdd-use-cases-conversion-plan.md`
- `reqs/UC/*.yml`
- `cmd/frtrace/main.go`

---

## Implementation References

- `cmd/frtrace/main.go`
- `docs/tasks/task_bdd_p4_uc_traceability_tooling.md`
