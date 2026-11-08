---
doc_type: plan
id: DB-FENIX-S2S-SECURITY
title: "DubBridge to Fenix S2S security foundation"
status: active
date: 2026-11-06
tags: [dubbridge, fenix, s2s, security]
---

# DubBridge to Fenix S2S security foundation

## Current direction

B2.A2 repairs the existing Fenix credential store on the feature branch before a new independent security review. The SQLite schema is for local and future deployment review only; no production migration, live credential, route activation, GitHub Actions change, or merge to main is authorized. RRI is N/A under the DB-FENIX-INT cloud-agent exception.

B2.A3.T1 prepares the isolated middleware authorization contract and independent readiness review. The current middleware authenticates a service bearer but leaves organization/project authorization to a separate optional helper. Future implementation must place mandatory source authorization before every S2S handler and operation lookup, preserve default-deny, and obtain task-scoped auth HITL before code changes. This planning step does not activate a route.

## Executable task

- [B2.A2 security remediation](../tasks/task_dubbridge_fenix_b2a2_security_remediation.md): fix H1 rotation, H2 revocation, M1 runtime SQLite integration and tests, and H3 issues attributable to B2.A2. Run Go 1.25.13 and repository QA, then request independent post-code review.
- [B2.A3.T1 middleware readiness](../tasks/task_dubbridge_fenix_b2a3_middleware_readiness.md): record the mandatory organization/project guard, negative authorization matrix, dependencies, and independent readiness verdict before any B2.A3 implementation.

## Acceptance

1. A revoked principal or credential is denied even when the revocation clock precedes creation or issue time.
2. Concurrent rotation leaves exactly one live credential per principal; failures leave the prior credential usable, and a successful rotation makes it unusable.
3. Embedded Fenix migrations create the constrained S2S schema on a fresh local database and support an isolated down/rollback check. Version, audit linkage, and empty default capability allowlist are represented without enabling a capability.
4. Tests use Fenix `NewDB` with modernc SQLite, exercise foreign keys, tenant scope, concurrent rotation, revocation, rollback, and secret redaction.
5. B2.A2 code passes its relevant format, complexity, lint, and tests. The full `scripts/qa-go-prepush.sh` result and unrelated baseline failures are reported precisely.
6. Independent readiness and code review gates are recorded. Deployment and activation remain separate decisions.
