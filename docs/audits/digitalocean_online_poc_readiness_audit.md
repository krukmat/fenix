---
doc_type: audit
id: DO-POC-READINESS-AUDIT-2026-08
title: "DigitalOcean online POC readiness audit"
status: conditional
date: 2026-08-24
scope: "Repository state, local verification, deployment assets, historical validation evidence, and current official DigitalOcean documentation."
related_plan: "[[plans/digitalocean_online_poc_readiness_plan]]"
related_task: DO-POC-READINESS-001
tags: [digitalocean, poc, deployment, audit, readiness, support-wedge]
---

# DigitalOcean Online POC Readiness Audit

## Executive Verdict

**Conditional: feasible for a limited external POC in October 2026, but not deploy-ready on 2026-08-24.** The product and delivery foundations are real: the governed support flow, evidence, approvals, audit trail, usage records, Go API, BFF, Caddy, container images, split LLM configuration, and readiness endpoint exist. The remaining work is principally a production-contract and live-validation gap, not a greenfield build.

September is suitable for closing the deployment contract and running a staging proof on DigitalOcean. An externally accessible, invited-user POC can start in October only after the P0 gates below pass. This assessment does not authorize provisioning, credentials, DNS, billing, deployment, or source-code changes.

## Scope and Evidence Date

Repository and local checks were performed on 2026-08-24. DigitalOcean facts were checked against official documentation current in July-August 2026. The legacy plan is a March 2026 snapshot and is retained only as historical detail.

## Current Capability Assessment

| Area | State | Evidence | POC interpretation |
|---|---|---|---|
| Product wedge | Implemented | Strategic spec and implementation plan identify Support Copilot/Agent as the primary wedge; Sales Copilot is secondary. | Keep the first online demonstration narrow and support-led. |
| Governed support runtime | Implemented, deterministic | `internal/domain/agent/agents/support.go` builds evidence packs, determines action by evidence score, routes tools, creates approvals, hands off, and records audit/usage data. | It is a governed workflow, not an LLM-authored support response. Demo language must say so. |
| Copilot and sales LLM path | Implemented, locally tested | `internal/domain/copilot` calls `LLMProvider`; July fixes made sales brief degrade rather than fail on action parsing and record usage attribution. | The correct candidate for DigitalOcean Serverless Inference validation. |
| External-chat adapter | Implemented, unverified against DO account | `openai-compat` calls `/v1/chat/completions` and `/v1/models`; config separates `CHAT_PROVIDER` and `EMBED_PROVIDER`. | A real inference key, model selection, and smoke request remain required. |
| Embeddings | Implemented, hybrid deployment unresolved | Only Ollama is accepted by `NewEmbedProvider`. Production compose defines an optional `ollama` service. | POC must run a local embedding-only Ollama service or explicitly implement another embedding provider. |
| SQLite | Implemented for one node | WAL, foreign keys, 5-second busy timeout, synchronous NORMAL, and a bounded pool are configured in `internal/infra/sqlite/db.go`. | Valid only for one app host and one attached filesystem; no horizontal scale/HA claim. |
| Backend/BFF quality | Fresh focused checks pass | Go config/LLM/API/handler/SQLite tests pass; BFF build and 418/418 coverage tests pass. | Good regression evidence, but not a cloud smoke test. |
| Deterministic governance eval | Fresh check passes | `make eval` passed with policy compliance threshold 1.0. | Useful safety regression evidence, not live-provider evidence. |
| BDD deployment gate | Not usable as evidence | `go test ./tests/bdd/... -v` passed but reported `No scenarios`. | Replace this historical gate with explicit POC smoke scenarios. |
| Production compose | Partially ready | `docker-compose.prod.yml` resolves syntactically with external-chat variables. | It is not yet the required DigitalOcean persistence, strict-readiness, or secret contract. |
| TLS/proxy | Present, unproven online | Caddy config routes `/bff/*` to BFF and the remainder to Go. | DNS ownership and real certificate issuance are not validated. |
| Operations | Documented only | No deploy, restore, or monitoring automation was found under `deploy/` or `scripts/`. | Backup/restore and alert proof are P0 before inviting users. |

## Reproducible Verification Performed

| Command | Result | Limit |
|---|---|---|
| `go test ./internal/infra/config ./internal/infra/llm ./internal/api/handlers ./internal/api` | PASS | Unit/integration only; no live provider. |
| `go test ./internal/infra/sqlite` | PASS | Confirms the local SQLite contract, not a mounted DO Volume. |
| `make eval` | PASS | Deterministic fixtures only. |
| `go test ./tests/bdd/... -v` | PASS, but `No scenarios` | Must not be cited as functional POC coverage. |
| `cd bff && npm run build && npm run test:coverage -- --runInBand` | PASS, 30 suites / 418 tests | BFF test process reports open handles after completion; no failure, but investigate before production hardening. |
| `docker-compose -f docker-compose.prod.yml config --quiet` with placeholder external-chat config | PASS | Syntax/interpolation only; no image build or runtime started. |

## P0 Gates: Must Close Before External Access

| ID | Finding and evidence | Required exit evidence | Owner role |
|---|---|---|---|
| P0-1 | **Contract updated, staging proof pending.** Compose now bind-mounts the operator-provided `FENIX_DATA_DIR` instead of `db_data`. | A restart proves database and attachment persistence on the mounted Volume. | Backend/platform |
| P0-2 | **Contract updated, live-model proof pending.** `ollama` is part of the normal compose graph and its persistent directory is explicit. | One documented production command starts the embedding service and `/readyz` reports database, chat, and embed all `ok`. | Backend/platform |
| P0-3 | **Contract updated, runtime proof pending.** The endpoint retains degraded diagnostics, but the backend healthcheck accepts only `"status":"ready"`. | A strict startup/readiness probe fails unless all three dependencies are ready, while the existing non-strict operational endpoint semantics remain deliberate and tested. | Backend/platform |
| P0-4 | **Contract updated, secret-delivery proof pending.** Compose requires secrets and Go/BFF CORS origins; BFF production startup rejects missing session/CORS values. | Secrets are required at startup, live only in the approved secret delivery mechanism, and both Go/BFF origins are explicit production values. | Security/platform |
| P0-5 | No current restore procedure or test proves recovery of SQLite data or attachments. | Snapshot/backup policy, a time-stamped restore drill on a disposable target, and documented RPO/RTO for the POC. | Platform/operator |
| P0-6 | The OpenAI-compatible adapter has never been proven with an actual DigitalOcean inference key/model in this audit. | Serverless model listing, chat completion, error-path, latency, usage, and cost-cap evidence from the chosen region/account. | Product/platform |
| P0-7 | No external host, DNS, TLS, firewall, or end-to-end smoke evidence exists. | HTTPS-only host; Cloud Firewall permits 80/443 and restricted SSH only; direct ports 3000/8080 are unreachable; all POC smoke cases pass. | Platform/operator |

## P1 Hardening Before or Immediately After Pilot Start

| ID | Finding | Mitigation |
|---|---|---|
| P1-1 | `/metrics` and `/bff/metrics` are public through the catch-all Caddy route. | Restrict to a monitoring source/private path or consciously document a non-sensitive public metrics policy. |
| P1-2 | Logs are mostly plain `log.Printf`/middleware output, not a coherent structured production event stream. | Add a minimal request/error/startup log contract with request ID, status, latency, provider, and no secret/prompt leakage. |
| P1-3 | Images use floating tags (`ollama:latest`, broad Caddy tag). | Pin tested image versions/digests and record an update procedure. |
| P1-4 | BFF test runner force-exits with open handles. | Diagnose before promoting its test setup as a long-running reliability signal. |
| P1-5 | 4 GiB is plausible for Go+BFF+Caddy+embedding-only Ollama but not measured. | Run a staged load/memory observation; pre-authorize a move to 8 GiB if sustained memory pressure or latency requires it. |

## Minimum Credible Online POC

The POC is an **invited, single-region, single-host demonstration**, not a general availability launch.

In scope:

- One or a few controlled workspaces and named pilot users.
- Login, account/contact/case creation, knowledge ingestion and retrieval.
- Support run with evidence, auditable outcome, safe tool routing, approval or human handoff where policy requires it.
- Copilot chat via the selected DigitalOcean serverless model, with citations and recorded usage.
- HTTPS, backup/restore proof, uptime checks, and operator runbook.

Out of scope:

- Multi-node SQLite, high availability, public self-service scale, formal SLA, multi-region recovery, broad mobile parity, or an LLM-authored Support Agent.
- Replacing Ollama embeddings without a separately approved provider task.

## Recommended DigitalOcean Shape and Cost Basis

```text
Pilot users
    |
 HTTPS :443
    v
DigitalOcean Cloud Firewall --> Droplet (single region, 4 GiB / 2 vCPU starting point)
                                     |
                                     +-- Caddy --> BFF --> Go API
                                     |                       |
                                     |                       +-- SQLite on mounted DO Volume
                                     |                       +-- Ollama, embeddings only
                                     |
                                     +-- Serverless Inference API (chat only)
```

- Start with a Basic 4 GiB / 2 vCPU Droplet at **$24/month** and a 50 GiB Volume at **$5/month**: **$29/month fixed infrastructure before backups, snapshots, DNS, and inference**. Move to 8 GiB / 4 vCPU ($48/month) only if the measured pilot workload warrants it.
- Serverless Inference is prepaid and usage-priced. For example, the official catalog currently lists DO-hosted Qwen3-32B at $0.25/M input and $0.55/M output tokens. Model availability and rates must be rechecked at purchase.
- A DO Volume is a raw, network-attached block device, tied to the Droplet's region and requiring a mount. It is appropriate for this single-host SQLite POC, not a shared multi-writer database.
- Use a Cloud Firewall with explicit allow rules. It is stateful and has no additional charge; the POC should expose only 80/443 and a restricted SSH source range.

Official sources: [Droplet pricing](https://www.digitalocean.com/pricing/droplets), [Volume pricing](https://docs.digitalocean.com/products/volumes/details/pricing/), [Volumes on Droplets](https://docs.digitalocean.com/products/volumes/how-to/create/), [Serverless Inference API](https://docs.digitalocean.com/products/inference/reference/api/serverless-inference/), [Inference pricing](https://docs.digitalocean.com/products/gradient-platform/details/pricing/), and [Cloud Firewall rules](https://docs.digitalocean.com/products/networking/firewalls/how-to/configure-rules/).

## Decisions Needed From the Project Owner

1. Approve the POC promise: deterministic governed Support Agent plus LLM Copilot, rather than representing the Support Agent itself as generative.
2. Provide/approve the DigitalOcean team, billing ceiling, region, domain, named operator, and a restricted SSH source range.
3. Choose the serverless model only after a cost/quality/latency smoke test; do not hard-code a March model name or price.
4. Accept a single-host SQLite POC and its stated recovery target, or fund a separate database architecture decision.
