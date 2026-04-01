# Agent Handoff — FenixCRM Next Iteration

**Date**: 2026-04-01
**Prepared by**: audit session (Claude Opus 4.6)
**Purpose**: Give the incoming agent a complete, unambiguous picture of repo state so it can start the next design plan without re-auditing.

---

## 1. Repo State

Branch: `main`

### Uncommitted changes (not yet committed — stage and commit before starting work)

| File | Status | What changed |
|---|---|---|
| `docs/architecture.md` | Modified | `source_type` enum corrected; `policy_version_id` annotated as planned |
| `docs/openapi.yaml` | Modified | 25+ routes added, method mismatches fixed, phantom route removed, FR-220 traces replaced |
| `features/uc-b1-safe-tool-routing.feature` | Modified | UC tag corrected from `@UC-C1` to `@UC-B1` |
| `docs/waves-1-9-audit-handoff.md` | Untracked | Full audit handoff document — new |
| `docs/parallel_requirements.md` | Untracked | Wave sequencing and dependency slicing — new |
| `docs/wave1-governance-audit-retrieval-analysis.md` through `wave9-*.md` | Untracked | Nine wave analysis documents — new |
| `reqs/FR/FR_005.yml` through `FR_320.yml` | Untracked | 11 new FR artifacts — new |

**Commit message suggestion**:
```
docs(audit): reconcile documentation before wave iteration

- Add 11 missing FR artifacts (FR-005, FR-050, FR-094, FR-120, FR-212,
  FR-233, FR-234, FR-241, FR-242, FR-310, FR-320)
- Fix architecture.md source_type enum drift and annotate policy_version_id
- Reconcile openapi.yaml with routes.go (25+ routes, method fixes, phantom removal)
- Add wave analysis docs and audit handoff
- Fix uc-b1-safe-tool-routing.feature UC tag
```

---

## 2. Source of Truth Hierarchy

Use this order when sources conflict. Higher = more authoritative.

1. `docs/requirements.md` — business intent
2. `reqs/FR/*.yml` — requirement traceability artifacts
3. `docs/parallel_requirements.md` — canonical wave sequencing and FR classification
4. `docs/architecture.md` — target system model and data shapes
5. `docs/openapi.yaml` — published supported API surface
6. Wave analysis documents (`docs/wave1-*.md` through `docs/wave9-*.md`) — design intent per wave
7. `docs/as-built-design-features.md` — current implementation baseline
8. `docs/fr-gaps-implementation-criteria.md` — gap framing and closure criteria
9. Runtime code under `internal/` — implementation reality

---

## 3. Program Structure

The work is organized into nine waves across three maturity bands.

### Waves 1–3: P0 Closure Backbone (highest priority)

These form the contract backbone that all later waves consume.

- **Wave 1**: governance, approvals, policy, audit, hybrid retrieval, scoped FR-091
- **Wave 2**: tooling safety, grounded copilot, prompt lifecycle, CRM hardening
- **Wave 3**: agent runtime, handoff, pending-approval state, minimum agent catalog

Status: architecturally complete with partial gaps. See section 5.

### Waves 4–5: Unblocked Post-P0 Expansion

- **Wave 4**: CRM extensibility, freshness policies, skills, replay, mobile push/offline
- **Wave 5**: packaging and distribution (FR-052 plugin SDK, internal registry)

Status: design documents exist. No external blockers. Start after Waves 1–3 gaps are closed.

### Waves 6–9: Blocked Expansion

- **Wave 6**: connector expansion — **blocked by FR-320** (unresolved gate)
- **Wave 7**: non-LLM automation, scheduling/triggers — **blocked by FR-120** (unresolved gate)
- **Wave 8**: dedupe maturity, unified quotas, standalone eval gating — **blocked by FR-310** (unresolved gate)
- **Wave 9**: Carta behavior contracts — **blocked by Wave 8** (needs stable quota model)

These waves have design documents and FR artifacts but none are `active: true`. Do not treat them as implementation-ready.

---

## 4. FR Artifact Status

**Total artifacts**: 33 files in `reqs/FR/`

**Active (implementation-ready)**:
FR-001, FR-002, FR-051, FR-052, FR-060, FR-061, FR-070, FR-071, FR-090, FR-091, FR-092, FR-202, FR-240, NFR-031

**Inactive — partial implementation, closure criteria defined**:
FR-200, FR-201, FR-210, FR-211, FR-230, FR-231, FR-232
(see `docs/fr-gaps-implementation-criteria.md` for exact closure criteria per FR)

**Inactive — gate artifacts (unresolved external dependencies)**:
FR-120, FR-310, FR-320
These are blocking gates. Set `active: true` only when the gate decision is made.

**Inactive — deferred expansion (post-P0, no blockers)**:
FR-005, FR-050, FR-094, FR-212, FR-233, FR-234, FR-241, FR-242

---

## 5. Known Implementation Gaps — Waves 1–3

These are the gaps that must close before Waves 4–5 can start. Source: `docs/fr-gaps-implementation-criteria.md`.

### FR-060 — RBAC/ABAC enforcement
- **What exists**: JWT auth, workspace isolation, policy engine
- **What is missing**: Consistent ABAC enforcement across all decision points; authorization matrix tests
- **Closure criterion**: Every protected route enforces role + attribute checks; integration tests cover denial paths

### FR-061 — Approval-before-execution
- **What exists**: Approval CRUD, list/decide endpoints
- **What is missing**: Sensitive actions do not universally block on pending approval; end-to-end linkage
- **Closure criterion**: Tool execution and agent actions with approval-required policy are blocked until approved

### FR-070 — Audit trail hardening
- **What exists**: Audit events stored, queryable, exportable
- **What is missing**: Append-only not enforced at DB level; event taxonomy not standardized; subscriber coverage incomplete
- **Closure criterion**: DB-level immutability; domain-specific event types logged; trace_id correlation verified

### FR-071 — Policy engine
- **What exists**: Policy engine with rule evaluation and caching
- **What is missing**: Workspace-scoped policy-set/version resolution not guaranteed consistent; deterministic resolution not tested
- **Closure criterion**: All resolution paths use the same policy-set lookup; rule traces logged consistently

### FR-090 — Hybrid retrieval
- **What exists**: FTS5 BM25 + in-memory cosine similarity with RRF ranking
- **What is missing**: sqlite-vec backend not implemented (using in-memory JSON); no benchmark coverage
- **Closure criterion**: sqlite-vec integrated; BM25 + ANN hybrid verified with latency benchmarks

### FR-091 — CDC and reindex
- **What exists**: CDC event subscription, reindex service with retry
- **What is missing**: Entity coverage incomplete (not all CRM mutations trigger CDC); <60s freshness SLA not measured
- **Closure criterion**: All CRM entity mutations covered; reindex latency metric exposed

### FR-200 — Copilot abstention
- **What exists**: SSE streaming chat, evidence-grounded responses
- **What is missing**: Mandatory abstention when evidence is insufficient not enforced
- **Closure criterion**: Abstention policy triggers when evidence confidence is below threshold; abstain_reason returned

### FR-202 — Tool registry lifecycle
- **What exists**: Tool list/create, schema validation, permission enforcement
- **What is missing**: Update/activate/deactivate/delete admin lifecycle; strong parameter validation
- **Closure criterion**: Full CRUD lifecycle; schema validation rejects invalid params at definition time

### FR-211 — Built-in tool execution
- **What exists**: Built-in tools registered and callable
- **What is missing**: No unified execution pipeline (policy check + schema validation + audit) for all tool calls
- **Closure criterion**: Single execution path used for all tools; every execution audited; policy decision logged

### FR-230 — Agent runtime
- **What exists**: Orchestrator, DSL runner, run persistence, per-run metrics
- **What is missing**: Multi-step state machine incomplete; standardized retry/recovery on step failure
- **Closure criterion**: Explicit state transitions (pending → running → success/abstained/failed); retry policy configurable per agent

### FR-231 — Agent catalog
- **What exists**: Support, prospecting, KB, insights agents
- **What is missing**: UC-C1 flow has placeholders; execution is not uniformly data-driven
- **Closure criterion**: Each catalog agent runs a fully deterministic flow against real data; no hardcoded stubs in happy path

### FR-232 — Human handoff
- **What exists**: Handoff package generation, initiate/get endpoints
- **What is missing**: Full conversation context not guaranteed; evidence package format not stable contract
- **Closure criterion**: Handoff payload includes complete run trace + evidence + reasoning; format is schema-validated

### FR-240 — Prompt lifecycle
- **What exists**: Create/list/promote/rollback, A/B experiment infrastructure
- **What is missing**: Eval-gated promotion not implemented; rollback API semantics inconsistent
- **Closure criterion**: Promotion blocked if eval suite exists and thresholds not met; rollback idempotent

---

## 6. Single Most Important Open Code Gap

**GroundsValidator is not wired in any production execution path.**

The validator exists at `internal/domain/agent/grounds_validator.go` and is tested in integration tests, but the three production `RunContext` instances that drive all agent execution do not set it:

| Location | Fix |
|---|---|
| `internal/api/handlers/workflow.go` — `executeDSLWorkflow()` | Add `GroundsValidator: agent.NewGroundsValidator(evidenceBuilder)` to `RunContext` |
| `internal/api/routes.go` — `resumeRC` (line ~395) | Same |
| `internal/api/handlers/insights_shadow.go` — `rc` | Same |

Because `dsl_runner.go:groundsPolicyApplies()` checks `rc.GroundsValidator != nil` and silently returns nil when it is not set, agents can execute without evidence in production. This violates the evidence-first design principle and means Wave 9 GROUNDS enforcement is infrastructure only, not behavior.

The `evidenceBuilder` to inject is `knowledge.NewEvidencePackService(...)` — already instantiated in `routes.go` as `evidenceSvc`.

---

## 7. API Contract State

`docs/openapi.yaml` was reconciled with `internal/api/routes.go` in this session.

**What was fixed**:
- 25+ missing routes added (eval, workflow lifecycle, audit, reports, tool admin, agents, metrics)
- Method mismatches corrected (PUT vs PATCH, PUT vs POST)
- Phantom `POST /signals` removed
- `FR-220` phantom traces replaced with `FR-240`/`FR-230`

**Rule going forward**: Any new route added to `routes.go` must be added to `openapi.yaml` in the same commit. The `x-fr-traces` field must reference a real FR identifier present in `reqs/FR/`.

---

## 8. Schema Drift State

| Item | State |
|---|---|
| `knowledge_item.source_type` | Resolved — `architecture.md` matches migration and code |
| `eval_run.policy_version_id` | Partially resolved — field annotated as "planned" in `architecture.md`; not in migration 020 or Go code; Wave 8 eval gating is prompt-only until this is added |

---

## 9. What to Do Next

### Step 1 — Commit the current working tree

Stage and commit everything in `git status --short`. Use the commit message from section 1.

### Step 2 — Decide on the next wave scope

Read `docs/parallel_requirements.md` section "Full Wave Roadmap" to pick the next workstream. The natural next targets are the Wave 1–3 gap closures in section 5 above, since all of Wave 4+ depends on them.

### Step 3 — Before starting any task

Follow the protocol in `CLAUDE.md`:
1. Read `docs/implementation-plan.md` for the task spec
2. Read `docs/architecture.md` for architectural constraints
3. Set agent attribution: `export AI_AGENT="claude-sonnet-4-6"` and `git config fenix.ai-agent "claude-sonnet-4-6"`

### Step 4 — Recommended first task

Wire `GroundsValidator` into the three production `RunContext` instances (section 6 above). This is a self-contained code change with clear scope, no external dependencies, and high architectural importance. The evidence builder is already available in `routes.go`.

### Step 5 — After wiring GroundsValidator

Close the remaining Wave 1–3 gaps in this order (each is a prerequisite for the next):
1. FR-070 audit hardening (append-only + taxonomy)
2. FR-061 approval-before-execution linkage
3. FR-060 ABAC enforcement consistency
4. FR-071 policy resolution determinism
5. FR-200 mandatory abstention
6. FR-091 CDC entity coverage + freshness metric
7. FR-090 sqlite-vec backend
8. FR-202 tool registry full lifecycle
9. FR-211 unified tool execution pipeline
10. FR-230 agent state machine
11. FR-231 catalog deterministic flows
12. FR-232 handoff contract stability
13. FR-240 eval-gated promotion

---

## 10. Files to Load for Context

Minimum context to start work:

```
docs/requirements.md
docs/architecture.md
docs/implementation-plan.md
docs/parallel_requirements.md
docs/as-built-design-features.md
docs/fr-gaps-implementation-criteria.md
docs/openapi.yaml
```

For wave-specific work, also load the relevant wave analysis document (e.g., `docs/wave1-governance-audit-retrieval-analysis.md`).

For audit or traceability work, also load:

```
docs/waves-1-9-audit-handoff.md
reqs/FR/<relevant FR>.yml
features/<relevant UC>.feature
```

---

## 11. Do Not Do

- Do not start Wave 6, 7, 8, or 9 implementation. Their gates (FR-120, FR-310, FR-320) are unresolved.
- Do not treat `active: false` FR artifacts as having closure criteria met.
- Do not add routes to `routes.go` without adding them to `openapi.yaml` in the same commit.
- Do not assume `GroundsValidator` is enforced in production — it is not (see section 6).
- Do not amend the previous commit. Create new commits.
- Do not skip the agent attribution step before committing (`export AI_AGENT` + `git config`).
