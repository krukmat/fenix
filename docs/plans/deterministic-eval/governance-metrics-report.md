---
doc_type: summary
title: Workflow Inspection and Governance Metrics Surface
status: complete
created: 2026-05-02
tags: [eval, governance, workflow, metrics, demo]
---

# Workflow Inspection and Governance Metrics Surface

## Purpose

Wave F12 exposes a reviewable surface that joins three things already present in the codebase:

- workflow logic projection
- workflow safety and conformance
- deterministic operational metrics from governed runs

The target reviewer question is:

> What does this workflow do, is it safe, and how did it behave in deterministic runs?

## Code Surface

- Workflow inspection export: `internal/domain/agent/workflow_inspection.go`
- Existing projection source reused by the export: `internal/domain/agent/visual_projection.go`
- Validation source reused by the export: `internal/domain/agent/workflow_validation.go`
- Governance metrics aggregation: `internal/domain/eval/governance_metrics_report.go`

## What The Workflow Inspection Surface Exposes

`BuildWorkflowInspectionSurface()` reuses the existing validation/projection path and exports:

- `visual_projection`
- `adjacency`
- `mermaid`
- `conformance`
- DSL coverage labels from `BuildDSLCoverageSummary()`
- deterministic scenario references where they are known

This keeps F12 aligned with the real workflow graph instead of inventing a second graph model.

## What The Governance Metrics Surface Exposes

`BuildGovernanceMetricsReport()` aggregates deterministic run evidence from `ActualRunTrace` plus regression results and exports:

- per-run rows with actor, workflow, scenario, outcome, cost, latency, retries, approvals, policy denials, and hard gate counts
- pass/fail summary
- breakdowns by workflow, scenario, actor, outcome, and tool
- Markdown and JSON export paths suitable for review packets, demo notes, or public technical writeups

This makes the governance view answerable without LLM judgment.

## Example Review Flow

1. Build `WorkflowValidationResult` for the workflow under review.
2. Build `WorkflowInspectionSurface` and inspect:
   - Mermaid graph
   - adjacency list
   - conformance profile
   - deterministic scenario references
3. Build `GovernanceMetricsReport` from deterministic review cases and inspect:
   - who triggered runs
   - which workflow was involved
   - which tools were used or denied
   - pass/fail rate
   - latency, retries, and cost
   - hard gate and denial counts

## Reviewer Narrative

F12 is not an interactive graph feature.

F12 is a public and technical review surface that demonstrates:

- business logic is inspectable
- workflow safety is visible
- deterministic coverage can be linked
- governance outcomes are measurable

## Demo Positioning

Use this surface when the audience needs to see that workflow automation is:

- authored as explicit logic
- constrained by conformance
- observable through deterministic run evidence
- exportable as governance reporting

## Verification

Validated by:

- `go test ./internal/domain/agent ./internal/domain/eval`
