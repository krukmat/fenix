---
doc_type: task
id: QG-COMP-DB-FX-001
title: "Compare DubBridge maintainability gates against fenix QA coverage"
status: done
phase: analysis
week: "2026-W26"
tags: [qa, governance, comparison, dubbridge, fenix, eslint, maintainability]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - /Users/matias/dubbridge
  - scripts/
  - mobile/
  - bff/
  - .github/workflows/
  - docs/plans/quality_gate_comparison_dubbridge_fenix_plan.md
  - docs/dubbridge_fenix_quality_gate_comparison_audit.md
created: 2026-06-28
completed: 2026-06-28
---

# Task QG-COMP-DB-FX-001

**Plan**: [DubBridge vs Fenix Quality Gate Comparison](../plans/quality_gate_comparison_dubbridge_fenix_plan.md)

## Summary

Inspect the maintainability and lint-related gates in `/Users/matias/dubbridge`, then compare them with the gates currently enforced in fenix for Backend, Mobile, and BFF. The goal is to determine whether fenix already includes equivalent controls that push agents toward refactoring, separation of concerns, and lower line-count / lower-complexity implementations.

## Acceptance Criteria

1. The DubBridge gate or gates referenced by the user are identified precisely by file and enforcement mechanism.
2. Fenix gates relevant to Backend, Mobile, and BFF are enumerated from scripts, hooks, Make targets, CI, and language tooling.
3. The final report distinguishes between direct equivalence, partial equivalence, and absence of coverage.
4. The report names concrete files and commands that support each conclusion.

## Scope

- **In**: Comparative audit of QA gates and enforcement surfaces.
- **Out**: Implementing new rules, changing thresholds, or editing product code.

## Risks

- DubBridge may express the policy indirectly through custom scripts rather than a single ESLint rule, requiring cross-file inspection.
- Some fenix coverage may be distributed across multiple gates instead of a single maintainability budget, so the comparison must separate behavior from implementation detail.
