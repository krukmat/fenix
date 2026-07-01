---
doc_type: audit
title: "External Validation Environment Readiness Audit"
status: complete
created: 2026-06-29
task: EXTVAL-PLAN-001
---

# External Validation Environment Readiness Audit

## Scope

This audit reviews the current machine as the backend/BFF host for an external validation dry run, the local agentic runtime options, and the mobile app/Maestro wiring. It is evidence-only planning work. No dependency installation, product-code change, or validation execution was performed.

## Executive Finding

The machine is suitable for a first external-validation environment after setup, but it is not ready right now. The strongest blocker is environment wiring, not product architecture: Go is not on `PATH`, Java is not linked for Maestro, Docker/Colima is not running, BFF dependencies are not installed, and the default Ollama embedding model is missing locally.

The main product-code risk is BFF screenshot fixture mode. `bff/src/routes/copilot.ts` exposes a runtime toggle at `/bff/api/v1/copilot/internal/screenshot-mode` that causes copilot SSE and sales-brief responses to return fixtures. It is useful for Maestro screenshots, but it must be disabled and preferably removed or auth-gated before any externally observed validation of agent behavior.

## Local Host Baseline

Evidence from local commands on 2026-06-29:

| Area | Evidence | Readiness |
|---|---|---|
| Hardware | MacBook Pro `Mac17,2`, Apple M5, 10 CPU cores, 10 GPU cores, 32 GB memory. | Good for native backend/BFF and medium local LLMs. |
| Disk | `/` has ~751 GiB available. | Good for local models, SQLite data, build artifacts. |
| OS | macOS 26.5.1 build 25F80. | Good. |
| Node/npm | `node v22.23.0`, `npm 10.9.8`. | Good for BFF/mobile. |
| Go | `go version` fails: command not found. `go.mod` requires `go 1.25.0` and toolchain `go1.25.10`. | Blocker. Install or link Go 1.25.10. |
| Docker | Docker CLI exists, but daemon fails to connect to `~/.colima/default/docker.sock`. `colima 0.10.3` is installed. | Blocker if using Compose. Start Colima or use native processes. |
| Java | Homebrew `openjdk 26.0.1` exists, but `java -version` fails because Java is not linked/in `PATH`. | Blocker for Maestro. |
| Android SDK | `~/Library/Android/sdk` and `platform-tools/adb` exist, but `ANDROID_HOME` is unset and `adb` is not on `PATH`. | Blocker for Maestro until env vars are exported. |
| Xcode | `xcodebuild -version` fails because active developer dir is CommandLineTools only. | Not blocking for Android-only validation. |
| Maestro | `/opt/homebrew/bin/maestro` exists, but `maestro --version` fails due missing Java runtime in shell. | Blocker. |
| Ollama | `ollama 0.30.10` responds at `localhost:11434`. | Good base. |
| Loaded Ollama models | `ollama ps` shows no currently loaded model. | Expected idle state. |
| Installed Ollama models | `gemma4:26b-a4b-it-qat`, `gemma4:12b-it-qat`, `gemma3:27b`. | Good for chat/tool candidates. Missing embedding model. |
| Dependencies | `mobile/node_modules` exists; `bff/node_modules` does not. | BFF needs `npm ci`. |
| Git hooks | `.git/hooks/pre-push`, `.git/hooks/pre-commit`, `.git/hooks/prepare-commit-msg` are missing. | Run `make install-hooks` after Go/path setup or directly if Make is available. |

## Backend And BFF Findings

### Backend Runtime

The backend is Go + SQLite. `cmd/fenix/main.go` opens `DATABASE_URL` or `./data/fenixcrm.db`, applies migrations, and starts the HTTP server. `internal/server/server.go` builds chat and embedding providers at startup and wires background workers for embeddings/reindexing/scheduler through router runtime.

Readiness risks:

- `go.mod` requires Go 1.25.10, but Go is not available in the shell.
- `deploy/Dockerfile` now uses `golang:1.25.10-alpine`, matching `go.mod` toolchain (resolved by EXTVAL-DOCKER-001).
- `internal/infra/config/config.go` defaults to `OLLAMA_CHAT_MODEL=gemma4:e4b` and `OLLAMA_MODEL=nomic-embed-text`.
- `.env.example`, `docker-compose.yml`, and `docker-compose.prod.yml` contain model defaults that do not match the local model list. Compose defaults use `llama3.2:3b-instruct-q4_K_M`; `.env.example` uses `gemma4:e4b`; neither is installed locally.
- `nomic-embed-text` is not installed locally, but embeddings are required by the knowledge/evidence paths and by the mobile seed script.

Required validation env values:

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
```

### BFF Runtime

`bff/src/config.ts` requires `BACKEND_URL` at startup, defaults `BFF_PORT` to `3000`, and defaults CORS origins to local web/Expo origins. `bff/src/app.ts` wires `/bff/health`, auth, copilot, builder, admin, approval aliases, inbox aggregation, and transparent proxy routing to Go.

Readiness risks:

- `bff/node_modules` is absent. Run `cd bff && npm ci`.
- `SESSION_SECRET` falls back to `fenix-admin-dev-secret-change-in-prod`; validation should set a real value.
- `BACKEND_URL` must be set explicitly for `validateConfig()`.
- `bff/src/routes/copilot.ts` has `SCREENSHOT_MODE` and runtime `POST /internal/screenshot-mode`. When enabled, copilot and sales-brief return deterministic fixtures and bypass the LLM.
- `bff/src/routes/builder.ts` uses fixture projection for standalone/no-live graph fallback. This affects builder UI preview, not the core agent validation path.

Required BFF env values:

```sh
BFF_PORT=3000
BACKEND_URL=http://localhost:8080
NODE_ENV=production
SESSION_SECRET=<validation-admin-session-secret>
BFF_CORS_ALLOWED_ORIGINS=http://localhost:8081,exp://127.0.0.1:8081,http://localhost:3000
SCREENSHOT_MODE=false
```

## Agentic Runtime Assessment

Project requirements from `agentic_crm_requirements_agent_ready.md` and runtime code:

- Evidence-first behavior with abstention/handoff when evidence is weak.
- Action execution through registered tools.
- Policy/RBAC/ABAC and approvals.
- Audit, usage, metrics, and eval traces.
- Self-hosted or BYO-model operation.

Current code supports Ollama and OpenAI-compatible chat providers. Embeddings are Ollama-only today (`internal/infra/llm/factory.go`). That makes Ollama the lowest-friction runtime for this machine.

Local models from `ollama show`:

| Model | Parameters | Quantization | Context | Capabilities | Recommendation |
|---|---:|---|---:|---|---|
| `gemma4:26b-a4b-it-qat` | 25.2B | Q4_0 | 262144 | completion, vision, tools, thinking | Primary validation chat model. Best local quality/capability balance on 32 GB unified memory. |
| `gemma4:12b-it-qat` | 11.9B | Q4_0 | 262144 | completion, vision, audio, tools, thinking | Fallback/smoke model when latency or memory pressure matters. |
| `gemma3:27b` | 27.4B | Q4_K_M | 131072 | completion, vision | Not preferred for agentic validation because local metadata does not advertise tools/thinking. |

External source checks:

- Ollama officially supports tool calling/function calling for tool-integrated models: <https://docs.ollama.com/capabilities/tool-calling>.
- Ollama provides OpenAI-compatible endpoints, useful if the backend later switches `CHAT_PROVIDER=openai-compat`: <https://docs.ollama.com/api/openai-compatibility>.
- Ollama context length defaults vary by available VRAM; its docs note agent/coding workloads may require large context: <https://docs.ollama.com/context-length>.
- LM Studio can serve local models with OpenAI-compatible endpoints, but it adds another server/runtime to operate: <https://lmstudio.ai/docs/developer/core/server>.
- vLLM has Apple Silicon paths, but macOS support involves vLLM-Metal/source-oriented setup and is heavier for this first validation: <https://docs.vllm.ai/projects/vllm-metal/en/latest/installation/>.

Recommendation:

Use Ollama as the agentic runtime, `gemma4:26b-a4b-it-qat` as the primary chat model, `gemma4:12b-it-qat` as fallback, and `nomic-embed-text` as the required embedding model after pulling it. Do not use LM Studio or vLLM for the first battery unless Ollama fails a specific validation requirement.

## Mobile And Maestro Findings

### Runtime Service Wiring

`mobile/src/services/api.client.ts` uses `EXPO_PUBLIC_BFF_URL` and falls back to `http://10.0.2.2:3000`, which is correct for Android emulator to host-machine BFF access. `mobile/.env.example` and `mobile/.env.e2e` both set that value.

The app does not appear to use MSW or network mocks in production source. Targeted search over `mobile/src` and `mobile/app` found only:

- `EXPO_PUBLIC_E2E_MODE` in `mobile/app/_layout.tsx` and `mobile/app/e2e-bootstrap.tsx`.
- BFF URL config in `mobile/src/services/api.client.ts`.
- Test IDs and seed references under `mobile/e2e` and `mobile/maestro`.

### E2E/Maestro Isolation

`mobile/app/_layout.tsx` changes behavior when `EXPO_PUBLIC_E2E_MODE=1`: Sentry is disabled, splash behavior changes, and React Query auto-refetch/retry behavior is reduced to keep Detox/Espresso idle.

`mobile/app/e2e-bootstrap.tsx` accepts auth injection only when `EXPO_PUBLIC_E2E_MODE=1`; otherwise it redirects to `/login` without mutating auth state.

`mobile/maestro/seed-and-run.sh` uses deterministic seed data, composes an `e2e-bootstrap` deep link, and toggles BFF screenshot mode. This runner is appropriate for screenshot capture and visual audits, but it is not valid evidence of real auth flow or real LLM-backed copilot/sales-brief behavior.

For external validation:

- Build/run mobile without `EXPO_PUBLIC_E2E_MODE=1` for real product behavior.
- Do not use `mobile/maestro/seed-and-run.sh` as the authoritative validation runner for agent behavior.
- If Maestro is used, use it as an input driver only: launch app, login through UI, navigate, tap, and assert real backend side effects.
- Keep `SCREENSHOT_MODE=false` and do not call `/screenshot-mode`.

### Support Agent Contract

The old documented gap where mobile sent generic `entity_type/entity_id` for support trigger appears corrected in current code:

- `mobile/src/services/api.agents.ts` calls `/bff/api/v1/agents/support/trigger` with `{case_id, customer_query, language?, priority?}`.
- `mobile/src/hooks/useWedge.ts` maps `{caseId, customerQuery}` into the canonical backend payload.
- `mobile/src/components/support/SupportCaseDetailContent.tsx` triggers the support agent with `caseData.id` and `caseData.subject ?? ''`.
- `internal/api/handlers/agent.go` rejects missing `case_id` or `customer_query`.

Remaining validation condition: the case used in external tests must have a non-empty subject or the trigger will produce a backend 400.

## Required Install/Setup List

Required before the first battery:

1. Install or link Go 1.25.10 so `go version` works.
2. Run `make install-hooks` once hooks can be installed.
3. Run `cd bff && npm ci`.
4. Ensure `mobile` dependencies are current with `cd mobile && npm ci` if the existing `node_modules` state is not trusted.
5. Link Homebrew OpenJDK into the shell, set `JAVA_HOME=/opt/homebrew/opt/openjdk/libexec/openjdk.jdk/Contents/Home`, and add `/opt/homebrew/opt/openjdk/bin` to `PATH`.
6. Set `ANDROID_HOME=$HOME/Library/Android/sdk` and add `$ANDROID_HOME/platform-tools` to `PATH`.
7. Verify `maestro --version` and `adb version`.
8. Pull the embedding model: `ollama pull nomic-embed-text`.
9. Keep using installed `gemma4:26b-a4b-it-qat`; validate with a short chat request before running agent tests.
10. Start Colima/Docker only if using Compose, or prefer native backend/BFF processes for the first battery.
11. Create a local, uncommitted validation env file with real validation secrets and explicit model names.

## Go/No-Go Risks

No-Go for external validation until fixed or consciously scoped out:

- Go not available in shell.
- Java/Maestro not available in shell if mobile automation is part of the battery.
- `nomic-embed-text` missing.
- BFF dependencies missing.
- `SCREENSHOT_MODE` enabled or screenshot-mode endpoint used during behavioral validation.
- Compose image still uses Go 1.24 if Docker path is chosen.

Proceed with caveats:

- Native process validation can proceed without Docker if Go, Node, BFF dependencies, and Ollama are configured.
- Mobile validation can proceed without Xcode because `mobile/app.json` targets Android.
- Maestro screenshot flow can remain available as a visual-only harness, but it must not be cited as evidence for real agent/copilot behavior.
