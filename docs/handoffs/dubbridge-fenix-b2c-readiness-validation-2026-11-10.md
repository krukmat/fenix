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

## Workflow synchronization checkpoint — 2026-11-10

- Verified remote source baseline before this handoff update: `aa26ad747b167e2eea839745616c9fb6ae5f3b0b`; workflow changes in [LOCAL-PRIMARY-PEER-REVIEW](../tasks/task_local_primary_peer_review.md), [workflow guide](../playbooks/AGENT_WORKFLOW_GUIDE.md) and `scripts/peer-workflow-review.py`.
- New reviewer route: standard = `FENIX_REVIEW_MODEL` (default `gemma4:26b-a4b-it-qat`); critical = `FENIX_REVIEW_CRITICAL_MODEL` (default `qwen3.6:35b-a3b`), both **blocking primary**. The previous Qwen advisory mode is superseded. Only local timeout/unavailability permits independent external fallback; unknown/local authors remain blocked.
- Local workflow task records `make qa-peer-workflow-review` **60 PASS** at the published SHA; this is **not** B2.C Go QA, C1 readiness, C1–C3 post-code review or B2.A3.T4 independent security acceptance.
- Repository-wide documentation gate is still reported failing on legacy plan frontmatter, ADR status/index mismatches and dangling ADR-036 references. This remains separate from B2.C and should be recorded, not silently waived.
- Owner's local `/Users/matias/fenix/.gitignore` contains pre-existing uncommitted ignore rule `scripts/__pycache__/*`; do not touch or stage it. Neither the workflow commit nor this handoff includes that local modification.
- RRI: N/A — DB-FENIX-INT/cloud. No routes, CI, migration activation, credentials, runtime dispatch, deployment or main merge.

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

**1. Re-run independent task-readiness FIRST on the updated C1 task + plan.** Inspect the prior local `needs_changes` artifact if available; report its real path/hash and actual reviewer. The documentation is revised, **not approved**. Follow the **published** workflow at `aa26ad7`, not the previous fallback/advisory policy. For independently identified `codex`, `claude-code`, or `remote-provider` authors, **standard** readiness/code review uses local Gemma as the blocking primary; **critical** readiness/code review uses local Qwen as the blocking primary. C1–C3 and T4 declare `criticality: critical`, so **Qwen is primary** unless the independently reviewed task is formally reclassified. External reviewers are permitted only if the assigned local primary times out or is unavailable; a substantive `needs_changes`, invalid verdict or oversized packet blocks without fallback. `local-provider` and `unknown` callers fail closed because author independence cannot be established. This B2.C source is Cloud-authored, so use `--caller-kind codex` when Codex/ChatGPT is the actual author, rather than incorrectly identifying the local validation runner as the author. Capture the actual local model, primary/fallback, verdict and artifact SHA. Example:

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

**3. Independent C1–C3 post-code review of the frozen Go/SQLite source**, with **Qwen primary for critical tasks** under the published `aa26ad7` policy, findings and verdict **per task** (or explicit broader B2.C packet scope, never falsely labeled isolated). A useful bounded code inventory is:

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

Use the published critical-task Qwen-primary independent security-review flow; only timeout/unavailability enables external fallback. If running the script, point it at a temporary detached worktree with HEAD exactly `acafa3d`, not the current branch HEAD, and verify the diff path set. Inspect S2S auth scope, revocation/rotation, capability registry, error wrapping and redaction/retry semantics. Publish independent reviewer, exact paths and artifact SHA on [#8](https://github.com/krukmat/fenix/issues/8); do **not** treat T3's reviewer as T4 certification.

**5. Return artifacts only.** On [#9](https://github.com/krukmat/fenix/issues/9), report a compact matrix `[C1/C2/C3 readiness, focused Go, race, vet, full test, prepush, post-code peer | PASS/FAIL/NEEDS_CHANGES/NOT_RUN | RC | SHA | artifact path/hash | relevant finding]`; include B2.A2 dependency assessment distinguishing scoped owner acceptance from independent security PASS. On #8 report T4 separately. If local GitHub publication is unavailable, return the complete redacted reports to the owner for ChatGPT publication. Keep changed source and docs untouched in the local agent session.

## Exit decision (Cloud-owned)

- **C1–C3 stay SOURCE_CANDIDATE** until authentic readiness, Go QA and independent post-code review; `needs_changes` is not a waived gate.
- **C4 remains DESIGN_ONLY/BLOCKED** until C1–C3 and T4 independent acceptance are explicit. There is no GET status production route.
- ChatGPT Cloud owns source remediation and status reconciliation on reported failures, and will prepare any later C4 implementation separately behind its security gate.
- This document certifies **only the existence of prior evidence and documentary corrections**. It does not certify any current Go or security verification.
