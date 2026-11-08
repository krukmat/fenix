---
doc_type: task
id: DB-FENIX-B2A2-SECURITY-REMEDIATION
title: "B2.A2 S2S credential store security remediation"
status: blocked_qa_pending_security_review
phase: B2.A
week: 2026-W45
tags: [dubbridge, fenix, s2s, security, sqlite]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: [DB-FENIX-B2A4]
files_affected:
  - internal/domain/auth/service_principal_store.go
  - internal/domain/auth/service_principal_store_test.go
  - internal/infra/sqlite/migrations/
  - internal/infra/sqlite/
  - docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.up.sql
  - docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.down.sql
  - docs/audit/dubbridge-fenix-b2a2-independent-security-review-2026-11-06.md
criticality: critical
criticality_basis: "Authentication revocation, concurrent credential replacement, and SQLite schema alter a security boundary; RRI N/A for DB-FENIX-INT cloud work."
created: 2026-11-06
completed: null
---

# B2.A2 S2S credential store security remediation

Plan: [DubBridge to Fenix S2S security foundation](../plans/dubbridge_fenix_s2s_security_plan.md).
Inputs: [B2.A packet](dubbridge-fenix-b2a-s2s-identity.md), [independent security review](../audit/dubbridge-fenix-b2a2-independent-security-review-2026-11-06.md), [proposed ADR](../decisions/ADR-DB-FENIX-S2S-PROPOSED.md).

## Scope

Adapt the existing B2.A2 store and in-branch SQL proposal to current HEAD. Repair H1, H2, M1, and only H3 failures caused by the B2.A2 code. Do not activate an endpoint, issue a real credential, alter GitHub Actions, run a production migration, or merge to main. RRI: N/A — DB-FENIX-INT/cloud. The user's 2026-11-06 instruction explicitly authorizes this bounded auth and local SQLite implementation; independent readiness and code review still apply.

## System context

```text
 [Authenticated operator control, future]          [Isolated S2S guard, inactive]
                 | register/rotate/revoke                       | lookup digest
                 v                                              v
       +---------------------------------------------------------------+
       | Fenix service credential store: principal scope, digest,      |
       | version, revocation, audit linkage; no bearer persistence      |
       +-------------------------------+-------------------------------+
                                       | SQL transaction / query
                                       v
                         [Fenix SQLite, FK enabled, WAL]
                                       |
                                       v
                      [Default-deny capability registry, inactive]

 Invariants: one live credential per principal; rotation commits revoke and
 insert together; failure rolls back both; revocation denies immediately;
 caller owns DB lifecycle and operator authorization; store owns its SQL
 transaction and returns a safe error; no goroutine is retained; workspace
 and source scope come only from Fenix rows; capability allowlist starts empty.
```

## Pseudocode

```text
register(principal, approved_scope): validate; persist constrained identity and empty allowlist
store_initial(digest): validate; insert only when principal active and no live credential
rotate(principal, new_digest, time): validate; acquire write transaction; revoke live credential; insert replacement with next version; write audit linkage; commit; rollback on any error
revoke_credential(id, time): mark active row revoked at max(time, issued_time)
revoke_principal(id, time): mark active principal revoked without creation-time guard
authenticate(token, time): read active digest and Fenix scope; deny revoked, premature, expired or malformed token
test(): use migrated modernc SQLite; verify FK/scope, concurrent rotation, rollback, revocation under clock skew, down migration, redaction
```

## Verification and exit

- Go 1.25.13: focused unit and real SQLite tests, race/concurrency repetition, and rollback checks.
- `scripts/qa-go-prepush.sh` plus focused H3 format/complexity/lint checks.
- Independent post-code security review with a scoped diff. Report PASS/FAIL for every gate and pending blockers.

## Current checkpoint

The B2.A2-specific implementation and tests pass. The repository pre-push gate is blocked by 26 unrelated unformatted files; the separately run complexity and lint gates report B2.A3 middleware, S2S capability registry, and existing human auth middleware findings, while wrapcheck reports other branch files. The formal post-code peer workflow gate returned PASS through local-gemma fallback after the primary Claude Code CLI exited unavailable. Its critical-task local-qwen advisory returned `needs_changes` due the global QA failures and a truncated verification packet. A fresh independent security acceptance and full QA PASS remain pending. No push or task closure is authorized.
