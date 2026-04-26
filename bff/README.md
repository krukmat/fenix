# BFF Gateway

Express.js thin proxy layer between the React Native mobile app and the Go backend.

**Responsibilities**: auth relay, request aggregation, SSE proxy, mobile-specific headers. Zero business logic, zero DB access.

---

## Quick start

```bash
npm install
npm run dev        # development with hot-reload (ts-node + nodemon)
npm run build      # compile to dist/
npm start          # run compiled output
```

Default port: `3000`. Proxies to Go backend at `http://localhost:8080` by default.

---

## Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Start with hot-reload |
| `npm run build` | TypeScript compile → `dist/` |
| `npm start` | Run compiled server |
| `npm test` | Jest unit tests |
| `npm run test:coverage` | Jest with coverage report |
| `npm run lint` | ESLint (zero warnings) |
| `npm run http-snapshots` | Capture live HTTP snapshots — see below |

---

## HTTP Snapshots

The snapshot runner hits every catalogued BFF endpoint against real running services, stores redacted JSON artifacts, and generates an HTML report for visual inspection and diff-based regression detection.

**Full documentation**: [`scripts/snapshots/README.md`](scripts/snapshots/README.md)

### Quick run

```bash
# Both services must be running first:
# Go:  JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080
# BFF: npm run dev

npm run http-snapshots
# Report → bff/artifacts/http-snapshots/report.html
# Index  → bff/artifacts/http-snapshots/index.md
```

### Adding an endpoint

Add one entry to [`scripts/snapshots/catalog.ts`](scripts/snapshots/catalog.ts) — no other files need to change. See [`scripts/snapshots/README.md`](scripts/snapshots/README.md) for the full `CatalogEntry` reference.

---

## Tests

```bash
npm test                  # all unit tests
npm run test:coverage     # with coverage
npm run test:bdd          # BDD scenarios only
```

Tests live in `tests/`. Snapshot runner tests live in `tests/snapshots/`.
