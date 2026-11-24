---
doc_type: plan
id: FENIX-AI-WORKFLOW-W6
title: Cloud-local interaction optimization
status: candidate_development_hold_adoption
created: 2026-11-24
scope: development-agent-workflow-only
---

# W6 — Cloud-local interaction optimization

Owner approved the W6 plan and autonomous preparation on 2026-11-24. This is a planning ledger, not a readiness or code-review PASS. Existing active policies remain authoritative. The previous W1–W5 execution plan and empirical evidence remain separate; reconcile outstanding gates before adoption.

## Constraints

- Work on main only. No new CI, model downloads, always-on service, Internet exposure of Ollama, or changes to CRM runtime.
- ChatGPT owns W6.1–W6.7 and W6.9. W6.8 is the last Mac execution block; consolidate eligible remaining W4/W5 checks rather than repeatedly invoking local models.
- For cloud-only W6 candidate development, the owner directed that peer readiness be omitted in the cloud environment (2026-11-24). Continue autonomous candidate preparation and deterministic tests here. Record cloud readiness as DEFERRED, not PASS; prior non-pass evidence remains valid. This instruction is scoped to cloud execution and is not an ADR-037 waiver or a change to the active workflow gates. Before integration/adoption, reconcile actual independent review, outstanding criticality resolution, RRI/HITL, and applicable QA. No local model delegation for routine development. Task estimates are conditional feasibility, not certification.
- No activation of the opt-in exchange path or change in routing defaults until W6.9 and separate required governance approval.

## Task ledger

| Task | Owner | Dependency | Confidence | State |
|---|---|---|---:|---|
| [W6.1](../tasks/task_fenix_ai_workflow_w6_1.md) | ChatGPT | None | 98% | source analysis recorded; needs_changes; revised task pending re-review |
| [W6.2](../tasks/task_fenix_ai_workflow_w6_2.md) | ChatGPT | W6.1 | 95% | candidate source tested; final readiness and criticality resolution pending |
| [W6.3](../tasks/task_fenix_ai_workflow_w6_3.md) | ChatGPT | W6.2 | 92% | candidate resolver tested; integration pending |
| [W6.4](../tasks/task_fenix_ai_workflow_w6_4.md) | ChatGPT | W6.2, W6.3 | 90% | shadow hooks, offline bridge and stable-prefix candidate tested; live pilot pending |
| [W6.5](../tasks/task_fenix_ai_workflow_w6_5.md) | ChatGPT | W6.2 | 90% | advisory paired-comparison candidate tested; no policy activation |
| [W6.6](../tasks/task_fenix_ai_workflow_w6_6.md) | ChatGPT | W6.2, W6.4 | 93% | native shadow events and CLI aggregate staged; verified supervisor closure pending |
| [W6.7](../tasks/task_fenix_ai_workflow_w6_7.md) | ChatGPT | W6.3, W6.4, W6.5, W6.6 | 92% | 26 fresh isolated tests PASS; historical 70 separate; full checkout regression and independent review pending |
| [W6.8](../tasks/task_fenix_ai_workflow_w6_8.md) | Local agent | W6.7 | 85% | HOLD: distinct model-efficiency candidate and applicable gates pending |
| [W6.9](../tasks/task_fenix_ai_workflow_w6_9.md) | ChatGPT | W6.8 | 95% | draft; not executed |

## Acceptance targets (hypotheses, not results)

- At least 30% fewer measured input tokens per accepted task (cloud and local reported separately); at least 25% lower paired end-to-end latency; at least 50% relative reduction of observed fallback rate. Report paired sample size, coverage, distribution and uncertainty. A target not measurable is HOLD, not PASS.
- Zero regression on critical defect detection and deterministic scope/approval checks; no increase in invalid acceptance or skipped required reviews. Stop adoption on any critical miss.
- Baseline and candidate use identical frozen case/source, expected checks, model versions and review requirements. Include all failed attempts and timeouts in total costs, not only successful calls. Separate provider caching and warm/cold model effects.
- No new permanent services. Default workflow remains unchanged unless an explicitly approved and independently reviewed adoption decision changes it.

## Current checkpoint (2026-11-24)

- Pinned audit source: 0c0ba347dd03f48b874d285c521a368c644adb96. Local developer-scored RRI W6.1–W6.7: 11, 28, 35, 45, 33, 25, 30; local artifacts not inspected in cloud. Codex readiness returned needs_changes on W6.1/W6.2; W6.3–W6.7 not run. W6.1 closure, W6.2 human criticality dispute (proposed critical) and independent gates remain pending, not waived.
- Candidate contracts, context, adapters, DSS, telemetry, shadow hooks, offline full/delta transport and native-input equivalence bridge published. Historical isolated suite reported 70/70 PASS; 9/9 bridge tests independently executed on verified source in the current cloud container. Full repository regression and live provider compatibility not run. No new model calls, routing/approval policy changes or active exchange activation.
- Enable observation only for authorized bounded runs: FENIX_W6_SHADOW=1 with a unique supervisor-supplied FENIX_W6_RUN_ID. Aggregate with --w6-shadow-dir logs/workflow-shadow. Legacy observations do not prove delivery or acceptance; trusted supervisor closure remains missing.
- Synthetic transfer: 20,750 to 2,065 bytes (envelope only, excluding packet/cache setup); reconstructed native input unchanged at 14,041 bytes. No inference token savings demonstrated. Codex native post-code --base excluded from explicit-context equivalence bridge. The bridge does not replace current provider invocation.
- W6.8 adoption benchmark HOLD: isolate a real efficiency candidate (validated prompt compaction, supported provider prefix caching or retained context). Measure model-reported input/cached usage, paired latency, complete closure and fallback traces, and independent QA/peer outcomes. Do not spend Mac inference on a transport-only claim. No extra or removed reviewer calls by default.
- Next: continue cloud candidate optimization and deterministic verification, then reconcile required RRI/HITL, W6.1/W6.2 readiness/criticality, full repository regression, independent post-code review, and run the consolidated Mac benchmark only when prerequisites hold.

## References

- [Candidate evidence and chronological results](../ai-workflow/w6_candidate_evidence.md)
- [Exchange draft](../ai-workflow/cloud_local_interaction_draft.md) and [wire contract](../ai-workflow/cloud_local_wire_contract_v1_draft.md)
- [Benchmark contract](../ai-workflow/cloud_local_integration_benchmark_draft.md)
- [Archived chronological plan at previous commit](https://github.com/krukmat/fenix/blob/c5c366364ac1f44ce2fff386731d166614a66701/docs/plans/ai_workflow_w6_cloud_local_execution_plan.md) — historical checkpoints retained; current ledger supersedes stale intermediate statuses. Linked evidence must be included when required by a reviewer; no authorization/evidence omission.

## W6.4d/W6.7b checkpoint — 2026-11-24

- Nine pinned actual task prompts: historical-to-compact average reduction 28.09% of bytes without extra evidence, 6.56% with identical four-document evidence bundle. Stable-prefix candidate keeps full identical blocks and increases shared prefix from 841 to 57,899 bytes, with no reduction in total bytes. Provider token/cache and latency measurements remain unknown. No change to original targets or authority.
- 26/26 fresh isolated tests PASS (9 bridge + 17 new layout/pilot); 70 historical tests not rerun in current environment. Offline preflight and candidate published without modifying native provider invocation. No local inference. Cloud lacks full checkout and authenticated CLIs; final checkout regression and independent/real-usage verification genuinely require the execution environment.
- [Single consolidated validation handoff](../ai-workflow/w6_final_validation_handoff.md). W6.8 remains HOLD until prerequisites. W6.9 remains ChatGPT's responsibility. Do not run a full benchmark on evidence of transport or cache-prefix bytes alone.

## W6.8 local validation correction (2026-11-24)

Owner-reported local tests: 126 workflow PASS, 3 native FAIL of 239; 9/9 preflight section equality. Three native Codex post-code fixtures corrected; two parser regressions added. Production code unchanged. Five extracted parser checks PASS, full native regression rerun required. C counters documented at exact Codex source tag but runtime completeness still UNKNOWN. W6.2 human criticality and actual gates remain pending. Adoption HOLD, no D. See candidate evidence and handoff.
