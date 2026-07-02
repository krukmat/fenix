---
doc_type: task
id: PAW-F14
title: "Clarify peer review approval presentation with explicit artifact path and approval status"
status: completed
phase: F
week: 2026-W27
tags: [paw, devex, workflow, peer-review, docs, reporting]
fr_refs: []
uc_refs: []
blocked_by: [PAW-F4, PAW-F8]
blocks: []
files_affected:
  - AGENTS.md
  - docs/playbooks/AGENT_WORKFLOW_GUIDE.md
  - docs/plans/portable_agent_workflow_port_plan.md
  - docs/tasks/.gitignore
  - docs/tasks/task_paw_f14_peer_review_artifact_presentation.md
created: 2026-07-01
completed: "2026-07-01"
---

# Task PAW-F14

**Plan**: [Portable Agent Workflow Port Plan](../plans/portable_agent_workflow_port_plan.md#8-task-decomposition)

## Task Card

Task: PAW-F14
Task file: docs/tasks/task_paw_f14_peer_review_artifact_presentation.md
Plan file: docs/plans/portable_agent_workflow_port_plan.md
Summary: Clarify the workflow presentation contract so peer review approval fields show the review artifact file path and the approval verdict in a clearly labeled structure. Keep the change documentation-only and make the display expectation explicit for both task cards and closure reports.
Code affected: No product code. Expected files are AGENTS.md, docs/playbooks/AGENT_WORKFLOW_GUIDE.md, docs/plans/portable_agent_workflow_port_plan.md, docs/tasks/.gitignore, and this task file.
Effort/reasoning: Low - This is a narrow workflow-doc clarification that makes an existing reporting contract easier to read and harder to misinterpret.
Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6
Estimated tokens: ~1400

## Summary

The workflow currently requires peer review fields, but the presentation format does not explicitly emphasize the evidence artifact path and verdict label as separate visual elements. This task updates the workflow guide and wrapper wording so the peer review approval section clearly surfaces both.

## Acceptance Criteria

1. `docs/playbooks/AGENT_WORKFLOW_GUIDE.md` states a presentation format for peer review fields that clearly labels the artifact file and approval status.
2. `AGENTS.md` mirrors the presentation requirement for task-card and closure reporting fields without creating a conflicting local format.
3. The workflow wording makes it obvious that the artifact path is the proof file and the verdict is the approval state.
4. The portable workflow plan records the clarification as a completed addendum after the documentation update is done.
5. No product code, peer review script behavior, or QA gate behavior changes in this task.

## Completion

Result: Clarified the peer-review reporting contract in the portable workflow guide and `AGENTS.md` so peer-review approval entries must show `reviewer`, `artifact`, and `status` explicitly. Updated the portable workflow plan to record the clarification as a completed addendum.

Verification:
- `python3 scripts/check_okf_frontmatter.py --changed-only --report-only`
- `git diff --check`

Files affected:
- `AGENTS.md`
- `docs/playbooks/AGENT_WORKFLOW_GUIDE.md`
- `docs/plans/portable_agent_workflow_port_plan.md`
- `docs/tasks/.gitignore`
- `docs/tasks/task_paw_f14_peer_review_artifact_presentation.md`

Effort/reasoning: Low - This was a focused docs-only clarification that improved reporting readability without changing workflow behavior.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Tokens: ~900
