---
doc_type: plan
title: "DubBridge vs Fenix quality gate comparison"
status: completed
owner: "Workflow / QA Governance"
created: 2026-06-28
updated: 2026-06-28
tags: [qa, governance, comparison, dubbridge, fenix]
---

# DubBridge vs Fenix Quality Gate Comparison

## Purpose

Review whether DubBridge quality gates that encourage refactoring through maintainability constraints, ESLint policy, and line-oriented discipline are already represented in fenix across Backend, Mobile, and BFF.

## Scope

- Inspect `/Users/matias/dubbridge` for the referenced quality gates.
- Inspect `fenix` QA surfaces in scripts, hooks, Make targets, CI, and language-specific tooling.
- Map findings by area: Backend, Mobile, BFF.
- Identify direct equivalents, partial coverage, and missing gates.

## Out of Scope

- No code changes to product runtime.
- No immediate implementation of new gates unless requested in a follow-up task.

## Deliverable

A comparison report stating whether fenix already enforces comparable gates, where they are wired, and what gaps remain.
