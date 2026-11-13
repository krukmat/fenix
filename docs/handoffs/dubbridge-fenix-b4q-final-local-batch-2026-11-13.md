# DubBridge consumes Fenix — B4.Q deferred single local certification batch

**Status:** PREPARED, **DO NOT EXECUTE YET** unless the owner explicitly resumes local work. Cloud C1–C3 checkpoint: [2026-11-13 audit](../audit/dubbridge-fenix-b4q-cloud-first-preflight-2026-11-13.md); [cloud-first execution plan](../tasks/task_dubbridge_fenix_b4q_cloud_first_execution.md).

## Q5 execution evidence amendment — 2026-11-13

The owner returned genuine local gate evidence for the original frozen SHA pair: Fenix `d07adb664bc1f73cdff2e297f945e1e211a6459b`, DubBridge `bf36a9d3751a533dbf6b0a337ea6828d4be07e13`. **L1 SQLite 23 assertions, Go focused and go vet PASS; L2 cargo check and tests PASS (321 pass/1 ignored).** Fenix prepush FAILS at formatting before later subgates; DubBridge `cargo fmt --all -- --check` FAILS in three files; Clippy FAILS for 5 `manual_async_fn` methods in the S2S connector; independent Devstral L3 review was inconclusive and reported spot-checked false positives. **There is no overall GO.** No runtime route or credential was enabled.

**L0 correction of overly strict worktree precondition:** the local verifier created **clean, detached isolated worktrees at both EXACT frozen SHAs**, while leaving two unrelated user-owned working checkouts untouched. The earlier L0 branch/checkout gate legitimately reported FAIL against the handoff's literal original-checkout requirement, but the isolated worktrees are a **sounder and safer mechanism** for certifying immutable source. Future L0 may PASS when (i) both isolated worktrees are at exactly the reviewed commits, (ii) both have clean working trees, and (iii) no code is accidentally taken from the original working checkouts. Keep the original L0 deviation in the audit, do not falsify the previous record or reset/rebase the user's original trees.

**Q5 repair policy:** ChatGPT owns source remediation. DubBridge S2S `manual_async_fn` fixes are published in the same integration feature branch; actual Rust tests/lints on the new SHA are **NOT_RUN**. The source-level Go format drift is still unresolved and MUST NOT be marked PASS on the strength of earlier focused tests. The workspace Rustfmt drift in `apps/api/src/routes/workspace.rs` and `apps/gateway/src/proxy.rs` predates this S2S fix; do not modify unrelated behavior merely to achieve a green global formatting gate. Choose a scoped formatter check with an explicitly documented baseline exception, or perform mechanically formatted-only changes after confirming zero functional diff and any required owner gate approval. L3 needs a truly independent complete-diff review; unsubstantiated model claims and absent/out-of-scope fragments cannot certify or disqualify a security control.

**Targeted future recertification (only changed/failed gates):** Fenix Go gofmt and prepush after repairs; DubBridge fmt and `cargo check/test/clippy` on the new Rust connector source; and complete independent security review after those gates. Do not needlessly rerun previously green SQLite 23, Go vet/focused, or full tests on unchanged SHAs; the repo prepush may run its mandated test suite as part of its own gate. The original frozen SHA pair is **historical** once Q5 source changes are committed: freeze and verify the new pair before any recertification. No new local-agent session was authorized by this amendment.

## Scope and freeze contract

- **Repositories:** `krukmat/fenix`, `krukmat/dubbridge`. Same existing branch in both: `feature/int-v2-dubbridge-consumes-fenix`. **DubBridge consumes Fenix**, not vice versa.
- **Exact frozen SHA pair:** set in the final 2026-11-13 GitHub issue comments after all docs commits ([Fenix #10](https://github.com/krukmat/fenix/issues/10), [DubBridge #13](https://github.com/krukmat/dubbridge/issues/13)). Revalidate both before executing: never certify a different SHA silently. This handoff intentionally does not self-reference a changing branch HEAD.
- **No deployment, HTTP route mount, S2S token, capability provision, CI, branch creation or merge**. Never enable C4 GET/status. Tests must use local fixture credentials and isolated databases only.
- **RRI:** N/A DB-FENIX-INT Cloud authoring. Independent security T4 is not waived; do not substitute a source audit for peer approval.
- Source changes from C1–C3 fixed two broken Go test fixtures and one mutation guard. SQLite migration 040 and B1/B4 schemas/Node checker remain exact blobs previously verified. Do not repeat unchanged offline checks unless there is a new mismatch.

## One local execution, strict order and concise output

**Environment:** Go **1.25.13**, compatible Rust/Cargo toolchain (edition 2024), Node 22+ and Python3. Run offline insofar as dependencies are already cached. The user/local orchestrator supplies `FENIX_REPO` and `DUBBRIDGE_REPO` paths to their existing checkouts. No agent should invent a path or spawn a branch.

### L0 — readonly snapshot and working-tree sanity

```bash
cd "$FENIX_REPO"
git branch --show-current
git rev-parse HEAD
git status --short
go version
cd "$DUBBRIDGE_REPO"
git branch --show-current
git rev-parse HEAD
git status --short
rustc --version
cargo --version
```

**Stop on SHA mismatch** with the final frozen pair, dirty tree, missing toolchain or unexpected branch. Do not `git reset --hard`, `git clean`, force-push or mutate user-owned files. A `git pull --ff-only` is permitted only before the freeze has been confirmed and with user-approved checkout state; never modify the frozen SHA unannounced.

### L1 — Fenix Go and SQLite (single batch)

```bash
cd "$FENIX_REPO"
python3 scripts/int_b2b4_capability_sqlite_smoke.py
go test -count=1 -run '^(TestB2B4|TestB4|TestB2E)' ./internal/domain/integration ./internal/api ./internal/infra/sqlite
go vet ./...
bash scripts/qa-go-prepush.sh
```

`qa-go-prepush.sh` already includes fmt, complexity, lint, tests, coverage, deadcode, race-stability and repository pattern gates. **Do not additionally run a separate full `go test ./...` or another full race pass when prepush already executed those gates**. On failure, return only failing command, test, short root cause, non-sensitive log excerpt, and frozen SHA. Do not skip/waive failing mandatory gates without explicit authorization.

### L2 — DubBridge Rust (same verification batch)

```bash
cd "$DUBBRIDGE_REPO"
cargo fmt --all -- --check
cargo check -p dubbridge-connectors -p dubbridge-api
cargo test -p dubbridge-connectors -p dubbridge-api
cargo clippy -p dubbridge-connectors -p dubbridge-api --all-targets -- -D warnings
```

No live `FenixHttpsTransport`, real HTTP credential, worker, PostgreSQL tenant provisioning or gateway is needed. Check only the two affected Rust crates; avoid unrelated workspace-wide suites unless a dependency demands them. The [pinned Node checker](https://github.com/krukmat/dubbridge/blob/feature/int-v2-dubbridge-consumes-fenix/scripts/check-fenix-b4-contract.mjs) already had 29/29 PASS on the same script + schema blob SHAs; optionally run `node scripts/check-fenix-b4-contract.mjs` **once** only if those inputs have changed or for final independent reproduction.

### L3 — independent security/code review

**Only after L1/L2 evidence exists.** One independent reviewer (not the implementing Cloud agent) assesses the two frozen integration diffs and real test evidence. Focus: S2S identity/token rotation, Fenix user/operator RBAC, tenant/org/project cross-scope, schema version defaults and SQL trigger atomicity, POST preflight/header parsing, idempotency/replay/unknown outcome, absence of ToolRegistry execution bypass, unmounted routes and disabled-by-default configuration. Record findings as H/M/L with file+line, affected gate and reproducible evidence. This is not an instruction to activate or implement any business workflow.

## Result handoff to ChatGPT Q5

Return one compact matrix like:

```text
FENIX_SHA=<40hex> DUBBRIDGE_SHA=<40hex>
L0_SHA_CLEAN=PASS|FAIL
L1_PY_SQLITE=PASS|FAIL|NOT_RUN
L1_GO_FOCUSED=PASS|FAIL|NOT_RUN
L1_GO_VET=PASS|FAIL|NOT_RUN
L1_GO_PREPUSH=PASS|FAIL|NOT_RUN
L2_RUST_FMT=PASS|FAIL|NOT_RUN
L2_RUST_CHECK=PASS|FAIL|NOT_RUN
L2_RUST_TEST=PASS|FAIL|NOT_RUN
L2_RUST_CLIPPY=PASS|FAIL|NOT_RUN
L3_INDEPENDENT_REVIEW=PASS|FAIL|NOT_RUN
BLOCKERS=<short file+line with failing command if any>
```

ChatGPT handles Q5: triage and fix verified findings **in GitHub Cloud** and re-request only the specifically invalidated gate subset after a new SHA freeze. No local agent should expand scope or rewrite the source. A clean source-only matrix does **not** authorize B4 real E2E, production endpoint/credential/capability activation, deployment or merge; those require a separate explicit decision.
