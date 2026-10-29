---
doc_type: assessment
id: DB-FENIX-B1-T1
title: "B1.T1 — Evidence-based selection of Fenix execution surface"
status: source_verified_runtime_unverified
created: 2026-10-29
---

# B1.T1 — Fenix agent / workflow execution API assessment

**Consumer:** DubBridge. **Provider:** Fenix. **Scope:** source audit on `feature/int-v2-dubbridge-consumes-fenix` only. **No running Fenix service, model provider, or real cross-system E2E has been exercised by this assessment.**

## Source-level inventory

| API | Evidence from source | Implication |
|---|---|---|
| `POST /api/v1/agents/trigger` | `internal/api/handlers/agent.go:TriggerAgent` calls `Orchestrator.TriggerAgent`; `internal/domain/agent/orchestrator.go:TriggerAgent` validates active definition, inserts `agent_run` and initial step, returns without calling `ExecuteAgent`. HTTP 201 and `{data:agentRunToResponse(run)}`. | **Do not equate 201 with actual agent execution.** Initial runtime status is `running` because of `newAgentRun`, not proof of a runner completing. Execution scheduling requires separate proof. |
| `POST /api/v1/workflows/{id}/execute` | `handlers/workflow.go:Execute` validates `workflows.execute`, workspace, active workflow with agent definition, then `executeDSLWorkflow` calls `DSLRunner.Run` inline and returns HTTP 200 `{data:{workflow_id,run}}`. | **Preferred internal engine** for v1; unlike `agents/trigger`, calls the actual runner. Synchronous HTTP boundary is **not** safe as an external long-running SLA: request timeout/ambiguous acceptance may leave an existing run. |
| `GET /api/v1/agents/runs/{id}` | `AgentHandler.GetAgentRun` calls `Orchestrator.GetAgentRun(ctx,workspaceID,runID)`; SQL predicate requires both run and workspace. HTTP 200 `{data:agentRunToResponse(run)}`, 404 for absent runs. | Useful source of persisted status; current projection includes `inputs`, `toolCalls`, `reasoningTrace` and `output` and is **not** acceptable unfiltered for an external service client. |
| `GET /api/v1/agents/runs` | Workspace-scoped listing exists. | Do **not** allow bulk run enumeration for DubBridge service. |
| `POST /api/v1/agents/runs/{id}/cancel` | Changes run status to failed, not necessarily a cancellation handshake with active computation. | Exclude from v1 until cancellation semantics are proven. |

## Security and trace findings

- Existing `/api/v1/*` routes use `internal/api/middleware/auth.go:AuthMiddleware` which invokes `pkg/auth.ParseJWT` and injects `UserID` and `WorkspaceID` from an end-user token. **There is no validated dedicated DubBridge service principal in these paths.**
- `pkg/auth/auth.go` uses `JWT_SECRET`, HS256 and user/workspace claims. It does not require explicit service audience, scoped operation or revocation on the code path inspected. **Do not mint a user JWT for DubBridge or share `JWT_SECRET`.**
- `WorkflowHandler.Execute` requires `workflows.execute` under the existing action-authorizer, and the workflow is ACTIVE and linked to an agent definition; these checks are useful but do not implement consumer-specific S2S security.
- `agentRunToResponse` exposes unminimized reasoning and tool calls; a separate provider projection must return `run_id, status, trace_id, sanitized result/error` only.
- `DSLRunner.Run` invokes `Orchestrator.TriggerAgent`, creates/persists a run and subsequently executes the DSL program. It has success, failed, pending/delegated branches: HTTP 200 cannot automatically be read as a terminal success.
- No idempotency check keyed to a DubBridge request ID was found in the inspected entrypoint; **double submit can create multiple runs**. Persist external `request_id` → Fenix `run_id` with a unique constraint before execution.
- Fenix generates a run trace ID internally; the previous external consumer correlation is not yet authenticated or made durable.

## Decision for design (not implementation approval)

Use the **workflow execution engine behind a new, narrow S2S facade** dedicated to DubBridge; do not forward service credentials to generic `/api/v1` user routes. V1 facade should accept a preconfigured allowlisted workflow, verify independently the DubBridge service identity + Fenix workspace binding, store a unique external operation key and return an accepted `run_id` with a separate read-only status/results endpoint.

**Async design caveat:** the currently inspected `workflows/{id}/execute` calls the DSL runner inline. A durable asynchronous dispatcher/job lifecycle, crash recovery and cancellation are **proposed work for B2**, not available by merely changing HTTP response status to 202. Avoid a cosmetic async wrapper without durable ownership.

## B1.T1 completion / remaining verification

**SOURCE REVIEW: PASS** for route/middleware/runner selection and identification of open gaps.

**LIVE BEHAVIOR GATE: NOT RUN** — must boot real Fenix using a real active workflow and supported model/runtime, record response latency, persisted initial/final status, pending/handoff cases, outage/restart behavior, agent/provider cost and tool invocation. Without it, B1.T1 is **partially complete**, not accepted as runtime-certified.

## Test matrix to execute in B2/B4

1. Active workflow+agent → actual execution, persisted `run_id`, audited status.
2. Inactive/not-linked or out-of-workspace workflow → denied, no run.
3. Ambiguous HTTP timeout after accepting command → querying the same idempotency key returns same run, never starts a second one.
4. Pending/delegated, failed, abstained, succeeded → distinct safe terminal/nonterminal projections.
5. Forbidden tools/side effects → denied by Fenix policy; no DubBridge domain mutation.
6. Revoked/expired/unscoped service credential and wrong workspace → denied before loading workflow.
7. Underlying Fenix app and model/service dependencies unavailable → explicit error, no fabricated analysis.

## Evidence boundary

This file is a source-inspection record. It is not an RRI receipt, peer-review PASS, live validation, security approval or cross-product test report.

## B1.T1 executable diagnostic (2026-10-29)

`scripts/int_b1_fenix_live_probe.py` is committed on the clean feature branch. It invokes the **real** existing Fenix `workflows/{id}/execute`, then reads the **real persisted** run by ID. It requires explicit `--ack-disposable-readonly` and an operator-supplied **disposable user test JWT** in `FENIX_B1_DISPOSABLE_TEST_JWT`. Only loopback hosts are accepted; it emits sanitized HTTP/status/trace evidence, not raw result or token. Operator must select a workflow proven read-only before running it.

```bash
export FENIX_B1_DISPOSABLE_TEST_JWT='<non-production local test-user token>'
python3 scripts/int_b1_fenix_live_probe.py \
  --base-url http://127.0.0.1:8080 \
  --workflow-id '<existing active READ-ONLY test workflow ID>' \
  --ack-disposable-readonly
unset FENIX_B1_DISPOSABLE_TEST_JWT
```

**This diagnostic is not a DubBridge S2S path.** No live Fenix service or authorized disposable test token is available in this execution environment, so the script has **not been executed against the actual runtime**, and B1.T1 remains open.

## Exact public-run status normalization observed in source

The inspected `PublicRunOutcome` maps `success → completed`, `partial → completed_with_warnings`, `abstained → abstained`, `failed → failed`, `accepted → running` or `awaiting_approval`, `delegated/escalated → handed_off`, and `rejected → denied_by_policy` when policy-related (otherwise failed). These statuses are not equivalent to a final business approval. A proposed S2S facade must explicitly map them to a versioned external enum and must preserve the ambiguity of pending/delegated executions; this mapping is **design input**, not implemented.
