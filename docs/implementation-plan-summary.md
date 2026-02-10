# Plan de Implementación — Resumen Ejecutivo

> **Duración**: 13 semanas | **Fases**: 4 | **Enfoque**: TDD, vertical slices, quality gates

---

## Cronograma de Alto Nivel

```
┌─────────────┬─────────────┬─────────────┬─────────────┐
│   Phase 1   │   Phase 2   │   Phase 3   │   Phase 4   │
│  Foundation │  Knowledge  │  AI Layer   │ Integration │
│   Weeks 1-3 │  Weeks 4-6  │  Weeks 7-10 │ Weeks 11-13 │
└─────────────┴─────────────┴─────────────┴─────────────┘
      ↓              ↓              ↓              ↓
  CRM CRUD      Hybrid Search   Copilot +      React UI +
  + Auth        + Evidence      Agents +       Audit +
  + Audit       + CDC           Tools +        Eval +
                                Policy         Observability
```

---

## Phase 1: Foundation (Weeks 1-3)

**Objetivo**: CRM operacional con CRUD APIs, auth, audit.

| Week | Task | Deliverable | Tests |
|------|------|-------------|-------|
| 1 | 1.1 Project Setup | Go scaffolding + CI | Pipeline green |
| 1 | 1.2 SQLite + Migrations | DB schema (workspace, user, role) | Schema verified |
| 2 | 1.3 Account Entity | Account CRUD + API | API tests pass |
| 2 | 1.4 Contact Entity | Contact CRUD + API | API tests pass |
| 3 | 1.5 Lead/Deal/Case + Supporting | All CRM entities + activity/note/timeline | **EXPANDED** |
| 3 | 1.6 Auth Middleware | JWT auth on all routes | Auth tests pass |
| 3 | 1.7 Audit Foundation | Immutable audit log | **NEW — CRITICAL** |

**Exit Criteria**: ✅ CRUD working ✅ Auth active ✅ Audit logging ✅ Multi-tenancy verified

---

## Phase 2: Knowledge & Retrieval (Weeks 4-6)

**Objetivo**: Hybrid search (BM25 + vector) con evidence packs.

| Week | Task | Deliverable | Tests |
|------|------|-------------|-------|
| 4 | 2.1 Knowledge Tables | Schema + FTS5 + sqlite-vec | **SECURITY FIX** (multi-tenant) |
| 4 | 2.2 Ingestion Pipeline | Document ingestion + chunking | Ingestion tests pass |
| 5 | 2.3 LLM Adapter | Ollama provider interface | LLM integration tests |
| 5 | 2.4 Embed & Index | Embedding generation + storage | Vectors in DB |
| 6 | 2.5 Hybrid Search | BM25 + vector with RRF merge | Search tests pass |
| 6 | 2.6 Evidence Pack Builder | Top-K with confidence scoring | Evidence tests pass |
| 6 | 2.7 CDC & Auto-Reindex | Change Data Capture flow | **NEW — CRITICAL** |

**Exit Criteria**: ✅ Hybrid search working ✅ Multi-tenant verified ✅ CDC <60s SLA ✅ Evidence packs

---

## Phase 3: AI Layer (Weeks 7-10)

**Objetivo**: Copilot, Support Agent (UC-C1), Tools, Policy.

| Week | Task | Deliverable | Tests |
|------|------|-------------|-------|
| 7 | 3.1 RBAC/ABAC Evaluator | Policy engine (4 enforcement points) | Permission tests |
| 7 | 3.2 Approval Workflow | approval_request table + handlers | Approval tests |
| 8 | 3.3 Tool Registry | Tool definition + validation | Registry tests |
| 8 | 3.4 Built-in Tools | create_task, update_case, send_reply | Tool execution tests |
| 9 | 3.5 Copilot Chat | SSE streaming + citations | Copilot integration tests |
| 9 | 3.6 Copilot Actions | Suggest actions + summarize | Action tests |
| 10 | 3.7 Agent Orchestrator | UC-C1 Support Agent end-to-end | **E2E critical** |
| 10 | 3.8 Handoff Manager | Escalation + context package | Handoff tests |
| 10 | 3.9 Prompt Versioning | Promote/rollback capability | **NEW** |

**Exit Criteria**: ✅ UC-C1 working ✅ Copilot streaming ✅ Tools + approvals ✅ Policy enforced ✅ Prompt versioning

---

## Phase 4: Integration & Polish (Weeks 11-13)

**Objetivo**: React UI, observability, eval, E2E tests.

| Week | Task | Deliverable | Tests |
|------|------|-------------|-------|
| 11 | 4.1 Frontend Setup | React + Vite + shadcn/ui | E2E login test |
| 11 | 4.2 CRM Pages | Account/Contact/Deal/Case pages | E2E CRUD tests |
| 12 | 4.3 Copilot Panel | Chat UI + SSE + evidence cards | E2E copilot tests |
| 12 | 4.4 Agent Runs Dashboard | Run list + detail views | E2E agent tests |
| 13 | 4.5 Audit Service Advanced | Query + export + full event bus | Export tests |
| 13 | 4.6 Eval Service | Basic suite + scoring | Eval tests |
| 13 | 4.7 E2E Tests + Docs | UC-C1 complete + README | **100% pass** |
| 13 | 4.8 Observability | /metrics + /health + dashboard | **NEW** |

**Exit Criteria**: ✅ React UI working ✅ Audit export ✅ Observability ✅ E2E tests 100% ✅ Docs updated

---

## Correcciones Críticas Aplicadas

| # | Corrección | Impacto | Rationale |
|---|------------|---------|-----------|
| 1 | **Task 1.7 Audit Foundation** (NEW) | P0 Blocker | Governed systems require audit-first, not audit-last |
| 2 | **Task 1.5 Expanded** (activity/note/timeline) | Dependency Fix | Tools in Phase 3 depend on these entities |
| 3 | **Task 2.1 Multi-tenant Vector** (Security Fix) | P0 Security Blocker | sqlite-vec has no native tenant filter — JOIN required |
| 4 | **Task 2.7 CDC/Reindex** (NEW) | Freshness SLA | Architecture assumes <60s visibility, no mechanism existed |
| 5 | **Task 3.9 Prompt Versioning** (NEW) | Change Management | Architecture shows FK, rollback capability required |
| 6 | **Task 4.5 Audit Advanced** (Updated) | Scope Adjustment | Base audit moved to Phase 1, Phase 4 = query/export |
| 7 | **Task 4.8 Observability** (NEW) | NFR Compliance | Architecture requires metrics endpoint + dashboard |

---

## Decisiones Pendientes

### 1. Prompt Versioning (Task 3.9)

**Opciones**:
- **A**: Mantener en P0 (recomendado) — 1 día, arquitectura lo requiere
- **B**: Mover a P1 — requiere actualizar ERD (remover FK)

**Decisión**: Pendiente de aprobación

---

### 2. Estructura de Carpetas

**Decisión Tomada**: Option B (con `internal/`)

```
fenixcrm/
├── cmd/fenixcrm/main.go
├── internal/              # Private application code
│   ├── domain/
│   ├── infra/
│   ├── api/
├── pkg/                   # Public shared libraries
```

**Acción**: Actualizar `docs/architecture.md` Appendix

---

## Traceability Matrix (Living Document)

| Component | Entity | Status | Phase | Task | Notes |
|-----------|--------|--------|-------|------|-------|
| Workspace | `workspace` | ✅ | 1 | 1.2 | |
| Account | `account` | ✅ | 1 | 1.3 | |
| Activity | `activity` | ✅ | 1 | 1.5 | **CORRECTED** |
| Audit Event | `audit_event` | ✅ | 1 | 1.7 | **CORRECTED** |
| Knowledge Item | `knowledge_item` | ⚠️ | 2 | 2.1 | Multi-tenant fix |
| Agent Run | `agent_run` | ❌ | 3 | 3.7 | |
| Prompt Version | `prompt_version` | ❌ | 3 | 3.9 | **CORRECTED** |

**Legend**: ✅ Complete | ⚠️ Partial | ❌ Pending | 🔵 Out of scope (P1)

---

## Success Criteria (MVP P0)

### Functional

- ✅ All CRM entities CRUD working
- ✅ Hybrid search (BM25 + vector) functional
- ✅ Copilot chat with SSE + citations
- ✅ UC-C1 Support Agent end-to-end
- ✅ Tool execution with permissions + approvals
- ✅ Policy engine 4 enforcement points active
- ✅ Handoff to human working
- ✅ Prompt versioning functional

### Non-Functional

- ✅ Auth + RBAC + audit trail active
- ✅ Copilot Q&A < 3s p95
- ✅ Multi-tenancy verified (no cross-tenant leaks)
- ✅ E2E tests 100% passing
- ✅ Single binary deployment functional
- ✅ Observability endpoints working

---

## Próximos Pasos

1. **Review** de correcciones con el equipo
2. **Decisión** sobre prompt versioning (mantener o diferir)
3. **Actualizar** `docs/architecture.md` (estructura de carpetas)
4. **Comenzar** Phase 1, Task 1.1 (Project Setup)

---

**Referencias**:
- Plan completo: `docs/implementation-plan.md`
- Correcciones: `docs/CORRECTIONS-APPLIED.md`
- Arquitectura: `docs/architecture.md`
