---
doc_type: task
id: CRIT-002
title: "Task criticality schema and labeling/concurrence protocol"
status: done
phase: implementation
week: 2026-W27
tags: [criticality, workflow, peer-review, governance, docs]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: [CRIT-003, CRIT-004, CRIT-005]
files_affected:
  - AGENTS.md
  - CLAUDE.md
  - docs/policies/HITL_AUTONOMY_POLICY.md
criticality: standard
criticality_basis: "RRI advisory signal is documentation-scoped and low-band; no auth, permission, migration, or runtime reviewer behavior change."
created: 2026-07-04
completed: 2026-07-04
---

# Task CRIT-002: Task criticality schema and labeling/concurrence protocol

**Plan**: [Critical-Task Classification and Local/Cross-Agent Reviewer Mix](../plans/critical_task_reviewer_mix_plan.md#rollout)

## Scope

Introduce the human- and agent-facing task-label contract that the plan
requires after CRIT-001 exposed the advisory RRI signal. This task is
documentation and workflow contract work only; it does not change runtime
reviewer behavior.

Changes in scope:

- Add `criticality: critical | standard` and `criticality_basis:` to the task
  record / task-card contract documented in `AGENTS.md`.
- Mirror the same contract in `CLAUDE.md` so both agent entrypoints describe
  the same required fields and responsibilities.
- Add an alignment note in `docs/policies/HITL_AUTONOMY_POLICY.md` clarifying
  that `criticality` is a workflow classification set by the developer agent,
  informed by RRI output, and that the readiness peer reviewer must concur
  with or dispute the declared label as part of task-readiness review.

## Out of scope

- Any change to `scripts/peer-workflow-review.py` prompts, verdict schema, or
  CLI flags (CRIT-003 and CRIT-004).
- Any new blocking approval gate tied directly to `criticality`.
- Any change to `scripts/check_okf_frontmatter.py` to enforce required
  frontmatter fields.
- Any runtime reviewer-model selection behavior.

## Acceptance criteria

- `AGENTS.md` documents `criticality:` and `criticality_basis:` as required
  task-card / task-file fields for applicable tasks under this plan.
- `CLAUDE.md` is aligned with the same field names and labeling protocol.
- `docs/policies/HITL_AUTONOMY_POLICY.md` contains a short alignment note that
  distinguishes RRI approval gating from the new workflow `criticality`
  classification.
- The documented protocol states that the developer agent sets the label using
  RRI output plus judgment, and the task-readiness peer reviewer must concur
  with or dispute it.
- No implementation code or peer-review runtime behavior changes in this task.

## Notes

The plan's intent is that `criticality_suggested` from CRIT-001 is advisory
input, not an automatic label. This task formalizes that distinction in the
governing docs before any reviewer-runtime behavior is added downstream.

## RRI

| Variable | Score | Evidence | Confidence |
|---|---|---|---|
| C cyclomatic | 0 | docs-only wording changes | High |
| F files | 2 | 3 governing docs | High |
| D domain | 0 | documentation / workflow contract only | High |
| T coverage | 0 | no code path changes | High |
| A ambiguity | 1 | wording must align across multiple governance docs | Medium |
| K coupling | 1 | contract affects downstream tasks 003-005 | Medium |
| P impact | 2 | changes workflow classification/reporting expectations | Medium |
| X context | 2 | plan + three governance docs | Medium |

Final RRI: 16 -> Low band (0-25) -> executable without full approval packet,
but still requires task-card presentation per repo reporting rules.
