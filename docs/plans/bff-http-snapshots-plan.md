---
doc_type: task
id: bff-http-snapshots
title: BFF HTTP Snapshots — black-box capture analogous to mobile screenshots
status: planned
phase: post-MVP
week: TBD
tags: [bff, observability, testing, traceability]
fr_refs: [FR-301, FR-200, FR-202, FR-070]
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - bff/package.json
  - bff/scripts/http-snapshots.ts
  - bff/scripts/snapshots/catalog.ts
  - bff/scripts/snapshots/runner.ts
  - bff/scripts/snapshots/redact.ts
  - bff/scripts/snapshots/sse-capture.ts
  - bff/scripts/snapshots/report.ts
  - bff/scripts/snapshots/seeder.ts
  - bff/scripts/snapshots/types.ts
  - bff/scripts/snapshots/README.md
  - bff/tests/snapshots/redact.test.ts
  - bff/tests/snapshots/runner.test.ts
  - bff/tests/snapshots/catalog.test.ts
  - bff/tests/snapshots/report.test.ts
  - bff/tests/snapshots/sse-capture.test.ts
  - bff/tests/snapshots/seeder.test.ts
  - bff/.gitignore
  - bff/README.md
created: 2026-04-26
completed:
---

# BFF HTTP Snapshots Plan

## Goal

Provide an `npm run http-snapshots` command in the BFF that, analogous to the
mobile `npm run screenshots`, runs a black-box automated traversal against the
BFF endpoints and persists versionable artifacts that allow:

1. Quickly seeing what functionality the BFF exposes (living catalog).
2. Detecting contract regressions (status, headers, body shape) via diff.
3. Visually reviewing output in a navigable HTML report, without needing
   external tools (Postman/Insomnia/Bruno).

This is not a replacement for unit tests or BDD: it is an **executable
traceability** layer of the observable BFF behavior in a real runtime
(BFF + Go backend + SQLite up).

## Preconditions

A fresh agent picking up this plan must verify or set up the following before
starting any task:

- **Node**: 18.x or higher (uses native `fetch`, `ReadableStream`).
- **Go**: 1.22+ (to compile and run the seeder + backend).
- **Repo state**: clean working tree on the working branch, `bff/` deps
  installed (`cd bff && npm install`).
- **Backend health**: Go backend can be started locally with:
  ```
  JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080
  ```
  Verified via `curl http://localhost:8080/health`.
- **BFF health**: BFF can be started with `cd bff && npm run dev`. Verified
  via `curl http://localhost:3000/bff/health`.
- **Seeder**: `go run ./scripts/e2e_seed_mobile_p2.go` from repo root prints
  a JSON payload to stdout containing at minimum `auth.token`, `auth.userId`,
  `auth.workspaceId`. Used today by `mobile/maestro/seed-and-run.sh`.
- **Env vars**: `JWT_SECRET` must match between backend startup and seeder
  expectations. Default value above works locally.

If any precondition fails, stop and report rather than improvising
substitutes — the seeder and backend contract are load-bearing.

## Scope

### Included
- HTTP REST capture: request + response (method, path, relevant headers,
  body, status, latency).
- SSE capture for `/bff/copilot/events`: first N events or timeout.
- Deterministic redaction of tokens, volatile UUIDs, timestamps, request IDs
  so diffs are stable.
- Static HTML report (`report.html`) with sidebar navigable by endpoint and
  request/response visualization with syntax highlighting.
- Declarative catalog of endpoints (data, not code).
- Reuse of existing seeder (`scripts/e2e_seed_mobile_p2.go`) to obtain real
  token + workspace + sample CRM entities.
- Pre-flight health gates: aborts if BFF (`:3000`) or Go (`:8080`) are not healthy.

### Excluded
- PNG generation (Option B/C discarded for maintainability — see comparative
  analysis below).
- Load / performance tests.
- Non-reversible destructive mutations (every scenario must be idempotent
  or re-seedable).
- BFF→Go internal traffic capture (that's the observability layer,
  out of scope).

## Technical decision

**Stack**: Node 18+ native (`fetch`, `ReadableStream`, `fs/promises`) +
TypeScript + `ts-node` (already in BFF devDeps). **Zero new dependencies.**

### Comparative analysis (why not other options)

| Criterion | Node native (chosen) | Playwright/Puppeteer | Maestro | Postman/Bruno |
|---|---|---|---|---|
| New deps | 0 | ~300MB chromium | mobile-only | external CLI |
| CI runtime | ~5-10s | ~30-90s | N/A for HTTP | requires runner |
| Upgrade fragility | Very low | High (chromium CVEs) | Medium | External tool drift |
| Output diffability | JSON text | PNG binaries (poor) | N/A | JSON exports |
| Fits BFF (no UI) | Yes | Overkill | No | Yes but external |
| Onboarding | Read JSON / open HTML | Learn Playwright | Learn Maestro | Install tool |

The BFF has no UI to render — using a headless browser to capture JSON
payloads adds infrastructure cost without information gain.

## Initial endpoint catalog

Based on `bff/src/app.ts` and `bff/src/routes/*.ts` (verified 2026-04-26):

| # | Name | Method | Path | Auth | SSE | Body source |
|---|---|---|---|---|---|---|
| 1 | health | GET | /bff/health | no | no | none |
| 2 | metrics | GET | /bff/metrics | no | no | none |
| 3 | auth-login-success | POST | /bff/auth/login | no | no | seeded credentials |
| 4 | auth-login-invalid | POST | /bff/auth/login | no | no | known-bad password |
| 5 | auth-register-success | POST | /bff/auth/register | no | no | random email |
| 6 | builder-list | GET | /bff/builder | yes | no | none |
| 7 | copilot-chat-grounded | POST | /bff/api/v1/copilot/chat | yes | no | seeded entity ID |
| 8 | copilot-chat-abstain | POST | /bff/api/v1/copilot/chat | yes | no | nonexistent entity |
| 9 | copilot-stream | GET | /bff/copilot/events | yes | yes | query params only |
| 10 | copilot-sales-brief | POST | /bff/copilot/sales-brief | yes | no | seeded deal ID |
| 11 | inbox-list-empty | GET | /bff/api/v1/mobile/inbox | yes | no | none |
| 12 | inbox-list-with-items | GET | /bff/api/v1/mobile/inbox | yes | no | none (post-seed) |
| 13 | approvals-approve | POST | /bff/api/v1/approvals/:id/approve | yes | no | seeded approval ID |
| 14 | approvals-reject | POST | /bff/api/v1/approvals/:id/reject | yes | no | seeded approval ID |
| 15 | proxy-passthrough-deals | GET | /bff/api/v1/deals | yes | no | none |

### Request payload sourcing

The BFF auth and copilot routes are passthrough — they forward `req.body`
to the Go backend (see `bff/src/routes/auth.ts:11,22` and
`bff/src/routes/copilot.ts:138`). Therefore canonical payload shapes live
in the Go handlers under `internal/api/handlers/`.

For each entry that needs a body, the catalog must reference one of:

- **From seeder output**: `auth.email`, `auth.password`, `deal.id`,
  `case.id`, `account.id`, etc. (full list in `seed-and-run.sh:143-160`).
- **Static fixture**: hand-crafted JSON for invalid/edge cases
  (e.g. `auth-login-invalid` uses `{ email: "<seeded>", password: "wrong" }`).
- **Generated**: e.g. `auth-register-success` uses `crypto.randomUUID()` to
  build a unique email per run — but the random value is redacted before
  persisting to artifacts.

The catalog entry shape (see T3) carries either a literal body or a function
`(seed: SeederOutput) => unknown` so values are bound at runtime.

### Approval ID dependency

T13/T14 require a pending approval. The current seeder
(`scripts/e2e_seed_mobile_p2.go`) does NOT create one. Two options:

- **A (preferred)**: extend the seeder to produce a pending approval and
  expose it as `seed.approval.id`. Single source of truth, reused by mobile.
- **B (fallback)**: in the BFF runner, before T13/T14, call the Go API
  `POST /api/v1/approvals` to create one inline. Adds runner complexity but
  keeps the seeder untouched.

**Decision**: go with A. Add seeder extension as a sub-task of T2.
If extending the seeder turns out to be larger than ~30 min of work,
fall back to B and flag the divergence in the task closing report.

## Artifact structure

```
bff/artifacts/http-snapshots/
  index.md                     # Human-readable summary (markdown)
  report.html                  # Navigable report
  raw/
    health.json
    auth/
      login-success.json
      login-invalid.json
      register-success.json
    builder/
      list.json
    copilot/
      chat-grounded.json
      chat-abstain.json
      stream.sse.json          # array of SSE events
      sales-brief.json
    inbox/
      list-empty.json
      list-with-items.json
    approvals/
      approve.json
      reject.json
    proxy/
      passthrough-deals.json
```

### TypeScript types (target shape for `types.ts`)

```ts
export type SeederOutput = {
  auth: { token: string; userId: string; workspaceId: string; email: string; password: string };
  account?: { id: string };
  contact?: { id: string; email: string };
  deal?: { id: string };
  case?: { id: string; subject: string };
  approval?: { id: string };
  // ...other entities exposed by scripts/e2e_seed_mobile_p2.go
};

export type CatalogEntry = {
  name: string;                       // unique, used as filename
  group: string;                      // subfolder under raw/
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  path: string;                       // BFF path, may contain :params
  auth: boolean;                      // attach Bearer token if true
  sse?: { maxEvents: number; timeoutMs: number };
  body?: unknown | ((seed: SeederOutput) => unknown);
  pathParams?: (seed: SeederOutput) => Record<string, string>;
  expectedStatus: number;             // for pass/fail in index.md
};

export type SnapshotArtifact = {
  name: string;
  method: string;
  path: string;
  request: { headers: Record<string, string>; body?: unknown };
  response: { status: number; headers: Record<string, string>; body?: unknown };
  latencyMs: number;
  capturedAt: string;                 // redacted to "<timestamp>"
};
```

### JSON output example

```json
{
  "name": "auth-login-success",
  "method": "POST",
  "path": "/bff/auth/login",
  "request": {
    "headers": { "content-type": "application/json", "authorization": "Bearer <REDACTED>" },
    "body": { "email": "seed@fenix.local", "password": "<REDACTED>" }
  },
  "response": {
    "status": 200,
    "headers": { "content-type": "application/json" },
    "body": { "token": "<REDACTED>", "userId": "<uuid:1>", "workspaceId": "<uuid:2>" }
  },
  "latencyMs": 84,
  "capturedAt": "<timestamp>"
}
```

Artifacts go to `.gitignore` by default (regenerable output). Only an optional
**baseline** is committed under `bff/artifacts/http-snapshots-baseline/` if
the team decides to use it for CI diff (future decision, out of scope).

## Tasks (TDD)

> CLAUDE.md rule: tests first, implementation after, run tests.
> Each task ends with a "Done when" checklist that the agent must satisfy
> before reporting completion.

### T1 — Health gates + script entry point
- Create `bff/scripts/http-snapshots.ts` with Go + BFF health verification.
- Test: `tests/snapshots/runner.test.ts` mocks `fetch` and validates abort
  if either health endpoint returns non-2xx or times out.
- **Done when**:
  - `npm test -- snapshots/runner` passes.
  - Running the script with backend down exits with code 1 and prints a
    message naming which dependency is unreachable to stderr.
  - Files: `bff/scripts/http-snapshots.ts`, `bff/scripts/snapshots/types.ts`,
    `bff/tests/snapshots/runner.test.ts`.

### T2 — Seeder integration (+ approval extension)
- `bff/scripts/snapshots/seeder.ts`: spawns `go run ./scripts/e2e_seed_mobile_p2.go`
  from repo root via `child_process.spawn`, captures stdout, parses as
  `SeederOutput`.
- Sub-task: extend `scripts/e2e_seed_mobile_p2.go` to emit `approval.id`
  for a pending approval. If sub-task exceeds ~30 min, fall back to
  inline-create in the runner and flag in closing report.
- Test: `tests/snapshots/seeder.test.ts` mocks `child_process.spawn` and
  validates parsing of canonical seed JSON + error path on non-zero exit.
- **Done when**:
  - `npm test -- snapshots/seeder` passes.
  - Manual run: `ts-node bff/scripts/snapshots/seeder.ts` against a live
    backend prints a parsed `SeederOutput` object including `approval.id`
    (or logs the documented fallback).
  - Files: `bff/scripts/snapshots/seeder.ts`,
    `bff/tests/snapshots/seeder.test.ts`,
    plus seeder extension if applicable.

### T3 — Declarative catalog
- `bff/scripts/snapshots/catalog.ts` with the 15 typed `CatalogEntry`
  entries. Bodies use seed-bound functions where applicable
  (see "Request payload sourcing" above).
- Test: validates each entry has required fields per `CatalogEntry` type,
  that `name` and `path+method` combos are unique, and that `expectedStatus`
  is set.
- **Done when**:
  - `npm test -- snapshots/catalog` passes.
  - Type check (`npm run typecheck` if defined, else `tsc --noEmit`) clean.
  - Files: `bff/scripts/snapshots/catalog.ts`,
    `bff/tests/snapshots/catalog.test.ts`.

### T4 — Deterministic redaction
- `bff/scripts/snapshots/redact.ts`: replaces tokens (`Bearer xxx`),
  UUIDs (regex `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
  case-insensitive), ISO-8601 timestamps, and request ID headers with
  deterministic placeholders.
- UUIDs are mapped by order of appearance → `<uuid:1>`, `<uuid:2>`, ...
  using a per-snapshot map so the same UUID gets the same placeholder
  within one snapshot.
- Test: input fixture with mixed tokens/UUIDs/timestamps → output with
  stable placeholders; running redact twice on same input produces
  byte-identical output.
- **Done when**:
  - `npm test -- snapshots/redact` passes including the idempotency assertion.
  - No raw UUID, JWT, or ISO timestamp present in fixture output.
  - Files: `bff/scripts/snapshots/redact.ts`,
    `bff/tests/snapshots/redact.test.ts`.

### T5 — REST runner
- `bff/scripts/snapshots/runner.ts`: iterates catalog, executes `fetch`,
  measures latency with `performance.now()`, applies redaction, writes JSON
  per entry to `bff/artifacts/http-snapshots/raw/<group>/<name>.json`.
- Skips SSE entries (handled by T6).
- Test: catalog of 2 mock REST entries + in-process express server →
  validates files exist with redacted content, status, and latency field.
- **Done when**:
  - `npm test -- snapshots/runner` passes (covering both health-gate and REST cases).
  - Files: `bff/scripts/snapshots/runner.ts`, expansion of `runner.test.ts`.

### T6 — SSE capture ✅ COMPLETED 2026-04-26
- `bff/scripts/snapshots/sse-capture.ts`: opens stream with `fetch`, reads
  via `response.body!.getReader()`, parses `data:` lines, captures first N
  events or until `timeoutMs`. Timeout implemented via `Promise.race` +
  per-read deadline sentinel (avoids AbortController/Jest interaction bug).
- Integrated into runner: SSE entries route through `executeSSEEntry`,
  write artifact as `<name>.sse.json`.
- Test: 8 tests covering basic capture, maxEvents limit, timeout (0 events),
  partial capture, auth forwarding, multi-line data, non-200, SSEEvent shape.
- runner.test.ts: added SSE routing integration test (in-process SSE server).
- **Files changed**:
  - `bff/scripts/snapshots/sse-capture.ts` ← new
  - `bff/tests/snapshots/sse-capture.test.ts` ← new
  - `bff/scripts/snapshots/runner.ts` ← SSE routing added
  - `bff/tests/snapshots/runner.test.ts` ← SSE routing test added

### T7 — HTML report generator ✅ COMPLETED 2026-04-26
- `bff/scripts/snapshots/report.ts`: reads `ArtifactWithGroup[]`, generates
  `report.html` (sidebar by group, detail panel, inline CSS/JS, syntax highlight)
  + `index.md` (status table with ✅/❌, latency, links to raw JSON).
- `loadArtifacts(rawDir)`: reads all `*.json` files under `raw/`, infers group
  from subdirectory name, returns `ArtifactWithGroup[]`.
- XSS safety: HTML-escape applied before colorization pass.
- 18 tests: HTML structure (balanced tags, names, status codes, paths),
  sidebar groups, status badges (2xx/4xx/5xx), XSS safety, index.md
  (rows, icons, latency), edge cases (empty list, SSE array body).
- **Files changed**:
  - `bff/scripts/snapshots/report.ts` ← new
  - `bff/tests/snapshots/report.test.ts` ← new (18 tests)

### T8 — Index.md generator ✅ COMPLETED 2026-04-26 (covered by T7/report.ts) ✅ COMPLETED 2026-04-26
- `SnapshotArtifact` now carries `expectedStatus` from the catalog when the
  REST/SSE runner writes artifacts.
- `report.ts` emits `index.md` with expected status, actual HTTP status,
  latency, and a link to the raw artifact.
- Pass/fail icons compare actual HTTP status to `expectedStatus`, so expected
  4xx snapshots such as invalid login are marked ✅ while mismatches are marked ❌.
- Unit coverage added for expected 4xx pass rows and mismatch failure rows.
- **Verification**:
  - `cd bff && npm test -- snapshots/report`
  - `cd bff && npm test -- snapshots/runner`
  - `cd bff && npm run build`

### T9 — Wire up `npm run http-snapshots` + e2e smoke ✅ COMPLETED 2026-04-26
- `bff/package.json` includes:
  `"http-snapshots": "ts-node scripts/http-snapshots.ts"`.
- `.gitignore` ignores generated `bff/artifacts/http-snapshots/`.
- The runner enables BFF fixture mode during capture to keep copilot chat/stream
  and sales brief deterministic, then disables it in `finally`.
- Auth relay forwards snapshot `X-Real-IP` so repeated auth snapshots do not trip
  Go auth rate limits; artifacts redact the synthetic IP.
- Seeder cleanup now includes pipeline and workflow fixtures, and emits separate
  approval IDs for approve/reject paths.
- Redaction normalizes duration-like fields, generated emails, seed suffixes,
  RFC1123/ISO timestamps embedded in strings, snapshot IPs, and metrics counters.
- **Verification**:
  - `curl http://localhost:8080/health`
  - `curl http://localhost:3000/bff/health`
  - `cd bff && npm test -- snapshots/redact snapshots/runner snapshots/report auth copilot`
  - `cd bff && npm test -- snapshots/seeder snapshots/catalog`
  - `cd bff && npm run build`
  - `cd bff && npx eslint src --max-warnings=0 -f json -o /tmp/fenix-bff-eslint.json`
  - `cd bff && npx tsc --noEmit --pretty false --target ES2022 --module commonjs --esModuleInterop --skipLibCheck scripts/http-snapshots.ts scripts/snapshots/*.ts`
  - `cd bff && npm run http-snapshots` twice: both runs passed with `16 passed,
    0 failed, 0 errors`.
  - `diff -qr /tmp/fenix-http-snapshots-raw-1 bff/artifacts/http-snapshots/raw`
    produced no differences.
  - `bff/artifacts/http-snapshots/raw` contains `16` JSON artifacts, and both
    `report.html` and `index.md` are generated.

### T10 — Documentation
- Create `bff/scripts/snapshots/README.md` with:
  - How to run (preconditions checklist linking to this plan).
  - How to add a new endpoint (catalog entry template).
  - How to interpret report status badges.
  - Troubleshooting (seeder failure, SSE timeout, port collisions).
- Update `bff/README.md`: new "HTTP Snapshots" section linking to the
  script README.
- **Done when**:
  - Both files updated and reviewed for accuracy against actual code.

## Final verification (whole feature)

1. `cd bff && npm test` — all tests pass.
2. `cd bff && npm run lint` — no warnings.
3. `cd bff && npm run typecheck` (if defined) — clean.
4. Bring up Go backend + BFF per Preconditions.
5. `npm run http-snapshots` — exit 0, 15 files generated.
6. Open `bff/artifacts/http-snapshots/report.html` in browser — all 15
   endpoints visible, request/response readable.
7. Re-execute the command — `git status` shows zero diff in `raw/`
   (deterministic redaction working).
8. Run pre-push hook locally (`scripts/hooks/pre-push`) — passes.

## Estimated effort

- **Complexity**: Medium
- **Estimated tokens for the full implementation across T1-T10**: ~25-35k
- **Estimated wall-clock time**: 4-6h implementation + 1h polish

## Expected maintenance

- Add new endpoint: 1 entry in `catalog.ts` + run the command.
- Response shape change: regenerate baseline (if adopted), commit the new one.
- Node upgrade: zero impact (only stable APIs `fetch` + `fs`).
- No chromium → zero incoming browser CVEs to triage.

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| Go seeder changes interface | Wrapper in `seeder.ts` isolates changes; test covers parsing |
| Seeder extension for `approval.id` is non-trivial | Documented fallback B (inline-create in runner) |
| SSE timeout in slow CI | Configurable via `FENIX_SNAPSHOTS_SSE_TIMEOUT_MS` |
| Catalog grows and `report.html` gets heavy | Lazy-load bodies (toggle expand/collapse) |
| Incomplete redaction leaves PII in artifacts | Type-specific redaction tests + manual review of first run |
| `ts-node` ESM/CJS issues with Node 20+ | Plan uses CJS-style imports per existing BFF tsconfig |

## Out of scope / Possible follow-ups

- Automatic CI diff against committed baseline (separate decision).
- OpenAPI spec generation from the catalog.
- Equivalent for the Go backend directly (without going through BFF).
- PNG rendering under flag (only if real use case appears, not preventive).
