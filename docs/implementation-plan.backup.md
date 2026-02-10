# Implementation Plan — FenixCRM MVP (P0)

> **Status**: Ready for execution
> **Duration**: 13 weeks (3 months)
> **Based on**: `docs/architecture.md` — Sections 9 & 11
> **Approach**: TDD (Test-Driven Development), incremental delivery, continuous integration

---

## Table of Contents

1. [Implementation Strategy](#1--implementation-strategy)
2. [Phase 1: Foundation (Weeks 1-3)](#2--phase-1-foundation-weeks-1-3)
3. [Phase 2: Knowledge & Retrieval (Weeks 4-6)](#3--phase-2-knowledge--retrieval-weeks-4-6)
4. [Phase 3: AI Layer (Weeks 7-10)](#4--phase-3-ai-layer-weeks-7-10)
5. [Phase 4: Integration & Polish (Weeks 11-13)](#5--phase-4-integration--polish-weeks-11-13)
6. [Testing Strategy](#6--testing-strategy)
7. [Risk Mitigation](#7--risk-mitigation)
8. [Success Criteria](#8--success-criteria)
9. [Post-MVP Roadmap](#9--post-mvp-roadmap)

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

## 2 — Phase 1: Foundation (Weeks 1-3)

**Goal**: Operational CRM with CRUD APIs, authentication, and basic observability.

**Deliverable**: A working REST API that can create/read/update/delete CRM entities with JWT auth and audit logging.

### Week 1: Project Scaffolding + Database

#### Task 1.1: Project Setup (2 days)

**Actions**:
- Initialize Go module: `go mod init github.com/yourorg/fenixcrm`
- Setup directory structure (as above)
- Create `Makefile` with targets: `test`, `build`, `run`, `migrate`, `lint`
- Setup CI workflow (GitHub Actions): run tests + linter on PR
- Create `README.md` with setup instructions

**Tests**:
- CI pipeline runs successfully
- `make build` produces `./fenixcrm` binary
- `./fenixcrm --version` displays version

**Resolves**: Infrastructure setup

---

#### Task 1.2: SQLite Setup + Migrations (3 days)

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

#### Task 1.3: Account Entity (3 days)

**Actions**:
- Create migration `002_crm_accounts.up.sql`:
  - `account` table (all fields from ERD)
  - Indexes: `workspace_id`, `owner_id`, `deleted_at`
- Write SQL queries in `infra/sqlite/queries/account.sql`:
  - `-- name: CreateAccount :one`
  - `-- name: GetAccountByID :one`
  - `-- name: ListAccounts :many` (with pagination)
  - `-- name: UpdateAccount :one`
  - `-- name: SoftDeleteAccount :exec`
- Run `sqlc generate` to produce Go code in `infra/sqlite/gen/`
- Implement `domain/crm/account.go`:
  - `type Account struct` (domain model)
  - `type AccountService struct { db *sql.DB }`
  - `Create(ctx, CreateAccountInput) (*Account, error)`
  - `Get(ctx, id string) (*Account, error)`
  - `List(ctx, ListAccountsInput) ([]*Account, error)`
  - `Update(ctx, id string, UpdateAccountInput) (*Account, error)`
  - `Delete(ctx, id string) error` (soft delete)
- Implement `api/handlers/crm.go`:
  - `POST /api/v1/accounts` → `CreateAccount`
  - `GET /api/v1/accounts` → `ListAccounts` (pagination)
  - `GET /api/v1/accounts/{id}` → `GetAccount`
  - `PUT /api/v1/accounts/{id}` → `UpdateAccount`
  - `DELETE /api/v1/accounts/{id}` → `DeleteAccount`

**Tests**:
- Unit tests for `AccountService` (mock DB)
- Integration tests:
  - Create account → verify in DB
  - List accounts → pagination works
  - Soft delete → `deleted_at` set, not visible in list
- API tests:
  - POST returns 201 + account JSON
  - GET returns 200 + account
  - PUT returns 200 + updated account
  - DELETE returns 204

**Resolves**: FR-001 (partial — Account CRUD)

---

#### Task 1.4: Contact Entity (2 days)

**Actions**:
- Create migration `003_crm_contacts.up.sql`:
  - `contact` table
  - FK to `account`, `owner_id`
  - Indexes
- Write SQL queries in `infra/sqlite/queries/contact.sql`
- Run `sqlc generate`
- Implement `domain/crm/contact.go` (same pattern as Account)
- Implement handlers:
  - `POST /api/v1/contacts`
  - `GET /api/v1/contacts`
  - `GET /api/v1/contacts/{id}`
  - `PUT /api/v1/contacts/{id}`
  - `DELETE /api/v1/contacts/{id}`
  - `GET /api/v1/accounts/{account_id}/contacts` (filter by account)

**Tests**:
- Same pattern as Account
- Additional: Filter contacts by account_id

**Resolves**: FR-001 (partial — Contact CRUD)

---

### Week 3: Lead, Deal, Case + Auth

#### Task 1.5: Lead, Deal, Case Entities (3 days)

**Actions**:
- Create migrations:
  - `004_crm_leads.up.sql`
  - `005_crm_deals.up.sql`
  - `006_crm_cases.up.sql`
  - `007_crm_pipelines.up.sql` (pipeline + pipeline_stage)
- Write SQL queries for each entity
- Run `sqlc generate`
- Implement domain services: `lead.go`, `deal.go`, `case.go`, `pipeline.go`
- Implement handlers (same CRUD pattern)

**Tests**:
- Unit + integration + API tests (same pattern)
- Test FK constraints (deal → account, stage)
- Test pipeline stage transitions

**Resolves**: FR-001 (Lead, Deal, Case CRUD), FR-002 (Pipeline basics)

---

#### Task 1.6: Authentication Middleware (2 days)

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

### Phase 1 Exit Criteria

✅ All CRM entity CRUD endpoints working
✅ JWT authentication active on all `/api/v1/*` routes
✅ 100% test coverage on critical paths
✅ Migrations up/down work cleanly
✅ CI pipeline green

---

## 3 — Phase 2: Knowledge & Retrieval (Weeks 4-6)

**Goal**: Hybrid search (BM25 + vector) with permission filtering and evidence pack assembly.

**Deliverable**: A working `/api/v1/knowledge/search` endpoint that returns ranked, permission-filtered results.

### Week 4: Knowledge Schema + Ingestion

#### Task 2.1: Knowledge Tables (2 days)

**Actions**:
- Create migration `009_knowledge.up.sql`:
  - `knowledge_item` table
  - `embedding_document` table
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
  ```
- Write SQL queries in `infra/sqlite/queries/knowledge.sql`
- Run `sqlc generate`

**Tests**:
- Integration test: Insert into `knowledge_item` + FTS5 sync
- Integration test: Query FTS5 with `MATCH`
- Integration test: Insert into `vec_embedding` + ANN query

**Resolves**: Database schema for knowledge

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

### Phase 2 Exit Criteria

✅ Knowledge ingestion working (text only)
✅ Hybrid search returns ranked results
✅ Evidence pack builder returns top-K with confidence
✅ LLM adapter (Ollama) functional
✅ 100% test coverage on search path

---

## 4 — Phase 3: AI Layer (Weeks 7-10)

**Goal**: Copilot Q&A, Support Agent (UC-C1), Tool Registry, Policy Engine.

**Deliverable**: End-to-end UC-C1 flow working — user triggers support agent → agent retrieves evidence → generates response → executes tools → updates case.

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

### Phase 3 Exit Criteria

✅ Copilot chat working with SSE streaming
✅ UC-C1 Support Agent end-to-end functional
✅ Tool execution with permissions + approvals + idempotency
✅ Policy engine 4 enforcement points active
✅ Handoff to human working

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

#### Task 4.5: Audit Service (2 days)

**Actions**:
- Create migration `014_audit.up.sql`:
  - `audit_event` table (if not already exists)
- Implement `domain/audit/service.go`:
  - `Log(ctx, AuditEvent) error` (append-only)
  - `Query(ctx, QueryInput) ([]*AuditEvent, error)` (filters, pagination)
  - `Export(ctx, format) (io.Reader, error)` (CSV/JSON)
- Subscribe to ALL events from event bus → log to `audit_event`
- Implement handlers:
  - `GET /api/v1/audit/events` (query + filters)
  - `POST /api/v1/audit/export` (download CSV)

**Tests**:
- Integration test: CRM action → audit event logged
- Integration test: Query audit events → returns filtered results
- Integration test: Export → CSV file generated

**Resolves**: FR-070 (audit trail)

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

### Phase 4 Exit Criteria

✅ React frontend functional (CRM pages + Copilot + Agent runs)
✅ Audit service logging all events
✅ Eval service basic functionality
✅ E2E tests passing
✅ Documentation updated

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
- **Overall**: ≥80%

### CI Pipeline

```yaml
# .github/workflows/ci.yml
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - run: make lint
      - run: make test
      - run: make build
  e2e:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm ci
      - run: make e2e
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

**End of Implementation Plan**

Next step: Start Phase 1, Task 1.1 (Project Setup).
