---
doc_type: handoff
id: DB-FENIX-B2C-C1C3-FIXPACK-2026-11-10
title: "B2.C1-C3 verified findings remediation, exact-source local QA"
status: source_candidate_unverified
date: 2026-11-10
branch: feature/int-v2-dubbridge-consumes-fenix
tags: [dubbridge, fenix, b2c, remediation, qa]
---

# B2.C1–C3 — post-review remediation packet

**Owner:** ChatGPT Cloud. **Current task scope:** C1/C2/C3 inert S2S journal only. **RRI:** N/A DB-FENIX-INT/cloud. C4 deferred/non-blocking; B2.A3.T4 security #8 remains a separate rollout gate. No routes, credentials, live migration, agent dispatch, CI activation, deployment or main merge.

## Evidence and fixed source

- Independent local validations on original HEAD `e8c307c7c2a4af1e9f7fc22a673cbb0841b83c68` reported **FAIL / needs_changes**, twice: [issue #9 first review](https://github.com/krukmat/fenix/issues/9#issuecomment-6769891423), [second review](https://github.com/krukmat/fenix/issues/9#issuecomment-6772517784).
- Last code/test remediation commit in this packet: `89552badcca8cc1228a92a864743a95c85d218f6`. Capture and verify the **actual current fetched branch HEAD** before running checks, as documentation commits may follow. Full original C1–C3 code plus remediation must be considered by peer review.

| Finding | Cloud remedy in existing branch | Remaining gate |
|---|---|---|
| F1 HIGH: SQLite NULL-CHECK allows owner without expiry | Migration 039 explicitly requires `lease_until_unix IS NOT NULL` whenever `lease_owner` exists; update negative schema test | Actual embedded migration + Go 1.25.13 |
| N2 MED: terminal `run_id` rewrite possible | Trigger freezes terminal run/lease/updated timestamp, plus completed and indeterminate mutation tests | Real migration + peer review |
| N3 LOW: deleting reservation permits reused ID | Inert durable-journal policy: deny all service_operation DELETE until a separately designed/audited retention protocol; down SQL explicitly drops triggers; tests for accepted and terminal delete | Real up/down migration |
| N1 MED: lone `\\uD800` / `\\uDFFF` digest alias | Strict UTF-8 AND escaped UTF-16 surrogate validation before decoding; valid pairs/literal escaped slash remain accepted; negative/positive tests | Real Go tests and peer review |
| gofmt/vet | Fixed C1 struct formatting and `append` without values; C2 request alignment and redundant blank line | Real `gofmt -l` / `go vet` |
| C2 complexity/lint/wrapcheck | Split canonical JSON input/object/field/container readers; wrap decoder token failures; simplify record hydration/guards; handle rollback failure as an error | Real project lint/complexity/wrapcheck/prepush |
| C3 speculative capability-disable TOCTOU | **Residual / not a new C1–C3 source-only gate:** in-process registry cannot be atomically locked with SQLite claim by current contract; leave default-deny/no dispatch. Reevaluate if enabling a live worker under B2.B4/B2.D, with explicit synchronization design | Future runtime security design, not current inert QA |

**Other source changes preserved:** C2 immutable `trace_id` comparison, Fenix registry-owned config digest, C3 active-principal check inside `MarkIndeterminate` UPDATE, conservative expired-lease reconciliation, tests for revoked principal and disabled capability.

## Minimal local validation — source is NOT certified

Read applicable `AGENTS.md`, [canonical C1–C3 task](../tasks/task_dubbridge_fenix_b2c_operation.md), [integration plan](../plans/dubbridge-consumes-fenix-integration.md), and the [original detailed handoff](dubbridge-fenix-b2c-readiness-validation-2026-11-10.md). Obey the current [independent-review workflow](../playbooks/AGENT_WORKFLOW_GUIDE.md); model selection belongs there, not to the integration plan.

Preserve the **pre-existing uncommitted** `/Users/matias/fenix/.gitignore` entry `scripts/__pycache__/*`. Do not stage/stash/reset/discard it; inspect the checkout before any fast-forward and use a temporary detached worktree if necessary. No implementation or CI/route changes by the local validator.

```bash
cd /Users/matias/fenix
git status --short
git fetch origin feature/int-v2-dubbridge-consumes-fenix
git rev-parse origin/feature/int-v2-dubbridge-consumes-fenix
# Inspect local divergence and preserve .gitignore; do not blindly pull on a dirty worktree.
GOTOOLCHAIN=go1.25.13 go version
gofmt -l internal/infra/sqlite/service_operation_migration_test.go internal/domain/integration/service_operation_*.go
GOTOOLCHAIN=go1.25.13 go test ./internal/infra/sqlite ./internal/domain/integration -run '^TestB2C' -count=1 -v
GOTOOLCHAIN=go1.25.13 go test -race -count=3 ./internal/infra/sqlite ./internal/domain/integration
GOTOOLCHAIN=go1.25.13 go vet ./internal/infra/sqlite ./internal/domain/integration
GOTOOLCHAIN=go1.25.13 go test -count=1 ./...
GOTOOLCHAIN=go1.25.13 bash scripts/qa-go-prepush.sh
git diff --check
```

Run **C1 independent task-readiness again** on the revised task and real B2.A2 scoped evidence #3; separately C2/C3 readiness after C1 PASS. Distinguish planning/readiness evidence from **later** QA/post-code-approval evidence. Avoid task/plan truncation in the reviewer packet; capture full factual inputs, criticality concurrence and actual reviewer artifact hashes. Run independent post-code review for C1–C3 on a frozen full relevant source diff, preserving reviewer independence.

**Important:** Prior Cloud Python sqlite3 smoke (6/6 and 14/14) missed the F1 NULL case. Additional isolated Go-standard-library surrogate scanner cases (10/10) and recreated SQLite predicate/trigger cases (10/10) were exercised during remediation, **not the real Fenix Go1.25.13/modernc integration**. No new PASS for any B2.C gate is claimed.

Report on [#9](https://github.com/krukmat/fenix/issues/9) each command/RC/log/hash, exact source SHA, task-specific readiness, tests/race/vet/prepush, independent code-review verdict and remediable findings. C1–C3 remain **SOURCE_CANDIDATE**, until independent validation is genuinely PASS. External T4 review and unrelated global documentation debt remain separate.
