---
doc_type: audit
id: DB-FENIX-B4Q-CLOUD-FIRST-2026-11-13
title: "Cloud-first B4.Q C1–C3 scoped preflight and evidence reuse"
status: cloud_source_ready_local_go_rust_pending
date: 2026-11-13
scope: "feature/int-v2-dubbridge-consumes-fenix; no production runtime"
---

# B4.Q C1–C3 Cloud-first preflight — 13 November 2026

**Owner-approved sequence:** [B4.Q Cloud-first task](../tasks/task_dubbridge_fenix_b4q_cloud_first_execution.md). Cloud-first development, one local batch only after C4 freeze. **Repositories:** `krukmat/fenix` and `krukmat/dubbridge`, existing `feature/int-v2-dubbridge-consumes-fenix` branches; DubBridge is the consumer. C4 status GET remains deferred. No local agents used in this checkpoint.

## C1 — Focused compiler/test-readiness findings and actual corrections

1. **Go API test fixture — confirmed call-arity defect.** `internal/api/s2s_admission_http_test.go` called `mustNewRouterWithConfig(db, testCfg())`, but the existing `internal/api/routes_test.go` helper requires `(t *testing.T, db *sql.DB, cfg config.Config)`. Corrected to `mustNewRouterWithConfig(t, db, testCfg())` in [commit `28595a4`](https://github.com/krukmat/fenix/commit/28595a454283eef426d2e2416f965749f0641dff). **Source-signature correction only; Go1.25 compilation not run.**
2. **Go fake-workspace negative test — confirmed ineffective test data.** `internal/domain/integration/s2s_b4_inert_interop_test.go` replaced a nonexistent `"input":{}` substring, while `b2eEnvelope()` actually contains `"input":{"v":1}`; it could leave a valid request unchanged and falsely assert a denial case. Replaced the exact fixture substring in [commit `9a8cf95`](https://github.com/krukmat/fenix/commit/9a8cf95b3433702a0be6815e2c1250e47e4ea7e2).
3. **Go negative fixture safety guard.** Added a precondition that every forged-envelope case with unchanged bearer must actually mutate the request before invoking admission; future fixture drift now fails explicitly, in [commit `a99bbba`](https://github.com/krukmat/fenix/commit/a99bbbaefa9972c9fa81869faa8d25a3c366000c).
4. **Rust API dependency check.** Compared the checked-in `reqwest 0.12` usage against published API docs; `Response::chunk()` does **not** require the separate `stream` feature (that feature applies to `bytes_stream()`), and `ClientBuilder::https_only`/no-redirect exist. No unnecessary dependency or lockfile churn was introduced. This is an API signature check, **not a cargo compile**.
5. Inspected in-scope Fenix operator RBAC policy, S2S admission/store/transactions, SQLite migration 040, Go/Rust contract types, test helper function signatures, workspace/user-account table columns and `Cargo.toml` dependencies. No further **proven** source incompatibilities recorded; unknown build/lint issues remain possible until local compiler execution.

## C2 — Evidence-preserving optimization (no redundant re-runs)

The previous [executed 2026-11-11 B4.Q audit](dubbridge-fenix-b4q-cloud-validation-2026-11-11.md) already recorded SQLite migration 040 **22/22 PASS** plus two-connection ordering **2/2 PASS** (24 total) and the exact pinned Node contract script **29/29 PASS**. Today's **read-only Git blob recheck** confirms **5/5 identical blobs**:

| Input | Git blob SHA | Reuse |
|---|---|---|
| Fenix SQL `040_s2s_capability_policy.up.sql` | `ad31aa7239243c350fd43da02771a8c8637cd1c7` | Previous 24 SQLite checks still apply to this unchanged SQL |
| DubBridge `scripts/check-fenix-b4-contract.mjs` | `012b1da765dc17d986fdd2833b8f8e4497d0c8df` | Previous 29 Node assertions still apply to this unchanged script |
| Fenix B1 request schema | `3ccbfc7e42c728e1cfcf4c15d86be00fcfcaa745` | Unchanged |
| Fenix B4 ack schema | `75e82ec4491b997922ca32504bc81f3e94095ca7` | Unchanged |
| Fenix B4 error schema | `b07df2b8eff120c7b265738f20a31e8a59363177` | Unchanged |

**Important:** This checkpoint did **not** execute those test binaries a second time. It reuses earlier actually executed evidence after checking content identity. **The repository Python wrapper `scripts/int_b2b4_capability_sqlite_smoke.py` (blob `701a8efce7d09edfd0ba01cb1b7fee13bab79233`) has still NOT_RUN as a file.** The Cloud sandbox cannot clone these repos (network DNS unavailable); do not claim this wrapper, full migrations 001–040 or Go's modernc driver is certified. If needed, run the wrapper **once** in the final consolidated local batch rather than inventing a new agent task.

## C3 — Contract/negative-fixture checks

**13/13 new read-only fixture checks PASS** using the current GitHub sources: B2.E canonical `b2eEnvelope()` and the HTTP test envelope are both nonempty; all forged-org, forged-project, missing/unknown capability, trace-conflict, and payload mutations target substrings actually present. The updated negative test checks mutation before admitting, and the router-helper call now supplies `testing.T`. Cross-language HTTP status/error wire, canonical Base64URL token and idempotency invariants already passed previous Q1–Q3 checks and were not modified by this C1 work.

**No HTTP interaction or live token used.** Rust client remains detached and Fenix policy tables are disabled-by-default.

## Verification levels and remaining local gates

| Level | Result |
|---|---|
| Scope source fixes C1 | Three concrete Go test corrections published in feature branch |
| Git blob retention C2 | **5/5 content identity PASS**, preserving prior 24 SQLite + 29 Node evidence |
| Current negative fixtures C3 | **13/13 source predicate PASS** |
| Full Go 1.25.13 tests, migration 001–040, fmt/race/vet/prepush | **NOT_RUN** |
| Cargo fmt/check/test/clippy, PostgreSQL tests | **NOT_RUN** |
| Independent code/security T4 review | **NOT_RUN** |
| S2S endpoint enable, capability/credential provisioning and real 2-service E2E | **NOT_AUTHORIZED / NOT_RUN** |

**Disposition:** scoped Cloud preflight C1–C3 finished with explicit content reuse, no formal release/gate elevation. C4 writes a single offline-local handoff and freezes branch SHAs in existing GitHub issue comments; the final local batch is deferred by owner request.
