---
doc_type: plan
title: "INT-W1 — Real read-only Fenix to DubBridge integration"
status: in_progress
created: 2026-10-28
tags: [fenix, dubbridge, toolregistry, security, real-e2e]
---

# INT-W1 — Real Fenix to DubBridge integration

**Source baseline:** [INT-W0](fenix-dubbridge-int-w0-integration.md). **Branch:** `feature/int-w1-dubbridge-readonly`.
**DubBridge authority:** [DB-INT-W1](https://github.com/krukmat/dubbridge/blob/feature/int-w1-fenix-service-identity/docs/plan/fenix-integration-int-w1-identity.md).

## Goal and explicit constraints

Genuine Fenix agent -> Blackboard -> governed ToolRegistry -> dedicated DubBridge **read-only** API -> PostgreSQL review queue -> minimal agent response, preserving authentication, project binding and execution audit. No mocked API, synthetic response, broad user JWT, direct DB coupling, cloud deployment or GitHub Actions run is accepted as E2E evidence.

The owner confirmed GitHub Actions disabled on 2026-10-28; connector exposes no read access to the repository Actions permissions setting. Prior DubBridge queue runs may remain. The owner requested autonomous work and tasks with at most two levels. Keep development isolated from main and follow repo RRI/reviewer/local QA before pushing runtime code.

## Two-level execution plan (parent block and task leaves)

| Block | Task leaves (no third level) | Depends on | Current state |
|---|---|---|---|
| **B0 — Cost & control** | **B0.T1** confirm owner Actions pause; **B0.T2** check no new runs | — | Owner confirmed; API run tally stable; settings unverifiable |
| **B1 — DubBridge identity** | **B1.T1** security ADR/least privilege; **B1.T2** principal persistence and revocation; **B1.T3** operator issuance and dedicated read route | B0 | B1.T1 proposed in DubBridge; runtime blocked |
| **B2 — Correlation** | **B2.T1** header validation; **B2.T2** actual DubBridge audit propagation | B1 | B2.T1 client-side code staged; B2.T2 not started |
| **B3 — Fenix capability** | **B3.T1** real GET transport and safe DTO; **B3.T2** ToolExecutor and permission/schema registration; **B3.T3** opt-in config and Blackboard validation | B1 | B3.T1 local unit QA PASS; B3.T2 staged, application build not verified |
| **B4 — Cross-product E2E** | **B4.T1** genuine DB/org/project/review seed; **B4.T2** allowed call; **B4.T3** 401/403/cross-tenant/expiry/revocation/mutation denials; **B4.T4** outage/retry/trace | B1-B3 | Not run |
| **B5 — Closure** | **B5.T1** independent review and local QA; **B5.T2** document exact commands/evidence; **B5.T3** GO/NO-GO | B4 | Not started |

Status must be reported as *staged*, *repository committed*, *locally unit-verified*, or *E2E verified*; these states are not interchangeable. A block is closed only when all task leaves meet its exit criteria.

## Wire contract

Proposed capability `dubbridge.review_queue.list` (v1, `SideEffectRead`, `ApprovalRequired=false`, maximum 2 attempts), input:
```json
{"contract_version":"1","org_id":"<uuid>","project_id":"<uuid>"}
```
Destination org/project are fixed from operator-controlled Fenix config; a free agent-selected UUID cannot enlarge authority. Fenix must ensure the workspace itself is entitled to that destination project. The endpoint targeted by the staged client is **proposed, not yet live**:

`GET /integrations/fenix/orgs/{org_id}/projects/{project_id}/review-tasks`
(or `/api/integrations/fenix/...` through the gateway).

It requires a DubBridge-issued limited **S2S-only** token and a DB-backed active principal. The existing general GET route is **not** a substitute. JWT secret stays only in DubBridge. See [proposed ADR-048](https://github.com/krukmat/dubbridge/blob/feature/int-w1-fenix-service-identity/docs/adr/ADR-048-fenix-service-readonly-identity-boundary.md).

Fenix returns agent-safe `{contract_version,status,counts,tasks:[{ref,state}],source}`, omitting assets, assignees, personal identifiers, JWTs and raw provider payload.

## Actual implementation staging

The Go source currently exists only in a separately validated local package, **not pushed** to this branch:
- `internal/infra/integration/dubbridge/client.go`: HTTP GET, fixed tenant/project, HTTPS except loopback, redirect rejection, bounded response, strict DTO/queue state validation, correlation headers.
- `internal/infra/integration/dubbridge/executor.go`: `ToolExecutor`-compatible semantic request and safe projection.
- `internal/api/integration_runtime_dubbridge.go`: planned opt-in registration, permission and retry adapter.
- `internal/api/integration_runtime.go`: one-function call insertion required.
- `internal/infra/integration/dubbridge/*_test.go`: config, schema, projection, denial, real-only opt-in test. No fake HTTP provider.

The local Go package is unit-verifiable independently; **full Fenix Go 1.25 application QA and independent code review have not run in this container**, so the runtime files are withheld from remote push per `AGENTS.md`. Replacing Fenix's real `go.mod` with a staging module is expressly forbidden.

## Functional acceptance

Run real Fenix + actual DubBridge Rust API and PostgreSQL; provision genuine restricted credential/organization/project/task; assert matching task IDs/states, zero write calls, fail-closed unauthorized tenants, bounded retries/outage, stable trace/execution audit, no secrets to Blackboard. Local unit checks or skip of an unconfigured E2E test are **not** cross-product PASS.

## Risks and decision checkpoints

- **B1 blocker:** dedicated S2S verifier/issuer and project binding do not exist yet; a human password-login JWT carries excessive permissions.
- **B3 blocker:** staging cannot be promoted without relevant full-app build, local QA and peer review.
- **B4 blocker:** no running Rust API/PostgreSQL/real token in current environment. Never claim success without observed evidence.
- **Release:** no PR merge/main deployment approved; real E2E required first.
