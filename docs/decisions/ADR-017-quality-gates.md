---
id: ADR-017
title: "Quality gates: gocognit ≤10, maintidx ≥20, gocyclo ≤7, applied to production code only"
date: 2026-02-05
status: accepted
deciders: [matias]
tags: [adr, go, lint, quality, ci]
related_tasks: [task_quality_gates]
related_frs: []
---

# ADR-017 — Quality gates: gocognit ≤10, maintidx ≥20, gocyclo ≤7, production code only

## Status

`accepted`

## Context

As the Go backend grew across Phases 1–3, several functions accumulated high complexity
that was not caught by basic linting. Two additional complexity metrics were introduced
alongside the existing gocyclo gate (ADR-006):

- **gocognit** (SonarSource algorithm): measures cognitive complexity — how hard code
  is to understand given nesting, breaks, and recursion. Stricter than cyclomatic complexity.
- **maintidx** (Microsoft rebased Maintainability Index): composite metric of cyclomatic
  complexity + Halstead volume + LOC. Measures how maintainable a function is overall.

The key refactoring case that triggered the gates: `cmd/frtrace/main.go` had:
- `validate()` → cognitive complexity 28 (way above threshold)
- `scanTraces()` → cognitive complexity 11 (above threshold)

## Decision

Enforce three quality thresholds, all configured in `.golangci.yml` (auto-detected by
golangci-lint, zero CI changes needed):

| Tool | Threshold | Interpretation |
|------|-----------|----------------|
| `gocyclo` | ≤ 7 | See ADR-006 |
| `gocognit` | ≤ 10 | SonarSource "maintainable" zone (default is 15) |
| `maintidx` | ≥ 20 | Microsoft green zone (0–9 = red, 10–19 = yellow, 20+ = green) |

**Exclusions** (configured in `.golangci.yml`):

```yaml
issues:
  exclude-rules:
    - path: _test.go
      linters: [gocognit, maintidx, gocyclo]
    - path: internal/infra/sqlite/sqlcgen/
      linters: [gocognit, maintidx, gocyclo]
    - path: "*.pb.go"
      linters: [gocognit, maintidx, gocyclo]
```

Test files are excluded because table-driven tests and setup functions legitimately
require higher complexity. Generated code (`sqlcgen/`, `.pb.go`) is excluded because
it cannot be refactored.

**Refactoring patterns to comply:**

1. Extract helpers by concern: `checkMissingAnnotations()`, `checkOrphanAnnotations()`
2. Extract regex parsing into named functions: `extractTraceAnnotation(line string)`
3. Use boolean helper functions: `containsTrace(annotations []string, id string) bool`

## Rationale

- `gocognit` is stricter than `gocyclo` for nested code — it penalizes nesting more
  heavily, which correlates better with actual developer confusion
- `maintidx ≥20` (Microsoft green zone) is a well-established industry threshold
- Running via golangci-lint means `make lint` covers all three gates — no separate
  `make cognitive` target needed
- Thresholds are documented so future contributors understand the bar, not just the tool

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| gocognit ≤15 (SonarSource default) | Too permissive — allowed `validate()` complexity=28 through |
| No maintidx gate | Missing the composite view (a function can be low-cyclomatic but still hard to maintain due to size) |
| Apply gates to test files | Legitimate test complexity would require artificial refactoring |
| Separate CI step for each gate | Already unified via golangci-lint — no benefit to splitting |

## Consequences

**Positive:**
- `make lint` is the single command to verify all three quality gates
- Consistent enforcement across all production packages
- Refactoring patterns (extract helpers, named parsers) improve readability beyond just passing the gate

**Negative / tradeoffs:**
- Higher bar means more refactoring effort for new features
- Some legitimate complex functions (e.g., a large but well-understood state machine)
  may need artificial extraction to pass — requires judgment call

## References

- `.golangci.yml` — linter configuration with thresholds
- `cmd/frtrace/main.go` — primary refactoring example (cognitive 28→<10)
- gocognit: https://github.com/uudashr/gocognit
- maintidx: https://github.com/yagipy/maintidx
