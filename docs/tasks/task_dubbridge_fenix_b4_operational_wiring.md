---
doc_type: task
id: DB-FENIX-B4-WIRING
title: "B4 — DubBridge consumes Fenix: conditional-GO technical wiring"
status: approved_for_scoped_development_not_activation
owner: ChatGPT Cloud
date: 2026-11-13
depends_on: [DB-FENIX-INT-B2, DB-FENIX-INT-B3]
---

# B4 — Technical integration plan (GO CONDICIONAL)

**Owner decision (2026-11-13): GO CONDICIONAL to continue implementation**, not GO for deploying, enabling S2S, provisioning operational credentials, or closing B4/B5. Technically resolvable gaps and no structural architectural blocker known. Prior B4.Q formatting / QA evidence is **historical**, handled at B5/final certification; it must not consume or prevent source-only development of the next technical work. Do not treat pending verification as a confirmed security defect. This decision supersedes earlier unqualified roadmap NO-GO statements **for the decision to continue development only**. It does not change historical gate results.

**Direction:** DubBridge API → DubBridge-authenticated opt-in Rust client → Fenix dedicated service principal/middleware → Fenix SQLite-backed, default-deny capability admission → durable operation record. The business workflow is NOT chosen here; B6 on demand. Do not reuse user JWTs, enable arbitrary prompts, let caller select Fenix agent/DSL, or bypass DubBridge org/project authorization. No new branches, CI, or production provisioning.

## Existing technical assets — not yet connected

- **Fenix:** `internal/api/s2s_admission_composition.go` builds `NewDormantS2SAdmissionFromPolicies`; `internal/api/s2s_admission_http.go` defines a **detached**, unmounted HTTP handler. S2S principal/credential, default-deny policy, durable operation store and schema checks have source implementations. No normal-router registration.
- **DubBridge:** `crates/connectors/src/fenix_s2s.rs` defines opt-in HTTPS, opaque S2S bearer, stable `operation_id` from a **preexisting persisted work item**, bounded one-shot POST, strict admission acknowledgement. `apps/api/src/fenix_scope.rs` has DB-backed org/project scope authorizer. No application route or worker instantiates this client; the persistence of the calling work item and application handoff remain to be wired.
- **Real-world gaps:** no active Fenix S2S endpoint, no operator-approved nonproduction capability or credential, no live two-service HTTPS acceptance. C4 GET/status deferred; ambiguity requires fail-closed, explicit unresolved state without auto-retry.

## Next executable block — B4.W (maximum two task levels)

| Task | Responsible | Dependencies | Delivery / acceptance | Activation boundary |
|---|---|---|---|---|
| **B4.W1 Fenix dedicated routing, default-OFF** | **ChatGPT** | Existing B2.B4/B2.E code | **W1.1** review single canonical path/method, source handler construction and service auth chain; **W1.2** implement configurable nonproduction-only composition/wiring with explicit default-OFF and rejection when dependencies absent; negative route tests. Do not mount onto existing human-JWT route or expose a public default. | Candidate source only. No live port, route activation or operator grant. |
| **B4.W2 Deterministic diagnostic capability** | **ChatGPT** | W1, existing SQLite policy/operation source | **W2.1** define `integration.diagnostics` versioned test schema and isolated deterministic handler; **W2.2** connect through explicit operator-scoped capability configuration in a nonproduction test harness, with no model/ToolRegistry arbitrary execution and production default-deny unchanged. | No real capability enabled until separately approved nonproduction session. |
| **B4.W3 DubBridge consumer application wiring** | **ChatGPT** | Existing B3 connector, local policy | **W3.1** identify an existing, durable local work-item record for stable UUIDv5 operation key (no creating new random key on retry); **W3.2** build an opt-in application adapter using real `PgFenixScopeAuthorizer`, bounded request, injected credential provider and HTTPS transport, with disabled-by-default config; **W3.3** persist/request-correlate local operation and sanitized outcomes. | No worker/endpoint or secret switched on. DubBridge retains org/project permission authority. |
| **B4.W4 Safe nonproduction enablement package** | **ChatGPT** (prepare); **owner/operator** (approve/execute) | W1–W3 | **W4.1** propose reversible, TTL-bounded/nonproduction-only service identity, mapping and capability; **W4.2** documented startup/shutdown/rotate/revoke steps, log redaction and kill switch. | **HARD STOP: requires separate explicit owner authorization** before any token issue, DB provisioning, enabled route, capability or service starts. |
| **B4.T1–T3 Real infrastructure E2E** | **ChatGPT** (plan, analyze); **compatible runner** (execute) | W4 activation approval + environment | **T1** real authenticated DubBridge → Fenix HTTPS and bounded ack; **T2** deny wrong org/project/workspace, revoked/missing bearer, disabled/unknown capability, zero agent runs; **T3** repeat same `operation_id` (same payload), conflict on changed payload, process interruption/trace and no duplicate effects. | Nonproduction only. GO for E2E evidence after actual execution; not a business workflow proof. |
| **B4.W5 Ambiguous outcome policy, no C4 dependency** | **ChatGPT** | W3; final evidence depends on T3 | **W5.1** on timeout/connection uncertainty persist `indeterminate` locally and prevent automatic retries; **W5.2** define operator-mediated reconciliation using durable source evidence before resending; **W5.3** record GET/status C4 as a later optional enhancement. | No fabricated success/status/run_id. Do not silently auto-retry. |
| **B5 Foundation certification and rollout decision** | **ChatGPT** (own closeout); **independent reviewer/toolchains** (validate) | W1–W5 and B4.T1–T3 | One final compatible Go/Rust verification and independent S2S security review. Separate source-development GO, nonproduction E2E GO, and deployment/production GO. | No deployment/main merge without its own authorization. |

**Immediate work order:** **W1** (provider default-off route) and **W3.1** (investigate existing persistent DubBridge work item) can proceed in parallel; then **W2**, **W3.2/W3.3**, **W5 source**; prepare W4 and hold for owner approval; afterwards nonproduction T1–T3. C4 GET/status remains deferred. The real business capability and agent workflow remain exclusively B6.

## Decision gates and meaning

- `GO_CONDITIONAL_DEVELOPMENT`: **YES**, approved. ChatGPT may work autonomously on source-only B4.W1–W3/W5 in the existing feature branches.
- `GO_NONPRODUCTION_ACTIVATION`: **NOT AUTHORIZED** until explicit owner approval of W4, credential issuance, environment and specific deterministic diagnostic.
- `GO_B4_TECHNICALLY_DEMONSTRATED`: **NOT YET**; requires actual two-service HTTP, authorization negatives and idempotency evidence.
- `GO_DEPLOYMENT`: **NOT AUTHORIZED**; B5 external validation, security review, rollout approval remain separate. Prior Q5 / L1–L3 checks are neither invented PASS nor reopened as a development blocker.

**ChatGPT owns** technical planning, code changes, wiring and failure triage. A local executor is needed only for an environment/actual integration run, an independent reviewer only for independent verification. No user task other than consent for future activation.
