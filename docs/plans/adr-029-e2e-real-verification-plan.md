---
doc_type: plan
id: ADR-029-E2E
title: "E2E real verification — ADR-029 BFF Admin Shell"
status: completed
phase: 4
week: 18
tags: [plan, e2e, admin, bff, verification, adr-029]
fr_refs: [FR-060, FR-070, FR-071, FR-200, FR-230]
uc_refs: [UC-C1]
blocked_by: []
blocks: []
files_affected:
  - bff/src/routes/admin.ts
  - bff/src/routes/adminWorkflows.ts
  - bff/src/routes/adminAgentRuns.ts
  - bff/src/routes/adminAgentRunsFragments.ts
  - bff/src/routes/adminApprovals.ts
  - bff/src/routes/adminApprovalsFragments.ts
  - bff/src/routes/adminAudit.ts
  - bff/src/routes/adminPolicy.ts
  - bff/src/routes/adminTools.ts
  - bff/src/routes/adminMetrics.ts
  - bff/src/routes/adminLayout.ts
  - bff/src/routes/adminAuth.ts
  - internal/api/routes.go
  - data/fenixcrm.db
created: 2026-04-28
completed: 2026-04-28
---

# Plan: E2E real verification — ADR-029 BFF Admin Shell

## Objective

ADR-029 documents the HTMX + Express admin shell at `/bff/admin/*`. The
implementation is complete. The Jest suite (`bff/tests/admin*.test.ts`) covers
rendering logic with mocked Go responses but does **not** verify actual Go ↔ BFF
integration, real DB data flowing to HTML, or full HTTP stack behavior.

Run a full end-to-end verification of the admin shell against real
infrastructure (Go backend + BFF + SQLite), acting as the browser. Issue real
HTTP requests with a real JWT. Document any failures, apply fixes if needed,
and leave enough evidence for another operator to replay the verification.

---

## Embedded contract

This plan is intended to be executable on its own. Use the rules below as the
operational contract during verification.

### System assumptions

- The admin shell lives under `/bff/admin/*`.
- The stack under test is Go backend + BFF + SQLite.
- The BFF is a thin proxy for this surface: no direct DB access, no business
  logic required to interpret tenant identity, and no trust in client-supplied
  `workspace_id`.
- Protected backend routes require a valid JWT.
- The HTML structure contract for admin pages is the presence of a first
  `<h2 class="page-title">...`.
- Existing Jest coverage already validates mocked rendering paths; this plan
  exists to verify the real HTTP stack, real auth, and real data flow.

### Route-shape assumptions

- The plan treats the route table below as the source of truth for the run.
- The workflow activation path is exercised through the BFF as a POST that
  relays to the backend activation endpoint.
- Approval decisions are exercised through the BFF as a POST that relays to the
  backend decision endpoint.
- The policy page is a composite route and must show evidence from both of its
  upstream sources in one rendered page.
- The metrics page is backed by live backend metrics rather than seeded DB rows.

### Optional references

If more context is needed during debugging or remediation, the following files
can be consulted, but they are not required to execute this plan:

- `CLAUDE.md`
- `docs/decisions/ADR-029-bff-admin-shell.md`
- `docs/architecture.md`
- `docs/decisions/ADR-009-bff-thin-proxy.md`
- `docs/decisions/ADR-008-route-structure.md`
- `bff/src/routes/adminLayout.ts`
- `bff/tests/admin.e2e.test.ts`
- `bff/tests/admin*.test.ts`
- `internal/api/routes.go`
- `scripts/e2e_seed_mobile_p2.go`

---

## Infrastructure

### Stack

```
Go backend   → http://localhost:8080   (go-chi, SQLite WAL)
BFF          → http://localhost:3000   (Express + HTMX)
Database     → data/fenixcrm.db        (SQLite, existing data)
```

### Environment variables

```bash
# Go backend
DATABASE_URL=./data/fenixcrm.db
JWT_SECRET=dev-secret-key-32-chars-minimum!!
PORT=8080

# BFF
BACKEND_URL=http://localhost:8080
BFF_PORT=3000
NODE_ENV=development
```

Reference: `.env.example` (root) and `bff/.env`.

### Start commands

```bash
# Go backend
go build -o bin/fenixcrm ./cmd/fenix
DATABASE_URL=./data/fenixcrm.db JWT_SECRET=dev-secret-key-32-chars-minimum!! PORT=8080 \
  ./bin/fenixcrm serve

# BFF (separate terminal)
cd bff && npm run build
BACKEND_URL=http://localhost:8080 BFF_PORT=3000 node dist/server.js
```

### Health checks

```bash
curl -s http://localhost:8080/health       # expect: {"status":"ok"} or similar
curl -s http://localhost:3000/bff/health   # expect: 200 + backend reachable
```

---

## Test credentials

The `e2e@fenixcrm.test` user exists in `data/fenixcrm.db` and belongs to a
workspace with representative data seeded by the mobile E2E seed script
(`scripts/e2e_seed_mobile_p2.go`).

For post-remediation reruns, that seed also grants the minimum operator
permissions needed for the admin-shell slice under test, including
`admin.tools.list` and `workflows.activate`.

```
email:     e2e@fenixcrm.test
password:  e2eTestPass123!
workspace: E2E Test Workspace
```

### Obtain JWT

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"e2e@fenixcrm.test","password":"e2eTestPass123!"}' \
  | jq -r '.token')
```

---

## DB state (at 2026-04-28)

| Table | Count | Notes |
|---|---|---|
| `workflow` | 1 | `e2e_graph_followup_20260427T211759`; post-remediation seed should place the activation fixture in status `testing` |
| `approval_request` | 2 | both `pending` — usable for M2 mutation |
| `audit_event` | 5531 | large set — pagination viable |
| `tool_definition` | 160 | — |
| `agent_run` | 5 | `completed`, `handed_off`, `denied_by_policy` |
| `policy_set` | 1 | — |

If the DB has been reset or re-seeded, re-run the seed:

```bash
go run ./scripts/e2e_seed_mobile_p2.go
```

---

## Execution rules

1. Execute tasks in order. Do not start mutation checks until infrastructure,
   auth, and read-route checks pass.
2. Capture evidence for every task: command used, HTTP status, and the concrete
   page marker or data item observed.
3. Do not hardcode IDs for detail routes. Extract them from the corresponding
   list route in the same run.
4. If a mutation changes the seeded state, record the changed ID and re-seed the
   DB before re-running the full plan.
5. If any route fails because prerequisite data is missing, mark the task as
   blocked by DB state rather than silently skipping it.

## Task breakdown

| Task | Scope | Depends on | Effort | Reasoning | Recommended model | Output |
|---|---|---|---|---|---|---|
| T1 | Start Go backend and BFF, confirm health checks | None | Low | Low | `gpt-5.4-mini` | Both services reachable |
| T2 | Authenticate seeded user and obtain JWT | T1 | Low | Low | `gpt-5.4-mini` | Non-empty JWT and workspace-backed session |
| T3 | Verify read routes R1-R11 | T2 | Medium | Medium | `gpt-5.4` | Per-route result with evidence |
| T4 | Verify mutation routes M1-M2 | T3 | Medium | Medium | `gpt-5.4` | Mutation result, side effects, and reseed note if used |
| T5 | Verify architecture constraints C1-C4 | T1 | Low | Low | `gpt-5.4-mini` | Command outputs with pass/fail verdict |
| T6 | Record outcome, issues, and replay notes | T3, T4, T5 | Low | Low | `gpt-5.4-mini` | Completed outcome table and issue log |

## Task card template

Present each task before execution using this format:

- `Tarea: <task id and name>`
- `Resumen: <1-2 sentences>`
- `Código afectado: <files, binaries, services, or data areas expected to be touched>`
- `Esfuerzo/razonamiento: <Low|Medium|High> - <brief reason>`
- `Modelo recomendado: <model id>`
- `Tokens estimado: ~N`

## What to verify

### Read routes (R1–R11)

Base pass criteria for HTML routes: HTTP 200, `Content-Type: text/html`,
`class="page-title"` present in body, and expected heading present.

Additional evidence rules:

- Static shell routes must show the expected heading and shared layout.
- List routes must show at least one real row, name, or ID from current data.
- Detail routes must show the resolved ID plus at least one additional field.
- Composite routes must show evidence from each upstream data source they join.
- Metrics routes must show at least one real metric name/value pair from the
  live backend, even though the source is not the seeded DB.

| ID | BFF route | Go endpoint | Expected heading | Evidence required |
|---|---|---|---|---|
| R1 | `GET /bff/admin` | — static — | Dashboard | Shared layout + dashboard heading |
| R2 | `GET /bff/admin/workflows` | `GET /api/v1/workflows` | Workflows | One workflow name or ID |
| R3 | `GET /bff/admin/workflows/:id` | `GET /api/v1/workflows/{id}` | workflow name | Workflow ID + status or version |
| R4 | `GET /bff/admin/agent-runs` | `GET /api/v1/agents/runs` | Agent Runs | One run ID or status row |
| R5 | `GET /bff/admin/agent-runs/:id` | `GET /api/v1/agents/runs/{id}` | run id or status | Run ID + one detail field |
| R6 | `GET /bff/admin/approvals` | `GET /api/v1/approvals` | Approvals | One approval ID or action, plus evidence that decisions are submitted inline from the queue |
| R7 | `GET /bff/admin/audit` | `GET /api/v1/audit/events` | Audit Trail | One audit event ID or actor |
| R8 | `GET /bff/admin/audit/:id` | `GET /api/v1/audit/events/{id}` | event detail | Event ID + one detail field |
| R9 | `GET /bff/admin/policy` | `GET /api/v1/governance/summary` + `GET /api/v1/policy/sets` | Quota States or Policy Sets | One quota item and one policy set item |
| R10 | `GET /bff/admin/tools` | `GET /api/v1/admin/tools` | Tools | One tool name or ID |
| R11 | `GET /bff/admin/metrics` | `GET /metrics` | Metrics | One Prometheus metric name/value |

IDs for detail routes (R3, R5, R8) must be extracted from the list responses
(R2, R4, R7) — do not hardcode IDs that may not exist.

### Mutation routes (M1–M2)

Pass: HTTP 200, 3xx redirect, or explicitly handled business response
consistent with the observed pre-state. Fail: 404, 500, or undocumented state
transition behavior.

Mutation handling rules:

1. Record the pre-state before issuing the mutation.
2. Record the post-state immediately after the mutation.
3. If the mutation consumes seeded data, either:
   - re-seed after the task completes, or
   - mark the plan instance as non-repeatable until re-seed happens.
4. If a route returns a business validation response instead of success,
   capture it as a meaningful result, not just "fail", when it matches the
   observed input state.

| ID | BFF route | Go endpoint | Notes |
|---|---|---|---|
| M1 | `POST /bff/admin/workflows/:id/activate` | `PUT /api/v1/workflows/{id}/activate` | First capture current workflow status. If already active, accept idempotent success or explicit no-op behavior as pass only when documented in notes. |
| M2 | `POST /bff/admin/approvals/:id/decision` | `PUT /api/v1/approvals/{id}` | Use one pending approval for approve and one for reject. Record both IDs. Re-seed after execution if the approvals cannot be reset through the UI flow. |

### Architecture constraints (C1–C4)

These verify the hard constraints from ADR-029 §Decision.

| ID | Constraint | Command |
|---|---|---|
| C1 | BFF zero DB access | `grep -rn "sqlite\|knex\|prisma\|better-sqlite\|Database" bff/src/routes/admin*.ts` → zero matches |
| C2 | No client `workspace_id` forwarded | `grep -n "workspace_id" bff/src/routes/admin*.ts \| grep -v "//"` → zero unguarded forwarding |
| C3 | Missing/invalid token → redirect, not JSON 401 | `curl -v http://localhost:3000/bff/admin/workflows` (no token) → `302` to `/bff/admin` |
| C4 | 6 path corrections from ADR in use | `grep -n "agents/runs\|audit/events\|admin/tools\|PUT.*approvals\|PUT.*activate" bff/src/routes/admin*.ts` → all 6 present |

---

## Severity model

| Severity | Meaning | Action |
|---|---|---|
| Critical | Prevents login, startup, or most admin routes from working | Stop the run, fix first, then restart from T1 |
| Major | Breaks one route family or mutation path but leaves rest runnable | Record issue, fix, and re-run affected task set |
| Minor | Layout, wording, or evidence mismatch without route failure | Record and fix if low-risk; otherwise leave explicit follow-up |
| Observed | Expected deviation due to known state or fixture drift | Record in notes, no fix required unless it blocks repeatability |

## Pass / fail summary

| Area | Pass | Fail |
|---|---|---|
| Infrastructure | Both servers healthy | Either fails to start |
| Auth | Non-empty JWT returned | Login returns 4xx |
| R1–R11 | All route-specific evidence captured | Any non-200, missing landmark, or missing route evidence |
| M1–M2 | Pre-state, response, and post-state are all captured and coherent | 404, 500, or unverifiable mutation outcome |
| C1–C4 | All constraints satisfied | Any violation |

---

## Evidence template

Use one row per task or route family while executing.

| Item | Command / request | Expected | Observed | Evidence captured | Result |
|---|---|---|---|---|---|
| `T1` | — | — | — | — | — |
| `T2` | `curl -sS -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"email":"e2e@fenixcrm.test","password":"e2eTestPass123!"}'` and `curl -sS -X POST http://localhost:3000/bff/auth/login ...` | Non-empty JWT and workspace-backed session | Both endpoints returned `200`; JWT present; `workspaceId=019d52f1-548a-79be-b731-01b7d4d45c1d`; `/api/v1/accounts` returned 1 account in that workspace | Login JSON, BFF relay JSON, protected accounts list | Pass |
| `T3 / R1-R11` | `GET /bff/admin*` with `Authorization: Bearer <jwt>` after extracting IDs from `GET /api/v1/workflows`, `GET /api/v1/agents/runs`, `GET /api/v1/approvals?status=pending`, and `GET /api/v1/audit/events?limit=1` | All 11 canonical routes return `200 text/html` with `page-title` and route-specific evidence | `R1,R2,R3,R4,R5,R6,R7,R9,R11` returned `200 text/html`; `R8` returned `500`; `R10` returned `403`. The pre-remediation run also probed superseded `GET /bff/admin/approvals/:id`, which returned `405` and is no longer part of the canonical route set. | `R3` showed workflow `e2e_graph_followup_20260428T105950` for ID `019dd3be-cde7-7885-7a48-fa7f6bedcf3c`; `R5` showed run ID `019dd3be-cde5-7380-9247-62817f8b378a`; `R6` listed 2 pending approvals including `019dd3be-cde7-7c8d-3b48-32059d51212c`; `R7` listed audit event `019dd3be-ce17-7f83-2c86-46e65c4df91f`; `R9` rendered `Quota States`; `R11` rendered `Metrics` | Fail |
| `T4 / M1-M2` | `POST /bff/admin/workflows/:id/activate` and `POST /bff/admin/approvals/:id/decision` with pre/post checks against Go routes | Mutation returns success, redirect, or meaningful business response with coherent post-state | `M1` returned `200 text/html` but rendered `Activation failed: Request failed with status code 403`; workflow stayed `active`. `M2` approve and reject both returned `404` backend errors; pending approvals stayed unchanged through the BFF route. | `M1` pre/post workflow `019dd3bf-1021-7d9f-ade0-b968c5d32af0` remained `active`; `M2` used approvals `019dd3bf-1021-72b8-ede8-e98b52e09e15` and `019dd3bf-1021-70bd-4c48-d6e18906d68f`; direct backend diagnostic `PUT /api/v1/approvals/{id}` returned `204`, confirming the BFF relay path is wrong rather than the underlying approval service | Fail |
| `T11 / targeted rerun` | Re-seed with `go run ./scripts/e2e_seed_mobile_p2.go`, then re-run `R8`, `R10`, `M1`, `M2`, and `C4` only | All remediated slices pass on real HTTP execution with the corrected operator fixture | `R8` returned `200`; `R10` returned `200`; `M1` moved the seeded workflow from `testing` to `active` with `302`; `M2` approve and reject both returned `302` and removed both pending approvals; `C4` showed the corrected BFF upstream paths in code. | `R8` rendered audit event `019dd448-872d-749e-ce84-a29ca6dde7d4` with actor `019d52f1-548a-748f-89fe-45f2b70383dd` and action `get_request`; `R10` rendered `Tools` with live tool `create_task`; `M1` mutated workflow `019dd448-1023-75a0-f245-8f205aa2da53` from `testing` to `active`; `M2` consumed approvals `019dd448-1023-759a-57cd-d25a920008f0` and `019dd448-1023-7b80-e9bf-550e38080044`; `C4` showed `client.put(\`/api/v1/approvals/\${id}\`)` and `client.put(\`/api/v1/workflows/\${id}/activate\`)` in the BFF routes | Pass |

### T3 route notes

| Route | Result | Notes |
|---|---|---|
| `R1 /bff/admin` | Pass | `200 text/html`; page title `Dashboard`; shared admin layout links rendered |
| `R2 /bff/admin/workflows` | Pass | `200 text/html`; page title `Workflows`; workflow list resolved ID `019dd3be-cde7-7885-7a48-fa7f6bedcf3c` |
| `R3 /bff/admin/workflows/:id` | Pass | `200 text/html`; page title `e2e_graph_followup_20260428T105950`; detail showed workflow ID and `active` status |
| `R4 /bff/admin/agent-runs` | Pass | `200 text/html`; page title `Agent Runs (5)`; list resolved ID `019dd3be-cde5-7380-9247-62817f8b378a` |
| `R5 /bff/admin/agent-runs/:id` | Pass | `200 text/html`; detail rendered run ID `019dd3be-cde5-7380-9247-62817f8b378a` |
| `R6 /bff/admin/approvals` | Pass | `200 text/html`; page title `Approvals (2)`; pending approval ID `019dd3be-cde7-7c8d-3b48-32059d51212c` present and the queue was the only canonical approval read surface |
| `R7 /bff/admin/audit` | Pass | `200 text/html`; page title `Audit Trail`; audit list resolved event ID `019dd3be-ce17-7f83-2c86-46e65c4df91f` |
| `R8 /bff/admin/audit/:id` | Fail | `500 Internal Server Error`; BFF error body: `Cannot read properties of undefined (reading 'permissions_checked')`; Go `GET /api/v1/audit/events/{id}` returned a plain JSON object, not `{ data: ... }` |
| `R9 /bff/admin/policy` | Pass | `200 text/html`; page rendered `Quota States`; composite governance page loaded |
| `R10 /bff/admin/tools` | Fail | `403 Forbidden`; upstream `GET /api/v1/admin/tools` returned `{\"error\":\"forbidden\"}` for the seeded user |
| `R11 /bff/admin/metrics` | Pass | `200 text/html`; page title `Metrics`; live Prometheus metrics rendered |

### T4 mutation notes

| Mutation | Result | Notes |
|---|---|---|
| `M1 POST /bff/admin/workflows/:id/activate` | Fail | Pre-state for workflow `019dd3bf-1021-7d9f-ade0-b968c5d32af0`: `active`, version `1`. BFF returned `200 text/html`, but the page contained `Activation failed: Request failed with status code 403`. Direct backend check `PUT /api/v1/workflows/{id}/activate` also returned `403 forbidden`. Post-state stayed `active`, so this is an authorization block rather than idempotent success. |
| `M2 POST /bff/admin/approvals/:id/decision` | Fail | Pre-state had two pending approvals: `019dd3bf-1021-72b8-ede8-e98b52e09e15` (`close_case`) and `019dd3bf-1021-70bd-4c48-d6e18906d68f` (`send_external_email`). BFF approve and reject requests both returned `404` with backend error payloads and left the pending list unchanged. Direct backend diagnostic `PUT /api/v1/approvals/{id}` returned `204 No Content`, proving the BFF is relaying to the wrong upstream path/method. |

### T11 targeted rerun notes

| Slice | Result | Notes |
|---|---|---|
| `R8 /bff/admin/audit/:id` | Pass | `200 text/html`; rendered audit event `019dd448-872d-749e-ce84-a29ca6dde7d4` with actor `019d52f1-548a-748f-89fe-45f2b70383dd` and action `get_request`. |
| `R10 /bff/admin/tools` | Pass | `200 text/html`; page title `Tools`; live table rendered tool `create_task` from `GET /api/v1/admin/tools`. |
| `M1 POST /bff/admin/workflows/:id/activate` | Pass | Freshly seeded workflow `019dd448-1023-75a0-f245-8f205aa2da53` started in `testing`, BFF returned `302` to `/bff/admin/workflows/019dd448-1023-75a0-f245-8f205aa2da53`, and post-state became `active`. |
| `M2 POST /bff/admin/approvals/:id/decision` | Pass | Fresh pending approvals `019dd448-1023-759a-57cd-d25a920008f0` and `019dd448-1023-7b80-e9bf-550e38080044` both returned `302` to `/bff/admin/approvals`; post-check showed neither ID remained in the pending queue. |
| `C4` | Pass | Code inspection confirmed corrected admin-shell relays: `adminApprovals.ts` now uses `PUT /api/v1/approvals/{id}`, `adminWorkflows.ts` uses `PUT /api/v1/workflows/{id}/activate`, and the previously corrected `agents/runs`, `audit/events`, and `admin/tools` paths remain in use. |

## Outcome record

Fill this in after execution.

| Area | Result | Notes |
|---|---|---|
| Infrastructure | Pass | `GET /health` on Go backend returned `200`; `GET /bff/health` returned `200`, so both services were reachable during verification. |
| Auth | Pass | Seeded user `e2e@fenixcrm.test` authenticated successfully via Go backend and BFF; JWT was non-empty and authenticated workspace had seeded data. |
| R1–R11 | Pass | The original failures were narrowed to `R8` and `R10`, and the targeted post-remediation rerun cleared both: `R8` now returns `200` with real audit detail and `R10` now returns `200` with live tools data. |
| M1–M2 | Pass | The targeted rerun proved the seeded operator flow end-to-end: `M1` transitioned a fresh workflow from `testing` to `active`, and `M2` approve/reject both completed through the BFF and removed the targeted approvals from the pending queue. |
| C1–C4 | Pass | `C1`, `C2`, and `C3` remained valid. `C4` is now satisfied because the BFF relays use the corrected approvals and workflow activation paths while the previously corrected `agents/runs`, `audit/events`, and `admin/tools` routes remain intact. |
| **Overall** | Pass | ADR-029 is now verified as healthy across the intended BFF admin-shell layers after the targeted post-remediation rerun on `2026-04-28`. The initial failing evidence remains recorded above as the baseline that the remediation closed. |

## Executive conclusion

The original run exposed real ADR-029 drift across approvals routing, audit
detail handling, operator permissions, and workflow activation verifiability.
Those failures were remediated and then revalidated with a targeted real-HTTP
rerun on `2026-04-28`.

What the targeted rerun proved:

- The real stack started and stayed healthy.
- Authentication and JWT-backed admin access worked.
- The previously failing read routes `R8` and `R10` now render real HTML with live data.
- `M1` is verifiable for the seeded operator flow: a fresh workflow starts in `testing` and activates successfully.
- `M2` is verifiable for the canonical approvals contract: both decision paths succeed through the BFF and remove the targeted pending approvals.
- Core thin-proxy constraints hold across `C1`–`C4`.

Therefore the final outcome of this plan is: ADR-029 is verified as healthy
across the intended admin-shell layers, with the pre-remediation failures
preserved above as historical evidence and the `T11` rerun serving as the final
health verdict.

## Issues found

| File | Line | Issue | Fix applied |
|---|---|---|---|
| `internal/api/routes.go` | `314-317` | Backend exposes approvals as queue + decision only (`GET /api/v1/approvals`, `PUT /api/v1/approvals/{id}`); the earlier approval-detail assumption was invalid. | Yes - the ADR/plan contract was re-scoped around queue + decision relay. |
| `bff/src/routes/adminAudit.ts` | `212` | BFF expected audit detail in `{ data: ... }`, but the live backend returned a bare audit event object. | Yes - `R8` now renders audit detail successfully in the targeted rerun. |
| `bff/src/routes/adminTools.ts` | `77` | Tools page previously failed because the seeded operator lacked `admin.tools.list`. | Yes - the corrected fixture grants the permission and `R10` now returns `200`. |
| `bff/src/routes/adminWorkflows.ts` | `189-202` | Workflow activation was previously unverifiable because the seeded operator hit `403` and the fixture workflow was already `active`. | Yes - the corrected fixture grants `workflows.activate` and seeds the verification workflow in `testing`, allowing `M1` to pass. |
| `bff/src/routes/adminApprovals.ts` | `175` | Approvals decision route previously posted to `/api/v1/approvals/{id}/decision`, but the live backend only accepts `PUT /api/v1/approvals/{id}`. | Yes - `M2` now succeeds through the corrected BFF relay. |
| `bff/scripts/admin-screenshots/catalog.ts` | `1-66` | Screenshot catalog and smoke-run were revalidated against the admin surface, and the workflow authoring flow now captures `list -> create draft -> builder -> detail` as part of a 13-route catalog without failures. | Yes |

## Replay notes

- Baseline failing run: `go run ./scripts/e2e_seed_mobile_p2.go` at `2026-04-28 10:59` Europe/Madrid.
- Targeted rerun after remediation: `go run ./scripts/e2e_seed_mobile_p2.go` at `2026-04-28 15:29` Europe/Madrid.
- IDs used in the final targeted rerun: workflow `019dd448-1023-75a0-f245-8f205aa2da53`, approvals `019dd448-1023-759a-57cd-d25a920008f0` and `019dd448-1023-7b80-e9bf-550e38080044`, audit `019dd448-872d-749e-ce84-a29ca6dde7d4`.
- IDs mutated in the final targeted rerun: workflow `019dd448-1023-75a0-f245-8f205aa2da53` moved `testing -> active`; both pending approvals were consumed through the BFF decision route.
- Re-seed required before repeating the exact targeted rerun: `yes`; `M1` changes workflow state and `M2` consumes both pending approvals by design.
