---
doc_type: audit
title: "External Validation Readiness — Independent Opus 4.8 Audit"
status: complete
created: 2026-06-29
task: EXTVAL-PLAN-001
audits:
  - docs/external_validation_environment_audit.md
  - docs/plans/external_validation_first_test_battery_plan.md
reviewer: claude-opus-4-8
---

# External Validation Readiness — Independent Opus 4.8 Audit

Independent review of `docs/external_validation_environment_audit.md` (the "Audit") and
`docs/plans/external_validation_first_test_battery_plan.md` (the "Plan"). Every claim below is
backed by direct inspection of repository source on 2026-06-29. No existing file was modified.

## 1. Executive Verdict

**The Audit and Plan are substantially correct and high quality.** The environment-wiring blockers
(Go absent from `PATH`, Java/Maestro unlinked, Android env unset, BFF deps missing, embedding model
missing) are all real and verified. The headline product-code risk — BFF screenshot fixture mode —
is real, and is in fact **more dangerous than the Audit states**. The Plan's core principle
("Maestro may drive the UI but must not provide product truth") is sound and the T1–T7 battery is
well-ordered.

However, there are **three material correctness problems** that would let a careless operator
declare a green run that is actually invalid or broken:

1. **`/readyz` is treated as proof of LLM/embedding readiness. It is not.** It returns HTTP 200 even
   when chat and embed providers are down; only a DB failure yields 503. And the Ollama health
   check only confirms the server is reachable (`GET /api/tags`) — it never verifies that
   `nomic-embed-text` is actually pulled. A missing embedding model surfaces only at first
   ingest/search (T2), not at startup. The Plan's go/no-go relies on `/readyz` passing, which is
   insufficient evidence.
2. **The Plan's BFF runtime config (`NODE_ENV=production`) will break browser-based admin login over
   plain HTTP.** With `NODE_ENV=production`, the session cookie is set `secure: true`, which browsers
   refuse to store over `http://localhost:3000`. T4 (Approvals via BFF admin web surface) will fail
   to persist a session and is likely to produce a confusing "login loops / not authenticated"
   failure that looks like a product bug but is a config bug introduced by the Plan itself.
3. **Screenshot fixture mode is global mutable process state mounted on two paths with no auth.** The
   Audit frames it as a "runtime toggle"; the blast radius is larger than implied and deserves a
   hard go/no-go guard, not just an env default.

**Verdict: NO-GO until setup is completed AND the three corrections above are incorporated into the
Plan.** After that, the native-process first battery is a reasonable and safe approach. Do not expose
port 3000 beyond localhost under any circumstance until the screenshot-mode endpoint is removed or
auth-gated.

## 2. Validated Findings (confirmed against source)

| # | Claim in Audit/Plan | Verification | Verdict |
|---|---|---|---|
| V1 | BFF screenshot fixture mode exists and bypasses the LLM | `bff/src/routes/copilot.ts`: `screenshotMode` gates `/chat` (L21), `/events` (L39), `/sales-brief` (L166); `POST /internal/screenshot-mode` toggles it (L157-161); fixtures `Snapshot fixture response.` / `fixture-source-001` (L100-154) | **Confirmed** |
| V2 | Go is required at 1.25.10 but the build image uses 1.24 | `go.mod`: `go 1.25.0` + `toolchain go1.25.10`; `deploy/Dockerfile` L3: `FROM golang:1.24-alpine` | **Confirmed** (mismatch real; see §3 for nuance) |
| V3 | Backend config defaults to `gemma4:e4b` / `nomic-embed-text` | `internal/infra/config/config.go` L70 (`gemma4:e4b`), L69 (`nomic-embed-text`) | **Confirmed** |
| V4 | Compose / `.env.example` model defaults do not match local models | `docker-compose.yml` L18 `llama3.2:3b-instruct-q4_K_M`; `.env.example` L22 `gemma4:e4b`; neither installed per Audit's `ollama list` | **Confirmed** |
| V5 | Embeddings are Ollama-only | `internal/infra/llm/factory.go` `NewEmbedProvider` returns Ollama or errors for any other provider | **Confirmed** |
| V6 | Support agent contract corrected (`case_id`,`customer_query`) | `mobile/src/services/api.agents.ts` L88-96 sends `{case_id, customer_query, language?, priority?}`; `internal/api/handlers/agent.go` L491-496 returns 400 on missing `case_id`/`customer_query`; route at `internal/api/routes.go:497` | **Confirmed** |
| V7 | `seed-and-run.sh` enables screenshot mode and uses `e2e-bootstrap` deep link | `mobile/maestro/seed-and-run.sh` L429-431 `{"enabled":true}`, L260 composes `fenixcrm:///e2e-bootstrap?token=...` | **Confirmed** |
| V8 | BFF `SESSION_SECRET` falls back to a dev default; `BACKEND_URL` is the only hard-required env | `bff/src/app.ts` L41 dev-secret fallback; `bff/src/config.ts` `validateConfig()` requires only `BACKEND_URL` | **Confirmed** |
| V9 | BFF deps absent / preflight scripts exist | `scripts/check-no-inline-eslint-disable.sh` present; `bff/package.json` has `lint`,`test:coverage`,`build`(=`tsc`),`start`; mobile has `typecheck`,`lint`,`quality:arch`,`test:coverage`; Makefile has `eval` (L349), `ci` (L423), `install-hooks` (L278) | **Confirmed** |
| V10 | Mobile uses real BFF URL, no MSW/network mocks in product source | `mobile/src/services/api.copilot.ts` builds `${BFF_URL}/bff/copilot/chat`; agent/sales-brief calls go to `/bff/api/v1/...` | **Confirmed** |

## 3. Disagreements / Corrections vs the Existing Audit & Plan

### C1 — `/readyz` does not prove LLM or embedding readiness (most important)
`internal/api/handlers/readyz.go`: the handler sets `status: degraded` and `chat/embed: error` when a
provider health check fails, **but returns HTTP 200 unless the database is down** (only `Database ==
error` → 503). The provider health check itself (`internal/infra/llm/ollama.go:187` `HealthCheck`)
issues `GET /api/tags` and only asserts a 200 — it never confirms `nomic-embed-text` (or the chat
model) is installed. Consequence:

- `curl -fsS http://localhost:8080/readyz` will succeed even with Ollama stopped or with no models
  pulled. The Plan's go/no-go item "Backend `/health` and `/readyz` pass" is **not** evidence of
  agentic readiness.
- A missing embedding model does not fail fast; it surfaces as a runtime error only at the first real
  `Embed` call (T2 ingest/search, T3 evidence). The Audit's framing ("embeddings are required by the
  knowledge/evidence paths") is right; the Plan's reliance on `/readyz` to gate that is wrong.

**Correction:** the go/no-go must parse the `/readyz` JSON body and require `"embed":"ok"` AND
`"chat":"ok"`, and must additionally assert `ollama list` contains `nomic-embed-text` AND the chosen
chat model. Treat T2's first successful embedding as the true readiness signal, not `/readyz`.

### C2 — `NODE_ENV=production` breaks browser admin-session login over HTTP
`bff/src/app.ts` L50 sets the session cookie `secure: config.isProduction`, and `config.isProduction`
is `NODE_ENV === 'production'`. The Plan's runtime sequence and §S0.4 env both set
`NODE_ENV=production` while serving the admin surface over `http://localhost:3000`. Browsers will not
store a `Secure` cookie sent over plaintext HTTP, so the BFF admin Approvals login (T4) will not
maintain a session. This is a **Plan-introduced defect**, not a product bug, and will waste debugging
time during a live external demo.

**Correction:** for the localhost-HTTP first battery, run the BFF with `NODE_ENV=development` (or
front it with a TLS terminator). API-token paths (mobile, curl) are unaffected; only the
cookie-backed admin web surface in T4 is impacted. The Plan should call this out explicitly.

### C3 — Dockerfile mismatch is real but "likely wrong" overstates it
The Audit says `golang:1.24-alpine` "is likely wrong." With the default `GOTOOLCHAIN=auto`, Go 1.24
honors the `toolchain go1.25.10` directive and will **download and switch to** go1.25.10 at build
time, so the image can still build given network access. The accurate statement is: the base image is
**stale and fragile** (silent toolchain download, breaks under `GOTOOLCHAIN=local` or offline builds),
not categorically broken. This does not change the recommendation (prefer native processes for the
first battery; bump the base image), but the Plan's hard No-Go "Docker/Compose path is used before
fixing Go version mismatch" is stronger than the evidence requires — it's a should-fix, not a
proven-broken.

### C4 — Screenshot mode blast radius understated
`screenshotMode` is a single module-level `let` (`bff/src/routes/copilot.ts` L127) shared across
**both** router mounts (`/bff/copilot` and `/bff/api/v1/copilot`, `bff/src/app.ts` L75 and L80) and
across **all sessions and requests** in the process, with **no auth** on the toggle endpoint
(comment at L156: "localhost only, no auth required"). Once flipped on, it stays on for the BFF
process lifetime. `seed-and-run.sh` resets it (`{"enabled":false}`, L449-451) only on a best-effort
basis (`|| true`), so a failed reset leaves a long-lived BFF serving fixtures. The Plan correctly
starts a fresh BFF and adds a negative check, which mitigates this — but the underlying endpoint
remains an unauthenticated, process-global behavioral override that must not exist on any
externally reachable port.

### C5 — Timeout layering is inconsistent across the copilot path
Not mentioned by either document. The copilot SSE relay uses a hardcoded **60s** axios timeout
(`bff/src/routes/copilot.ts` L59); the sales-brief JSON relay uses the shared Go client at **120s**
(`bff/src/services/goClient.ts` L6,L20); the mobile sales-brief client times out at **90s**
(`mobile/src/services/api.agents.ts` L141-148). With `gemma4:26b-a4b-it-qat` cold-loading plus
generation (the code comments cite ~35s observed warm), a cold copilot chat can exceed the BFF's 60s
ceiling and fail mid-stream, while a slow sales-brief can hit the mobile 90s limit before the BFF's
120s limit. T5's go/no-go ("timeout stays under agreed threshold or documented") should explicitly
record which of these three ceilings is hit, because they fail at different layers with different
symptoms.

## 4. Missing Risks

- **MR1 — 32 GB unified-memory contention.** `gemma4:26b-a4b-it-qat` (25.2B, Q4_0) resident is
  roughly 15–17 GB; add `nomic-embed-text`, an Android emulator (~2–4 GB), Metro/Node, the Go
  backend, and the BFF on a 32 GB machine. This is feasible but tight; expect model
  load/evict thrash and elevated p95 latency when the emulator and the 26B model are hot
  simultaneously. The Plan's choice of `gemma4:12b-it-qat` for smoke and 26B for the real run is
  correct; consider keeping the emulator and 26B run temporally separate to avoid swap.
- **MR2 — First-call cold-start latency vs demo expectations.** The 60s/90s/120s ceilings (C5) plus
  cold model load mean the **first** copilot/sales-brief call after backend start is the most likely
  to time out. Add a warm-up call in the Runtime Start Sequence before any observed test.
- **MR3 — `/auth/register` rate limit.** `internal/api/routes.go:117` applies a 3/hour register
  limiter. Iterative T1 setup (multiple failed registrations during debugging) can lock out the
  operator for an hour. Pre-create the validation user once, or note the limit.
- **MR4 — `make ci` is over-scoped for a dry run.** The Plan offers `make ci` "if time allows," but
  the `ci` target (Makefile L423) chains `govulncheck` (network), coverage gates, deadcode,
  contract-test-strict, RRI/docs gates, etc. Any unrelated gate failure will block the validation on
  noise. Recommend `go test ./...` + `make eval` only for the battery; treat full `make ci` as a
  separate pre-existing gate, not a validation prerequisite.
- **MR5 — Workspace bootstrap unverified.** T1 assumes self-service registration yields a usable
  workspace. `/auth/register` exists and is proxied, but whether a fresh DB auto-provisions a
  workspace + default policy/role for the new user was not confirmed in this audit. If it does not,
  T1 needs an explicit workspace-seed step. Verify before the run.
- **MR6 — No DB isolation guarantee.** The Plan points `DATABASE_URL` at
  `./data/external-validation/fenixcrm.db` but does not require starting from an empty DB. A reused
  DB can carry over screenshot-era or seed data and pollute "real DB state" evidence. Add: delete or
  rotate the validation DB file before the run, and record the starting row counts as baseline.
- **MR7 — Silent provider degradation at startup.** `internal/server/server.go` builds chat/embed
  providers at startup but (correctly) does not block on Ollama being up. Combined with C1, the
  system will start "successfully" with no working LLM. The operator must actively prove the LLM
  path; absence of a startup error is not success.

## 5. Confidence Per Major Claim

| Claim | Confidence | Basis |
|---|---|---|
| Screenshot fixture mode is a real, unauth, process-global bypass (V1, C4) | **High** | Direct read of `copilot.ts` + `app.ts` mounts |
| Env-wiring blockers (Go/Java/Android/Ollama model/BFF deps) are real | **High** | `go.mod`, Dockerfile, config, compose, package.json all confirm; host-command claims trusted from Audit (not re-run here) |
| Support agent contract is fixed (V6) | **High** | Client payload + handler 400 validation + route all read |
| `/readyz` does not prove LLM/embedding readiness (C1) | **High** | `readyz.go` 503-only-on-DB + `ollama.go` HealthCheck = `/api/tags` only |
| `NODE_ENV=production` breaks HTTP admin login (C2) | **High** | `app.ts` `secure: isProduction`; standard browser Secure-cookie behavior |
| Dockerfile is stale/fragile rather than hard-broken (C3) | **Medium-High** | `GOTOOLCHAIN=auto` auto-download behavior is well-defined but depends on build network/env |
| Timeout layering inconsistency (C5) | **High** | Three timeout constants read directly |
| 32 GB memory contention (MR1) | **Medium** | Model sizes/quant from Audit; resident footprint is an estimate, not measured here |
| Workspace bootstrap on fresh DB (MR5) | **Low** | Register route confirmed; auto-workspace provisioning not traced |
| Local host baseline table (hardware/disk/versions) | **Medium** | Not independently re-run; trusting Audit's 2026-06-29 command evidence |

## 6. Recommended Next Actions (ordered by urgency)

1. **Fix the Plan's `/readyz` go/no-go (C1, MR7).** Replace "`/readyz` passes" with: assert
   `readyz` body `"embed":"ok"` and `"chat":"ok"`, assert `ollama list` contains `nomic-embed-text`
   and the chat model, and treat the first successful T2 embedding as the real readiness gate. This
   prevents a false green.
2. **Change BFF to `NODE_ENV=development` for the localhost-HTTP battery (C2)**, or terminate TLS in
   front of it, so T4 admin-session login works. Update §S0.4 and the Runtime Start Sequence.
3. **Hard-gate screenshot mode before any non-localhost exposure (V1, C4).** Minimum for this
   battery: keep `SCREENSHOT_MODE=false`, start a fresh BFF, run the negative toggle check, and bind
   port 3000 to loopback only. Schedule the follow-up task to delete or auth-gate
   `POST /internal/screenshot-mode` and remove the dual mount.
4. **Add a model warm-up step + record which timeout fires (C5, MR2).** Issue one chat and one
   embedding call after backend start, before observed tests; document the 60s/90s/120s ceilings in
   T5.
5. **Start the battery from a clean validation DB and capture baseline row counts (MR6).** Rotate
   `./data/external-validation/fenixcrm.db` before the run.
6. **Pre-create the validation user once and note the 3/hour register limit (MR3).** Avoid lockout
   during iterative T1.
7. **Drop full `make ci` from validation prerequisites (MR4).** Keep `go test ./...` + `make eval`;
   run `make ci` separately if desired.
8. **Verify fresh-DB workspace provisioning (MR5)** before relying on T1 self-registration; add a
   seed step if registration does not create a workspace.
9. **Bump `deploy/Dockerfile` base image to Go 1.25.x (C3)** as a should-fix follow-up; keep native
   processes for the first battery.
10. **Align `.env.example` and Compose model defaults with the validated local models (V4)** so the
    documented defaults stop pointing at uninstalled models — already in the Plan's follow-ups; keep
    it.

---

*Scope note: host-command outputs in the Audit's "Local Host Baseline" (hardware, disk, installed
tool versions, `ollama list`) were not independently re-executed in this audit; they are trusted as
of the Audit's 2026-06-29 capture. All source-code claims above were verified by direct file reads in
this session.*
