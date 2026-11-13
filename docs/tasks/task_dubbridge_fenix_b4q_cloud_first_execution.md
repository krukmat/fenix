---
doc_type: task
id: DB-FENIX-B4.Q-CLOUD-FIRST
title: "B4.Q Cloud-first technical certification — deferred single local batch"
status: cloud_c1_c4_delivered_local_l1_l3_deferred
owner: ChatGPT Cloud
date: 2026-11-13
phase: B4
tags: [dubbridge, fenix, s2s, cloud, qa]
---

# B4.Q — Cloud-first sequencing (owner decision, 2026-11-13)

**Decision:** finish all feasible development, source repair, offline checks and documentation in Cloud **before** any local action. One final, consolidated local/environment-specific verification batch is the only planned local work. **No local-agent prompts until Cloud freeze.** No extra branches, CI, route mounting, operational credentials, live capability, paid resources, deployment or main merge. C4 GET/status remains deferred.

**Repositories:** `krukmat/fenix` and `krukmat/dubbridge`, existing branch `feature/int-v2-dubbridge-consumes-fenix`. **DubBridge consumes Fenix.** No business workflow/model activation.

## 2026-11-13 Cloud C1–C4 completion checkpoint

**Cloud audit:** [B4.Q C1–C3 preflight](../audit/dubbridge-fenix-b4q-cloud-first-preflight-2026-11-13.md). **Final deferred local handoff:** [one L0–L3 batch](../handoffs/dubbridge-fenix-b4q-final-local-batch-2026-11-13.md).

| Task | Cloud completion | Actual evidence / boundary |
|---|---|---|
| **C1** | Source readiness audited; two confirmed Go test bugs fixed and one fixture-drift guard added | Router test now supplies `testing.T`; forged-workspace negative test mutates the actual input; tests remain NOT_RUN on Go1.25 |
| **C2** | **Evidence reused, no redundant runs:** exact current migration 040, Node B4 script and three Fenix schemas match 5/5 previously tested Git blobs | Prior actual SQLite **24/24** and Node **29/29** stand for unchanged inputs; new repo Python wrapper remains **NOT_RUN** and is in the deferred L1 batch |
| **C3** | Current Go test-fixture/contract anchors checked **13/13 static PASS**, with genuine mutation guard | Not an HTTP E2E or executed Go/Rust unit test |
| **C4** | Audits consolidated, one handoff with stop conditions/commands created; exact final SHA pair is published in Fenix #10 / DubBridge #13 GitHub freeze comments after documentation commit | **Local L1–L3 deliberately deferred**; no activation, merge or review waiver |

**Completion meaning:** all ChatGPT Cloud authoring/reconciliation tasks delivered; it does **not** mean B4.Q formal certification has passed. Await explicit owner choice before running L1–L3. Q5 findings remediation remains Cloud-owned.

## Baseline — do not repeat completed checks without changed scope

- **B4.Q1–Q3 completed:** Cloud static Go/Rust comparison 21/21 PASS; SQLite migration 040 isolated 24/24 PASS; pinned B1/B4 Node contract script 29/29 PASS; isolated Go token-parser compatibility PASS.
- Both codebases contain **inert source candidates** only, without formal Go 1.25.13/Rust compiler QA.
- Evidence: `docs/audit/dubbridge-fenix-b4q-cloud-validation-2026-11-11.md`; existing GitHub issue Fenix #10, DubBridge #13. Do not relabel old checks as a compiled suite.

## Cloud work first — ChatGPT responsible for every subtask

| Order | Task | Focused implementation/evidence | Depends on | Result gate |
|---|---|---|---|---|
| **C1** | **Source/compiler-readiness audit** | Inspect only **changed** Go/Rust source, test fixtures, imports, migrations/constraints and manifest/lock edges. Trace symbols using exact repo files, fix proven compile/test defects. Avoid broad whole-repo sweeps | Q1–Q3 existing evidence | No known static blockers in changed scope; diffs published |
| **C2** | **Execute existing offline checks from repo** | Run the committed Python SQLite 038–040 smoke (not just an equivalent harness) and pinned Node B1/B4 script in Cloud when feasible; fix deterministic failures, record exact commands and exit codes. Do not run expensive local tools | C1 | Reproducible source-to-evidence links; no fake Go/Rust PASS |
| **C3** | **Negative/contract reconciliation** | Reconcile only cross-repo changed boundaries: request & response status/body, canonical token, UUIDv5 key, principal/org/project, policy-disable under reservation lock, denied/revoked/unknown capability, ambiguous outcome. Add missing Go/Rust test **source** where substantive | C1, C2 | No known contradictory contract behavior; no live traffic |
| **C4** | **Final freeze and handoff** | Review scoped diffs, consolidate existing Q1–Q3 and C1–C3 evidence, list known blockers and NOT_RUN gates, freeze **both exact branch SHAs**, and write ONE self-contained local execution handoff. No additional Cloud tasks after freeze unless a verified blocker appears | C1–C3 | Cloud-source-ready checkpoint; local batch unlocked |

### Cloud task microsteps

- **C1.a / C1.b:** Fenix Go/SQLite and DubBridge Rust/postgres changed-file review; implement fixes, preserving inert/default-deny boundaries.
- **C2.a / C2.b:** repository Python SQLite smoke and Node contract checker; store one concise combined report, avoid repeat runs when inputs are unchanged.
- **C3.a / C3.b:** semantic contract drift and failure-path analysis; author/revise targeted tests once, no new dummy workflows.
- **C4.a / C4.b:** final scoped commit comparison, SHA freeze and one local handoff with commands, expected outputs, and stop conditions.

## Local work LAST — one consolidated batch, not a development lane

| Final order | Task | Owner | Required execution | Evidence to return |
|---|---|---|---|---|
| **L1** | Fenix Go cert | Compatible Go 1.25.13 environment; local agent only if needed | `go test` focused on changed packages, full `go test ./...`, `go test -race` changed packages, `go vet ./...`, actual migrations 001–040/modernc, and repo `bash scripts/qa-go-prepush.sh` **once** where dependencies permit | Short PASS/FAIL matrix, exact command, exit status, failing test names, commit SHA; no verbose logs unless a failure |
| **L2** | DubBridge Rust cert | Compatible Rust toolchain; same local session as L1 when feasible | `cargo fmt --all -- --check`, `cargo check -p dubbridge-connectors -p dubbridge-api`, `cargo test -p dubbridge-connectors -p dubbridge-api`, `cargo clippy -p dubbridge-connectors -p dubbridge-api --all-targets -- -D warnings` | Same concise matrix and exact SHA; no unrelated workspace-wide runs by default |
| **L3** | Independent code/security review | Independent authorized reviewer | Review **frozen two-repo diff**, credential/authorization boundary and QA outputs, security T4 separately. No code implementation by reviewer. | Findings H/M/L with file/line and blocker vs follow-up disposition |
| **Q5** | Findings/closure synthesis | ChatGPT Cloud (no new local tasks unless needed) | Triage findings, make scoped Cloud fixes if needed, request only affected test subsets once, prepare GO/NO-GO evidence | Signed-off result **only** if real gates actually passed |

**Efficiency rules:** one frozen SHA pair, one local handoff, one local session, no duplicate agent loops, no re-running unaffected suites. A change after freeze invalidates *only relevant* gate evidence; re-run necessary affected checks and disclose SHA drift. Avoid claiming certification from static checks or missing-output tests. Keep the S2S route disabled until a separate explicit activation/security approval; this plan does not authorize E2E real traffic.

**Progress reporting:** use per-task progress bars. 100% means Cloud source/evidence delivered or an executed local gate with logs, never equivalence between the two.
