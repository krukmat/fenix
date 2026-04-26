---
id: ADR-006
title: "Enforce cyclomatic complexity threshold of 7 via gocyclo"
date: 2026-02-05
status: accepted
deciders: [matias]
tags: [adr, go, lint, quality]
related_tasks: [task_quality_gates]
related_frs: []
---

# ADR-006 — Enforce cyclomatic complexity threshold of 7 via gocyclo

## Status

`accepted`

## Context

As handler functions and domain services grew to support full CRUD for all CRM entities
(Account, Contact, Lead, Deal, Case), several functions accumulated high cyclomatic
complexity — deeply nested conditionals, compound `||`/`&&` validation chains, and
long if-chains for partial update logic.

The default gocyclo threshold of 10 was too permissive. Functions were passing the gate
but were still difficult to understand and test in isolation.

## Decision

Enforce a cyclomatic complexity threshold of **7** via `gocyclo`, run as part of `make complexity`:

```makefile
complexity:
    gocyclo -over 7 $(shell find . -name "*.go" -not -path "*/vendor/*" -not -name "*_test.go" -not -path "*/sqlcgen/*")
```

The threshold applies to **production code only** — test files are excluded because test
helpers legitimately require higher complexity (table-driven tests, setup logic).

**Mandatory refactoring patterns to stay within threshold:**

1. **Extract validation functions:**
```go
// Instead of compound conditions inline:
func isDealRequestValid(r DealRequest) bool {
    return r.Title != "" && r.StageID != "" && r.AccountID != ""
}
```

2. **Extract update input builders:**
```go
func buildUpdateDealInput(existing Deal, req UpdateDealRequest) UpdateDealInput {
    // if-chain for partial update logic lives here, not in the handler
}
```

3. **Use coalesce helpers** (defined in `internal/api/handlers/helpers.go`):
```go
func coalesce(val, fallback string) string {
    if val != "" { return val }
    return fallback
}

func coalescePtr(val string, fallback *string) *string {
    if val != "" { return &val }
    return fallback
}
```

## Rationale

- Threshold of 7 aligns with the SonarSource "maintainable" zone (vs. their default 15)
- Lower complexity → smaller, single-responsibility functions → easier unit testing
- Extraction patterns are reusable across all entity handlers (consistent style)
- `make complexity` is a fast local gate — runs in <2s, no external service needed

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Threshold of 10 (gocyclo default) | Too permissive — allowed hard-to-maintain functions through |
| Threshold of 5 | Too strict — legitimate switch statements and error handling chains would fail |
| No gate, only code review | Manual review misses complexity growth over time |
| `gocognit` instead of `gocyclo` | Used in addition (see ADR-017), not as a replacement |

## Consequences

**Positive:**
- Handler functions are consistently small and testable in isolation
- New contributors get immediate feedback from `make complexity` before PR
- Coalesce helpers reduce boilerplate in update handlers across all entities

**Negative / tradeoffs:**
- More files/functions for the same feature surface area
- Extraction sometimes feels artificial for simple 3-branch conditionals

## References

- `Makefile` — `complexity` target
- `internal/api/handlers/helpers.go` — `coalesce`, `coalescePtr`
- gocyclo: https://github.com/fzipp/gocyclo
