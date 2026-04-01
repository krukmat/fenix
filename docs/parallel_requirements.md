# Parallel Requirements: Remaining P0 Workstreams and Full FR Coverage

> Purpose: reorganize the still-partial or still-pending P0 work into parallelizable workstreams while also classifying the full FR universe defined in `docs/requirements.md`.
> Sources of truth: `docs/requirements.md`, `docs/as-built-design-features.md`, `docs/fr-gaps-implementation-criteria.md`, `docs/architecture.md`.
> Baseline used for this audit: the as-built matrix dated `2026-03-05` and the current FR gap backlog.

---

## FR Numbering Concordance

There is a known divergence between `docs/requirements.md` and the implemented codebase (Doorstop specs in `reqs/FR/`, BDD feature tags, `docs/as-built-design-features.md`, `docs/fr-gaps-implementation-criteria.md`).

**This plan normalizes to the codebase convention** because it is the numbering used by Doorstop, BDD features, fr-gaps, and as-built — i.e., the artifacts that sessions will actually load.

| FR ID | Codebase meaning (used here) | `requirements.md` meaning | Doorstop file |
|-------|------------------------------|---------------------------|---------------|
| FR-060 | Authentication RBAC/ABAC | RBAC/ABAC | `reqs/FR/FR_060.yml` |
| FR-061 | Approval Workflows | PII/no-cloud classification | `reqs/FR/FR_061.yml` |
| FR-070 | Audit Trail | Audit trail + Agent Run Log | `reqs/FR/FR_070.yml` |
| FR-071 | Policy Engine (RBAC/ABAC rule evaluator) | Approvals workflow | `reqs/FR/FR_071.yml` |

**PII/no-cloud classification** (`requirements.md` FR-061) does not have its own Doorstop spec, BDD feature, or fr-gaps entry. It is tracked in this plan as an explicit sub-scope of WS-01 under the label **GOV-PII** (see WS-01 brief below).

> **Session rule**: when loading `docs/fr-gaps-implementation-criteria.md` or `.feature` files, FR-061 means Approval Workflows and FR-071 means Policy Engine. Do not cross-reference against `requirements.md` numbering without consulting this concordance table.

---

## Audit Findings

- The previous version assumed 8 greenfield sub-projects. That is no longer consistent with the repo's live documentation.
- The as-built documentation already treats CRM, Knowledge, Copilot, Agent Runtime, Prompt Versioning, Mobile, and BFF as present, even if some areas are still open or only partially closed.
- `docs/fr-gaps-implementation-criteria.md` frames the remaining work as closure and hardening of in-progress FRs, not as a full build from scratch.
- As a result, the correct parallelization model is no longer "foundation -> mobile -> final AI layer". It is now "governance, audit, retrieval, tooling, copilot, and runtime" with frozen shared contracts between waves.

### Changes from the Previous Version

- `FR-003`, `FR-092`, `FR-300`, and `FR-301` are removed as primary P0 closure workstreams.
- `FR-070` and `FR-001` are brought back into the critical path because the as-built and gap docs still classify them as partial.
- `FR-050` is now tracked explicitly as a deferred prerequisite lane instead of being left implicit behind `FR-091`.
- `FR-210` remains a mandatory acceptance criterion inside Copilot and Runtime, but not a standalone lane.
- The dependency on `docs/sp/SP-XX-brief.md` is removed because that tree does not exist in the repo.
- Feature references are normalized to the actual `.feature` filenames that exist in the repo.
- FR numbering is normalized to the codebase convention (Doorstop + BDD + fr-gaps + as-built). See concordance table above.
- PII/no-cloud classification is tracked as sub-scope **GOV-PII** inside WS-01, since it has no standalone Doorstop spec or BDD feature.
- Existing migrations that partially close gaps are acknowledged: `023_audit_append_only` (WS-02), `022_prompt_experiments` (WS-06), `021_agent_run_steps` (WS-07).

---

## Current Documentary Baseline

| Category | FR | Treatment in this plan |
|----------|----|------------------------|
| Implemented or outside the primary P0 closure path | `FR-002`, `FR-003`, `FR-051`, `FR-092`, `FR-300`, `FR-301` | Not modeled as primary workstreams |
| Partial or not fully implemented | `FR-001`, `FR-060`, `FR-061`, `FR-070`, `FR-071`, `FR-090`, `FR-091`, `FR-200`, `FR-201`, `FR-202`, `FR-211`, `FR-230`, `FR-231`, `FR-232`, `FR-240` | Planned below |
| Deferred or dependency-blocked follow-on scope | `FR-004`, `FR-005`, `FR-050`, `FR-052`, `FR-093`, `FR-094`, `FR-203`, `FR-212`, `FR-233`, `FR-234`, `FR-241`, `FR-243`, `FR-302`, `FR-303` | Planned as deferred expansion lanes |
| Embedded cross-cutting criteria | `FR-210`, `FR-242`, `NFR-030`, `NFR-031`, `NFR-033` | Enforced inside consumer lanes, not as separate lanes |
| Sub-scope without standalone FR | **GOV-PII** (PII/no-cloud classification) | Tracked inside WS-01 with explicit closure criteria |

### Coverage Accounting

This document classifies the full FR universe defined in `docs/requirements.md` plus the GOV-PII sub-scope.

| Coverage Class | Count | FR |
|----------------|-------|----|
| Active primary workstreams | 15 | `FR-001`, `FR-060`, `FR-061`, `FR-070`, `FR-071`, `FR-090`, `FR-091`, `FR-200`, `FR-201`, `FR-202`, `FR-211`, `FR-230`, `FR-231`, `FR-232`, `FR-240` |
| Embedded acceptance criteria | 2 | `FR-210`, `FR-242` |
| Baseline prerequisites and consumer surfaces | 6 | `FR-002`, `FR-003`, `FR-051`, `FR-092`, `FR-300`, `FR-301` |
| Deferred or dependency-blocked expansion lanes | 14 | `FR-004`, `FR-005`, `FR-050`, `FR-052`, `FR-093`, `FR-094`, `FR-203`, `FR-212`, `FR-233`, `FR-234`, `FR-241`, `FR-243`, `FR-302`, `FR-303` |

**Total**: `37/37` FR are classified exactly once. GOV-PII is tracked as additional sub-scope.

### Pending Inventory

From a sequencing perspective, the project still has `31` FR in play:

| Pending Class | Count | FR |
|---------------|-------|----|
| Active closure backlog | 15 + GOV-PII | `FR-001`, `FR-060`, `FR-061`, `FR-070`, `FR-071`, `FR-090`, `FR-091`, `FR-200`, `FR-201`, `FR-202`, `FR-211`, `FR-230`, `FR-231`, `FR-232`, `FR-240`, **GOV-PII** |
| Embedded closure and release criteria | 2 | `FR-210`, `FR-242` |
| Deferred expansion backlog | 14 | `FR-004`, `FR-005`, `FR-050`, `FR-052`, `FR-093`, `FR-094`, `FR-203`, `FR-212`, `FR-233`, `FR-234`, `FR-241`, `FR-243`, `FR-302`, `FR-303` |

The remaining `6` FR (`FR-002`, `FR-003`, `FR-051`, `FR-092`, `FR-300`, `FR-301`) are treated as baseline prerequisites or consumer surfaces rather than active pending lanes.

---

## Workstream Overview

| WS | Name | Wave | Days | FR Covered | Depends On |
|----|------|------|------|------------|------------|
| WS-01 | Governance Core | 1 | ~3 | `FR-060`, `FR-061`, `FR-071`, **GOV-PII** | - |
| WS-02 | Audit Hardening | 1 | ~2 | `FR-070` | - |
| WS-03 | Knowledge Reliability | 1 | ~3 | `FR-090`, `FR-091` | - |
| WS-04 | Tooling Safety | 2 | ~3 | `FR-202`, `FR-211` | `WS-01`, `WS-02` |
| WS-05 | Copilot Safety | 2 | ~3 | `FR-200`, `FR-201`, `FR-210` | `WS-01`, `WS-02`, `WS-03`, baseline `FR-092` smoke ✅ |
| WS-06 | Prompt Lifecycle | 2 | ~1 | `FR-240` | `WS-02` |
| WS-07 | Agent Runtime Completion | 3 | ~4 | `FR-230`, `FR-231`, `FR-232` | `WS-01`, `WS-02`, `WS-03`, `WS-04`, `WS-05`, `WS-06` |
| WS-08 | CRM Consistency Hardening | 2 | ~2 | `FR-001` | `WS-02` (weak dependency) |

**Estimated total**: ~21 days sequential, ~10-12 days with realistic parallel execution.

---

## Dependency DAG

```text
Wave 0 (pre-gate)
  Baseline smoke: FR-092 Evidence Pack contract verification

Wave 1 (parallel)
  WS-01 Governance Core
  WS-02 Audit Hardening
  WS-03 Knowledge Reliability

Wave 2 (parallel)
  WS-04 Tooling Safety      <- WS-01 + WS-02
  WS-05 Copilot Safety      <- WS-01 + WS-02 + WS-03 + FR-092 smoke
  WS-06 Prompt Lifecycle    <- WS-02
  WS-08 CRM Consistency     <- WS-02 (weak dependency)

Wave 3
  WS-07 Agent Runtime Completion
    <- WS-01 + WS-02 + WS-03 + WS-04 + WS-05 + WS-06
```

### Dependency Rules

- **Wave 0 gate**: Before starting Wave 2, run a contract smoke test on FR-092 (Evidence Pack). WS-05 and WS-07 depend on its output shape. If the smoke fails, fix Evidence Pack before opening Wave 2.
- `WS-07` must not start until authz, audit, retrieval, tooling, copilot, and prompt contracts are frozen.
- `WS-08` may start at the end of Wave 1 if `WS-02` has already closed the audit/timeline taxonomy.
- BFF and mobile are not modeled as standalone P0 closure lanes. Their changes should be embedded in `WS-05` and `WS-07` as downstream consumer surfaces.
- Migrations, `sqlc`, and API contracts are serialized hotspots. Do not edit them in parallel without a single owner for the wave.

---

## Workstream Briefs

### WS-01: Governance Core

- **Scope**: make RBAC/ABAC enforcement, approval workflows, policy engine evaluation, and PII/no-cloud handling consistent across the system.
- **FR covered**: `FR-060` (RBAC/ABAC auth), `FR-061` (approval workflows), `FR-071` (policy engine), **GOV-PII** (PII/no-cloud classification).
- **Load**: `docs/requirements.md` section `7.8`, `docs/architecture.md` sections `5`, `6`, `7`, `docs/fr-gaps-implementation-criteria.md` sections `FR-060`, `FR-061`, `FR-071`.
- **Primary features**: `features/uc-g1-governance.feature`, `features/uc-a7-human-override-and-approval.feature`.
- **Freeze outputs**: authorization matrix, redaction/PII contract, approval-before-execution rules, and policy set/version evaluation contract.
- **Hotspots**: `internal/domain/policy/evaluator.go`, `internal/domain/policy/approval.go`, `internal/api/middleware/auth.go`, retrieval permission filters, pre-tool execution gates.

#### GOV-PII Sub-Scope (PII/no-cloud classification)

This sub-scope has **no Doorstop spec, no BDD feature, and no fr-gaps entry**. It corresponds to `requirements.md` section 7.8 "Clasificación de datos y PII/no-cloud" (FR-061 in requirements.md numbering).

**Current state**: `internal/domain/policy/evaluator.go` contains some sensitivity-related logic but it is not isolated as a testable contract.

**Closure criteria**:
- Sensitivity tags (PII/PHI/secret) assignable per field or record.
- `RedactPII()` function masking sensitive fields before LLM prompt construction.
- No-cloud policy enforcement: if active, block LLM calls to cloud providers for tagged data.
- Retention/anonymization rules per tenant/unit.
- Integration tests covering: redaction before prompt, cloud-provider blocking, sensitivity tag propagation.

**Validation strategy**: No BDD feature exists. Validate through:
1. Unit tests for `RedactPII()` and sensitivity tag assignment.
2. Integration tests proving redaction in the copilot chat path (pre-prompt).
3. Integration test proving cloud-provider blocking when no-cloud policy is active.

---

### WS-02: Audit Hardening

- **Scope**: enforce append-only behavior, stabilize event taxonomy, guarantee flow-level traceability, and lock query/export behavior.
- **FR covered**: `FR-070`.
- **Load**: `docs/requirements.md` section `7.8`, the audit block in `docs/as-built-design-features.md`, `docs/fr-gaps-implementation-criteria.md` section `FR-070`.
- **Primary features**: `features/uc-g1-governance.feature`, `features/uc-a4-workflow-execution.feature`.
- **Freeze outputs**: auditable action names, minimum event payload, and query/export rules.
- **Hotspots**: `audit_event` table, `internal/api/middleware/audit.go`, cross-cutting subscribers, `details` schema, and `trace_id` correlation.

#### Existing Infrastructure to Verify First

Migration `023_audit_append_only.up.sql` already exists. Before starting this WS, verify what it implements:
- If it enforces DELETE/UPDATE restrictions on `audit_event` → the append-only gap may already be partially closed.
- If it only creates triggers or partial constraints → document remaining work.

**Validation strategy**: BDD features cover governance scenarios. Remaining gaps (append-only enforcement, event taxonomy standardization) need:
1. Database-level test proving DELETE/UPDATE on `audit_event` is rejected.
2. Integration test verifying domain-specific audit actions (not just generic request-level actions) for critical mutations.
3. Traceability test: given a workflow execution, verify the full chain of audit events with correlated `trace_id`.

---

### WS-03: Knowledge Reliability

- **Scope**: close the remaining gap on compliant vector backend and hybrid indexing, and complete CDC coverage with freshness evidence.
- **FR covered**: `FR-090`, scoped `FR-091`.
- **Load**: `docs/requirements.md` section `7.2`, the "Knowledge hybrid + Evidence Pack" block in `docs/as-built-design-features.md`, `docs/fr-gaps-implementation-criteria.md` sections `FR-090`, `FR-091`.
- **Primary features**: `features/uc-a5-signal-detection-and-lifecycle.feature`, `features/uc-d1-data-insights-agent.feature`.
- **Freeze outputs**: retrieval contract, entity coverage matrix for auto-reindex, and freshness SLI/SLO.
- **Scope boundary**: `FR-091` closure in this lane is limited to reliability and coverage for already-unlocked source families. Expanding source families or connector breadth activates `FX-01 / FR-050`.
- **Hotspots**: `internal/domain/knowledge/search.go` (current in-memory cosine), `internal/domain/knowledge/reindex.go`, knowledge migrations `011/012/013`, vector strategy.

**Note on permission filtering**: WS-03 does not depend on WS-01 because workspace-level isolation already exists in search queries. RBAC-granular permission filtering (field-level, record-level) will be integrated when WS-05 or WS-07 wire retrieval with the policy engine output from WS-01.

**Validation strategy**: BDD features cover signal detection and data insights. Remaining gaps need:
1. Test proving vector retrieval uses sqlite-vec (or approved equivalent) instead of in-memory JSON cosine.
2. CDC entity coverage test: mutation on each CRM entity type triggers reindex event.
3. Freshness SLA test: measure reindex latency, verify <60s target with observable metric.

---

### WS-04: Tooling Safety

- **Scope**: complete the tool registry lifecycle, tighten schema validation, and unify the execution pipeline behind policy and audit gates.
- **FR covered**: `FR-202`, `FR-211`.
- **Load**: `docs/requirements.md` sections `7.3`, `7.4`, `7.8`, `docs/fr-gaps-implementation-criteria.md` sections `FR-202`, `FR-211`.
- **Primary features**: `features/uc-b1-safe-tool-routing.feature`, `features/uc-a1-agent-studio.feature`, `features/uc-a4-workflow-execution.feature`.
- **Freeze outputs**: admin lifecycle contract, execution error contract, and mandatory pre-execution gate behavior.
- **Hotspots**: `internal/domain/tool/registry.go`, `internal/domain/tool/execution_pipeline.go`, `internal/domain/tool/builtin.go`, `internal/domain/tool/builtin_executors.go`, `internal/api/handlers/tool.go`, migration `016_tools.up.sql`.

**Note on existing infrastructure**: The codebase already has `execution_pipeline.go` and `mcp_adapter.go` which the fr-gaps doc does not mention. Verify whether the execution pipeline already enforces the policy+validation+audit gate sequence before scoping new work.

**Validation strategy**: BDD features cover safe routing (allowlist + dangerous param denial) and agent studio validation. Remaining gaps need:
1. Admin lifecycle test: create → activate → deactivate → delete for tool definitions.
2. Schema validation test: reject tool execution with params that violate JSON Schema.
3. Permission gate test: deny execution when user lacks required tool permission.

---

### WS-05: Copilot Safety

- **Scope**: enforce grounded answers, mandatory abstention, confidence and eligibility in suggestions, and a stable SSE/API/BFF/mobile contract.
- **FR covered**: `FR-200`, `FR-201`, embedded `FR-210`.
- **Load**: `docs/requirements.md` sections `7.3`, `7.4`, `7.9`, the "Copilot streaming" block in `docs/as-built-design-features.md`, `docs/fr-gaps-implementation-criteria.md` sections `FR-200`, `FR-201`.
- **Primary features**: `features/uc-s1-sales-copilot.feature`, `features/uc-c1-support-agent.feature`.
- **Freeze outputs**: final Copilot response shape, abstention criteria, suggested-actions contract (with confidence per action), and `FR-210` enforcement in the copilot path.
- **Hotspots**: `internal/domain/copilot/chat.go`, `internal/domain/copilot/suggest_actions.go`, `internal/api/handlers/copilot_chat.go`, `internal/api/handlers/copilot_actions.go`, `bff/src/routes/copilot.ts`, `mobile/src/hooks/useSSE.ts`.

**FR-210 (mandatory abstention) closure in this lane**: The copilot must enforce deterministic abstention when evidence confidence is below threshold. This means:
- `chat.go` checks `EvidencePack.Confidence` before generating a response.
- If insufficient → return abstention chunk with reason + escalation option.
- Tested via the abstention scenario in `uc-c1-support-agent.feature`.

**Pre-condition**: FR-092 baseline smoke test must pass before starting this WS (see Wave 0 gate).

**Validation strategy**: BDD features cover copilot launch from CRM records and abstention. Remaining gaps need:
1. Abstention determinism test: given evidence below threshold, verify abstention response is always produced.
2. Confidence-per-action test: `SuggestActions()` output includes confidence score per suggestion.
3. SSE contract test: verify chunk format `{type: "token"|"evidence"|"action"|"done"}` across Go → BFF → mobile.

---

### WS-06: Prompt Lifecycle

- **Scope**: clean up create/promote/rollback semantics, make identity usage consistent, and define the handoff point into eval gating.
- **FR covered**: `FR-240`.
- **Load**: `docs/requirements.md` section `7.6`, `docs/fr-gaps-implementation-criteria.md` section `FR-240`, the prompts section in `docs/architecture.md`.
- **Primary features**: `features/uc-a2-workflow-authoring.feature`, `features/uc-a3-workflow-verification-and-activation.feature`, `features/uc-a8-workflow-versioning-and-rollback.feature`.
- **Freeze outputs**: `draft -> testing -> active -> archived` state model, rollback semantics, and integration point with `FR-242`.
- **Hotspots**: `internal/domain/agent/prompt.go`, `internal/domain/agent/prompt_experiment.go`, `internal/api/handlers/prompt.go`, `internal/api/handlers/prompt_experiment.go`, migrations `017_prompt_versions.up.sql`, `022_prompt_experiments.up.sql`.

#### Existing Infrastructure to Verify First

Migration `022_prompt_experiments.up.sql` and `prompt_experiment.go` already exist. Before scoping A/B testing work:
- Verify what experiment support is already implemented.
- If A/B routing exists → the fr-gaps "A/B testing support" may be partially closed.
- If only the schema exists → document the remaining runtime routing logic.

**Validation strategy**: BDD features cover workflow authoring, verification, and versioning. Remaining gaps need:
1. API contract test: rollback uses consistent identity (version ID, not agent ID ambiguity).
2. Eval gate test: promotion blocked when eval suite has not passed thresholds.
3. A/B experiment test: two active prompt versions can be routed by experiment configuration.

---

### WS-07: Agent Runtime Completion

- **Scope**: finish the runtime state machine, close the real UC-C1 path, stabilize human handoff, and complete the minimum agent catalog with step-level traceability.
- **FR covered**: `FR-230`, `FR-231`, `FR-232`.
- **Load**: `docs/requirements.md` section `7.5`, the "Agent Runtime" block in `docs/as-built-design-features.md`, `docs/fr-gaps-implementation-criteria.md` sections `FR-230`, `FR-231`, `FR-232`.
- **Primary features**: `features/uc-c1-support-agent.feature`, `features/uc-a4-workflow-execution.feature`, `features/uc-a6-deferred-actions.feature`, `features/uc-a9-agent-delegation.feature`, `features/uc-k1-kb-agent.feature`, `features/uc-s2-prospecting-agent.feature`, `features/uc-s3-deal-risk-agent.feature`, `features/uc-d1-data-insights-agent.feature`.
- **Freeze outputs**: `agent_run` contract, step trace contract, handoff package contract, and final UC-C1 flow.
- **Hotspots**: `internal/domain/agent/orchestrator.go`, `internal/domain/agent/handoff.go`, `internal/domain/agent/agents/support.go`, `internal/domain/agent/runner.go`, `internal/domain/agent/runner_registry.go`, `internal/api/handlers/agent.go`, `internal/api/handlers/handoff.go`.

#### Existing Infrastructure Warning

The agent domain contains extensive infrastructure beyond what fr-gaps describes:
- **DSL runtime**: `dsl_runtime.go`, `dsl_runtime_executor.go`, lexer/parser/AST, expression evaluator — a complete workflow execution engine.
- **CARTA parser**: `carta_lexer.go`, `carta_parser.go`, `carta_ast.go`, `carta_policy_bridge.go` — declarative agent behavior specs.
- **Judge system**: `judge.go`, `judge_carta.go`, `grounds_validator.go`, `delegate_evaluator.go` — validation and grounding checks.
- **Protocol handlers**: `protocol_handler.go`, `protocol_handler_a2a.go` — agent-to-agent communication.
- **Migration 021**: `agent_run_steps` table already exists for step-level tracing.

**Session rule for WS-07**: Before scoping work, read `internal/domain/agent/orchestrator.go` and `internal/domain/agent/runner_registry.go` to understand the existing state machine. Do not assume the runtime is a stub — it has significant infrastructure. Focus on gap closure (real data-driven execution, deterministic handoff, production-complete UC-C1) rather than rebuilding.

**FR-210 (mandatory abstention) closure in this lane**: The agent runtime must enforce abstention when confidence is low. The `grounds_validator.go` already exists — verify whether it satisfies the deterministic abstention requirement before writing new logic.

**Validation strategy**: BDD features provide extensive coverage (13 scenarios across 8 feature files). Remaining gaps need:
1. UC-C1 E2E test: case created → agent triggered → real evidence retrieved → LLM reasoning → tools executed → case resolved OR abstained OR escalated.
2. Step trace test: given an agent run, verify per-step records in `agent_run_steps` with tokens/cost/latency.
3. Handoff payload test: verify complete package (conversation context + reasoning trace + evidence pack + CRM context).

---

### WS-08: CRM Consistency Hardening

- **Scope**: make CRUD behavior uniform, tighten business validation, and standardize audit/timeline side effects for core mutations.
- **FR covered**: `FR-001`.
- **Load**: `docs/requirements.md` section `7.1`, `docs/fr-gaps-implementation-criteria.md` section `FR-001`, `docs/openapi.yaml`.
- **Primary driver**: there is no dedicated `.feature` file for this gap; validate through API contract and mutation semantics.
- **Freeze outputs**: uniform CRUD checklist, per-entity validation rules, and shared audit/timeline side-effect contract.
- **Hotspots**: `internal/api/handlers/` (deal.go, case.go, lead.go, account.go, contact.go, activity.go, note.go, attachment.go), `internal/domain/crm/validation.go`, `internal/domain/crm/timeline_auto.go`, `internal/domain/crm/crm_mutation_side_effects_test.go`.

**Validation strategy**: No BDD feature. Validate through:
1. Per-entity CRUD contract test: every entity supports list/create/update/delete with uniform response shape and error codes.
2. Business validation test: Deal stage transitions, Case status transitions, required fields enforcement.
3. Side-effect test: every create/update/delete emits both an audit event and a timeline event (use existing `crm_mutation_side_effects_test.go` as baseline, extend to all entities).

---

## Baseline Prerequisites and Consumer Surfaces

These FR are included in the plan, but not as standalone active lanes. They either act as already-established baseline capabilities or as consumer surfaces that must remain compatible while active lanes close.

| FR | Classification | Why it is not an active lane | Still required by |
|----|----------------|------------------------------|-------------------|
| `FR-002` | Baseline prerequisite | Pipeline foundations exist; current work uses it as a stable CRM surface instead of reopening it as a primary lane | `WS-08`, `WS-07` |
| `FR-003` | Baseline consumer surface | Reporting base is not part of the active P0 closure path in this document | Follow-on reporting expansion only |
| `FR-051` | Baseline prerequisite | Public API/webhook baseline exists and is consumed by later extensibility work | `FX-02`, `FX-05` |
| `FR-092` | Mandatory baseline gate | Evidence Pack is already treated as available baseline, but remains a required gate for Copilot and Runtime closure. **Smoke test required before Wave 2** | `WS-05`, `WS-07` |
| `FR-300` | Consumer surface | Mobile is a downstream surface, not a standalone closure lane | `WS-05`, `WS-07`, `FX-06` |
| `FR-301` | Consumer surface | BFF gateway is a downstream surface, not a standalone closure lane | `WS-05`, `WS-07`, `FX-06` |

### FR-092 Baseline Smoke Test (Wave 0 Gate)

Before Wave 2 starts, verify that the Evidence Pack contract works end-to-end:

1. `BuildEvidencePack()` returns sources with `id`, `snippet`, `score`, `timestamp`.
2. Confidence classification (`high`/`medium`/`low`) is deterministic given fixed inputs.
3. Deduplication reduces near-duplicate evidence.
4. The output shape matches what `WS-05` (Copilot) and `WS-07` (Runtime) will consume.

If any of these fail, fix Evidence Pack before opening Wave 2. This is a blocking gate.

---

## Validation Strategy by FR (Non-BDD Gaps)

Several active FR have gaps that are not directly covered by existing BDD scenarios. This table defines how each will be validated.

| FR | WS | Gap Description | Validation Approach |
|----|----|-----------------|---------------------|
| `FR-001` | WS-08 | Uniform CRUD, business validation, side effects | API contract tests + mutation side-effect integration tests |
| `FR-070` | WS-02 | Append-only enforcement, event taxonomy | DB-level immutability test + domain-action audit integration test + trace correlation test. Check migration `023` first |
| `FR-091` | WS-03 | CDC entity coverage, <60s freshness SLA | Entity coverage matrix test + reindex latency metric test |
| `FR-240` | WS-06 | A/B experiment routing, eval-gated promotion | Experiment routing integration test + promotion-blocked-by-eval test. Check migration `022` + `prompt_experiment.go` first |
| **GOV-PII** | WS-01 | Sensitivity tags, RedactPII, no-cloud blocking | Unit tests for redaction + integration test for pre-prompt masking + cloud-provider blocking test |
| `FR-060` | WS-01 | Consistent ABAC enforcement across all decision points | Integration tests for role+attribute denial across API, tools, and retrieval paths |
| `FR-061` | WS-01 | End-to-end approval-before-execution linkage | Integration test: sensitive action → auto-creates approval → execution blocked → approve → execution proceeds |
| `FR-071` | WS-01 | Workspace-scoped policy resolution consistency | Regression tests for conflicting policies + precedence resolution with logged rule trace |

---

## Full Wave Roadmap

This roadmap defines the full wave sequence for the entire FR universe while preserving small, dependency-local context packs for LLM sessions.

### Wave 0: Baseline Verification

Pre-gate before Wave 2:

- FR-092 Evidence Pack smoke test (see above)
- Verify existing migrations: `023_audit_append_only`, `022_prompt_experiments`, `021_agent_run_steps`

### Wave 1: Governance, Audit, Retrieval Foundations

Active lanes:

- `WS-01` -> `FR-060`, `FR-061`, `FR-071`, **GOV-PII**
- `WS-02` -> `FR-070`
- `WS-03` -> `FR-090`, scoped `FR-091`

Why this wave exists:

- It freezes authorization, audit, and retrieval contracts before any downstream runtime or tooling work starts.
- It is the minimum safe context pack for P0 closure.

### Wave 2: Tooling, Copilot, Prompting, CRM Hardening

Active lanes:

- `WS-04` -> `FR-202`, `FR-211`
- `WS-05` -> `FR-200`, `FR-201`, embedded `FR-210`
- `WS-06` -> `FR-240`
- `WS-08` -> `FR-001`

Why this wave exists:

- It closes the user-facing and execution-facing contracts that Agent Runtime consumes.
- It keeps tool, copilot, prompt, and CRM hardening work independent enough for parallel LLM sessions.

### Wave 3: Runtime Closure

Active lane:

- `WS-07` -> `FR-230`, `FR-231`, `FR-232`

Wave gate:

- Runtime-facing closure of embedded `FR-210`

Why this wave exists:

- Runtime is the main integration point and should only start after upstream contracts are frozen.

### Wave 4: Unblocked Expansion After P0 Closure

These items can start once Waves 1-3 are closed, without waiting on external prerequisites:

- `FR-004` after `WS-01` + `WS-03`
- `FR-093` after `WS-03`
- `FR-203` after `WS-06`
- `FR-241` after `WS-07` + baseline `FR-051`
- `FR-243` after `WS-02` + `WS-03`
- `FR-302` after baseline `FR-301` + `WS-05` + `WS-07` + `FR-061` + `FR-232`
- `FR-303` after baseline `FR-300` and stable `WS-05` + `WS-07`

Why this wave exists:

- All items are internally unblocked and can run in parallel with narrow context packs.
- This is the first post-P0 expansion wave with no external prerequisite gate.

### Wave 5: Packaging and Distribution

Starts after Wave 4:

- `FR-052` after `FR-241` + `FR-240`

Why this wave exists:

- Plugin packaging should not start before the reusable authoring and tool-builder surfaces are stable.

### Wave 6: Connector Expansion Gate

Starts only when `FR-320` is clarified or implemented:

- `FR-050`
- Full-source breadth expansion of `FR-091` beyond the already-unlocked source families handled by `WS-03`

Why this wave exists:

- Connector breadth is gated by a dependency that is outside the active closure set.
- Keeping it separate avoids contaminating retrieval sessions with integration-specific context too early.

### Wave 7: Non-LLM Automation and Triggering

Starts only when `FR-120` is available:

- `FR-005`
- `FR-234`

Why this wave exists:

- Both items are blocked by the same external workflow/trigger foundation and share a compact automation context pack.

### Wave 8: Dedupe, Budgets, and Full Release Gating

Starts only when `FR-310` is available:

- `FR-094`
- `FR-233`
- Full standalone maturity of `FR-242`

Why this wave exists:

- These items depend on cost/quality/release-control infrastructure that is not yet closed.
- Grouping them keeps governance-quality context local to one wave.

### Wave 9: Behavior Contracts / Carta

Starts only after Wave 8:

- `FR-212` after `FR-210`, `FR-211`, `FR-232`, `FR-233`, `FR-240`

Why this wave exists:

- Carta depends on runtime safety, budgets, escalation, and prompt lifecycle all being stable first.
- It is the highest-order declarative layer and should remain last for context isolation and contract safety.

---

## Deferred Expansion Workstreams

These lanes keep the full FR universe covered without inflating the active P0 closure set. They are deliberately grouped to minimize LLM context load per session.

| FX | Name | FR Covered | Can Start When | Context Pack |
|----|------|------------|----------------|--------------|
| FX-01 | Connector Enablement | `FR-050` | Only after `FR-320` is clarified or implemented | `docs/requirements.md` section `7.7`, integration architecture, connector-specific docs only |
| FX-02 | CRM Extensibility and Rule Automation | `FR-004`, `FR-005` | `FR-004` after `WS-01` + `WS-03`; `FR-005` only after `FR-120` is available | CRM schema, retrieval mapping, API contract, workflow rules only |
| FX-03 | Retrieval Freshness, Dedupe, and Replay | `FR-093`, `FR-094`, `FR-243` | `FR-093` after `WS-03`; `FR-243` after `WS-02` + `WS-03`; `FR-094` only after `FR-310` is available | Knowledge, audit, and runtime replay surfaces only |
| FX-04 | Runtime Governance Expansion | `FR-212`, `FR-233`, `FR-234` | `FR-233` only after `FR-310`; `FR-234` only after `FR-120`; `FR-212` only after `FR-233` + `WS-04` + `WS-06` + `WS-07` | Runtime, policy, and Carta docs only |
| FX-05 | Authoring, Skills, and Packaging | `FR-203`, `FR-241`, `FR-052` | `FR-203` after `WS-06`; `FR-241` after `WS-07` + `FR-051`; `FR-052` after `FR-241` + `WS-06` | Prompt/versioning, tool builder, and plugin packaging docs only |
| FX-06 | Mobile Expansion | `FR-302`, `FR-303` | After `FR-301` baseline and `WS-05` + `WS-07` are stable; `FR-302` also requires `FR-061` + `FR-232` | Mobile/BFF docs and mobile-facing contracts only |

### Expansion Wave Rules

- Do not activate an `FX-*` lane unless its dependency gate is already closed in documentary form.
- Prefer Wave 4 work before any externally gated wave because it delivers the highest backlog reduction with the lowest context-switch cost.
- Treat Waves 6-9 as conditional waves: they are defined now, but they are not executable until their external or upstream dependency gates are closed.
- Keep each `FX-*` lane isolated to its own context pack; do not load unrelated active P0 lanes unless a contract changed.
- If an `FX-*` lane reopens a shared contract, it must publish a handoff note before another lane starts.

---

## Shared Hotspots to Serialize

- `docs/openapi.yaml` and any generated types consumed by mobile or BFF must have a single owner per wave.
- SQLite migrations (currently up to `027`) and any change that requires regenerating `sqlc` must not be edited in parallel.
- Shared contracts such as `agent_run`, `tool_call`, `evidence_pack`, and SSE chunks must be frozen before downstream consumers start implementation.
- Use the exact `.feature` filenames that exist in `features/`. Do not use shortened aliases from the previous plan.
- If a workstream changes a cross-cutting contract, publish a short handoff note before the next wave starts.

---

## Session Protocol

### 1. Minimal Context Load

```text
Load: docs/parallel_requirements.md (this file — includes concordance table)
     + target sections from requirements/as-built/gaps/architecture
     + the workstream's feature files
Do not load by default: full implementation-plan, full architecture, or the entire repo
```

### 2. Source of Truth Order

- FR numbering: **this document's concordance table** (codebase convention)
- FR definitions and dependencies: `docs/requirements.md` (apply concordance when reading)
- Current implementation state: `docs/as-built-design-features.md`
- Closure criteria: `docs/fr-gaps-implementation-criteria.md`
- Canonical module decomposition: `docs/architecture.md`

### 3. Validation Rules

- Use BDD when a mapped feature exists for the gap.
- Use contract, API, unit, or integration testing when the gap is primarily hardening and does not have a dedicated feature (see "Validation Strategy by FR" table above).
- Do not assume that all remaining coverage must live in `tests/bdd/go`.

### 4. Handoff Rules

- Wave 1 must freeze authz, audit, and retrieval before opening tooling, copilot, or runtime work.
- Wave 2 must freeze tools, prompts, and copilot contracts before opening runtime and handoff completion.

### 5. Ownership Rule

- Mobile and BFF remain consumer surfaces. Do not open a separate workstream for them unless the documented scope changes.

### 6. Existing Infrastructure Rule

- Before starting any WS, read the existing code for the hotspot files listed in the brief. Do not assume stubs — the codebase has significant infrastructure (DSL runtime, CARTA parser, judge system, execution pipeline, MCP adapter, prompt experiments, etc.).
- Verify what existing migrations already close before scoping new work: `021_agent_run_steps`, `022_prompt_experiments`, `023_audit_append_only`, `024_workflows`, `025_workflow_active_uniqueness`, `026_signals`, `027_scheduled_jobs`.

---

## FR Coverage Matrix

| FR | Primary Workstream |
|----|--------------------|
| `FR-001` | `WS-08` |
| `FR-060` | `WS-01` |
| `FR-061` | `WS-01` |
| `FR-070` | `WS-02` |
| `FR-071` | `WS-01` |
| `FR-090` | `WS-03` |
| `FR-091` | `WS-03` |
| `FR-200` | `WS-05` |
| `FR-201` | `WS-05` |
| `FR-202` | `WS-04` |
| `FR-211` | `WS-04` |
| `FR-230` | `WS-07` |
| `FR-231` | `WS-07` |
| `FR-232` | `WS-07` |
| `FR-240` | `WS-06` |
| **GOV-PII** | `WS-01` |

**Closure coverage**: `15/15` FRs documented as partial or not fully implemented are assigned to exactly one primary lane, plus GOV-PII sub-scope.

## Full FR Classification Summary

This summary is the compact, dependency-aware accounting model for all FR in the repo.

| Class | FR |
|-------|----|
| Active primary lanes | `FR-001`, `FR-060`, `FR-061`, `FR-070`, `FR-071`, `FR-090`, `FR-091`, `FR-200`, `FR-201`, `FR-202`, `FR-211`, `FR-230`, `FR-231`, `FR-232`, `FR-240`, **GOV-PII** |
| Embedded criteria | `FR-210`, `FR-242` |
| Baseline prerequisites / consumer surfaces | `FR-002`, `FR-003`, `FR-051`, `FR-092`, `FR-300`, `FR-301` |
| Deferred / blocked expansion lanes | `FR-004`, `FR-005`, `FR-050`, `FR-052`, `FR-093`, `FR-094`, `FR-203`, `FR-212`, `FR-233`, `FR-234`, `FR-241`, `FR-243`, `FR-302`, `FR-303` |

**Exhaustive coverage**: `37/37` FR are now represented in this plan without forcing all of them into the active P0 closure wave set. GOV-PII is tracked as additional sub-scope with explicit closure criteria.
