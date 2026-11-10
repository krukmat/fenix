---
doc_type: plan
id: DB-FENIX-INT
title: "Fenix service provider for DubBridge — capability-neutral integration"
status: active_design
created: 2026-10-29
rebaseline: 2026-10-31
tags: [fenix, dubbridge, integration, s2s, capability-registry]
---

# Canonical plan — DubBridge consumes Fenix

> **Owner decision, 2026-10-31 — cloud-agent execution:** this initiative does **not** use RRI as a planning, model-routing or implementation gate. All earlier RRI references below are superseded for `DB-FENIX-INT`. Keep task decomposition, Go QA, independent peer/security review, and explicit approval for authentication, migrations and deployment. [Scoped policy](../policies/RRI_POLICY.md).



**Owner correction 2026-10-31:** Actual business workflows and agents **emerge from product usage**. Their discovery, selection, definition, model quality and execution are **NOT entry criteria for B1, B2, B3 or infrastructure B4**. The old review-backlog-analysis example is a deferred, nonbinding idea, not the first mandatory capability.

DubBridge is the consumer/product and owns user/project permissions, media, review decisions, publication, rights and P2P. Fenix is the agent/workflow infrastructure provider and owns its runtime, S2S authority, tool governance, Blackboard, workspace isolation and execution records.

## Current C1–C3 source work completed in Cloud — 2026-11-10

**Scope:** Only C1–C3 were changed. This is an unvalidated feature-branch candidate and **not** a code/release PASS.

| Task | Concrete Cloud change | Evidence still needed |
|---|---|---|
| C1 | Tightened embedded SQLite 039 checks so `accepted` cannot hold a fabricated `run_id`, and run-required states cannot store an empty `run_id`; added two negative migration assertions | Real Go 1.25.13 modernc, readiness and independent code review |
| C2 | Corrected replay equality to compare immutable `trace_id`; added mismatched-trace conflict test even if the digest is incorrectly reused | Real Go 1.25.13 focused/race, readiness and independent code review |
| C3 | Added revoked-principal/expired-lease reconciliation regression: `indeterminate`, cleared lease, lookup denied | Real Go 1.25.13 focused/race, readiness and independent code review |

The new code/test diff is bounded to five paths: `internal/infra/sqlite/migrations/039_service_operation.up.sql`, `internal/infra/sqlite/service_operation_migration_test.go`, `internal/domain/integration/service_operation_store.go`, `internal/domain/integration/service_operation_store_test.go`, `internal/domain/integration/service_operation_recovery_test.go`. **No C4 source, production HTTP route, CI, credential, operator action or merge**. The existing C1 readiness `needs_changes` remains pending re-review; the work performed does not waive it.

## Current execution baseline — B2.C1–C3 only (2026-11-10)

**Goal:** complete and independently verify the *inert*, Fenix-owned durable S2S operation foundation; no production integration, status endpoint or agent workflow is activated. The current block is **B2.C1–C3**. B2.C4 is deferred and never a prerequisite for bounded C1–C3 source acceptance. The independent security verification in [#8](https://github.com/krukmat/fenix/issues/8) is tracked separately for the auth/release boundary.

| Order | Task | Concrete deliverable | Dependency | Current acceptance state |
|---|---|---|---|---|
| 1 | [B2.C1](../tasks/task_dubbridge_fenix_b2c1_sqlite_schema.md) | Embedded SQLite 039 up/down with immutable principal/workspace/org/project operation identity, constraints and migration tests | B2.A2 scoped persisted-principal baseline [#3](https://github.com/krukmat/fenix/issues/3) | Source candidate; prior readiness `needs_changes`; documentary gaps corrected; renewed independent readiness + Go 1.25.13 QA still required |
| 2 | [B2.C2](../tasks/task_dubbridge_fenix_b2c2_operation_store.md) | Atomic reservation, Fenix-derived scope and request digest replay/conflict, deterministic bounded input, concurrency tests | C1 and default-deny B2.B capability registry | Source candidate; local Go QA/readiness/code review not completed |
| 3 | [B2.C3](../tasks/task_dubbridge_fenix_b2c3_recovery.md) | One-shot lease, no automatic redispatch, revoked-principal handling and conservative `indeterminate` recovery with tests | C2 | Source candidate; local Go QA/readiness/code review not completed |
| 4 | Verification | Actual Go 1.25.13 focused/full tests, race, vet, repository QA with honest failure attribution; genuine independent task readiness and exact-source code review | C1–C3 documentary/code candidates frozen | NOT RUN for the B2.C candidate |
| 5 | Reconciliation | Fix attributable findings, repeat affected verification, publish final source SHA and proof; record bounded C1–C3 GO/NO-GO | Verified task and QA results | Pending |

**Definition of done for this block:** C1–C3 acceptance assertions verified on real SQLite/Go, independent readiness and post-code review recorded on the correct code/document SHA, tests/QA and any exceptions reported with actual artifacts, no unresolved attributable blocking findings. A scoped source-only completion is distinct from **production readiness**. Overall integration rollout, live capability registration, B2.B4 policy, B2.D network/response controls and T4 security remain their own future gates.

**Responsibility boundary:** ChatGPT Cloud owns source, targeted remediation, task/plan reconciliation and source-only closure record; local agent owns actual environment-bound verification and independent-review evidence. Reviewer implementation/model selection is governed by [AGENT_WORKFLOW_GUIDE](../playbooks/AGENT_WORKFLOW_GUIDE.md) and must **not** become a dependency or requirement in this integration plan.

**Immediate action:** rerun C1 readiness on the corrected task, then perform C1–C3 Go QA and independent code review; return evidence in [#9](https://github.com/krukmat/fenix/issues/9). [Validation handoff](../handoffs/dubbridge-fenix-b2c-readiness-validation-2026-11-10.md). Historical/global documentation QA debt remains separately tracked and must not be represented as PASS.

## Operator scope change — 2026-11-10: C4 deferred, non-blocking

The operator removed **B2.C4 (scoped GET/status)** from the current execution/closure gates. The bounded target is now **B2.C1 migration 039 + C2 reservation/idempotency + C3 lease/ambiguity recovery**. C1–C3 may advance to independent readiness, actual Go 1.25.13 QA and independent code review and **close as an inert source-only block without C4** when those gates PASS. C4's future implementation/security gates remain in its own deferred task; GET/status stays default-deny and no production route, status-based E2E, capability activation or deployment is authorized.

B2.A3.T4 independent security review is a **separate pending security gate** for the S2S security boundary and rollout, not a dependency on C1–C3 source-only verification. Follow-on B2.D work may be *planned* for transport/error/trace controls without status functionality, but any real GET/status, polling, B4 status E2E or release acceptance must be separately scoped and reviewed later. No gate is being waived; irrelevant future work is removed from this phase.

## B2.C feature-branch checkpoint — 2026-11-10

**Owner approved C1–C4**. ChatGPT Cloud implemented **C1–C3 code candidates** in `feature/int-v2-dubbridge-consumes-fenix`, not a production integration; coordinate through [B2.C #9](https://github.com/krukmat/fenix/issues/9) and [parent task](../tasks/task_dubbridge_fenix_b2c_operation.md).

| Task | Accountable | Deliverable | Verified state |
|---|---|---|---|
| [C1 SQLite](../tasks/task_dubbridge_fenix_b2c1_sqlite_schema.md) | ChatGPT Cloud | Embedded reversible migration 039, exact active Fenix scope insert trigger, immutable identity and terminal-state no-rewind constraints, Go test candidate | Cloud synthetic SQLite negative/rollback smoke PASS; real Go 1.25.13 and independent review PENDING |
| [C2 Reservation](../tasks/task_dubbridge_fenix_b2c2_operation_store.md) | ChatGPT Cloud | Real-store atomic INSERT SELECT with principal-scoped replay/conflict, bounded canonical JSON digest, Fenix registry-derived handler fingerprint, Go tests | SOURCE PUBLISHED; Go/race/peer PENDING |
| [C3 Recovery](../tasks/task_dubbridge_fenix_b2c3_recovery.md) | ChatGPT Cloud | Single lease, capability fingerprint fencing, scoped lookup, explicit indeterminate recovery without automatic retries or synthetic agent runs, Go tests | SOURCE PUBLISHED; Go/race/peer PENDING |
| [C4 GET/status](../tasks/task_dubbridge_fenix_b2c4_status_resolver.md) | ChatGPT Cloud (later) | Historical contract and test matrix; no HTTP adapter | **DEFERRED / NOT A C1–C3 BLOCKER** |

The prior `docs/contracts/dubbridge-fenix-b2c-operation-proposal.*.sql` is a historical **proposal**; embedded `039_service_operation` is the current feature-branch implementation candidate. Independent task-readiness has **not been run or waived for B2.C**, and local Go 1.25.13 full test/race/vet/lint/review is unverified. Do not claim code PASS, operation execution, exactly-once external effects, a live S2S endpoint, production migration or `main` merge. Fenix capability registry remains empty/default-deny outside isolated tests; C4 cannot be activated until T4's separate review **if the deferred work is resumed**; it is not required to verify or close inert C1–C3.

## B2.C readiness checkpoint — 2026-11-10 (documentation only)

- C1 readiness at `67729056` was reported `needs_changes` because the B2.A2 dependency evidence and FR/UC non-applicability rationale were absent from its packet; **no C1 PASS yet**.
- B2.A2: [owner accepted scoped source and H1/H2/M1 QA evidence, issue #3](https://github.com/krukmat/fenix/issues/3), on `87fd10d`. Embedded migration 038 and principal store exist for branch-local C1 SQL 039 verification. Fresh independent security acceptance is **not** reported; original independent FAIL is not overwritten.
- B2.C is S2S **infrastructure**, not a delivered CRM product feature. Empty FR/UC references on C1–C4 are intentional and individually explained; canonical traceability is this integration plan, neutral S2S envelope and task-specific acceptance. No product UC, agent run or configured capability should be inferred.
- C1–C3 Go QA, C1 renewed readiness, C1–C3 independent code reviews and B2.A3.T4 exact-diff security review are **NOT RUN on this revised documentation**. [Executable handoff](../handoffs/dubbridge-fenix-b2c-readiness-validation-2026-11-10.md) is the next local action; C4 stays deferred/default-deny and does not block C1–C3. RRI N/A DB-FENIX-INT/cloud.

## Prior B2.C cloud-first sequencing — superseded by current execution baseline

**Scheduling decision:** Complete C1–C3 source review, bounded remediation preparation and documentation before local Go/reviewer execution. C4 is deferred and is not on this critical path. Avoid parallel local implementations that would invalidate the reviewed SHA. Source changes in CF2 remain subject to the repository's independent task-readiness requirements; a plan approval is not a peer-review waiver.

| Sequence | Responsibility | Work package | Dependency / result |
|---|---|---|---|
| CF1 | ChatGPT Cloud | C1/C2/C3 source audit and tests/contract inspection | Current feature-branch source |
| CF2 | ChatGPT Cloud | Narrow implementation/test fixes; only if required readiness permits | CF1 findings; compile/review not claimed |
| CF3 | ChatGPT Cloud | Freeze only C1–C3 source candidate SHA and update local QA handoff | C4 excluded |
| LG1 | Local | C1–C3 independent readiness, Go 1.25.13 focused/full/race/vet/prepush, exact-scope independent code review | C1–C3 source-only gate |
| LG2 | Independent local security reviewer | T4 exact-diff review in #8, tracked separately for security release | Not a C4-derived blocker for inert C1–C3 |
| CF4 | ChatGPT Cloud | Address C1–C3 findings, reconcile reviewed SHA, close scoped code only on real PASS | No C4 implementation |
| NEXT | ChatGPT Cloud | Propose non-status B2.D follow-on analysis and tasks | No live GET/status or integration activation |

**External-gate split:** C1–C3 code remains SOURCE_CANDIDATE until authentic local evidence; C4 remains NOT_IMPLEMENTED/default-deny and deferred; it is **not** a gate for C1–C3 bounded closure. Operator capability policy B2.B4, B2.D transport/response controls and production HITL remain separately gated. A local reviewer should never be asked to implement Cloud-owned code unless a documented environment-only fix is indispensable.

## Architecture and interface boundary

```text
DubBridge API (domain authorization)
   -> DubBridge S2S connector (credential + operation_id + trace)
   -> Fenix S2S gateway (service principal, tenant/workspace scope, rate limit)
   -> capability router (registered capabilities only; inactive by default)
   -> [FUTURE: actual Fenix agent/workflow linked to registered capability]
   -> operation status/result correlation -> DubBridge
```

**A capability is a logical, versioned integration contract, NOT an implementation workflow ID.** Fenix's registry maps a permitted capability key to a workflow/agent only when that capability is actually configured. If none is registered/enabled, deny invocation deterministically (`CAPABILITY_NOT_CONFIGURED` or `CAPABILITY_DISABLED`), without accepting or fabricating a run. No dynamic caller-selected workflow IDs, user-supplied DSL, arbitrary prompts or broad tool access.

**Infrastructure-only test seam:** To test accepted-operation/idempotency/status across real apps in B4 without a business workflow, permit one explicitly test-only `integration.diagnostics` capability in an isolated nonproduction configuration. It is a real, bounded, deterministic provider-side diagnostic operation using the real S2S middleware and SQLite operation store, **not** an agent, mock business outcome or production-enabled feature. Production registry remains empty/default-deny. B4 evidence must say **infrastructure E2E**, never functional agent E2E.

B1 determines only the stable envelope and authorization boundary; it does **not** choose any concrete capability, prompt, agent or workflow. Real workflows are added through a **separate B6/on-demand capability onboarding lifecycle**.

## Current source findings (not S2S implementation)

- Existing human-user APIs: `POST /api/v1/agents/trigger` only persists an initial run; `POST /api/v1/workflows/{id}/execute` runs DSL inline; `GET /api/v1/agents/runs/{id}` is workspace-scoped but exposes verbose internal fields. None is an approved DubBridge S2S interface.
- Fenix persists agent runs in its own **SQLite** store; DubBridge keeps its own PostgreSQL database. No shared DB/JWT signing key.
- Historical review-count schemas `docs/contracts/dubbridge-fenix-v1-{request,run}.schema.json`, scripts `int_b1_contract_selftest.py`, `int_b1_contract_status_matrix.py` and `int_b1_fenix_live_probe.py` demonstrate **only the retired example**. They remain as historical design evidence, not normative blockers for this plan.
- Current normative candidate is the **capability-neutral envelope** in `docs/contracts/dubbridge-fenix-v1-envelope-{request,response}.schema.json`, with no concrete capability activated.

## Block/task plan (two levels only)

| Block | Tasks | Depends on | Acceptance evidence |
|---|---|---|---|
| **B0 — Rebaseline** | **B0.T1** direction DubBridge→Fenix; **B0.T2** quarantine reverse connector; **B0.T3** existing API/source inventory | — | Source and architectural boundary recorded — DONE |
| **B1 — Neutral integration contract** | **B1.T1** verify existing API shape (source review only); **B1.T2** security/principal & tenant/workspace contract; **B1.T3** versioned neutral envelope, idempotency, lifecycle, errors, capability registration semantics; **B1.T4** risk/ADR and readiness gate | B0 | Consistent machine-readable schemas; negative schema checks; threat/RRI forecast and security review. **No workflow runtime required** |
| **B2 — Fenix provider foundation** | **B2.A** isolated revocable S2S identity; **B2.B** fail-closed capability registry + tenant/policy checks (no built-in business workflow); **B2.C** SQLite durable operation/reservation/idempotency; **B2.D** status, errors, trace, safe projections and recovery; **B2.E** provider auth/transport/negative tests with unconfigured capability | B1 design & appropriate task-by-task security review | Real Fenix gateway; absent capability rejected; invalid tokens rejected; no run invented |
| **B3 — DubBridge consumer foundation** | **B3.T1** opt-in Rust S2S client/config; **B3.T2** map authorized local tenancy and capability requests without implementing domain workflow; **B3.T3** operation/status polling, retries and error mapping | B1 envelope, B2 API contract | Real Rust client tests; no business workflow/LLM dependency |
| **B4 — Infrastructure interoperability E2E** | **B4.T1** real DubBridge↔Fenix authenticated connectivity; **B4.T2** capability-not-configured path + invalid/revoked auth + cross-tenant negatives; **B4.T3** idempotency/status/outage/recovery traces | B2+B3 | Two real services exchanging HTTP, real stores/config, denial and recovery proven. **No real workflow required** |
| **B5 — Foundation closure** | **B5.T1** Go/Rust local QA + peer/security review; **B5.T2** evidence and operations runbook; **B5.T3** GO/NO-GO for foundation rollout | B4 | A secure dormant integration; no business outcome claimed |
| **B6 — Capabilities on demand** | **B6.T1** discover and prioritize a real user scenario; **B6.T2** register/version input/output schema + allowed runtime and tools; **B6.T3** implement/test actual agent/workflow and business E2E | B5 + explicit use-case decision | Only here is **real workflow execution** a required acceptance criterion |

B2 is deliberately split because the earlier **monolithic** hypothetical risk forecast (RRI 122, not official script run) exceeded the repository's decomposition gate. Calculate task-specific RRI before each implementation leaf; do not treat the forecast or a missing business workflow as a blanket implementation veto.

## B1 contract and safety

The generic envelope carries `contract_version`, `operation_id`, `capability_key`, `input_schema_version`, `trace_id` and minimal authorized source context. Payload is a bounded object whose content **MUST validate against the capability-specific schema when (and only when) that capability is registered**. The generic envelope alone never authorizes arbitrary payload execution. Result is null until a configured capability returns an allowlisted, schema-validated output. `run_id` is nullable until an actual Fenix run exists; `operation_id` is the stable polling key.

Security: service principal owned by Fenix, separate from human auth; revocable/rotatable credentials; bindings to appropriate workspace and allowed tenant/capabilities, fail-closed unknown mappings; no authority derived solely from source JSON; deny arbitrary Fenix CRM, tools, workflow management and DubBridge writes. Enforce request limits, budget controls and audit.

Lifecycle: unknown/disabled capability is rejected **before durable accepted execution** (e.g. HTTP 409/422 with `CAPABILITY_NOT_CONFIGURED`) and must not be recorded as a successful run. For a registered capability, reserve `operation_id` durably before executing, hash request for idempotent replay, and reconcile uncertain failures without duplicating effects.

## C1–C5 gate correction (2026-10-31)

The [earlier B1.C gate audit](../audit/dubbridge-fenix-b1c-gate-2026-10-31.md) is **historical, superseded for sequencing**. Its blocker requiring a live review-analysis workflow was based on the wrong phase boundary. Correct status:
- **C1** completed as source-only interface inventory; real workflow probe deferred to B6, not blocking B1/B2.
- **C2** monolithic RRI forecast is a decomposition warning, not an official measured RRI or a ban on B2 task-level design.
- **C3** ADR and independent security review still needed for relevant S2S work; this is a genuine security gate, unrelated to choosing a workflow.
- **C4** generic contract replaces the review-count-specific request/result; individual capability schemas and model results deferred to B6.
- **C5** **GO to scoped foundation preparation/design and test planning**; **NO-GO to deployment, merging security code, calling an unregistered workflow, or claiming a full integration E2E** until task-specific review/tests.

Historical docs and previous audit/test counts describe the older example and are retained for provenance only. No GitHub Actions activation or main merge is authorized.

## Next executable action

Start **B1.T2/T3 final contract security review and capability-neutral schema tests** (no model credentials), then **B2.A** as an independently reviewed, isolated authentication slice. Later capabilities are registered only when product usage requires them. Companion DubBridge [canonical plan](https://github.com/krukmat/dubbridge/blob/feature/int-v2-dubbridge-consumes-fenix/docs/plan/dubbridge-consumes-fenix-integration.md) and [tasks](../tasks/task_dubbridge_fenix_provider_int.md).


## B1 neutral contract checkpoint — 2026-10-31

[Exact-blob offline validation and source-backed security findings](../audit/dubbridge-fenix-b1-neutral-checkpoint-2026-10-31.md): **17/17 directed cases and 120/120 state cases PASS** for the *generic* capability-neutral envelope (not old review-specific schemas). No actual business workflow required or executed. [S2S ADR](../decisions/ADR-DB-FENIX-S2S-PROPOSED.md) remains **proposed**, with explicit middleware/JWT trust separation and bounds to review. [B2.A security leaves](../tasks/dubbridge-fenix-b2a-s2s-identity.md) are ready for separate RRI/peer/HITL gates. **B1 design work advanced; security acceptance and rollout remain NO-GO pending independent review.**


## B2.A1 readiness gate — 2026-10-31

B2.A owner approval recorded. Pure S2S [token primitive task](../tasks/task_dubbridge_fenix_b2a1_credential_readiness.md) scoped to two future Go files; **RRI 67 forecast (Complex), official script not run**. Independent peer readiness unavailable in the current container; **implementation is not started** under Fenix HITL/peer governance. The next work is peer/RRI approval of A1, not any real business workflow, and no S2S runtime/tenant/SQLite code is yet live.
