---
doc_type: audit
id: DUBBRIDGE-FENIX-B1C-2026-10-31
title: "B1.C1–C5 execution checkpoint and risk gate"
status: superseded_for_sequencing
d> **SUPERSEDED SEQUENCING DECISION (owner clarification 2026-10-31):** The NO-GO below is historical and **must not block B1–B5 because a real workflow is undefined**. Specific workflow/agent tests now belong to B6 (usage-driven capability onboarding). Risk forecast 122 remains a nonofficial warning about a monolithic B2 implementation; independent security review and per-leaf RRI still apply. Current authoritative plan: [neutral Fenix provider plan](../plans/dubbridge-consumes-fenix-integration.md). Historical evidence retained unchanged below.

ate: 2026-10-31
---

# B1.C checkpoint — DubBridge consumes Fenix

**Direction:** DubBridge calls Fenix's agents/workflow provider. Branch `feature/int-v2-dubbridge-consumes-fenix`. Scope: contract, architecture, risk review. This is **NOT** a completed security review, S2S implementation or live E2E certification.

## C1 — Actual Fenix runtime diagnostic: BLOCKED

- Fenix's existing `POST /api/v1/workflows/{id}/execute` is an inline DSL run behind a **human JWT**; this cannot be used as a service credential. Its old `POST /api/v1/agents/trigger` only initially persists a run.
- The available tool container has no checkout of the Fenix app, running Fenix service, authorized test JWT, active disposable read-only workflow or reachable authenticated model configuration. The locally listening `127.0.0.1:8080` responds with a 404 and `server: uvicorn`, not Fenix. **It would be a false positive to treat an open port as the Fenix runtime.**
- A guarded local real-workflow probe exists at `scripts/int_b1_fenix_live_probe.py` (explicit disposable-readonly ack; loopback only; proxy disabled). No genuine execution or persisted run has been observed in this session.
- Operator handoff: in a private Fenix checkout, launch the app and an authorized **read-only** active test workflow, export a disposable **non-production** test JWT, then run the guarded probe. Record sanitized output/commit and the run's terminal/pending status. Do not put the bearer in GitHub or use a DubBridge user JWT as a Fenix credential.

## C2 — Risk/RRI: PRELIMINARY HIGH-RISK SCOPE ASSESSMENT

- Repository policy and deterministic implementation: `docs/policies/RRI_POLICY.md`, `scripts/rri.py`. Cross-checked the formula/penalty tables against those sources.
- The next **hypothetical** B2 S2S auth+workflow+transaction implementation is a multi-module *forecast*, **not yet a measured implementation diff**. Example candidate touch set (not committed changes): `internal/api/middleware/dubbridge_s2s.go`, `internal/api/handlers/dubbridge_s2s.go`, `internal/api/routes.go`, `internal/domain/auth/service_principal.go`, `internal/domain/workflow/dubbridge_operation.go`, `internal/domain/agent/dsl_runner.go`, `internal/domain/tool/registry.go`, `internal/infra/sqlite/migrations/<next_number>_dubbridge_operations.up.sql`.
- **Eight candidate files → F=3 (forecast)**. No measured target implementation cyclomatic complexity; use a conservative **C=3 pre-score, low-confidence bump to C=4**. Judgment scores: `D=5,T=4,A=2,K=5,P=5,X=4`. Security impact, current lack of dedicated S2S tests and distributed failure modes justify the selected anchors.
- Formula estimate: `100×((.18×4 + .12×3 + .15×5 + .15×4 + .12×2 + .12×5 + .10×5 + .06×4)/5) = 80` (rounded).
- Penalties according to `scripts/rri.py`: `complex_and_domain +10`, `no_tests_high_impact +10`, `auth_security +10` (touching domain auth), manually signaled `arch_decision +12`. **Forecast RRI=122, Exces​sive (>100)**; requires decomposition, security ADR/review and a new official RRI result on measured actual paths and CC before code implementation.
- **This numeric result is an independent reproduction of the repo's published formula using forecast inputs, NOT an execution of `scripts/rri.py` and NOT a signed RRI receipt.** A project checkout and real implementation diff are needed for an official post-code receipt. In particular do not treat F/C forecasts as measurements.
- Run from real checkout for task-presentation forecast:

```bash
python3 scripts/rri.py --platform fenix --C 3 --low-confidence C \
  --touches internal/api/middleware/dubbridge_s2s.go \
  --touches internal/api/handlers/dubbridge_s2s.go \
  --touches internal/api/routes.go \
  --touches internal/domain/auth/service_principal.go \
  --touches internal/domain/workflow/dubbridge_operation.go \
  --touches internal/domain/agent/dsl_runner.go \
  --touches internal/domain/tool/registry.go \
  --touches 'internal/infra/sqlite/migrations/<next_number>_dubbridge_operations.up.sql' \
  --D 5 --T 4 --A 2 --K 5 --P 5 --X 4 \
  --penalty arch_decision --json
```

## C3 — Security architecture self-review: FINISHED AS FINDINGS, EXTERNAL REVIEW PENDING

[Draft S2S ADR](../decisions/ADR-DB-FENIX-S2S-PROPOSED.md) records seven specific HIGH/MEDIUM-HIGH threats and their proposed gates: human/service token confusion, cross-org/project/workspace context leakage, tool privilege expansion, raw model tool/reasoning leakage, duplicate execution after timeout/crash, request replay, and operational abuse. **It is not accepted.** No independent qualified security review, token issuance/revocation, database migration or exploit/negative tests have been performed.

## C4 — Versioned contract reconciliation: CANDIDATE UPDATED

- **Discovered design inconsistency:** `202 Accepted` after durable reservation previously required an already allocated `run_id`, but the existing Fenix `DSLRunner` allocates its real `agent_run` later. If Fenix crashes between reservation and execution, there may be **no run ID** to return or poll.
- Fixed proposed [run response schema](../contracts/dubbridge-fenix-v1-run.schema.json): `run_id` nullable during `accepted` and pre-run denial/failure; required for `running`, `awaiting_approval`, `handed_off`, `abstained`, `succeeded`, `completed_with_warnings`. Preserve exclusive success/error/result semantics.
- **Trace consistency:** the response's required trace is an **operation-level** identifier generated at reservation; Fenix later associates the actual agent run's own trace ID. This avoids requiring `agent_run.trace_id` before the run exists.
- The **stable query key becomes `operation_id`**, so the proposed read endpoint is `GET /integrations/dubbridge/operations/{operation_id}`. POST stays `POST /integrations/dubbridge/runs`. The operation key is principal-scoped, idempotent and digest-bound; any real Fenix run ID is linked once allocated.
- Extended directed selftest for null run ID acceptance, pre-run denial and rejection of terminal/pending states without a run. Extended offline matrix from 60 to 120 status×run-allocation×result×error cases (expected **14 allowed, 106 rejected**). These are **committed test definitions**; the tool container performed a smaller **local independent schema-pattern sanity check** (3 positive, 5 negative and a 120-combination matrix = 14/106). It did **not reconstruct and execute exact latest GitHub test files**. Prior 18/18 and 60/60 receipts apply to the earlier scheme only; **do not carry them forward as fresh exact-blob PASS**.
- Consumer [DubBridge candidate wire contract](https://github.com/krukmat/dubbridge/blob/feature/int-v2-dubbridge-consumes-fenix/docs/plan/dubbridge-fenix-b1-contract-v1-proposal.md) and the proposed provider ADR are updated. A 202/GET API has not been implemented.

## C5 — Explicit gate decision: NO-GO to B2 runtime integration

Rationale: C1 actual real Fenix run blocked; C2 official RRI/peer receipt pending; C3 independent security review pending; C4 candidate schema implementation not frozen. No genuine JWT S2S principal, runtime, safe result projection, idempotency DB transaction or E2E exists. **This checkpoint is a correct no-go**, not a failure to write documents.

### Required decomposition before B2 approval

1. **B2.A:** isolated principal secret/rotation/revocation and narrow S2S middleware, no workflow invocation. SQLite persistence and negative auth tests.
2. **B2.B:** immutable workflow version and effective tool/side-effect policy gate, workspace and DubBridge org/project isolation.
3. **B2.C:** durable operation reservation/fingerprint/lease with `operation_id` polling, crash/replay characterization.
4. **B2.D:** real DSL execution dispatch and reconciliation; prove no duplicate side effects before retries.
5. **B2.E:** sanitized result projection, real runtime negative tests, combined two-product E2E with DubBridge.

Each leaf needs its **own** measured RRI, explicit acceptance tests, independent review and owner governance. No GitHub Actions activation or main merge. No credential/payment/hosting changes.
