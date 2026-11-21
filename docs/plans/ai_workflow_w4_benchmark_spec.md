---
doc_type: plan
id: FENIX-AI-WORKFLOW-BENCHMARK-W4
status: corpus_indexed_not_frozen
updated: 2026-11-21
---

# W4 — Model benchmark and cost: test contract

The target is the **agent-assisted development workflow**, not the Fénix product. All 20 existing source task paths were verified on the baseline main revision `0912e0137f50f3bec20fd840b4abb03c9a22a5d5` before the W1–W5 documentation rollout. These are potential historical replays, **not demonstrated current defects**. Historical code snapshots and task packet hashes are still to be frozen.

## Candidate corpus

| ID | Task path under `docs/tasks/` | Evaluation lane |
|---|---|---|
| 01 | `task_agentdef_bootstrap_docs_sync.md` | documentation |
| 02 | `task_paw_f14_peer_review_artifact_presentation.md` | documentation |
| 03 | `task_paw_f17_local_peer_review_prompt_compaction.md` | analysis |
| 04 | `task_fenix_maintainability_gate_wiring_alignment.md` | governance (cloud-only) |
| 05 | `task_extval_android_recovery_plan.md` | planning |
| 06 | `task_agentdef_bootstrap_impl_b1_characterization.md` | tests |
| 07 | `task_agentdef_bootstrap_impl_b2_lookup_fix.md` | small fix |
| 08 | `task_mobile_approvals_response_shape_crash.md` | small fix |
| 09 | `task_mobile_bff_remediation_5.md` | small fix |
| 10 | `task_extval_o4_o5a_salesbrief_graceful_degradation.md` | small fix |
| 11 | `task_extval_o5b_llm_usage_model_token_attribution.md` | instrumentation |
| 12 | `task_extval_o7_signals_403_workspace_owner.md` | authorization-sensitive (cloud-only) |
| 13 | `task_fenix_integration_w6a_functional_slice.md` | complex code |
| 14 | `task_fenix_integration_w6b_multi_agent_coordination.md` | complex coordination |
| 15 | `task_fenix_integration_w6c_functional_resilience.md` | resilience |
| 16 | `task_fenix_dubbridge_int_w0.md` | cross-system analysis |
| 17 | `task_fenix_integration_w5c_operational_readiness.md` | readiness verification |
| 18 | `task_fenix_integration_w6d_product_demo_handoff.md` | handoff |
| 19 | `task_fr_071_gap_closure.md` | traceability |
| 20 | `task_fr_202_gap_closure.md` | traceability |

## W4.1 — Freeze protocol

For each eligible task, record source SHA, packet SHA-256, acceptance criteria, allowed paths, expected output contract, deterministic verification commands, historical expected diff/behavior (if demonstrable), risk/criticality, timeout and input token budget. Do not replay fixed bugs on current main without a historical source snapshot. Exclude auth-sensitive and policy-changing cases from unsupervised local runs. No secrets, private prompts or production data in benchmark packets.

## W4.2 — Compare verified task outcomes, not output speed alone

| Metric | Definition |
|---|---|
| Verified completion rate | Successful acceptance cases / attempted cases; failures retained |
| Cloud tokens per accepted task | Actual provider tokens for supervision, primary reviews, fixes and retries divided by verified completions |
| Local wall time and tokens/s | Compare same frozen packets and context budget, minimum 3 repetitions for timing |
| Cold/warm load and memory | Ollama measured load/eval durations and host memory pressure; distinguish cold from merely first call |
| Retry and escalation | Attempts, timeouts and fallbacks per case |
| Review safety | Primary readiness/post-code artifact coverage; zero unauthorized actions |
| Input reproducibility | Frozen source SHA, packet SHA, model tag, flags and deterministic tests recorded |

Use exact installed Ollama tags; do **not** download new models for a benchmark. GPT-OSS, Qwen, Gemma and Nimble are candidates only, and Nimble is tested on structured triage rather than code-generation completion. Compare within lanes, never assume models have interchangeable roles.

A model role is retained/added only when verified completion is non-inferior and the total accepted-task time or paid tokens improves. Do not weaken the mandatory primary peer gates or change the 3-pass local review default before a reproducible defect-recall comparison. p95 for small samples is exploratory.

## W4.3 — Final Mac handoff (deferred by user)

Run this corpus in isolated source copies, not directly in `main`. Capture model availability, resource use, prompts and outputs privately and publish only redacted outcome tables and hashes. Run this together with W5.4 in **one last local work block** after cloud preparation.

## W4.4 — Conditional decision table

For each lane: incumbent, candidate, valid test count, verified completion rate, p50/p95 wall time, cloud tokens/accepted task, local memory, retry/repair rate, reviewer artifact coverage, decision (retain/pilot/reject/insufficient evidence), confidence and outstanding gaps. Conclusions require local measurements.
