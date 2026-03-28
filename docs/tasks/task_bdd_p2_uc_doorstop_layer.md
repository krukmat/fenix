# Task BDD P2 - Create the Doorstop UC Layer

**Status**: Completed
**Phase**: BDD Strategy and Traceability Consolidation
**Depends on**: `docs/bdd-use-cases-conversion-plan.md`, `docs/requirements.md`, `docs/agent-spec-overview.md`, `docs/agent-spec-use-cases.md`, `docs/agent-spec-traceability.md`
**Required by**: P3, P4, P5, BDD feature rollout

---

## Objective

Create a first-class Doorstop UC layer so the full top-level use case catalog is no
longer documentation-only and can become the parent traceability layer for BDD.

---

## Scope

1. Add a new `reqs/UC` family to the requirements tree
2. Create one UC item for each top-level business and AGENT_SPEC use case
3. Link each UC item to the FR items that already exist in `reqs/FR`
4. Preserve canonical UC identifiers from documentation
5. Record the current limitation where some documented FRs are not yet formalized in Doorstop

---

## Out of Scope

- validating UC items in `cmd/frtrace`
- updating `reqs/README.md`
- defining the final UC schema standard
- creating `.feature` files
- wiring BDD runners

---

## Expected Output

- `reqs/UC/.doorstop.yml`
- 16 UC YAML items under `reqs/UC`
- updated master BDD strategy plan
- updated master BDD task tracker

---

## Acceptance Criteria

- every top-level UC from the consolidated catalog exists under `reqs/UC`
- business UCs and AGENT_SPEC UCs preserve their canonical IDs
- UC items link to currently formalized FR items in `reqs/FR`
- AGENT_SPEC UC items declare their behavior family
- the limitation around non-Doorstop FRs is explicitly documented

---

## Implemented

- created the `reqs/UC` Doorstop family
- added the full top-level UC catalog:
  - `UC-S1`, `UC-S2`, `UC-S3`, `UC-C1`, `UC-K1`, `UC-D1`, `UC-G1`, `UC-A1`
  - `UC-A2`, `UC-A3`, `UC-A4`, `UC-A5`, `UC-A6`, `UC-A7`, `UC-A8`, `UC-A9`
- linked each UC item to the currently existing Doorstop FR items
- captured AGENT_SPEC behavior family identifiers in UC items
- updated the master plan and master tracker to reflect `P2` completion

---

## Constraints and Notes

- the repository requirements documentation references some FRs that are not yet formalized in `reqs/FR`
- this task only links UCs to FRs that already exist in Doorstop today
- completing the broader requirements normalization remains separate from this task

---

## Sources of Truth

- `docs/bdd-use-cases-conversion-plan.md`
- `docs/requirements.md`
- `docs/agent-spec-overview.md`
- `docs/agent-spec-use-cases.md`
- `docs/agent-spec-traceability.md`

---

## Implementation References

- `reqs/UC/.doorstop.yml`
- `reqs/UC/*.yml`
- `docs/tasks/task_bdd_p2_uc_doorstop_layer.md`

