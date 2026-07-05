---
doc_type: task
id: PAW-F1
title: "Define provider-aware peer-review policy and reporting contract"
status: done
phase: F
week: ""
tags: [paw, devex, workflow, peer-review, policy, docs]
fr_refs: []
uc_refs: []
blocked_by: [PAW-F0, PAW-A3, PAW-B3, PAW-C2]
blocks: [PAW-F2]
files_affected:
  - AGENTS.md
  - CLAUDE.md
  - README_AGENT_ORDER.md
  - docs/policies/HITL_AUTONOMY_POLICY.md
  - docs/policies/RRI_POLICY.md
  - docs/plans/portable_agent_workflow_port_plan.md
  - docs/tasks/task_paw_f1_peer_review_policy_contract.md
created: 2026-07-01
completed: "2026-07-01"
rri: 24
rri_band: Low
hp: "Claude Code caller resolves to Codex peer review, Codex caller resolves to Claude peer review, and third-party local or remote providers resolve to Claude peer review"
ec: "Unknown caller kind defaults to Claude; unavailable peer CLI is specified as a blocked artifact instead of self-review; peer review never replaces HITL approval"
coverage_cert: ""
---

# Task PAW-F1

**Plan**: [Portable Agent Workflow Port Plan](../plans/portable_agent_workflow_port_plan.md#8-task-decomposition)

## Task Card

Task: PAW-F1
Task file: docs/tasks/task_paw_f1_peer_review_policy_contract.md
Plan file: docs/plans/portable_agent_workflow_port_plan.md
Summary: Define the provider-aware peer-review policy and reporting contract for Phase F. This task updates workflow documentation only; it does not implement the peer-review script or Makefile target.
Code affected: No product code. Expected files are AGENTS.md, CLAUDE.md, README_AGENT_ORDER.md, docs/policies/HITL_AUTONOMY_POLICY.md, docs/policies/RRI_POLICY.md, docs/plans/portable_agent_workflow_port_plan.md, and this task file.
Effort/reasoning: Low - Documentation and policy contract alignment only.
Recommended model: claude-sonnet-4-6
Estimated tokens: ~3500

## Summary

Define the policy-level contract for independent peer review before task-card presentation and before closing code tasks. The policy must be provider-aware: Claude Code work is reviewed by Codex, Codex work is reviewed by Claude, and other local or remote providers default to Claude.

## Acceptance Criteria

1. Workflow docs define peer-review checkpoints before task-card presentation and before code-task closure.
2. Provider resolution is explicit: `claude-code -> codex`, `codex -> claude`, `local-provider -> claude`, `remote-provider -> claude`, `unknown -> claude`.
3. The task card contract includes `Peer readiness review: <reviewer> <artifact path> - PASS`.
4. The code-task closure report contract includes `Peer code review: <reviewer> <artifact path> - PASS`.
5. The docs state that non-pass peer verdicts block presentation or closure until revised, explicitly waived by the user, or reported as blocked. **Superseded 2026-07-05 by `ADR-035` / `PAW-F15`: the waiver and BLOCKED-as-acceptable-terminal-state language defined here was removed. Only a `PASS` verdict unblocks presentation or closure; see `docs/decisions/ADR-035-peer-review-gate-unconditional-block.md`.**
6. The docs state that peer review does not replace human approval required by RRI/HITL.
7. No script, Makefile target, hook, or CI change is implemented in this task.

## Scope

- **In**: Policy wording, task-card/reporting contract updates, caller/reviewer resolution rules, and failure-mode wording.
- **Out**: No `scripts/peer-workflow-review.py`, no `make qa-peer-workflow-review`, no hook enforcement, no CI job, no product code.

## Risks

- The new review rule touches several workflow documents that already contain overlapping reporting language. Keep wording consistent and avoid creating a second incompatible task-card format.
- A too-broad policy could imply hook enforcement before the script exists. Keep enforcement phrased as a workflow/reporting contract until PAW-F2 and PAW-F3 are implemented.
