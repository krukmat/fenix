# Consolidated BDD Strategy for the Full Use Case Catalog

> Date: 2026-03-28
> Status: proposed
> Scope sources of truth: `docs/requirements.md`, `docs/agent-spec-overview.md`, `docs/agent-spec-use-cases.md`, `docs/agent-spec-traceability.md`
> Traceability base: `reqs/README.md`, `cmd/frtrace/main.go`, `Makefile`

---

## Summary

This document defines the unified BDD strategy for the full use case catalog present in the repository.

The scope is not limited to the business use cases defined in `docs/requirements.md`. It also includes the AGENT_SPEC top-level use cases and the detailed `BEHAVIOR` scenarios defined in the AGENT_SPEC canonical documents.

BDD will be introduced as an executable behavior layer on top of the current traceability and testing system, not as a replacement for it.

Precondition:

- all top-level `UC` identifiers must be formalized in Doorstop before BDD is introduced on top of them
- BDD is not the mechanism used to consolidate undocumented or documentation-only use cases
- the UC catalog must first become a traceable requirements artifact layer, and only then become the parent layer for Gherkin features

The target traceability chain is:

`UC -> FR -> BEHAVIOR/Feature -> TST -> Runner`

The current system remains valid:

- `UC` remains the stable top-level capability identifier
- `FR` remains the functional requirement source of truth
- `TST` remains the technical test artifact source of truth
- current Go, BFF, Mobile, contract, and Detox suites remain active

BDD is the strategy that consolidates all business and platform use cases into one executable behavior model.

---

## Task Tracking

Master tracking document:

- `docs/tasks/task_bdd_use_cases_conversion_master.md`

Current TODO status:

- `P1` Consolidate the full top-level UC catalog: completed in planning
- `P2` Create the Doorstop UC layer: completed
- `P3` Define the UC Doorstop schema and authoring rules: completed
- `P4` Extend traceability tooling for the UC layer: completed
- `P5` Update the requirements workflow documentation: completed
- `P6` Define the BDD metadata contract for TST items: completed
- `P7` Add BDD-capable CI and Makefile entry points: completed
- `Gate Fix` Restore Doorstop pipeline integrity after introducing `reqs/UC`: completed
- `Wave 3 Pack 1` Create initial business feature files for `UC-S1`, `UC-C1`, and `UC-G1`: in progress
  Current status: Go baseline executable, Mobile baseline implemented and blocked locally by missing Detox Android SDK env
- `Wave 3 Pack 2` Expand Go business coverage for `UC-S2`, `UC-S3`, `UC-K1`, and `UC-D1`: completed
  Current status: Features, TST mappings, and Go baseline runner are in place and passing locally
- `Wave 3 Pack 3` Add a baseline `UC-A1` Agent Studio scenario: completed
  Current status: Feature, dedicated `TST_051` mapping, and Go baseline runner are in place
- `Wave 4 Pack 1` Add the first AGENT_SPEC baseline for `UC-A2`, `UC-A3`, and `UC-A4`: completed
  Current status: Features, dedicated `TST` mappings, and Go baseline runner are in place and passing locally
- `Wave 4 Pack 2` Add the AGENT_SPEC baseline for `UC-A5`, `UC-A6`, and `UC-A7`: completed
  Current status: Features, dedicated `TST` mappings, and Go baseline runner are in place and passing locally
- `Wave 4 Pack 3` Add the AGENT_SPEC baseline for `UC-A8` and `UC-A9`: completed
  Current status: Features, dedicated `TST` mappings, and Go baseline runner are in place and passing locally

Wave status:

- `Wave 4` baseline rollout is complete for `UC-A2` to `UC-A9` on the Go stack
- `Wave 5 Pack 1` Harden BDD traceability with AGENT_SPEC behavior tags and FR-link consistency: completed
  Current status: `frtrace` now validates `bdd.behavior`, AGENT_SPEC behavior-family tags, and scenario `FR` tags against mapped `TST` links
- `Wave 5 Pack 2` Enforce the current BDD baseline in CI: completed
  Current status: CI and `make ci` now include the BDD traceability gate and the Go BDD baseline runner
- `Wave 5 Pack 3` Harden selected Go workflow scenarios with real domain service assertions: completed
  Current status: `UC-A2`, `UC-A3`, and `UC-A8` use the real workflow service inside the Go BDD runner

Next execution backlog:

- `Wave 3` Convert business UCs into executable BDD features
- `Wave 4` Convert AGENT_SPEC UCs into executable BDD features
- `Wave 5` Harden behavior coverage and CI enforcement

Rule:

- every task executed for this plan must update both this master plan and its corresponding `docs/tasks` document

Task documents created so far:

- `docs/tasks/task_bdd_use_cases_conversion_master.md`
- `docs/tasks/task_bdd_p2_uc_doorstop_layer.md`
- `docs/tasks/task_bdd_p3_uc_schema_and_authoring.md`
- `docs/tasks/task_bdd_p4_uc_traceability_tooling.md`
- `docs/tasks/task_bdd_p5_requirements_workflow_docs.md`
- `docs/tasks/task_bdd_p6_tst_bdd_metadata_contract.md`
- `docs/tasks/task_bdd_p7_ci_and_runner_entrypoints.md`
- `docs/tasks/task_bdd_pipeline_doorstop_fix.md`

---

## Consolidated Use Case Catalog

This strategy covers the complete top-level `UC` catalog already present in the repository.

### Business Use Cases from `docs/requirements.md`

- `UC-S1` Sales Copilot
- `UC-S2` Prospecting Agent
- `UC-S3` Deal Risk Agent
- `UC-C1` Support Agent
- `UC-K1` KB Agent
- `UC-D1` Data Insights Agent
- `UC-G1` Governance
- `UC-A1` Agent Studio

### AGENT_SPEC Use Cases from the Canonical AGENT_SPEC Set

- `UC-A2` Workflow Authoring
- `UC-A3` Workflow Verification and Activation
- `UC-A4` Workflow Execution
- `UC-A5` Signal Detection and Lifecycle
- `UC-A6` Deferred Actions
- `UC-A7` Human Override and Approval
- `UC-A8` Workflow Versioning and Rollback
- `UC-A9` Agent Delegation

### Detailed AGENT_SPEC Behavior Layer

The AGENT_SPEC documents define an additional scenario layer under the top-level UC catalog:

- `define_workflow*`
- `verify_workflow*`
- `execute_workflow*`
- `detect_signal*`
- `defer_action*`
- `human_override*`
- `version_workflow*`
- `delegate_workflow*`

These `BEHAVIOR` identifiers are not separate top-level UCs. They are detailed executable scenarios inside `UC-A2` to `UC-A9`, and BDD must treat them as first-class scenario identifiers.

---

## Scope and Non-Goals

In scope:

- all top-level use cases already defined in the repository
- all AGENT_SPEC behavior families already defined in canonical docs
- consolidation of the full UC catalog into one BDD strategy
- extension of Doorstop and traceability tooling so BDD is auditable
- reuse of the current Go, BFF, and Mobile test stacks

Out of scope:

- rewriting all technical unit tests into Gherkin
- replacing Doorstop
- replacing `cmd/frtrace`
- replacing current CI gates

---

## Prerequisite Tasks

The following tasks are mandatory and belong to this same plan. BDD execution must not start until these prerequisite tasks are complete.

### Task P1: Consolidate the Full Top-Level UC Catalog

- collect all top-level UC identifiers from `docs/requirements.md` and the canonical AGENT_SPEC document set
- resolve any naming ambiguity in favor of the canonical source documents already declared in this plan
- freeze the consolidated top-level UC catalog as the single implementation baseline for BDD rollout

Done when:

- every top-level UC already present in repository documentation is listed in this plan
- there is no remaining documented top-level UC outside the consolidated catalog

### Task P2: Create the Doorstop UC Layer

- add a new Doorstop family at `reqs/UC/*.yml`
- create one Doorstop item for every top-level UC in the consolidated catalog
- ensure every UC item has a stable identifier, title, summary, and links to implementing FR items
- ensure AGENT_SPEC UC items also declare their canonical behavior family prefix

Done when:

- every top-level UC is represented in Doorstop
- the UC layer is no longer documentation-only

Current status:

- completed
- implementation reference: `docs/tasks/task_bdd_p2_uc_doorstop_layer.md`

### Task P3: Define the UC Doorstop Schema and Authoring Rules

- define the required YAML fields for `reqs/UC/*.yml`
- define naming, file naming, linking, and review rules for UC items
- document how business UCs and AGENT_SPEC UCs are authored consistently

Done when:

- the UC Doorstop format is explicit and repeatable
- contributors can add or modify UC items without inventing new conventions

Current status:

- completed
- implementation reference: `docs/tasks/task_bdd_p3_uc_schema_and_authoring.md`

### 3.1 UC Doorstop Schema

Canonical file location:

- `reqs/UC/*.yml`

Canonical filename rule:

- `UC_<domain><number>.yml`
- examples:
  - `UC_S1.yml`
  - `UC_C1.yml`
  - `UC_A4.yml`

Required fields for every UC item:

- `active`
- `derived`
- `header`
- `level`
- `links`
- `normative`
- `ref`
- `reviewed`
- `text`

Optional field:

- `behavior_family`

Field constraints:

- `active`: `true` for active catalog entries
- `derived`: `false`
- `header`: empty string under the current repo convention
- `level`: `1` for all top-level UC items
- `links`: only `FR_*` items that already exist in Doorstop
- `normative`: `true`
- `ref`: canonical source document path
- `reviewed`: `null` until the review workflow is formalized for UC items
- `text`: canonical UC title plus concise English summary
- `behavior_family`: only for AGENT_SPEC UCs

Authoring rules:

- business UCs use `docs/requirements.md` in `ref`
- AGENT_SPEC UCs use `docs/agent-spec-overview.md` in `ref`
- the canonical identifier is derived from filename, while the `text` field uses the hyphenated form such as `UC-S1`
- do not create placeholder FR links for FRs that are not yet present in `reqs/FR`
- AGENT_SPEC UC items must use the canonical family name already declared in AGENT_SPEC docs, such as `execute_workflow*`

### Task P4: Extend Traceability Tooling for the UC Layer

- extend `cmd/frtrace/main.go` so UC items are loaded and validated alongside FR and TST items
- validate that every UC item links to real FR items
- validate that the full documented UC catalog is present in Doorstop
- keep current FR/TST validation behavior intact

Done when:

- traceability checks fail if a documented UC is missing from Doorstop
- traceability checks fail if a UC references invalid FR links

Current status:

- completed
- implementation reference: `docs/tasks/task_bdd_p4_uc_traceability_tooling.md`

### Task P5: Update the Requirements Workflow Documentation

- update `reqs/README.md` to include UC creation and maintenance
- define the order of work as `UC -> FR -> TST` before BDD rollout begins
- document that BDD is layered on top of the established Doorstop graph

Done when:

- the repo documents a single official requirements and traceability workflow

Current status:

- completed
- implementation reference: `docs/tasks/task_bdd_p5_requirements_workflow_docs.md`

### Task P6: Define the BDD Metadata Contract for TST Items

- define `bdd.feature`
- define `bdd.scenario`
- define `bdd.stack`
- define optional `bdd.behavior` for AGENT_SPEC scenarios

Done when:

- TST items can act as the formal join point between feature scenarios and technical runners

Current status:

- completed
- implementation reference: `docs/tasks/task_bdd_p6_tst_bdd_metadata_contract.md`

### 6.1 Canonical TST BDD Metadata Contract

Required metadata fields:

- `bdd.feature`
- `bdd.scenario`
- `bdd.stack`

Optional metadata field:

- `bdd.behavior`

Field rules:

- `bdd.feature`: repository-relative path to the canonical `.feature` file
- `bdd.scenario`: exact scenario title inside the feature file
- `bdd.stack`: one of `go`, `bff`, `mobile`
- `bdd.behavior`: canonical AGENT_SPEC behavior identifier when applicable

### Task P7: Add BDD-Capable CI and Makefile Entry Points

- define `bdd-trace-check`
- define `test-bdd-go`
- define `test-bdd-bff`
- define `test-bdd-mobile`
- define `test-bdd`
- keep current CI and quality gates active

Done when:

- the repo has explicit entry points for BDD validation and execution

Current status:

- completed
- implementation reference: `docs/tasks/task_bdd_p7_ci_and_runner_entrypoints.md`

---

## BDD Strategy

### 1. Canonical Hierarchy

The repository will use this hierarchy for behavior traceability:

1. canonical documentation
2. `UC` item
3. linked `FR` items
4. `BEHAVIOR` or business scenario
5. `.feature` scenario
6. `TST` item
7. technical runner artifact

Canonical documentation sources:

- `docs/requirements.md` for business use cases
- `docs/agent-spec-overview.md` for AGENT_SPEC top-level naming
- `docs/agent-spec-use-cases.md` for AGENT_SPEC behavior scenarios
- `docs/agent-spec-traceability.md` for AGENT_SPEC identifier rules

### 1.1 Doorstop-First Consolidation Rule

Doorstop consolidation is a mandatory prerequisite for the BDD rollout.

The required sequence is:

1. collect the full top-level UC catalog from canonical documentation
2. formalize every top-level UC in `reqs/UC/*.yml`
3. link each UC item to its implementing `FR_*` items
4. extend traceability tooling so UC items are validated like FR and TST items
5. only after that, add `.feature` files and BDD runner metadata

This rule exists to avoid building Gherkin scenarios on top of identifiers that are still documentation-only and not traceable by tooling.

### 2. Traceability Model

BDD must make all of the following explicit:

- one scenario belongs to exactly one top-level `UC`
- one scenario links to one or more `FR`
- one AGENT_SPEC scenario may also link to one `BEHAVIOR`
- one scenario maps to exactly one `TST`
- one `TST` maps to one technical runner artifact

Business scenarios do not require a `BEHAVIOR` identifier unless one is explicitly defined.

AGENT_SPEC scenarios do require a `BEHAVIOR` identifier because that layer already exists as canonical design input.

### 3. Doorstop Extension

Add a new Doorstop family:

- `reqs/UC/*.yml`

Create one item for every top-level UC listed in this document.

This is the first implementation step, not an optional enhancement.

Each UC item must:

- use the canonical top-level use case identifier
- include the title and short capability summary
- link to relevant `FR_*` items
- remain implementation-agnostic

For AGENT_SPEC capabilities, the UC item must also declare the behavior family prefix:

- `define_workflow*`
- `verify_workflow*`
- `execute_workflow*`
- `detect_signal*`
- `defer_action*`
- `human_override*`
- `version_workflow*`
- `delegate_workflow*`

BDD implementation must not start until the full UC catalog has been added to Doorstop and validated by tooling.

Add BDD metadata to `reqs/TST/*.yml`:

- `bdd.feature`
- `bdd.scenario`
- `bdd.stack`
- optional `bdd.behavior`

### 4. Feature File Strategy

Add Gherkin features under `features/`.

Top-level file naming rule:

- one or more feature files per `UC`
- use stable names derived from the canonical UC ID

Examples:

- `features/uc-s1-sales-copilot.feature`
- `features/uc-c1-support-agent.feature`
- `features/uc-a4-workflow-execution.feature`
- `features/uc-a7-human-override-and-approval.feature`

When a UC is large, split by behavior family or domain slice:

- `features/uc-c1-support-agent-resolution.feature`
- `features/uc-c1-support-agent-handoff.feature`
- `features/uc-a3-workflow-verification.feature`
- `features/uc-a3-workflow-activation.feature`

All feature files must be written in English.

### 5. Scenario Tagging Standard

Every BDD scenario must use these minimum tags:

- `@UC-S1`
- `@FR-200`
- `@TST-050`
- one stack tag: `@stack-go`, `@stack-bff`, or `@stack-mobile`

AGENT_SPEC scenarios must additionally include the behavior tag:

- `@behavior-define_workflow`
- `@behavior-verify_workflow`
- `@behavior-execute_workflow_policy_blocked`

Optional tags are allowed for classification:

- `@happy`
- `@error`
- `@approval`
- `@handoff`
- `@abstention`
- `@audit`
- `@policy`
- `@warning`
- `@rollback`
- `@replay`

### 6. Runner Ownership Rule

Each scenario must have exactly one primary execution owner:

- `@stack-go`
- `@stack-bff`
- `@stack-mobile`

The same business capability may be covered by multiple technical tests at different layers, but each BDD scenario must have one canonical runner only.

### 7. Step Reuse Rule

BDD step implementations must reuse the existing test harness wherever possible.

Do not duplicate:

- auth helpers
- seed helpers
- mobile navigation helpers
- HTTP setup
- domain orchestration already covered by current fixtures

Use the approved stack:

- Go with `godog`
- BFF with `jest-cucumber`
- Mobile/Detox with `jest-cucumber`

Step layout:

- `tests/bdd/go/`
- `bff/tests/bdd/`
- `mobile/e2e/bdd/`

---

## Full Use Case Coverage Strategy

All top-level UCs in the repository must be represented in the BDD strategy.

### Business UC Coverage

#### `UC-S1` Sales Copilot

Primary runner: `mobile`

Minimum scenario set:

- launch from account detail with account context
- launch from deal detail with deal context
- show evidence-backed answer or evidence panel
- safe fallback when context or evidence is insufficient

#### `UC-S2` Prospecting Agent

Primary runner: `go` or `bff`, based on the actual orchestration entrypoint

Minimum scenario set:

- research prospect context
- generate outreach draft
- create follow-up task or action proposal
- gate execution when policy requires it

#### `UC-S3` Deal Risk Agent

Primary runner: `go`

Minimum scenario set:

- detect a deal at risk
- explain risk using evidence
- suggest mitigation
- avoid unsupported claims when evidence is weak

#### `UC-C1` Support Agent

Primary runner: `go`

Minimum scenario set:

- resolve case with sufficient evidence
- abstain with insufficient evidence
- hand off to human with preserved context
- require approval for sensitive action

#### `UC-K1` KB Agent

Primary runner: `go`

Minimum scenario set:

- generate KB draft from support resolution
- preserve evidence links
- route draft to review
- reject invalid promotion or publication

#### `UC-D1` Data Insights Agent

Primary runner: `go` or `bff`

Minimum scenario set:

- answer analytical query with evidence
- reject unsupported conclusion
- preserve source traceability
- fail safely when grounding is insufficient

#### `UC-G1` Governance

Primary runner: `go`

Minimum scenario set:

- inspect agent run and audit trace
- replay when allowed
- reject replay or rollback when denied by policy
- preserve audit trail for approval and denial

#### `UC-A1` Agent Studio

Primary runner: `go` or `bff`

Minimum scenario set:

- create skill, policy, eval, or behavior contract
- validate before promotion
- promote to production when checks pass
- reject promotion when governance checks fail

### AGENT_SPEC UC Coverage

#### `UC-A2` Workflow Authoring

Primary runner: `go`

BDD must cover the `define_workflow*` behavior family.

Minimum scenario set:

- draft workflow creation
- creation with spec source
- duplicate name rejection
- missing fields rejection
- non-draft edit rejection
- empty DSL rejection
- invalid syntax stored in draft
- size limit rejection
- concurrent update last-write-wins

#### `UC-A3` Workflow Verification and Activation

Primary runner: `go`

BDD must cover the `verify_workflow*` behavior family.

Minimum scenario set:

- successful verification
- violation reporting
- verification without spec
- syntax error reporting
- warnings without failure
- re-verification after correction
- activation after verification
- wrong status rejection
- incomplete spec warning path
- unknown verb rejection

#### `UC-A4` Workflow Execution

Primary runner: `go`

BDD must cover the `execute_workflow*` behavior family.

Minimum scenario set:

- successful workflow execution
- conditional branch skipped
- tool execution failure path
- policy-blocked execution
- approval-required execution
- runtime step audit trail
- final success and final failure state handling

#### `UC-A5` Signal Detection and Lifecycle

Primary runner: `go`

BDD must cover the `detect_signal*` behavior family.

Minimum scenario set:

- signal created from evidence-backed event
- signal visible in downstream views
- stale or invalid evidence handling
- lifecycle update or closure

#### `UC-A6` Deferred Actions

Primary runner: `go`

BDD must cover the `defer_action*` behavior family.

Minimum scenario set:

- deferred action scheduled successfully
- deferred action resumed at the correct time
- invalid schedule or resume rejection
- audit of scheduled and resumed execution

#### `UC-A7` Human Override and Approval

Primary runner: `go`

BDD must cover the `human_override*` behavior family.

Minimum scenario set:

- approval requested before sensitive action
- action proceeds after approval
- action blocked after denial
- expiration handling
- override state preserved in run audit trail

#### `UC-A8` Workflow Versioning and Rollback

Primary runner: `go`

BDD must cover the `version_workflow*` behavior family.

Minimum scenario set:

- create new workflow version
- activate new version and archive prior active version
- rollback to prior valid version
- reject invalid rollback target

#### `UC-A9` Agent Delegation

Primary runner: `go`

BDD must cover the `delegate_workflow*` behavior family.

Minimum scenario set:

- delegate to another agent successfully
- preserve delegation record in run trace
- reject invalid delegation target
- handle delegated failure safely

### Behavior-Level Strategy

For AGENT_SPEC, the strategy does not stop at top-level UC coverage.

Every defined `BEHAVIOR` in `docs/agent-spec-use-cases.md` must end up in one of these states:

- directly implemented as one Gherkin scenario
- grouped into a scenario outline with explicit examples
- mapped as an edge case under a parent scenario with explicit trace metadata

No `BEHAVIOR` documented in the canonical AGENT_SPEC set should remain outside the BDD strategy.

---

## Traceability Tooling Changes

### `cmd/frtrace`

Extend `cmd/frtrace/main.go` so validation covers:

- active UC items
- linked FR items
- AGENT_SPEC behavior identifiers
- tagged feature files
- mapped TST items
- referenced runner artifacts

The scanner must validate:

- every active top-level UC has at least one feature scenario
- every AGENT_SPEC behavior family has scenario coverage
- every scenario tagged with `@UC-*` refers to an existing UC item
- every scenario tagged with `@behavior-*` refers to a canonical behavior
- every FR linked from a UC appears in at least one scenario
- every `@TST-*` tag maps to an existing TST item
- every TST with BDD metadata resolves to a real scenario
- no orphan UC, FR, BEHAVIOR, or TST tags exist

### `reqs/README.md`

Update the documented requirements workflow so new capability work follows:

1. define or update `UC`
2. define or update linked `FR`
3. define or update AGENT_SPEC `BEHAVIOR` when applicable
4. create or update `.feature`
5. create or update `TST`
6. implement or update runner steps
7. validate classic and BDD traceability

---

## CI and Quality Gates

### New Make Targets

Add:

- `bdd-trace-check`
- `test-bdd-go`
- `test-bdd-bff`
- `test-bdd-mobile`
- `test-bdd`

### Rollout Stages

#### Stage 1

- `bdd-trace-check` required
- BDD runners informative only

#### Stage 2

- `test-bdd-go` required
- `test-bdd-bff` required
- `test-bdd-mobile` required only in Detox-capable CI jobs

### Compatibility Rule

Current gates remain active:

- `make trace-check`
- `make test`
- `make contract-test`
- current mobile quality checks

BDD extends the quality model. It does not replace it.

---

## Execution Waves

### Wave 1: Doorstop UC Consolidation

Deliver:

- unified UC catalog in `reqs/UC`
- identification of all AGENT_SPEC behavior families
- normalized cross-document source of truth
- updated traceability validation for the UC layer
- documented UC schema and authoring rules
- updated `reqs/README.md` workflow

Exit criteria:

- every top-level UC already present in documentation exists as a Doorstop UC item
- every UC item links to its relevant FR items
- no documented UC remains outside the Doorstop traceability graph
- the repo documents how UC items must be created and maintained

### Wave 2: BDD Infrastructure

Deliver:

- `features/`
- TST BDD metadata
- `frtrace` extension
- `Makefile` BDD targets
- runner-level BDD entry points for Go, BFF, and Mobile

### Wave 3: Business UC Conversion

Convert:

- `UC-S1`
- `UC-S2`
- `UC-S3`
- `UC-C1`
- `UC-K1`
- `UC-D1`
- `UC-G1`
- `UC-A1`

### Wave 4: AGENT_SPEC UC Conversion

Convert:

- `UC-A2`
- `UC-A3`
- `UC-A4`
- `UC-A5`
- `UC-A6`
- `UC-A7`
- `UC-A8`
- `UC-A9`

### Wave 5: Behavior Hardening

Enforce:

- every active UC has executable BDD coverage
- every AGENT_SPEC behavior family has scenario coverage
- every UC has at least one happy-path and one alternate-path scenario
- every linked FR appears in BDD coverage
- CI blocks missing BDD traceability

---

## Acceptance Criteria

This strategy is considered implemented when all of the following are true:

- all top-level UCs already defined in the repository exist as traceable UC artifacts
- all top-level UCs have executable English `.feature` coverage
- all canonical AGENT_SPEC behavior families are represented in BDD coverage
- every BDD scenario is tagged with valid `UC`, `FR`, `TST`, and stack tags
- AGENT_SPEC scenarios also use valid `@behavior-*` tags
- `cmd/frtrace` validates classic and BDD traceability together
- `Makefile` exposes dedicated BDD targets
- CI enforces the agreed BDD rollout
- current technical tests remain active and unchanged in role

---

## Assumptions

- `docs/requirements.md` is the canonical source for business UCs.
- `docs/agent-spec-overview.md`, `docs/agent-spec-use-cases.md`, and `docs/agent-spec-traceability.md` are the canonical source for AGENT_SPEC UCs and behavior identifiers.
- English is mandatory for BDD features and scenarios.
- Each BDD scenario has one primary runner only.
- Full pipeline execution continues to rely on the repo's current POSIX/WSL-compatible CI assumptions.
