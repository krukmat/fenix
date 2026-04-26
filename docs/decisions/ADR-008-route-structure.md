---
id: ADR-008
title: "Route structure — public vs protected, JWT claims replace X-Workspace-ID header"
date: 2026-01-30
status: accepted
deciders: [matias]
tags: [adr, go, routing, auth, middleware]
related_tasks: [task_1.6]
related_frs: [FR-060, FR-070]
---

# ADR-008 — Route structure: public vs protected, JWT claims replace X-Workspace-ID header

## Status

`accepted`

## Context

Task 1.6 introduced authentication (JWT + bcrypt). Before this task, all routes were
unprotected and workspace context was injected via an `X-Workspace-ID` HTTP header
processed by a `WorkspaceMiddleware`.

After introducing auth, two decisions were required:

1. **Which routes are public vs. protected?**
2. **How does workspace context flow once the user is authenticated?**

The original `WorkspaceMiddleware` approach required every client request to include
an explicit `X-Workspace-ID` header — error-prone and redundant once the JWT contains
the workspace ID.

## Decision

**Route groups:**

```
Public (no auth required):
  GET  /health
  POST /auth/register
  POST /auth/login

Protected (Bearer JWT required):
  /api/v1/* — all CRM and AI endpoints
```

The `AuthMiddleware` from `internal/api/middleware` validates the Bearer JWT on every
`/api/v1/*` request and injects user + workspace context into the request context.

**Workspace context:**

The JWT payload includes `workspace_id`. The `AuthMiddleware` extracts it and stores it
in the request context via `context.WithValue`. The `WorkspaceMiddleware` (X-Workspace-ID
header) is removed.

Handler tests bypass middleware entirely by injecting workspace context directly:

```go
ctx := contextWithWorkspaceID(context.Background(), "test-workspace-id")
req = req.WithContext(ctx)
```

## Rationale

- JWT already carries workspace identity — requiring a separate header is redundant and
  a security surface (client could claim a different workspace)
- Single middleware (`AuthMiddleware`) handles both auth and workspace injection
- Public routes are minimal — only what is needed before a session exists
- Handler tests remain fast and isolated — no JWT generation required

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Keep `X-Workspace-ID` header alongside JWT | Redundant; two sources of truth for workspace identity |
| Embed workspace in URL path (`/api/v1/{workspaceID}/...`) | Verbose; requires every route to carry the workspace segment |
| Per-route auth annotations | More flexible but complex; overkill for a two-tier (public/protected) model |

## Consequences

**Positive:**
- Workspace context is always authoritative (from JWT, not client-supplied header)
- Simpler middleware stack — one middleware for auth + workspace injection
- Handler tests don't need JWT infrastructure

**Negative / tradeoffs:**
- Changing a user's workspace requires re-issuing a JWT (not just changing a header)
- Multi-workspace users (future P1) will need a workspace-switching flow at the token level

## References

- `internal/api/middleware/auth.go` — `AuthMiddleware`
- `internal/api/handlers/routes.go` — route registration
- `internal/api/handlers/` — `contextWithWorkspaceID` test helper
