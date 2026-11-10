---
doc_type: task
id: LOCAL-PRIMARY-PEER-REVIEW
title: "Prioritize assigned local peer reviewers"
status: completed
phase: portable-agent-workflow
week: 2026-W46
tags: [workflow, peer-review, local-models]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: []
files_affected: [scripts/peer-workflow-review.py, scripts/peer_workflow_review_test.py, docs/playbooks/AGENT_WORKFLOW_GUIDE.md, docs/policies/LOCAL_MODEL_POLICY.md, docs/policies/HITL_AUTONOMY_POLICY.md, docs/plans/portable_agent_workflow_port_plan.md]
criticality: critical
criticality_basis: "Changes the blocking independent review gate for security-sensitive work."
created: 2026-11-10
completed: 2026-11-10
---

# Prioritize assigned local peer reviewers

Plan: [Portable Agent Workflow](../plans/portable_agent_workflow_port_plan.md).

## Task Card

Task: LOCAL-PRIMARY-PEER-REVIEW
Task file: docs/tasks/task_local_primary_peer_review.md
Plan file: docs/plans/portable_agent_workflow_port_plan.md
Summary: Route independent peer review to the assigned local models first and synchronize the governing workflow and policies. Preserve fail-closed verdicts, independence, auditable attempts and human approval requirements.
Code affected: scripts/peer-workflow-review.py, scripts/peer_workflow_review_test.py, workflow guide, local model and HITL policies, portable workflow plan.
Criticality: critical
Criticality basis: Changes the blocking independent review gate for security-sensitive work.
Effort/reasoning: High - Reviewer routing, failure semantics, and governance must remain consistent.
Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6
Estimated tokens: ~9000
Pseudocode: Resolve criticality from the task for readiness and from the explicit criticality input for code review; select FENIX_REVIEW_CRITICAL_MODEL (default qwen3.6:35b-a3b) for critical tasks and FENIX_REVIEW_MODEL (default gemma4:26b-a4b-it-qat) otherwise; require a reviewer independent from the author; run the local reviewer with the configured timeout and record the attempt; only on reviewer unavailability or timeout try an independent external reviewer, never on a substantive or invalid verdict; preserve the blocking verdict and human approvals; test routing, identity, fallback and failure handling.
Peer readiness review approval: reviewer=local-gemma; artifact=logs/peer-workflow-review/task-readiness_codex_by_local-gemma_20261110T101242Z.json; status=PASS

## Acceptance

- Assigned local reviewer is attempted first; critical review uses the assigned critical model.
- A reviewer must be independent of the author; unavailability or invalid output cannot silently pass.
- The gate blocks local-provider and unknown authors until it can verify that author and reviewer differ; the current interface has no local author model identity.
- Critical Qwen review replaces the former non-blocking Qwen advisory attempt and directly governs the verdict.
- External fallback is permitted only after local reviewer unavailability or timeout; substantive failures and invalid verdicts do not trigger fallback. Preserve human approval requirements.
- Artifacts identify reviewer, model, attempts and verdict accurately.
- Preserve unrelated working-tree changes and do not activate routes, CI, deployments or merge.

FR/UC references are not applicable to this workflow governance task.

## Outcome

Assigned local models now perform the blocking primary review. The gate fails
closed for unknown/local authors, invalid verdicts, missing criticality, empty
or oversized diffs, and preserves independent external fallback only for local
timeout or unavailability. Critical Qwen review replaces the advisory attempt.
`make qa-peer-workflow-review` passed 60 tests; `git diff --check` passed.
Peer code review approval: reviewer=local-qwen; artifact=logs/peer-workflow-review/post-code-review_codex_by_local-qwen_20261110T101903Z.json; status=PASS.

## Publication checkpoint

User requested commit and push on 2026-11-10. Local branch was fast-forwarded
to the existing remote HEAD 6772905649d8390b4943273f4e52416696194087;
the remote commits did not overlap this task's changed paths.
The 60 peer-workflow tests and task completion evidence check passed.
The initial publication attempt was blocked by the repository documentation gate: legacy plan
frontmatter failures, ADR status/index inconsistencies and dangling ADR-036
references outside this change. Logs: /tmp/fenix-publish-docs-qa.log and
/tmp/fenix-publish-consistency-qa.log.

Owner subsequently explicitly instructed publication and consolidation with the
other agents' work. The workflow commit was rebased without conflicts onto
4af514f (10 additional remote documentation commits after 67729056), preserving
their B2.A2 acceptance, B2.C traceability and validation handoff changes. The
pre-existing local .gitignore cache exclusion is preserved outside the commit.
The publication authorization does not turn the global documentation failures
into a PASS or certify the still-pending integration QA/security reviews.
