# AGENT_SPEC Integration Analysis

> **Date**: 2026-03-06
> **Status**: Pre-analysis complete. Pending: progressive incorporation plan.
> **Source**: `docs/AGENT_SPEC.md`
> **Context**: P0 MVP in progress (Phase 1 complete, Phases 2-4 partial).
> **Naming source of truth**: `docs/agent-spec-overview.md`
> **Traceability rules**: `docs/agent-spec-traceability.md`

---

## 1 -- What AGENT_SPEC Proposes

### Naming Alignment

The repository already uses stable top-level use case IDs such as `UC-C1` and `UC-A1`.

For AGENT_SPEC, the new platform capabilities are normalized as:
- `UC-A2` Workflow Authoring
- `UC-A3` Workflow Verification and Activation
- `UC-A4` Workflow Execution
- `UC-A5` Signal Detection and Lifecycle
- `UC-A6` Deferred Actions
- `UC-A7` Human Override and Approval
- `UC-A8` Workflow Versioning and Rollback
- `UC-A9` Agent Delegation

`BEHAVIOR` names remain valid as lower-level scenario identifiers inside each `UC`.

A collision check against the existing requirements catalog found no prior use of `UC-A2` to `UC-A9`.

A declarative framework where **the DSL is the program**. Business workflows are described, not coded. CRM state emerges from execution rather than manual data entry.

### Core Pipeline

```
Human / MCP / A2A  -->  BPMN translation  -->  DSL  -->  Judge  -->  Runtime  -->  CRM operations
```

### Key Concepts

| Concept | Definition |
|---|---|
| **DSL** | Executable specification language with verbs: ON, IF, SET, WAIT, NOTIFY, SURFACE, DISPATCH, AGENT |
| **BPMN Layer** | Resolves business concept ambiguity before DSL compilation. Skipped for agent-to-agent input. |
| **Judge** | Verifies consistency between spec and DSL via 8 checks before execution is allowed. |
| **Spec Anatomy** | 4 mandatory blocks: CONTEXT, ACTORS, BEHAVIOR (GIVEN/WHEN/THEN), CONSTRAINTS |
| **Protocol Modes** | EXECUTE (run locally), DISPATCH (send to other agent via A2A), VERIFY (validate only) |
| **Protocol Responses** | ACCEPTED, REJECTED (with reason), DELEGATED (forwarded) |
| **Signal** | The unit of truth -- replaces "contact" as the central concept |
| **Workflow Entity** | First-class citizen, defined by Salesperson, executed by Agent, verified by Judge |
| **Constraints** | Invariants that must never be violated regardless of execution path |

### DSL Example

```
WORKFLOW detect_and_surface_intent
  ON      contact.interaction_recorded
  IF      contact.interactions.count >= 1
    AGENT evaluate_intent(contact.interaction_history)
    SET   contact.intent_signal = agent.result
    IF    contact.intent_signal == HIGH
      SURFACE contact TO salesperson.view WITH reason
      WAIT  48 hours
      IF    salesperson.has_not_acted
        NOTIFY salesperson WITH contact + reason
```

### Judge Verification Checks

1. Is every THEN clause observable without knowing the implementation?
2. Does any BEHAVIOR contradict a CONSTRAINT?
3. Are there ACTORS in BEHAVIOR not defined in ACTORS?
4. Are there GIVEN states never produced by another BEHAVIOR?
5. Does the DSL BLOCK match what the BEHAVIOR blocks describe?
6. Does the DSL BLOCK use only BPMN-grounded concepts?
7. Are all protocol responses (ACCEPTED/REJECTED/DELEGATED) covered?
8. List any term that could be interpreted in more than one way.

---

## 2 -- Current Architecture Snapshot (as of 2026-03-06)

### What Exists (built and tested)

| Component | Location | Status |
|---|---|---|
| CRM CRUD (Account, Contact, Lead, Deal, Case, Activity, Note, Attachment, Timeline) | `internal/domain/crm/` | Complete |
| SQLite + FTS5 + sqlite-vec | `internal/infra/sqlite/` | Complete |
| Knowledge ingestion + chunking | `internal/domain/knowledge/` | Partial |
| Hybrid search (BM25 + vector + RRF) | `internal/domain/knowledge/` | Partial |
| Evidence pack builder | `internal/domain/knowledge/` | Partial |
| LLM adapter (Ollama) | `internal/infra/llm/` | Partial |
| Policy engine (4 enforcement points) | `internal/domain/policy/` | Partial |
| Tool registry + built-in tools | `internal/domain/tool/` | Partial |
| Agent orchestrator (Go hardcoded) | `internal/domain/agent/` | Partial |
| Support agent (UC-C1) | `internal/domain/agent/agents/support.go` | Done |
| Prospecting agent | `internal/domain/agent/agents/prospecting.go` | Done |
| KB agent | `internal/domain/agent/agents/kb.go` | Done |
| Insights agent | `internal/domain/agent/agents/insights.go` | Pending |
| Audit trail (immutable) | `internal/domain/audit/` | Done |
| Eval service (basic) | `internal/domain/eval/` | Done |
| BFF gateway (Express.js) | `bff/` | Done |
| Mobile app (React Native) | `mobile/` | Pending |

### How Agents Work Today

Agents are Go structs with hardcoded logic:
- `AgentDefinition` in DB defines: name, type, allowed tools, limits, trigger config, policy set
- `AgentRun` records execution: inputs, evidence, reasoning trace, tool calls, output, cost
- Orchestrator (`TriggerAgent`) runs a fixed state machine: context -> evidence -> LLM -> tools -> audit
- Tools are registered in `ToolRegistry` with JSON Schema validation + permissions + rate limits
- Policy engine checks 4 enforcement points: before retrieval, before prompt, before tool, after execution

### ERD Entities Relevant to AGENT_SPEC

| Entity | Relevance to AGENT_SPEC |
|---|---|
| `agent_definition` | Could evolve to reference DSL workflows instead of hardcoded agent types |
| `skill_definition` | Has `steps` (JSON) -- closest thing to a proto-DSL. Currently out of scope (P1) |
| `tool_definition` | DSL verbs (SET, NOTIFY, SURFACE) would map to tool calls |
| `agent_run` | Execution record -- DSL Runtime would produce equivalent runs |
| `approval_request` | Maps to AGENT_SPEC's approval gates for sensitive actions |
| `policy_set` | Maps to CONSTRAINTS block |
| `audit_event` | Immutable log -- unchanged by AGENT_SPEC |
| `eval_suite` / `eval_run` | Judge verification could extend eval framework |
| `knowledge_item` / `evidence` | AGENT concept of "signal" maps to evidence + knowledge items |
| `timeline_event` | Captures state changes -- "CRM state emerges" aligns with this |

---

## 3 -- Gap Map: AGENT_SPEC vs. Current Architecture

### 3.1 -- Total Gaps (nothing exists)

| AGENT_SPEC Concept | What's Needed | Estimated Complexity | Suggested Phase |
|---|---|---|---|
| **DSL Parser** | Lexer + parser for DSL grammar (ON, IF, SET, WAIT, NOTIFY, SURFACE, DISPATCH, AGENT) | High -- formal grammar, error reporting, type checking | P2 |
| **DSL Runtime/Interpreter** | Execution engine that reads DSL and produces CRM operations | High -- state management, async (WAIT), event-driven | P2 |
| **BPMN Translator** | NL -> BPMN -> DSL compiler. Requires LLM + BPMN schema knowledge | High -- a project in itself | P2 |
| **Protocol Handler** | DISPATCH mode: map workflow execution to A2A-compatible dispatch. VERIFY mode: dry-run validation | Medium | P2 |
| **MCP Gateway** | Expose and consume tools/context using MCP transports and auth model | Medium | P2 |
| **A2A Gateway** | Inter-agent communication using A2A over HTTP(S) + JSON-RPC | High -- transport, auth, discovery, streaming | P2 |
| **Workflow Entity** | New DB table: `workflow { id, workspace_id, name, dsl_source, bpmn_source, status, version, created_by, ... }` | Low -- schema only | P1 |
| **Spec Parser** | Parse CONTEXT/ACTORS/BEHAVIOR/CONSTRAINTS blocks from spec text | Medium | P2 |

### 3.2 -- Partial Gaps (concept exists, needs extension)

| AGENT_SPEC Concept | Current Equivalent | Extension Needed | Suggested Phase |
|---|---|---|---|
| **Judge** | `domain/eval/` (eval suites + scoring) | Add workflow-level verification (8 checks). Reuse eval framework structure | P1 (basic) / P2 (full) |
| **Constraints** | `policy_set.rules` (ABAC rules JSON) | Map DSL CONSTRAINTS to policy rules. Add spec-level constraint validation | P1 |
| **Protocol Responses** | `agent_run.status` (success/failed/abstained/escalated) | Add ACCEPTED/REJECTED/DELEGATED semantics. `escalated` ~= DELEGATED | P1 |
| **Signal** | `knowledge_item` + `evidence` + `timeline_event` | Formalize "signal" as a concept: event + evidence + confidence + action trigger | P1 |
| **Workflow** | `skill_definition.steps` (JSON array of tool calls + conditions) | Evolve steps JSON toward DSL-like structure as intermediate format | P1 |
| **Agent as DSL executor** | `domain/agent/orchestrator.go` (hardcoded state machine) | Make orchestrator pluggable: Go agent OR DSL-driven agent | P1 (interface) / P2 (DSL) |

### 3.3 -- No Gap (already compatible)

| AGENT_SPEC Concept | Current Implementation | Notes |
|---|---|---|
| Tool-gated actions | `domain/tool/` registry + validation + permissions | DSL verbs would call the same tools |
| Policy enforcement | `domain/policy/` 4 enforcement points | CONSTRAINTS map naturally to policy rules |
| Audit trail | `domain/audit/` immutable append-only | DSL Runtime would emit audit events identically |
| Evidence packs | `domain/knowledge/evidence.go` | AGENT verb would trigger evidence pack building |
| Approval workflows | `domain/policy/approval.go` | DSL actions requiring approval use same flow |
| Eval/quality gates | `domain/eval/` | Judge extends eval, doesn't replace it |
| Cost/quota controls | `agent_definition.limits` + `agent_run.total_cost` | DSL workflows inherit same budget controls |

---

## 4 -- Paradigm Tension

The fundamental tension is **CRM-first vs. Agent-first**:

| Aspect | Current Architecture | AGENT_SPEC Vision |
|---|---|---|
| **Primary entity** | CRM records (Account, Contact, Deal, Case) | Signals and Workflows |
| **Data entry** | Manual CRUD + agent assistance | State emerges from workflow execution |
| **Agent definition** | Go struct with hardcoded logic | DSL source text, interpreted by Runtime |
| **Extensibility** | New Go code per agent type | New DSL workflow, no code change |
| **Inter-agent** | Not supported | A2A dispatch protocol |
| **Business logic** | In Go services | In DSL + BPMN |
| **Verification** | Unit/integration tests | Judge (spec consistency) + tests |

**Resolution strategy**: These are not mutually exclusive. The current architecture provides the **execution substrate** (tools, policies, evidence, audit). AGENT_SPEC provides the **orchestration layer** on top. The bridge is `skill_definition` evolving into DSL workflows.

---

## 5 -- Progressive Incorporation Strategy (Summary)

### Guiding Principle

**Don't rewrite. Extend.** Every P0 component becomes infrastructure for the DSL layer.

### Evolution Path

```
P0 (current)                    P1 (next)                         P2 (future)
-------------------------------+-------------------------------------+---------------------------
Go hardcoded agents             Pluggable orchestrator interface      DSL Runtime replaces
                                + skill_definition as proto-DSL       hardcoded agents

Tool Registry                   Tools exposed via DSL verb mapping    DSL calls tools natively
                                (SET -> update_case, NOTIFY -> send)

Policy Engine                   CONSTRAINTS -> policy rule mapping    CONSTRAINTS block parsed
                                                                      and enforced at DSL level

Evidence Packs                  Signal concept formalized             AGENT verb auto-retrieves
                                (event + evidence + confidence)       evidence

Eval Service                    Judge basic checks (spec consistency) Full 8-check Judge
                                reusing eval framework                integrated in pipeline

Event Bus (Go channels)         Event-driven triggers for workflows   Signal-driven DSL execution

agent_run status                Add ACCEPTED/REJECTED/DELEGATED       Full protocol mode support
                                as extended status values

(nothing)                       Workflow entity in DB                 BPMN layer + DSL parser
                                (DSL source + version + status)       + full Runtime
```

### Risks to Monitor

| Risk | Mitigation |
|---|---|
| Scope creep -- P0 never finishes | Complete P0 first, no AGENT_SPEC changes until Phase 4 done |
| Two paradigms coexisting without resolution | Define clear interface boundary: orchestrator is pluggable |
| BPMN parser is a project in itself | Defer to P2. Use LLM-assisted BPMN generation, not custom parser |
| Estandares de interoperabilidad evolucionan | Definir desde el inicio A2A para dispatch y MCP para tools; encapsularlos detras de adapters |
| DSL grammar design takes too long | Start with skill_definition JSON as proto-DSL in P1 |

---

## 6 -- Next Steps (when resuming this work)

1. **Finish P0 MVP** -- complete remaining Phase 2-4 tasks per `docs/implementation-plan.md`
2. **Design pluggable orchestrator interface** -- abstract `TriggerAgent` so it can be backed by Go code or DSL
3. **Evolve skill_definition** -- design the `steps` JSON format to be DSL-compatible
4. **Formalize "Signal" concept** -- define entity/event that bridges knowledge items and agent triggers
5. **Prototype Judge** -- extend eval framework with spec-level consistency checks
6. **Define Workflow entity** -- DB schema for storing DSL source + BPMN source + versions
7. **Create DSL grammar specification** -- formal BNF/PEG grammar for the DSL verbs and syntax
8. **Define interoperability target** -- A2A for dispatch, MCP for tools/context, before external integration
9. **Build DSL interpreter** -- Runtime that reads DSL and calls existing tools/policies/evidence

---

## Appendix A -- File References

| Document | Path | Relevance |
|---|---|---|
| AGENT_SPEC (source) | `docs/AGENT_SPEC.md` | The vision document being analyzed |
| Architecture | `docs/architecture.md` | Current system design (ERD, modules, API, diagrams) |
| Implementation Plan | `docs/implementation-plan.md` | 13-week P0 plan with task status |
| Requirements | `agentic_crm_requirements_agent_ready.md` | Original FR/NFR requirements |
| Corrections Applied | `docs/CORRECTIONS-APPLIED.md` | Audit of plan corrections |

## Appendix B -- AGENT_SPEC DSL Verb -> Current Tool Mapping

| DSL Verb | Closest Current Tool/Service | Gap |
|---|---|---|
| `ON <event>` | `infra/eventbus/bus.go` Subscribe | Event names need standardization |
| `IF <condition>` | Go conditionals in agent logic | Needs expression evaluator |
| `SET <entity.field> = <value>` | `update_case`, `create_task` tools | Field-level mapping needed |
| `WAIT <duration>` | Not implemented | Needs scheduler/timer service |
| `NOTIFY <actor> WITH <data>` | `send_reply` tool (partial) | Needs generic notification service |
| `SURFACE <entity> TO <actor.view>` | Not implemented | Needs UI push/priority queue |
| `DISPATCH TO <agent> WITH <workflow>` | Not implemented | Needs A2A-compatible dispatch |
| `AGENT <function>(<args>)` | `TriggerAgent()` in orchestrator | Needs sub-agent invocation |

## Appendix C -- Interoperability Direction

- `DISPATCH` externo debe ser A2A-first usando `a2aproject/a2a-go` como primera implementacion en Go.
- Exposicion y consumo de tools/contexto debe ser MCP-first usando el SDK oficial de MCP para Go.
- `ProtocolHandler` debe ser un puerto interno con adapters compatibles con estandares, no un protocolo propietario.
- HTTP no es el contrato objetivo; es el transporte del estandar cuando aplique.
