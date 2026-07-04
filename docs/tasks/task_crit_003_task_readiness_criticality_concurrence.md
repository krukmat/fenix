---
doc_type: task
id: CRIT-003
title: "Task-readiness criticality concurrence"
status: done
phase: implementation
week: 2026-W27
tags: [criticality, peer-review, workflow, python, testing]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: [CRIT-005]
files_affected:
  - scripts/peer-workflow-review.py
  - scripts/peer_workflow_review_test.py
criticality: standard
criticality_basis: "Workflow-gate prompt and test changes only; no auth boundary, migration, or post-code-review model-mix behavior change."
created: 2026-07-04
completed: 2026-07-04
---

# Task CRIT-003: Task-readiness criticality concurrence

**Plan**: [Critical-Task Classification and Local/Cross-Agent Reviewer Mix](../plans/critical_task_reviewer_mix_plan.md#rollout)

## Scope

Teach the task-readiness peer-review gate to evaluate the declared task
`criticality` label. This task updates the readiness-review instructions and
tests so the reviewer must explicitly concur with or dispute the declared
label as part of the readiness verdict.

Changes in scope:

- Update `scripts/peer-workflow-review.py` review instructions for
  `task-readiness` so the reviewer is told to assess the declared
  `criticality` label and its stated basis.
- Keep the dispute semantics advisory: a disagreement is returned as a
  reviewer finding for human resolution, not as an automatic relabel.
- Add or update tests in `scripts/peer_workflow_review_test.py` to cover the
  new readiness-review instruction path.

## Out of scope

- Any new CLI flag or runtime behavior for `post-code-review` (CRIT-004).
- Any local-qwen advisory reviewer path.
- Any change to RRI scoring, task-file schema, or HITL policy text.

## Task Card

Task: CRIT-003
Task file: docs/tasks/task_crit_003_task_readiness_criticality_concurrence.md
Plan file: docs/plans/critical_task_reviewer_mix_plan.md
Summary: Update the task-readiness peer-review instructions so reviewers must concur with or dispute the declared `criticality` label and its basis. Add tests for the new readiness-review contract without changing post-code-review behavior.
Code affected: scripts/peer-workflow-review.py, scripts/peer_workflow_review_test.py
Criticality: standard
Criticality basis: Workflow-gate prompt and test changes only; no auth boundary, migration, or post-code-review reviewer-mix behavior change.
Effort/reasoning: Medium - small code surface, but the wording must be precise because it changes a fail-closed workflow gate and its tests.
Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6
Estimated tokens: ~2400
Pseudocode: If mode is task-readiness, append explicit criticality-concurrence instructions to the reviewer prompt; require the reviewer to assess the declared label and basis; keep disputes as findings rather than automatic label rewrites; extend tests to assert the readiness prompt and verdict expectations.

## Acceptance criteria

- `task-readiness` reviewer instructions explicitly tell the reviewer to assess
  the declared `criticality` label and `criticality_basis`.
- The documented reviewer behavior distinguishes concurrence from dispute and
  keeps a dispute as a recorded finding for human resolution.
- `post-code-review` behavior is unchanged in this task.
- Tests cover the new readiness-review instruction path and continue to pass.

## High-Level Pseudocode

```
build_prompt(mode, packet):
  prompt = base_review_instructions
  if mode == "task-readiness":
    prompt += readiness_criticality_instruction
  append task, plan, and task-card material
  return prompt

readiness_criticality_instruction:
  tell reviewer to inspect declared criticality label and basis
  tell reviewer to concur when justified
  tell reviewer to dispute through findings when unjustified
  tell reviewer not to silently relabel the task

tests:
  assert readiness prompt includes criticality-concurrence language
  assert post-code-review prompt/path remains unchanged
```

## RRI

| Variable | Score | Evidence | Confidence |
|---|---|---|---|
| C cyclomatic | 1 | small prompt-assembly branch | High |
| F files | 1 | 2 files | High |
| D domain | 1 | workflow script, no product path | High |
| T coverage | 1 | existing unit test suite for the script | High |
| A ambiguity | 1 | wording must distinguish dispute from blocking behavior | Medium |
| K coupling | 1 | affects readiness-review contract only | Medium |
| P impact | 2 | governance gate output can influence human approval flow | Medium |
| X context | 2 | plan, script, tests | Medium |

Final RRI: 17 -> Low band (0-25) -> executable without full approval packet,
but still requires task-card presentation per repo reporting rules.
