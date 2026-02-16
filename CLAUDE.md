# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Project: Agentic CRM OS

**What**: A self-hosted, AI-native CRM platform combining operational CRM with evidence-based agents, RAG retrieval, and policy-driven governance.

**Why**: Close the gap between traditional CRMs (no agentic layer) and enterprise suites (vendor lock-in). Enable teams to build trustworthy AI workflows with full audit trails, evidence requirements, and cost controls.

**Where we are**: Architecture design complete. Entering implementation phase.

**Key documents**:
- Requirements: `agentic_crm_requirements_agent_ready.md`
- Architecture & design (ERD, diagrams, API, build order): `docs/architecture.md`
- Implementation plan (13 weeks, 4 phases, TDD): `docs/implementation-plan.md`
- Corrections applied (audit report): `docs/CORRECTIONS-APPLIED.md`

**Source of truth rules (MANDATORY)**:
- Before planning or implementing any task, ALWAYS read `docs/implementation-plan.md` for the task spec and `docs/architecture.md` for architectural constraints.
- These two documents are the primary source of truth. Only deviate with explicit user approval.
- If the implementation plan is ambiguous or conflicts with the architecture doc, ask the user before proceeding.

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
- CRM entities (FR-001/002): Account, Contact, Lead, Deal, Case, Activity
- Hybrid retrieval (FR-090/092): Keyword + vector with mandatory evidence packs
- Copilot + tools (FR-200/202): In-flow UI, executable actions
- One end-to-end agent (UC-C1, FR-230): Support agent resolves cases
- Governance (FR-060/070/071): Permissions, audit trail, approvals
- Handoff (FR-232): Escalate to human with evidence
- Basic observability (NFR-030/031): Metrics per agent

**P1 (v1)** — Enabled by P0:
- Multi-source ingestion (FR-091): Email, docs, calls
- Agent catalog (FR-231): Prospecting, KB, insights agents
- Agent Studio (FR-240/241/242): Versioning, skills builder, evals
- Quotas + degradation (FR-233, NFR-040/041): Budget controls
- Replay/simulation (FR-243): Troubleshoot agent runs

**P2 (v2)** — Enabled by P1:
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
