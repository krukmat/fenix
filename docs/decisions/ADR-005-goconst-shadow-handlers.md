---
id: ADR-005
title: "Centralize string constants in helpers.go and avoid variable shadowing in handlers"
date: 2026-02-01
status: accepted
deciders: [matias]
tags: [adr, go, handlers, lint]
related_tasks: [task_1.5, task_1.6]
related_frs: []
---

# ADR-005 — Centralize string constants in helpers.go and avoid variable shadowing in handlers

## Status

`accepted`

## Context

The `internal/api/handlers` package has many handler functions that repeat string
literals (content-type headers, error messages, route parameter names). The `goconst`
linter flags repeated string literals that should be constants. The `govet/shadow`
linter flags variable declarations that shadow an outer `err` variable declared with `:=`.

Two specific failure patterns emerged during linting:

**Pattern 1 — Repeated string literals:**
```go
// In multiple handler files:
w.Header().Set("Content-Type", "application/json")
http.Error(w, "missing workspace ID", http.StatusBadRequest)
chi.URLParam(r, "id")
```

**Pattern 2 — Variable shadowing:**
```go
err := doSomething()
if err != nil { ... }
// Later in the same function:
result, err := doSomethingElse() // shadows outer err — govet warns
```

Additionally, `replace_all` edits on string literals accidentally replaced the RHS of
the `const` declaration itself, creating a circular initialization error.

## Decision

**For string constants:**
All shared string constants live in `internal/api/handlers/helpers.go`. Key constants:

```go
const (
    headerContentType        = "Content-Type"
    mimeJSON                 = "application/json"
    errMissingWorkspaceID    = "missing workspace ID"
    errMissingWorkspaceContext = "missing workspace context"
    errInvalidBody           = "invalid request body"
    errFailedToEncode        = "failed to encode response"
    paramID                  = "id"
    paramStageID             = "stage_id"
)
```

Route patterns in `routes.go` use:
```go
const routeByID = "/{id}"
```

**For variable shadowing:**
Rename inner `:=` declarations to `decodeErr`, `scanErr`, `updateErr`, etc. when an
outer `err` already exists in the function scope.

**For `replace_all` edits:**
Never use `replace_all=true` on string literal values — it also replaces the RHS of
the `const` declaration. Always use targeted `Edit` calls (`replace_all=false`) for
const lines.

## Rationale

- Single source of truth for all error/header strings — change in one place
- Eliminates goconst and govet/shadow lint failures
- `helpers.go` is already the canonical location for shared handler utilities
- Named error variables (`decodeErr`, `scanErr`) improve readability at the call site

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Inline constants per file | Duplication; goconst will still flag them |
| Package-level `errors.go` | Extra file for what is already covered by `helpers.go` |
| Ignore lint warnings | Lint gates are enforced in CI — not an option |

## Consequences

**Positive:**
- `make lint` passes cleanly on the handlers package
- Consistent error messages across all endpoints
- Easier to refactor error strings in one place

**Negative / tradeoffs:**
- New contributors must know to add constants to `helpers.go` rather than inline
- Shadow variable renaming (`decodeErr`, etc.) adds minor verbosity

## References

- `internal/api/handlers/helpers.go` — canonical location for shared constants
- `internal/api/handlers/routes.go` — `routeByID` pattern
- golangci-lint docs: `goconst` and `govet` linters
