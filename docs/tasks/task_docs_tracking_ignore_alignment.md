---
doc_type: task
id: DOC-TRACK-ALIGN-001
title: "Align ignore rules for shared docs governance artifacts"
status: done
phase: governance
week: "2026-W26"
tags: [docs, governance, gitignore, knowledge-management]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - .gitignore
  - docs/tasks/.gitignore
  - docs/plans/docs_tracking_ignore_alignment_plan.md
created: 2026-06-28
completed: 2026-06-28
---

# Task DOC-TRACK-ALIGN-001

**Plan**: [Docs tracking ignore alignment](../plans/docs_tracking_ignore_alignment_plan.md)

## Summary

Adjust ignore rules so the QA-governance plan and task artifacts that should be shared are not silently excluded from Git tracking. The goal is to preserve the local-only task workflow where useful while making canonical coordination records trackable.

## Acceptance Criteria

1. The ignore rule causing the plan/task tracking conflict is identified precisely.
2. The approved shared artifacts for this work are no longer ignored.
3. Existing local-only task behavior is not widened accidentally beyond the intended scope.
4. The final report states which artifacts remain intentionally ignored versus promoted.

## Scope

- **In**: targeted ignore-rule adjustments for this governance slice.
- **Out**: mass promotion of all historical task records.

## Risks

- Over-broad unignore patterns could start surfacing many operational task files that were intentionally local-only.
- Under-broad rules could fix the current files but leave the same governance problem recurring for the next canonical artifact.
