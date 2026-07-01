---
doc_type: plan
id: portable-agent-workflow-port-plan
title: "Portable Agent Workflow Port Plan"
status: completed
created: 2026-06-28
updated: 2026-07-01
---

# Portable Agent Workflow Port Plan

> **Status**: Complete (all listed tasks complete as of 2026-07-01)
> **Date**: 2026-06-28
> **Owner**: Workflow / DevEx
> **Source**: `/Users/matias/dubbridge/docs/proposals/portable-agent-workflow.md` (DubBridge "Portable Agent Workflow Extraction")
> **Primary references**: `CLAUDE.md` / `AGENTS.md` local wrappers, `README_AGENT_ORDER.md`, `docs/playbooks/AGENT_WORKFLOW_GUIDE.md`, `docs/architecture.md`
> **Precedence rule**: `CLAUDE.md` and `AGENTS.md` remain local agent-specific authority wrappers. The portable workflow sequence lives in `docs/playbooks/AGENT_WORKFLOW_GUIDE.md`; wrappers should reference it instead of duplicating its order. If a wrapper states a stricter repository rule, the stricter local rule wins until both documents are reconciled in an explicit edit.

---

## 1. Purpose

Port the reusable, project-agnostic agent operating contract from DubBridge into fenix, **adapting** (not copy-pasting) every artifact to fenix's existing conventions: local `CLAUDE.md` / `AGENTS.md` wrappers, `doc_type` frontmatter vocabulary, `docs/decisions/` ADRs, `docs/plans/` plans, `features/` BDD, the existing `make` QA surface, and the `scripts/hooks/*` install model.

The central invariant being adopted:

> No implementation starts until the agent has loaded the workflow, identified governing context, **computed risk (RRI)**, and satisfied the **approval gate** for that task.

Fenix already has most of the qualitative version of this (task cards, approval prose, hooks, ADRs). The gap this plan closes is the **deterministic** layer: a measured risk score (RRI), consolidated approval policy keyed to that score, enforced doc vocabulary, anti-forgetting preflight, and two missing QA gates.

## 2. Objective

When this plan is complete, a fresh agent session in fenix can:

1. Load workflow authority deterministically at session start (preflight).
2. Compute a reproducible risk score for any task with a script, not by hand.
3. Resolve the approval gate, effort, model tier, and decomposition requirement from that score.
4. Have its plan/task/ADR docs validated against a closed `doc_type` vocabulary.
5. Run two new QA gates (maintainability, config-secrets) locally and in CI.

## 3. Scope included

Phases **A–F** of the adapted port:

- **Phase A — Control-plane core**: `scripts/rri.py` (+ tests), `docs/policies/RRI_POLICY.md`, `docs/policies/HITL_AUTONOMY_POLICY.md`, `README_AGENT_ORDER.md`, `CLAUDE.md` authority wiring, `make qa-rri`. ✅ Done.
- **Phase B — Docs enforcement**: `scripts/check_okf_frontmatter.py` adapted to `doc_type`, ADR index `docs/decisions/README.md` + reference-integrity check, task-ledger contract extension (RRI + HP/EC + coverage certification) + `scripts/check-task-unit-coverage.sh`, `make qa-docs` / `qa-okf-frontmatter`. ✅ Done.
- **Phase C — Anti-forgetting**: `scripts/agent-preflight.py` (+ tests), `.claude/settings.json` (`SessionStart` + `PreToolUse`), `.gitignore` `.agent/`. ✅ Done.
- **Phase D — QA gates**: `scripts/check-maintainability.py` + `scripts/check-config-secrets.sh`, wired to pre-push and CI. ✅ Done.
- **Phase E — Local-model (Gemma) layer**: `gemma_local.py` transport, `delegate-low-rri.py`, `gemma-code-review.py`, `check-review-budget.py`, `adjudicator-packet.py`, `gemma-push-review.py`, `push_review_commit.py`, `gemma-audit-report.py`, `docs/policies/LOCAL_MODEL_POLICY.md`, Makefile targets, pre-push budget gate, opt-in CI job. Complete as of 2026-07-01.
- **Phase F — Provider-aware peer workflow review gates**: a scripted blocking peer-review layer for task readiness and post-code task closure. Claude Code callers are reviewed by Codex; Codex callers are reviewed by Claude; other local or remote providers default to Claude review. Approved 2026-07-01; implemented 2026-07-01 (PAW-F2 script + tests, PAW-F4 QA target + workflow-guide wiring). Hardening task `PAW-F7` completed 2026-07-01 to make reviewer CLI discovery robust across cross-agent sessions.
- **Phase F8 addendum — Reviewer CLI override operations**: the workflow guide documents `FENIX_CODEX_BIN` and `FENIX_CLAUDE_BIN`, plus the expected troubleshooting order for blocked peer review caused by discovery or authentication issues.
- **Phase F9 addendum — Codex review adapter compatibility (Codex CLI 0.142.5)**: `PAW-F9` fixed the `post-code-review` Codex adapter after `codex review` dropped `--instructions`, made `--base` and a positional prompt mutually exclusive, and standardized on natural-language findings. The adapter now invokes `codex review --base <ref>` and translates its native `- [P#]` findings into the gate JSON verdict (`needs_changes` if any finding, else `pass`) via `parse_codex_review_output`. Without this, every Claude Code post-code review silently fell back to the local Gemma reviewer. Completed 2026-07-01.
- **Phase F10 addendum — Peer-review diff scoping and fallback timeout fixes**: `PAW-F10` fixed two correctness defects surfaced by the Codex review of the gate itself. (1) `read_diff` used the three-dot `<base>...HEAD` form, which compares commits only and returned an empty diff in the common pre-commit closure flow — letting a reviewer `pass` unreviewed code; it now uses the two-dot `git diff <base>` (working tree) form. (2) `invoke_local_fallback_reviewer` ignored the caller `--timeout`; it now caps the local model's idle and wall limits at the supplied timeout. Reviewed by local Gemma at both checkpoints (readiness + code review, both PASS). Completed 2026-07-01.
- **Phase F6 addendum — Local fallback reviewer policy**: peer workflow review may use a local-model backup reviewer only after the primary reviewer is blocked by timeout or reviewer unavailability. The fallback remains fail-closed, must be explicit in artifacts, and does not replace HITL approval.
- **Phase F3 addendum — Agnostic workflow guide**: `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` is now ported as the portable workflow source of truth for ordering, with `README_AGENT_ORDER.md`, `CLAUDE.md`, `AGENTS.md`, and preflight referencing it instead of encoding separate workflow orders.
- **Phase F5 addendum — OpenAI task-card model defaults**: task-card workflow guidance now defaults OpenAI recommendations to `gpt-5.4`, uses `gpt-5.4-mini` for cost-prioritized work, and reserves `gpt-5.5` for clearly higher-autonomy or higher-complexity tasks.

## 4. Scope excluded

- No fenix **runtime** changes (`internal/`, `bff/`, `mobile/` product code). This is a workflow/DevEx port only.
- No wholesale rewrite of existing CLAUDE.md/AGENTS.md rules — keep local wrappers, but route shared workflow order through `docs/playbooks/AGENT_WORKFLOW_GUIDE.md`.
- No hook-enforced peer review in Phase F. Enforcement is scripted and report-contract blocking, not a PreToolUse denial.

## 5. Governing constraints

- `CLAUDE.md` is the highest authority and OVERRIDES; ported policies are subordinate.
- Task records in `docs/tasks/` must carry the mandatory frontmatter defined in `CLAUDE.md`.
- Canonical plans (`docs/plans/`) and ADRs (`docs/decisions/`) must stay Git-trackable.
- Knowledge-management / Obsidian rules in `CLAUDE.md` apply to every doc this plan touches.
- Existing public behavior of `make`, hooks, and CI must not break; all additions are new targets/jobs wired in additively.

## 6. Design decisions (adaptation mapping)

| Concern | DubBridge source | Fenix adaptation |
|---|---|---|
| Workflow guide | `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` | `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` ported as portable workflow source; `CLAUDE.md` / `AGENTS.md` stay local wrappers |
| Plans dir | `docs/plan/` | `docs/plans/` |
| ADRs dir | `docs/adr/` | `docs/decisions/` |
| BDD dir | `docs/bdd/` | `features/` (`*.feature` + `features/README.md`) |
| Doc vocabulary key | OKF `type` | fenix **`doc_type`**, closed set extended to `task, adr, summary, audit, handoff, plan, policy, playbook, proposal, roadmap` |
| Env var prefix | `DUBBRIDGE_*` | `FENIX_*` |
| RRI platform profile | rust/go/rn/python/generic | reuse `go` (gocyclo) + `rn`/JS (ESLint) — both already supported by `rri.py` |
| Model tiers | vendor-resolved | economy=`claude-haiku-4-5`, balanced=`claude-sonnet-4-6`, premium=`claude-opus-4-8`; local default=`gemma4:26b-a4b-it-qat`, local fallback=`gemma4:12b-it-qat` |
| Hook install | `.githooks/` + `scripts/hooks/*` | keep `scripts/hooks/*` + `make install-hooks`; add `.claude/settings.json` for SessionStart/PreToolUse preflight |
| Anchor rubric (D/K/P) | DubBridge crate paths | fenix paths: high=`internal/domain/{policy,audit,auth}`, migrations, tool/agent runtime; medium=`internal/api`, `mobile/`, `bff/`; low=`docs/`, tests |
| Cross-provider review | — | provider-aware peer review: Claude Code -> Codex, Codex -> Claude, all other local/remote providers -> Claude |

## 7. Affected components

```
README_AGENT_ORDER.md                     (new)
CLAUDE.md                                  (edit: authority-order + policy refs)
AGENTS.md                                  (edit: shared workflow wrapper + doc_type vocabulary alignment)
docs/playbooks/AGENT_WORKFLOW_GUIDE.md     (new: portable workflow source of truth)
.gitignore                                 (edit: .agent/)
.claude/settings.json                      (new: SessionStart + PreToolUse)
docs/policies/RRI_POLICY.md                (new)
docs/policies/HITL_AUTONOMY_POLICY.md      (new)
docs/decisions/README.md                   (new: ADR index)
scripts/rri.py                             (new, adapted)
scripts/rri_test.py                        (new, adapted)
scripts/check_okf_frontmatter.py           (new, adapted to doc_type)
scripts/check-doc-consistency.sh           (new, adapted to docs/decisions)
scripts/check-task-unit-coverage.sh        (new, adapted)
scripts/agent-preflight.py                 (new, adapted)
scripts/agent_preflight_test.py            (new, adapted)
scripts/check-maintainability.py           (new, adapted)
scripts/check-config-secrets.sh            (new, adapted)
scripts/peer-workflow-review.py            (done: provider-aware task readiness + post-code review gate)
Makefile                                   (edit: qa-rri, qa-okf-frontmatter, qa-docs, qa-maintainability, qa-config-secrets; wire into ci)
scripts/hooks/pre-push                     (edit: route new gates by changed paths)
.github/workflows/ci.yml                   (edit: add jobs mirroring new gates)
docs/tasks/<task templates>                (edit: RRI + HP/EC + coverage-cert fields)
```

## 8. Task decomposition

Each task has its own ledger file in `docs/tasks/`. Tasks are executed one at a time with a task card and explicit approval, per `CLAUDE.md`.

| ID | Task file | Title | Type | Phase | Status |
|---|---|---|---|---|---|
| PAW-A1 | `task_paw_a1.md` | Port RRI calculator + tests + `make qa-rri` | development | A | ✅ done |
| PAW-A2 | `task_paw_a2.md` | Author `RRI_POLICY.md` (fenix anchor rubric + model tiers) | docs | A | ✅ done |
| PAW-A3 | `task_paw_a3.md` | Author `HITL_AUTONOMY_POLICY.md` + `README_AGENT_ORDER.md` + CLAUDE.md wiring | docs | A | ✅ done |
| PAW-B1 | `task_paw_b1.md` | Port doc_type frontmatter validator + `make qa-okf-frontmatter` | development | B | ✅ done |
| PAW-B2 | `task_paw_b2.md` | ADR index + reference-integrity check + `make qa-docs` | development | B | ✅ done |
| PAW-B3 | `task_paw_b3.md` | Task-ledger contract extension (RRI/HP-EC/coverage) + coverage validator | development | B | ✅ done |
| PAW-C1 | `task_paw_c1.md` | Port `agent-preflight.py` + tests | development | C | ✅ done |
| PAW-C2 | `task_paw_c2.md` | `.claude/settings.json` hooks + `.gitignore` `.agent/` | config | C | ✅ done |
| PAW-D1 | `task_paw_d1.md` | Port maintainability gate (fenix stack budgets) | development | D | ✅ done |
| PAW-D2 | `task_paw_d2.md` | Port config-secrets gate | development | D | ✅ done |
| PAW-E1 | `task_paw_e1.md` | Port `gemma_local.py` transport + tests + `LOCAL_MODEL_POLICY.md` | dev+docs | E | ✅ done |
| PAW-E2 | `task_paw_e2.md` | Port `delegate-low-rri.py` + tests | development | E | ✅ done |
| PAW-E3 | `task_paw_e3.md` | Port `gemma-code-review.py` + tests + `check-review-budget.py` + tests | development | E | ✅ done |
| PAW-E4 | `task_paw_e4.md` | Port `adjudicator-packet.py` + tests | development | E | ✅ done |
| PAW-E5 | `task_paw_e5.md` | Port `gemma-push-review.py` + `push_review_commit.py` + `gemma-audit-report.py` + tests | development | E | ✅ done |
| PAW-E6 | `task_paw_e6.md` | Phase E wiring: Makefile targets + pre-push budget gate + opt-in CI job | config | E | ✅ done |
| PAW-F0 | `task_paw_f0_peer_review_dependency_reconciliation.md` | Reconcile PAW dependency drift before Phase F gates | docs | F | ✅ done |
| PAW-F1 | `task_paw_f1_peer_review_policy_contract.md` | Define provider-aware peer-review policy and reporting contract | docs | F | ✅ done |
| PAW-F2 | `task_paw_f2_peer_workflow_review_script.md` | Implement peer workflow review script, adapters, and tests | development | F | ✅ done |
| PAW-F4 | `task_paw_f4_peer_review_wiring.md` | Wire QA target and workflow guidance for peer review gates | config | F | ✅ done |
| PAW-F6A | `task_paw_f6a_peer_review_local_fallback_policy.md` | Define policy for local fallback in peer workflow review | docs | F | ✅ done |
| PAW-F6B | `task_paw_f6b_peer_review_local_fallback_script.md` | Implement local fallback in peer workflow review script | development | F | ✅ done |
| PAW-F7 | `task_paw_f7_peer_reviewer_cli_discovery.md` | Harden reviewer CLI discovery for cross-agent peer review sessions | development | F | ✅ done |
| PAW-F8 | `task_paw_f8_peer_review_cli_override_docs.md` | Document reviewer CLI overrides and peer-review troubleshooting flow | docs | F | ✅ done |

> **Note on id numbering:** the peer-review wiring task is numbered **PAW-F4**, not PAW-F3. The id `PAW-F3` was already claimed by the completed "Agent workflow documentation order refactor" addendum (`task_paw_f3_agent_workflow_document_order.md`, see §3 Phase F3 addendum). The wiring task was renumbered to PAW-F4 to avoid a duplicate id and preserve traceability.

## 9. Dependencies

```
PAW-A1 ─┐
PAW-A2 ─┼─> PAW-A3 (policy refs RRI script + RRI policy)
        │
PAW-A1 ──> PAW-B3 (coverage cert references RRI bands)
PAW-A3 ──> PAW-C1 (preflight summary cites authority order + policies)
PAW-B1 ──> PAW-B2 (doc gate aggregate qa-docs composes validators)
PAW-B1, PAW-B2, PAW-B3 ──> Makefile qa-docs aggregate
PAW-C1 ──> PAW-C2 (hooks invoke preflight script)
PAW-D1, PAW-D2 independent; wire into pre-push/CI last
```

PAW-E1 ──> PAW-E2 ──> PAW-E5
PAW-E1 ──> PAW-E3 ──> PAW-E4
                  └──> PAW-E5
PAW-E2, PAW-E3, PAW-E4, PAW-E5 ──> PAW-E6

PAW-F0 ──> PAW-F1 ──> PAW-F2 ──> PAW-F4
PAW-F4 ──> PAW-F6A ──> PAW-F6B
PAW-F4, PAW-F6B ──> PAW-F7
PAW-F7 ──> PAW-F8

Phase F prerequisite anchors:

- `PAW-F1` depends on `PAW-A3`, `PAW-B3`, and `PAW-C2` because it changes the approval/reporting contract, task ledger expectations, and preflight guidance.
- `PAW-F2` depends on `PAW-F1`; implementation must follow the approved provider-resolution contract.
- `PAW-F4` (peer-review wiring; renumbered from the plan's original F3 slot to avoid the completed PAW-F3 addendum id) depends on `PAW-F2`; wiring cannot happen before the script and mocked tests exist.
- `PAW-F6A` depends on `PAW-F4`; the fallback policy must be documented before behavior changes.
- `PAW-F6B` depends on `PAW-F6A`; the script must implement only the documented fallback contract.
- `PAW-F7` depends on `PAW-F4` and `PAW-F6B`; CLI discovery hardening must preserve the wired workflow contract and the existing fallback behavior.
- `PAW-F8` depends on `PAW-F7`; operator guidance should reflect the implemented discovery behavior and override knobs.
- Phase F does **not** depend on unfinished/advisory Gemma push-review work. It is a provider-independence safety gate for future task execution.

Hard ordering: A before B before C before D before E for the original *wiring* steps; within a phase, doc/script authoring can proceed in parallel until the Makefile/hook/CI wiring step, which is serialized. Within Phase E: E1 first (transport); E2 and E3 in parallel (both depend on E1 only); E4 after E3; E5 after E2 and E3; E6 last. Within Phase F: F0 reconciles dependency/status drift first, F1 defines the policy contract, F2 implements the script, F4 wires the workflow, F6A defines fallback policy, and F6B implements the fallback behavior.

## 10. Risks and open questions

- **R1 — gocyclo availability**: `rri.py --auto-cc` (go profile) needs `gocyclo`. Fenix's `make complexity` already uses it via `install-tools`; PAW-A1 must confirm and fall back to manual `--C`/`--cc` + Low-confidence when absent.
- **R2 — Python 3.9**: local Python is 3.9.6. Ported scripts must avoid 3.10+ syntax (`match`, `X | Y` unions in annotations) or pin a runtime. PAW tasks must verify under 3.9.
- **R3 — doc_type vocabulary expansion**: extending the closed set (`plan/policy/playbook/proposal/roadmap`) touches `CLAUDE.md`'s declared vocabulary. PAW-B1 must update `CLAUDE.md` and the validator together to avoid drift.
- **R4 — Existing docs may fail the new validators**: 30+ ADRs and many plans predate frontmatter rules. PAW-B1/B2 must run in **report-only** mode first, inventory violations, and gate only new/changed docs initially (fail-closed on new, grandfather existing) to avoid a repo-wide red gate.
- **R5 — PreToolUse hook ergonomics**: a too-strict `PreToolUse` deny can block legitimate edits. PAW-C2 must scope the hook to fail-closed only when preflight was never marked this session, and document the one-command remedy.
- **R6 — Peer-review availability**: Phase F relies on local Claude Code and Codex CLIs for scripted review. If the required peer CLI is unavailable or unauthenticated, the script must write a blocked artifact and the caller must stop rather than self-review.
- **R7 — Provider identity drift**: third-party local or remote providers must pass a concrete `caller-provider` when known and a normalized `caller-kind`; unknown providers default to Claude peer review.
- **R8 — Task dependency drift**: current task ledgers and the PAW plan may disagree on Phase E statuses and blockers. PAW-F0 exists to reconcile the canonical plan before Phase F implementation begins.
- **Open**: whether to retrofit RRI/HP-EC fields onto the 57 existing task files (proposed: template forward-only; retrofit out of scope).

## 11. Verification strategy

- **Scripts**: each ported Python script ships with adapted unit tests; `make qa-rri` runs `rri_test.py` and `agent_preflight_test.py`. Shell gates verified by fixture runs (passing + failing sample).
- **Validators**: run in report-only mode against the current repo, capture baseline, then enable fail-closed on changed paths.
- **Wiring**: `make ci` mirrors the new gates; pre-push routes them by changed path category. Each task's acceptance criteria lists exact commands.
- **Dogfooding**: these very task files adopt the target ledger shape (RRI placeholder + HP/EC) so the contract is exercised before it is enforced.
- **Peer review gates**: Phase F mocked tests must cover reviewer resolution, Claude adapter invocation, Codex adapter invocation, invalid JSON, unavailable peer CLI, non-pass verdicts, artifact writing, and secret redaction. Live peer review is required by workflow once PAW-F3 is complete.

## Appendix — Source → target file map

| DubBridge source | Fenix target | Strategy |
|---|---|---|
| `scripts/rri.py`, `scripts/rri_test.py` | same paths | copy + FENIX_ env + go/rn profile + fenix anchor rubric |
| `docs/policies/RRI_POLICY.md` | `docs/policies/RRI_POLICY.md` | copy + fenix paths/tiers |
| `docs/policies/HITL_AUTONOMY_POLICY.md` | same | copy + reconcile with CLAUDE.md/AGENTS.md approval prose |
| `README_AGENT_ORDER.md` | same | copy, CLAUDE.md as #1 |
| `scripts/check_okf_frontmatter.py` | same | adapt OKF `type` → fenix `doc_type` |
| `scripts/check-doc-consistency.sh` | same | adapt `docs/adr/` → `docs/decisions/` |
| `scripts/check-task-unit-coverage.sh` | same | adapt task frontmatter + HP/EC parsing |
| `scripts/agent-preflight.py` (+test) | same | FENIX_ env, repo-root resolution, fenix summary |
| `scripts/check-maintainability.py` | same | fenix Go/RN/BFF path classifiers + budgets |
| `scripts/check-config-secrets.sh` | same | fenix config layout |
| `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` | `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` | adapt as portable workflow source; keep local wrappers for repo-specific authority |
| `scripts/gemma_local.py`, `scripts/gemma_local_test.py` | same paths | copy + FENIX_ env + default model `gemma4:26b-a4b-it-qat` (same as DubBridge; fallback `gemma4:12b-it-qat`) |
| `scripts/delegate-low-rri.py`, `scripts/delegate_low_rri_test.py` | same paths | copy + FENIX_ env + fenix path allowlist |
| `scripts/gemma-code-review.py`, `scripts/gemma_code_review_test.py` | same paths | copy + FENIX_ env |
| `scripts/check-review-budget.py`, `scripts/check_review_budget_test.py` | same paths | copy + imports fenix `check-maintainability.py` |
| `scripts/adjudicator-packet.py`, `scripts/adjudicator_packet_test.py` | same paths | copy + policy ref → fenix HITL_AUTONOMY_POLICY |
| `scripts/gemma-push-review.py`, `scripts/gemma_push_review_test.py` | same paths | copy + FENIX_ env + fenix CI job names |
| `scripts/push_review_commit.py`, `scripts/gemma_push_ops_test.py` | same paths | copy + FENIX_ attribution |
| `scripts/gemma-audit-report.py`, `scripts/gemma_audit_report_test.py` | same paths | copy + FENIX_ env |
| — | `docs/policies/LOCAL_MODEL_POLICY.md` | **New**: fenix local-model policy (eligible bands, default model, budgets, audit contract) |
