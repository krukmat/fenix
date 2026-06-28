---
doc_type: plan
title: "Docs tracking ignore alignment"
status: completed
owner: "Workflow / QA Governance"
created: 2026-06-28
updated: 2026-06-28
tags: [docs, governance, gitignore, knowledge-management]
---

# Docs tracking ignore alignment

## Purpose

Resolve ignore-rule conflicts that prevent intended governance artifacts under `docs/plans/` and selected `docs/tasks/` paths from being tracked when they should act as shared coordination records.

## Scope

- Inspect current `.gitignore` and `docs/tasks/.gitignore` behavior.
- Decide which plan/task artifacts should remain local-only versus Git-trackable.
- Adjust ignore rules only for the approved shared artifacts.
- Preserve the existing local scratch-task workflow where appropriate.

## Out of Scope

- Bulk promotion of all task files into Git tracking.
- Rewriting the broader documentation structure.

## Deliverable

A repo state where the intended canonical plan/task artifacts for this QA governance work are no longer silently blocked by ignore rules.
