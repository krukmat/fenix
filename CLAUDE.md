# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Project: Governed AI CRM Operations Layer

**What**: A self-hosted governed AI layer for customer operations, combining CRM context, retrieval grounding, safe tools, approvals, auditability, and agent/copilot execution.

**Why**: Enable trustworthy AI workflows over customer context without forcing teams into opaque automation or broad CRM replacement bets. The moat is governance, evidence, approval, audit, and controlled execution.

**Where we are**: Core implementation exists. The architecture has been strategically realigned toward support and sales workflow wedges, with planning to follow for the remaining backlog reshaping.

**Key documents**:
- Requirements: `docs/requirements.md`
- Architecture & design (ERD, diagrams, API, build order): `docs/architecture.md`
- Strategic repositioning spec: `docs/plans/fenixcrm_strategic_repositioning_spec.md`
- Canonical implementation plan: `docs/plans/fenixcrm_strategic_repositioning_implementation_plan.md`
- Historical MVP implementation plan (reference only): `docs/implementation-plan.md`
- Corrections applied (audit report): `docs/CORRECTIONS-APPLIED.md`

**Source of truth rules (MANDATORY)**:
- Before planning or implementing any task, ALWAYS read `docs/architecture.md` for current architectural constraints and `docs/plans/fenixcrm_strategic_repositioning_spec.md` for product-direction constraints.
- `docs/plans/fenixcrm_strategic_repositioning_implementation_plan.md` is the canonical implementation ordering for the current strategy.
- `docs/implementation-plan.md` remains useful historical execution context, but if it conflicts with the canonical repositioning plan, the architecture doc, or the repositioning spec, the newer documents take precedence.
- If the historical implementation plan is ambiguous or conflicts with the architecture doc, follow the canonical repositioning plan and architecture doc, then flag the drift explicitly.

**Agent attribution (MANDATORY)**:
- Before making any git commit, ALWAYS run both to avoid stale attribution from a previous agent:
  ```
  export AI_AGENT="claude-sonnet-4-6"
  git config fenix.ai-agent "claude-sonnet-4-6"
  ```
- Multi-agent session: `export AI_AGENTS="orchestrator:claude-opus-4-6,coder:claude-sonnet-4-6"`
- The hook reads env var first, then git config, then falls back to `unknown`.
- Setting both guarantees correct attribution even if the env var is lost between shell sessions.

**Local environment setup (MANDATORY after clone)**:
- After cloning the repo or setting up a new local environment, ALWAYS run:
  ```
  make install-hooks
  ```
- This installs the pre-push hook that enforces Go and mobile quality gates locally before any push reaches GitHub Actions.
- Without this step, broken code can reach the CI pipeline undetected.

**Push discipline (MANDATORY)**:
- Treat `git push` as the final step after local validation, not as the first validation step.
- The pre-push hook (`scripts/hooks/pre-push`) automatically detects what changed and runs the appropriate QA gates:
  - **Go changes** (`internal/`, `cmd/`, `pkg/`, `go.mod`, `go.sum`, `.golangci.yml`, `Makefile`): runs `scripts/qa-go-prepush.sh` (fmt-check, complexity, lint, test, coverage, deadcode, traceability, govulncheck, pattern-gate)
  - **Mobile changes** (`mobile/`, mobile scripts, `ci.yml`): runs `scripts/qa-mobile-prepush.sh` (typecheck, lint, arch, coverage)
- If a required local QA gate cannot be executed, stop and report that explicitly before pushing.
- Hooks must be installed with `make install-hooks` (see setup above).

**Task file discipline (MANDATORY)**:
- Before starting any implementation work on a task, a corresponding task file MUST exist in `docs/tasks/` with the correct `doc_type: task` frontmatter.
- If a task file does not exist, STOP. Create the task file first and present it to the user for review before writing any code.
- Never work on a task that has no individual task file — not even exploratory edits, test stubs, or scaffolding.
- This applies to all task types: eval waves, feature tasks, BDD tasks, infra tasks, bugfixes, and any other planned work unit.
- The task file is the contract: it defines scope, acceptance criteria, and files affected. No task file = no task.
- Every task file MUST include a `**Plan**:` reference link to its parent plan document immediately after the `# Task ...` heading. Format: `**Plan**: [Plan Title](relative/path/to/plan.md#anchor)`.

**Reporting (MANDATORY)**:
- Every substantive report to the user must include:
  - `Complejidad: Baja | Media | Alta | Muy alta`
  - `Tokens: ~N` (approximate estimate of the response/report size)
- Apply this to progress updates and final summaries.

**Knowledge management / Obsidian rules (MANDATORY)**:
- Obsidian is the repository knowledge-management layer for project tracking docs, not a product feature.
- Maintain the doc vault proactively. If a task changes architecture, scope, roadmap, requirements, APIs, data model, operational rules, or delivery status, update the relevant Obsidian artifacts in the same turn without waiting for an explicit user request.
- Do not assume markdown files under `docs/` are structured task records unless they start with YAML frontmatter.
- Any new tracking artifact intended for Obsidian must declare `doc_type` in YAML frontmatter at the top of the file.
- Allowed `doc_type` values are: `task`, `adr`, `summary`, `audit`, `handoff`.
- If a change creates project-understanding drift, update the source document and also create or update the appropriate vault artifact (`summary`, `audit`, `adr`, or `task`) when future planning, governance, or traceability would otherwise be weakened.
- `docs/tasks/` is reserved for actual task records only. Do not place summaries, audits, handoffs, or scratch notes there unless the user explicitly asks for that structure.
- New task records in `docs/tasks/` must include at minimum:
  ```
  doc_type: task
  id:
  title:
  status:
  phase:
  week:
  tags: []
  fr_refs: []
  uc_refs: []
  blocked_by: []
  blocks: []
  files_affected: []
  created:
  completed:
  ```
- ADRs belong in `docs/decisions/`, not in `docs/tasks/`.
- Durable vault artifacts that define shared project reality must remain trackable in Git. This applies by default to canonical plans in `docs/plans/` and ADRs in `docs/decisions/`.
- `docs/tasks/` may contain operational task records that are useful in Obsidian without necessarily being promoted to shared Git history. Do not assume every task record must be committed.
- If a task record becomes the canonical source for coordination, delivery tracking, or cross-session handoff, promote it to a Git-trackable artifact explicitly.
- If ignore rules block a canonical plan or ADR that should be shared, fix the ignore rule or flag the conflict immediately.
- If you create or edit Obsidian dashboards / Dataview queries, filter by `doc_type` instead of assuming all files in a folder share the same schema.
- When strategic priorities change, update the relevant dashboards or summary notes so the vault continues to reflect current project reality.

---

## Core Design Principles

1. **Evidence-first**: No AI action without grounded evidence. Abstain when uncertain.
2. **Tools, not mutations**: AI executes via registered, allowlisted tools. Never direct data writes.
3. **Governed**: RBAC/ABAC, PII/no-cloud policies, approval chains, immutable audit logs.
4. **Operable**: Full tracing, dry-run, replay, eval-gated releases, cost tracking per agent/tenant.
5. **Model-agnostic**: Local or cloud LLMs, no vendor lock-in.

---

## Priorities: What Builds On What

**P0 (MVP)** — Must complete before P1:
- Context entities needed by the wedge: Account, Contact, Deal, Case, Activity, supporting notes and timeline
- Hybrid retrieval (FR-090/092): Keyword + vector with mandatory evidence packs
- Copilot + tools (FR-200/202): Grounded response flow, executable actions
- One end-to-end agent (UC-C1, FR-230): Support agent resolves cases
- Governance (FR-060/070/071): Permissions, audit trail, approvals
- Handoff (FR-232): Escalate to human with evidence
- Basic observability (NFR-030/031): Metrics per agent
- Usage and cost metering foundation (NFR-040/041 direction): per run, per workspace, per tool attribution

**P1 (v1)** — Enabled by P0:
- Multi-source ingestion (FR-091): Email, docs, calls
- Sales Copilot end-to-end
- Better connector coverage
- Agent catalog (FR-231): Prospecting, KB, insights agents
- Agent Studio (FR-240/241/242): Versioning, skills builder, evals
- Quotas + degradation (FR-233, NFR-040/041): Budget controls
- Replay/simulation (FR-243): Troubleshoot agent runs

**P2 (v2)** — Enabled by P1:
- Broad mobile parity and non-wedge CRM expansion
- Marketplace (FR-052): Plugin SDK and store

---

## Architecture Layers

> Full diagrams: `docs/architecture.md` (Section 3 — System Architecture)

**Stack**: Go 1.22+ / go-chi | SQLite (WAL) + sqlite-vec + FTS5 | Express.js BFF | React Native + Expo + React Native Paper | Docker Compose

**Mobile App (React Native)** → **BFF Gateway (Express.js)** → **Go Backend (go-chi REST)** → **SQLite**

**BFF responsibilities**: Auth relay, request aggregation, SSE proxy, mobile headers. Zero business logic, zero DB access.

**CRM Store (SQLite OLTP)** → **Event Bus (Go channels)** → **Connectors (Email/Docs/Calls) + Indexer** → **Hybrid Index (FTS5 BM25 + sqlite-vec ANN)**

**Go API** → **Policy Engine (RBAC/PII/no-cloud/approvals)** + **Copilot Service (SSE)** + **Agent Orchestrator** + **Tool Registry**

**Audit + Telemetry + Eval Service** (cross-cutting, immutable)

**LLM**: Model-agnostic adapter — Ollama/vLLM (local) + OpenAI/Anthropic (cloud). Provider router for no-cloud, budget, and fallback.

---

## What's Different From Traditional CRM

### 1. Evidence Packs (Mandatory)
Every copilot response and agent action must cite sources:
```
{
  sources: [
    { id: "email_123", snippet: "...", score: 0.95, timestamp: "2026-02-09T10:00Z" },
    { id: "case_456", snippet: "...", score: 0.88, timestamp: "2026-02-08T15:30Z" }
  ],
  confidence: "high",
  abstain_reason: null  // or reason if no response
}
```
If insufficient evidence → abstain + escalate to human.

### 2. Tool-Gated Actions
AI cannot mutate data directly. All changes go through registered tools:
```
Tool: "create_task"
  Schema: { owner, title, due_date, ... }
  Permissions: ["sales_rep", "manager"]
  Rate_limit: 10/min per user
  Idempotency: yes (via key)

Tool: "send_email"
  Schema: { to, template_id, ... }
  Permissions: ["support_agent", "sales_rep"]
  Approvals: ["manager"] if recipient external
```
Invalid params or missing perms → deny + log.

### 3. Policy-Driven Governance
**Example rules:**
- "PII data (phone, email, SSN) → redact before LLM, never send to cloud"
- "Case.internal_notes → only support team can retrieve"
- "Send email to external contact → requires manager approval"
- "Agente de prospecting → max 50 leads/day, max €10 cost/day"

Enforced at: retrieval, prompt building, tool execution, output formatting.

### 4. Agent Runs Are First-Class
Every agent execution is traceable:
```
AgentRun {
  id, agent_id, triggered_by, trigger_type (event/schedule/manual),
  inputs, retrieval_queries, retrieved_evidence,
  reasoning_trace, tool_calls,
  output, status (success/partial/abstained/failed),
  cost_tokens, cost_euros, latency_ms,
  audit_events: [who executed, perms checked, approvals],
  created_at, updated_at
}
```

### 5. Eval-Gated Releases
New prompts/policies must pass quality gates before prod:
- Groundedness: % outputs with sufficient evidence
- Exactitude: % accuracy vs. CRM ground truth
- Abstention correctness: % false positives in self-denial
- Policy adherence: 0 violations

Only promote if thresholds met. Else rollback via one-click.

---

## Key Entities & Relationships

### CRM Records
- **Workspace/Tenant**: Isolation boundary
- **User, Role, PolicySet**: Identity + permissions
- **Account, Contact, Lead, Deal, Case**: Operational records
- **Activity (Task/Event), Note, Attachment**: Supporting records

### AI/Evidence Layer
- **KnowledgeItem**: Ingested unit (email, doc snippet, call transcript, etc.)
- **EmbeddingDocument**: KnowledgeItem chunk + vector + metadata
- **Evidence**: Reference to KnowledgeItem + snippet + score + permission snapshot
- **AgentDefinition**: Goal + allowed tools + limits (quotas, retries) + policy tags
- **SkillDefinition**: Multi-step workflow (chain of tools)
- **ToolDefinition**: Schema + auth scopes + rate limits + idempotency support
- **AgentRun**: Execution record (above)
- **ApprovalRequest**: Proposed action + approvers + decision + timestamp
- **AuditEvent**: Actor + resource + change + timestamp (immutable)
- **EvalSuite / EvalRun**: Datasets + scoring results

---

## Retrieval & Indexing Strategy

**Hybrid ranking**: BM25 (keyword match) + vector similarity (semantic), configurable weight.

**Filtering**:
1. **Permission filter**: Only records user can access (per RBAC/ABAC)
2. **Sensitivity filter**: Redact PII before ranking (if no-cloud policy active)
3. **Freshness filter**: Warn if evidence is stale (e.g., deal stage changed yesterday)

**Incremental**: Changes via CDC (Change Data Capture) trigger reindex. Goal: visible within 60s.

**Deduplication**: Consolidate near-duplicate evidence in packs to reduce noise and cost.

---

## Tool Execution Flow

1. **LLM proposes**: `use_tool("create_task", { owner: "user_123", title: "Follow up" })`
2. **Tool Validator**: Check schema, check perms, validate params
3. **Policy Check**: Does this require approval? → Route to ApprovalRequest if yes
4. **Execute**: Call tool (idempotent), log result
5. **Audit**: Record in AgentRun + AuditEvent

If any step fails, respond with error + reason to agent (may retry or abstain).

---

## Cost Control & Quotas

**Per agent/role/tenant**:
- Tokens per day (LLM call volume)
- Cost per day (€, or credits)
- Executions per day (API rate)

**When hitting limit**:
- Circuit breaker: pause new executions
- Degradation: switch to cheaper model, reduce context, increase abstention threshold
- Alert: notify owner + admin

**Metrics tracked**: tokens per request, cost per outcome (€/ticket, €/deal).

---

## Compliance & Audit

**Immutable log** (append-only):
- All retrieval queries, evidence retrieved, LLM prompts, tool calls, outputs
- Actor identity, timestamp, permissions checked, decisions (approve/deny)
- Exportable, queryable with filters

**Policy violations**: Logged but execution prevented (e.g., no PII leak, no unpermitted access).

---

## Deployment Model

> Full details: `docs/architecture.md` (Section 10 — Deployment Architecture)

- **MVP**: Two processes — `./fenixcrm serve --port 8080` (Go) + `node bff/dist/index.js --port 3000` (BFF) + SQLite file.
- **Docker Compose**: `docker-compose up` starts Go backend + BFF + Ollama.
- **Mobile**: React Native + Expo (Android APK via EAS Build). iOS in P1.
- **BYO-LLM**: Local (Ollama, vLLM) or bring your own API key (OpenAI, Anthropic)
- **Multi-tenant**: Workspace isolation per tenant
- **Future**: PostgreSQL + Redis + NATS + Kubernetes for scale (P1/P2)

No vendor lock-in.

---

## Success Metrics (NFR)

- **Speed**: Copilot Q&A ≤2.5s p95, summaries ≤5s p95
- **Quality**: Groundedness >95%, abstention correctness >98%
- **Cost**: <€0.10 per copilot interaction on average
- **Adoption**: >50% WAU of copilot + actions in first quarter
- **Reliability**: 99.5% uptime, <0.1% tool failures

---

## Reporting

- Discrete tasks must be executed one at a time. After closing a task with the required outcome report, stop and wait for explicit user confirmation before starting the next task, even when a plan lists multiple tasks or waves.
- Before starting work on each discrete task, present the task card to the user first, before reading, editing, or running task-specific commands except for minimal inspection needed to identify the next task. The task card must include:
  - `Tarea: <name or ID>`
  - `Task file: <path to docs/tasks/task_*.md>`
  - `Plan file: <path to docs/plans/*.md>`
  - `Resumen: <what will be done in 1-2 sentences>`
  - `Código afectado: <expected files or areas>`
  - `Esfuerzo/razonamiento: Bajo | Medio | Alto - <brief reason>`
  - `Modelo recomendado: <model id>`
  - `Tokens estimado: ~N`
- **Task card language (MANDATORY)**: The task card must be written in unambiguous agentic English. No Spanish field labels, no mixed language. Every field value must be a direct, machine-parseable statement: what will be done, which files, why, estimated cost. Avoid narrative prose — prefer declarative sentences. This ensures the card is usable by any agent or orchestrator reading the conversation without language ambiguity.
- When closing a task, report the outcome with:
  - `Resultado: <what changed>`
  - `Verificación: <commands run, or why QA was not applicable>`
  - `Archivos afectados: <files changed>`
  - `Complejidad: Baja | Media | Alta | Muy alta`
  - `Modelo recomendado: <model id>`
  - `Tokens: ~N`
- After the closing report, proactively present the next task card using the same starting-task format, but do not begin that next task until the user explicitly confirms.
