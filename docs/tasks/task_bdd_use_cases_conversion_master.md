# Task BDD Master - Full Use Case Catalog Conversion

**Status**: In Progress
**Phase**: BDD Strategy and Traceability Consolidation
**Depends on**: `docs/requirements.md`, `docs/agent-spec-overview.md`, `docs/agent-spec-use-cases.md`, `docs/agent-spec-traceability.md`, `reqs/README.md`, `cmd/frtrace/main.go`
**Required by**: UC Doorstop rollout, BDD runner wiring, CI BDD gates

---

## Objective

Consolidate the full repository use case catalog into a single traceable model and
prepare the project for BDD rollout on top of Doorstop.

---

## Scope

1. Consolidate all top-level business and AGENT_SPEC use cases into one plan
2. Establish the rule that all top-level UCs must be formalized in Doorstop before BDD rollout
3. Define prerequisite tasks required before feature files and BDD runners can be introduced
4. Keep the master BDD strategy document updated as the source of planning truth
5. Maintain a visible task-level progress record under `docs/tasks`

---

## Out of Scope

- Implementing `reqs/UC/*.yml`
- Extending `cmd/frtrace/main.go`
- Adding `.feature` files
- Wiring `godog`, `jest-cucumber`, or Detox BDD runners
- Updating CI jobs or `Makefile` targets beyond planning

---

## Expected Output

- consolidated strategy document in `docs/bdd-use-cases-conversion-plan.md`
- task tracking document in `docs/tasks`
- explicit prerequisite backlog for Doorstop-first UC formalization

---

## Acceptance Criteria

- the plan includes all top-level UCs already present in repository sources of truth
- the plan includes AGENT_SPEC `BEHAVIOR` coverage strategy
- the plan states that Doorstop UC formalization is a mandatory prerequisite
- prerequisite work is expressed as explicit tasks inside the plan
- task progress is tracked in a dedicated `docs/tasks` document

---

## Task TODO List

### Completed

- consolidate business UCs from `docs/requirements.md`
- consolidate AGENT_SPEC top-level UCs from canonical AGENT_SPEC docs
- include AGENT_SPEC `BEHAVIOR` as part of the BDD strategy
- document the Doorstop-first prerequisite for UC formalization
- add prerequisite tasks to the main BDD strategy plan
- create this master task tracker in `docs/tasks`
- create the Doorstop UC layer in `reqs/UC`
- define the UC Doorstop schema and authoring rules
- update the requirements workflow documentation
- extend `cmd/frtrace` to validate the UC layer
- define TST BDD metadata conventions in implementation detail
- add BDD targets to `Makefile`
- wire BDD runners by stack

### Pending

- Wave 3: convert business UCs into executable BDD features
- Wave 4: convert AGENT_SPEC UCs into executable BDD features
- Wave 5: harden behavior coverage, TST metadata usage, and CI enforcement

---

## Sources of Truth

- `docs/bdd-use-cases-conversion-plan.md`
- `docs/requirements.md`
- `docs/agent-spec-overview.md`
- `docs/agent-spec-use-cases.md`
- `docs/agent-spec-traceability.md`
- `reqs/README.md`

---

## Implementation References

- `docs/bdd-use-cases-conversion-plan.md`
- `docs/tasks/task_bdd_use_cases_conversion_master.md`
- `docs/tasks/task_bdd_p2_uc_doorstop_layer.md`
- `docs/tasks/task_bdd_p3_uc_schema_and_authoring.md`
- `docs/tasks/task_bdd_p4_uc_traceability_tooling.md`
- `docs/tasks/task_bdd_p5_requirements_workflow_docs.md`
- `docs/tasks/task_bdd_p6_tst_bdd_metadata_contract.md`
- `docs/tasks/task_bdd_p7_ci_and_runner_entrypoints.md`
- `reqs/UC/.doorstop.yml`
- `reqs/UC/*.yml`
- `Makefile`
- `bff/package.json`
- `mobile/package.json`
