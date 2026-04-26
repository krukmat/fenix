---
id: ADR-007
title: "Use pkgauth alias when importing pkg/auth from internal/domain/auth"
date: 2026-01-28
status: accepted
deciders: [matias]
tags: [adr, go, packages, auth]
related_tasks: [task_1.6]
related_frs: [FR-060]
---

# ADR-007 — Use pkgauth alias when importing pkg/auth from internal/domain/auth

## Status

`accepted`

## Context

The project has two packages with identical base names:

- `pkg/auth` — Go package name `auth`. Contains low-level JWT primitives: `GenerateJWT`,
  `ParseJWT`, `getJWTSecret`.
- `internal/domain/auth` — Go package name `auth`. Contains the domain-level `AuthService`,
  `LoginRequest`, `RegisterRequest`, business logic.

When `internal/domain/auth` needs to call `pkg/auth.GenerateJWT`, Go's import system
produces a naming collision — both packages are named `auth` in the same file scope.

```go
// WRONG — compiler error: imported and not used / redeclared
import (
    "github.com/.../pkg/auth"
    "github.com/.../internal/domain/auth"  // collision
)
```

## Decision

When importing `pkg/auth` from within `internal/domain/auth`, always use the alias `pkgauth`:

```go
import (
    pkgauth "github.com/fenixcrm/fenixcrm/pkg/auth"
)

// Usage:
token, err := pkgauth.GenerateJWT(userID, workspaceID)
```

This alias is only needed in files that reside inside `internal/domain/auth`. All other
packages import `pkg/auth` without an alias (no collision).

## Rationale

- Go requires unique identifiers per import in a file scope
- The alias `pkgauth` clearly signals "this is the low-level package-level auth utility"
  vs. the domain service
- No structural refactor needed — both packages have distinct, valid responsibilities

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Rename `pkg/auth` to `pkg/jwt` | Breaks all existing import paths across the codebase |
| Rename `internal/domain/auth` to `internal/domain/authdomain` | Go convention is to name the package after its domain, not add suffixes |
| Move JWT primitives into `internal/domain/auth` | Violates layering — pkg/ should be reusable without depending on internal/ |
| Use blank import + init() side effects | Not applicable — we need to call exported functions |

## Consequences

**Positive:**
- No compiler errors, no structural refactor
- `pkgauth.GenerateJWT(...)` reads clearly as a cross-layer call
- Pattern is localized to one package — no global impact

**Negative / tradeoffs:**
- Developers unfamiliar with the codebase may be confused by the alias on first read
- Must be documented (this ADR) so it is not removed as "unnecessary" in a future cleanup

## References

- `internal/domain/auth/service.go` — where `pkgauth` alias is used
- `pkg/auth/auth.go` — `GenerateJWT`, `ParseJWT`, `getJWTSecret`
- Go spec: import declarations — https://go.dev/ref/spec#Import_declarations
