# FenixCRM Strategic Repositioning and Architecture Adjustment Specification

**Document ID:** STRAT-ARCH-001  
**Status:** Proposed  
**Language:** English  
**Audience:** Founder, Product, Architecture, Implementation Agents  
**Decision Type:** Business Direction + Architecture Adjustment  
**Authoring Mode:** Spec-driven, agent-ready, unambiguous  
**Date:** 2026-04-06

---

## 1. Executive Decision

FenixCRM **shall not** be positioned as a general-purpose CRM platform competing head-on with full-suite incumbents.

FenixCRM **shall** be repositioned as a **governed AI operating layer for CRM workflows**, focused on:

1. **Support Copilot and Support Agent** for case handling
2. **Sales Copilot** for account, deal, and relationship context
3. **Evidence-grounded AI execution** with approvals, auditability, and policy enforcement

The product direction is therefore:

> **A governed AI layer that sits on top of CRM workflows and enterprise knowledge, rather than a full CRM replacement.**

This document defines the business repositioning, product scope changes, architecture changes, and implementation priorities required to align the project with a realistic market direction.

---

## 2. Problem Statement

The current implementation direction combines too many product theses at once:

- CRM core
- knowledge ingestion and retrieval
- copilot UI
- agent runtime
- approvals and audit
- mobile app
- BFF
- agent studio
- evals

This creates three strategic risks:

### 2.1 Market Positioning Risk

If presented as a new CRM, FenixCRM will be compared against mature incumbents on breadth, integrations, ecosystem, and operational maturity.

### 2.2 Scope Dilution Risk

The current breadth makes it difficult to prove one high-value use case end-to-end.

### 2.3 Value Communication Risk

The strongest differentiators already implemented or designed are not generic CRM features. They are:

- evidence-grounded retrieval
- abstention behavior
- safe tool routing
- approvals
- immutable audit trail
- policy enforcement
- prompt and policy versioning
- traceable delivery discipline

These differentiators are currently under-positioned in business terms.

---

## 3. Product Thesis

## 3.1 Primary Thesis

FenixCRM shall be positioned as a **Governed AI CRM Operations Layer**.

It shall provide:

- grounded answers over CRM and knowledge data
- controlled tool execution
- human approval when required
- full auditability of AI-driven actions
- policy-based safety and data handling

## 3.2 Product Category Statement

FenixCRM is **not** a horizontal CRM suite.

FenixCRM is a **workflow intelligence and governed agent execution layer** for customer-facing operations.

## 3.3 Core Promise

FenixCRM shall help teams use AI in CRM-related workflows **without requiring blind trust in model outputs**.

## 3.4 Positioning Statement

For organizations that need AI assistance in customer operations but cannot accept opaque or unsafe automation, FenixCRM provides a governed execution layer with retrieval grounding, approvals, policy controls, and audit evidence.

---

## 4. Ideal Customer Profile (ICP)

FenixCRM shall target the following initial ICP:

### 4.1 Company Profile

- B2B or B2B2C organizations
- Small to mid-sized teams, or focused enterprise departments
- Teams already using a CRM, ticketing platform, or internal customer data system
- Organizations with operational friction in support and sales handoffs
- Organizations that care about audit, policy compliance, and traceability

### 4.2 Buyer Profile

Primary buyers:

- Head of Support Operations
- RevOps Lead
- CX Operations Lead
- AI Transformation Lead
- Product/Platform Lead responsible for workflow automation

Secondary buyers:

- CTO
- Director of Engineering
- Enterprise Architect

### 4.3 Pain Profile

The ICP shall have at least three of the following pains:

- agents and copilots are not trusted due to hallucination risk
- support teams lose time gathering fragmented context
- sales teams need faster account and deal summarization
- approvals and manual review are required before outbound or risky actions
- AI initiatives stall because governance is weak
- current CRM AI features are too generic, too expensive, or too opaque

---

## 5. Beachhead Use Cases

FenixCRM shall prioritize two beachhead use cases and one secondary use case.

### 5.1 Beachhead Use Case A — Support Agent with Governed Execution

A support user or automation flow triggers an AI agent to:

1. retrieve case context and related knowledge
2. assemble an evidence pack
3. draft a grounded response or next-step recommendation
4. execute safe tools only when allowed
5. request approval when required
6. update case context and preserve full audit trail

This shall be the **primary commercial wedge**.

### 5.2 Beachhead Use Case B — Sales Copilot with Grounded Context

A sales user opens an account or deal context and receives:

- grounded account summary
- open risks and objections
- next best actions
- relevant timeline synthesis
- recommended follow-up content based on evidence

This shall be the **secondary wedge**, but still within initial scope.

### 5.3 Secondary Use Case — Human Handoff with Evidence

When confidence is low or approval is required, the system shall:

- abstain or defer
- attach evidence pack
- preserve rationale and workflow state
- hand off to a human without losing context

This is not optional. It is a core product promise.

---

## 6. Explicit Non-Goals

The following shall be treated as explicit non-goals for the current product direction:

### 6.1 Non-Goal: Full CRM Replacement

FenixCRM shall not aim to replace Salesforce, HubSpot, Dynamics, or equivalent full CRM platforms in its initial product strategy.

### 6.2 Non-Goal: Broad Horizontal CRM Breadth

The product shall not optimize for wide CRM parity across every object, workflow, and admin capability.

### 6.3 Non-Goal: Mobile-First Differentiation

Mobile shall not be treated as a strategic differentiator in the initial go-to-market direction.

### 6.4 Non-Goal: Agent Studio as Initial Commercial Front Door

Agent Studio may remain in roadmap scope, but it shall not be the initial market-facing core offer.

### 6.5 Non-Goal: Plugin Marketplace

A marketplace model shall remain out of scope until the beachhead use cases prove repeatable value.

---

## 7. Commercial Packaging Direction

FenixCRM shall be packaged commercially as follows.

### 7.1 Offer Type

The product shall be framed as one of these two equivalent business models:

1. **AI layer for existing CRM and support workflows**
2. **Governed AI workspace for customer operations**

### 7.2 Packaging Units

The packaging model shall be organized around operational capability, not generic seat bundles.

Required packages:

#### Package 1 — Support Copilot

Includes:
- retrieval-grounded case assistant
- evidence pack generation
- abstention logic
- agent draft mode
- audit trail

#### Package 2 — Support Agent

Includes all Support Copilot features plus:
- safe tool execution
- approval workflows
- governed actions on cases/tasks/notes
- handoff and escalation state

#### Package 3 — Sales Copilot

Includes:
- account/deal grounding
- relationship and timeline synthesis
- next-step suggestions
- risk and objection summarization

### 7.3 Pricing Logic

The commercial model should be able to evolve toward a combination of:

- workspace fee
- operator seats
- agent execution volume
- governed action volume

The architecture shall therefore preserve traceability for:

- per-agent usage
- per-workspace usage
- per-tool usage
- per-approval events

---

## 8. Strategic Architecture Shift

## 8.1 Current Direction to Preserve

The following architectural investments are aligned with the revised business direction and **shall be preserved**:

- evidence pack architecture
- hybrid retrieval
- approval workflow concepts
- immutable audit model
- policy engine
- prompt and policy versioning
- safe tool registry
- multi-tenant isolation
- BDD and traceability discipline

## 8.2 Strategic Shift Required

The architecture shall shift from:

> **monolithic product vision: CRM + AI + mobile + studio + broad platform**

To:

> **governed AI execution layer attached to CRM workflows and customer knowledge**

This means the architecture must optimize first for:

1. trust
2. workflow fit
3. integration boundaries
4. measurable operational value

Not for:

1. broad CRM parity
2. UI surface breadth
3. mobile coverage breadth
4. platform breadth before wedge validation

---

## 9. Target Product Architecture

The target architecture shall be described as six primary capability layers.

### 9.1 Capability Layer A — System of Context

This layer shall hold or expose operational context:

- accounts
- contacts
- deals
- cases
- activities
- notes
- attachments
- timeline events
- linked knowledge items

This layer may be native or integrated. It does not need to be positioned as the final system of record for every customer.

### 9.2 Capability Layer B — Knowledge and Retrieval

This layer shall:

- ingest customer-operation-relevant content
- maintain search freshness
- support hybrid retrieval
- produce deduplicated evidence packs
- enforce workspace isolation

### 9.3 Capability Layer C — Policy and Governance

This layer shall:

- evaluate permissions and action eligibility
- enforce no-cloud / PII policies where configured
- determine approval requirements
- constrain tool routing
- record policy decision context

### 9.4 Capability Layer D — Agent Runtime

This layer shall:

- orchestrate agent execution
- bind prompt versions and policy versions
- consume evidence packs
- call safe tools
- request approvals when required
- support abstention and handoff

### 9.5 Capability Layer E — Operational Interfaces

This layer shall provide:

- support copilot UI
- sales copilot UI
- approval UI
- audit review UI
- optional BFF proxy where technically justified

### 9.6 Capability Layer F — Evaluation and Cost Control

This layer shall:

- evaluate workflows and prompts
- track groundedness and abstention outcomes
- measure execution costs
- enforce budgets and quotas
- support replay and simulation in later phases

---

## 10. Mandatory Architecture Adjustments

The following changes are mandatory.

### 10.1 Architecture Adjustment A — Reframe CRM Core as Context Layer

The CRM domain shall be treated as a **context layer** that enables the AI workflows.

It shall not be treated as the main product moat.

Implications:

- new CRM entity work shall only be added when directly required by beachhead workflows
- generic CRM breadth expansion shall be deprioritized
- architecture documents shall stop describing CRM breadth as the main business axis

### 10.2 Architecture Adjustment B — Make Retrieval + Evidence a First-Class Boundary

The retrieval system shall be treated as a distinct product boundary, not just an internal utility.

Required consequences:

- evidence pack schema shall be explicit and versioned
- evidence confidence rules shall be explicit and testable
- abstention thresholds shall be policy-driven where possible
- search freshness SLA shall remain visible and measurable

### 10.3 Architecture Adjustment C — Promote Governance to a Core Runtime Concern

Policy, approval, and audit shall be runtime-critical capabilities, not optional overlays.

Required consequences:

- every governed tool execution shall emit audit events
- every approval-required action shall have a deterministic approval state model
- policy decisions shall be explainable in machine-readable form
- denial outcomes shall be first-class execution results

### 10.4 Architecture Adjustment D — Introduce Cost Governance as P0/P1

Budget controls and usage accounting shall move forward in priority.

Required consequences:

- every agent run shall record token/cost/latency metadata where available
- every tool call shall be attributable to workspace, agent, and actor
- workspace-level quotas shall be enforceable in a later incremental phase
- architecture shall reserve a dedicated usage ledger capability

### 10.5 Architecture Adjustment E — Treat Integrations as Core, Not Peripheral

The architecture shall assume that value increases when FenixCRM can operate with external systems.

Required consequences:

- ingestion adapters shall be modeled as formal connectors
- external system references shall be preserved in source metadata
- case/account/deal context may come from internal tables or external systems
- integration contracts shall be stable and versioned

### 10.6 Architecture Adjustment F — Reduce Mobile Priority

Mobile shall remain optional for the wedge unless a specific target workflow proves mobile is critical.

Required consequences:

- mobile parity shall not block core workflow completion
- BFF and mobile work shall be justified by beachhead workflow evidence, not by symmetry

---

## 11. Required Domain Model Changes

The following domain changes shall be applied.

### 11.1 Introduce a Usage and Cost Domain

A new domain area shall be created:

- `usage_ledger`
- `quota_policy`
- `execution_metering`

Minimum entities shall include:

#### `usage_event`
- id
- workspace_id
- actor_id
- actor_type
- run_id
- tool_name
- model_name
- input_units
- output_units
- estimated_cost
- latency_ms
- created_at

#### `quota_policy`
- id
- workspace_id
- policy_type
- limit_value
- reset_period
- enforcement_mode
- created_at
- updated_at

#### `quota_state`
- id
- workspace_id
- policy_id
- current_value
- period_start
- period_end
- updated_at

### 11.2 Formalize Approval State Model

`approval_request` shall support an explicit finite-state model:

- `pending`
- `approved`
- `rejected`
- `expired`
- `cancelled`

State transitions shall be deterministic and audited.

### 11.3 Formalize Evidence Pack Versioning

The evidence pack structure shall include a versioned contract.

Minimum required fields:

- schema_version
- query
- sources
- source_count
- dedup_count
- filtered_count
- confidence
- warnings
- retrieval_methods_used
- built_at

### 11.4 Formalize Agent Outcome Types

Agent runtime outcomes shall be normalized into a bounded set:

- `completed`
- `completed_with_warnings`
- `abstained`
- `awaiting_approval`
- `handed_off`
- `denied_by_policy`
- `failed`

This shall prevent ambiguous runtime semantics.

---

## 12. Required API and Contract Adjustments

The API surface shall be adjusted to reflect the repositioned product.

### 12.1 APIs That Become Strategic

The following APIs shall be treated as strategic core APIs:

- `POST /knowledge/search`
- `POST /knowledge/ingest`
- `POST /copilot/query`
- `POST /agent-runs`
- `POST /approvals/{id}/approve`
- `POST /approvals/{id}/reject`
- `GET /audit-events`
- `GET /usage`
- `GET /health`
- `GET /metrics`

### 12.2 APIs That Become Secondary

Generic CRUD APIs remain necessary but shall be described as support APIs, not category-defining APIs.

### 12.3 Contract Requirement

Every strategic API shall define:

- input schema
- output schema
- deterministic error codes
- audit emission behavior
- policy evaluation touchpoint
- tenant isolation rule

---

## 13. Required Backlog Reprioritization

The backlog shall be reprioritized as follows.

## 13.1 New Priority Order

### Priority 0 — Must Exist for the Wedge

1. Support Agent end-to-end
2. Support Copilot grounded response flow
3. Evidence pack quality and confidence behavior
4. Approval flow and handoff behavior
5. Immutable audit and policy trace
6. Usage and cost metering foundation
7. Integration-ready ingestion contracts

### Priority 1 — Strongly Valuable Next

1. Sales Copilot end-to-end
2. Better connector coverage
3. Eval suite for groundedness and action safety
4. Budget and quota enforcement
5. Replay and simulation

### Priority 2 — Defer

1. broad Agent Studio capabilities beyond what beachhead workflows need
2. plugin marketplace
3. mobile breadth parity
4. broad CRM expansion not tied to the wedge

## 13.2 Features to Deprioritize Immediately

The following items shall not block the go-to-market wedge:

- broad mobile feature parity
- plugin marketplace
- generic skills builder breadth
- non-essential CRM object expansion
- platform-level extensibility before integration proof

## 13.3 Features to Pull Forward

The following items shall move forward:

- usage metering
- quotas and cost controls
- connector formalization
- approval UX and runtime determinism
- audit review experience

---

## 14. Architecture Work Packages

Implementation agents shall produce the following work packages.

### WP-01 — Repositioning ADR Set

Create ADRs for:

1. product category shift: governed AI layer, not full CRM replacement
2. cost governance as mandatory runtime concern
3. integration-first context strategy
4. mobile deprioritization rationale

### WP-02 — Usage and Quota Technical Spec

Create a technical specification for:

- usage_event schema
- cost estimation rules
- quota enforcement points
- per-run attribution model

### WP-03 — Evidence Contract Spec

Create a stable schema definition for evidence pack generation, confidence tiers, abstention triggers, and handoff payload.

### WP-04 — Support Agent Reference Flow

Produce one canonical, fully specified reference workflow:

1. trigger source
2. context resolution
3. retrieval
4. evidence build
5. policy evaluation
6. draft generation
7. approval gate if required
8. action execution
9. audit logging
10. handoff fallback

### WP-05 — Sales Copilot Reference Flow

Produce one canonical grounded copilot flow for account/deal context.

### WP-06 — Connector Contract Spec

Define connector interfaces for:

- ingest source identity
- provenance metadata
- refresh strategy
- delete behavior
- permission mapping

---

## 15. Acceptance Criteria for Strategic Alignment

The repositioned direction shall be considered accepted only if all of the following are true.

### 15.1 Business Acceptance Criteria

- product messaging describes FenixCRM as a governed AI operations layer
- the primary wedge is support agent / support copilot
- sales copilot is positioned as a second wedge, not the only offer
- full CRM replacement language is removed from primary positioning

### 15.2 Architecture Acceptance Criteria

- architecture docs explicitly separate context layer from governed AI layer
- usage and quota domain is added
- evidence pack contract is formalized and versioned
- approval states are explicit and deterministic
- strategic APIs are defined with policy and audit behavior

### 15.3 Delivery Acceptance Criteria

- one end-to-end support workflow is fully demonstrable
- one end-to-end sales copilot workflow is fully demonstrable
- audit trail is visible for all governed actions
- abstention and handoff are test-covered behaviors
- cost and usage can be reported per workspace and per run

---

## 16. Immediate Next Actions

The next implementation iteration shall do the following, in order:

1. update architecture documentation to reflect the governed AI layer positioning
2. create the usage and quota domain specification
3. lock the evidence pack contract and agent outcome model
4. produce a canonical support agent end-to-end spec
5. remove or downgrade roadmap statements implying broad CRM replacement ambition
6. treat mobile and broad studio expansion as non-blocking

---

## 17. Final Strategic Recommendation

FenixCRM should continue.

However, it shall continue under a more focused and defensible direction:

- **not** as a new broad CRM suite
- **but** as a governed AI execution layer for customer operations workflows

This direction is better aligned with:

- the strongest existing technical assets in the project
- the most defensible value proposition
- the lowest-risk route to meaningful market validation

---

## 18. One-Sentence Product Definition

**FenixCRM is a governed AI layer for customer operations that turns CRM and knowledge context into grounded, auditable, approval-aware assistance and agent execution.**

