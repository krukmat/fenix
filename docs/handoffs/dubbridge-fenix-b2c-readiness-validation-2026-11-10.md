---
doc_type: handoff
id: DB-FENIX-B2C-READINESS-VALIDATION-2026-11-10
title: "B2.C C1 readiness remediation, Go verification and separate T4 exact-diff security handoff"
status: ready_for_independent_local_validation_no_pass_claim
date: 2026-11-10
branch: feature/int-v2-dubbridge-consumes-fenix
tags: [dubbridge, fenix, b2c, readiness, go, security, handoff]
related:
  - ../tasks/task_dubbridge_fenix_b2c1_sqlite_schema.md
  - ../tasks/task_dubbridge_fenix_b2c2_operation_store.md
  - ../tasks/task_dubbridge_fenix_b2c3_recovery.md
  - ../tasks/task_dubbridge_fenix_b2c_operation.md
  - ../plans/dubbridge-consumes-fenix-integration.md
---

# DB-FENIX-B2.C — local validation packet after Cloud documentary remediation

**Owner request**: repair documentation first, then have the local agent run real validation on the published feature branch. **RRI: N/A — DB-FENIX-INT/cloud**. No live credentials, route, CI reactivation, running workflow, schema deployment, production rollback or main merge.

## Fixed references and evidence boundaries

| Item | SHA / record | What it proves | What it does NOT prove |
|---|---|---|---|
| B2.A2 base owner decision | [issue #3](https://github.com/krukmat/fenix/issues/3), 2026-11-08, `87fd10d3e10658179d6f4ab626d33e2cf555539c` | **Owner-accepted scoped** H1 atomic rotation/rollback, H2 skew revocation, M1 embedded migration 038; local Go 1.25.13 modernc race x3 PASS reported | Independent B2.A2 security PASS, global QA PASS, C1 tests PASS or production approval |
| Original independent B2.A2 review | [2026-11-06 security review](../audit/dubbridge-fenix-b2a2-independent-security-review-2026-11-06.md) on `376209e` | Historical FAIL findings H1–H3/M1 before remediation | A post-remediation independent PASS |
| B2.A3 T3 | [issue #7](https://github.com/krukmat/fenix/issues/7), `acafa3d` | Previously reported branch-scope Go QA and code review | C1–C3 validation or B2.A3.T4 independent security signoff |
| B2.C implementation candidate | `b17162743ed8ee4f3e99be09e910e3b3d1c01fb6..dc7354738dfc9633230f9b31f31b5bf625c73b3d` | Feature-branch C1–C3 Go/sqlite code and tests, plus original docs | Any C1–C3 Go compile/test/race/reviewer PASS |
| B2.C1 previous readiness | owner-reported `needs_changes` at `6772905649d8390b4943273f4e52416696194087` | Two documentary findings: B2.A2 evidence missing; FR/UC empty without rationale | A new readiness verdict; local artifact path/reviewer is **not supplied to Cloud** |
| B2.A3.T4 exact security scope | `43d935c4288a565b6c4d202208ebef16cc35e012..acafa3d3b59f0539b7fbc5ec3114a43c332bce80` | Frozen code+docs QA-refactor diff to independently review | T4 reviewer verdict; any diff from current HEAD adds unrelated B2.C work |

**Traceability reconciliation:** Each C1–C4 task now explicitly explains empty `fr_refs: []` and `uc_refs: []`: S2S infrastructure without a CRM business use case. The normative references are the [integration plan](../plans/dubbridge-consumes-fenix-integration.md), [neutral S2S envelope](../contracts/dubbridge-fenix-v1-envelope-request.schema.json), B2.A2 persisted identity and individual C1–C4 acceptance criteria. No FR/UC invented. B2.A2 technical source prerequisite is **available for branch QA**, while independent security/production closure remains **pending**; keep the C1 dependency visible.

## Execution governance (not a task dependency)

Read the active [agent workflow guide](../playbooks/AGENT_WORKFLOW_GUIDE.md) when invoking independent readiness or post-code review. Its provider/model routing is an **implementation detail of the review gate**, not an acceptance criterion or dependency in the DubBridge–Fenix plan. Preserve reviewer independence, actual artifacts, explicit PASS/FAIL/NOT_RUN and scope/criticality rules.

The published peer-review workflow's 60 unit tests do **not** certify C1–C3. Repository-wide documentation QA debt remains unresolved and must not be reported as PASS. Preserve the owner's uncommitted `/Users/matias/fenix/.gitignore` entry `scripts/__pycache__/*` verbatim. RRI: N/A — DB-FENIX-INT/cloud.

## Latest C1–C3 candidate changes to include in local QA

The prior frozen C1–C3 candidate was amended on this feature branch. **Do not certify against the older source SHA.** Capture the actual new HEAD and review this exact six-file source/test delta in addition to the original C1–C3 implementation; reviewer packets must include the full relevant C1–C3 source state, not just the added tests:

- `internal/infra/sqlite/migrations/039_service_operation.up.sql`: `accepted` no run ID, nonempty run ID in required states.
- `internal/infra/sqlite/service_operation_migration_test.go`: rejects fake accepted run IDs and empty running run IDs.
- `internal/domain/integration/service_operation_store.go`: replay requires equal immutable trace identity.
- `internal/domain/integration/service_operation_store_test.go`: mismatch trace with same request digest is a conflict.
- `internal/domain/integration/service_operation_recovery_test.go`: revoked principal's expired lease is reconciled safely and stays unreadable.
- `internal/domain/integration/service_operation_recovery.go`: `MarkIndeterminate` rechecks persisted active principal inside the atomic scoped UPDATE; a revocation after Lookup cannot authorize the write.
- `internal/domain/integration/service_operation_recovery_test.go`: worker mark is denied after revocation; operator reconciliation remains possible, with no `agent_run`.
- `internal/domain/integration/service_operation_store_test.go`: caller-forged ConfigSHA256 is ignored; the stored registry-derived fingerprint remains authoritative.

**Execution order:** independent readiness C1–C3 → actual Go 1.25.13 focused/full/race/vet/prepush QA → independent exact-source post-code review → return redacted logs and findings to [#9](https://github.com/krukmat/fenix/issues/9). If the readiness gate remains non-PASS, do not infer acceptance from tests. Preserve uncommitted root `.gitignore` content (`scripts/__pycache__/*`). C4 is deferred; do not run C4 QA or introduce routes/CI/credentials/deployment. T4 security review stays separately tracked in #8 and does not block C1–C3 source-only closure.

## Operator scope update — 2026-11-10

**Execute only C1–C3 validation**. B2.C4 GET/status was expressly deferred by the owner and is **not a blocker** for C1–C3 source-only QA, independent review or scoped completion. Do not execute C4 readiness/code-review/test work, implement a resolver, or require C4 PASS in the #9 closure matrix. GET/status remains default-deny, so do not claim production integration, status polling or B4 status E2E. The T4 exact security diff in #8 remains separately pending for the auth boundary/release and can be reviewed independently; it is not a C4-driven prerequisite for inert C1–C3 code verification.

## Local procedure — follow this order; do not implement

**0. Guarded checkout / context.** The authoritative remote baseline is `aa26ad747b167e2eea839745616c9fb6ae5f3b0b` (workflow change rebased onto 10 Cloud documentation commits). Read `CLAUDE.md`, `AGENTS.md`, `docs/playbooks/AGENT_WORKFLOW_GUIDE.md`, `docs/policies/HITL_AUTONOMY_POLICY.md`, this handoff, C1–C3 tasks, [#3](https://github.com/krukmat/fenix/issues/3), [#8](https://github.com/krukmat/fenix/issues/8), [#9](https://github.com/krukmat/fenix/issues/9). **Known local pre-existing change**: uncommitted `.gitignore` addition for `scripts/__pycache__/*`. Preserve it verbatim; do not stage, discard, stash, overwrite, or include it in any B2.C reviewer packet. Fetching is safe; fast-forward only when the worktree can be preserved and no unrelated changes will be overwritten. When in doubt, use a temporary detached read-only worktree without switching the owner's working directory.

```bash
cd /Users/matias/fenix
git status --short
git branch --show-current
git rev-parse HEAD
git fetch origin feature/int-v2-dubbridge-consumes-fenix
git rev-parse origin/feature/int-v2-dubbridge-consumes-fenix
# Confirm the upstream contains baseline aa26ad747b167e2eea839745616c9fb6ae5f3b0b.
# Never reset/stash/stage the pre-existing .gitignore modification.
# Inspect divergence first; do not automatically pull on the owner's dirty worktree.
git log --oneline -5 origin/feature/int-v2-dubbridge-consumes-fenix
git diff -- .gitignore
git status --short
git rev-parse HEAD
GOTOOLCHAIN=go1.25.13 go version
```

**1. Re-run independent task-readiness FIRST on the revised C1 task and canonical plan.** The prior `needs_changes` is not a PASS. Confirm the actual task author kind and independent reviewer from the active workflow guide, and capture reviewer, criticality concurrence, verdict, artifact path/hash and source SHA. If C1 PASS, repeat readiness independently for C2 and C3; do not reuse C1's verdict. The reviewer workflow handles provider/model selection internally, **not the integration plan**. Example for code authored by ChatGPT/Codex (only if truthful):

```bash
CALLER_KIND=codex  # Only if Codex/ChatGPT authored the reviewed task; do not misrepresent authorship.
python3 scripts/peer-workflow-review.py task-readiness \
  --caller-kind "$CALLER_KIND" \
  --task docs/tasks/task_dubbridge_fenix_b2c1_sqlite_schema.md \
  --plan docs/plans/dubbridge-consumes-fenix-integration.md
```

If C1 remains `needs_changes` / `blocked`, post findings and **stop closure and implementation**; Go tests may still be executed as *diagnostic* evidence, never interpreted as acceptance. Once C1 readiness PASS is genuine, separately perform C2 and C3 independent readiness with their own `--task` paths and the same plan; do not recycle the C1 verdict. Record criticality concurrence and artifact SHA-256 per task. RRI is N/A for this initiative; do not run RRI scripts.

**2. Run read-only/source-preserving Go QA using the actual branch SHA.** These commands execute tests only with temporary test databases; never apply an up/down migration to a production or persistent project DB.

```bash
gofmt -l internal/infra/sqlite/service_operation_migration_test.go internal/domain/integration/service_operation_*.go
GOTOOLCHAIN=go1.25.13 go test ./internal/infra/sqlite ./internal/domain/integration -run '^TestB2C' -count=1 -v
GOTOOLCHAIN=go1.25.13 go test -race -count=3 ./internal/infra/sqlite ./internal/domain/integration
GOTOOLCHAIN=go1.25.13 go vet ./internal/infra/sqlite ./internal/domain/integration
GOTOOLCHAIN=go1.25.13 go test ./...
GOTOOLCHAIN=go1.25.13 bash scripts/qa-go-prepush.sh
git diff --check
```

Record each exact command, RC, toolchain, output log location and sha256, and first redacted failure. `gofmt -l` is listing only; do **not** run `gofmt -w`. Disaggregate changed-code B2.C failures from [global QA debt #4](https://github.com/krukmat/fenix/issues/4); skipped traceability/govulncheck are **SKIPPED**, not PASS. Preserve the actual changed-code reviewer baseline.

**3. Independent C1–C3 post-code review of the frozen Go/SQLite source**, using the active independent-review workflow, with findings and verdict **per task** (or explicit broader B2.C packet scope, never falsely labeled isolated). A useful bounded code inventory is:

```bash
git diff --name-only b17162743ed8ee4f3e99be09e910e3b3d1c01fb6 dc7354738dfc9633230f9b31f31b5bf625c73b3d -- internal/infra/sqlite/migrations/039_service_operation.up.sql internal/infra/sqlite/migrations/039_service_operation.down.sql internal/infra/sqlite/service_operation_migration_test.go 'internal/domain/integration/service_operation_*.go'
git diff --no-color b17162743ed8ee4f3e99be09e910e3b3d1c01fb6 dc7354738dfc9633230f9b31f31b5bf625c73b3d -- internal/infra/sqlite/migrations/039_service_operation.up.sql internal/infra/sqlite/migrations/039_service_operation.down.sql internal/infra/sqlite/service_operation_migration_test.go 'internal/domain/integration/service_operation_*.go'
```

The repo script's `post-code-review --base` covers **all diff to current worktree HEAD**. This branch now contains later documentation edits; never claim `--base b1716274` against current HEAD reviewed only C1–C3 source. Use a separate read-only exact-diff reviewer packet or a temporary **detached** worktree at the frozen code SHA; record actual path list, task, reviewer/version, source commit, base commit, verdict, artifact SHA. Do not create branches/commits or mutate code.

**4. Independently assess B2.A3.T4 with EXACT frozen diff**, separate from B2.C. Do not accidentally review all intervening B2.C documents.

```bash
git diff --name-only 43d935c4288a565b6c4d202208ebef16cc35e012 acafa3d3b59f0539b7fbc5ec3114a43c332bce80
git diff --no-color 43d935c4288a565b6c4d202208ebef16cc35e012 acafa3d3b59f0539b7fbc5ec3114a43c332bce80
```

Use the active independent security-review workflow. If running the script, point it at a temporary detached worktree with HEAD exactly `acafa3d`, not the current branch HEAD, and verify the diff path set. Inspect S2S auth scope, revocation/rotation, capability registry, error wrapping and redaction/retry semantics. Publish independent reviewer, exact paths and artifact SHA on [#8](https://github.com/krukmat/fenix/issues/8); do **not** treat T3's reviewer as T4 certification.

**5. Return C1–C3 artifacts only.** On [#9](https://github.com/krukmat/fenix/issues/9), report a compact matrix `[C1/C2/C3 readiness, focused Go, race, vet, full test, prepush, post-code peer | PASS/FAIL/NEEDS_CHANGES/NOT_RUN | RC | SHA | artifact path/hash | relevant finding]`; include B2.A2 dependency assessment distinguishing scoped owner acceptance from independent security PASS. On #8 report T4 separately. If local GitHub publication is unavailable, return the complete redacted reports to the owner for ChatGPT publication. Keep changed source and docs untouched in the local agent session.

## Exit decision (Cloud-owned)

- **C1–C3 stay SOURCE_CANDIDATE** until authentic readiness, Go QA and independent post-code review; `needs_changes` is not a waived gate.
- **C4 is DEFERRED / NON-BLOCKING FOR C1–C3** by operator direction. The prior technical prerequisites apply only if it is revived; GET/status remains default-deny. Do not include C4 in the C1–C3 gate table.
- ChatGPT Cloud owns C1–C3 remediation and reconciliation. Future C4 implementation is a separate, not-currently-scheduled task; no new C4 handoff.
- This document certifies **only the existence of prior evidence and documentary corrections**. It does not certify any current Go or security verification.
