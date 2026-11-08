---
doc_type: audit
id: DB-FENIX-B2A2-REMEDIATION-2026-11-06
title: "B2.A2 security remediation and independent review packet"
status: pending_independent_review
date: 2026-11-06
branch: feature/int-v2-dubbridge-consumes-fenix
base_commit: 6447d1b
tags: [dubbridge, s2s, security, sqlite, qa]
related:
  - ../tasks/task_dubbridge_fenix_b2a2_security_remediation.md
  - dubbridge-fenix-b2a2-independent-security-review-2026-11-06.md
---

# B2.A2 security remediation packet

This packet covers the B2.A2 diff against `6447d1b` on `feature/int-v2-dubbridge-consumes-fenix`. No separate attachment was present in the workspace; the in-branch B2.A2 store and SQL proposal were adapted. RRI: N/A — DB-FENIX-INT/cloud. The prior independent [FAIL review](dubbridge-fenix-b2a2-independent-security-review-2026-11-06.md) remains historical evidence, not an approval.

## Findings addressed

| Finding | Remediation | Local evidence |
|---|---|---|
| H1 rotation | `RotateServiceCredential` holds `BEGIN IMMEDIATE`; revoke, versioned insert and audit occur before one commit. A partial unique index permits one unrevoked credential per principal. | Five concurrent rotations leave one active credential; duplicate-ID insertion rolls back the prior revocation and audit. |
| H2 clock skew | Credential revoke stores `max(operator_time, issued_time)` without a timestamp eligibility guard. Principal revoke has no creation-time guard. | Both revocations deny authentication when the supplied time is five seconds earlier. |
| M1 runtime SQLite | Embedded migration 038 adds principal, credential and append-only audit tables; credential version and empty, constrained capability allowlist are persisted. The down SQL is exercised locally. | Go tests use Fenix `NewDB` with modernc, `MigrateUp`, FK constraints, tenant scope, expiry, redaction and rollback. |
| H3 B2.A2 code | Store functions were split below the repository complexity threshold; B2.A2 lint issues were fixed. The public token prefix has a justified G101 exclusion. | Focused `gocyclo -over 7` and `golangci-lint run ./internal/domain/auth/... ./internal/infra/sqlite/...` PASS. |

## Verification

- PASS: `GOTOOLCHAIN=go1.25.13 go test ./...`.
- PASS: `GOTOOLCHAIN=go1.25.13 go test -race -count=5 ./internal/domain/auth ./internal/infra/sqlite` before final audit-trigger and test updates; final focused race run `-count=2 -run TestServiceCredential` PASS.
- PASS: `python3 scripts/int_b2a2_sqlite_proposal_selftest.py` and `python3 scripts/int_b2a2_store_sqlite_smoke.py`.
- PASS: B2.A2-focused `gocyclo -over 7` and `golangci-lint run`.
- FAIL: `GOTOOLCHAIN=go1.25.13 bash scripts/qa-go-prepush.sh` stops at `fmt-check` on 26 unrelated Go files.
- FAIL: `make complexity` reports B2.A3 `DubBridgeServiceAuth` (20) and `AuthorizeDubBridgeSource` (8), plus the separate S2S capability registry (12 and 8); no B2.A2 store function is reported.
- FAIL: `make lint` reports B2.A3 middleware and existing human auth middleware findings; no B2.A2 auth or SQLite findings.
- FAIL: `make wrapcheck-gate` reports unrelated branch files under evidence/tool/integration; no B2.A2 file.
- 2026-11-08 recheck: PASS `GOTOOLCHAIN=go1.25.13 go test ./...`, focused `go test -race -count=1` for service credentials/identity, both B2.A2 Python checks, and `git diff --check HEAD`. The mandatory pre-push script again FAILS at `fmt-check` on the same 26 files. Separate `make complexity`, `make lint`, and `make wrapcheck-gate` runs confirm the listed failures remain outside this B2.A2 diff.

## Pending gates and scope

The formal post-code peer workflow gate returned PASS through a local-gemma fallback after the primary Claude Code CLI exited unavailable. Proof: `logs/peer-workflow-review/post-code-review_codex_by_local-gemma_20261106T132546Z.json`. The critical-task local-qwen advisory returned `needs_changes`, citing global QA failures and the reviewer's truncated verification log. This fallback PASS is a process gate result, not fresh independent security acceptance equivalent to the prior Claude Opus review.

At the 2026-11-06 checkpoint, the repository QA gate was FAIL and no push or task closure was permitted. The A3 middleware and unrelated formatting/wrapcheck debt needed separate scoped remediation. No live endpoint, production migration, real credential, GitHub Actions change, or merge to main was performed. Local schema migration 038 existed only in this feature-branch diff; applying it to production still requires explicit operator approval and successful security review.

## Branch pipeline pause, 2026-11-08

The operator explicitly requested that GitHub Actions and the local pre-push pipeline be paused only on `feature/int-v2-dubbridge-consumes-fenix` to reduce integration cost, with normal execution on `main` after merge. The branch-only guards are in `.github/workflows/ci.yml` and `scripts/hooks/pre-push`. The hook remains active on every other branch. With temporary QA-cleanup changes applied, the last local pre-push attempt passed format, complexity, lint, wrapcheck, race-enabled tests, coverage (83.0%), TDD coverage (84.7%), and deadcode; it was interrupted during race-stability at the operator's direction. Earlier B2.A2-specific tests passed. The broad QA-cleanup working tree was preserved in a local stash and is not part of the review diff. Independent security acceptance, production migration approval, and merge approval remain pending.
