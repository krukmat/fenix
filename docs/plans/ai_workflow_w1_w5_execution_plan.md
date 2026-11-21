---
doc_type: plan
id: FENIX-AI-WORKFLOW-W1-W5
status: active_staged_integration
scope: development-agent-workflow-only
updated: 2026-11-21
---

# Fénix AI-assisted development workflow W1–W5

This plan addresses the **development agent workflow only**, not product runtime performance. The canonical reconciliation is [AI Workflow W1–W5 Reconciliation](../audits/ai_workflow_w1_w5_reconciliation.md).

## Working rules

- Existing main branch only; no new CI jobs, branches or routine local agent use.
- ChatGPT owns non-local tasks. W4.3 and W5.4 run once, together, at the end on the Mac.
- Keep active RRI/HITL and both mandatory peer gates; no review verdict may be inferred from a staging unittest.
- Moderate-band local delegation is **disabled** until an explicit approved policy change is fully validated.
- Nimble is an *optional advisory*, not an automatic extra reviewer or an approval gate.
- Do not replace an established model on theoretical speed/cost grounds. Compare task-correctness and end-to-end efficiency.
- `DONE_ANALYSIS` means source-backed analysis/documentation only; `REPO_INTEGRATED` means committed; `REPO_VERIFIED` requires relevant QA and independent peer artifacts; `LOCAL_MEASURED` requires Mac data.
- CI and peer gates are not equivalent. Skipping new CI creation does not waive independent review or required existing QA.
- Publishing this ledger does not certify the implementations.

## Execution order

| Block | Tasks | Owner | Gate |
|---|---|---|---|
| B1: baseline/audits | W1.1, W2.1, W3.1, W4.1, W4.2, W5.1, W5.2, W5.3 | ChatGPT | Record evidence |
| B2: review correctness | W3.4, W3.3 | ChatGPT | Regression tests, independent review |
| B3: model routing | W1.2, W1.3, W1.4 | ChatGPT | Current low-band policy |
| B4: governance + advisory | W2.2, W2.3, W2.4, W3.2 | ChatGPT | Explicit governance change before activation |
| B5: final Mac-only run | W4.3, W5.4 | Local | Frozen corpus, measured telemetry |
| B6: measured conclusions | W4.4, W5.5 | ChatGPT | Mac evidence, no unmeasured claims |

Dependencies are tracked per task in `docs/tasks/task_fenix_ai_workflow_wN_M.md`. Task IDs are stable; W3.4 may be implemented in multiple reviews but counts once.

## Task confidence

Confidence is a **feasibility forecast**, not certification, quality score or measured probability. It is revised when evidence changes.

| Task | Owner | B | Confidence | Current state |
|---|---|---|---|---|
| W1.1 | ChatGPT | B1 | 99% | documented |
| W1.2 | ChatGPT | B3 | 98% | documented |
| W1.3 | ChatGPT | B3 | 95% | documented |
| W1.4 | ChatGPT | B3 | 90% | integration_pending |
| W2.1 | ChatGPT | B1 | 99% | documented |
| W2.2 | ChatGPT | B4 | 95% | documented |
| W2.3 | ChatGPT | B4 | 90% | integration_pending |
| W2.4 | ChatGPT | B4 | 85% | integration_pending |
| W3.1 | ChatGPT | B1 | 98% | documented |
| W3.2 | ChatGPT | B4 | 90% | documented |
| W3.3 | ChatGPT | B2 | 95% | integration_pending |
| W3.4 | ChatGPT | B2 | 85% | integration_pending |
| W4.1 | ChatGPT | B1 | 98% | documented |
| W4.2 | ChatGPT | B1 | 98% | documented |
| W4.3 | Local | B5 | 85% | local_last |
| W4.4 | ChatGPT | B6 | 95% | awaiting_local_evidence |
| W5.1 | ChatGPT | B1 | 99% | documented |
| W5.2 | ChatGPT | B1 | 95% | documented |
| W5.3 | ChatGPT | B1 | 95% | documented |
| W5.4 | Local | B5 | 90% | local_last |
| W5.5 | ChatGPT | B6 | 95% | awaiting_local_evidence |

## Certification gap

The repository uses `scripts/peer-workflow-review.py` for task readiness and post-code review. The connector can publish files, but cannot invoke the operator's local Claude/Codex/ Ollama executables. Thus source changes can be committed with clearly marked *uncertified* status; do not mark `REPO_VERIFIED` until a genuine gate artifact exists. No silent waiver is permitted.

## Last local handoff

W4.3 + W5.4 are executed together after source preparation. Measure frozen scenarios, actual Ollama tags, cold/warm wall time, eval rate, peak RSS/memory pressure, first-token time, retry/timeout rate, and cloud tokens per accepted task where available. Redact any private prompts or credentials. Then complete W4.4 and W5.5.
