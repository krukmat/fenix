# Task BDD P6 - Define the BDD Metadata Contract for TST Items

**Status**: Completed
**Phase**: BDD Strategy and Traceability Consolidation
**Depends on**: `reqs/TST/*.yml`, `docs/bdd-use-cases-conversion-plan.md`
**Required by**: BDD feature rollout, TST-to-runner mapping

---

## Objective

Define the metadata contract that allows TST items to act as the formal join point
between BDD scenarios and technical runner artifacts.

---

## Scope

1. Define required BDD metadata fields for TST items
2. Define optional AGENT_SPEC-specific metadata
3. Record the contract in the master BDD strategy document
4. Record the contract in requirements workflow documentation

---

## Acceptance Criteria

- the metadata contract is explicit in the master plan
- the metadata contract is explicit in `reqs/README.md`
- required and optional fields are clearly separated

---

## Canonical TST BDD Metadata Contract

Required fields:

- `bdd.feature`
- `bdd.scenario`
- `bdd.stack`

Optional field:

- `bdd.behavior`

Field rules:

- `bdd.feature`: repository-relative path to the canonical `.feature` file
- `bdd.scenario`: exact scenario title in the `.feature` file
- `bdd.stack`: one of `go`, `bff`, `mobile`
- `bdd.behavior`: canonical AGENT_SPEC behavior identifier when the scenario belongs to an AGENT_SPEC behavior family

---

## Implemented

- documented the canonical TST BDD metadata contract
- recorded the contract in the master strategy document
- recorded the contract in `reqs/README.md`

---

## Implementation References

- `docs/bdd-use-cases-conversion-plan.md`
- `reqs/README.md`
- `docs/tasks/task_bdd_p6_tst_bdd_metadata_contract.md`

