# FenixCRM — FR Gaps Required for “Implemented” Status

> Date: 2026-03-05  
> Baseline: `docs/as-built-design-features.md`  
> Scope: FR items marked as Partial or Pending in the as-built matrix  
> Explicit exclusion: **FR-052 is not considered**

---

## 1) Scope Summary

The following FRs are **not fully implemented** and still require work:

- FR-001
- FR-060
- FR-061
- FR-070
- FR-071
- FR-090
- FR-091
- FR-200
- FR-201
- FR-202
- FR-211
- FR-230
- FR-231
- FR-232
- FR-240

This document states, for each FR:
1. what exists now,
2. what is still missing,
3. what concrete completion criteria must be met.

---

## FR-001 — Core CRM Entities

**Current implementation**
- CRUD routes and services exist for core entities (including Deal/Case list-create-update-delete).

**Missing to mark as implemented**
- Full CRUD behavior is not uniform across all listed entities.
- Domain validation is still limited for business consistency checks.
- Audit/timeline evidence is not fully standardized as explicit domain-level events per entity mutation.

**Completion criteria**
- Uniform CRUD contract for all entities in FR-001.
- Strong business validation for Deal/Case updates.
- Integration tests proving audit + timeline emission for create/update/delete.

---

## FR-060 — Authentication RBAC/ABAC

**Current implementation**
- JWT auth and bcrypt password hashing are implemented.

**Missing to mark as implemented**
- RBAC/ABAC enforcement is not consistently applied to all protected actions.
- ABAC evaluation is not comprehensive across resource/action decision points.

**Completion criteria**
- Authorization matrix enforced in API and tool execution paths.
- ABAC allow/deny decisions traceable and audited.
- E2E tests for role- and attribute-based denial scenarios.

---

## FR-061 — Approval Workflows

**Current implementation**
- Pending approvals can be listed and decided; expiration logic exists.

**Missing to mark as implemented**
- Sensitive actions are not universally gated by an approval-before-execution flow.
- End-to-end linkage between requested approval and blocked/unblocked action is incomplete.

**Completion criteria**
- Sensitive actions automatically create approval requests when required.
- Action execution is blocked until explicit decision.
- E2E tests covering pending → approve/deny/expire behavior.

---

## FR-070 — Audit Trail

**Current implementation**
- Audit events are stored and queried; request-level audit middleware is active.

**Missing to mark as implemented**
- Append-only behavior is not hardened at database policy level.
- Some audit records still rely on generic request actions instead of specific domain actions.
- Event subscriber handling does not guarantee full coverage for all event payload types.

**Completion criteria**
- Enforced immutability controls for `audit_event` data.
- Domain-specific audit actions for critical mutations and auth/system actions.
- Traceability test suite validating expected audit events per workflow.

---

## FR-071 — Policy Engine

**Current implementation**
- Policy engine exists with RBAC-oriented evaluation helpers.

**Missing to mark as implemented**
- Workspace-scoped policy-set/version semantics are not fully enforced in runtime authorization.
- Resource/action/effect matching is not fully standardized across all decision paths.

**Completion criteria**
- Runtime decisions consistently resolved by policy set/version.
- Deterministic allow/deny behavior with logged rule trace.
- Regression tests for conflicting and precedence-sensitive policies.

---

## FR-090 — Hybrid Indexing

**Current implementation**
- FTS5 is active; embeddings are stored; hybrid ranking logic exists.

**Missing to mark as implemented**
- Vector search is performed in memory over JSON vectors; it is not backed by `sqlite-vec` as required.

**Completion criteria**
- Vector retrieval implemented with a compliant vector backend (`sqlite-vec` or approved equivalent).
- Verified hybrid ranking behavior with reproducible tests.
- Performance and relevance benchmarks against current in-memory approach.

---

## FR-091 — CDC and Auto-Reindex

**Current implementation**
- CDC event handling and reindex service exist for part of CRM-linked flows.

**Missing to mark as implemented**
- Entity coverage is not complete for all required CRM changes.
- Search freshness SLA (<60s) is not enforced with measurable SLI/SLO evidence.

**Completion criteria**
- Full required CDC entity coverage.
- Observable reindex pipeline with error handling and retries.
- Automated test and metric proof for <60s visibility SLA.

---

## FR-200 — Copilot Q&A

**Current implementation**
- SSE chat path is implemented end-to-end (API/BFF/mobile).

**Missing to mark as implemented**
- Mandatory abstention when evidence is insufficient is not enforced.
- CRM context injection is present but not strong enough to guarantee grounded behavior in all cases.

**Completion criteria**
- Explicit abstention policy with deterministic triggers.
- Evidence-grounded responses with consistent source traceability.
- E2E tests for both sufficient-evidence and insufficient-evidence scenarios.

---

## FR-201 — Suggested Actions

**Current implementation**
- Suggested action generation endpoint and service exist.

**Missing to mark as implemented**
- Output does not include a formal confidence score per suggested action.
- Guardrails tied to entity state are not strict enough.

**Completion criteria**
- Contract includes confidence per action.
- Deterministic validation of action eligibility by context.
- Quality metrics for action usefulness and acceptance.

---

## FR-202 — Tool Registry

**Current implementation**
- Workspace-scoped tool definitions with list/create are implemented.

**Missing to mark as implemented**
- Admin API lifecycle is incomplete (update/activate/deactivate/delete not complete as a full managed lifecycle).
- Schema validation is minimal and allows weak schemas.
- Required-permission enforcement is not consistently guaranteed at execution time.

**Completion criteria**
- Full lifecycle API for tool definitions.
- Strong schema validation and runtime parameter validation.
- Mandatory permission gate before any tool execution.

---

## FR-211 — Built-in Tools

**Current implementation**
- Built-in tools are registered and executors are available.

**Missing to mark as implemented**
- A single, universal execution path with policy+validation+audit gates is not consistently enforced.
- Operational hardening is incomplete (uniform error contracts and execution guarantees).

**Completion criteria**
- Unified execution pipeline for all built-in tools.
- Consistent validation, authorization, and auditing.
- Security tests for permission-denied and malformed-input cases.

---

## FR-230 — Agent Runtime

**Current implementation**
- Orchestrator persists runs and supports basic run lifecycle APIs.

**Missing to mark as implemented**
- Generic multi-step runtime control loop is incomplete.
- Standardized handling of retries, recovery, and step-level orchestration is incomplete.

**Completion criteria**
- Explicit multi-step runtime state machine.
- Standardized tool/evidence/reasoning loop handling.
- Consistent per-run metrics (tokens, cost, latency, final status).

---

## FR-231 — Support Agent UC-C1

**Current implementation**
- Support/prospecting/kb/insights agent modules and endpoints exist.

**Missing to mark as implemented**
- UC-C1 support flow still contains placeholders and non-uniform real execution behavior in key steps.
- End-to-end “resolve or handoff” behavior is not fully validated as a production-complete flow.

**Completion criteria**
- Real data-driven support flow without placeholder context logic.
- Deterministic handoff when confidence is low.
- E2E UC-C1 test coverage for success, abstention, and handoff paths.

---

## FR-232 — Human Handoff

**Current implementation**
- Handoff package generation and endpoints are implemented.

**Missing to mark as implemented**
- Full preserved conversation context is not guaranteed as a stable contract.
- Human-consumable, standardized evidence package format is incomplete.

**Completion criteria**
- Complete handoff payload including conversation context, reasoning trace, evidence pack, and CRM context.
- Deterministic persistence/retrieval of handoff state.
- UAT validation for agent-to-human operational continuity.

---

## FR-240 — Prompt Versioning

**Current implementation**
- Create/list/promote/rollback are available.

**Missing to mark as implemented**
- A/B testing support is not implemented.
- Eval-gated release enforcement is not implemented.
- API semantics are inconsistent in rollback identity usage.

**Completion criteria**
- A/B experiment support per prompt version.
- Promotion blocked/enabled by eval gate outcome.
- Consistent and unambiguous API contracts for versioning operations.

---

## 2) Recommended Execution Order

1. Governance and authorization: FR-060, FR-061, FR-071, FR-202, FR-211  
2. AI runtime quality and safety: FR-200, FR-201, FR-230, FR-231, FR-232  
3. Knowledge indexing reliability: FR-090, FR-091  
4. Prompt lifecycle maturity: FR-240  
5. CRM consistency hardening: FR-001
