---
title: FenixCRM — FR & UC Implementation Status
last_updated: 2026-04-03
tags: [dashboard, status]
---

# FenixCRM — FR & UC Implementation Status

> Last updated: 2026-04-03
> Source of truth: `docs/requirements.md` (v2.0) + BDD feature files in `features/`

---

## 1. Functional Requirements (FR) — Implementation Status

Legend: ✅ Implemented | ⏳ Partial | ❌ Not implemented | 🔒 P1/P2 (deferred)

### CRM Core

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-001 | Core CRM entities (Account, Contact, Lead, Deal, Case) | P0 | ✅ | task_1.3, task_1.4, task_1.5 | @FR-001 in uc-s1, uc-c1 |
| FR-002 | Activity, Note, Attachment | P0 | ✅ | task_1.5 | — |
| FR-003 | Pipeline stages | P0 | ✅ | task_1.5 | — |
| FR-004 | Timeline / audit per record | P0 | ✅ | task_1.7 | — |
| FR-052 | Plugin marketplace | P2 | 🔒 | — | — |

### Knowledge & RAG

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-090 | Hybrid retrieval (BM25 + vector) | P0 | ✅ | task_2.5 | @FR-090 in uc-d1, uc-s1, uc-c1 |
| FR-091 | Multi-source ingestion (email, docs, calls) | P0/P1 | ⏳ | task_2.2 (base) | @FR-091 in uc-d1 |
| FR-092 | Evidence pack (mandatory, grounded) | P0 | ✅ | task_2.6 | @FR-092 in uc-s1, uc-c1, uc-s2, uc-k1, uc-d1 |
| FR-093 | Incremental reindex via CDC (<60s SLA) | P0 | ✅ | task_2.7 | — |
| FR-094 | Knowledge item CRUD + workspace isolation | P0 | ✅ | task_2.1, task_2.2 | — |

### Copilot

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-200 | Copilot in-flow UI (SSE streaming) | P0 | ✅ | task_3.4, task_4.4 | @FR-200 in uc-s1 |
| FR-201 | Copilot context (retrieval + evidence) | P0 | ✅ | task_3.4 | @FR-200 in uc-s1 |
| FR-202 | Copilot actions (tool execution from UI) | P0 | ✅ | task_3.3, task_4.4 | @FR-202 in uc-s1, uc-a4 |
| FR-212 | Behavior contracts (Carta spec) | P1 | 🔒 | — | — |

### AI Behavior

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-210 | Mandatory abstention (insufficient evidence) | P0 | ✅ | task_3.4 | @FR-210 in uc-c1 (abstention scenario) |
| FR-211 | Safe tool routing (no dangerous tools) | P0 | ✅ | task_3.1, task_3.3 | @FR-211 in uc-b1 |
| FR-213 | Model-agnostic LLM adapter | P1 | ✅ | task_2.3 | — |

### Agent Runtime

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-230 | Agent definition + execution | P0 | ✅ | task_3.5 | multiple UCs |
| FR-231 | Agent catalog (prospecting, KB, insights) | P1 | ✅ | task_4.5b, task_4.5c, task_4.5d | uc-s2, uc-k1, uc-d1 |
| FR-232 | Human handoff with evidence | P0 | ✅ | task_3.7 | uc-c1 (handoff scenario) |
| FR-233 | Budget controls / quotas per agent | P1 | 🔒 | — | — |
| FR-243 | Replay / simulation | P1 | 🔒 | — | — |

### Agent Studio

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-240 | Prompt / policy versioning | P0 | ✅ | task_3.9 | uc-a2, uc-a3, uc-a8 |
| FR-241 | Workflow authoring (DSL) | P1 | ✅ | task_4.5a | uc-a2 |
| FR-242 | Skills builder | P1 | ✅ | task_4.5a | uc-a4 |
| FR-243 | Eval suite + gating | P1 | ✅ | task_4.7 | — |

### Integrations

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-050 | Email connector | P0 | ⏳ | task_2.2 (stub) | — |
| FR-051 | Document connector | P1 | 🔒 | — | — |
| FR-052 | Plugin SDK | P2 | 🔒 | — | — |

### Security & Audit

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-060 | RBAC / ABAC + policy engine | P0 | ✅ | task_1.6, task_3.1 | uc-g1 |
| FR-070 | Immutable audit trail | P0 | ✅ | task_1.7, task_4.6 | uc-g1, uc-a7 |
| FR-071 | Approval chains | P0 | ✅ | task_3.2 | uc-c1 (approval scenario), uc-a7 |
| FR-072 | PII redaction (no-cloud policy) | P0 | ✅ | task_3.1 | — |

### Mobile & BFF

| FR | Title | Priority | Status | Implementing Task | BDD Coverage |
|----|-------|----------|--------|-------------------|--------------|
| FR-300 | Mobile app (React Native + Expo) | P0 | ✅ | task_4.2, task_4.3, task_4.4, task_4.5 | uc-s1 (mobile, partial) |
| FR-301 | BFF Gateway (Express.js thin proxy) | P0 | ✅ | task_4.1 | — |
| FR-302 | Observability (/metrics, /health) | P0 | ✅ | task_4.9 | — |
| FR-303 | Mobile eval service | P1 | 🔒 | — | — |

---

## 2. Use Case Coverage Matrix

| UC | Title | Feature File | Go BDD | BFF BDD | Mobile BDD | Doorstop | Gaps |
|----|-------|-------------|--------|---------|------------|----------|------|
| UC-S1 | Sales Copilot | `uc-s1-sales-copilot.feature` | ✅ | ⏳ | ⏳ partial | ✅ UC_S1.yml | Copilot section on Account/Deal screens |
| UC-S2 | Prospecting Agent | `uc-s2-prospecting-agent.feature` | ✅ | ⏳ | ❌ | ✅ UC_S2.yml | Mobile trigger button |
| UC-S3 | Deal Risk Agent | `uc-s3-deal-risk-agent.feature` | ✅ | ⏳ | ❌ | ✅ UC_S3.yml | deal_risk.go agent + mobile |
| UC-C1 | Support Agent | `uc-c1-support-agent.feature` | ✅ | ⏳ | ❌ | ✅ UC_C1.yml | — (Go fully covered) |
| UC-K1 | KB Agent | `uc-k1-kb-agent.feature` | ✅ | ⏳ | ❌ | ✅ UC_K1.yml | Mobile KB trigger |
| UC-D1 | Data Insights Agent | `uc-d1-data-insights-agent.feature` | ✅ | ⏳ | ❌ | ✅ UC_D1.yml | Mobile Insights screen |
| UC-G1 | Governance | `uc-g1-governance.feature` | ✅ | ⏳ | ❌ | ✅ UC_G1.yml | Audit Log mobile screen |
| UC-A1 | Agent Studio | `uc-a1-agent-studio.feature` | ✅ | ⏳ | ❌ | ✅ UC_A1.yml | — |
| UC-A2 | Workflow Authoring | `uc-a2-workflow-authoring.feature` | ✅ | ⏳ | ❌ | ✅ UC_A2.yml | — |
| UC-A3 | Workflow Verification | `uc-a3-workflow-verification-and-activation.feature` | ✅ | ⏳ | ❌ | ✅ UC_A3.yml | — |
| UC-A4 | Workflow Execution | `uc-a4-workflow-execution.feature` | ✅ | ⏳ | ❌ | ✅ UC_A4.yml | — (4 scenarios: happy, condition_false, tool_failure, approval) |
| UC-A5 | Signal Detection | `uc-a5-signal-detection-and-lifecycle.feature` | ✅ | ⏳ | ❌ | ✅ UC_A5.yml | — |
| UC-A6 | Deferred Actions | `uc-a6-deferred-actions.feature` | ✅ | ⏳ | ❌ | ✅ UC_A6.yml | — (3 scenarios: happy, archived, failure) |
| UC-A7 | Human Override & Approval | `uc-a7-human-override-and-approval.feature` | ✅ | ⏳ | ❌ | ✅ UC_A7.yml | — |
| UC-A8 | Workflow Versioning | `uc-a8-workflow-versioning-and-rollback.feature` | ✅ | ⏳ | ❌ | ✅ UC_A8.yml | — |
| UC-A9 | Agent Delegation | `uc-a9-agent-delegation.feature` | ✅ | ⏳ | ❌ | ✅ UC_A9.yml | — |
| UC-B1 | Safe Tool Routing | `uc-b1-safe-tool-routing.feature` | ✅ | ⏳ | ❌ | ✅ UC_B1.yml | — |

**Totals:** 17 UCs | Go: 17/17 ✅ | BFF: 0/17 ⏳ | Mobile: 0/17 ❌ (1 partial)

---

## 3. UC Gap Closure — Pending Tasks

Source: `docs/tasks/task_uc_gap_closure.md`

### P0 Gaps (must close before release)

| Task | UC | Gap | Status |
|------|-----|-----|--------|
| Gap-1 | UC-S1 | Extend `suggest_actions.go` for Account/Deal (BE only responds to Case today) | ❌ pending |
| Gap-2 | UC-S1, UC-S3 | Add `get_deal` and `update_deal` to tool registry | ❌ pending |
| Gap-3 | UC-S1 | Copilot panel section on Account detail + Deal detail screens (Mobile) | ❌ pending |
| Gap-4 | UC-G1 | Audit Log screen in mobile app + `/audit/events` API methods | ❌ pending |

### P1 Gaps (next release)

| Task | UC | Gap | Status |
|------|-----|-----|--------|
| Gap-5 | UC-S3 | Implement `deal_risk.go` agent | ❌ pending |
| Gap-6 | UC-S3 | Wire `deal_risk` runner adapter + registry + handler endpoint | ❌ pending |
| Gap-7 | UC-S3, UC-S2 | Agent trigger buttons in Deal detail + Leads screen (Mobile) | ❌ pending |
| Gap-8 | UC-K1, UC-D1 | KB search trigger + Data Insights screen (Mobile) | ❌ pending |

---

## 4. BDD Pipeline Status

| Gate | CI Status | Command |
|------|-----------|---------|
| Doorstop integrity | ✅ Passing | `make doorstop-check` |
| UC→FR→TST traceability | ✅ Passing | `make bdd-trace-check` |
| Go BDD runner (33 scenarios) | ✅ Passing | `make test-bdd-go` |
| BFF BDD runner | ⏳ Not implemented | `make test-bdd-bff` (reserved) |
| Mobile BDD runner | ❌ Blocked (Android SDK) | `make test-bdd-mobile` |

---

## 5. ADR Index

All architectural decisions are in `docs/decisions/`:

| ADR | Title | Area |
|-----|-------|------|
| ADR-001 | UUID full slug (no truncation) | Database / Testing |
| ADR-002 | sqlite-vec multi-tenant JOIN | Security / Vector Search |
| ADR-003 | TestMain + os.Setenv for JWT_SECRET | Testing / Auth |
| ADR-004 | t.Parallel + t.Setenv incompatibility | Testing / Go |
| ADR-005 | goconst + shadow in handlers | Go / Lint |
| ADR-006 | gocyclo complexity gate ≤7 | Quality / Go |
| ADR-007 | pkgauth alias for domain/auth | Go / Packages |
| ADR-008 | Route structure — public vs protected | Auth / Routing |
| ADR-009 | BFF as thin proxy | Architecture |
| ADR-010 | modernc.org/sqlite CGO_ENABLED=0 | Deploy / SQLite |
| ADR-011 | FTS5 sync via triggers | SQLite / Search |
| ADR-012 | Chunking strategy 512/50 | RAG / Knowledge |
| ADR-013 | Embedding async pipeline | RAG / Event Bus |
| ADR-014 | Hybrid search RRF k=60 | RAG / Search |
| ADR-015 | Evidence deduplication cosine >0.95 | RAG / Evidence |
| ADR-016 | CDC reindex SLA <60s | CDC / Search |
| ADR-017 | Quality gates gocognit/maintidx | Quality / Lint |
| ADR-018 | BDD pipeline strategy | Testing / BDD |
