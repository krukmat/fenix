---
doc_type: plan
id: FENIX-AI-WORKFLOW-W6
title: Cloud-local interaction optimization
status: approved_plan_preimplementation_drafts
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
| [W6.4](../tasks/task_fenix_ai_workflow_w6_4.md) | ChatGPT | W6.2, W6.3 | 90% | native opt-in observation hooks staged; actual exchange transport and certification pending |
| [W6.5](../tasks/task_fenix_ai_workflow_w6_5.md) | ChatGPT | W6.2 | 90% | advisory paired-comparison candidate tested; no policy activation |
| [W6.6](../tasks/task_fenix_ai_workflow_w6_6.md) | ChatGPT | W6.2, W6.4 | 93% | native shadow events and CLI aggregate staged; verified supervisor closure pending |
| [W6.7](../tasks/task_fenix_ai_workflow_w6_7.md) | ChatGPT | W6.3, W6.4, W6.5, W6.6 | 92% | synthetic and extracted native hook tests PASS; full checkout regression pending |
| [W6.8](../tasks/task_fenix_ai_workflow_w6_8.md) | Local agent | W6.7 | 85% | draft; not executed |
| [W6.9](../tasks/task_fenix_ai_workflow_w6_9.md) | ChatGPT | W6.8 | 95% | draft; not executed |

## Acceptance targets (hypotheses, not results)

- At least 30% fewer measured input tokens per accepted task (cloud and local reported separately); at least 25% lower paired end-to-end latency; at least 50% relative reduction of observed fallback rate. Report paired sample size, coverage, distribution and uncertainty. A target not measurable is HOLD, not PASS.
- Zero regression on critical defect detection and deterministic scope/approval checks; no increase in invalid acceptance or skipped required reviews. Stop adoption on any critical miss.
- Baseline and candidate use identical frozen case/source, expected checks, model versions and review requirements. Include all failed attempts and timeouts in total costs, not only successful calls. Separate provider caching and warm/cold model effects.
- No new permanent services. Default workflow remains unchanged unless an explicitly approved and independently reviewed adoption decision changes it.

## Current checkpoint

Pinned audit source: 0c0ba347dd03f48b874d285c521a368c644adb96. W6.1 source analysis recorded; W6.2–W6.7 design requirements prepared in the [exchange draft](../ai-workflow/cloud_local_interaction_draft.md). Local session on 2026-11-24 reported RRI scores W6.1–W6.7 = 11, 28, 35, 45, 33, 25, 30 (developer-scored; gitignored evidence not inspected in cloud), and Codex readiness needs_changes for W6.1 and W6.2. W6.3–W6.7 were not run. The W6.1 source audit has been performed, but the task is not closed; the revised record requires genuine readiness PASS. W6.2 remains blocked by W6.1 and human resolution of the criticality dispute (proposed critical). No implementation, code-review certification, local Ollama inference or benchmark has run. W6.8/W6.9 remain pending. Next: rerun W6.1 readiness and close its audit; resolve W6.2 criticality explicitly, then rerun readiness with the linked wire contract and obtain any required task-specific approval. Continue W6.3/W6.5 → W6.4 → W6.6 → W6.7 per dependencies. Missing executable or authentication is a blocker, not a PASS.

## Cloud-only execution checkpoint — 2026-11-24

Owner directed: omit peer readiness in cloud. Candidate preparation and isolated deterministic verification may proceed here; do not claim certified readiness, change the active policy, or activate the candidate. Local gate evidence and the W6.2 criticality dispute remain outstanding for integration. A standalone W6.2 Python prototype was prepared in an isolated cloud workspace and passed 13/13 unittest cases; its source is not yet in this repository. This is prototype evidence only, not task closure or full repository QA. No local inference, provider calls, or active workflow changes were performed.

## Candidate source checkpoint — 2026-11-24

W6.2–W6.4 have standalone candidate modules in scripts/ with 33/33 isolated Python unittest methods passing (13 wire, 11 context, 9 adapter). Exact tested bytes were confirmed by the Git blob SHAs in [candidate evidence](../ai-workflow/w6_candidate_evidence.md). No existing CLI or default policy was changed. The adapter accepts trusted observed coverage, not inferred model claims; actual provider delivery instrumentation is still required. This supersedes the previous prototype-not-published status, not the recorded independent non-pass evidence. W6.1 closure, W6.2 human criticality dispute, actual independent review, applicable repository QA and activation remain unresolved. W6.5–W6.7 pending.

## W6.5–W6.7 candidate checkpoint — 2026-11-24

Added isolated advisory DSS, attempt telemetry, and synthetic pipeline integration tests. Combined suite: 40/40 PASS, verified exact source blobs. No current model selection or peer gate changes. No live transport hooks or trusted supervisor closure records yet; end-to-end latency, accepted-task tokens and fallback rate remain unknown. W6.8 cannot start an adoption-grade benchmark until the candidate is actually instrumented and applicable governance/QA is reconciled. See [candidate evidence](../ai-workflow/w6_candidate_evidence.md).

## Native shadow integration checkpoint — 2026-11-24

Owner approved W6.4a/b, W6.6a/b and W6.7a. Opt-in observation hooks are now staged in peer-workflow-review.py, delegate-low-rri.py and workflow_audit_aggregate.py; no model routing, verdict, fallback condition, or default input changes. FENIX_W6_SHADOW=1 and a supervisor-supplied FENIX_W6_RUN_ID enable per-attempt observations, including advisory retries; without opt-in no shadow records are written. The new aggregator option --w6-shadow-dir logs/workflow-shadow includes observations separately from legacy totals. Legacy native calls do not establish W6 receipt delivery or a frozen snapshot; never convert observations into accepted-task evidence. The strict candidate exchange path remains disconnected from actual transport. Tests: 60/60 PASS in isolated cloud with exact candidate files and five verified extracted native functions, provider and IO boundaries mocked. Full checkout regression, live provider compatibility, independent gate resolution and W6.8 benchmark remain pending. No new local inference, CI, branch or active default change. See [candidate evidence](../ai-workflow/w6_candidate_evidence.md).
