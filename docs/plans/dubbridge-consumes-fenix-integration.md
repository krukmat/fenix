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
