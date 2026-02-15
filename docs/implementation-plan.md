# Implementation Plan — FenixCRM MVP (P0)

> **Status**: Ready for execution
> **Duration**: 13 weeks (3 months)
> **Based on**: `docs/architecture.md` — Sections 9 & 11
> **Approach**: TDD (Test-Driven Development), incremental delivery, continuous integration

---

## Table of Contents

1. [Implementation Strategy](#1--implementation-strategy)
2. [Architecture-to-Implementation Traceability Matrix](#2--architecture-to-implementation-traceability-matrix)
3. [Phase 1: Foundation (Weeks 1-3)](#3--phase-1-foundation-weeks-1-3)
4. [Phase 2: Knowledge & Retrieval (Weeks 4-6)](#4--phase-2-knowledge--retrieval-weeks-4-6)
5. [Phase 3: AI Layer (Weeks 7-10)](#5--phase-3-ai-layer-weeks-7-10)
6. [Phase 4: Integration & Polish (Weeks 11-13)](#6--phase-4-integration--polish-weeks-11-13)
7. [Testing Strategy](#7--testing-strategy)
8. [Risk Mitigation](#8--risk-mitigation)
9. [Success Criteria](#9--success-criteria)
10. [Post-MVP Roadmap](#10--post-mvp-roadmap)

---

## 1 — Implementation Strategy

### Principles

1. **Test-First**: Write tests before implementation (TDD)
2. **Vertical Slices**: Each task delivers end-to-end value (DB → API → test)
3. **Incremental**: Each phase builds on the previous, no big-bang integration
4. **Quality Gates**: No phase starts until previous phase tests pass
5. **Documentation**: Update architecture doc with "as-built" details

### Development Flow per Task

```
1. Read requirements (FR/NFR from agentic_crm_requirements_agent_ready.md)
2. Write failing test (unit + integration)
3. Implement minimum code to pass test
4. Refactor (if needed)
5. Run full test suite (must pass 100%)
6. Update docs/architecture.md (mark completed FRs)
7. Commit with: "feat(module): description [FR-XXX]"
```

### Tooling Setup

- **Go**: 1.22+ with `go mod`, `go test`, `go generate`
- **SQLite**: modernc.org/sqlite (pure Go, no CGO)
- **sqlc**: Generate type-safe DB code from SQL
- **golangci-lint**: Code quality checks
- **Make**: Task automation (`make test`, `make build`, `make migrate`)
- **Docker**: Dev environment with Ollama
- **Git**: Feature branches, PR reviews, squash merge to main

### Directory Structure (Initial)

```
fenixcrm/
├── .github/workflows/ci.yml       # CI: test + lint + build
├── cmd/fenixcrm/
│   └── main.go                    # Entry point
├── internal/                      # Private application code
│   ├── config/                    # Configuration loading
│   ├── server/                    # HTTP server setup
│   └── version/                   # Version info
├── api/                           # HTTP layer
│   ├── handlers/                  # Route handlers
│   ├── middleware/                # Auth, logging, etc.
│   └── routes.go
├── domain/                        # Business logic
│   ├── crm/
│   ├── knowledge/
│   ├── copilot/
│   ├── agent/
│   ├── policy/
│   ├── tool/
│   ├── audit/
│   └── eval/
├── infra/                         # Infrastructure adapters
│   ├── sqlite/
│   │   ├── migrations/           # SQL migration files
│   │   ├── queries/              # SQL queries for sqlc
│   │   └── gen/                  # Generated code (sqlc)
│   ├── cache/
│   ├── eventbus/
│   ├── llm/
│   └── otel/
├── pkg/                           # Shared libraries (can be exported)
│   ├── uuid/                     # UUID v7 generation
│   ├── validator/                # Input validation
│   └── errors/                   # Error types
├── tests/
│   ├── integration/              # Integration tests
│   ├── e2e/                      # End-to-end tests
│   └── fixtures/                 # Test data
├── docs/
│   ├── architecture.md
│   └── implementation-plan.md    # THIS DOCUMENT
├── Makefile
├── go.mod
├── go.sum
├── sqlc.yaml
├── .golangci.yml
├── CLAUDE.md
└── README.md
```

---

## 2 — Architecture-to-Implementation Traceability Matrix

> **Purpose**: Ensure every architecture component has explicit implementation coverage.
> **Status**: Living document — update as tasks complete.

| Architecture Component | ERD Entity | Implementation Status | Phase | Task | Notes |
|------------------------|------------|----------------------|-------|------|-------|
| **Tenant & Auth** |
| Workspace | `workspace` | ✅ Completed | 1 | 1.2 | Migration 001 |
| User Account | `user_account` | ✅ Completed | 1 | 1.2, 1.6 | Auth + JWT |
| Role | `role` | ✅ Completed | 1 | 1.2 | RBAC foundation |
| User Role | `user_role` | ✅ Completed | 1 | 1.2 | Role assignments |
| Policy Set | `policy_set` | ⚠️ Partial | 3 | 3.1 | Engine in Phase 3 |
| **CRM Core** |
| Account | `account` | ✅ Completed | 1 | 1.3 | CRUD + API + HTTP handlers (14 tests passing) |
| Contact | `contact` | ✅ Completed | 1 | 1.4 | CRUD + API |
| Lead | `lead` | ✅ Completed | 1 | 1.5 | CRUD + API |
| Deal | `deal` | ✅ Completed | 1 | 1.5 | CRUD + API |
| Case Ticket | `case_ticket` | ✅ Completed | 1 | 1.5 | CRUD + API |
| Pipeline | `pipeline` | ✅ Completed | 1 | 1.5 | Stage management |
| Pipeline Stage | `pipeline_stage` | ✅ Completed | 1 | 1.5 | Stage transitions |
| Activity | `activity` | ✅ Completed | 1 | 1.5 (expanded) | **CORRECTED** |
| Note | `note` | ✅ Completed | 1 | 1.5 (expanded) | **CORRECTED** |
| Attachment | `attachment` | ✅ Completed | 1 | 1.5 (expanded) | **CORRECTED** |
| Timeline Event | `timeline_event` | ✅ Completed | 1 | 1.5 (expanded) | **CORRECTED** |
| **Knowledge & Retrieval** |
| Knowledge Item | `knowledge_item` | ⚠️ Partial | 2 | 2.1, 2.2 | + FTS5 sync |
| Embedding Document | `embedding_document` | ⚠️ Partial | 2 | 2.1, 2.4 | + sqlite-vec |
| Evidence | `evidence` | ⚠️ Partial | 2 | 2.6 | Evidence pack |
| **Agent & Tools** |
| Agent Definition | `agent_definition` | ❌ Pending | 3 | 3.7 | Orchestrator |
| Skill Definition | `skill_definition` | 🔵 Out of scope (P1) | - | - | Not in P0 MVP |
| Tool Definition | `tool_definition` | ❌ Pending | 3 | 3.3 | Registry |
| Agent Run | `agent_run` | ❌ Pending | 3 | 3.7 | State machine |
| Approval Request | `approval_request` | ❌ Pending | 3 | 3.2 | Workflows |
| **Audit** |
| Audit Event | `audit_event` | ✅ Completed | 1 | 1.7 (new) | **CORRECTED: moved from Phase 4** |
| **Prompt & Eval** |
| Prompt Version | `prompt_version` | ❌ Pending | 3 | 3.9 (new) | **CORRECTED: added explicit task** |
| Policy Version | `policy_version` | ❌ Pending | 3 | 3.1 | With policy engine |
| Eval Suite | `eval_suite` | ❌ Pending | 4 | 4.6 | Basic only |
| Eval Run | `eval_run` | ❌ Pending | 4 | 4.6 | Basic only |

### Critical Corrections Applied

1. **✅ Audit Event (Task 1.7)**: Moved from Week 13 to Week 3 — audit must work from Phase 1
2. **✅ Activity/Note/Attachment/Timeline (Task 1.5)**: Expanded to include all supporting entities — tools depend on these
3. **✅ Prompt Versioning (Task 3.9)**: Added explicit task — architecture requires it for agent runtime
4. **⚠️ CDC/Reindex (Task 2.7)**: Added explicit task for Change Data Capture flow
5. **⚠️ Multi-tenant Vector Search**: Security fix in Task 2.1 for `workspace_id` filtering

### Legend

- ✅ **Completed**: Has migration + service + API + tests
- ⚠️ **Partial**: Schema exists but incomplete implementation
- ❌ **Pending**: Not yet started
- 🔵 **Out of scope**: Formally moved to P1/P2

---

## 3 — Phase 1: Foundation (Weeks 1-3)

**Goal**: Operational CRM with CRUD APIs, authentication, and basic observability.

**Deliverable**: A working REST API that can create/read/update/delete CRM entities with JWT auth and audit logging.

### Week 1: Project Scaffolding + Database

#### Task 1.1: Project Setup (2 days) ✅ COMPLETED

**Status**: ✅ Done — 2026-02-10
**Module**: `github.com/matiasleandrokruk/fenix` (adjusted from plan)

**Actions**:
- [x] Initialize Go module: `go mod init github.com/matiasleandrokruk/fenix`
- [x] Setup directory structure with `internal/` (ADR-001 Option B)
- [x] Create `Makefile` with targets: `test`, `build`, `run`, `migrate`, `lint`
- [x] Setup CI workflow (GitHub Actions): run tests + linter on PR
- [x] Create `README.md` with setup instructions
- [x] Implement `internal/version` package with 100% test coverage
- [x] Implement `cmd/fenix/main.go` entry point

**Tests**:
- [x] CI pipeline runs successfully
- [x] `make build` produces `./fenix` binary
- [x] `./fenix --version` displays version
- [x] `go test` passes with coverage reporting

**Resolves**: Infrastructure setup

**Files Created**:
- `go.mod`, `Makefile`, `README.md`
- `.github/workflows/ci.yml`
- `cmd/fenix/main.go`
- `internal/version/version.go` + `version_test.go`
- Full directory structure per ADR-001

---

#### Task 1.2: SQLite Setup + Migrations (3 days)

**Status**: 🟡 **IN PROGRESS** — INC-001 resuelta, continuar desde sub-tarea 1.2.2

**Incidencia resuelta**:
- **ID**: INC-001 ✅ **RESUELTA** — 2026-02-10
- **Descripción**: Versión de Go del sistema (1.18.1) incompatible con dependencias requeridas
- **Causa raíz**: `modernc.org/sqlite` → `golang.org/x/exp/constraints` → requiere paquete `cmp` (stdlib desde Go 1.21). El symlink de brew apuntaba a Go 1.18.1.
- **Resolución aplicada**: `brew install go@1.22 && brew link go@1.22 --force`. `modernc.org/sqlite v1.45.0` requiere Go 1.24 — Go toolchain management descargó automáticamente `go1.24.13`.
- **Estado post-resolución**: `go test ./...` pasa ✅. `go.mod` actualizado a `go 1.24.0` + `toolchain go1.24.13`.
- **Nota**: Plan decía Go 1.22+ como mínimo. La versión efectiva del proyecto es **Go 1.24** por requerimiento transitivo de `modernc.org/sqlite v1.45.0`.

**Sub-tareas desglosadas**:
| # | Sub-tarea | Estado | Notas |
|---|-----------|--------|-------|
| 1.2.1 | Add SQLite and sqlc dependencies | ✅ **COMPLETADA** | INC-001 resuelta — modernc.org/sqlite v1.45.0 + sqlc v1.30.0 |
| 1.2.2 | Create sqlc.yaml configuration | ✅ **COMPLETADA** | internal/infra/sqlite/ paths, ADR-001 aligned |
| 1.2.3 | Write tests for database connection | ✅ **COMPLETADA** | 9 tests: WAL, FK, busy_timeout, pool, in-memory, file creation |
| 1.2.4 | Implement database connection (Open/Close) | ✅ **COMPLETADA** | internal/infra/sqlite/db.go — WAL+FK+timeout via DSN PRAGMAs |
| 1.2.5 | Create migration system | ✅ **COMPLETADA** | internal/infra/sqlite/migrate.go — embed.FS, idempotent |
| 1.2.6 | Write migration 001_init_schema | ✅ **COMPLETADA** | workspace, user_account, role, user_role + indexes |
| 1.2.7 | Write SQL queries for sqlc | ✅ **COMPLETADA** | workspace.sql + user.sql + role.sql — sqlc generate ok (1008 líneas generadas) |
| 1.2.8 | Write integration tests for migrations | ✅ **COMPLETADA** | 13 tests: FK, UNIQUE, table existence, idempotency |
| 1.2.9 | Update Makefile with db commands | ✅ **COMPLETADA** | migrate-version, db-shell agregados |
| 1.2.10 | Run all tests and verify | ✅ **COMPLETADA** | 22 tests pasan, cobertura 75.7% (sqlite pkg) |

**Actions**:
- Install dependencies: `modernc.org/sqlite`, `github.com/sqlc-dev/sqlc`
- Create `sqlc.yaml` configuration
- Create migration system (use `golang-migrate` or simple version table)
- Write migration `001_init_schema.up.sql`:
  - Create `workspace` table
  - Create `user_account` table
  - Create `role` table
  - Create `user_role` table
  - Add indexes on FKs
- Write migration `001_init_schema.down.sql` (rollback)
- Implement `infra/sqlite/db.go`:
  - `Open(path string) (*sql.DB, error)` — with WAL mode
  - `Migrate(db *sql.DB) error` — run pending migrations
  - `Close(db *sql.DB) error`

**Tests**:
- Unit test: Open DB, run migrations, verify schema exists
- Unit test: Rollback migrations, verify clean state
- Integration test: Insert/select from `workspace` table

**Resolves**: Database foundation

---

### Week 2: CRM Entities (Accounts, Contacts)

#### Task 1.3: Account Entity (3 days) ✅ **COMPLETED**

**Status**: ✅ Done — 2026-02-10

**Sub-tareas desglosadas**:
| # | Sub-tarea | Estado | Notas |
|---|-----------|--------|-------|
| 1.3.1 | Create migration 002_crm_accounts.up.sql | ✅ **COMPLETADA** | account table + UNIQUE (workspace_id, name) + soft delete indexes |
| 1.3.2 | Write SQL queries account.sql | ✅ **COMPLETADA** | 8 queries: Create, GetByID, ListByWorkspace, ListByOwner, Update, SoftDelete, Count |
| 1.3.3 | Run sqlc generate | ✅ **COMPLETADA** | internal/infra/sqlite/sqlcgen/account.go — type-safe generated code |
| 1.3.4 | Write TDD tests for AccountService | ✅ **COMPLETADA** | 8 tests: Create, Get, GetNotFound, List, ListExcludesDeleted, Update, Delete, ListByOwner |
| 1.3.5 | Implement domain/crm/account.go | ✅ **COMPLETADA** | AccountService with rowToAccount() mapper, nullString() helper, UUID v7 generation |
| 1.3.6 | Write TDD tests for HTTP handlers | ✅ **COMPLETADA** | 6 handler tests: CreateAccount, GetAccount, GetNotFound, ListAccounts, UpdateAccount, DeleteAccount |
| 1.3.7 | Implement internal/api/handlers/account.go | ✅ **COMPLETADA** | 5 CRUD endpoints + multi-tenancy isolation via context |
| 1.3.8 | Register routes + middleware setup | ✅ **COMPLETADA** | NewRouter(), WorkspaceMiddleware(), account endpoints registered |
| 1.3.9 | Create server initialization | ✅ **COMPLETADA** | internal/server/server.go — HTTP server with graceful shutdown |

**Test Results**:
- ✅ 8/8 AccountService tests pass (86.7% coverage)
- ✅ 6/6 AccountHandler tests pass (65.8% coverage)
- ✅ Total: 14 tests, all passing, ~72.8% combined coverage

**Actions** (✅ ALL COMPLETED):
- [x] Create migration `002_crm_accounts.up.sql`:
  - [x] `account` table (all fields from ERD)
  - [x] Indexes: `workspace_id`, `owner_id`, `deleted_at`
  - [x] UNIQUE constraint on (workspace_id, name) for active accounts
- [x] Write SQL queries in `infra/sqlite/queries/account.sql`:
  - [x] `-- name: CreateAccount :exec`
  - [x] `-- name: GetAccountByID :one`
  - [x] `-- name: ListAccountsByWorkspace :many` (with pagination)
  - [x] `-- name: ListAccountsByOwner :many`
  - [x] `-- name: UpdateAccount :exec`
  - [x] `-- name: SoftDeleteAccount :exec`
  - [x] `-- name: CountAccountsByWorkspace :one`
- [x] Run `sqlc generate` to produce Go code in `internal/infra/sqlite/sqlcgen/`
- [x] Implement `internal/domain/crm/account.go`:
  - [x] `type Account struct` (domain model with pointers for nullable fields)
  - [x] `type AccountService struct { db *sql.DB, querier sqlcgen.Querier }`
  - [x] `Create(ctx, CreateAccountInput) (*Account, error)` — generates UUID v7, calls Get()
  - [x] `Get(ctx, workspaceID, accountID string) (*Account, error)` — excludes soft-deleted
  - [x] `List(ctx, workspaceID, ListAccountsInput) ([]*Account, int, error)` — pagination + count
  - [x] `ListByOwner(ctx, workspaceID, ownerID string) ([]*Account, error)`
  - [x] `Update(ctx, workspaceID, accountID string, UpdateAccountInput) (*Account, error)` — calls Get()
  - [x] `Delete(ctx, workspaceID, accountID string) error` — soft delete with timestamp
- [x] Implement `internal/api/handlers/account.go`:
  - [x] `POST /api/v1/accounts` → `CreateAccount` (201 Created)
  - [x] `GET /api/v1/accounts?limit=N&offset=M` → `ListAccounts` (200 + pagination meta)
  - [x] `GET /api/v1/accounts/{id}` → `GetAccount` (200 or 404)
  - [x] `PUT /api/v1/accounts/{id}` → `UpdateAccount` (200)
  - [x] `DELETE /api/v1/accounts/{id}` → `DeleteAccount` (204 No Content)
- [x] Setup routing:
  - [x] Create `internal/api/routes.go` — NewRouter() with chi + middleware
  - [x] Create `internal/api/context.go` — shared context key helpers
  - [x] Create `internal/api/errors.go` — API error definitions
  - [x] Create `internal/api/handlers/helpers.go` — handler helpers (getWorkspaceID)
- [x] Create HTTP server:
  - [x] `internal/server/server.go` — Server struct + Start() + Shutdown()

**Files Created/Modified**:
- ✅ `internal/infra/sqlite/migrations/002_crm_accounts.up.sql` (41 lines)
- ✅ `internal/infra/sqlite/queries/account.sql` (59 lines)
- ✅ `internal/domain/crm/account.go` (243 lines)
- ✅ `internal/domain/crm/account_test.go` (336 lines)
- ✅ `internal/api/handlers/account.go` (340 lines)
- ✅ `internal/api/handlers/account_test.go` (341 lines)
- ✅ `internal/api/handlers/helpers.go` (16 lines)
- ✅ `internal/api/routes.go` (61 lines)
- ✅ `internal/api/context.go` (20 lines)
- ✅ `internal/api/errors.go` (6 lines)
- ✅ `internal/server/server.go` (60 lines)
- ✅ `pkg/uuid/uuid.go` (64 lines)

**Architectural Patterns Applied**:
1. **Multi-Tenancy Isolation**: Every query includes `workspace_id = ?` filter to prevent cross-tenant data leaks
2. **Soft Deletes**: Using `deleted_at IS NULL` filter instead of hard deletes for audit trail
3. **Service Pattern**: Service layer wraps sqlc Querier interface + adds business logic
4. **UUID v7**: Sortable by timestamp (better for database indexes than random v4)
5. **Pointer Types**: For nullable database columns (Domain, Industry, etc.)
6. **Context Injection**: Workspace ID passed via context (later: JWT claims in 1.6)
7. **In-Memory Testing**: Tests use real SQLite with migrations (no mocks)

**Resolves**: FR-001 (Account CRUD), FR-070 (basic tenant isolation)

---

#### Task 1.4: Contact Entity (2 days) ✅ **COMPLETED**

**Status**: ✅ Done — 2026-02-10

**Actions** (✅ ALL COMPLETED):
- [x] Create migration `003_crm_contacts.up.sql`:
  - [x] `contact` table
  - [x] FK to `account`, `owner_id`
  - [x] Indexes
- [x] Write SQL queries in `internal/infra/sqlite/queries/contact.sql`
- [x] Run `sqlc generate`
- [x] Implement `internal/domain/crm/contact.go` (same pattern as Account)
- [x] Implement handlers:
  - [x] `POST /api/v1/contacts`
  - [x] `GET /api/v1/contacts`
  - [x] `GET /api/v1/contacts/{id}`
  - [x] `PUT /api/v1/contacts/{id}`
  - [x] `DELETE /api/v1/contacts/{id}`
  - [x] `GET /api/v1/accounts/{account_id}/contacts` (filter by account)
- [x] Register routes in `internal/api/routes.go`

**Tests**:
- [x] Service tests (ContactService CRUD + soft delete)
- [x] Handler tests (CRUD + list by account_id)
- [x] Go test verification:
  - `go test ./internal/domain/crm ./internal/api/handlers ./internal/api ./internal/infra/sqlite`

**Resolves**: FR-001 (partial — Contact CRUD)

---

### Week 3: Lead, Deal, Case + Supporting Entities + Auth

#### Task 1.5: Lead, Deal, Case + Supporting Entities (4 days — **EXPANDED**)

**Status**: ✅ **COMPLETED** — 2026-02-10

**Evidencia de cierre (as-built):**
- Handlers implementados y cableados en router para: `lead`, `deal`, `case`, `pipeline` + `pipeline_stage`, `activity`, `note`, `attachment`, `timeline`.
- Rutas registradas en `internal/api/routes.go` para todos los recursos de Task 1.5.
- Timeline automático integrado en servicios core (`lead`, `deal`, `case`, `activity`, `note`, `attachment`) mediante creación de `timeline_event` en operaciones create/update/delete según corresponda.
- Validación técnica en verde: `go test ./...`.

**Actions**:
- Create migrations:
  - `004_crm_leads.up.sql`
  - `005_crm_deals.up.sql`
  - `006_crm_cases.up.sql`
  - `007_crm_pipelines.up.sql` (pipeline + pipeline_stage)
  - **NEW**: `008_crm_supporting.up.sql` (activity, note, attachment, timeline_event)
- Write SQL queries for each entity
- Run `sqlc generate`
- Implement domain services: `lead.go`, `deal.go`, `case.go`, `pipeline.go`
- **NEW**: Implement supporting services: `activity.go`, `note.go`, `attachment.go`, `timeline.go`
- Implement handlers (same CRUD pattern for all entities)
- **NEW**: Connect timeline auto-recording on entity changes (via event bus stub)

**Tests**:
- Unit + integration + API tests (same pattern)
- Test FK constraints (deal → account, stage)
- Test pipeline stage transitions
- **NEW**: Test activity polymorphic FK (entity_type + entity_id)
- **NEW**: Test timeline event auto-generated on create/update
- **NEW**: Test attachment upload + storage path

**Resolves**: FR-001 (Lead, Deal, Case, Activity CRUD), FR-002 (Pipeline basics), FR-051 (Timeline partial)

**Rationale**: These entities are direct dependencies for tools (`create_task` → `activity`, `send_reply` → `note`) and handoff (requires `timeline_event` for context). Moving them to Phase 1 unblocks Phase 3 tool implementation.

---

#### Task 1.6: Authentication Middleware (1 day — **REDUCED**)

**Actions**:
- Create migration `008_auth.up.sql`:
  - Update `user_account` table with `password_hash` field
- Implement `pkg/auth/`:
  - `HashPassword(password string) (string, error)` (bcrypt)
  - `VerifyPassword(hash, password string) bool`
  - `GenerateJWT(userID, workspaceID string) (string, error)`
  - `ParseJWT(token string) (*Claims, error)`
- Implement `api/middleware/auth.go`:
  - `AuthMiddleware(next http.Handler) http.Handler`
  - Extract JWT from `Authorization: Bearer <token>`
  - Validate, extract claims (user_id, workspace_id)
  - Store in `context.Context`
- Implement handlers:
  - `POST /api/v1/auth/login` (email + password → JWT)
  - `POST /api/v1/auth/register` (MVP: create user + workspace)

**Tests**:
- Unit test: Hash + verify password
- Unit test: Generate + parse JWT
- Integration test: Login with valid credentials → JWT
- Integration test: Access protected endpoint without token → 401
- Integration test: Access with valid token → 200

**Resolves**: FR-060 (basic auth), NFR-030 (authentication)

---

#### Task 1.7: Audit Logging Foundation (1 day — **NEW**)

**Actions**:
- Create migration `009_audit_base.up.sql`:
  - `audit_event` table (append-only, immutable)
  - Fields: id, workspace_id, actor_id, actor_type (user|agent|system), action, entity_type, entity_id, details (JSON), permissions_checked (JSON), outcome (success|denied|error), trace_id, ip_address, created_at
  - Index on: workspace_id, actor_id, entity_type, created_at, outcome
- Implement `domain/audit/service.go`:
  - `type AuditService struct { db *sql.DB }`
  - `Log(ctx context.Context, event AuditEvent) error` — append-only insert
  - No updates, no deletes (immutable log)
- Connect audit logging to critical paths:
  - Auth: login success/failure, token refresh, logout
  - CRM: create/update/delete for all entities
  - Authorization: 401/403 denials (log attempted action + reason)
- Implement middleware: `audit.LogRequest(next http.Handler) http.Handler`
  - Extract actor from JWT claims
  - Log after response (capture outcome)

**Tests**:
- Integration test: Login success → audit event created (action: login, outcome: success)
- Integration test: Login failure → audit event created (outcome: error)
- Integration test: Create account → audit event with old_value=null, new_value={...}
- Integration test: Delete with 403 → audit event (outcome: denied, permissions_checked)
- Integration test: Query audit_event by workspace_id → isolated per tenant

**Resolves**: FR-070 (audit trail — foundation), NFR-031 (traceability from Phase 1)

**Rationale**: **CRITICAL CORRECTION** — Architecture mandates immutable audit trail from inception. Moving audit from Week 13 to Week 3 ensures all Phase 2-4 actions are logged from the start. This is non-negotiable for governed systems where retrospective audit is impossible.

---

### Phase 1 Exit Criteria

✅ All CRM entity CRUD endpoints working (Account, Contact, Lead, Deal, Case, Activity, Note, Attachment)
✅ Timeline events auto-generated on entity changes
✅ JWT authentication active on all `/api/v1/*` routes
✅ **Audit logging functional** (all auth + CRM actions logged to `audit_event`)
✅ 100% test coverage on critical paths
✅ Migrations up/down work cleanly
✅ CI pipeline green
✅ **Multi-tenancy verified** (workspace_id isolation in all queries)

---

## 3 — Phase 2: Knowledge & Retrieval (Weeks 4-6)

**Goal**: Hybrid search (BM25 + vector) with permission filtering and evidence pack assembly.

**Deliverable**: A working `/api/v1/knowledge/search` endpoint that returns ranked, permission-filtered results.

### Week 4: Knowledge Schema + Ingestion

#### Task 2.1: Knowledge Tables (2 days — **CORRECTED for multi-tenancy**)

**Actions**:
- Create migration `010_knowledge.up.sql`:
  - `knowledge_item` table
  - `embedding_document` table (includes `workspace_id` FK)
  - `evidence` table
- Create FTS5 virtual table:
  ```sql
  CREATE VIRTUAL TABLE knowledge_item_fts USING fts5(
    id UNINDEXED,
    workspace_id UNINDEXED,
    title,
    normalized_content,
    tokenize='unicode61'
  );
  ```
- Create sqlite-vec virtual table:
  ```sql
  CREATE VIRTUAL TABLE vec_embedding USING vec0(
    id TEXT PRIMARY KEY,
    embedding FLOAT[1536]
  );
  -- Note: sqlite-vec does NOT support multi-column indexes natively
  -- Multi-tenancy MUST be enforced via JOIN with embedding_document.workspace_id
  ```
- **SECURITY FIX**: Document mandatory query pattern for vector search:
  ```sql
  -- CORRECT (tenant-safe):
  SELECT e.id, e.chunk_text, e.distance
  FROM vec_embedding v
  JOIN embedding_document e ON v.id = e.id
  WHERE e.workspace_id = ?
  AND v.embedding MATCH ?
  ORDER BY v.distance
  LIMIT ?;

  -- WRONG (tenant leak risk):
  SELECT id, distance FROM vec_embedding WHERE embedding MATCH ?;
  ```
- Write SQL queries in `infra/sqlite/queries/knowledge.sql` (all with `workspace_id` filter)
- Run `sqlc generate`

**Tests**:
- Integration test: Insert into `knowledge_item` + FTS5 sync
- Integration test: Query FTS5 with `MATCH` + `workspace_id` filter
- Integration test: Insert into `vec_embedding` + ANN query
- **SECURITY TEST**: Vector search with workspace_id=A NEVER returns docs from workspace_id=B

**Resolves**: Database schema for knowledge + **multi-tenancy security fix**

**Rationale**: **CRITICAL SECURITY CORRECTION** — sqlite-vec has no native tenant filtering. Without explicit JOIN on `embedding_document.workspace_id`, vector queries could leak cross-tenant data. This is a P0 blocker.

---

#### Task 2.2: Ingestion Pipeline (3 days)

**Actions**:
- Implement `domain/knowledge/ingestion.go`:
  - `IngestDocument(ctx, IngestInput) (*KnowledgeItem, error)`
  - Normalize content (strip HTML, lowercase, etc.)
  - Chunk into 512-token segments with 50-token overlap
  - Store in `knowledge_item`
  - Sync to `knowledge_item_fts`
- Implement `domain/knowledge/chunker.go`:
  - `ChunkText(text string, maxTokens int, overlap int) []Chunk`
  - Use simple whitespace tokenizer (or tiktoken for accuracy)
- Implement handler:
  - `POST /api/v1/knowledge/ingest`
  - Body: `{ source_type, title, raw_content, entity_type, entity_id }`
  - Returns: `{ knowledge_item_id, chunks_created }`

**Tests**:
- Unit test: Chunker produces correct number of chunks
- Integration test: Ingest document → verify in DB + FTS5
- API test: POST ingest → 201 + chunks created

**Resolves**: FR-090 (ingestion — text only for MVP)

---

### Week 5: LLM Adapter + Embedding

#### Task 2.3: LLM Provider Interface (2 days)

**Actions**:
- Implement `infra/llm/provider.go`:
  - `type LLMProvider interface` (from architecture.md Section 8)
  - `type ChatRequest struct`
  - `type ChatResponse struct`
  - `type EmbedRequest struct`
  - `type EmbedResponse struct`
- Implement `infra/llm/ollama.go`:
  - `type OllamaProvider struct { baseURL string }`
  - `ChatCompletion(ctx, req) (*ChatResponse, error)`
  - `Embed(ctx, req) (*EmbedResponse, error)` — call `/api/embeddings`
  - `ModelInfo() ModelMeta`
  - `HealthCheck(ctx) error` — ping Ollama
- Implement `infra/llm/router.go`:
  - `type Router struct { providers map[string]LLMProvider }`
  - `Route(ctx, req, policy) (LLMProvider, error)` — select provider
  - For MVP: Always use Ollama (local)

**Tests**:
- Integration test (requires Ollama running):
  - Call `Embed()` → returns vector float[]
  - Call `ChatCompletion()` → returns text response
- Unit test: Router selects Ollama when no-cloud policy active

**Resolves**: LLM adapter foundation

---

#### Task 2.4: Embed & Index (3 days)

**Actions**:
- Implement `domain/knowledge/embedder.go`:
  - `EmbedChunks(ctx, knowledgeItemID) error`
  - For each chunk in `knowledge_item`:
    - Call `llm.Embed(chunk.text)`
    - Store in `embedding_document` table
    - Insert into `vec_embedding` virtual table
- Implement async job: `EmbedKnowledgeItemJob`
  - Triggered after ingestion
  - Retry logic (3 attempts)
- Implement `infra/eventbus/bus.go`:
  - `type Bus struct { subscribers map[string][]chan Event }`
  - `Publish(event Event)`
  - `Subscribe(eventType string) <-chan Event`
- Connect ingestion → event bus → embedder

**Tests**:
- Integration test: Ingest document → embedding job runs → vectors in DB
- Integration test: Query vec_embedding with sample vector → returns nearest neighbors

**Resolves**: FR-092 (vector embeddings)

---

### Week 6: Hybrid Search + Evidence Pack

#### Task 2.5: Hybrid Search (3 days)

**Actions**:
- Implement `domain/knowledge/search.go`:
  - `HybridSearch(ctx, SearchInput) (*SearchResults, error)`
  - Parallel execution:
    - BM25: Query `knowledge_item_fts` with FTS5 `MATCH`, get `bm25()` scores
    - Vector: Embed query → query `vec_embedding` with `MATCH`, get distances
  - Merge results via Reciprocal Rank Fusion (RRF):
    ```go
    for doc := range allDocs {
      rrf[doc] = sum(1 / (k + rank_in_method[doc]))
    }
    ```
  - k = 60
  - Sort by RRF score descending
  - Return top 50 candidates
- Implement handler:
  - `POST /api/v1/knowledge/search`
  - Body: `{ query, workspace_id, limit }`
  - Returns: `{ results: [{ id, snippet, score, method }] }`

**Tests**:
- Integration test: BM25 search for "pricing" → returns relevant docs
- Integration test: Vector search for "pricing" → returns relevant docs
- Integration test: Hybrid search combines both, scores are RRF
- Performance test: Search < 500ms p95

**Resolves**: FR-092 (hybrid search)

---

#### Task 2.6: Evidence Pack Builder (2 days)

**Actions**:
- Implement `domain/knowledge/evidence.go`:
  - `BuildEvidencePack(ctx, query, userID) (*EvidencePack, error)`
  - Call `HybridSearch()`
  - Filter by permissions (stub for now — Phase 3 implements policy)
  - Check freshness (warn if TTL expired)
  - Deduplicate near-duplicates (cosine similarity > 0.95)
  - Select top K (default 10)
  - Calculate confidence: high/medium/low based on top score
  - Return `EvidencePack`:
    ```go
    type EvidencePack struct {
      Sources []Evidence
      Confidence string
      TotalCandidates int
      FilteredCount int
      Warnings []string
    }
    ```

**Tests**:
- Integration test: Build evidence pack → returns top 10 results
- Integration test: Deduplication removes near-duplicates
- Integration test: Confidence = "high" when top score > 0.8

**Resolves**: Evidence pack foundation (full implementation in Phase 3)

---

#### Task 2.7: CDC & Auto-Reindex (1 day — **NEW**)

**Actions**:
- Implement CDC (Change Data Capture) flow:
  - Subscribe to event bus: `record.created`, `record.updated`, `record.deleted`
  - Event payload: `{ entity_type, entity_id, workspace_id, change_type }`
- Implement reindex consumer:
  - `domain/knowledge/reindex.go`:
    - `HandleRecordChange(ctx, event) error`
    - Logic:
      - If entity has linked `knowledge_item` (via `entity_type` + `entity_id`):
        - Refresh `normalized_content` from current entity state
        - Update FTS5 index
        - Re-embed if content changed (queue `EmbedKnowledgeItemJob`)
      - Log reindex event to `audit_event`
- Implement handlers:
  - `POST /api/v1/knowledge/reindex` (manual trigger for workspace)
  - Returns: `{ items_queued, estimated_time }`
- Add SLA tracking:
  - Measure: event timestamp → index refresh timestamp
  - Target: <60s in dev, <10s in prod (future optimization)

**Tests**:
- Integration test: Update `case_ticket.description` → linked knowledge_item refreshed in FTS5
- Integration test: Delete `account` → linked knowledge_item marked as stale (or deleted if policy says so)
- Integration test: Manual reindex → all items queued
- Performance test: 100 updates → all reindexed within SLA

**Resolves**: FR-091 (partial — auto-reindex on CRM changes), NFR (data freshness)

**Rationale**: **CRITICAL ADDITION** — Architecture assumes "changes visible within 60s" but plan had no explicit reindex mechanism. This task closes the gap between CRM updates and knowledge retrieval freshness.

---

### Phase 2 Exit Criteria

✅ Knowledge ingestion working (text only)
✅ Hybrid search returns ranked results
✅ **Multi-tenant vector search verified** (workspace_id isolation)
✅ Evidence pack builder returns top-K with confidence
✅ LLM adapter (Ollama) functional
✅ **CDC/Auto-reindex working** (CRM changes reflected in search within 60s)
✅ 100% test coverage on search path

---

## 4 — Phase 3: AI Layer (Weeks 7-10)

**Goal**: Copilot Q&A, Support Agent (UC-C1), Tool Registry, Policy Engine.

**Deliverable**: End-to-end UC-C1 flow working — user triggers support agent → agent retrieves evidence → generates response → executes tools → updates case.

### Phase 3 Dependency Map (Execution Order)

To avoid sequencing ambiguity, Phase 3 tasks have the following dependency constraints:

- **Task 3.1 (Policy Engine)** is a foundational dependency for:
  - **Task 3.5** (permission filter + PII redaction in Copilot Chat)
  - **Task 3.7** (permission checks + audit hooks in Agent Runtime)
- **Task 3.2 (Approval Workflow)** is required by:
  - **Task 3.7** (approval-required tool calls)
- **Task 3.3 (Tool Registry)** must be completed before:
  - **Task 3.4** (built-in tools registration + validation)
  - **Task 3.7** (tool resolution/validation at runtime)
- **Task 3.4 (Built-in Tools)** is required by:
  - **Task 3.7** (support agent executes `update_case`, `send_reply`, `create_task`)
- **Task 3.9 (Prompt Versioning)** must be integrated before closing:
  - **Task 3.7** (runtime loads active prompt version)

**Recommended sequence with safe parallelism**:
1. Start **3.1 + 3.2** (partial parallel)
2. Execute **3.3 → 3.4**
3. Execute **3.5 + 3.6** (once 3.1 minimum is available)
4. Execute **3.9** before finalizing 3.7
5. Execute **3.7 → 3.8**

**Condensed DAG**:
`3.1 ─┬─> 3.5`
`     └─> 3.7`
`3.2 ─────> 3.7`
`3.3 ──> 3.4 ──> 3.7`
`3.9 ─────> 3.7`
`3.7 ─────> 3.8`

### Week 7: Policy Engine (4 Enforcement Points)

#### Task 3.1: RBAC/ABAC Evaluator (3 days)

**Actions**:
- Create migration `010_policies.up.sql`:
  - `policy_set` table
  - `policy_version` table
- Implement `domain/policy/evaluator.go`:
  - `type PolicyEngine struct { db *sql.DB, cache cache.Cache }`
  - **EP1: Before Retrieval**:
    - `BuildPermissionFilter(ctx, userID) (Filter, error)`
    - Load user roles + ABAC attributes
    - Build WHERE clauses for workspace_id, owner_id, etc.
  - **EP2: Before Prompt**:
    - `RedactPII(ctx, evidence, policy) ([]Evidence, error)`
    - Detect PII: regex (phone, email, SSN)
    - Replace with tokens `[PHONE_1]`, `[EMAIL_2]`
    - Store reverse mapping
  - **EP3: Before Tool Call**:
    - `CheckToolPermission(ctx, userID, toolID) (bool, error)`
    - Load tool.required_permissions
    - Check against user roles
  - **EP4: After Execution**:
    - `LogAuditEvent(ctx, event) error`
    - Append to `audit_event` table

**Tests**:
- Unit test: BuildPermissionFilter returns correct WHERE clauses
- Unit test: RedactPII replaces phone numbers with tokens
- Integration test: User without permission → tool denied
- Integration test: Audit event logged after tool execution

**Resolves**: FR-060, FR-070, FR-071 (policy basics)

---

#### Task 3.2: Approval Workflow (2 days)

**Actions**:
- Create migration `011_approvals.up.sql`:
  - `approval_request` table
- Implement `domain/policy/approval.go`:
  - `CreateApprovalRequest(ctx, input) (*ApprovalRequest, error)`
  - `DecideApprovalRequest(ctx, id, decision, decidedBy) error`
  - `GetPendingApprovals(ctx, userID) ([]*ApprovalRequest, error)`
- Implement handlers:
  - `GET /api/v1/approvals` (pending for current user)
  - `PUT /api/v1/approvals/{id}` (approve/deny)

**Tests**:
- Integration test: Create approval request → status = pending
- Integration test: Approve → status = approved
- Integration test: Deny → status = denied
- Integration test: Expired request → status = expired (TTL check)

**Resolves**: FR-061 (approval workflows)

---

### Week 8: Tool Registry + Built-in Tools

#### Task 3.3: Tool Definition & Registry (2 days)

**Actions**:
- Create migration `012_tools.up.sql`:
  - `tool_definition` table
- Implement `domain/tool/registry.go`:
  - `type ToolRegistry struct { db *sql.DB, executors map[string]ToolExecutor }`
  - `Register(name string, executor ToolExecutor) error`
  - `Get(name string) (ToolExecutor, error)`
  - `ValidateParams(toolName, params) error` — JSON Schema validation
- Implement `domain/tool/executor.go`:
  - `type ToolExecutor interface { Execute(ctx, params) (result, error) }`
- Implement handlers:
  - `GET /api/v1/admin/tools` (list all tools)
  - `POST /api/v1/admin/tools` (register new tool)

**Tests**:
- Unit test: Register tool → retrieve by name
- Unit test: ValidateParams with invalid JSON → error
- Integration test: Get tool from DB → deserialize schema

**Resolves**: Tool registry foundation

---

#### Task 3.4: Built-in Tools (3 days)

**Actions**:
- Implement `domain/tool/builtin/create_task.go`:
  - Input schema: `{ owner_id, title, due_date, entity_type, entity_id }`
  - Execute: Insert into `activity` table
  - Returns: `{ task_id, created_at }`
- Implement `domain/tool/builtin/update_case.go`:
  - Input schema: `{ case_id, status?, priority?, tags? }`
  - Execute: Update `case_ticket` table
  - Emit event: `record.updated`
  - Returns: `{ case_id, updated_at }`
- Implement `domain/tool/builtin/send_reply.go`:
  - Input schema: `{ case_id, body, is_internal }`
  - Execute: Insert into `note` table
  - Returns: `{ note_id, created_at }`
- Register all tools in `ToolRegistry` on startup

**Tests**:
- Integration test: create_task → activity created in DB
- Integration test: update_case → case status updated + event emitted
- Integration test: send_reply → note created

**Resolves**: FR-211 (built-in tools)

---

### Week 9: Copilot Service + SSE Streaming

#### Task 3.5: Copilot Chat (3 days)

**Actions**:
- Implement `domain/copilot/chat.go`:
  - `Chat(ctx, ChatInput) (<-chan StreamChunk, error)`
  - Steps:
    1. Fetch entity context (if entity_type + entity_id provided)
    2. Build evidence pack (call `knowledge.BuildEvidencePack()`)
    3. Apply policy: permission filter + PII redaction
    4. Build prompt:
       - System: "You are FenixCRM Copilot. Always cite sources."
       - Context: entity data + evidence pack
       - User query
    5. Call `llm.ChatCompletionStream()`
    6. Stream chunks back to caller
    7. Post-generation: PII leak check
    8. Log audit event
- Implement handler:
  - `POST /api/v1/copilot/chat` (SSE response)
  - Set headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`
  - Stream format:
    ```
    data: {"type": "token", "delta": "Hello"}

    data: {"type": "token", "delta": " there"}

    data: {"type": "evidence", "sources": [...]}

    data: {"type": "done"}
    ```

**Tests**:
- Integration test (with Ollama): Chat → SSE stream received
- Integration test: Evidence pack included in prompt
- Integration test: PII redacted before LLM call
- Integration test: Audit event logged

**Resolves**: FR-200, FR-201, FR-202 (Copilot Q&A)

---

#### Task 3.6: Copilot Actions (2 days)

**Actions**:
- Implement `domain/copilot/suggest_actions.go`:
  - `SuggestActions(ctx, entity_type, entity_id) ([]SuggestedAction, error)`
  - Build evidence pack for entity
  - Prompt: "Suggest 3 actionable next steps"
  - Parse LLM response → extract actions
  - Return: `[{ title, description, tool, params }]`
- Implement handlers:
  - `POST /api/v1/copilot/suggest-actions`
  - `POST /api/v1/copilot/summarize` (summarize entity history)

**Tests**:
- Integration test: Suggest actions for case → returns 3 suggestions
- Integration test: Summarize case → returns summary

**Resolves**: FR-201 (suggested actions), FR-202 (summaries)

---

### Week 10: Agent Orchestrator + UC-C1

#### Task 3.7: Agent Runtime (3 days)

**Actions**:
- Create migration `013_agents.up.sql`:
  - `agent_definition` table
  - `skill_definition` table
  - `agent_run` table
- Implement `domain/agent/orchestrator.go`:
  - `TriggerAgent(ctx, agentID, input) (*AgentRun, error)`
  - State machine:
    1. Create `agent_run` (status: running)
    2. Fetch context (case + account + contact + activities)
    3. Build evidence pack
    4. Check confidence → abstain if low
    5. Call LLM with tools enabled
    6. Parse tool calls from response
    7. For each tool call:
       - Validate via `ToolRegistry`
       - Check permissions via `PolicyEngine`
       - Check if approval required → create `ApprovalRequest` + wait
       - Check idempotency (cache)
       - Execute tool
       - Log audit event
    8. Update `agent_run` (status: success/failed/escalated)
    9. Emit event: `agent.completed`
- Implement `domain/agent/agents/support.go`:
  - UC-C1 Support Agent
  - Objective: Resolve customer support cases
  - Allowed tools: `update_case`, `send_reply`, `create_task`

**Tests**:
- Integration test: Trigger agent → agent_run created
- Integration test: Evidence insufficient → status = abstained
- Integration test: Tool call executed → case updated
- Integration test: Approval required → approval_request created + agent waits
- **E2E test: UC-C1 complete flow** (case → agent → evidence → LLM → tools → case resolved)

**Resolves**: FR-230, FR-231 (support agent), FR-232 (handoff partial)

---

#### Task 3.8: Handoff Manager (2 days)

**Actions**:
- Implement `domain/agent/handoff.go`:
  - `InitiateHandoff(ctx, agentRunID, reason) error`
  - Build handoff package:
    - Load agent_run (evidence, reasoning_trace, tool_calls)
    - Load case + conversation history
    - Determine routing (policy-based)
  - Update case: status = escalated, assigned_to = human_id
  - Emit event: `agent.handoff`
- Implement handlers:
  - `GET /api/v1/agents/runs/{id}/handoff` (get handoff package)

**Tests**:
- Integration test: Initiate handoff → case status = escalated
- Integration test: Handoff package contains all context

**Resolves**: FR-232 (human handoff)

---

#### Task 3.9: Prompt Versioning (1 day — **NEW**)

**Actions**:
- Create migration `015_prompt_versioning.up.sql`:
  - `prompt_version` table (if not exists from earlier migration)
  - Fields: id, workspace_id, agent_definition_id, version_number, system_prompt, user_prompt_template, config (JSON: temperature, max_tokens, etc.), status (draft|testing|active|archived), created_by, created_at
  - Index: agent_definition_id, status
- Implement `domain/agent/prompt.go`:
  - `CreatePromptVersion(ctx, input) (*PromptVersion, error)`
  - `GetActivePrompt(ctx, agentID) (*PromptVersion, error)`
  - `PromotePrompt(ctx, promptVersionID) error` — set status=active, deactivate previous
  - `RollbackPrompt(ctx, agentID) error` — reactivate previous version
- Implement handlers:
  - `GET /api/v1/admin/prompts?agent_id={id}` (list versions)
  - `POST /api/v1/admin/prompts` (create new version)
  - `PUT /api/v1/admin/prompts/{id}/promote` (activate)
  - `PUT /api/v1/admin/prompts/{id}/rollback` (revert to previous)
- Integrate with Agent Orchestrator:
  - `TriggerAgent()` loads `GetActivePrompt()` for agent
  - Uses `system_prompt` + `user_prompt_template` + `config`

**Tests**:
- Integration test: Create prompt version → stored with status=draft
- Integration test: Promote prompt → status=active, previous version archived
- Integration test: Rollback → previous version reactivated
- Integration test: Trigger agent → uses active prompt version
- Integration test: Multiple agents → each has independent prompt versions

**Resolves**: FR-240 (partial — prompt versioning foundation), NFR (change management)

**Rationale**: **CRITICAL ADDITION** — Architecture shows `agent_definition.active_prompt_version_id` FK but implementation plan had no task for this. Prompt versioning is essential for eval-gated releases and rollback capability. This task provides minimum viable versioning for P0; full eval-gating is P1.

**Decision**: ✅ **APPROVED — Keep in P0** (2026-02-09). Prompt versioning remains in P0 as minimum viable architecture requirement.

---

### Phase 3 Exit Criteria

✅ Copilot chat working with SSE streaming
✅ UC-C1 Support Agent end-to-end functional
✅ Tool execution with permissions + approvals + idempotency
✅ Policy engine 4 enforcement points active
✅ Handoff to human working
✅ **Prompt versioning functional** (create, promote, rollback)

---

## 5 — Phase 4: Integration & Polish (Weeks 11-13)

**Goal**: React frontend, observability, audit service, eval service, E2E tests.

**Deliverable**: Full MVP ready for demo — UI + backend fully integrated.

### Week 11: React Frontend MVP

#### Task 4.1: Frontend Setup (2 days)

**Actions**:
- Initialize React project:
  - `npm create vite@latest web -- --template react-ts`
- Install dependencies:
  - `@tanstack/react-query`, `zustand`, `react-router-dom`
  - `shadcn/ui` components, `tailwindcss`
- Setup dev server: `vite` proxies `/api/*` to Go backend
- Create layout: Sidebar + Header + Content
- Implement auth: Login page → store JWT in localStorage
- Implement router:
  - `/login`
  - `/accounts`, `/contacts`, `/deals`, `/cases`
  - `/copilot`
  - `/agents/runs`

**Tests**:
- E2E test (Playwright): Login → redirects to `/accounts`

**Resolves**: Frontend foundation

---

#### Task 4.2: CRM Pages (3 days)

**Actions**:
- Implement pages:
  - `/accounts` — Table with search, pagination, create button
  - `/accounts/:id` — Detail view + timeline + copilot panel
  - Same for `/contacts`, `/deals`, `/cases`
- Implement forms:
  - Create/Edit account modal
  - Form validation (required fields)
- Implement timeline component:
  - Fetch `/api/v1/accounts/:id/timeline`
  - Display events: created, updated, note added, agent action

**Tests**:
- E2E test: Create account → appears in list
- E2E test: Edit account → changes saved
- E2E test: Timeline shows events

**Resolves**: CRM UI

---

### Week 12: Copilot Panel + Agent Runs UI

#### Task 4.3: Copilot Panel (2 days)

**Actions**:
- Implement `CopilotPanel` component:
  - Chat interface (input + message list)
  - SSE connection to `/api/v1/copilot/chat`
  - Display streaming response with citation markers
  - Expandable evidence cards (click `[1]` → show source snippet)
- Implement suggested actions:
  - Display action cards below chat
  - Click action → execute tool (with confirmation)

**Tests**:
- E2E test: Ask question → response streams in
- E2E test: Click citation → evidence card expands
- E2E test: Click suggested action → tool executes

**Resolves**: FR-200, FR-201 (Copilot UI)

---

#### Task 4.4: Agent Runs Dashboard (3 days)

**Actions**:
- Implement `/agents/runs` page:
  - Table: agent name, status, started_at, latency, cost
  - Filters: status, agent_type, date range
- Implement `/agents/runs/:id` detail page:
  - Show: inputs, retrieval queries, evidence retrieved
  - Reasoning trace (expandable)
  - Tool calls (params + results)
  - Output
  - Audit events
- Implement trigger button: "Run Agent"
  - Select agent, select entity → trigger
  - Show progress (status updates)

**Tests**:
- E2E test: Trigger agent → run appears in dashboard
- E2E test: View run detail → all sections visible

**Resolves**: Agent observability UI

---

### Week 13: Audit, Eval, Final Polish

#### Task 4.5: Audit Service — Advanced Features (1.5 days — **UPDATED**)

**Actions**:
- **NOTE**: `audit_event` table and basic logging already exist from Task 1.7 (Phase 1)
- Extend `domain/audit/service.go` with advanced features:
  - `Query(ctx, QueryInput) ([]*AuditEvent, error)` (complex filters, pagination)
    - Filters: date range, actor_id, entity_type, action, outcome
    - Full-text search in `details` JSON field
  - `Export(ctx, ExportInput) (io.Reader, error)` (CSV/JSON export)
    - Format options: CSV (flat), JSON (full), NDJSON (streaming)
    - Filters applied before export
- Complete event bus integration:
  - Subscribe to ALL event types (not just CRM + auth)
  - Agent events: `agent.started`, `agent.completed`, `agent.abstained`, `agent.escalated`
  - Tool events: `tool.executed`, `tool.failed`
  - Policy events: `policy.violated`, `approval.requested`, `approval.decided`
- Implement handlers:
  - `GET /api/v1/audit/events` (query + complex filters)
  - `GET /api/v1/audit/events/{id}` (get single event with full context)
  - `POST /api/v1/audit/export` (download CSV/JSON)

**Tests**:
- Integration test: CRM action → audit event logged (already covered in Phase 1)
- Integration test: Query with filters → returns correct subset
- Integration test: Full-text search in details field → finds events
- Integration test: Export 1000 events → CSV generated correctly
- Integration test: Agent run → all sub-events logged (retrieval, tool calls, outcome)

**Resolves**: FR-070 (audit trail — advanced), FR-071 (audit query + export)

**Rationale**: **UPDATED** — Base audit logging moved to Phase 1 (Task 1.7). Phase 4 focuses on query UI, export, and complete event bus coverage.

---

#### Task 4.6: Eval Service (Basic) (2 days)

**Actions**:
- Create migration `015_eval.up.sql`:
  - `eval_suite` table
  - `eval_run` table
- Implement `domain/eval/suite.go`:
  - `CreateSuite(ctx, input) (*EvalSuite, error)`
  - Suite contains: test cases (input + expected output)
- Implement `domain/eval/runner.go`:
  - `RunEval(ctx, suiteID, promptVersionID) (*EvalRun, error)`
  - For each test case:
    - Call agent with input
    - Compare output vs expected
    - Score: groundedness (has evidence?), exactitude (correct?)
  - Calculate aggregate scores
  - Pass/fail based on thresholds
- Implement handlers:
  - `POST /api/v1/admin/eval/suites` (create suite)
  - `POST /api/v1/admin/eval/run` (run eval)
  - `GET /api/v1/admin/eval/runs` (list results)

**Tests**:
- Integration test: Create eval suite → stored in DB
- Integration test: Run eval → scores calculated

**Resolves**: FR-240 (eval basics — full in P1)

---

#### Task 4.7: E2E Tests + Documentation (1 day)

**Actions**:
- Write E2E test for UC-C1:
  - Login as support agent
  - Navigate to case detail
  - Trigger support agent
  - Verify: evidence retrieved, response generated, case updated
- Write E2E test for Copilot:
  - Open account detail
  - Ask question in copilot panel
  - Verify: response streams, citations clickable
- Update `docs/architecture.md`:
  - Mark all completed FRs with ✅
  - Add "as-built" notes (any deviations from plan)
- Update `README.md`:
  - Installation instructions
  - Quick start guide
  - Screenshot of UI

**Tests**:
- E2E test suite: 100% pass rate

**Resolves**: Documentation + final validation

---

#### Task 4.8: Observability Endpoints (1 day — **NEW**)

**Actions**:
- Implement metrics endpoint:
  - `GET /api/v1/metrics` (Prometheus-compatible format)
  - Metrics exposed:
    - `fenixcrm_http_requests_total{method, path, status}` (counter)
    - `fenixcrm_http_request_duration_seconds{method, path}` (histogram)
    - `fenixcrm_agent_runs_total{agent_type, status}` (counter)
    - `fenixcrm_agent_run_duration_seconds{agent_type}` (histogram)
    - `fenixcrm_agent_run_cost_euros{agent_type}` (histogram)
    - `fenixcrm_agent_run_tokens_total{agent_type}` (counter)
    - `fenixcrm_tool_calls_total{tool_name, outcome}` (counter)
    - `fenixcrm_evidence_retrieval_duration_seconds` (histogram)
- Implement health endpoint:
  - `GET /api/v1/health` (200 if healthy, 503 if degraded)
  - Checks: DB connection, LLM provider health, event bus
- Implement basic dashboard (optional UI or JSON endpoint):
  - `GET /api/v1/admin/dashboard`
  - Returns: last 24h stats (agent runs, costs, latencies)
- Add structured logging:
  - JSON format: `{"level":"info", "timestamp":"...", "message":"...", "trace_id":"...", "actor_id":"...", "workspace_id":"..."}`
  - Log to stdout (12-factor app)

**Tests**:
- Integration test: Call `/metrics` → Prometheus format returned
- Integration test: Trigger agent run → metrics incremented
- Integration test: Call `/health` → 200 if all checks pass
- Integration test: DB down → `/health` returns 503

**Resolves**: NFR-030 (observability), NFR-031 (metrics per agent)

**Rationale**: **CRITICAL ADDITION** — Architecture requires "metrics endpoint, agent run dashboard" but plan had no explicit task. This provides minimal observability for MVP. Full Grafana dashboards are P1.

---

### Phase 4 Exit Criteria

✅ React frontend functional (CRM pages + Copilot + Agent runs)
✅ Audit service logging all events + query + export working
✅ Eval service basic functionality
✅ **Observability endpoints functional** (/metrics, /health, dashboard)
✅ E2E tests passing (100% critical flows)
✅ Documentation updated (architecture.md + README.md)

---

## 6 — Testing Strategy

### Test Pyramid

```
       /\
      /E2E\         ~10 tests (critical flows)
     /------\
    /  Integ \      ~50 tests (API + DB interactions)
   /----------\
  /    Unit    \    ~200 tests (business logic, pure functions)
 /--------------\
```

### Testing Tools

- **Unit tests**: `go test` with table-driven tests
- **Integration tests**: `go test` with real SQLite DB (`:memory:` or temp file)
- **API tests**: `httptest.NewServer()` + real handlers
- **E2E tests**: Playwright (TypeScript) — headless browser automation
- **Mocking**: Minimal (only for external LLM in unit tests)

### Coverage Targets

- **Critical paths**: 100% (auth, policy, tool execution)
- **Business logic**: ≥90%
- **Overall (app-relevant gate scope)**: ≥80%

**As-built update (Task 2.6 coverage hardening):**
- CI enforces 3 coverage gates:
  - `coverage-gate` (global app-relevant)
  - `coverage-app-gate` (app-only filtered profile)
  - `coverage-tdd` (focus gate for TDD-heavy packages)
- Current enforced thresholds in CI/Makefile: **79 / 79 / 79**
- Last green reference (run `21986153777`):
  - Global coverage (gate scope): **80.5%**
  - App coverage: **80.5%**
  - TDD coverage: **79.1%**

### CI Pipeline

```yaml
# .github/workflows/ci.yml
on: [push, pull_request]
jobs:
  complexity:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: make complexity

  test:
    needs: complexity
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: make lint
      - run: make test
      - run: make race-stability
      - run: COVERAGE_MIN=79 make coverage-gate
      - run: COVERAGE_APP_MIN=79 make coverage-app-gate
      - run: TDD_COVERAGE_MIN=79 make coverage-tdd
      - run: make build

  e2e:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      # E2E runs only when tests/e2e project is present.
      # Current implementation skips gracefully if absent.
```

---

## 7 — Risk Mitigation

### Risk 1: sqlite-vec Not Production-Ready

**Likelihood**: Medium
**Impact**: High
**Mitigation**:
- Test thoroughly in Phase 2 (Week 5-6)
- Benchmark performance: 10K vectors, query latency
- Fallback plan: Use PostgreSQL + pgvector if issues arise (architecture supports swap)

---

### Risk 2: LLM Latency Too High (Ollama)

**Likelihood**: Medium
**Impact**: Medium
**Mitigation**:
- Use small model for MVP (e.g., `llama3.2:3b`)
- Optimize prompt length (trim evidence pack to top 5 sources)
- Implement timeout (10s)
- Fallback: Offer cloud LLM option (OpenAI GPT-3.5)

---

### Risk 3: Evidence Pack Quality Low

**Likelihood**: Medium
**Impact**: High
**Mitigation**:
- Start with simple test data (well-structured docs)
- Tune RRF weights (BM25 vs vector)
- Measure groundedness in evals (target >95%)
- Iterate on chunking strategy (512 tokens → 256 if needed)

---

### Risk 4: Scope Creep (User Requests P1 Features)

**Likelihood**: High
**Impact**: Medium
**Mitigation**:
- Clearly communicate P0 scope upfront
- Maintain P1 backlog, commit to timeline
- Defer all non-P0 requests with rationale

---

### Risk 5: Test Coverage Slips

**Likelihood**: Medium
**Impact**: Medium
**Mitigation**:
- Enforce TDD in code reviews (no PR without tests)
- CI fails if coverage < 80%
- Weekly coverage report

---

## 8 — Success Criteria

### Functional Success (P0 Complete)

✅ **FR-001/002**: All CRM entities (Account, Contact, Lead, Deal, Case) CRUD working
✅ **FR-060/070/071**: Auth + RBAC + audit trail active
✅ **FR-090/092**: Hybrid search (BM25 + vector) functional
✅ **FR-200/201/202**: Copilot chat + actions + summaries working
✅ **FR-210/211**: Tool registry + built-in tools functional
✅ **FR-230/231**: Support Agent (UC-C1) working end-to-end
✅ **FR-232**: Handoff to human with context package

### Non-Functional Success (NFR)

✅ **NFR-030/031**: Auth + metrics per agent tracked
✅ **Speed**: Copilot Q&A < 3s p95 (target: 2.5s)
✅ **Reliability**: E2E tests pass 100%
✅ **Security**: No PII leaks in logs/audit
✅ **Deployment**: Single binary runs on Mac/Linux/Docker

### Demo Scenarios

1. **CRM CRUD**: Create account → add contact → create deal → move through pipeline
2. **Copilot Q&A**: Ask "What's the status of Deal X?" → receive answer with citations
3. **Support Agent (UC-C1)**:
   - Create case: "Customer can't login"
   - Trigger support agent
   - Agent retrieves KB articles
   - Agent proposes: update case status, send reply
   - Approve action → case resolved
4. **Audit Trail**: View audit log → see all agent actions + tool calls

---

## 9 — Post-MVP Roadmap

### P1 (v1) — Weeks 14-26 (3 months)

**Focus**: Multi-source ingestion, agent catalog, agent studio, quotas.

**Key deliverables**:
- FR-091: Email connector (IMAP), Google Docs connector, call transcript ingestion
- FR-231: Prospecting agent, KB agent, insights agent
- FR-240/241/242: Prompt versioning UI, skills builder, eval suites
- FR-233, NFR-040/041: Quotas (tokens/day, cost/day), degradation (cheaper model)
- FR-243: Replay/simulation mode

---

### P2 (v2) — Weeks 27-39 (3 months)

**Focus**: Marketplace, scale, enterprise features.

**Key deliverables**:
- FR-052: Plugin SDK + marketplace
- Scale: PostgreSQL + Redis + NATS + Kubernetes
- Enterprise: SSO (OIDC), field-level encryption, multi-region
- Advanced analytics: Cost per outcome (€/ticket, €/deal)

---

## Appendix A: Task Checklist Template

For each task:

```markdown
## Task X.Y: <Name>

**Duration**: X days
**Assigned to**: TBD
**Status**: ❌ Not started | 🟡 In progress | ✅ Done

### Actions
- [ ] Action 1
- [ ] Action 2

### Tests
- [ ] Test 1
- [ ] Test 2

### Resolves
FR-XXX, NFR-XXX

### Notes
(Add deviations, blockers, learnings here)
```

---

## Appendix B: Command Reference

```bash
# Development
make test          # Run all tests
make test-unit     # Unit tests only
make test-integration  # Integration tests
make test-e2e      # E2E tests (requires UI built)
make lint          # Run golangci-lint
make fmt           # Format code (gofmt)
make build         # Build binary → ./fenixcrm
make run           # Run server (dev mode)

# Database
make migrate-up    # Apply pending migrations
make migrate-down  # Rollback last migration
make migrate-create NAME=<name>  # Create new migration
make sqlc-generate # Generate Go code from SQL queries

# Frontend
cd web && npm install   # Install dependencies
cd web && npm run dev   # Start Vite dev server
cd web && npm run build # Build production bundle

# Docker
make docker-build  # Build Docker image
make docker-run    # Run container
docker-compose up  # Start full stack (app + Ollama)
```

---

## Appendix C: Environment Variables

```bash
# .env.example

# Server
PORT=8080
ENV=development  # development | production

# Database
DB_PATH=./data/fenixcrm.db

# Auth
JWT_SECRET=your-secret-key-here
JWT_EXPIRY=24h

# LLM
LLM_PROVIDER=ollama  # ollama | openai | anthropic
OLLAMA_BASE_URL=http://localhost:11434
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...

# Observability
LOG_LEVEL=info  # debug | info | warn | error
OTEL_ENABLED=false
OTEL_ENDPOINT=http://localhost:4318

# Limits
MAX_UPLOAD_SIZE_MB=10
RATE_LIMIT_PER_MINUTE=100
```

---

## 11 — Architecture Decision Record: Project Structure

### ADR-001: Project Directory Structure

**Status**: Proposed (pending team decision)
**Date**: 2026-02-09
**Context**: Divergence between `docs/architecture.md` Appendix (no `internal/`) and `docs/implementation-plan.md` (with `internal/`).

**Decision Options**:

**Option A: Pure Domain-Driven (no `internal/`)**
```
fenixcrm/
├── cmd/fenixcrm/main.go
├── domain/              # Business logic
├── infra/               # Infrastructure adapters
├── api/                 # HTTP handlers
├── pkg/                 # Shared utilities (exportable)
```

**Pros**: Simpler, matches architecture doc, clear domain boundaries
**Cons**: All packages exportable (Go convention: no `internal/` = public API)

---

**Option B: Encapsulated (with `internal/`)**
```
fenixcrm/
├── cmd/fenixcrm/main.go
├── internal/            # Private application code
│   ├── domain/
│   ├── infra/
│   ├── api/
├── pkg/                 # Public shared libraries
```

**Pros**: Explicit encapsulation, prevents external imports, Go convention for apps
**Cons**: Extra nesting level

---

**Recommendation**: **Option B (with `internal/`)**

**Rationale**:
1. FenixCRM is an application, not a library — nothing should import our domain packages
2. Go convention: use `internal/` to prevent accidental external dependencies
3. Clear separation: `internal/` = application code, `pkg/` = reusable utilities
4. Future-proof: easier to extract libraries later

**Action**: Update `docs/architecture.md` Appendix to match this structure.

**Consequences**:
- Import paths: `github.com/yourorg/fenixcrm/internal/domain/crm`
- External tools cannot import our `domain/` or `api/` packages (enforced by Go compiler)

---

## 12 — Corrections Summary

### Changes Applied

1. **✅ Task 1.5 Expanded** (Week 3): Added `activity`, `note`, `attachment`, `timeline_event` — tools depend on these
2. **✅ Task 1.7 New** (Week 3): Audit logging moved from Week 13 — immutable trail from Phase 1
3. **✅ Task 2.1 Security Fix**: Multi-tenant vector search requires JOIN on `workspace_id`
4. **✅ Task 2.7 New** (Week 6): CDC/Auto-reindex explicit — freshness SLA enforcement
5. **✅ Task 3.9 New** (Week 10): Prompt versioning explicit — promote/rollback capability
6. **✅ Task 4.5 Updated**: Audit base moved to Phase 1, Phase 4 = advanced features only
7. **✅ Task 4.8 New** (Week 13): Observability endpoints — /metrics, /health, dashboard
8. **✅ Traceability Matrix Added** (Section 2): Living document for architecture coverage
9. **✅ ADR-001 Added** (Section 11): Directory structure decision (Option B recommended)

### Impacto en Cronograma

- Phase 1: +1 día (Task 1.5 expanded + Task 1.7 new) — mantiene 3 semanas con redistribución
- Phase 2: +1 día (Task 2.7 new) — mantiene 3 semanas
- Phase 3: +1 día (Task 3.9 new) — mantiene 4 semanas
- Phase 4: +1 día (Task 4.8 new) — mantiene 3 semanas

**Total**: Sigue siendo **13 semanas** con redistribución interna. No hay impacto en deadline.

### Decisiones Pendientes

- **Prompt Versioning (Task 3.9)**: ✅ **DECIDIDO — Opción A: Mantener en P0** (2026-02-09)
  - `active_prompt_version_id` FK permanece en ERD
  - Task 3.9 en Week 10 (Phase 3) confirmada
  - Rollback capability disponible en MVP

---

**End of Implementation Plan (Corrected)**

Next step: Review corrections → Accept/adjust → Start Phase 1, Task 1.1 (Project Setup).
