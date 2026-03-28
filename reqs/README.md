# Requirements Management (Doorstop)

## Prerequisites

`./.venv/bin/doorstop`

## Requirements Workflow

The project now uses a Doorstop-first hierarchy:

`UC -> FR -> TST`

BDD is layered on top of this graph and must not be introduced before the Doorstop
chain exists.

## Adding a New Capability or Requirement

1. Create or update the top-level UC item in `reqs/UC`.
2. Create or update the relevant FR item in `reqs/FR`.
3. Link the UC item to all implementing FR items.
4. Add tests with `// Traces: FR-NNN` where applicable.
5. Create or update the TST item in `reqs/TST` and set `ref:`.
6. Implement feature behavior.
7. Update `docs/openapi.yaml` if API changes.
8. Validate:
   - `./.venv/bin/doorstop`
   - `make trace-check`
   - `make test`
   - `make contract-test`

## UC Authoring Notes

- File location: `reqs/UC/*.yml`
- File naming: `UC_<domain><number>.yml`
- Business UCs use `docs/requirements.md` as `ref`
- AGENT_SPEC UCs use `docs/agent-spec-overview.md` as `ref`
- UC items link only to FR items that already exist in Doorstop
- AGENT_SPEC UC items may add `behavior_family`

## TST BDD Metadata

When BDD rollout begins, TST items become the join point between a Gherkin scenario
and a technical runner artifact.

Supported BDD metadata fields:

- `bdd.feature`
- `bdd.scenario`
- `bdd.stack`
- optional `bdd.behavior`

## Publishing Reports

`./.venv/bin/doorstop publish all ./docs/trace-report`
