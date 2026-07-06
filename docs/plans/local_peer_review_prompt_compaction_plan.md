---
doc_type: plan
id: local-peer-review-prompt-compaction-plan
title: "Local Peer Review Prompt Compaction"
status: completed
created: 2026-07-06
updated: 2026-07-06
tags: [workflow, peer-review, local-model, prompt-budget, remediation]
---

# Local Peer Review Prompt Compaction

## Purpose

Restore reliable local peer-review execution in `scripts/peer-workflow-review.py`
by replacing naive section truncation with deterministic prompt compaction that
preserves review-critical material and enforces a total prompt budget for local
fallback and advisory reviewer paths.

## Scope

- Define a deterministic local-review packet compaction strategy.
- Preserve review-critical inputs in priority order: task contract, acceptance
  criteria, criticality declaration when present, verification evidence, and
  diff content.
- Prevent long narrative plan/task history from dominating the local-model
  prompt.
- Add targeted tests that assert what local reviewers receive.

## Out of Scope

- Changing primary Claude/Codex reviewer behavior.
- Altering artifact schema beyond what is needed to support prompt-compaction
  verification.
- Reworking unrelated push-review or CI local-model flows.

## Problem Statement

The current local fallback/advisory path can fail or degrade when the task/plan
payload contains long narrative history. A head-only truncation of `task` and
`plan` is insufficient because it can drop the most recent operational context,
does not budget the total prompt, and leaves other sections unconstrained.

## Proposed Direction

1. Extract or preserve high-signal sections first, rather than taking the first
   N characters of the raw task/plan body.
2. Apply a total local-review prompt budget with per-section priorities.
3. Keep `diff` and `verification_log` reviewable while still respecting the
   aggregate budget.
4. Cover the compaction behavior with tests that inspect the packet handed to
   `gemma_local.build_chat_payload`.

## Verification Plan

- `python3 -m unittest scripts.peer_workflow_review_test`
- `python3 -m py_compile scripts/peer-workflow-review.py scripts/peer_workflow_review_test.py`
- `git diff --check`

## Outcome

Implemented on 2026-07-06 via `PAW-F17`. The local fallback and advisory
reviewer paths now compact task/plan content by priority, enforce a total local
prompt budget, preserve diff and verification evidence under that budget, and
ship targeted tests that assert the prompt passed to the local reviewer payload
builder.
