---
doc_type: plan
id: DB-FENIX-INT
title: "Fenix as agentic services provider for DubBridge"
status: proposed
created: 2026-10-29
tags: [fenix, dubbridge, provider, agent-runtime, integration]
---

# Fenix provider plan — DubBridge → Fenix

**Correction by product owner (2026-10-29): DubBridge consumes Fenix.** This document supersedes the reverse-direction `fenix-dubbridge-int-w0-integration.md` and `fenix-dubbridge-int-w1-real-integration.md` for the DubBridge integration initiative.

**Product ownership:** DubBridge selects domain use cases, authorizes source data and receives outcomes. Fenix executes governed agent/workflow capabilities as a **service provider**. DubBridge remains owner of review decisions, rights and P2P state. Fenix does not automatically approve/publish on DubBridge's behalf.

Companion primary consumer plan: [DubBridge integration plan](https://github.com/krukmat/dubbridge/blob/feature/int-v2-dubbridge-consumes-fenix/docs/plan/dubbridge-consumes-fenix-integration.md).

## As-built code evidence and open gap

Current Fenix `internal/api/routes.go` declares:
- `POST /api/v1/agents/trigger`; `GET /api/v1/agents/runs/{id}`;
- `POST /api/v1/workflows/{id}/execute`.
`internal/api/handlers/agent.go` accepts `agent_id`, `trigger_context` and `inputs`, and returns a run record; `internal/api/handlers/workflow.go` declares `ExecuteWorkflowRequest{trigger_context,inputs}`. The routes are under the existing `AuthMiddleware` for end-user JWTs.

**Unverified:** dedicated S2S identity, strict workspace/agent/workflow allowlist, run acceptance and idempotency semantics, proper async polling, durable external event mapping, and negative authorization matrix. These are design/implementation tasks, not existing claims.

## Goal

Enable a **DubBridge-owned** request to trigger one bounded, allowlisted Fenix workflow/agent; return execution reference/status, later retrieve minimized terminal result and correlate the source request. Fenix service auth must be Fenix-owned with minimal permissions, revocation and isolation. No DubBridge JWT secret shared with Fenix and no Fenix user password/token copied into DubBridge. No Fenix-to-DubBridge callbacks in the first vertical slice.

Suggested initial domain case: DubBridge creates an *authorized, privacy-minimized review backlog analysis request*, using facts read from its own DB. Fenix analyzes them and returns advisory output. DubBridge chooses whether/how to present it; no domain mutation.

## Two-level tasks (block → task)

| Block | Task leaves | Depends on | Status |
|---|---|---|---|
| **B0 — Baseline** | **B0.T1** confirm direction; **B0.T2** quarantine outgoing DubBridge capability; **B0.T3** inspect current API/runtime | — | Direction confirmed; inventory source-verified |
| **B1 — Contract** | **B1.T1** compare `agents/trigger` vs `workflows/{id}/execute`; **B1.T2** Fenix service identity/workspace binding and security ADR; **B1.T3** versioned request/status/result/idempotency contract | B0 | Pending |
| **B2 — Provider** | **B2.T1** auth middleware S2S principal; **B2.T2** allowlisted agent/workflow execution and minimum input schema; **B2.T3** status/result/trace and negative tests | B1 + repo review gates | Pending |
| **B3 — Consumer support** | **B3.T1** publish client contract for DubBridge; **B3.T2** verify error/cancellation/timeout semantics | B2 | Pending |
| **B4 — E2E** | **B4.T1** prove DubBridge-issued request reaches Fenix; **B4.T2** actual agent/workflow run, result and trace; **B4.T3** abuse and replay tests | B2 + DubBridge B3 | Pending |
| **B5 — Close** | **B5.T1** full Go QA and independent review; **B5.T2** evidence and GO/NO-GO | B4 | Pending |

## Security and governance rules

- Fenix validates a **Fenix-issued/recognized** service credential with restricted scopes, allowed workspace and allowlisted workflow/agent. Never treat DubBridge's JWT as Fenix authority.
- Existing Fenix user JWT API and CRM access are not automatically granted to the DubBridge service principal; use separate middleware and permission surface if needed.
- Fenix governs agent/tool execution, approvals and audit. DubBridge governs resource rights, project data and final business operations.
- Fenix must not receive unnecessary personal reviewer/asset details. Only minimized, schema-validated inputs and outputs cross the boundary.
- Retries after ambiguous acceptance must be idempotent; provide external operation correlation and terminal run state. Trace IDs are hints, not authorization.
- E2E requires the real Fenix runtime and DubBridge API/DB, **not mocks or standalone adapter unit tests**. Run Go/Rust local QA and required RRI/peer review before merge.
- GitHub Actions stay disabled until owner explicitly changes this policy.

## Prior reverse-direction branch work

`internal/infra/integration/dubbridge/*`, `internal/api/integration_runtime_dubbridge.go`, `cmd/dbqueue-read`, and startup wiring in `internal/api/integration_runtime.go` on the **legacy branch `feature/int-w1-dubbridge-readonly`** were produced under the mistaken Fenix → DubBridge direction. They are **quarantined**, not accepted for this initiative, and must not be merged into main as part of this plan. Their local tests prove nothing about the corrected direction. A future implementation branch should be based on main without these paths.

## Decision and checkpoint

**Planning rebaseline:** COMPLETE. **Implementation:** NOT STARTED for the corrected direction. **Integration/E2E:** NO-GO until B1-B4 deliver evidence. Start with **B1.T1** real API/run contract inspection.
