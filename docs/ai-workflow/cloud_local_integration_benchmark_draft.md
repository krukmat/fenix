---
doc_type: proposal
id: FENIX-AI-W6-INTEGRATION-DRAFT
title: W6 adapter, telemetry and benchmark implementation design
status: draft_pending_independent_readiness
created: 2026-11-24
---

# W6.4–W6.9 — Integration design and final evidence protocol

Scope: development workflow only. Refines the [wire draft](cloud_local_wire_contract_v1_draft.md) and [W6 plan](../plans/ai_workflow_w6_cloud_local_execution_plan.md). This is documentation, not a change to active behavior. Source inspection pinned to 113491874d42591f854eaddb666f032f3625caea. RRI and independent peer gates not run.

## W6.4 — Adapter boundaries and compatibility

| Existing entry point | Shadow instrumentation point | Invariant |
|---|---|---|
| peer-workflow-review.py build_packet/build_prompt | Capture canonical packet before invocation; separately hash the actual redacted payload sent | Hashing constructed context is not proof of delivery |
| invoke_reviewer/_run | Per-provider actual invocation and terminal outcome | Preserve timeout, exit status and native verdict semantics |
| codex_review_command | Bind base and observed workspace snapshot; record native scope | Do not assert explicit task/plan/QA sections were sent with --base |
| invoke_local_fallback_reviewer | Record primary failure and distinct fallback attempt | No fallback on arbitrary findings; existing should_attempt_fallback governs |
| delegate-low-rri.py main | Wrap native result after parser/scope validation | Native tagged protocol and existing path checks unchanged |
| delegate-low-rri.py exception boundary | Record terminal error if correlation exists | Failed attempt never becomes accepted; preserve exit codes 124/2/1 |
| workflow_audit_aggregate.py aggregate | Read additional versioned terminal events without changing legacy totals | Correlated metrics separate from historical aggregates |

The candidate starts in shadow mode, meaning observations of the same invocation, not duplicate calls to models. Explicitly separate an A/B candidate run from shadow instrumentation. No additional reviewer invocation merely to populate telemetry. Review output is still independently produced. If candidate telemetry fails, mark the candidate trace incomplete; do not replace or fabricate a native verdict. If the new exchange path is eventually active, missing mandatory receipts block its acceptance; do not silently retry as an uninstrumented accepted candidate.

For actual context equivalence, compare immutable scope/criteria snapshots and delivery coverage. The initial experiment does not replace native Codex review with a different invocation or weaken readiness/post-code requirements. CLI capability probes and usage extraction depend on the actual installed version, not an assumed flag or provider API. ChatGPT remote usage is unknown unless an authorized export reports it.

## W6.5 — DSS calibration without policy activation

Use a lane-specific, version-specific profile: model ID/tag, prompt contract version, hardware class, context range, verified completions, total attempts, measured elapsed time and available token counts. Compare only like-for-like eligible cases. Represent uncertainty using sample count and a binomial Wilson interval for observed acceptance; report median paired time difference and the raw pairs. Do not combine time, cost and reliability with arbitrary weights. Return one of UNKNOWN, CANDIDATE_PREFERRED, INCUMBENT_PREFERRED, INCONCLUSIVE, with evidence references. UNKNOWN when coverage, comparable samples or required QA is absent. This is advice, not an execution decision; active mandatory low-RRI attempt, protected boundaries, and RRI 26+ prohibition remain unchanged. Any policy modification needs a separate approved and independently reviewed decision.

## W6.6 — Correlated events and coverage

An attempt event has run_id, attempt_id, stage, provider, model, transport, source/snapshot digest, actual receipt digest or null, start/end (monotonic elapsed plus wall timestamp for correlation), terminal status, fallback reason, input/output/cached usage, measurement source, and evidence references. Event identity includes task_id, run_id, attempt_id and stage. Canonical duplicate records count once; conflicting duplicates invalidate the affected trace. A supervisor closure event is separate, and references current native QA and peer artifacts. It must not be generated from an inference verdict alone.

Report these counters separately:
- Frozen case count, observed runs, terminal attempts, complete traces, invalid/incomplete traces, accepted runs.
- Provider-reported input/output tokens by cloud/local and provider; coverage of attempted calls. Estimated tokens and transport bytes in separate columns.
- All-attempt elapsed duration per run from first action to terminal supervisor state; do not sum overlapping stage durations as wall-clock time.
- All measured attempted input tokens / accepted tasks only when the compared arm has complete relevant token coverage; otherwise report partial observed totals and UNKNOWN for the complete metric. Zero accepted tasks is undefined, not zero.
- Fallback rate: peer review instances with any eligible fallback / peer review instances with known attempt history. Keep local delegation escalation rate separate.
- No raw prompts, patches, private paths or credentials in exported metrics. Legacy logs without correlation remain historical observations.

## W6.7 — Verification order

1. W6.2 standalone validator: wire vectors 01–05; canonicalization edge cases including Unicode and duplicate keys.
2. W6.3 resolver: frozen manifest, dirty tree, symlink/traversal, deletion, digest mismatch and receiver capabilities.
3. W6.4 adapters: mocked subprocess and Ollama transport, native parser equivalence, failure and timeout mapping, actual coverage, unchanged reviewer selection and fallback.
4. W6.5 advisory: insufficient data, mismatched lane/version, RRI 26+ and protected paths; no authorization side effects.
5. W6.6 aggregation: identical/conflicting event replay, incomplete traces, null tokens, zero acceptances, elapsed time vs overlapping stage duration, distinct fallback denominators.
6. Run the existing repository suites for every touched component, independent task-readiness and post-code gates as applicable, and document actual commands/results. Tests and mocks do not satisfy real independent review. No new CI.

## W6.8 — Paired benchmark, last local block

Prerequisite: reviewed implementation and relevant QA; no active change before adoption. Reuse eligible W4 frozen cases and any existing W4/W5 measurements only when source, criteria, model/provider versions, cache/residency and trace definitions match. Historic unmatched observations are descriptive, not A/B controls.

- Select balanced frozen cases across existing lanes, with at least one relevant deliberate critical-defect review fixture per represented review lane. A smoke pilot establishes measurement integrity, not statistical confidence.
- Arm A: existing workflow. Arm B: opt-in W6 exchange path. Same underlying model, context requirements, reviewer identity, acceptance checks, timeouts, and policy. Alternate A/B order by case and report cold/warm separately. Do not run both arms concurrently on the Mac.
- Freeze before inference: source snapshot, task and criteria, model IDs, CLI versions, configuration, dataset and expected checks. Keep fixture data synthetic or authorized and redacted.
- Record per-run status, acceptance evidence, known defect outcomes, model calls, latency including retries, measured usage where available, and memory pressure. Stop candidate runs on critical misses, policy regression or unsafe memory pressure; report partial coverage.
- No new model downloads or infrastructure. No benchmark of unapproved high-risk local delegation.
- For a small pilot, report exact pair counts and ranges; do not assert statistically proven p95 or extrapolated annual savings.

## W6.9 — Decision rules

| Evidence | Decision |
|---|---|
| Any observed critical defect missed, unauthorized action, skipped required gate, or false acceptance | NO_GO candidate |
| Missing gate, incomparable arms, unknown required metric, insufficient cases for the claimed improvement | HOLD |
| Valid paired evidence, no observed critical regression and agreed efficiency thresholds met | GO for scoped pilot, subject to existing approvals |
| No reliability regression but targets missed | Retain baseline; isolate beneficial components and re-evaluate |

Targets from W6 plan: 30% input-token reduction, 25% elapsed reduction, 50% relative peer fallback reduction, each against its matched baseline. Zero baseline fallback makes relative reduction undefined; report absolute rates and HOLD for that target. Distinguish observed zero misses from proof of zero future risk.

## Current state and blocker

W6.1 source audit recorded; W6.2–W6.7 design documented, not implemented or certified. W6.8 protocol prepared, not executed; W6.9 requires actual evidence. A direct environment check found no claude, codex or ollama executable on PATH and no Fénix checkout. There is no independent gate artifact. Do not interpret plan approval as a waiver. Implementation remains blocked until authentic readiness can be obtained through an authorized environment. No new branches, CI, model invocations, or changes to active policy.
