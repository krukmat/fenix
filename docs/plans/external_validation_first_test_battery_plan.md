---
doc_type: plan
title: "External Validation First Test Battery Plan"
status: active
created: 2026-06-29
task: EXTVAL-PLAN-001
depends_on:
  - docs/external_validation_environment_audit.md
---

# External Validation First Test Battery Plan

## Purpose

Prepare and execute the first test battery for an external validation environment where this MacBook Pro hosts the backend and BFF. The battery must validate real product behavior, not Maestro screenshot fixtures or mocked agent responses.

## Validation Principle

Maestro may drive the UI, but it must not provide product truth. Real validation evidence must come from backend/BFF responses, persisted SQLite state, audit/usage records, agent runs, and mobile UI state connected to the real BFF.

For behavioral validation:

- `SCREENSHOT_MODE=false`
- `ENABLE_SCREENSHOT_FIXTURES=false`
- `EXPO_PUBLIC_E2E_MODE` unset
- no call to `/bff/api/v1/copilot/internal/screenshot-mode`
- no `mobile/maestro/seed-and-run.sh` as authoritative evidence for LLM or agent behavior

## Target Stack

Backend/BFF host:

- macOS host native processes for first battery.
- Go backend on `localhost:8080`.
- BFF on `localhost:3000`.
- SQLite database under `./data/external-validation/fenixcrm.db`.
- Ollama on host `localhost:11434`.

Agentic runtime:

- Primary chat model: `gemma4:26b-a4b-it-qat`.
- Fallback chat model: `gemma4:12b-mlx` (installed on host; supersedes the earlier `gemma4:12b-it-qat` reference, which is not installed).
- Required embedding model: `nomic-embed-text`.
- Runtime: Ollama, not LM Studio or vLLM for the first battery.

Mobile:

- Android emulator/device.
- Expo/React Native app built without `EXPO_PUBLIC_E2E_MODE=1`.
- `EXPO_PUBLIC_BFF_URL=http://10.0.2.2:3000`.
- Maestro optional as a driver after Java/ADB are fixed, but login and agent flows must use real UI and backend paths.

## Setup Phase

### S0.1 Local Tooling

Install or fix shell wiring until these commands pass. Commands marked `[required]` are hard gates; `[optional]` are advisory for the first battery only (Maestro is not on the critical path for manual API + UI validation):

```sh
go version          # [required]
node --version      # [required]
npm --version       # [required]
java -version       # [required]
adb version         # [required]
ollama --version    # [required] — binary must be on PATH before starting daemon
maestro --version   # [optional] — Maestro is not required for first battery (manual flows); fix for scripted Maestro runs
```

Ollama daemon readiness (`curl -fsS http://localhost:11434/api/tags`) is a separate prerequisite: run `ollama serve` first, then verify. It is required before any model or embedding check, but is a daemon concern, not a toolchain PATH concern.

Required actions:

```sh
brew install go
brew link --overwrite go
export JAVA_HOME=/opt/homebrew/opt/openjdk/libexec/openjdk.jdk/Contents/Home
export PATH="/opt/homebrew/opt/openjdk/bin:$HOME/Library/Android/sdk/platform-tools:$PATH"
export ANDROID_HOME="$HOME/Library/Android/sdk"
```

If Docker Compose is used:

```sh
colima start
docker info
```

Native backend/BFF is preferred for this first battery. The Dockerfile Go version mismatch has been resolved (EXTVAL-DOCKER-001): `deploy/Dockerfile` now uses `golang:1.25.10-alpine`, matching `go.mod`.

### S0.2 Repo Dependencies

Run:

```sh
make install-hooks
cd bff && npm ci
cd ../mobile && npm ci
```

The existing `mobile/node_modules` may be reused only if `npm ci` is intentionally skipped and recorded as a risk.

### S0.3 Ollama Models

Run:

```sh
ollama pull nomic-embed-text
ollama show gemma4:26b-a4b-it-qat
ollama show gemma4:12b-mlx
```

Smoke test chat model:

```sh
curl -fsS http://localhost:11434/api/chat \
  -H 'Content-Type: application/json' \
  -d '{"model":"gemma4:12b-mlx","stream":false,"messages":[{"role":"user","content":"Return exactly: ok"}]}'
```

Use `gemma4:12b-mlx` for fast smoke checks and `gemma4:26b-a4b-it-qat` for the actual external validation run.

### S0.4 Validation Env

Create a local untracked env file or shell export set:

```sh
JWT_SECRET=<validation-secret-at-least-32-chars>
DATABASE_URL=./data/external-validation/fenixcrm.db
PORT=8080
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_CHAT_MODEL=gemma4:26b-a4b-it-qat
OLLAMA_MODEL=nomic-embed-text
CHAT_PROVIDER=ollama
EMBED_PROVIDER=ollama
BFF_ORIGIN=http://localhost:3000
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000,exp://127.0.0.1:8081
BFF_PORT=3000
BACKEND_URL=http://localhost:8080
NODE_ENV=production
SESSION_SECRET=<validation-admin-session-secret>
BFF_CORS_ALLOWED_ORIGINS=http://localhost:8081,exp://127.0.0.1:8081,http://localhost:3000
SCREENSHOT_MODE=false
ENABLE_SCREENSHOT_FIXTURES=false
EXPO_PUBLIC_BFF_URL=http://10.0.2.2:3000
```

Do not set `EXPO_PUBLIC_E2E_MODE`.

## Preflight QA Gates

Run these before any externally observed test:

```sh
bash scripts/check-no-inline-eslint-disable.sh
cd mobile && npm run typecheck
cd mobile && npm run lint
cd mobile && npm run quality:arch
cd mobile && npm run test:coverage
cd bff && npm run build -- --noEmit
cd bff && npm run lint
cd bff && npm run test:coverage
go test ./...
make eval
```

If time allows, add:

```sh
make ci
```

If any command cannot run because of environment limits, stop and record the blocker before proceeding.

## Runtime Start Sequence

Start Ollama:

```sh
ollama serve
```

Start backend:

```sh
JWT_SECRET="$JWT_SECRET" \
DATABASE_URL="$DATABASE_URL" \
OLLAMA_BASE_URL="$OLLAMA_BASE_URL" \
OLLAMA_CHAT_MODEL="$OLLAMA_CHAT_MODEL" \
OLLAMA_MODEL="$OLLAMA_MODEL" \
CHAT_PROVIDER=ollama \
EMBED_PROVIDER=ollama \
go run ./cmd/fenix serve --port 8080
```

Verify backend:

```sh
curl -fsS http://localhost:8080/health
curl -fsS http://localhost:8080/readyz
```

Start BFF:

```sh
cd bff
BACKEND_URL=http://localhost:8080 \
BFF_PORT=3000 \
NODE_ENV=production \
SESSION_SECRET="$SESSION_SECRET" \
SCREENSHOT_MODE=false \
ENABLE_SCREENSHOT_FIXTURES=false \
npm run build
BACKEND_URL=http://localhost:8080 \
BFF_PORT=3000 \
NODE_ENV=production \
SESSION_SECRET="$SESSION_SECRET" \
SCREENSHOT_MODE=false \
ENABLE_SCREENSHOT_FIXTURES=false \
npm run start
```

Verify BFF:

```sh
curl -fsS http://localhost:3000/bff/health
```

Negative fixture check:

```sh
curl -s -o /dev/null -w '%{http_code}\n' -X POST http://localhost:3000/bff/api/v1/copilot/internal/screenshot-mode \
  -H 'Content-Type: application/json' \
  -d '{"enabled":false}'
```

Expected result: `404`. In the external validation environment, screenshot fixtures must stay unreachable because `NODE_ENV=production` and `ENABLE_SCREENSHOT_FIXTURES=false`.

Local Maestro screenshot runs may still opt in by starting a separate non-production BFF with `ENABLE_SCREENSHOT_FIXTURES=true`.

## Test Battery

### T1 Backend/BFF Smoke And Auth

Goal: prove core API and BFF proxy are live with real DB state.

Steps:

1. Register or login validation operator through BFF auth route.
2. Store bearer token for manual API checks.
3. Create account, contact, deal, and support case through BFF proxied `/bff/api/v1/*` routes.
   For workspaces registered from a checkout that includes `ADR032-BOOTSTRAP-IMPL-001`,
   use the default `deal` and `case` pipelines created by registration rather than
   manually provisioning pipelines first.
4. Verify data through read endpoints and SQLite persistence.
5. Verify audit events are created for protected API calls.

Evidence:

- `curl` request/response logs with redacted token.
- Backend `/health` and `/readyz`.
- BFF `/bff/health`.
- SQLite row counts for created workspace entities.
- Audit endpoint output.

Go/no-go:

- No 5xx in auth or CRUD.
- BFF health reports backend reachable.
- No fixture responses.

### T2 Knowledge And Evidence

Goal: prove embeddings and evidence pack generation work against real Ollama embedding model.

Steps:

1. Ingest a validation knowledge document linked to a deal and one linked to a support case.
2. Run knowledge search/evidence endpoint with a query that should match.
3. Confirm evidence source count, score, and provenance.
4. Confirm weak query returns low evidence and does not fabricate.

Evidence:

- `/api/v1/knowledge/ingest`, `/search`, and `/evidence` responses.
- Ollama model list includes `nomic-embed-text`.
- Backend logs show no missing model errors.

Go/no-go:

- Evidence pack is non-empty for seeded relevant data.
- Weak evidence path is handled without crash.

### T3 Support Agent Real Trigger

Goal: validate the support agent path with real backend state and no Maestro/BFF fixture.

Current bootstrap note: `ADR032-BOOTSTRAP-IMPL-001` provisions first-user role
assignment and default `deal`/`case` pipelines for newly registered workspaces.
It does not seed `agent_definition`; support-agent trigger validation still
requires an active support-agent definition until that separate provisioning gap
is resolved.

Steps:

1. Create a support case with non-empty `subject`, priority, account, and contact.
2. Ingest support knowledge aligned to the case subject/customer question.
3. Trigger `/bff/api/v1/agents/support/trigger` from API first.
4. Verify run status, output, evidence ids, reasoning trace, tool calls, usage event, and audit event.
5. Repeat via mobile UI button `support-trigger-agent-button`.

Evidence:

- Agent run detail.
- Usage events for the run.
- Audit events.
- Case status/handoff/approval side effects.
- Mobile screenshot/video only as secondary evidence.

Go/no-go:

- Trigger uses `{case_id, customer_query}`.
- Run does not fail with missing case or missing query.
- Output is not the BFF screenshot fixture text.

### T4 Approval And Handoff Path

Goal: prove human-in-the-loop control is visible and operational.

Steps:

1. Use a high-priority or weak-evidence support scenario that should produce approval or handoff.
2. Open mobile Inbox and BFF admin Approvals.
3. Approve or reject from one surface.
4. Verify run, approval status, audit, and inbox refresh.

Evidence:

- Approval row/status before and after decision.
- BFF admin page or API response.
- Mobile inbox state.
- Audit event.

Go/no-go:

- Approval decision changes persistent state.
- A non-authorized path does not silently approve.

### T5 Sales Brief And Copilot Real LLM

Goal: validate LLM-backed copilot/sales-brief without `screenshotMode`.

Steps:

1. Use a deal/account with ingested knowledge.
2. Call `/bff/api/v1/copilot/sales-brief`.
3. Open mobile Sales Brief.
4. Open Copilot contextual route and ask a bounded question.
5. Confirm evidence-backed answer or abstention.

Evidence:

- BFF response body.
- Mobile screen state.
- Backend evidence pack and audit/usage records.

Go/no-go:

- No response contains `Snapshot fixture response` or `fixture-source-001`.
- Timeout stays under agreed external demo threshold or is documented as performance risk.

### T6 Deterministic Eval Regression

Goal: prove deterministic governance/eval fixtures still pass before external tests are trusted.

Steps:

1. Run `make eval`.
2. Run any scenario-specific regression command used by deterministic eval docs.
3. Capture pass/fail and output artifact paths.

Evidence:

- Command output.
- Eval run records if executed through API.

Go/no-go:

- Policy compliance threshold passes.

### T7 Mobile Real-Mode Navigation

Goal: prove mobile is not relying on E2E auth or query-idle behavior for validation.

Steps:

1. Start Expo/mobile without `EXPO_PUBLIC_E2E_MODE`.
2. Login through visible auth UI.
3. Navigate to Support, Inbox, Activity, Sales Brief, Governance.
4. Trigger support agent from support case detail.
5. Observe state refresh from BFF.

Evidence:

- Environment dump showing `EXPO_PUBLIC_E2E_MODE` unset.
- Mobile logs.
- BFF/backend logs.
- Optional Maestro run that does not call `e2e-bootstrap` or screenshot mode.

Go/no-go:

- No auth bootstrap deep link.
- No screenshot-mode toggle.
- UI state matches backend persisted state.

## Evidence Packet

Each battery run should produce:

- Environment manifest: OS, hardware, Go/Node/npm/Java/Maestro/Ollama versions, model list.
- Redacted env manifest: variable names and non-secret values only.
- Service logs: backend, BFF, Ollama tail.
- API transcript: redacted curl outputs.
- SQLite evidence: row counts and selected IDs.
- Mobile evidence: screenshots/video only after backend truth is captured.
- Known deviations: exact failures, skipped gates, or fixture-only flows.

## External Validation Go/No-Go Checklist

Go:

- All setup commands pass.
- All preflight QA gates pass or are explicitly scoped out before external observation.
- Backend `/health` and `/readyz` pass.
- BFF `/bff/health` passes.
- `nomic-embed-text` is installed.
- Primary/fallback chat models are installed.
- `SCREENSHOT_MODE=false`.
- `ENABLE_SCREENSHOT_FIXTURES=false`.
- `EXPO_PUBLIC_E2E_MODE` unset.
- Screenshot-mode endpoint returns `404` in the production validation environment.

No-Go:

- Go or Java missing.
- BFF dependencies missing.
- Ollama model mismatch.
- Any validation relies on BFF fixture responses.
- Mobile build uses `.env.e2e` for real product claims.
- Docker/Compose path is used before verifying the Go version in `deploy/Dockerfile` (currently `golang:1.25.10-alpine`, matching `go.mod`).

## Follow-Up Tasks

Recommended follow-up task records:

1. ~~Align Dockerfile Go version with `go.mod`.~~ Done via EXTVAL-DOCKER-001: `deploy/Dockerfile` now uses `golang:1.25.10-alpine`.
2. Align `.env.example` and Compose model defaults with locally validated models.
3. Add a dedicated external-validation Maestro flow that does not use `seed-and-run.sh`, `e2e-bootstrap`, or screenshot-mode.
4. Add a readiness script that checks Go, Java, Android SDK, Ollama models, backend/BFF health, and fixture-mode disabled state.
