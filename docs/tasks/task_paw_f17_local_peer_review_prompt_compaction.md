---
doc_type: task
id: PAW-F17
title: "Harden local peer-review prompt compaction and budget control"
status: completed
phase: portable-agent-workflow
week: 2026-W28
tags:
  - development
  - workflow
  - peer-review
  - local-model
  - prompt-budget
  - remediation
fr_refs: []
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - .gitignore
  - docs/tasks/.gitignore
  - scripts/peer-workflow-review.py
  - scripts/peer_workflow_review_test.py
  - docs/tasks/task_paw_f17_local_peer_review_prompt_compaction.md
  - docs/plans/local_peer_review_prompt_compaction_plan.md
criticality: standard
criticality_basis: "Workflow-gate hardening in a local fallback/advisory reviewer path; no product auth, migration, or user-data boundary change."
created: 2026-07-06
completed: 2026-07-06
---

# Task PAW-F17

**Plan**: [Local Peer Review Prompt Compaction](../plans/local_peer_review_prompt_compaction_plan.md)

## Task Card

Task: PAW-F17
Task file: docs/tasks/task_paw_f17_local_peer_review_prompt_compaction.md
Plan file: docs/plans/local_peer_review_prompt_compaction_plan.md
Summary: Replace naive local-review section truncation with deterministic prompt compaction and a total local prompt budget so fallback and advisory local reviewers keep the review-critical context without being dominated by long narrative history. Add tests that validate the exact packet handed to the local model path and promote the governing plan/task artifacts so they are Git-trackable.
Code affected: .gitignore, docs/tasks/.gitignore, scripts/peer-workflow-review.py, scripts/peer_workflow_review_test.py, docs/tasks/task_paw_f17_local_peer_review_prompt_compaction.md, docs/plans/local_peer_review_prompt_compaction_plan.md
Criticality: standard
Criticality basis: Workflow-gate hardening in a local fallback/advisory reviewer path; no product auth, migration, or user-data boundary change.
Effort/reasoning: High - the fix must change prompt-shaping behavior in a blocking workflow gate, preserve the right evidence under constrained context, and prove the new contract with focused tests.
Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6
Estimated tokens: ~4200
Pseudocode: Parse the review packet into prioritized sections; extract high-signal task/plan slices instead of raw head truncation; allocate a total character budget across task, plan, task card, verification log, and diff for local-only reviewer paths; build the local prompt from the compacted packet; add tests that assert budgeting and section preservation for fallback and advisory local review.
Peer readiness review approval: reviewer=local-gemma; artifact=logs/peer-workflow-review/task-readiness_codex_by_local-gemma_20260706T055406Z.json; status=PASS

## Acceptance Criteria

1. Local fallback and advisory reviewer paths use a deterministic packet
   compaction strategy rather than a raw head-only truncation.
2. The compaction strategy preserves review-critical context in priority order,
   including task contract, acceptance criteria, and diff evidence.
3. The local-only packet respects an aggregate prompt budget instead of leaving
   total prompt size unconstrained.
4. Primary Claude/Codex reviewer behavior remains unchanged.
5. Tests assert the packet or prompt content passed to local review payload
   construction for both fallback and advisory paths.

## Scope

- **In**: local fallback prompt shaping, advisory local prompt shaping, test
  coverage, and workflow-doc synchronization for this remediation task.
- **Out**: no change to primary reviewer selection, no new artifact gate, no CI
  redesign.

## Risks

- Over-aggressive compaction can hide findings-worthy context from the local
  reviewer.
- Under-constrained compaction can leave the local path susceptible to the same
  context blow-up that caused the regression.

## High-Level Pseudocode

```text
build_local_review_packet(packet):
  normalize known sections
  extract high-signal slices from task and plan
  allocate a total local prompt budget by section priority
  truncate or summarize lowest-priority narrative content last
  return compacted packet

invoke_local_fallback_reviewer():
  prompt = build_prompt(mode, build_local_review_packet(packet))
  send prompt to local model

invoke_advisory_qwen_reviewer():
  prompt = build_prompt(mode, build_local_review_packet(packet))
  send prompt to local model

tests:
  assert compacted packet preserves acceptance criteria and diff
  assert total local prompt stays within configured budget
  assert primary reviewer path still uses the original packet
```

## RRI

| Variable | Score | Evidence | Confidence |
|---|---|---|---|
| C cyclomatic | 1 | `rri.py --auto-cc` fallback; `radon` unavailable, low-confidence score used | Low |
| F files | 1 | 2 touched code files plus task/plan docs | High |
| D domain | 1 | workflow script and tests only | High |
| T coverage | 2 | existing tests help, but new prompt-budget assertions are needed | High |
| A ambiguity | 2 | preservation priorities and budget semantics must be explicit | High |
| K coupling | 2 | affects local fallback and advisory reviewer prompt shaping plus tests | High |
| P impact | 3 | changes behavior in a blocking workflow gate's local-review path | High |
| X context | 2 | requires plan, script, tests, and local reviewer contract context | High |

Final RRI: 33 -> Moderate band (26-40) -> present task card and wait for
explicit approval before implementation.

## Completion

Result: Replaced head-only local-review truncation with deterministic packet
compaction that prioritizes high-signal task/plan sections, enforces a total
local prompt budget, and preserves diff and verification evidence for local
fallback/advisory reviewers. Added targeted tests that capture the prompt sent
to `gemma_local.build_chat_payload`, and unignored the new canonical plan/task
artifacts so they remain trackable.

Verification:
- `python3 -m py_compile scripts/peer-workflow-review.py scripts/peer_workflow_review_test.py`
- `python3 -m unittest scripts.peer_workflow_review_test`
- `git diff --check`

Peer code review approval: reviewer=local-gemma; artifact=logs/peer-workflow-review/post-code-review_codex_by_local-gemma_20260706T060659Z.json; status=PASS

Files affected:
- `.gitignore`
- `docs/tasks/.gitignore`
- `scripts/peer-workflow-review.py`
- `scripts/peer_workflow_review_test.py`
- `docs/plans/local_peer_review_prompt_compaction_plan.md`
- `docs/tasks/task_paw_f17_local_peer_review_prompt_compaction.md`

Effort/reasoning: High - the task changed behavior in a blocking workflow gate,
needed a budgeted prompt-shaping algorithm, and required isolated peer review so
unrelated working-tree changes did not contaminate the diff under review.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Tokens: ~1900
