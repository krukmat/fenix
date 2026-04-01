# Waves 1-9 Audit Handoff

## 1. Purpose

This document is the audit handoff for the full nine-wave delivery set defined in `docs/parallel_requirements.md` and expanded in the wave-specific analysis documents.

The intent is not to restate every wave document. The intent is to give the auditor a clear review frame:

- what each wave is meant to close
- what each wave explicitly does not claim
- which documentation and implementation artifacts are the dependency base
- which cross-wave inconsistencies need deeper review
- where closure claims are strong, partial, blocked, or not yet traceable enough

## 2. Audit Objective

The auditor should review the nine waves as one dependency graph, not as nine isolated design notes.

The main audit questions are:

1. Is each wave scoped clearly enough to avoid overclaiming?
2. Do requirement intent, operational traceability, architecture, API contracts, and runtime implementation stay aligned?
3. Are dependency gates explicit and treated honestly when they are still unresolved?
4. Are later waves consuming earlier-wave contracts in a stable way?
5. Are there any waves whose current documentation is stronger or weaker than their implementation reality?

## 3. Source Hierarchy

Use the following source hierarchy during the audit:

1. `docs/requirements.md`
   Business intent and declared FR dependencies.
2. `reqs/FR/*.yml`
   Requirement traceability artifacts when they exist.
3. `docs/parallel_requirements.md`
   Canonical wave sequencing, dependency slicing, and workstream grouping.
4. Wave analysis documents in `docs/`
   Design and implementation planning per wave.
5. `docs/architecture.md`
   Target system model, contracts, and data shapes.
6. `docs/as-built-design-features.md`
   Current implementation baseline.
7. `docs/fr-gaps-implementation-criteria.md`
   Gap framing and closure criteria.
8. `docs/openapi.yaml`
   Published supported API surface.
9. Runtime code under `internal/`
   Effective implementation truth when checking whether documentation overstates support.

If sources conflict:

- treat `docs/requirements.md` as business intent
- treat `reqs/FR/*.yml` as requirement-traceability truth when present
- treat `docs/openapi.yaml` as the supported public API contract
- treat runtime code as implementation reality
- treat the wave documents as design intent that must not overclaim beyond the three items above

## 4. Audit Corpus

Primary planning documents:

- `docs/parallel_requirements.md`
- `docs/wave1-governance-audit-retrieval-analysis.md`
- `docs/wave2-tooling-copilot-prompt-crm-analysis.md`
- `docs/wave3-agent-runtime-handoff-analysis.md`
- `docs/wave4-unblocked-expansion-analysis.md`
- `docs/wave5-packaging-and-distribution-analysis.md`
- `docs/wave6-connector-expansion-analysis.md`
- `docs/wave7-non-llm-automation-and-triggering-analysis.md`
- `docs/wave8-dedupe-budgets-and-release-gating-analysis.md`
- `docs/wave9-behavior-contracts-carta-analysis.md`

The audit scope includes all of the planning work above. None of the wave analysis documents should be treated as optional background material. Each one defines scope boundaries, dependency handling, and closure claims that the auditor is expected to validate.

Supporting baseline documents:

- `docs/requirements.md`
- `docs/architecture.md`
- `docs/as-built-design-features.md`
- `docs/fr-gaps-implementation-criteria.md`
- `docs/implementation-plan.md`
- `docs/carta-spec.md`
- `docs/carta-implementation-plan.md`
- `docs/openapi.yaml`

## 5. Program Structure

The wave set is intentionally split into three maturity bands.

### 5.1 Waves 1-3: P0 Closure Backbone

These waves define the contracts that later waves consume.

- Wave 1: governance, audit, and retrieval foundations
- Wave 2: tooling, copilot, prompt lifecycle, and CRM hardening
- Wave 3: runtime, handoff, pending approval, and minimal catalog closure

Audit expectation:

- these waves should have the strongest documentary and implementation grounding
- if they are unstable, later waves become design fiction

### 5.2 Waves 4-5: Unblocked Expansion

These waves extend the system only where the current repo already supports a safe next slice.

- Wave 4: expansion items not blocked by external dependencies
- Wave 5: packaging and distribution, intentionally narrow

Audit expectation:

- review them as conservative, contract-first expansions
- challenge any place where they drift from that conservative posture

### 5.3 Waves 6-9: Blocked or Partially-Specified Expansion

These waves are not all equally ready. Some are explicitly blocked by unresolved prerequisite requirements.

- Wave 6: blocked by `FR-320`
- Wave 7: blocked by `FR-120`
- Wave 8: blocked by `FR-310`
- Wave 9: depends on Wave 8, especially the `FR-233` quota model

Audit expectation:

- these waves should be reviewed primarily for honesty of scope, dependency handling, and traceability discipline
- they should not be treated as implementation-ready unless their gates are resolved

## 6. Cross-Wave Conclusions

The nine-wave set is broadly coherent, but the auditor should assume that the highest risk is not in local diagram quality. The highest risk is in cross-wave contract drift.

Current top-level conclusion:

- Waves 1-3 are the strongest part of the set.
- Waves 4-5 are intentionally conservative and mostly safe if they stay narrow.
- Waves 6-9 are useful as planning documents, but several of their dependency gates and traceability anchors remain unresolved.

The most important repo-wide issues are listed below.

## 7. Cross-Wave Issues That Require Deep Review

### 7.1 Undefined Dependency Gates — RESOLVED

Three requirements were used as dependency gates but had no definition anywhere in the repo. Minimal Doorstop artifacts have been created to formalize them.

- `FR-320` — External Provider and PII Control Gate → `reqs/FR/FR_320.yml` (inactive)
- `FR-120` — Event and Trigger Foundation → `reqs/FR/FR_120.yml` (inactive)
- `FR-310` — Quality and Release Control Infrastructure → `reqs/FR/FR_310.yml` (inactive)

All three are marked `active: false` because they remain unresolved gates. Their existence as artifacts means:

- they now have a defined scope and acceptance criteria baseline
- downstream waves can reference a concrete artifact instead of a phantom identifier
- resolution (setting `active: true`) requires explicit implementation or design decision

Impact (unchanged — formalization does not resolve the gates themselves):

- Wave 1 governance notes already feel the `FR-320` ambiguity around stronger PII or provider-control claims.
- Wave 6 cannot be treated as implementation-ready while `FR-320` remains unresolved.
- Wave 7 is formally blocked by `FR-120`.
- Wave 8 is formally blocked by `FR-310`.
- Wave 9 inherits risk from Wave 8 because `FR-212` depends on a stable quota model from `FR-233`.

Audit action:

- verify the new artifacts accurately define what "resolved" means for each gate
- when a gate is resolved, update the artifact to `active: true` and add a `reviewed` hash

### 7.2 Missing Requirement Artifacts

Several later-wave requirements did not have `reqs/FR/*.yml` artifacts. The following table tracks the original gaps and their resolution status.

| FR | Affected wave | Audit implication | Status |
| --- | --- | --- | --- |
| `FR-120` | Wave 7 (gate) | event/trigger gate was used as a formal blocker with no definition | **RESOLVED** — `reqs/FR/FR_120.yml` created (inactive gate artifact) |
| `FR-310` | Wave 8 (gate) | quality/release-control gate was used as a formal blocker with no definition | **RESOLVED** — `reqs/FR/FR_310.yml` created (inactive gate artifact) |
| `FR-320` | Wave 6 (gate) | provider/PII-control gate was used as a formal blocker with no definition | **RESOLVED** — `reqs/FR/FR_320.yml` created (inactive gate artifact) |
| `FR-050` | Wave 6 | connector baseline is not fully traceable | **RESOLVED** — `reqs/FR/FR_050.yml` created (inactive) |
| `FR-005` | Wave 7 | automation requirement is not fully traceable | **RESOLVED** — `reqs/FR/FR_005.yml` created (inactive) |
| `FR-234` | Wave 7 | triggered workflow requirement is not fully traceable | **RESOLVED** — `reqs/FR/FR_234.yml` created (inactive) |
| `FR-094` | Wave 8 | dedupe closure lacks formal traceability | **RESOLVED** — `reqs/FR/FR_094.yml` created (inactive) |
| `FR-233` | Wave 8 | quota closure lacks formal traceability | **RESOLVED** — `reqs/FR/FR_233.yml` created (inactive) |
| `FR-242` | Wave 8 | standalone eval maturity lacks formal traceability | **RESOLVED** — `reqs/FR/FR_242.yml` created (inactive) |
| `FR-241` | Wave 4 | skills builder lacks formal traceability | **RESOLVED** — `reqs/FR/FR_241.yml` created (inactive) |
| `FR-212` | Wave 9 | Carta closure lacks formal traceability | **RESOLVED** — `reqs/FR/FR_212.yml` created (inactive) |

All 11 artifacts were created as `active: false` gate or deferred requirement files. They define scope and dependencies but do not claim implementation readiness.

Audit action:

- verify the new artifacts accurately reflect the requirement intent from `docs/requirements.md`
- do not accept full closure claims for these FRs until they are marked `active: true` with acceptance criteria met

### 7.3 Published API Contract Drift

Several supported or semi-supported runtime surfaces are not reflected in `docs/openapi.yaml`.

Confirmed high-value examples:

- workflow lifecycle routes: `POST /{id}/verify`, `POST /{id}/execute`, `PUT /{id}/activate`, `DELETE /{id}` exist in `routes.go` but not in `openapi.yaml`
- eval routes: 6 routes (`admin/eval/suites` CRUD + `admin/eval/run`/`runs`/`runs/{id}`) exist in `routes.go` but not in `openapi.yaml`
- tool admin lifecycle routes: `PUT /{id}`, `PUT /{id}/activate`, `PUT /{id}/deactivate`, `DELETE /{id}` exist in `routes.go` but not in `openapi.yaml`
- report routes: 6 routes (`sales/funnel`, `sales/aging`, `support/backlog`, `support/volume`, plus 2 CSV exports) exist in `routes.go` but not in `openapi.yaml`
- audit routes: `GET /audit/events`, `GET /audit/events/{id}`, `POST /audit/export` exist in `routes.go` but not in `openapi.yaml`
- agent-specific trigger routes: `POST /agents/trigger`, `POST /agents/{type}/trigger` (support, prospecting, kb, insights), `GET /agents/definitions`, `POST /agents/runs/{id}/cancel` exist in `routes.go` but not in `openapi.yaml`
- method mismatches: workflow update uses `PUT` in code but `PATCH` in spec; workflow rollback uses `PUT` in code but `POST` in spec; signal dismiss uses `PUT` in code but `POST` in spec
- phantom route: `POST /api/v1/signals` in `openapi.yaml` has no corresponding route in code (only `GET` list and `PUT` dismiss exist)
- phantom FR trace: workflow routes in `openapi.yaml` reference `FR-220` which is not defined anywhere in `docs/requirements.md` or `reqs/FR/`
- later-wave designs depend on some of these surfaces as supported lifecycle steps

**Resolution status**: The following corrections have been applied to `docs/openapi.yaml`:

- All workflow lifecycle routes (`verify`, `activate`, `execute`, `delete`, `rollback`) added with correct HTTP methods (`PUT`/`POST` matching `routes.go`)
- All 6 eval routes added under `admin/eval/` with `FR-242` traces
- Tool admin lifecycle routes (`PUT /{id}`, `PUT /{id}/activate`, `PUT /{id}/deactivate`, `DELETE /{id}`) added with `FR-202` traces
- All 6 report routes added with `FR-003` traces
- All 3 audit routes added with `FR-070` traces
- Agent trigger, cancel, definitions, and 4 agent-type-specific trigger routes added with `FR-230`/`FR-231`/`FR-232` traces
- Handoff route split into `GET` (get package) + `POST` (initiate handoff) matching `routes.go`
- Phantom `POST /signals` removed (code only implements `GET` list + `PUT` dismiss)
- Signal dismiss method corrected from `POST` to `PUT`
- Workflow update method corrected from `PATCH` to `PUT`
- Workflow rollback method corrected from `POST` to `PUT`
- All `FR-220` phantom traces replaced with correct `FR-240` (authoring) or `FR-230` (execution)
- `/metrics` endpoint added with `NFR-030` trace

Audit action:

- verify that the updated `docs/openapi.yaml` matches `routes.go` — the reconciliation above should close this finding
- any new routes added in future iterations must be documented in `openapi.yaml` before being treated as supported

### 7.4 Data Model and Architecture Drift

Two cross-wave schema or model mismatches were identified and resolved.

1. Knowledge source taxonomy drift — **RESOLVED**

- `docs/architecture.md` previously used `email|doc|call_transcript|chat|kb_article|crm_record`
- `internal/infra/sqlite/migrations/011_knowledge.up.sql` uses `email|document|kb_article|api|note|call|case|ticket|other`
- **Fix applied**: `architecture.md` updated to match the canonical migration/code taxonomy
- `source_id` field annotated as "planned — stored in metadata JSON" to match implementation reality

2. Eval schema drift — **PARTIALLY RESOLVED**

- `docs/architecture.md` models `eval_run.policy_version_id`
- migration `020_eval.up.sql` and current runtime support focus on `prompt_version_id` only
- **Fix applied**: `architecture.md` annotated `policy_version_id` as "planned, not yet in migration 020"
- This means Wave 8 eval gating is currently prompt-only; policy-version gating remains a future expansion target

Audit action:

- verify the canonical model choices are reflected in wave designs
- do not accept policy-version release gating claims until `policy_version_id` is added to the migration

### 7.5 Partial Enforcement Wiring

Some safety and behavior-control mechanisms exist in code but are not consistently wired through supported paths.

Most important example:

- `GroundsValidator` exists in `internal/domain/agent/grounds_validator.go` and is defined as a field on `RunContext`, but the production `RunContext` instances constructed in `internal/api/handlers/workflow.go` (executeDSLWorkflow), `internal/api/routes.go` (resumeRC for scheduler), and `internal/api/handlers/insights_shadow.go` do not set `GroundsValidator`. It is only wired in test code (`integration_test.go`). The guard function `groundsPolicyApplies()` in `dsl_runner.go` checks `rc.GroundsValidator != nil` and silently skips validation when it is nil — no error is raised and execution proceeds normally.

Impact:

- Wave 9 cannot claim dynamic GROUNDS enforcement as closure — it is available infrastructure, not enforced behavior
- All three production execution paths (workflow execute, scheduler resume, insights shadow) skip grounds validation silently

Audit action:

- evaluate supported execution paths only
- do not count optional or test-only wiring as full closure
- classify GROUNDS enforcement as partial infrastructure until `GroundsValidator` is injected in all production `RunContext` instances

### 7.6 Semantic Split of `FR-091`

`FR-091` behaves as two different things across the repo:

- a Wave 1 baseline around retrieval freshness and source handling already unlocked by current architecture
- a Wave 6 breadth expansion across connector families

Audit action:

- confirm the wave set preserves this split honestly
- reject any claim that Wave 1 closes the full connector-breadth interpretation of `FR-091`

## 8. Wave-by-Wave Audit Brief

### 8.1 Wave 1

Primary scope:

- governance
- approvals and policy gates
- auditability
- grounded retrieval
- reindex or freshness baseline

What it includes:

- approval-before-execution flows
- policy decisions as explicit contracts
- immutable or reconstructable audit trails
- retrieval with evidence and compliance controls
- scoped `FR-091` closure only for already-unlocked source families

What it does not claim:

- full connector-breadth ingestion expansion
- unqualified PII-complete closure if `FR-320` is still unresolved

Documentation dependencies:

- `docs/wave1-governance-audit-retrieval-analysis.md`
- `docs/parallel_requirements.md`
- `docs/requirements.md` sections `7.2` and `7.8`
- `reqs/FR/FR_060.yml`, `reqs/FR/FR_061.yml`, `reqs/FR/FR_070.yml`, `reqs/FR/FR_071.yml`
- `docs/architecture.md`
- `docs/as-built-design-features.md`
- `docs/fr-gaps-implementation-criteria.md`
- `features/uc-g1-governance.feature`
- `features/uc-a4-workflow-execution.feature`
- `features/uc-a5-signal-detection-and-lifecycle.feature`
- `features/uc-d1-data-insights-agent.feature`

Implementation anchors to inspect first:

- `internal/domain/policy/evaluator.go`
- `internal/domain/policy/approval.go`
- `internal/domain/audit/service.go`
- `internal/api/handlers/approval.go`
- `internal/api/handlers/audit.go`
- `internal/domain/knowledge/search.go`
- `internal/domain/knowledge/reindex.go`
- `internal/api/handlers/knowledge_search.go`
- `internal/api/handlers/knowledge_reindex.go`

Deep review points:

- confirm the exact boundary of `FR-091` in this wave
- verify governance and audit claims are tied to implementation-safe contracts
- verify any PII note is framed as partial if the external gate is not closed

### 8.2 Wave 2

Primary scope:

- tooling control plane
- grounded copilot
- prompt lifecycle
- CRM hardening lane

What it includes:

- tool registration and routing baseline
- grounded copilot path with policy and retrieval dependencies
- prompt versioning and promotion flow
- CRM mutation hardening limited to the documented slice

What it does not claim:

- standalone `FR-210` as its own lane
- standalone `FR-242` closure
- mobile or BFF expansion as primary workstreams
- reopening `FR-002` or `FR-003` through the CRM hardening lane

Documentation dependencies:

- `docs/wave2-tooling-copilot-prompt-crm-analysis.md`
- `docs/parallel_requirements.md`
- `docs/requirements.md` sections `7.1`, `7.3`, `7.4`, `7.6`
- `docs/openapi.yaml`
- `docs/architecture.md`
- `docs/as-built-design-features.md`
- `docs/fr-gaps-implementation-criteria.md`
- `features/uc-b1-safe-tool-routing.feature`
- `features/uc-a1-agent-studio.feature`
- `features/uc-a2-workflow-authoring.feature`
- `features/uc-a3-workflow-verification-and-activation.feature`
- `features/uc-a8-workflow-versioning-and-rollback.feature`
- `features/uc-s1-sales-copilot.feature`
- `features/uc-c1-support-agent.feature`

Implementation anchors to inspect first:

- `internal/api/handlers/prompt.go`
- `internal/api/handlers/agent.go`
- `internal/domain/agent/prompt.go`
- `internal/domain/agent/runner_registry.go`
- `internal/api/handlers/tool.go`
- `internal/domain/agent/skill_runner.go`

Deep review points:

- verify Wave 2 consumes Wave 1 contracts without redefining them
- verify prompt lifecycle assumptions stay aligned with current eval support
- verify supported API surface matches what the wave describes

### 8.3 Wave 3

Primary scope:

- runtime execution
- pending approval
- handoff
- minimal agent catalog

What it includes:

- support-style run lifecycle
- pending-approval state handling
- handoff and escalation path
- minimum catalog closure for the runtime slice

What it does not claim:

- broader automation or scheduling behavior from later waves
- quota or advanced budget semantics from later waves
- full surface closure if API documentation remains behind runtime routing

Documentation dependencies:

- `docs/wave3-agent-runtime-handoff-analysis.md`
- `docs/parallel_requirements.md`
- `docs/requirements.md` sections `7.4` and `7.5`
- `docs/wave1-governance-audit-retrieval-analysis.md`
- `docs/wave2-tooling-copilot-prompt-crm-analysis.md`
- `docs/openapi.yaml`
- `docs/architecture.md`
- `docs/as-built-design-features.md`
- `docs/fr-gaps-implementation-criteria.md`
- `features/uc-c1-support-agent.feature`
- `features/uc-a4-workflow-execution.feature`
- `features/uc-a7-human-override-and-approval.feature`
- `features/uc-s2-prospecting-agent.feature`
- `features/uc-k1-kb-agent.feature`
- `features/uc-d1-data-insights-agent.feature`

Implementation anchors to inspect first:

- `internal/api/handlers/agent.go`
- `internal/api/routes.go`
- `internal/domain/agent/orchestrator.go`
- `internal/domain/agent/dsl_runner.go`
- `internal/api/handlers/workflow.go`
- `internal/api/handlers/approval.go`

Deep review points:

- verify runtime contracts only consume frozen Wave 1 and Wave 2 boundaries
- verify handoff and run lifecycle traceability against published API surfaces
- inspect the documented versus implemented route story for run and handoff support

### 8.4 Wave 4

Primary scope:

- unblocked expansion after P0
- only where the repo already supports a safe next slice

What it includes:

- CRM extensibility slice
- freshness or replay slice
- templates or skills slice
- mobile expansion slice

What it does not claim:

- invented APIs
- invented feature files
- undocumented contracts presented as if they were baseline facts
- dedicated BDD feature files — Wave 4 has no `.feature` files; scope is validated through contract and integration tests only

Documentation dependencies:

- `docs/wave4-unblocked-expansion-analysis.md`
- `docs/parallel_requirements.md`
- `docs/requirements.md` sections `7.1`, `7.2`, `7.3`, `7.6`, `7.9`
- `docs/wave1-governance-audit-retrieval-analysis.md`
- `docs/wave2-tooling-copilot-prompt-crm-analysis.md`
- `docs/wave3-agent-runtime-handoff-analysis.md`
- `docs/architecture.md`
- `docs/openapi.yaml`
- `docs/as-built-design-features.md`
- `docs/implementation-plan.md`

Implementation anchors to inspect first:

- `internal/domain/agent/skill_runner.go`
- `internal/domain/knowledge/reindex.go`
- `internal/domain/knowledge/evidence.go`
- `internal/infra/sqlite/migrations/018_agents.up.sql`
- `internal/infra/sqlite/migrations/011_knowledge.up.sql`
- `mobile/src/services/api.ts`
- `mobile/src/services/api.agents.ts`
- `bff/src/routes/copilot.ts`

Deep review points:

- verify every derived contract is clearly labeled as derived
- verify every subtrack keeps a conservative scope boundary
- reject any area where thin documentation is being used to justify broad implementation claims

### 8.5 Wave 5

Primary scope:

- packaging and distribution

What it includes:

- packaging of agents or skills
- manifest, install, upgrade, and minimal registry semantics

What it does not claim:

- widgets as part of the initial slice
- UI extension surfaces not documented elsewhere in the repo
- dedicated BDD feature files — Wave 5 has no `.feature` files; scope is validated through contract and integration tests only

Documentation dependencies:

- `docs/wave5-packaging-and-distribution-analysis.md`
- `docs/parallel_requirements.md`
- `docs/requirements.md` `FR-052`
- `reqs/FR/FR_052.yml`
- `docs/implementation-plan.md`
- `docs/wave2-tooling-copilot-prompt-crm-analysis.md`
- `docs/wave3-agent-runtime-handoff-analysis.md`
- `docs/wave4-unblocked-expansion-analysis.md`
- `docs/architecture.md`
- `docs/as-built-design-features.md`

Implementation anchors to inspect first:

- `internal/domain/agent/skill_runner.go`
- `internal/domain/agent/prompt.go`
- `internal/domain/agent/runner_registry.go`
- `internal/infra/sqlite/migrations/018_agents.up.sql`
- `internal/infra/sqlite/migrations/020_eval.up.sql`
- `internal/infra/sqlite/queries/agent.sql`
- `internal/infra/sqlite/queries/prompt.sql`

Deep review points:

- verify the wave remains narrow
- verify any future distribution surface is published in `docs/openapi.yaml` before being treated as supported

### 8.6 Wave 6

Primary scope:

- connector expansion readiness and first connector slice after gate resolution

What it includes:

- connector-model preparation
- traceability repair work
- source taxonomy reconciliation
- breadth expansion planning for ingestion

What it does not claim:

- immediate implementation readiness while `FR-320` remains unresolved
- full `FR-091` closure by simply inheriting Wave 1 retrieval baseline

Documentation dependencies:

- `docs/wave6-connector-expansion-analysis.md`
- `docs/parallel_requirements.md`
- `docs/requirements.md` `FR-050` and `FR-091`
- `docs/as-built-design-features.md`
- `docs/fr-gaps-implementation-criteria.md`
- `reqs/FR/FR_091.yml`
- `docs/wave1-governance-audit-retrieval-analysis.md`
- `docs/wave3-agent-runtime-handoff-analysis.md`
- `docs/architecture.md`
- `docs/openapi.yaml`
- `docs/implementation-plan.md`

Implementation anchors to inspect first:

- `internal/api/handlers/knowledge_ingest.go`
- `internal/domain/knowledge/ingest.go`
- `internal/domain/knowledge/reindex.go`
- `internal/infra/sqlite/migrations/011_knowledge.up.sql`

Deep review points:

- require a disposition for `FR-320`
- require a requirement artifact or explicit resolution path for `FR-050`
- reconcile `source_type` taxonomy between architecture and migration
- validate the Wave 1 versus Wave 6 split on `FR-091`

### 8.7 Wave 7

Primary scope:

- non-LLM automation
- workflow triggering

What it includes:

- workflow trigger model analysis
- schedule and webhook entry-point planning
- manual-to-automated execution bridge

What it does not claim:

- implementation readiness while `FR-120` is unresolved
- schedule support beyond current scheduler capability
- webhook closure without clearer delivery guarantees from the relevant baseline

Documentation dependencies:

- `docs/wave7-non-llm-automation-and-triggering-analysis.md`
- `docs/parallel_requirements.md`
- `docs/requirements.md` `FR-005`, `FR-051`, `FR-070`, `FR-234`
- `docs/wave1-governance-audit-retrieval-analysis.md`
- `docs/wave2-tooling-copilot-prompt-crm-analysis.md`
- `docs/wave3-agent-runtime-handoff-analysis.md`
- `docs/architecture.md`
- `docs/openapi.yaml`
- `features/uc-a4-workflow-execution.feature`
- `docs/agent-spec-phase6-analysis.md`
- `docs/agent-spec-use-cases.md`
- `docs/agent-spec-transition-plan.md`

Implementation anchors to inspect first:

- `internal/api/handlers/workflow.go`
- `internal/api/handlers/agent.go`
- `internal/domain/agent/orchestrator.go`
- `internal/domain/agent/dsl_runner.go`
- `internal/domain/scheduler/repository.go`
- `internal/domain/scheduler/service.go`
- `internal/infra/sqlite/migrations/027_scheduled_jobs.up.sql`

Deep review points:

- require missing requirement artifacts for `FR-005` and `FR-234`
- verify that the scheduler today only supports `workflow_resume`
- verify workflow lifecycle routes are published if the wave treats them as supported

### 8.8 Wave 8

Primary scope:

- dedupe maturity
- quota or budget maturity
- standalone release-gating maturity

What it includes:

- evidence dedupe as a measurable contract
- unified quota intent across agent, role, and tenant scopes
- eval maturity beyond basic CRUD or run support

What it does not claim:

- implementation readiness while `FR-310` is unresolved
- full standalone `FR-242` closure if gating remains prompt-only
- response-consolidation closure unless a concrete contract is frozen

Documentation dependencies:

- `docs/wave8-dedupe-budgets-and-release-gating-analysis.md`
- `docs/parallel_requirements.md`
- `docs/requirements.md` `FR-094`, `FR-233`, `FR-242`
- `docs/wave1-governance-audit-retrieval-analysis.md`
- `docs/wave2-tooling-copilot-prompt-crm-analysis.md`
- `docs/wave3-agent-runtime-handoff-analysis.md`
- `docs/architecture.md`
- `docs/implementation-plan.md`
- `docs/as-built-design-features.md`
- `docs/fr-gaps-implementation-criteria.md`
- `docs/openapi.yaml`

Implementation anchors to inspect first:

- `internal/domain/knowledge/evidence.go`
- `internal/api/handlers/eval.go`
- `internal/api/routes.go`
- `internal/domain/eval/suite.go`
- `internal/domain/eval/runner.go`
- `internal/domain/agent/prompt.go`
- `internal/domain/workflow/service.go`
- `internal/domain/agent/carta_policy_bridge.go`
- `internal/infra/sqlite/migrations/020_eval.up.sql`

Deep review points:

- require missing requirement artifacts for `FR-094`, `FR-233`, and `FR-242`
- verify `admin/eval` supported-surface publication
- reconcile `policy_version_id` versus `prompt_version_id`
- reject quota closure claims based only on scattered local checks
- require a measurable `duplication rate` contract

### 8.9 Wave 9

Primary scope:

- Carta behavior contracts

What it includes:

- parser and AST support
- judge and verification flow
- delegate and GROUNDS preflight
- activation bridges
- backward-compatibility handling

What it does not claim:

- full closure without a formal `FR-212` artifact
- full dynamic GROUNDS enforcement if validator wiring remains partial
- full lifecycle closure if workflow `verify`, `activate`, and `execute` remain undocumented in the public contract

Double dependency note:

Wave 9 depends on Wave 8 through two distinct paths:

1. `FR-212` depends on `FR-233` (quota model must be stable for Carta BUDGET bridge via `carta_policy_bridge.go`)
2. `FR-212` also depends on `FR-210`, `FR-211`, `FR-232`, and `FR-240` (runtime safety stack from Waves 1-3)

This means Wave 9 cannot start until both Wave 8 closure AND the full runtime safety stack from Waves 1-3 are stable. The `FR-233` dependency is the more fragile path because it is itself gated by `FR-310` (undefined until this audit).

Documentation dependencies:

- `docs/wave9-behavior-contracts-carta-analysis.md`
- `docs/parallel_requirements.md`
- `docs/requirements.md` `FR-212`
- `docs/carta-spec.md`
- `docs/carta-implementation-plan.md`
- `docs/architecture.md`
- `docs/wave2-tooling-copilot-prompt-crm-analysis.md`
- `docs/wave3-agent-runtime-handoff-analysis.md`
- `docs/wave8-dedupe-budgets-and-release-gating-analysis.md`
- `docs/as-built-design-features.md`
- `docs/fr-gaps-implementation-criteria.md`
- `docs/openapi.yaml`

Implementation anchors to inspect first:

- `internal/domain/agent/judge.go`
- `internal/domain/agent/judge_carta.go`
- `internal/domain/agent/dsl_runner.go`
- `internal/domain/agent/grounds_validator.go`
- `internal/domain/agent/carta_policy_bridge.go`
- `internal/domain/agent/workflow_carta_budget_sync.go`
- `internal/domain/workflow/service.go`
- `internal/api/handlers/workflow.go`
- `internal/api/routes.go`

Deep review points:

- require `FR-212` traceability
- verify whether coverage checks actually run in the supported judge path
- verify whether `GroundsValidator` is wired into the supported execution paths — as of this audit, it is NOT set in any production `RunContext` (`routes.go` resumeRC, `workflow.go` executeDSLWorkflow, `insights_shadow.go`), which means GROUNDS checks are test-only
- verify that Wave 9 depends on a stable Wave 8 quota model and does not bypass it

## 9. Recommended Audit Order

To keep review context small and avoid reloading the whole program each pass, use the following order.

### Pass A: Program Gates and Traceability

Load:

- `docs/requirements.md`
- `docs/parallel_requirements.md`
- `docs/openapi.yaml`
- `docs/architecture.md`
- `docs/as-built-design-features.md`
- `docs/fr-gaps-implementation-criteria.md`

Review:

- undefined dependency gates
- missing requirement artifacts
- supported API versus runtime drift
- architecture versus schema drift

### Pass B: Waves 1-3 Backbone Review

Load:

- Wave 1, Wave 2, and Wave 3 analysis docs
- only the implementation files needed to validate a challenged claim

Review:

- whether Waves 1-3 form a stable contract backbone
- whether later waves are depending on the right boundaries

### Pass C: Waves 6-9 High-Risk Expansion Review

Load:

- Wave 6, Wave 7, Wave 8, and Wave 9 analysis docs
- selective code surfaces:
  - `internal/api/routes.go`
  - `internal/api/handlers/workflow.go`
  - `internal/api/handlers/eval.go`
  - `internal/domain/agent/judge.go`
  - `internal/domain/agent/runner.go`
  - `internal/domain/agent/grounds_validator.go`
  - `internal/infra/sqlite/migrations/011_knowledge.up.sql`
  - `internal/infra/sqlite/migrations/020_eval.up.sql`
  - `internal/infra/sqlite/migrations/027_scheduled_jobs.up.sql`

Review:

- whether blocked waves are documented as blocked, not silently assumed
- whether their later-slice claims exceed actual support
- whether cross-wave dependencies are realistic

### Pass D: Waves 4-5 Conservative Expansion Review

Load:

- Wave 4 and Wave 5 analysis docs

Review:

- whether they remain conservative
- whether any derived contract is mislabeled as baseline fact

## 10. Suggested Detailed Audit Checklist

Use this checklist when auditing the set.

### 10.1 Requirement and Traceability Discipline

- Does every wave reference a documented requirement basis?
- Are missing `reqs/FR/*.yml` artifacts treated as real debt?
- Are blocked waves clearly blocked rather than softly assumed?

### 10.2 Scope Honesty

- Does the wave describe only what it can safely close?
- Are embedded criteria kept distinct from standalone workstreams?
- Are partial closures labeled as partial?

### 10.3 API Contract Parity

- Is every publicly claimed route present in `docs/openapi.yaml`?
- Are unpublished but implemented routes being described as internal rather than supported?
- Do downstream wave assumptions depend on unpublished routes?

### 10.4 Architecture and Schema Parity

- Do architecture enums and fields match migrations and runtime structures?
- Has the repo selected a canonical data model where drift exists?
- Are wave designs using the canonical model consistently?

### 10.5 Runtime Enforcement Parity

- Are safety and control mechanisms wired into supported execution paths?
- Is optional wiring being overstated as enforced behavior?
- Are route-level and background execution paths treated consistently?

### 10.6 Dependency Integrity

- Does each later wave consume earlier-wave contracts without redefining them?
- Are dependency gates explicit at wave boundaries?
- Is the Wave 1 versus Wave 6 split on `FR-091` preserved cleanly?

### 10.7 Release-Claim Discipline

- Does the wave say "implemented", "partially implemented", "blocked", or "planned" precisely?
- Are there any places where existence of code is being mistaken for supported closure?
- Are the later waves using MVP surfaces as if they were production-grade release gates?

## 11. Audit Boundaries and Limits

This handoff should be read with the following limits in mind.

- This review is documentation-first, with targeted implementation inspection.
- No runtime tests, integration tests, or end-to-end validation were executed as part of this handoff.
- The wave analysis documents are planning artifacts, not release certification.
- The main audit value is in checking traceability, dependency handling, scope honesty, and supported-surface alignment.

## 12. Bottom-Line Audit Guidance

If time is limited, the auditor should spend the most attention on these five questions:

1. Are `FR-120`, `FR-310`, and `FR-320` treated as unresolved gates — not as solved dependencies? Artifacts now exist (`reqs/FR/FR_120.yml`, `FR_310.yml`, `FR_320.yml`), but all three remain `active: false`. The gates are formalized, not closed.
2. Are Waves 6-9 prevented from overclaiming while their requirement artifacts remain `active: false`? All 11 missing artifacts have been created, but none are active — closure claims still require implementation.
3. Do `docs/openapi.yaml` and runtime support tell the same story for workflow lifecycle and eval surfaces? The OpenAPI has been reconciled with `routes.go` — verify the reconciliation is complete and no new routes have been added since.
4. Are the architecture-to-schema drifts on `source_type` and eval version targeting resolved? `source_type` is resolved. `policy_version_id` on `eval_run` remains a planned field not yet in any migration.
5. Does Wave 9 have real supported-path enforcement for Carta checks, especially coverage and GROUNDS, or only partial infrastructure? `GroundsValidator` is confirmed test-only — not wired in any production `RunContext`. This is the single most important open implementation gap.

If those five questions are answered rigorously, the rest of the wave set becomes much easier to trust.
