---
doc_type: plan
title: "Fenix maintainability gate wiring alignment"
status: completed
owner: "Workflow / QA Governance"
created: 2026-06-28
updated: 2026-06-28
tags: [qa, governance, maintainability, ci, bff]
---

# Fenix maintainability gate wiring alignment

## Purpose

Bring fenix maintainability gate enforcement in line with the intended governance model by ensuring the existing diff-based maintainability checker runs for all relevant change surfaces in both pre-push and CI.

## Scope

- Wire `qa-maintainability` into CI with an appropriate diff base.
- Extend pre-push so BFF-only changes also run `scripts/check-maintainability.py`.
- Preserve existing Go, Mobile, and BFF QA flows.
- Update any affected governance or audit artifacts if enforcement behavior changes.

## Out of Scope

- Changing maintainability thresholds.
- Adding a new semantic rule to the checker.
- Refactoring product code to satisfy new findings.

## Deliverable

A repository state where the maintainability gate blocks regressions consistently for Go, Mobile, and BFF changes in both local pre-push and CI.
