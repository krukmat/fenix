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

## CURRENT — W3.3/W5 source delivered, W4 activation not approved (2026-11-13)

**W3.3 DubBridge:** default-off `apps/api/src/fenix_dispatch.rs` explicitly composes trusted `PgFenixScopeAuthorizer`, immutable Fenix dispatch intent, exact-body SHA256 parity, UUIDv5 correlation, one CAS claim and a single outbound call. It is **not mounted in the normal DubBridge API/router or any worker**, and its SQL source remains outside executable migrations. **W5:** durable unresolved listing and manual stale sending→indeterminate; no retry, status GET or fabricated execution. [Manual operation guide](https://github.com/krukmat/dubbridge/blob/feature/int-v2-dubbridge-consumes-fenix/docs/handoffs/dubbridge-fenix-b4w5-manual-reconciliation-2026-11-13.md).

**W2.2 Fenix** adds a test-tagged harness method which calls persisted S2S admission before deterministic diagnosis; it does not expose that capability through normal HTTP. **W4.1/W4.2 preparation** is complete as a [nonproduction enable/stop proposal](../handoffs/dubbridge-fenix-b4w4-nonproduction-activation-proposal-2026-11-13.md), but tokens, operator policies, runtime registration, service routes, migration application and actual HTTPS remain **not authorized**. QA/run evidence on new source remains NOT_RUN. GO_CONDICIONAL only covers source development; real B4.T1–T3 and B5 await W4 activation and independent certification.

## B4.W2 + W3.2 implementation checkpoint — 2026-11-13

**Fenix W2.1/W2.2 source delivered:** `docs/contracts/fenix-b4w2-integration-diagnostics-v1.schema.json` defines the exact operator-schema-supported `integration.diagnostics` version-1 bounded input. `internal/api/s2s_nonproduction_diagnostic.go` and tagged test define a pure, deterministic, hashed diagnostic. It is not invoked by the existing admission path; no managed handler/capability registration, token, model or HTTP operation has been enabled. W4 operator test activation remains a separate decision.

**DubBridge W3.2.a/b source delivered:** [non-executable migration candidate](https://github.com/krukmat/dubbridge/blob/feature/int-v2-dubbridge-consumes-fenix/docs/proposals/0040_fenix_dispatch_intents.up.sql) in `docs/proposals`, not under executable `infra/migrations`, plus typed `crates/db/src/fenix_dispatch_intent_repo.rs` for immutable, tenant-scoped intent creation and single CAS claim. No DB migration application or live caller. A shared `crates/domain/src/fenix_operation_identity.rs` UUIDv5 helper is used by both Rust HTTPS connector and DB draft validation, preventing mismatched operation/trace IDs. `submitted` is only an HTTP admission, not an agent result. No automatic retry from ambiguous/sending/terminal states.

**Next W3.3** (NOT_STARTED): bind the final canonical wire digest to the persisted work ID in a two-phase prepare/build flow before one-shot send; then opt-in API composition. **Next W5** (NOT_STARTED): persist indeterminate/crashed attempts and require manual proof-based reconciliation, C4 GET/status deferred. **W4** activation remains unapproved. Go tagged/Cargo/PostgreSQL tests and independent review have NOT_RUN on these newly published candidates; certification is not claimed.

## B4.W1 / W3.1 execution checkpoint — 2026-11-13

**W1.1 source reviewed, W1.2 candidate authored:** Fenix has a standalone build-tagged `internal/api/s2s_nonproduction_router.go` and negative tagged test file. Only explicit `-tags=s2s_nonproduction` + `Enabled=true` + `Environment="integration-test"` + non-nil DB create a **separate isolated router** for `POST /integrations/dubbridge/operations`. It uses persisted service-identity/capability/operation admission and an IP rate limiter; neither `NewRouter` nor `NewServer` is changed or invokes the new factory. No actual endpoint enabled. The two candidate file copies passed standalone local `gofmt -d` parsing/format inspection; full Go compilation/conditional-tag tests and independent review **NOT_RUN**. W1 **source candidate**, not completed certification.

**W3.1 inventory done:** existing DubBridge `review_tasks.id`, `localization_generation_claims.generation_request_id` and `p2p_publications.id` are business-specific, not neutral persisted Fenix work IDs. [Inventory and immutable-ledger recommendation](https://github.com/krukmat/dubbridge/blob/feature/int-v2-dubbridge-consumes-fenix/docs/audit/dubbridge-fenix-b4w3-1-durable-intent-inventory-2026-11-13.md). **Next W3.2:** design/source a neutral durable `fenix_dispatch_intents` record bound to trusted DubBridge actor/org/project, storing stable UUIDv5-derived operation/correlation ID and immutable request digest *before* one-shot send. W3.3 API caller/wiring follows this ledger, not a new random UUID or repurposed business entity.

Current [authoritative B4.W tasks](../tasks/task_dubbridge_fenix_b4_operational_wiring.md) include this dependency correction. **GO CONDICIONAL unchanged:** source development allowed, no credential/route/capability activation, database migration execution, CI or deployment.

## CURRENT OWNER DECISION — GO CONDICIONAL, resume B4 technical plan (2026-11-13)

**Authoritative status:** **GO_CONDICIONAL for scoped technical development** of DubBridge → Fenix; **NOT** GO to enable S2S or close B4/B5. No structural architectural blocker is currently known. The former unconditional **NO-GO** was a **certification/activation verdict**, not a ban on continuing development. It remains true only for certifying or activating the current unconnected integration, not for working in the feature branches.

**Next phase is B4.W — operational wiring, source candidate first.** [Authoritative two-level tasks and responsibilities](../tasks/task_dubbridge_fenix_b4_operational_wiring.md). Critical path: **W1** default-OFF, dedicated Fenix route and auth chain → **W2** strictly deterministic, nonproduction diagnostic capability; parallel **W3** existing durable DubBridge work-item/UUIDv5 linkage plus opt-in application caller → **W5** fail-closed ambiguous-result handling without automatic retry. **W4** owner-approved nonproduction credential/capability/endpoint enablement is a separate hard stop → B4.T1–T3 authentic HTTPS, negative isolation and idempotency proof → B5 certification and deployment decision.

**C4 GET/status remains deferred and nonblocking for source-only B4.W;** ambiguity is recorded as indeterminate and reconciled manually from persistent evidence, never silently retried or declared success. No real business workflow/agent/LLM is necessary before B6. No CI, new branches, migration application, live route, credential provisioning, production activation, deployment or main merge is implied by this GO. ChatGPT owns implementation and planning; the user only approves future enablement.

**Historical Q5 formatting and independent review failures remain recorded below, but are not the critical path for unactivated source development.** No claims of passed QA or production readiness. Older plan sections describing B2–B5 as not implemented or saying next action is B1 are archival snapshots and are superseded by this checkpoint.

## B4.Q5 — Actual frozen-pair local gate evidence and targeted repair (2026-11-13)

**Owner's local evidence for frozen pair:** Fenix `d07adb66`, DubBridge `bf36a9d3`. Source was verified in clean detached worktrees (original checkout mismatch is a procedural L0 FAIL, not a test-source mismatch). Actual L1 Python SQLite **23 PASS**, Go focused **PASS**, vet **PASS**; L1 prepush **FAIL** at 11-file `fmt-check` with later subgates NOT_RUN. L2 Cargo check **PASS**, tests **321 PASS / 1 ignored**, global fmt **FAIL** (connector + two pre-existing unrelated workspace/gateway files), strict Clippy **FAIL** for 5 `manual_async_fn`. L3 reviewer output unreliable/false positives, independent security approval **absent**; no confirmed H/M vulnerability established.

**ChatGPT Q5 fix:** Rust connector's 5 manual async implementations converted to `async fn` in [DubBridge `2fc5b2b`](https://github.com/krukmat/dubbridge/commit/2fc5b2b7a38222a9c9a65c91c35c45de30207fd5), preserving `impl Future + Send` trait signatures. **New code not cargo-retested**, and Go formatting is **still unresolved**. [Full gate triage](https://github.com/krukmat/fenix/blob/feature/int-v2-dubbridge-consumes-fenix/docs/audit/dubbridge-fenix-b4q-q5-gate-triage-2026-11-13.md). Next run only affected format/prepush and Cargo check/test/clippy on updated SHAs, followed by an independent review of the **complete** integration diff. Reuse SQLite and already-green Go focused/vet evidence where unchanged. **NO overall QA PASS** and no route/credential/capability/CI/deploy/merge activation.

## B4.Q C1–C4 Cloud delivery — local certification remains last (2026-11-13)

**Published C1 findings/fixes:** Go API router test helper now receives `testing.T`; Go forged-workspace negative test correctly mutates the B2.E `"input":{"v":1}` fixture, and an explicit no-op mutation guard prevents recurrence. Rust `reqwest::Response::chunk` API checked; no extra stream feature needed, no dependency churn.

**C2:** confirmed **5/5 exact Git blob identities** for Fenix SQLite 040 migration, DubBridge Node B4 checker and all three pinned schemas against prior executed evidence. Earlier 24/24 actual isolated SQLite + 29/29 actual Node contract checks remain applicable to unchanged inputs; **no duplicate execution in this checkpoint** and the new Fenix Python wrapper **NOT_RUN**. **C3:** 13/13 targeted source fixture/mutation checks PASS (not Go/Rust QA). **C4:** [preflight and explicit verification boundary](https://github.com/krukmat/fenix/blob/feature/int-v2-dubbridge-consumes-fenix/docs/audit/dubbridge-fenix-b4q-cloud-first-preflight-2026-11-13.md), [single deferred local batch handoff](https://github.com/krukmat/fenix/blob/feature/int-v2-dubbridge-consumes-fenix/docs/handoffs/dubbridge-fenix-b4q-final-local-batch-2026-11-13.md), and exact final source SHA freeze in Fenix issue #10/DubBridge issue #13 once documentation is committed.

**Status:** C1–C4 Cloud authoring delivered; Q4 formal Go1.25.13/Cargo/independent review and real authenticated E2E remain **NOT_RUN**. No local agents, CI, route, operator grants, service tokens, deployment, status GET or merge. User explicitly requested **all local work LAST**. Q5 remediation remains ChatGPT-owned after consolidated verification.

## B4.Q new owner execution order — Cloud-first, local-final (2026-11-13)

**Owner instruction (supersedes earlier Q4/Q5 ordering):** prioritize **all ChatGPT Cloud work**, defer **all local/toolchain/reviewer execution** until the last phase. Authoritative two-level [B4.Q task ledger](https://github.com/krukmat/fenix/blob/feature/int-v2-dubbridge-consumes-fenix/docs/tasks/task_dubbridge_fenix_b4q_cloud_first_execution.md): **C1** review/fix changed Go/Rust source and compile-risk issues; **C2** run committed Python SQLite 038–040 and Node pinned-contract scripts from Cloud; **C3** reconcile cross-repo negative contracts and test source; **C4** audit scoped diffs, freeze both SHA references and prepare **one** local handoff. Completed B4.Q1–Q3 evidence remains historical and must not be needlessly rerun.

Only after Cloud C1–C4 closure: **L1** Fenix Go1.25.13 build/tests/race/vet/prepush and full migrations, **L2** Rust fmt/check/tests/clippy, **L3** independent focused code/security review; run as **one compact local session/batch**, no parallel code implementation by local agents and no repeated unmodified suites. Then ChatGPT Q5 triages findings. Any source changes invalidate affected evidence only. **No** local-agent requests before C4 freeze; no CI, live route/credential/capability, paid infrastructure, status GET, deployment or merge. Source candidate status is unchanged: Q4/Q5 QA NOT_RUN.

## B4.Q1–Q3 — Executed Cloud verification checkpoint (2026-11-11)

**Execution and evidence:** [B4.Q audit](../audit/dubbridge-fenix-b4q-cloud-validation-2026-11-11.md). Q1: 21/21 static cross-repo Go/Rust checks PASS; isolated Go1.23.2 stdlib service-token parser regression PASS. Q2: actual SQLite 3.46.1 **migration 040** constraints **22/22 PASS** against minimal parent tables and two-connection transaction/disable ordering **2/2 PASS**; migration exact Git blob `ad31aa7239243c350fd43da02771a8c8637cd1c7`. Q3: exact DubBridge Node script and pinned Fenix schemas actually executed **29/29 PASS**, with provider schema blobs still matching current Fenix GitHub source.

**Corrections:** DubBridge Rust S2S token input now enforces Fenix's canonical 32-byte unpadded Base64URL final bits; Node contract checker now rejects unsupported future schema keywords/types rather than ignoring them. The new Fenix stdlib [SQLite smoke wrapper](../../scripts/int_b2b4_capability_sqlite_smoke.py) is committed but **not executed itself** in this Cloud run; the 24 SQLite checks above were executed with separate isolated harnesses. No local agents/tokens.

**Remaining formal gates:** full Go1.25.13 project compile/fmt/race/vet/prepush, full Fenix 001–040 migrations/modernc test, Rust cargo suite/clippy, independent code/security review, reviewed operator provisioning and authenticated two-service HTTP E2E are **NOT_RUN**. No route/credential/capability activation, CI, deployment, GET/status C4 or merge. B4.Q4/Q5 are pending; B4 is not production-certified.

## B2.B4 durable operator capability policy — Cloud candidate (2026-11-11)

**Approved block delivered by ChatGPT:** [B2.B4.1–B2.B4.5 task ledger](../tasks/task_dubbridge_fenix_b2b_registry.md). Inert source in `internal/domain/integration/s2s_capability_{policy_store,schema,registry}.go`, new migration `040_s2s_capability_policy.up.sql` and Go tests. `internal/api/s2s_policy_operator.go` reuses existing `PolicyEngine.CheckActionPermission` under an **active same-workspace Fenix operator** for `s2s_capabilities:manage`, with **no default grant or management route**. No independent ToolRegistry/executor/approval bypass is created: admission policies control eligibility, not runtime execution rights.

Policies are durable, audited, **disabled at provision**, immutable per schema version, and prevent multiple simultaneously enabled versions for one principal/workspace/capability. Operator changes require explicit policy authorization. Managed admission uses a read-only policy store and a strict bounded input-schema validator, not a permissive stub. C2 now binds the schema digest to its config fingerprint and **rechecks enabled principal/capability/version/schema under its SQLite `BEGIN IMMEDIATE` reservation transaction**, including replay. Missing/disabled/changed policies fail closed.

**Cloud check:** 46/46 read-only source assertions PASS. **Actual Go1.25.13/migrations/SQLite tests/race/vet/prepush/independent security review NOT_RUN**; B2.B4 is **published source candidate only**, not production-approved or even Go QA certified. Migration 040 is committed only in feature branch, **not applied to production**. No operator token, role grant, real capability, HTTP route, CI, deploy or main merge. C4 GET/status deferred; T4 #8 release-security separate. User local-agent tokens remain unavailable and none are requested.

## B4.T1.1–T1.3 detached HTTP adapter — Cloud source checkpoint, 2026-11-11

**ChatGPT-owned delivery:** [B4.T1 task](../tasks/task_dubbridge_fenix_b4_t1_detached_http_admission.md) now contains the unmounted HTTP handler `internal/api/s2s_admission_http.go`, persisted-store constructor `s2s_admission_composition.go`, and negative `httptest` and SQLite test sources. `S2SAdmission.AdmitWithIdempotencyKey` authenticates persisted Fenix identity first, validates B1 JSON and enforces equality between the `Idempotency-Key` header and the B1 operation ID before C2's atomic reservation. Success and error responses reuse the existing B4 wire projectors, with no principal/config/hash/status leakage.

**Source-level checks:** 27/27 read-only cross-repo assertions PASS, including matching DubBridge path/header/bounds, canonical token, single headers, opaque failure projection, timeout, and complete absence of registration in `internal/api/routes.go`, `internal/server/server.go` and `cmd/fenix/main.go`. Authored tests exercise real migrated SQLite credentials/reservation/replay/conflict/denial, plus malformed HTTP framing, oversize chunked body and no agent dispatch. **No Go 1.25.13 compilation, gofmt, race, vet, prepush or independent security/code review was executed in Cloud**. Source candidate only; not an approved or operational endpoint.

**Activation remains prohibited:** detached constructor is never invoked by the normal production router. No S2S URL, public HTTP request, operator token/capability, default input validator, GET/status C4, CI, new migration, deployment or merge. B2.B4 operator authorization, T4 security, Go/Rust QA and explicit owner activation remain separate.

## B4 cloud-only admission-wire interoperability — 2026-11-11

**Actual provider-side source:** `docs/contracts/dubbridge-fenix-v1-admission-{ack,error}.schema.json` (two **new proposed transport** schemas, separate from the historical B1 lifecycle/preacceptance error proposals); `internal/domain/integration/s2s_admission_wire.go` (typed 202=new/200=replayed acknowledgement, static 400 invalid / 403 denied / 409 conflict / 503 unavailable rejection); `s2s_admission_wire_test.go` and `s2s_b4_inert_interop_test.go` (source-authored serialization, SQLite reservation/replay/denial/no-run negative checks). Last code/test SHA `7998231fb606e9dd92a22f7ceba9792dcc734675`.

**Cross-repo verification:** DubBridge [B4 audit](https://github.com/krukmat/dubbridge/blob/feature/int-v2-dubbridge-consumes-fenix/docs/audit/dubbridge-fenix-b4-offline-interop-2026-11-11.md) records **28/28 executed offline JS contract checks** against exact Fenix schema blobs and **24/24 static Go/Rust/contract predicates**, not genuine HTTP interop. DubBridge Rust now validates the proposed denial status/code and 200/replayed vs 202/new consistency. **Actual Go1.25 QA, Rust QA, Node CLI script and independent security review NOT_RUN.**

**No mounted Fenix S2S route or operator credential/capability; B4.T1 real authenticated E2E remains BLOCKED.** No GET/status C4, agent/worker, generated run ID, CI, deployment or production migration. B2.B4 audited operator capability and T4 independent security review still separate. No local-agent execution is requested.

## B2.E1–E5 Cloud implementation checkpoint — 2026-11-11

**Implementation scope:** ChatGPT Cloud authored an **inert**, in-process S2S admission path in `internal/domain/integration` on `feature/int-v2-dubbridge-consumes-fenix`: strict versioned B1 request validation (E1); verified Fenix principal + active registry + mandatory capability-input validator + C2 reserve-only adaptor (E2); public-safe error projection without HTTP (E3); real credential/SQLite test cases authored (E4); and a [21/21 source-only consistency audit](../audit/dubbridge-fenix-b2e-cloud-source-review-2026-11-11.md) (E5). Task tracker: [B2.E parent](../tasks/task_dubbridge_fenix_b2e_inert_admission.md); [GitHub issue #10](https://github.com/krukmat/fenix/issues/10).

**Code/test snapshot:** `b9549ca6ad05f602a49457ae23292cc3f1a7ef2c` (later documentation commits only). **Source committed, Go 1.25.13 focused/race/vet/prepush and independent peer gates NOT_RUN**. Cloud does not have required Go toolchain/dependencies; local tokens are exhausted, therefore no further local execution is requested. This remains a **source candidate**, not a production or QA PASS. B2.B4 must provide operator-controlled capability-specific schemas before any live ingress can be enabled; nil validators deny by default.

**C4 GET/status stays deferred; T4 security #8 stays separate.** No HTTP route, public transport, capability activation, agent workflow execution, CI, credential, production migration, deployment or main merge was performed.

## Current Cloud execution checkpoint — 2026-11-10 (no local tokens)

**Latest published source:** `feature/int-v2-dubbridge-consumes-fenix`, commit [`e53fad7`](https://github.com/krukmat/fenix/commit/e53fad7f54fba46af00d9f2b7e6e287b217479ad), contains the C1–C3 fixes previously tested as a local overlay and the server race-test adjustment. The [final local QA report](https://github.com/krukmat/fenix/issues/9#issuecomment-6777960499) records focused tests, race ×3, vet, full suite and prepush **PASS on the effective local tree**, with traceability/govulncheck separately SKIPPED. The critical independent review was recorded as `peer-codex`; fallback-route provenance and a clean-clone published-SHA replay have not been established. **Do not promote this to formal peer-policy closure or production approval.**

**Current operator constraint:** local-token budget exhausted. ChatGPT Cloud proceeds autonomously with deliverables that can be committed directly; no more local-agent prompts, tests, reviewer sessions or token-dependent gates are requested in this stage.

**Next Cloud block: [B2.D non-status](../tasks/task_dubbridge_fenix_b2d_nonstatus.md)**. Three two-level leaves have been created: [D1 admission-denial/error contract](../tasks/task_dubbridge_fenix_b2d1_denial_contract.md), [D2 trace/audit/redaction](../tasks/task_dubbridge_fenix_b2d2_trace_audit.md), [D3 future ingress preflight and negative cases](../tasks/task_dubbridge_fenix_b2d3_admission_preflight.md). These are **Cloud design deliverables**, not active HTTP implementations, GET/status work, new credentials or runtime QA. C4 remains deferred; T4 security review stays separate for release.

**B2.D Cloud design verification:** [source-backed gap audit](../audit/dubbridge-fenix-b2d-nonstatus-source-audit-2026-11-10.md) completed **12/12 static source-contract checks**, identifying seven documented boundaries D-G1..D-G7, with no HTTP/Go execution claim. Contract-only D1–D3 tasks are delivered; future transport implementation remains separately scoped. No local agents requested.

## Current C1–C3 source work completed in Cloud — 2026-11-10

**Scope:** Only C1–C3 were changed. This is an unvalidated feature-branch candidate and **not** a code/release PASS.

| Task | Concrete Cloud change | Evidence still needed |
|---|---|---|
| C1 | Tightened embedded SQLite 039 checks so `accepted` cannot hold a fabricated `run_id`, and run-required states cannot store an empty `run_id`; added two negative migration assertions | Real Go 1.25.13 modernc, readiness and independent code review |
| C2 | Corrected replay equality to compare immutable `trace_id`; added mismatched-trace conflict test even if the digest is incorrectly reused | Real Go 1.25.13 focused/race, readiness and independent code review |
| C3 | Added revoked-principal/expired-lease reconciliation regression: `indeterminate`, cleared lease, lookup denied | Real Go 1.25.13 focused/race, readiness and independent code review |

**Latest C1/C2 hardening:** C1's migration 039 now rejects worker leases on *all terminal statuses*, plus Go migration negative assertions. The reduced in-memory sqlite3 terminal/lease invariant matrix passed **14/14**, not an embedded migration or Go1.25.13 run. C2's canonical input validator now rejects malformed UTF-8 before JSON normalization, with regression coverage to prevent digest aliasing. These changes remain unverified Go source candidates.\n\n**Additional C2/C3 hardening:** C2 now has a test proving caller-forged `ConfigSHA256` is ignored on replay (the existing registry-derived fingerprint remains authoritative). C3's scoped `MarkIndeterminate` now checks the persisted active principal atomically in its UPDATE, closing a lookup/update revocation window; revoked-worker denial and Fenix-owned reconciliation are covered by the regression test. A narrow in-memory SQLite check of that predicate passed active and revoked cases, **not** a real Go QA claim.

**Additional coverage:** C2 explicitly tests denial of both an old replay and a new operation ID when the persisted principal is revoked; C3 explicitly tests denial of a lease when the registry capability is disabled after reservation. Both are Go regression candidates only, with no execution evidence yet.\n\nThe current source/test delta is bounded to seven paths: `internal/infra/sqlite/migrations/039_service_operation.up.sql`, `internal/infra/sqlite/service_operation_migration_test.go`, `internal/domain/integration/service_operation_store.go`, `internal/domain/integration/service_operation_digest.go`, `internal/domain/integration/service_operation_store_test.go`, `internal/domain/integration/service_operation_recovery.go`, `internal/domain/integration/service_operation_recovery_test.go`. **No C4 source, production HTTP route, CI, credential, operator action or merge**. The existing C1 readiness `needs_changes` remains pending re-review; the work performed does not waive it.

## C1–C3 independent QA remediation checkpoint — 2026-11-10

**Source-only status:** prior independent reviews on `e8c307c7` returned `needs_changes` and Go QA FAIL; this **is not closed**. ChatGPT published candidate remedies against confirmed F1 (SQL NULL-CHECK lease without expiry), N1 (lone UTF-16 surrogate digest collisions), N2 (mutable terminal run metadata), N3 (deletion/reuse of durable operation ID), plus fmt/vet/complexity/lint/wrapcheck hardening. See the [single current fixpack](../handoffs/dubbridge-fenix-b2c-fixpack-2026-11-10.md) for paths and execution gates. The registry-disable/lease race N4 is documented as a future runtime/dispatch design concern, **not an activation approval**. C4 remains deferred/nonblocking.

**Next gate:** rerun C1–C3 readiness (with full evidence), actual Go 1.25.13/modernc integration, race, vet, prepush and exact-source independent review. **All new source fixes remain unverified by those gates.** Prior synthetic SQLite and isolated Go scanner checks are only diagnostics; global documentation debt and T4 security issue #8 remain separate. No CI, route, credential, production migration, deployment or merge.

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
| **B2 — Fenix provider foundation** | **B2.A** isolated revocable S2S identity; **B2.B** fail-closed capability registry + tenant/policy checks (no built-in business workflow); **B2.C** SQLite durable operation/reservation/idempotency; **B2.D** non-status denial/error, trace/redaction and admission-preflight contract (GET/status deferred separately); **B2.E** inert authenticated admission and bounded negative test source (transport activation remains separately gated) | B1 design & appropriate task-by-task security review | Real Fenix gateway; absent capability rejected; invalid tokens rejected; no run invented |
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
