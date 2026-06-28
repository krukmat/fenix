---
doc_type: audit
title: "Go context.Background ratchet design"
status: completed
created: 2026-06-28
updated: 2026-06-28
tags: [audit, go, qa, context, governance, lint]
---

# Go context.Background ratchet design

## Objective

Design a narrow, high-signal ratchet for `context.Background()` usage in Go runtime paths without penalizing legitimate bootstrap and detached background ownership cases.

## Current non-test call sites

Detected non-test `context.Background()` uses:

- [internal/server/server.go:62](/Users/matias/fenix/internal/server/server.go:62)
- [internal/domain/audit/service.go:365](/Users/matias/fenix/internal/domain/audit/service.go:365)
- [internal/api/handlers/readyz.go:62](/Users/matias/fenix/internal/api/handlers/readyz.go:62)
- [internal/api/handlers/readyz.go:71](/Users/matias/fenix/internal/api/handlers/readyz.go:71)
- [internal/api/routes.go:341](/Users/matias/fenix/internal/api/routes.go:341)
- [internal/api/routes.go:513](/Users/matias/fenix/internal/api/routes.go:513)
- [internal/domain/agent/orchestrator.go:1076](/Users/matias/fenix/internal/domain/agent/orchestrator.go:1076)

## Classification

### Allowed by design

These uses should remain allowed because they establish explicit ownership roots or bootstrap defaults:

- [internal/server/server.go:62](/Users/matias/fenix/internal/server/server.go:62)
  Server-owned background context root. This is the process lifecycle owner and is immediately wrapped with `context.WithCancel`.
- [internal/api/routes.go:513](/Users/matias/fenix/internal/api/routes.go:513)
  Default fallback for `RouterRuntime.BackgroundContext` when callers provide no explicit root.
- [internal/domain/agent/orchestrator.go:1076](/Users/matias/fenix/internal/domain/agent/orchestrator.go:1076)
  Explicit detached background pipeline trigger, immediately bounded with `context.WithTimeout`.

### Debatable but acceptable if documented

These uses are not automatically wrong, but they are exactly the sort of places a narrow ratchet should make visible for human review:

- [internal/api/routes.go:341](/Users/matias/fenix/internal/api/routes.go:341)
  Tool-definition bootstrap using a background root during router construction. This is setup-time work, but detached from caller cancellation.
- [internal/domain/audit/service.go:365](/Users/matias/fenix/internal/domain/audit/service.go:365)
  Audit consumption loop logs events with a fresh root context rather than deriving from a bus- or service-owned root.
- [internal/api/handlers/readyz.go:62](/Users/matias/fenix/internal/api/handlers/readyz.go:62)
- [internal/api/handlers/readyz.go:71](/Users/matias/fenix/internal/api/handlers/readyz.go:71)
  Readiness checks create timeout-bounded roots instead of accepting a caller context. This is pragmatic, but it bypasses request cancellation entirely.

## Proposed ratchet scope

Do **not** ban `context.Background()` repository-wide.

Recommended scope:

- Flag new `context.Background()` uses under `internal/` and `pkg/`.
- Exempt the following categories:
  - explicit server/bootstrap roots wrapped immediately by `WithCancel` / `WithTimeout`;
  - default runtime fallback wiring where a struct field explicitly represents a background root;
  - detached background workers/pipelines that are both documented and time-bounded or cancelable;
  - `cmd/` entry points and CLI bootstrap code.

In practical terms, the ratchet should be framed as:

> New `context.Background()` calls in runtime code require an explicit ownership justification or a stronger parent context.

## Recommended enforcement surface

Preferred implementation surface:

- **`ruleguard/rules-performance.go`**, because the rule belongs with semantic runtime-safety checks and can stay visible in the main Go lint surface.

Fallback implementation surface:

- **`scripts/check-maintainability.py`** as a changed-lines-only ratchet if the team wants lower rollout risk than a repo-wide lint.

## Existing remediation required before rollout

If the team chooses a strict rule immediately, these current sites would need either remediation or explicit exception design:

- [internal/domain/audit/service.go:365](/Users/matias/fenix/internal/domain/audit/service.go:365)
- [internal/api/handlers/readyz.go:62](/Users/matias/fenix/internal/api/handlers/readyz.go:62)
- [internal/api/handlers/readyz.go:71](/Users/matias/fenix/internal/api/handlers/readyz.go:71)
- [internal/api/routes.go:341](/Users/matias/fenix/internal/api/routes.go:341)

If the team chooses a changed-lines-only ratchet, no immediate remediation is required; the current baseline can remain grandfathered.

## Recommendation

Recommended next step:

1. If the goal is low-risk governance, implement this as a **changed-lines-only ratchet** first.
2. Scope the first version narrowly to `internal/` and `pkg/`, excluding `cmd/`.
3. Require an inline justification comment or a named helper wrapper for intentional detached roots if the ruleguard path is chosen later.

This gives fenix a real semantic ratchet with higher signal than a generic ban while avoiding broad false positives.
