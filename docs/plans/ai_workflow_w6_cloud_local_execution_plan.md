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
- Do not skip task-specific RRI, readiness review, required QA, post-code review, or HITL. Autonomous planning authorization is not an ADR-037 waiver. Task estimates are conditional feasibility, not certification.
- No activation of the opt-in exchange path or change in routing defaults until W6.9 and separate required governance approval.

## Task ledger

| Task | Owner | Dependency | Confidence | State |
|---|---|---|---:|---|
| [W6.1](../tasks/task_fenix_ai_workflow_w6_1.md) | ChatGPT | None | 98% | source analysis recorded; needs_changes; revised task pending re-review |
| [W6.2](../tasks/task_fenix_ai_workflow_w6_2.md) | ChatGPT | W6.1 | 95% | needs_changes; human criticality resolution pending |
| [W6.3](../tasks/task_fenix_ai_workflow_w6_3.md) | ChatGPT | W6.2 | 92% | draft; not executed |
| [W6.4](../tasks/task_fenix_ai_workflow_w6_4.md) | ChatGPT | W6.2, W6.3 | 90% | draft; not executed |
| [W6.5](../tasks/task_fenix_ai_workflow_w6_5.md) | ChatGPT | W6.2 | 90% | draft; not executed |
| [W6.6](../tasks/task_fenix_ai_workflow_w6_6.md) | ChatGPT | W6.2, W6.4 | 93% | draft; not executed |
| [W6.7](../tasks/task_fenix_ai_workflow_w6_7.md) | ChatGPT | W6.3, W6.4, W6.5, W6.6 | 92% | draft; not executed |
| [W6.8](../tasks/task_fenix_ai_workflow_w6_8.md) | Local agent | W6.7 | 85% | draft; not executed |
| [W6.9](../tasks/task_fenix_ai_workflow_w6_9.md) | ChatGPT | W6.8 | 95% | draft; not executed |

## Acceptance targets (hypotheses, not results)

- At least 30% fewer measured input tokens per accepted task (cloud and local reported separately); at least 25% lower paired end-to-end latency; at least 50% relative reduction of observed fallback rate. Report paired sample size, coverage, distribution and uncertainty. A target not measurable is HOLD, not PASS.
- Zero regression on critical defect detection and deterministic scope/approval checks; no increase in invalid acceptance or skipped required reviews. Stop adoption on any critical miss.
- Baseline and candidate use identical frozen case/source, expected checks, model versions and review requirements. Include all failed attempts and timeouts in total costs, not only successful calls. Separate provider caching and warm/cold model effects.
- No new permanent services. Default workflow remains unchanged unless an explicitly approved and independently reviewed adoption decision changes it.

## Current checkpoint

Pinned audit source: 0c0ba347dd03f48b874d285c521a368c644adb96. W6.1 source analysis recorded; W6.2–W6.7 design requirements prepared in the [exchange draft](../ai-workflow/cloud_local_interaction_draft.md). Local session on 2026-11-24 reported RRI scores W6.1–W6.7 = 11, 28, 35, 45, 33, 25, 30 (developer-scored; gitignored evidence not inspected in cloud), and Codex readiness needs_changes for W6.1 and W6.2. W6.3–W6.7 were not run. The W6.1 source audit has been performed, but the task is not closed; the revised record requires genuine readiness PASS. W6.2 remains blocked by W6.1 and human resolution of the criticality dispute (proposed critical). No implementation, code-review certification, local Ollama inference or benchmark has run. W6.8/W6.9 remain pending. Next: rerun W6.1 readiness and close its audit; resolve W6.2 criticality explicitly, then rerun readiness with the linked wire contract and obtain any required task-specific approval. Continue W6.3/W6.5 → W6.4 → W6.6 → W6.7 per dependencies. Missing executable or authentication is a blocker, not a PASS.
