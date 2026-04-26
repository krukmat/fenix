---
id: ADR-004
title: "Avoid t.Setenv in parallel tests — use os.Setenv in TestMain"
date: 2026-01-25
status: accepted
deciders: [matias]
tags: [adr, testing, go]
related_tasks: [task_1.6]
related_frs: []
---

# ADR-004 — Avoid t.Setenv in parallel tests — use os.Setenv in TestMain

## Status

`accepted`

## Context

Several test packages need environment variables set before tests run (e.g. `JWT_SECRET`
for auth, `DATABASE_URL` for integration tests).

The first approach was to call `t.Setenv("KEY", "value")` inside each test function.
This caused a Go runtime panic when combined with `t.Parallel()`:

```
testing: test using t.Setenv can not use t.Parallel
```

Go's `t.Setenv` registers a cleanup function that restores the original env var value
when the test ends. This is incompatible with parallel goroutines that may still be
reading the variable after the cleanup runs.

## Decision

Use `TestMain` with `os.Setenv` to set package-level environment variables once, before
any test in the package runs:

```go
func TestMain(m *testing.M) {
    os.Setenv("JWT_SECRET", "test-secret-key-32-chars-min!!!")
    os.Exit(m.Run())
}
```

This pattern is applied to every package that needs shared env vars and uses `t.Parallel()`.

## Rationale

- `os.Setenv` persists for the entire process lifetime — fully compatible with parallel goroutines
- `TestMain` is Go's canonical hook for package-level setup/teardown
- One `TestMain` per package eliminates duplication across individual test functions
- The test secret value satisfies the 32-character minimum required by `getJWTSecret()`

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| `t.Setenv` per test | Incompatible with `t.Parallel()` — Go runtime panic |
| `init()` in `_test.go` file | Execution order relative to TestMain is not guaranteed |
| Mock `getJWTSecret()` | Requires dependency injection in production code for a test concern |
| Disable `t.Parallel()` | Significantly increases test suite execution time |

## Consequences

**Positive:**
- Parallel tests run without panics
- Minimal boilerplate per package (one `TestMain` function)
- Standard Go idiom, easy to understand for new contributors

**Negative / tradeoffs:**
- `JWT_SECRET` (and other vars) are set at the process level for the entire package test run.
  These are test-only secrets with no production value — negligible risk.

## References

- Go testing docs: `TestMain` — https://pkg.go.dev/testing#hdr-Main
- Go issue on `t.Setenv` + `t.Parallel` incompatibility: https://github.com/golang/go/issues/52817
- Affected packages: `pkg/auth`, `internal/domain/auth`, `internal/api/middleware`, `internal/api/handlers`
