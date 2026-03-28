# Task BDD P3 - Define the UC Doorstop Schema and Authoring Rules

**Status**: Completed
**Phase**: BDD Strategy and Traceability Consolidation
**Depends on**: `reqs/UC/*.yml`, `docs/bdd-use-cases-conversion-plan.md`, `reqs/README.md`
**Required by**: P4, P5, future UC maintenance

---

## Objective

Define the canonical schema and authoring rules for the new `reqs/UC/*.yml` Doorstop
layer so contributors can maintain it consistently and traceability tooling can rely
on stable fields.

---

## Scope

1. Define the required YAML fields for UC items
2. Define optional fields allowed for AGENT_SPEC UC items
3. Define file naming and identifier conventions
4. Define linking rules from UC to FR
5. Define authoring rules for `ref`, `text`, and catalog maintenance
6. Record the current schema in the master BDD strategy document

---

## Out of Scope

- implementing `cmd/frtrace` changes
- updating `reqs/README.md`
- validating old Doorstop items against the new UC schema
- adding BDD metadata to TST items

---

## Acceptance Criteria

- the UC schema is explicitly documented
- required and optional fields are clearly separated
- file naming rules are explicit
- link rules to FR items are explicit
- business UCs and AGENT_SPEC UCs use one documented convention set
- the master plan and master tracker reflect completion of `P3`

---

## Canonical UC Schema

Required fields for every `reqs/UC/*.yml` item:

- `active`
- `derived`
- `header`
- `level`
- `links`
- `normative`
- `ref`
- `reviewed`
- `text`

Optional fields:

- `behavior_family`

Field rules:

- `active`: `true` for active catalog entries
- `derived`: always `false` for top-level UC items
- `header`: keep as empty string unless the repo standard changes
- `level`: always `1` for top-level UC items
- `links`: list of `FR_*` identifiers that already exist in Doorstop
- `normative`: always `true`
- `ref`: canonical source document path
- `reviewed`: may be `null` until Doorstop review workflow is adopted for UC items
- `text`: two-line minimum block with canonical UC title and concise summary
- `behavior_family`: only for AGENT_SPEC UC items, using the canonical family from AGENT_SPEC docs

---

## Authoring Rules

### File Naming

- use `UC_<domain><number>.yml`
- examples:
  - `UC_S1.yml`
  - `UC_C1.yml`
  - `UC_A4.yml`

### Identifier Rule

- the canonical identifier is derived from the filename
- text must use the hyphenated form:
  - `UC-S1`
  - `UC-C1`
  - `UC-A4`

### `ref` Rule

- business UCs use `docs/requirements.md`
- AGENT_SPEC UCs use `docs/agent-spec-overview.md`
- if the canonical source changes later, update `ref` accordingly

### `links` Rule

- only link to FR items that already exist in `reqs/FR`
- do not invent placeholder FR IDs
- if a documented FR does not yet exist in Doorstop, omit it and capture the gap in planning documentation

### `text` Rule

- line 1: canonical UC title with ID
- line 2: concise capability summary in English
- keep text implementation-agnostic

### AGENT_SPEC Rule

- AGENT_SPEC UC items must include `behavior_family`
- the value must match the canonical family already declared in AGENT_SPEC docs
- do not store individual behavior scenarios in Doorstop as top-level UC items

---

## Implemented

- defined the canonical UC Doorstop schema
- defined required and optional fields
- defined naming and linking conventions
- documented the AGENT_SPEC `behavior_family` rule
- recorded the schema in the master BDD strategy plan

---

## Sources of Truth

- `docs/bdd-use-cases-conversion-plan.md`
- `docs/requirements.md`
- `docs/agent-spec-overview.md`
- `docs/agent-spec-use-cases.md`
- `docs/agent-spec-traceability.md`

---

## Implementation References

- `docs/bdd-use-cases-conversion-plan.md`
- `docs/tasks/task_bdd_p3_uc_schema_and_authoring.md`
- `reqs/UC/*.yml`

