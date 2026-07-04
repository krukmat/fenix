---
doc_type: task
id: CRIT-004
title: "Post-code-review reviewer mix for critical tasks"
status: done
phase: implementation
week: 2026-W27
tags: [criticality, peer-review, workflow, python, testing, local-models]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: [CRIT-005]
files_affected:
  - scripts/peer-workflow-review.py
  - scripts/peer_workflow_review_test.py
  - scripts/gemma_local.py
criticality: standard
criticality_basis: "Workflow-gate behavior change with advisory local-model review, but no product auth/data boundary or migration impact."
created: 2026-07-04
completed: 2026-07-04
---

# Task CRIT-004: Post-code-review reviewer mix for critical tasks

**Plan**: [Critical-Task Classification and Local/Cross-Agent Reviewer Mix](../plans/critical_task_reviewer_mix_plan.md#rollout)

## Scope

Implement the critical-only post-code-review reviewer mix. The existing
cross-agent reviewer remains the primary blocking verdict. When
`--criticality=critical` is declared for `post-code-review`, the script must
also run an advisory-only local qwen review and record both attempts in the
artifact without changing the exit code contract.

Changes in scope:

- Add a `--criticality` flag to `post-code-review`.
- Keep the primary cross-agent reviewer fail-closed and exit-code-governing.
- Add an advisory-only local qwen review path for `critical` tasks, including
  one-at-a-time execution, immediate unload, and exactly one retry on timeout
  or runtime/memory failure.
- Record advisory success or advisory-blocked results in the artifact without
  changing the process exit code.
- Add tests for standard vs critical behavior and the retry/degrade path.

## Out of scope

- Any change to task-file schema or HITL policy docs.
- Any change to the standard-task gemma fallback path unrelated to critical
  advisory review.
- Any change to CI push-review scripts or `qa-gemma-review`.

## Task Card

Task: CRIT-004
Task file: docs/tasks/task_crit_004_post_code_review_reviewer_mix.md
Plan file: docs/plans/critical_task_reviewer_mix_plan.md
Summary: Extend `post-code-review` so `critical` tasks keep the existing blocking cross-agent reviewer and also run an advisory-only local qwen review. Add tests for the critical-only path, retry/unload hardening, and non-blocking degradation without changing standard-task behavior.
Code affected: scripts/peer-workflow-review.py, scripts/peer_workflow_review_test.py, scripts/gemma_local.py
Criticality: standard
Criticality basis: Workflow-gate behavior change with advisory local-model review, but no product auth/data boundary or migration impact.
Effort/reasoning: High - this is the core behavioral change in the rollout and it combines CLI contract changes, artifact semantics, local-model lifecycle handling, and failure-path tests.
Recommended model: OpenAI: gpt-5.5 | Anthropic: claude-opus-4-8
Estimated tokens: ~5200
Pseudocode: Parse `--criticality` for post-code-review; always run the primary cross-agent reviewer first and keep its verdict exit-code-governing; if critical, invoke advisory qwen locally after the primary reviewer, unload after every attempt, retry once on timeout/runtime failure, record the advisory attempt outcome in the artifact, and never let the advisory verdict change the process exit code; extend tests for standard-path invariance and critical-path success/failure behavior.

## Acceptance criteria

- `post-code-review` accepts `--criticality standard|critical`.
- `standard` behavior stays unchanged from today's flow.
- `critical` runs the primary reviewer first and then a non-blocking advisory
  local qwen review recorded in the artifact.
- Advisory qwen retries exactly once on timeout or runtime failure and unloads
  the model after every attempt regardless of outcome.
- If both advisory attempts fail, the artifact records a non-blocking
  advisory-blocked result and the overall exit code still depends only on the
  primary verdict.
- Tests cover standard-path invariance, critical-path success, and the forced
  retry/degrade path.

## High-Level Pseudocode

```
parse_args():
  add --criticality to post-code-review with default standard

main():
  run primary reviewer exactly as today
  attempts = [primary attempt]

  if mode is post-code-review and criticality is critical:
    advisory = run_advisory_qwen_with_retry(packet, timeout)
    attempts.append(advisory attempt record)

  write artifact with primary + advisory attempts
  return exit code based only on primary verdict

run_advisory_qwen_with_retry():
  for attempt in [1, 2]:
    invoke local qwen one packet at a time
    unload model immediately after attempt
    if success: return advisory pass/findings record
    if failure is timeout/runtime and first attempt: retry once
  return advisory-blocked record

tests:
  standard post-code-review path unchanged
  critical path appends advisory-local attempt
  forced first failure retries once
  double failure records advisory-blocked without changing primary exit code
```

## RRI

| Variable | Score | Evidence | Confidence |
|---|---|---|---|
| C cyclomatic | 3 | new branching for critical-only advisory path and retry logic | Medium |
| F files | 1 | 2 files | High |
| D domain | 1 | workflow script, no product path | High |
| T coverage | 2 | existing suite helps, but failure-path tests are non-trivial | Medium |
| A ambiguity | 2 | artifact semantics and retry boundaries must match the plan exactly | Medium |
| K coupling | 2 | affects artifact shape, CLI contract, and local-model invocation path | Medium |
| P impact | 3 | changes post-code-review behavior for critical tasks | Medium |
| X context | 2 | plan, script, tests, local-model helper behavior | Medium |

Final RRI: 29 -> Mid band (26-40) -> requires task-card presentation and
explicit approval before implementation.
