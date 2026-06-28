---
doc_type: task
id: QG-ALIGN-FX-001
title: "Align fenix maintainability gate wiring across CI and BFF"
status: done
phase: remediation
week: "2026-W26"
tags: [qa, governance, maintainability, ci, bff]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - scripts/hooks/pre-push
  - .github/workflows/ci.yml
  - Makefile
  - docs/plans/fenix_maintainability_gate_wiring_alignment_plan.md
created: 2026-06-28
completed: 2026-06-28
---

# Task QG-ALIGN-FX-001

**Plan**: [Fenix maintainability gate wiring alignment](../plans/fenix_maintainability_gate_wiring_alignment_plan.md)

## Summary

Wire the existing `scripts/check-maintainability.py` gate so it executes consistently for BFF-triggered changes in pre-push and for repository diffs in CI. The checker already supports Go, Mobile, and BFF classification; this task closes the enforcement gap without changing thresholds or product logic.

## Acceptance Criteria

1. BFF-only changes trigger the maintainability checker from pre-push.
2. CI runs the maintainability checker with a stable base SHA for pull requests and pushes.
3. Existing Go, Mobile, and BFF quality jobs continue to run without regression.
4. The final report identifies the exact commands or workflow steps that now enforce the gate.

## Scope

- **In**: hook wiring, CI wiring, and any small Makefile adjustment required for consistent invocation.
- **Out**: threshold tuning, checker logic changes, and product-code refactors.

## Risks

- CI diff-base resolution can fail closed if fetch depth or event-specific SHAs are not handled carefully.
- Running the checker more often may expose pre-existing maintainability violations in BFF diffs that were previously unenforced.

## High-Level Pseudocode

```text
detect changed surfaces from diff range
if BFF changes are present:
  run existing BFF QA
  run check-maintainability with the same diff base

in CI:
  compute maintainability base from PR base SHA or push before SHA
  checkout with enough history
  invoke qa-maintainability once in a dedicated job
  keep existing mobile/BFF/backend jobs unchanged
```
