# Local Environment Validation — QA Gates Full Checkup

**Project**: FenixCRM / Agentic CRM OS
**Purpose**: Procedure for restoring and validating the full local development environment after a machine change or prolonged absence. Execute the applicable steps in order; skip destructive sync actions if you need to preserve local work.

---

## 1. Repository Synchronization (Optional / Destructive)

```bash
git fetch origin main
git reset --hard origin/main
git clean -fd
```

> Warning: this step removes local modifications and untracked files. Skip it if the worktree contains local work you want to keep.

Verify:
```bash
git status          # must be: "nothing to commit, working tree clean"
git log --oneline -1
```

---

## 2. Environment Diagnostics

Required tools and expected versions:

| Tool | Required Version | Install Path | Verify |
|---|---|---|---|
| Go | 1.25.0 | system | `go version` |
| Node.js | 20+ (LTS recommended) | system | `node --version` |
| npm | 10+ | system | `npm --version` |
| golangci-lint | 1.64.8 | `~/go/bin` | `golangci-lint version` |
| gocyclo | installed | `~/go/bin` | `which gocyclo` |
| sqlc | installed | `~/go/bin` | `sqlc version` |
| Python | 3.11+ | `.venv` | `.venv/bin/python --version` |
| Doorstop | any | `.venv/bin` | `.venv/bin/doorstop --version` |
| Schemathesis | any | `.venv/bin` | `.venv/bin/schemathesis --version` |

> The Go version installed must match `go.mod` exactly. Mismatches cause `golangci-lint` build errors.
> The repo does not pin Node.js via `.nvmrc` or `engines`; use a current LTS when possible.
> `sqlc` is only required when regenerating database code. The checked-in generated files were last produced by `sqlc v1.28.0`, so avoid regenerating casually with a newer version unless you intend to update generated output.

---

## 3. Setup Procedure

### 3.1 Install missing Go tools

```bash
# gocyclo — required for make complexity
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

# golangci-lint — required for make lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# sqlc — only needed when regenerating sqlc output
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Verify all Go tools are present
which gocyclo golangci-lint sqlc
```

Alternative: `make install-tools`

### 3.2 Install BFF dependencies

```bash
cd bff
npm ci
cd ..
```

### 3.3 Install Mobile dependencies

```bash
cd mobile
npm ci --legacy-peer-deps
cd ..
```

> `--legacy-peer-deps` is required due to React 19 peer dependency conflicts in the React Native ecosystem.

---

## 4. Go QA Gates

Execute in the following order. Each gate must pass before proceeding.

### 4.1 Format Check

```bash
make fmt-check
```

Expected: exit 0 and a `PASSED` message. If it fails, run `make fmt` to auto-fix and re-check.

### 4.2 Cyclomatic Complexity

```bash
make complexity
```

Threshold: **7** (production code only; `*_test.go` excluded).
Tool: `gocyclo`.

### 4.3 Static Analysis (Linter)

```bash
make lint
```

Linter: `golangci-lint` with `.golangci.yml`.
Active rules include: `gocognit` (≤8), `maintidx` (≥20), `nestif` (≤3), `funlen` (≤70 lines / ≤35 statements), `govet` (shadow), `gosec`, `depguard`, `exhaustive`, `errorlint`, `interfacebloat`.

### 4.4 Unit + Integration Tests

```bash
make test
```

Runs two passes:
- coverage pass on application packages with `go test -v -race -coverprofile=coverage.out ...`
- auxiliary pass on `ruleguard`, `scripts`, and `internal/infra/sqlite/sqlcgen` without `-coverprofile`

This split avoids a current Go `covdata` toolchain issue triggered by those auxiliary packages while still executing their tests.
Race detector is always active.

### 4.5 Coverage Gate

```bash
make coverage-gate
```

Minimum: **83%** on application code (excludes generated `sqlcgen/`, server/bootstrap wiring, `cmd/fenix/main.go`, `cmd/frtrace/main.go`, `internal/version/`, `ruleguard/`, and `tests/bdd/go/`).

### 4.6 Race Stability

```bash
make race-stability
```

Runs the race detector **3 times** across critical packages:
- `internal/api/handlers`
- `internal/domain/copilot`
- `internal/domain/agent`
- `internal/domain/tool`

For a faster local run during development: `RACE_STABILITY_COUNT=1 make race-stability`

---

## 5. Traceability Gates

### 5.1 FR-to-Test Trace Check

```bash
make trace-check
```

Tool: `cmd/frtrace` (Go binary).
Validates that every Functional Requirement (FR-*) in `reqs/` has at least one linked test annotation.

### 5.2 Doorstop Integrity Check

```bash
make doorstop-check
```

Tool: `.venv/bin/doorstop`.
Validates requirement document links and cross-references within `reqs/`.
Current repo behavior: the command exits `0` but may emit legacy `suspect link` warnings from the existing requirements tree. Treat a non-zero exit as failure; warnings should be reviewed but do not currently block the gate.

---

## 6. BFF QA Gates

```bash
cd bff
npm test
```

Framework: Jest + Supertest.
Tests are located in `bff/tests/`.

If you need a coverage report for the BFF, run:

```bash
cd bff
npm run test:coverage
```

---

## 7. Mobile QA Gates

```bash
cd mobile
npm run test:coverage
```

Framework: Jest with `jest.logic.config.ts`.
Runs logic and domain tests only (excludes E2E and UI component tests).
Environment flags: `CI=1 JEST_HASTE_MAP_FORCE_NODE_FS=1` (set in the npm script).

---

## 8. Contract Tests

The contract test script manages the full server lifecycle automatically — no manual server startup required.

```bash
# Build is already included as a dependency of the target
make contract-test-strict
```

**What the script does** (`tests/contract/run.sh`):
1. Starts the Go binary on port **8081** with a temporary SQLite database
2. Polls `/health` until the server is ready (30 retries at 1-second intervals)
3. Registers a test user (`contract@test.com`) and extracts a JWT token using `python3`
4. Runs `schemathesis` from `.venv` against `/docs/openapi.yaml`
5. Validates: `not_a_server_error`, `status_code_conformance`, `response_schema_conformance`, `content_type_conformance`
6. Kills the server process on exit (trap-based cleanup)

Smoke mode (faster, no fuzzing): `make contract-test`

---

## 9. E2E Tests (Detox — Android)

Requires three long-lived processes plus a Detox build/test command. Start in the following order:

### Terminal 1 — Go Backend

```bash
JWT_SECRET="test-secret-32-chars-minimum!!!" go run ./cmd/fenix serve --port 8080
```

### Terminal 2 — BFF

```bash
cd bff
npm run dev
```

### Terminal 3 — Android Emulator

```bash
emulator -avd Pixel_7_API_33
```

> The emulator AVD `Pixel_7_API_33` must be present at `~/.android/avd/`. From the emulator, the host machine is reachable at `10.0.2.2` (not `localhost`). This is configured in `mobile/.env.e2e`.

### Terminal 4 — Detox Tests

```bash
cd mobile
npm run e2e:build
npm run e2e:test
# if the APK is already built, you can run only:
make test-e2e
```

**Coverage**: auth (register/login), accounts list, deals, cases, workflows, copilot SSE, agent-runs.

**Alternative (no emulator)**: Run UI component tests instead:
```bash
cd mobile
npm run test:ui
```

---

## 10. Full Gate Summary

| Gate | Command | Pass Criterion |
|---|---|---|
| Format check | `make fmt-check` | exit 0 |
| Cyclomatic complexity | `make complexity` | no function exceeds threshold 7 |
| Static analysis | `make lint` | no issues reported |
| Unit + integration tests | `make test` | all tests pass |
| Coverage gate | `make coverage-gate` | ≥ 83% |
| Race stability | `make race-stability` | no data races detected |
| FR-to-test traceability | `make trace-check` | no unlinked FRs |
| Requirement integrity | `make doorstop-check` | exit 0; review warnings if present |
| BFF tests | `cd bff && npm test` | all tests pass |
| Mobile logic coverage | `cd mobile && npm run test:coverage` | all tests pass |
| API contract (strict) | `make contract-test-strict` | 0 schema violations |
| E2E mobile flows | `make test-e2e` | all Detox scenarios pass |

All gates must pass before considering the local environment validated.
