# FenixCRM Strategic Repositioning Implementation Plan

> **Status**: Active
> **Date**: 2026-04-06
> **Time horizon**: 8 weeks
> **Execution model**: 4 waves, 2 weeks each
> **Primary references**: `docs/plans/fenixcrm_strategic_repositioning_spec.md`, `docs/architecture.md`, `docs/requirements.md`
> **Precedence rule**: this document is the canonical implementation plan for the current strategic direction. If `docs/implementation-plan.md` conflicts with this document, the repositioning spec, or `docs/architecture.md`, this document takes precedence.
> **Mobile execution note**: mobile, BFF, and mobile-facing API harmonization for the wedge is specified in `docs/plans/mobile_wedge_harmonization_plan.md`. If older mobile plans conflict with that document, the mobile harmonization plan takes precedence for `mobile/`, `bff/`, and the supporting mobile-facing API layer.

---

## 1. Purpose

This plan translates the strategic repositioning decision into an executable implementation sequence.

The goal is to reacondition FenixCRM from an overly broad product trajectory into a focused, defensible wedge:

1. **Support Copilot and Support Agent** as the primary commercial wedge
2. **Sales Copilot** as the secondary wedge
3. **Governed AI execution** with evidence, approvals, auditability, policy enforcement, and usage attribution as the product moat

This is **not** a rewrite plan.
It is a prioritization, contract-hardening, and delivery-alignment plan over the codebase that already exists.

---

## 2. Current State Assessment

The repository already contains meaningful implementation across the target wedge:

- support and sales domain entities
- knowledge ingestion, hybrid retrieval, and evidence assembly
- copilot and agent runtime surfaces
- approvals, policy, and audit foundations
- optional BFF and mobile surfaces

Additionally, the following capabilities have already been implemented during prior waves and are now part of the baseline:

- **Usage/quota domain**: persistence (`029_usage_and_quota_domain.up.sql`), service (`internal/domain/usage/service.go`), runtime emission, and read APIs (`GET /api/v1/usage`) for workspace and run visibility
- **Approval normalization**: migration `028_approval_status_normalization.up.sql` normalizes approval states to the target finite-state model (`pending`, `approved`, `rejected`, `expired`, `cancelled`)
- **Connector boundary**: migration `030_knowledge_connector_boundary.up.sql` persists `source_system`, `source_type`, `source_object_id`, `refresh_strategy`, `delete_behavior`, and `permission_context` on `knowledge_item`; `POST /api/v1/knowledge/ingest` accepts and returns these provenance fields
- **Evidence pack contract**: locked across evidence, copilot, support handoff, and the canonical sales brief flow (`POST /api/v1/copilot/sales-brief`)
- **Agent outcome normalization**: support wedge runs end-to-end with evidence, approval, audit, handoff, and usage traces
- **Packaging and demo**: `README.md`, `docs/architecture.md`, dashboard-facing status notes, and `docs/wedge-demo-uat-summary.md` align to the wedge package model

However, the planning layer still shows strategic drift in one material area:

1. the historical execution plan still overweights `mobile`, `BFF`, and broad UI surface completion — this is now addressed by the dedicated `docs/plans/mobile_wedge_harmonization_plan.md`, which constrains the mobile surface to the five wedge-aligned tabs and removes non-wedge breadth

This plan closes remaining drift and formalizes the contracts that underpin the implemented capabilities.

---

## 3. Delivery Strategy

### 3.1 Guiding rules

- Do not expand generic CRM breadth unless the change is required by the support or sales wedge.
- Do not treat mobile parity as a release gate for the wedge.
- Do not schedule marketplace or broad Agent Studio work in this iteration.
- Preserve current runtime investments: retrieval, evidence, policy, approvals, audit, prompt versioning, safe tools, and tenant isolation.
- Prefer compatibility layers over route churn where public behavior can be normalized without breaking existing clients.

### 3.2 Exit condition for this plan

This plan is complete only when all of the following are true:

- one support workflow is demonstrable end-to-end with evidence, approval, audit, and usage attribution
- one sales copilot workflow is demonstrable end-to-end with grounded context and abstention behavior
- strategic APIs expose stable contracts for evidence, approval, audit, and usage
- the repository documentation no longer presents mobile breadth or broad CRM replacement as the governing delivery path

### 3.3 Dependency model

The waves are accepted sequentially, but several tasks inside them can run in parallel once their hard prerequisites are closed.

Execution rules:

- do not start wedge hardening on support or sales until the public contracts for evidence, approval, and outcomes are locked
- do not finalize packaging or demo scripts before the support wedge and the sales wedge are technically stable
- connector boundary work is required before new connector expansion, but it is not on the critical path for the first wedge demo
- usage schema can begin once Wave 1 is closed, but usage emission and usage-facing APIs must wait for outcome normalization

### 3.4 Ordered task graph

| ID | Task | Hard dependencies | Unblocks | Status |
|---|---|---|---|--- |
| `W1-T1` | Canonical plan + precedence notes | none | all remaining work | **done** |
| `W1-T2` | Summary/dashboard alignment | `W1-T1` | unambiguous repo navigation | **done** |
| `W2-T1` | Evidence Pack v1 contract | `W1-T1` | `W2-T3`, `W3-T3`, `W4-T1` | **done** |
| `W2-T2` | Approval state normalization | `W1-T1` | `W2-T3`, `W3-T3` | **done** (migration 028) |
| `W2-T3` | Agent outcome normalization | `W2-T1`, `W2-T2` | `W3-T2`, `W3-T3`, `W4-T1` | **done** |
| `W2-T4` | Connector boundary contract | `W1-T1` | later connector work; not wedge-demo critical | **done** (migration 030) |
| `W3-T1` | Usage domain schema + service | `W1-T1` | `W3-T2`, `W3-T4` | **done** (migration 029) |
| `W3-T2` | Metering emission points | `W2-T3`, `W3-T1` | `W3-T3`, `W3-T4`, `W4-T1` | **done** |
| `W3-T3` | Support wedge hardening | `W2-T1`, `W2-T2`, `W2-T3`, `W3-T2` | `W4-T2`, `W4-T3` | **done** |
| `W3-T4` | Usage/quota read APIs | `W3-T1`, `W3-T2` | `W4-T3` | **done** |
| `W4-T1` | Sales Copilot reference flow | `W2-T1`, `W2-T3`, `W3-T2` | `W4-T2`, `W4-T3` | **done** |
| `W4-T2` | Packaging and messaging alignment | `W3-T3`, `W4-T1` | `W4-T3` | **done** |
| `W4-T3` | Demo/UAT bundle | `W3-T3`, `W3-T4`, `W4-T1`, `W4-T2` | final acceptance | **done** |

### 3.5 Critical path

The critical path for the first ordered release is:

`W1-T1` -> `W2-T1` + `W2-T2` -> `W2-T3` -> `W3-T2` -> `W3-T3` + `W4-T1` -> `W4-T2` -> `W4-T3`

Interpretation:

- `W2-T1` and `W2-T2` should run in parallel, but `W2-T3` must wait for both
- `W3-T1` (usage domain schema) depends only on `W1-T1` and may start in parallel with `W2-T1`/`W2-T2`; it joins the critical path through `W3-T2` which depends on both `W2-T3` and `W3-T1`
- `W3-T3` (support hardening) and `W4-T1` (sales copilot) are both at depth 4 after `W3-T2` and may run in parallel; `W4-T2` must wait for both
- `W2-T4` is intentionally off the critical path
- `W3-T4` is not required to start `W4-T1`, but it is required before final demo closure because usage must be visible

### 3.6 Mapping to existing implementation areas

This sequencing is grounded in the current repository shape:

- `W2-T1` — implemented in `internal/domain/knowledge/*`, `internal/api/handlers/knowledge_evidence.go`, and copilot evidence usage
- `W2-T2` — implemented via migration `028_approval_status_normalization.up.sql`, normalizing `internal/domain/policy/approval.go` and `internal/api/handlers/approval.go`
- `W2-T3` — implemented in `agent_run`, `internal/domain/agent/orchestrator.go`, `internal/domain/agent/handoff.go`, and `/api/v1/agents/*`
- `W2-T4` — implemented via migration `030_knowledge_connector_boundary.up.sql` with provenance fields on `knowledge_item`
- `W3-T1` to `W3-T4` — implemented in `internal/domain/usage/service.go`, migration `029_usage_and_quota_domain.up.sql`, `internal/api/handlers/usage.go`, and route wiring
- `W3-T3` — hardened via `internal/domain/agent/agents/support.go`, tool execution, policy, and audit paths
- `W4-T1` reuses existing copilot infrastructure and must not introduce new CRM breadth to achieve the sales wedge

---

## 4. Wave 1 — Strategic Reset and Plan Alignment

**Duration**: Weeks 1-2

### Objective

Make the repository internally coherent so implementation work follows the repositioned strategy instead of the superseded MVP framing.

### Scope

- establish this document as the governing implementation plan
- downgrade the old `docs/implementation-plan.md` to historical/reference status
- update project-facing planning summaries and dashboards
- create execution tasks from this plan only after the wave is accepted

### Required outputs

- canonical plan in `docs/plans/`
- precedence note in `docs/implementation-plan.md`
- updated strategic summary and dashboard links
- explicit defer list for:
  - mobile breadth parity
  - marketplace
  - broad Agent Studio breadth
  - CRM expansion not directly tied to support or sales workflows

### Acceptance criteria

- no planning document still implies mobile parity is required for wedge completion
- no planning document still frames FenixCRM as a broad CRM replacement
- the implementation starting point for engineers is unambiguous

> **Wave 1 status: COMPLETED.** This document is the canonical plan. Precedence notes added to `docs/implementation-plan.md`. Summaries and dashboards updated. Mobile drift addressed by `docs/plans/mobile_wedge_harmonization_plan.md`.

---

## 5. Wave 2 — Contract Hardening and Governance Normalization

**Duration**: Weeks 3-4

### Objective

Lock the contracts that define the wedge so runtime behavior becomes stable, explainable, and testable.

### Scope

#### A. Evidence contract

Formalize **Evidence Pack v1** as the public contract for retrieval-backed workflows.

Minimum required fields:

- `schema_version`
- `query`
- `sources`
- `source_count`
- `dedup_count`
- `filtered_count`
- `confidence`
- `warnings`
- `retrieval_methods_used`
- `built_at`

This contract must be used consistently by:

- `POST /api/v1/knowledge/evidence`
- `POST /api/v1/copilot/chat`
- support handoff payloads
- support agent evidence attachment paths

#### B. Approval contract

Normalize `approval_request` to the target finite-state model:

- `pending`
- `approved`
- `rejected`
- `expired`
- `cancelled`

Implementation rule:

- existing persisted `denied` values remain readable for compatibility
- external behavior and documentation standardize on `rejected`
- every transition is deterministic and audited

#### C. Agent outcome contract

Normalize public `agent_run` outcomes to:

- `completed`
- `completed_with_warnings`
- `abstained`
- `awaiting_approval`
- `handed_off`
- `denied_by_policy`
- `failed`

Implementation rule:

- internal transitional statuses may remain if needed
- only the normalized public outcomes are exposed in wedge-facing APIs and docs

#### D. Connector boundary

Require ingestion and connector flows to preserve these fields:

- `source_system`
- `source_type`
- `source_object_id`
- `refresh_strategy`
- `delete_behavior`
- `permission_context`

Current status:

- `knowledge_item` now persists this minimum connector boundary
- `POST /api/v1/knowledge/ingest` accepts and returns the same provenance fields
- future connector work can extend source families without redefining the ingest contract

### Acceptance criteria

- evidence, approval, and outcome contracts are documented and wired to current strategic APIs
- approval states are no longer ambiguous at the API/documentation layer
- handoff payloads preserve rationale plus evidence context under a stable contract
- connector provenance fields are persisted and documented before connector breadth expands

> **Wave 2 status: COMPLETED.** Evidence Pack v1 contract is locked. Approval normalization shipped (migration 028). Agent outcomes normalized. Connector boundary persisted (migration 030). Remaining work is hardening and documentation-only.

---

## 6. Wave 3 — Support Wedge Hardening and Usage Foundation

**Duration**: Weeks 5-6

### Objective

Make the support wedge commercially credible by ensuring traceable execution, deterministic approvals, and per-run attribution.

### Scope

#### A. Usage domain introduction

Implement the missing runtime domain:

- `usage_event`
- `quota_policy`
- `quota_state`

Minimum `usage_event` fields:

- `id`
- `workspace_id`
- `actor_id`
- `actor_type`
- `run_id`
- `tool_name`
- `model_name`
- `input_units`
- `output_units`
- `estimated_cost`
- `latency_ms`
- `created_at`

#### B. Metering emission points

Emit usage attribution at minimum from:

- grounded copilot execution
- support agent runs
- governed tool calls

Attribution rule:

- every event must be attributable to workspace, actor, and run when a run exists
- model usage is recorded when provider metadata is available
- lack of provider cost metadata must not prevent event emission

#### C. Support reference flow

Canonical support flow to enforce:

1. trigger
2. context resolution
3. retrieval
4. evidence build
5. policy evaluation
6. draft generation
7. approval gate if required
8. tool execution
9. audit logging
10. handoff fallback

No support path may skip audit or policy evaluation.

#### D. Strategic API additions

Expose reserved runtime reporting endpoints:

- `GET /api/v1/usage`
- `GET /api/v1/quota-state`

Quota enforcement remains staged:

- this wave implements data model and reporting foundation
- hard enforcement remains a later increment unless a low-risk soft limit can be added without destabilizing the wedge

### Acceptance criteria

- one support run can be inspected end-to-end with evidence, approval state, audit trace, and usage trace
- usage can be reported per workspace and per run
- the support wedge no longer depends on mobile delivery to be considered complete

> **Wave 3 status: COMPLETED.** Usage domain shipped (migration 029, service, handler, read APIs). Metering emission wired to copilot, support agent, and tool calls. Support wedge runs end-to-end with all traces.

---

## 7. Wave 4 — Sales Wedge and Commercial Packaging Alignment

**Duration**: Weeks 7-8

### Objective

Close the second wedge and align packaging, demo, and repository messaging with the implemented product direction.

### Scope

#### A. Sales Copilot reference flow

Deliver one canonical grounded flow for `account/deal` context that returns:

- grounded summary
- risks or objections
- next best actions
- abstention when evidence quality is insufficient

This flow must reuse the same runtime contracts as support:

- Evidence Pack v1
- normalized public outcomes
- audit visibility
- usage attribution

#### B. Commercial packaging alignment

Align the repository narrative to these packages:

1. `Support Copilot`
2. `Support Agent`
3. `Sales Copilot`

Update product-facing docs so the primary offer is the governed AI layer, not generic CRM breadth.

Current status:

- `README.md`, `docs/architecture.md`, and dashboard-facing status notes align to the package model above
- support remains the primary wedge, sales remains the secondary wedge
- mobile breadth is still documented as a delivery surface, not a package or release gate

#### C. Demo/UAT packaging

Prepare a stable demonstration path:

- one support scenario with approval
- one support scenario with abstention or handoff
- one sales copilot scenario with grounded summary
- visible audit and usage evidence for each scenario

Current status:

- the canonical bundle now lives in `docs/wedge-demo-uat-summary.md`
- the bundle uses the shipped support, handoff, sales brief, audit, and usage APIs
- wedge acceptance no longer depends on reconstructing a demo path from scattered notes

### Acceptance criteria

- support and sales wedges are both demonstrable
- packaging language is consistent across the top-level docs that steer project understanding
- no release-readiness statement depends on broad mobile or marketplace delivery

> **Wave 4 status: COMPLETED.** Sales brief flow shipped (`POST /api/v1/copilot/sales-brief`). Packaging/messaging aligned in README.md and architecture docs. Demo/UAT bundle documented in `docs/wedge-demo-uat-summary.md`.

---

## 8. Public API and Interface Adjustments

The following APIs are strategic and must be documented as such:

- `POST /api/v1/knowledge/ingest`
- `POST /api/v1/knowledge/search`
- `POST /api/v1/knowledge/evidence`
- `POST /api/v1/copilot/chat`
- `POST /api/v1/agents/trigger`
- `POST /api/v1/agents/support/trigger`
- `GET /api/v1/agents/runs/{id}`
- `GET /api/v1/approvals`
- `PUT /api/v1/approvals/{id}`
- `GET /api/v1/audit/events`
- `GET /health`
- `GET /metrics`
- `GET /api/v1/usage`
- `GET /api/v1/quota-state`

### Spec-to-implementation API reconciliation

The repositioning spec (`STRAT-ARCH-001`, Section 12.1) defined strategic APIs using abstract route names. The implementation uses the concrete routes already present in the codebase. The mapping is:

| Spec route (Section 12.1) | Implementation route | Reason for difference |
|---|---|---|
| `POST /knowledge/search` | `POST /api/v1/knowledge/search` | Prefixed with `/api/v1` per existing convention |
| `POST /knowledge/ingest` | `POST /api/v1/knowledge/ingest` | Same |
| `POST /copilot/query` | `POST /api/v1/copilot/chat` | Existing codebase uses `chat` — SSE streaming semantics |
| `POST /agent-runs` | `POST /api/v1/agents/trigger`, `POST /api/v1/agents/support/trigger` | Split into generic and support-specific triggers |
| `POST /approvals/{id}/approve` | `PUT /api/v1/approvals/{id}` | Single endpoint with `decision` body field; BFF exposes verb aliases |
| `POST /approvals/{id}/reject` | `PUT /api/v1/approvals/{id}` | Same endpoint, `decision: "reject"` |
| `GET /audit-events` | `GET /api/v1/audit/events` | Prefixed |
| `GET /usage` | `GET /api/v1/usage` | Prefixed |
| `GET /health` | `GET /health` | Unchanged |
| `GET /metrics` | `GET /metrics` | Unchanged |
| *(not in spec)* | `POST /api/v1/knowledge/evidence` | Added — dedicated evidence pack endpoint required by wedge |
| *(not in spec)* | `GET /api/v1/agents/runs/{id}` | Added — run inspection required by activity log and audit |
| *(not in spec)* | `GET /api/v1/quota-state` | Added — quota visibility required by governance surface |
| *(not in spec)* | `POST /api/v1/copilot/sales-brief` | Added — dedicated sales brief contract required by sales wedge |

For each strategic API, the implementation work must define:

- input schema
- output schema
- deterministic error codes
- policy evaluation touchpoint
- audit emission behavior
- tenant isolation rule
- correlation and idempotency semantics where writes occur

Compatibility rule:

- keep current route shapes where practical
- normalize behavior first
- defer cosmetic route redesign unless it unlocks a wedge requirement

---

## 9. Deferred Work

The following items are explicitly deferred and shall not block this plan:

- broad mobile feature parity
- new mobile-first workflows
- plugin marketplace work
- broad Agent Studio capabilities beyond wedge needs
- CRM object expansion not directly required by support or sales flows
- platform-first migration work whose primary goal is DSL breadth rather than wedge delivery

Allowed exception:

- compatibility or stability fixes in `mobile/` or `bff/` are acceptable if they protect an existing flow or test path

Related mobile plans:

- `docs/plans/mobile_wedge_harmonization_plan.md` — constrains the mobile surface to five wedge-aligned tabs and removes non-wedge breadth. Functional E2E stays on Detox; Maestro stays for screenshots only.
- `docs/plans/maestro-screenshot-migration.md` — migrates the screenshot suite from Detox to Maestro. After the mobile harmonization Wave 6 completes, the Maestro visual-audit flow must be updated to reflect the wedge-first navigation (see coordination note in the mobile harmonization plan, Section 8.8).

---

## 10. Testing and Acceptance Plan

### Contract tests

- evidence responses return the full `Evidence Pack v1` field set
- approval APIs expose normalized state semantics
- agent run inspection exposes normalized public outcomes
- usage endpoints return per-workspace and per-run attribution without tenant leakage

### Integration tests

- support run with allowed tool execution
- support run that enters `awaiting_approval`
- support run that becomes `handed_off`
- support run that becomes `denied_by_policy`
- sales copilot query with sufficient evidence
- sales copilot query that abstains due to insufficient evidence

### Regression tests

- no cross-tenant evidence leakage
- audit events are emitted for governed actions and approval transitions
- usage events are emitted even when cost metadata is partial
- existing BFF/mobile smoke flows keep working where already covered

### Final acceptance

This plan is accepted only when:

- support wedge is demonstrable end-to-end
- sales wedge is demonstrable end-to-end
- audit trail is visible for governed actions
- usage is reportable per workspace and per run
- top-level planning/docs no longer route execution back to the superseded MVP framing

---

## 11. Immediate Follow-Up After Plan Acceptance

Once this document is accepted, the next planning artifacts to create are:

1. implementation tasks for each wave in `docs/tasks/` using valid task frontmatter
2. a technical spec for the `usage/quota` domain
3. a contract spec for `Evidence Pack v1`
4. a canonical support reference flow document
5. a canonical sales copilot reference flow document

Those documents must decompose this plan without changing its priority order.
