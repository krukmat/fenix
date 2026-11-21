---
doc_type: status
id: FENIX-AI-WORKFLOW-W1-W5-STATUS
status: source_published_peer_qa_pending_local_last
updated: 2026-11-21
---

# W1–W5 — Evidence-based execution status

**Repository:** `krukmat/fenix` / `main`. **Scope:** development-agent workflow ONLY. Reconcile against [the canonical plan](../plans/ai_workflow_w1_w5_execution_plan.md) and [the initial repo audit](../audits/ai_workflow_w1_w5_reconciliation.md).

## Progress counters — do not conflate them

| Dimension | Actual status |
|---|---|
| Task cards committed | **21/21** |
| ChatGPT-owned tasks with source/docs/conditional artifacts | **19/19** (some are proposals or partial implementations) |
| Source-backed analysis/design tasks documented | **12/12** |
| Source integration/governance candidates | **5/5 have staged or committed artifacts**, but not full gates; RRI 26–40 remains forbidden |
| Final local performance/model runs | **0/2** (W4.3 and W5.4 only) |
| Final empirical conclusions | **0/2** (W4.4 and W5.5 cannot close before local data) |
| Full workflow/repository QA + independent peer-gate certifications | **0/21**; no fabricated PASS |
| New isolated cloud unit tests on exact repo blobs | **28/28 PASS**; not Mac/CI/full repository tests |
| Codex output parser analogue smoke | **6/6 PASS**; not the full peer adapter test suite |

## Verified source artifacts

| Scope | Paths / evidence |
|---|---|
| Canonical task plan | `docs/plans/ai_workflow_w1_w5_execution_plan.md`, `docs/tasks/task_fenix_ai_workflow_w*.md` |
| Model selection | `scripts/workflow_model_router.py`, `scripts/workflow_model_router_test.py`, opt-in `scripts/delegate-low-rri.py` |
| Review safety | `scripts/peer-workflow-review.py`, updated `scripts/peer_workflow_review_test.py` |
| Workflow telemetry | `scripts/workflow_audit_aggregate.py`, its tests, and selected model/input-token audit extensions to developer/reviewer writers |
| Historical benchmark | `docs/plans/ai_workflow_w4_benchmark_spec.md`, `docs/ai-workflow/benchmark_corpus_manifest.json` (20 historical source blobs) |
| W2 proposal | `docs/ai-workflow/proposals/moderate_delegation_adr_draft.md`, inactive |
| W3 contract | `docs/ai-workflow/review_optimization_contract.md` |
| W5 analysis | `docs/audits/ai_workflow_local_performance_baseline.md` |
| Final Mac preparation | `scripts/workflow_local_probe.py`, tests, `docs/ai-workflow/final_mac_handoff.md` |
| Conditional conclusions | `docs/ai-workflow/conditional_decisions.md` |

### Reproducible isolated test evidence

The cloud runtime reproduced the following GitHub files byte-for-byte and compared Git blob SHA-1 identities before running Python's unittest:

| Module and test file | Verified Git blob SHA |
|---|---|
| `scripts/workflow_model_router.py` | `e07e36deb05db10a2dd68df45d4d8f8b3beb45ce` |
| `scripts/workflow_model_router_test.py` | `2a23f5f4c2789ba27689eef1daefc7f6bd1cf5e3` |
| `scripts/workflow_audit_aggregate.py` | `a1334e9b7bf5d8a7dbe619ffcc9d25a649c8e2cd` |
| `scripts/workflow_audit_aggregate_test.py` | `16444a884cbf9934b898151cde05d373b156a7cc` |
| `scripts/workflow_local_probe.py` | `1c6090d058ff0fc6b972f9a705964301b4ae1c2e` |
| `scripts/workflow_local_probe_test.py` | `a190a2abb570c2043a0ddd72b9a9a723a34e9708` |

Test result: model router **18 PASS**, audit aggregator **5 PASS**, local probe mocks/dry-run **5 PASS**, total **28 PASS**. The Codex parser excerpt additionally passed 6 isolated cases. These results **do not certify** the modified 1,000-line delegates or provider-aware review adapter in their full production context, and are not evidence of Ollama availability.

## Task-by-task status and original execution confidence

Confidence is an estimated likelihood of technical feasibility **conditional on having the required source QA, reviewers and Mac access**; not a PASS grade. Missing gates lower current readiness.

| Task | Status | Execution confidence |
|---|---|---|
| W1.1 | Repo routing audit documented | 99% |
| W1.2 | Role matrix documented | 98% |
| W1.3 | Delegation/fallback contract documented | 95% |
| W1.4 | Opt-in low-RRI routing code committed, integration gates missing | 90% |
| W2.1 | Active RRI/HITL policy audited | 99% |
| W2.2 | Risk-capability eligibility documented | 95% |
| W2.3 | Inactive ADR proposal; active policy not adopted | 90% |
| W2.4 | Non-executing moderate advisory scaffold committed | 85% |
| W3.1 | Peer gate pipeline audited | 98% |
| W3.2 | Optional structured precheck parser committed | 90% |
| W3.3 | Review optimization contract + telemetry; default unchanged | 95% |
| W3.4 | Codex parser fail-closed patch committed; peer/QA pending | 85% |
| W4.1 | 20 historical source blobs indexed; frozen packets missing | 98% |
| W4.2 | KPI and comparison contract versioned | 98% |
| W4.3 | FINAL Mac model benchmark: not run | 85% |
| W4.4 | Conditional conclusions only, awaiting W4.3 | 95% |
| W5.1 | Historical workflow audit references reconciled | 99% |
| W5.2 | Aggregation and fallback analysis versioned | 95% |
| W5.3 | Prompt/model lifecycle analysis + measured-token fields | 95% |
| W5.4 | FINAL Mac latency/RAM/model performance run: not run | 90% |
| W5.5 | Conditional optimization backlog awaiting W5.4 | 95% |

## Remaining gated work

1. **Project-context QA:** execute existing `peer_workflow_review_test.py`, `delegate_low_rri_test.py`, `gemma_code_review_test.py`, `agent_preflight_test.py`, and any applicable pre-push checks. No additional CI added.
2. **Independent peer evidence:** run actual task-readiness and post-code reviews with supported CLIs; record factual artifacts. A missing CLI is BLOCKED, not PASS.
3. **Governance:** maintain active ban on local RRI 26+; decide adoption separately after independent policy review and readiness. No waiver issued.
4. **Last local block:** perform W4.3 and W5.4 together using actual installed Ollama tags and frozen inputs, with redacted evidence.
5. **After evidence:** ChatGPT closes W4.4/W5.5; until then, default model recommendations and performance savings remain hypotheses.

## Operating constraints

No new CI, no new branches, no local agent until the final block, no unsupported install, no Fénix product-runtime changes. All permanent output is in the repository, not an external ZIP.
