---
doc_type: audit
id: DB-FENIX-B2C-CLOUD-CHECKPOINT-2026-11-10
title: "B2.C SQLite operations — Cloud source checkpoint and pending independent gates"
date: 2026-11-10
status: cloud_sqlite_synthetic_pass_go_review_pending
tags: [fenix, dubbridge, sqlite, idempotency, security, recovery]
---

# B2.C — Source checkpoint (Cloud, not independent security certification)

**Plan**: [Canonical integration](../plans/dubbridge-consumes-fenix-integration.md#b2c-feature-branch-checkpoint--2026-11-10).
**Issues**: [B2.C #9](https://github.com/krukmat/fenix/issues/9), [B2.A3.T4 #8](https://github.com/krukmat/fenix/issues/8).
**Branch**: `feature/int-v2-dubbridge-consumes-fenix`.
**Authority**: User approved C1–C4 tasks for this feature branch on 2026-11-10. RRI N/A DB-FENIX-INT cloud. Independent task readiness for C1–C3 **not run**, no waiver. Production authorization absent.

## Implementation inventory

| Task | Source | Behavior / authority boundary | Verification |
|---|---|---|---|
| C1 | `migrations/039_service_operation.{up,down}.sql`, `service_operation_migration_test.go` | (principal_id, operation_id) PK; 32-byte digest/config; immutable tenant scope from active stored Fenix principal; terminal/indeterminate cannot reset; down local-only | **Synthetic SQLite PASS** for valid insert, scope mismatch/inactive, immutable identity, terminal rewind and local rollback. **Real Go MigrateUp test NOT RUN** |
| C2 | `internal/domain/integration/service_operation_store.go`, `service_operation_digest.go`, `service_operation_store_test.go` | `BEGIN IMMEDIATE` serialized INSERT SELECT with authoritative Fenix scope; exact replay vs conflict; unknown/disabled capability denies; config SHA-256 derived from approved Fenix registry binding, not incoming JSON; canonical JSON bounded 64KiB/32 depth, rejects duplicate nested keys | Source and tests published; real Go test/race/vet/lint **NOT RUN** |
| C3 | `service_operation_recovery.go`, `service_operation_recovery_test.go` | One-shot lease only on accepted, reconfiguration-fingerprint guard, explicit `indeterminate` after expiry/ambiguous failure, active scoped read, no agent run/dispatch | Source and tests published; real Go test/race/vet/lint **NOT RUN** |
| C4 | `docs/tasks/task_dubbridge_fenix_b2c4_status_resolver.md` | Contract to implement real persisted `DubBridgeOperationResolver` and allowlisted sanitized status projection, without mounting router | **NOT IMPLEMENTED**. Requires C1–C3 QA and independent B2.A3.T4 review |

## Cloud SQL-only verification

An isolated **Python sqlite3** in-memory fixture was used to simulate the Fenix workspace and service_principal columns and execute candidate migration 039. Negative probes rejected forged workspace/org/project, revoked principal, invalid operation ID/digest, mutation of immutable identity, and transition `indeterminate -> accepted`; accepted operation could be reconciled `accepted -> indeterminate`; local down removed operation tables/triggers. Additional parameterized `INSERT SELECT ... ON CONFLICT DO NOTHING` query yielded first insert=1, identical replay insert=0, mismatched project insert=0. **These are not the real Fenix Go modernc tests.**

## Known risks / next gates

| ID | Severity / scope | Finding | Disposition |
|---|---|---|---|
| B2C-G1 | Critical / branch validation | No Go 1.25.13 compiler, real Fenix migrations, race, global prepush or independent review executed on B2.C source | **BLOCKS C1–C3 task closure.** Local agent runs QA/reviewer, Cloud corrects source findings and synchronizes status |
| B2C-G2 | Critical / gateway security | B2.A3.T4 independent security verdict remains pending. HTTP status adapter and production route do not exist | **BLOCKS C4 activation.** Follow #8; no fake B2.C GET persistence claims |
| B2C-R1 | Medium / B2.B4 + B6 | Current config fingerprint covers resolved registry principal/workspace/capability/schema/handler. It is not a full immutable approved schema/policy/budget fingerprint; registry is in-memory and can change between validation and reservation | Before live dispatch, durable versioned approved capability policy and concurrency fencing are required |
| B2C-R2 | Medium / B2.D + B6 | Canonical JSON sorts map keys and removes spacing; numeric lexical variants (e.g., 1 vs 1.0) intentionally conflict. No strict capability-specific input schema or full protocol error/status mapping exists | Later transport/spec and semantic canonicalization approval required; no HTTP 409 reported yet |
| B2C-R3 | High / runtime | No actual provider is invoked; lease claim does not create a real run_id. Reconciliation is conservative and makes ambiguous operations indeterminate, not exactly-once side effect execution | Future B6 governed runtime must provide actual run linkage, idempotent tool effects and recovery evidence |
| B2C-R4 | High / rollout | Embedded migration 039 creates schema on next local `MigrateUp`; its down is destructive. No production data backup/restore/activation evidence | **No production migration/rollback.** Separate owner approval for deployment and destructive operations |
| B2C-R5 | Evidence | C1–C3 independent task-readiness not executed; user approval of plan is not a literal reviewer waiver; C4 review is blocked | Record unexecuted gates, obtain genuine independent results before formal closure |

## Local execution contract

Read `README_AGENT_ORDER.md`, `CLAUDE.md`, `AGENTS.md`, and relevant task records. Work on existing feature branch, preserve unrelated local modifications, no CI, route, credential, deployment, production migration or main merge.

```bash
cd /Users/matias/fenix
git switch feature/int-v2-dubbridge-consumes-fenix
git pull --ff-only
gofmt -l internal/infra/sqlite/service_operation_migration_test.go internal/domain/integration/service_operation_*.go
GOTOOLCHAIN=go1.25.13 go test ./internal/infra/sqlite ./internal/domain/integration -run '^TestB2C' -count=1 -v
GOTOOLCHAIN=go1.25.13 go test -race -count=3 ./internal/infra/sqlite ./internal/domain/integration
GOTOOLCHAIN=go1.25.13 go vet ./internal/infra/sqlite ./internal/domain/integration
GOTOOLCHAIN=go1.25.13 go test ./...
GOTOOLCHAIN=go1.25.13 bash scripts/qa-go-prepush.sh
```

Run actual independent task-readiness and post-code-review as required by repository governance for C1–C3. Do not reuse B2.A3's readiness deferral or code-review artifact as B2.C evidence; do not self-review. If code/QA fails, publish exact diagnostics and logs to #9 for ChatGPT Cloud remediation. Changed-code-only format fixes should be scoped to B2.C files; do not opportunistically expand to the whole repository.

**Dispositions:** C1–C3 = SOURCE CANDIDATES, NOT QA PASS; C4 = BLOCKED. Gate/rollout = NO-GO.
