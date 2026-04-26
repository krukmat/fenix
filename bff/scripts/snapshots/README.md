# BFF HTTP Snapshots

Captures live HTTP responses from the BFF layer, stores them as redacted JSON artifacts, and generates an HTML report + Markdown index.

---

## Preconditions checklist

Before running, all of the following must be true:

| # | Condition | How to satisfy |
|---|-----------|---------------|
| 1 | Go backend is running on port 8080 | `JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080` |
| 2 | BFF is running on port 3000 | `cd bff && npm run dev` |
| 3 | SQLite database is writable | Default path is `fenixcrm.db` at repo root; ensure no other process holds a write lock |
| 4 | `go` is on `$PATH` | The runner spawns `go run ./scripts/e2e_seed_mobile_p2.go` to create real seed data |
| 5 | BFF dependencies installed | `cd bff && npm install` |

Both service health endpoints are checked automatically before any request is made. If either fails, the runner exits with a clear error before wasting time on seeding.

---

## Running

```bash
# From repo root or bff/
npm run http-snapshots          # uses defaults: Go=http://localhost:8080, BFF=http://localhost:3000

# Override service URLs
FENIX_GO_URL=http://localhost:9090 \
FENIX_BFF_URL=http://localhost:4000 \
npm run http-snapshots

# Disable fixture/screenshot mode (enabled by default)
FENIX_SNAPSHOTS_FIXTURE_MODE=0 npm run http-snapshots

# Override SSE capture timeout (default: 5000ms)
FENIX_SNAPSHOTS_SSE_TIMEOUT_MS=10000 npm run http-snapshots
```

### Output locations

| Artifact | Path |
|----------|------|
| Raw JSON artifacts | `bff/artifacts/http-snapshots/raw/<group>/<name>.json` |
| SSE artifacts | `bff/artifacts/http-snapshots/raw/<group>/<name>.sse.json` |
| HTML report | `bff/artifacts/http-snapshots/report.html` |
| Markdown index | `bff/artifacts/http-snapshots/index.md` |

Open `report.html` in a browser — it has a sidebar grouped by endpoint category and a detail panel with request/response bodies.

---

## How to add an endpoint

All endpoints are declared in [`catalog.ts`](catalog.ts). Adding a new endpoint requires **only one new object** in the `catalog` array — no runner or report changes needed.

### Minimal example (static path, no auth)

```ts
{
  name: 'my-endpoint',       // unique slug, becomes the artifact filename
  group: 'my-group',         // sidebar group in the HTML report
  method: 'GET',
  path: '/bff/my/endpoint',
  auth: false,
  expectedStatus: 200,
},
```

### Authenticated endpoint with dynamic path

```ts
{
  name: 'deal-detail',
  group: 'deals',
  method: 'GET',
  path: '/bff/api/v1/deals/:id',
  pathParams: (seed) => ({ id: seed.deal.id }),   // resolved from seeder output
  auth: true,                                      // injects Authorization: Bearer <token>
  expectedStatus: 200,
},
```

### POST with seed-derived body

```ts
{
  name: 'case-create',
  group: 'cases',
  method: 'POST',
  path: '/bff/api/v1/cases',
  auth: true,
  body: (seed) => ({
    accountId: seed.account.id,
    subject: 'Snapshot test case',
  }),
  expectedStatus: 201,
},
```

### SSE endpoint

```ts
{
  name: 'copilot-stream',
  group: 'copilot',
  method: 'POST',
  path: '/bff/api/v1/copilot/stream',
  auth: true,
  body: (seed) => ({ query: 'snapshot test', accountId: seed.account.id }),
  sse: { maxEvents: 5, timeoutMs: 8000 },   // collects up to 5 events or until timeout
  expectedStatus: 200,
},
```

### Available seed fields

The seeder (`scripts/e2e_seed_mobile_p2.go`) creates real entities and returns:

```
seed.credentials.{email, password}
seed.auth.{token, userId, workspaceId}
seed.account.id
seed.contact.{id, email}
seed.lead.id
seed.deal.id / seed.staleDeal.id
seed.pipeline.id / seed.stage.id
seed.case.{id, subject} / seed.resolvedCase.{id, subject}
seed.agentRuns.{completedId, handoffId, deniedByPolicyId}
seed.inbox.{approvalId, rejectApprovalId, signalId}
seed.workflow.id
```

---

## Interpreting the report

### HTML report sidebar badges

| Badge color | Meaning |
|-------------|---------|
| Green `2xx` | Response status is 2xx |
| Yellow `3xx` | Response status is 3xx |
| Red `4xx`/`5xx` | Response status is 4xx or 5xx |

### Markdown index icons

| Icon | Meaning |
|------|---------|
| ✅ | Actual HTTP status matches `expectedStatus` |
| ❌ | Actual HTTP status does **not** match `expectedStatus` |

An endpoint can return a green badge (e.g. 401) and still show ✅ if its `expectedStatus` is also 401. The badge reflects the HTTP status; the icon reflects whether the behavior matched the contract.

### Redaction in artifacts

Raw artifacts are deterministically redacted so diffs stay stable:

| Value type | Replaced with |
|------------|--------------|
| JWT / Bearer token | `Bearer <REDACTED>` |
| Any UUID | `<uuid:1>`, `<uuid:2>`, … (stable per-run) |
| Timestamps (ISO 8601) | `<timestamp>` |
| Email addresses | `<email>` |
| Snapshot IP (`10.244.x.x`) | `<snapshot-ip>` |
| Fields: `password`, `token`, `secret`, `apiKey`, `accessToken` | `<REDACTED>` |

---

## Troubleshooting

### `Dependency health check failed`

One or both services are not reachable. Start them per the preconditions table above, then re-run.

### `Seeder exited with code 1`

The Go seeder (`scripts/e2e_seed_mobile_p2.go`) failed. Common causes:
- Database locked by another process — stop other running instances.
- `JWT_SECRET` not set or too short — must be ≥ 32 characters.
- Missing migration — run `go run ./cmd/fenix migrate` first.

### `Seeder output missing auth.token`

The seeder ran but printed unexpected output (warnings before the JSON). Check that the seeder's stdout contains only valid JSON (warnings go to stderr by convention).

### SSE entries show empty events

The service may not have emitted events within `FENIX_SNAPSHOTS_SSE_TIMEOUT_MS`. Increase the timeout:

```bash
FENIX_SNAPSHOTS_SSE_TIMEOUT_MS=15000 npm run http-snapshots
```

### Report shows no snapshots

The report generator reads artifacts from `bff/artifacts/http-snapshots/raw/`. If that directory is empty (e.g. first run, all entries errored out), the HTML page will show a "No snapshots captured yet" message. Check the runner's stdout for per-entry errors.

### Exit code 1 in CI

The runner exits non-zero if any catalog entry's actual status differs from `expectedStatus`. Check `bff/artifacts/http-snapshots/index.md` for ❌ rows to find the offending endpoint.
