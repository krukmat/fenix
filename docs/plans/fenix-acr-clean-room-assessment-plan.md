---
doc_type: plan
id: FENIX-ACR-CR-001
title: "Adaptive Cognitive Runtime - Clean-Room Assessment and Handoff"
status: proposed
phase: assessment
created: 2026-11-13
updated: 2026-11-13
tags: [fenix, acr, clean-room, agent-runtime, local-models, product-assessment]
fr_refs: [FR-230, FR-233]
uc_refs: [UC-C1]
---

# ACR Clean-Room Assessment and Handoff Plan

## 1. Purpose and decision boundary

Evaluate externally observed agent-runtime capabilities for potential business value in FenixCRM, establish the actual Fenix implementation baseline, and produce an independently authored functional requirements package for a *separate, later* implementation plan.

**ACR is a working name for a set of possible capabilities, not an approved new service, SDK, subsystem, or architectural layer.** OpenWorker is an external subject of research, not a dependency or implementation source.

This plan is complete only when Fenix can decide which, if any, missing or partial capabilities warrant implementation. Neither adoption of OpenWorker nor development of ACR is authorized by completion of this assessment.

Product priority: Support Agent and Support Copilot first; Sales Copilot only where the same requirement has demonstrated value. The canonical strategic-repositioning plan remains authoritative for implementation ordering.

## 2. Governing documents and current anchors

Read before each task, subject to repository authority and version freeze:

- CLAUDE.md and AGENTS.md; docs/playbooks/AGENT_WORKFLOW_GUIDE.md.
- docs/architecture.md; docs/plans/fenixcrm_strategic_repositioning_spec.md.
- docs/plans/fenixcrm_strategic_repositioning_implementation_plan.md.
- docs/decisions/ADR-100-agentic-blackboard-architecture.md.
- docs/integration-overview.md; docs/policies/HITL_AUTONOMY_POLICY.md.
- docs/new-release/wave8-dedupe-budgets-and-release-gating-analysis.md.

**As-built anchors requiring gap verification (not presumed gaps):**

| Existing Fenix area | Code or contract anchor | Assessment question |
| --- | --- | --- |
| Runtime and runners | internal/domain/agent/runner.go; internal/domain/agent/orchestrator.go | What lifecycle and recovery semantics are already provided? |
| Workflow resumption | internal/domain/agent/dsl_runner.go; internal/domain/agent/workflow_resume_handler.go | Which restart/recovery paths work, and do they cover conversational investigations? |
| Collaborative cognition | internal/domain/blackboard/; ADR-100 | Does the Blackboard already preserve the necessary reasoning state? |
| Governed execution | internal/domain/tool/registry.go; policy/approval domains | Can proposed improvements reuse existing authority without a second tool or approval plane? |
| Metering and quotas | internal/domain/usage/service.go; Wave 8 | Which limits are enforced in executable paths rather than only represented in schemas/plans? |
| Evidence recovery | internal/infra/integration/evidenceoutbox/; docs/integration-overview.md | Does the existing no-business-replay invariant remain sufficient? |
| Provider abstraction | internal/infra/llm/provider.go; internal/infra/llm/router.go | Which model capability or resource signals are missing? |
| Context reduction | scripts/peer-workflow-review.py; docs/plans/local_peer_review_prompt_compaction_plan.md | Is compaction confined to developer peer-review tooling or supported by product runtime? |

Written contracts, executable tests, observed runtime behavior, and target architecture must be labeled separately. Finding a method or a plan is not proof that a use case works end-to-end.

## 3. Assessment scope

Investigate the following **business problems**, not their implementation patterns:

1. **Cognitive continuity:** retain relevant case facts, decisions, citations, pending work, and approvals over long support investigations without an unbounded model context.
2. **Interruption-safe continuation:** recover an in-progress investigation after process or host restart while reconciling uncertain effects.
3. **Resource-aware inference:** use available local or cloud models within memory, context, capacity, privacy, and cost constraints; queue or decline work predictably.
4. **Budget accountability:** constrain consumption per execution and tenant, including delegated requests, and represent unknown usage honestly.
5. **Operational diagnosis:** explain model choices, interruptions, tool proposals, recoveries, and costs with already authorized Fenix evidence and telemetry.
6. **Approval continuity:** resume permitted work after human intervention without confusing session permission with business authorization.

Out of scope: replacing Blackboard/Orchestrator/RunnerRegistry/PolicyEngine/ToolRegistry; adopting OpenWorker SDK or its architecture; implementing a new agent service; modifying business schemas or runtime; broad CRM/Agent Studio expansion; claiming production certification from one laptop.

## 4. Clean-room protocol

### 4.1 Role and information boundaries

| Role | Owner | Permitted input | Must not receive |
| --- | --- | --- | --- |
| Reference researcher (A) | ChatGPT, source-facing context | Public OpenWorker product documentation, public behavior, Fenix source/test baseline | A mandate to implement or translate external code |
| Independent requirements author (B) | ChatGPT in a **separate fresh, source-isolated context** | Fenix product goals, verified gaps, neutral problem statements, own acceptance scenarios | OpenWorker source, guides, snippets, field/schema/API names, screenshots, reference dossier |
| Local baseline operator (C) | Local agent | Fenix repository, clean acceptance plan, model inventory and synthetic test fixtures | OpenWorker materials and implementation instructions |
| Independent handoff reviewer (D) | ChatGPT in a source-isolated review context | Clean requirements, Fenix evidence, test results, provenance checklist | External reference internals |

The same live conversation that examined OpenWorker **must not be treated as a genuinely isolated authoring context** for the clean functional specification. Create a separate context with a restricted input set for role B. Access restrictions should be enforced through available tools/workspaces, not merely by asking an agent not to consult a source. If isolation cannot be established, mark the deliverable **NOT_CLEAN** and do not release it as an implementation input.

### 4.2 Artifact separation

- **Restricted reference dossier (A-only, outside the Fenix implementation workspace):** source URLs, frozen revisions, observations, and provenance. It is *not* transferred to the implementer.
- **Neutral problem brief (B input):** original Fenix business problems, observed Fenix baseline/gaps, success criteria, and constraints. No OpenWorker names, references, copied terminology, or internal structures.
- **Clean specification and handoff (B/C/D):** independently written Fenix behavior, invariants, test cases and baseline measurements. This is the **only** assessment package transferred to a future implementation planner.

No external source code, tests, pseudocode, prompts, diagrams, configuration schemas, or internal architectural structures are to be copied or adapted. Functional ideas expressed as independently justified customer outcomes may be assessed. A source-isolated context limits exposure but is not, by itself, a legal guarantee of independent creation.

### 4.3 Traceability without contamination

Keep detailed source attribution in the restricted dossier; keep Fenix source/test references in the clean baseline. A separate reviewer records the provenance check and the mapping from the restricted investigation to the approved *neutral* problem identifiers. The clean package is complete without access to the restricted dossier.

## 5. Execution sequence and responsibility

Execution is strictly sequential. No parallel branches or workstreams; downstream work requires the preceding exit gate.

| Order | Task | Owner | Hard dependency | Required output |
| --- | --- | --- | --- | --- |
| 1 | CR.T0 - Protocol and version freeze | ChatGPT (A) | None | Frozen-source manifest; isolation rules |
| 2 | CR.T1 - Reference capability inventory | ChatGPT (A) | T0 | Restricted, behavioral capability inventory |
| 3 | CR.T2 - Verified Fenix baseline | ChatGPT (A) | T1 | Source/test-backed as-built and gap matrix |
| 4 | CR.T3 - Business-value triage | ChatGPT (A) | T2 | Ranked opportunities, rejected duplications |
| 5 | CR.T4 - Neutralization and isolation gate | ChatGPT (A), independent reviewer | T3 | Sanitized Fenix problem brief; release checklist |
| 6 | CR.T5 - Independent functional specification | ChatGPT (B) | T4 PASS | Clean requirements and acceptance contracts |
| 7 | CR.T6 - Hardware/model evaluation specification | ChatGPT (B) | T5 | Portability profiles and predeclared test protocol |
| 8 | CR.T7 - Read-only baseline measurements | Local agent (C) | T6 | Reproducible results and limitations |
| 9 | CR.T8 - Independent review and handoff | ChatGPT (D) | T7 | Final disposition and implementation-planning input |

### CR.T0 - Protocol and version freeze

- T0.1: Record exact Fenix and reference revisions, date, evidence locations, repository authority order, and the separation model.
- T0.2: Specify access control and check that implementation-facing actors cannot read the external dossier.

**Exit:** reproducible source baseline and enforceable clean-room boundary. If the required isolation cannot be achieved, stop or downgrade the output to a non-clean architectural study, never silently substitute the latter.

### CR.T1 - External behavioral inventory

- T1.1: Examine published behavioral contracts for continuity, recovery, budget control, tool authorization, telemetry, and model choice; record publication/revision and evidence confidence.
- T1.2: Translate observations into **problem categories**, explicitly discarding external APIs, implementations and vendor-specific designs.

**Exit:** restricted inventory stating what was observed, what was merely documented, and what remains unverified.

### CR.T2 - Fenix as-built gap assessment

- T2.1: Follow relevant production paths and tests from Agent/Copilot through provider, policy, tool execution, persistence, and audit; identify existing behavior.
- T2.2: Assign each candidate one of IMPLEMENTED, PARTIAL, ABSENT, or NOT_VERIFIED. Link every conclusion to Fenix code and an executed test or an explicit testing gap.

**Exit:** no candidate described as missing solely because documentation was silent; distinguish evidence delivery recovery from business-run or conversational recovery.

### CR.T3 - Business-value triage

- T3.1: State the affected user journey, severity/frequency hypothesis, and business outcome for Support Agent/Copilot or Sales Copilot.
- T3.2: Assess incremental value, complexity, operational risk, portability and overlap; classify KEEP, INVESTIGATE, or DROP.

**Exit:** only real, measurable Fenix gaps proceed; findings with no incremental product value or duplicate governance are dropped.

### CR.T4 - Neutralization and isolation gate

- T4.1: Prepare a vendor-neutral brief containing only Fenix problems, validated gaps, constraints and testable outcomes. Keep links to external references exclusively in the restricted dossier.
- T4.2: Independently screen for borrowed terminology, implementation prescriptions, copied structures and source leakage; confirm isolation for role B.

**Exit:** PASS/FAIL provenance checklist. On FAIL, revise the brief; do not send reference material downstream.

### CR.T5 - Clean functional requirements

- T5.1: In a fresh, source-isolated context, author standalone Fenix requirements, invariants, user journeys and behavior at failure boundaries.
- T5.2: Define original acceptance cases with positive, negative and interrupted-operation variants, tenant and authorization constraints, and measurable outcomes.

**Exit:** a developer can implement the requested behaviors using only Fenix sources and the clean packet. No requirement prescribes a reference implementation.

### CR.T6 - Hardware and model evaluation specification

- T6.1: Define capability-driven profiles (available memory, quantization, context allocation, model/tool behavior, latency, privacy policy, concurrency and fallback constraints), independent of Apple/Ollama or any particular model ID.
- T6.2: Before observing results, declare synthetic datasets, measurement procedures and acceptance thresholds for continuity, correctness, memory pressure, latency, resource exhaustion and interruption safety.

**Exit:** reproducible protocol covering at least a small/medium local model, a constrained large model and a provider-unavailable scenario. Models are chosen by actual installed inventory, not assumed availability.

### CR.T7 - Read-only hardware baseline

- T7.1: Inventory actual MacBook M5 32 GB conditions, installed local models, quantization, server settings and available system memory; run the approved Fenix baseline scenarios with synthetic data.
- T7.2: Record model load/peak memory, memory pressure/swap, input/output token usage when available, latency, recovery results, citation/decision retention and queue/overload behavior.

**Exit:** raw measurements, reproduction instructions, PASS/PARTIAL/BLOCKED/NOT_TESTED statuses and limitations. No changes to Fenix product code; no OpenWorker installation. Do not extrapolate a single laptop benchmark to minimum product hardware.

### CR.T8 - Independent review and implementation handoff

- T8.1: Independently verify provenance, acceptance coverage, business-value justification, as-built evidence and benchmark integrity; dispose of each candidate as IMPLEMENTED, IMPROVE, INVESTIGATE, or DROP.
- T8.2: Produce a clean implementation-planning input containing **only** selected requirements, affected Fenix-owned boundaries, dependency ordering, acceptance cases, risks and unresolved decisions.

**Exit:** implementation-planning package accepted for *planning only*; user explicitly selects whether any requirement proceeds. No development or roadmap mutation is implied.

## 6. Non-negotiable product invariants

1. A successful business action cannot be replayed merely to recover dialogue, a tool result, or evidence.
2. Unknown external action outcomes must be reconciled under an existing execution identity before a retry.
3. Context reduction must preserve original evidence references, materially relevant decisions, pending work and their provenance; summaries cannot replace the evidence authority.
4. Session continuation must not silently renew or widen expired/denied human approvals.
5. Tenant isolation and policy-sensitive information must hold across history storage, restoration, provider selection and fallback.
6. Local-to-cloud fallback must obey existing no-cloud/PII policies and explicit authorization, not only capacity pressure.
7. Parent and delegated agent requests share effective ceilings; delegation cannot reset budgets or permissions.
8. Unknown token counts, costs or outcomes must remain UNKNOWN, not zero or success.
9. No capability may require a specific laptop, model family, inference server or deployment topology.

## 7. Experimental profiles and metrics

The MacBook M5 with 32 GB of unified memory is an initial **validation fixture**, not a product minimum. Use the local model inventory at execution time; representative candidates can include the installed Gemma, GPT-OSS and Qwen variants only after verifying exact IDs, quantizations and compatibility.

| Profile | Purpose | Observations |
| --- | --- | --- |
| P-SMALL | One compact/medium local model | Feasible context, latency and resource headroom |
| P-MEDIUM | One stronger model | Tradeoff between task quality, memory and context |
| P-CONSTRAINED | Model/context near resource limit | Pressure, refusal/queue behavior and safe recovery |
| P-UNAVAILABLE | Model or process interrupted | Status recovery, reconciliation and no duplicate effects |
| P-PORTABLE | Abstract capacity profile below/above laptop | Identify configurable versus hardware-specific assumptions |

Required measures: memory peak and pressure, swap, model load time, time to first token where supported, generation throughput, input/output tokens (or UNKNOWN), context size, successful recovery rate, evidence retention, policy adherence and double-effect count.

Set numerical thresholds in CR.T6 before CR.T7. Do not invent performance results or promise that an advertised model context fits physically in memory.

## 8. Final artifacts and storage

| Artifact | Owner | Destination | Implementation-facing? |
| --- | --- | --- | --- |
| Frozen manifest + reference dossier | A | Restricted workspace **outside** Fenix implementation checkout | No |
| Fenix baseline/gap report | A | docs/reports/acr-clean-room/fenix-baseline.md | Yes, if source-neutral |
| Business-value disposition | A | docs/reports/acr-clean-room/business-value-matrix.md | Yes, after neutralization |
| Neutral Fenix problem brief | A | docs/reports/acr-clean-room/problem-brief.md | Yes; sole external-study handoff to B |
| Clean functional requirements | B | docs/specs/acr-functional-requirements.md (planned) | Yes |
| Clean acceptance contract | B | docs/specs/acr-acceptance-contract.md (planned) | Yes |
| Hardware/model baseline | C | docs/reports/acr-clean-room/local-benchmark.md | Yes |
| Final independent review | D | docs/reports/acr-clean-room/handoff-review.md | Yes |
| Clean implementation-plan input | D | docs/plans/acr-implementation-input.md | Yes; **not** the implementation plan |

Planned paths are subject to checking existing repository conventions at creation time. This plan creates none of those downstream artifacts yet. The restricted dossier must never be committed into the Fenix implementation repository.

## 9. Acceptance and non-goals

**Completion gate (all required):**

- Every retained capability has Fenix product justification and a code/test-backed baseline.
- Distinguish what is written, what was tested, and what is only hypothesized.
- Clean-author separation is documented and verified; any violation blocks clean-room sign-off.
- No duplicated implementation of existing governance, execution, audit or recovery mechanisms is proposed without a demonstrated gap.
- Requirements are provider- and hardware-agnostic, with a measurable local baseline.
- Behavior for unknown results, interruptions, session permissions, tenant isolation and policy-constrained fallback is explicit.
- The final handoff can support an implementation plan without access to reference materials.

**Not implied by completion:** approval to implement, adoption of a third-party SDK, a new ACR service, new infrastructure, production-readiness certification or a modification to the canonical strategic implementation order.

## 10. Execution discipline and next phase

This document authorizes **planning**, not execution. Before executing each CR.Tn, create a proper docs/tasks/task_*.md record with the mandatory frontmatter, explicit parent-plan link, scope, acceptance criteria and dependency status. Follow the active task-review, human-approval, attribution and local-QA rules from CLAUDE.md and the canonical workflow guide. Execute one task at a time; do not parallelize.

After CR.T8, draft a **separate** Fenix implementation plan only for the capabilities classified IMPROVE, with discrete tasks and dependencies bound to the canonical roadmap. The future implementer receives the clean packet and Fenix repository only. Product-code changes remain subject to independent approval.

## 11. References (governance only; not a clean implementation input)

- External evaluation subject: https://github.com/andrewyng/openworker/tree/main/harness
- Fenix repository: https://github.com/krukmat/fenix

Exact external revisions and documentary provenance must be frozen under CR.T0 and retained in the restricted dossier, not transferred into functional requirements.
