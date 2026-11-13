---
doc_type: handoff
id: DB-FENIX-B4.W4
title: "B4.W4 nonproduction-only S2S integration activation proposal"
status: proposal_not_approved_for_activation
phase: B4
date: 2026-11-13
owner: ChatGPT
---

# B4.W4 — Explicit activation gate (PROPOSAL, not approved)

**Owner's 2026-11-13 GO CONDICIONAL applies to SOURCE DEVELOPMENT only.** This document does not authorize live ports, credential issuance, policy changes, database migrations, operator permissions, new hosting/paid costs, deployment or merge. No operator/runtime actions have been performed. The system remains default deny.

## Dependency/technical readiness before requesting the *separate* owner decision

1. B4.W1: Fenix dedicated **build-tagged** isolated router (`s2s_nonproduction`), not mounted by regular `NewRouter`/`NewServer`. Configuration `Environment="integration-test"` and `Enabled=true` is only a candidate for a separately started nonproduction router after approval.
2. B4.W2: `integration.diagnostics` schema and deterministic, build-tagged pure handler exist as source. The Fenix admission layer currently **only reserves** a validated operation; there is **no authorized live diagnostic dispatch hook** and no operator-enabled capability. That mapping must be deliberately reviewed in a nonproduction harness before any accepted-operation-with-handler result is claimed.
3. B4.W3.2: candidate `docs/proposals/0040_fenix_dispatch_intents.up.sql` in DubBridge is **not** an executable migration or applied DB. The source-only repo contains one immutable intent and a single send claim.
4. B4.W3.3: dormant Rust `FenixDispatchAdapter` wires trusted `PgFenixScopeAuthorizer`, persisted request digest, stable UUIDv5 IDs and one-shot `FenixClient` with manual-on-ambiguity handling; it is **not mounted in DubBridge's API router or background workers**.
5. B4.W5: unresolved intent audit/reporting and explicit stale-claim classification exist only as code. [Manual reconciliation protocol](https://github.com/krukmat/dubbridge/blob/feature/int-v2-dubbridge-consumes-fenix/docs/handoffs/dubbridge-fenix-b4w5-manual-reconciliation-2026-11-13.md). C4 GET/status intentionally deferred.

## Proposed W4.1 — reversible scope

- One specifically named, isolated and disposable **nonproduction** environment. Production router remains untouched.
- Fenix-issued opaque **service principal**, never a DubBridge human JWT. Bind consumer `dubbridge`, exact workspace, org and optional project ID, status active, revocation capability. Keep bearer exclusively in a supported secret manager with masked telemetry; never store it in source, CI, database intent, stdout or logs.
- Separate authorized Fenix **operator identity** for provisioning a **disabled** `integration.diagnostics` v1 policy/schema/handler binding, with authenticated operator RBAC; approval then required to enable it **only in the isolated environment**. Do not assume that registering a handler ID makes a runtime handler invocable.
- Controlled provider route lifecycle: distinct nonproduction listener/front door with network restrictions, rate limits and request bounds. No addition to normal `NewRouter` or human JWT routes.
- DubBridge Postgres candidate migration: only a separately approved nonproduction application with a backup/rollback plan; immutable in-flight rows retained as evidence. No direct production migrations. One request trace binds actual authorized DubBridge organization/project to the Fenix service identity.
- Client config stays opt-in, HTTPS-only, redacted bearer, redirects disabled, bounded payloads and request duration. No automatic second POST or C4 status GET.

## W4.2 — proposed enable/stop sequence (do not execute without new consent)

| Phase | Action | Stop condition |
|---|---|---|
| 0. Scope | Owner specifies the exact nonproduction host(s), allowed user/org/project, duration, operator and whether there is any infrastructure cost. | Missing approval or unclear scope. |
| 1. Storage | Approve/test candidate migration in isolated Postgres and Fenix SQLite already versioned for principal/operation policy. | Schema mismatch, missing backup or unwanted production mutation. |
| 2. Identity | Issue/revoke-able, short-lived service principal and bind exact Fenix workspace and mapped DubBridge tenant. | No isolated secret manager or credential boundary. |
| 3. Diagnostic | Operator provisions disabled `integration.diagnostics` v1; only afterward explicitly enables **test** binding and verifies trusted handler ID resolution, schema and zero workflow/LLM execution. | Handler absent, capability enabled too broadly, unverified operator grant. |
| 4. Listener | Start isolated tagged S2S router with explicit `integration-test` opt-in, restricted exposure; verify normal application router does not mount it. | Unintended normal/public endpoint exposure. |
| 5. Client | Opt-in DubBridge adapter, create one durable intent *before* send, authorize project/org again, one HTTPS POST. | Changed digest, unprovisioned client, repeated claim, invalid response or unknown outcome. |
| 6. Stop | Turn off DubBridge opt-in and stop isolated listener; disable diagnostic policy; revoke service credential; preserve immutable operation ledger/evidence. | Incomplete teardown or unclear ambiguous operation. |

**B4.T1–T3** only after this W4 owner gate: live HTTPS authentic admission, revocation/tenant/capability denials and durable idempotency/outage observations, *not* an agent workflow proof.

## Exact owner decision required

Before a runtime step, request a new explicit approval naming **environment, tenant scope, duration, provisioning boundary and acceptable costs**. A generic "GO CONDICIONAL" or approval for W2/W3.2/W3.3/W5 does not authorize this. No additional user confirmation is required to continue inert source/dependency work.

**Current:** `W4_SOURCE_PROPOSAL_READY`, `W4_ACTIVATION_NOT_AUTHORIZED`. No E2E/production GO and no security certification claim.
