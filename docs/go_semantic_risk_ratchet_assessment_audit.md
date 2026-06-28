---
doc_type: audit
title: "Go semantic risk ratchet assessment"
status: completed
created: 2026-06-28
updated: 2026-06-28
tags: [audit, go, qa, lint, governance, maintainability]
---

# Go semantic risk ratchet assessment

## Objective

Assess whether fenix should add a Go-specific semantic risk ratchet analogous in intent to DubBridge's Rust diff ratchet for risky convenience calls.

## Current Go gate coverage

Fenix already has strong backend coverage across multiple dimensions:

- Cyclomatic complexity via `make complexity` / `gocyclo <= 7`.
- Cognitive complexity via `gocognit <= 8`.
- Maintainability index via `maintidx >= 20`.
- Function size via `funlen`.
- Duplication via `dupl`.
- Architectural boundaries via `depguard`.
- Error-handling correctness via `errorlint`.
- Security checks via `gosec`.
- Additional ruleguard-based semantic checks in `ruleguard/rules-performance.go` and `ruleguard/rules-smells.go`.

Relevant existing semantic protections already present in `ruleguard/rules-performance.go` include:

- `panic(...)` in production code.
- silent error drops `_, _ = ...`.
- `http.Get(...)` / `http.Post(...)` with no timeout.
- `log.Fatal*` / `log.Panic*`.
- `time.Sleep(...)` in production code.

These are already enforced through `gocritic` + ruleguard in the lint stack. See [.golangci.yml](/Users/matias/fenix/.golangci.yml:115), [rules-performance.go](/Users/matias/fenix/ruleguard/rules-performance.go:1), and [rules-smells.go](/Users/matias/fenix/ruleguard/rules-smells.go:1).

## Candidate patterns reviewed

### 1. `panic` / `log.Fatal*` / `log.Panic*`

Verdict: **Already covered well enough.**

- `panic(...)` is already reported by ruleguard.
- `log.Fatal*` / `log.Panic*` are already reported by ruleguard.
- Existing known exceptions are narrowly allowlisted in `.golangci.yml`, such as `pkg/auth/auth.go` startup panic and `internal/infra/sqlite/vector_functions.go` init-time `log.Fatalf`.

Recommendation: no new ratchet needed here.

### 2. `http.DefaultClient`

Verdict: **Not worth a new ratchet right now.**

- The existing ruleguard already blocks `http.Get(...)` / `http.Post(...)`, which are the most obvious timeout-free shortcuts.
- The only `http.DefaultClient` use found in the current repo scan is in `scripts/e2e_seed_mobile_p2.go`, not hot-path production runtime.

Recommendation: no new ratchet needed now.

### 3. `os.Exit(...)`

Verdict: **Too context-dependent for a repository-wide ratchet.**

- Current occurrences are in `cmd/` entry points (`cmd/fenixlsp`, `cmd/frtrace`), which are legitimate process-boundary locations.
- A broad ban would likely just create allowlist churn around CLI bootstrap code.

Recommendation: no new ratchet needed.

### 4. `fmt.Print*`

Verdict: **Low signal for a new ratchet.**

- Current production occurrences are mostly server lifecycle prints and CLI output.
- A blanket ban would overlap with operational logging choices and produce more style noise than correctness value.

Recommendation: no new ratchet needed.

### 5. `context.Background()`

Verdict: **Only candidate worth considering for a future narrow ratchet.**

Current non-test occurrences found:

- [internal/server/server.go:62](/Users/matias/fenix/internal/server/server.go:62)
- [internal/domain/agent/orchestrator.go:1076](/Users/matias/fenix/internal/domain/agent/orchestrator.go:1076)
- [internal/domain/audit/service.go:365](/Users/matias/fenix/internal/domain/audit/service.go:365)
- [internal/api/routes.go:341](/Users/matias/fenix/internal/api/routes.go:341)
- [internal/api/routes.go:513](/Users/matias/fenix/internal/api/routes.go:513)
- [internal/api/handlers/readyz.go:62](/Users/matias/fenix/internal/api/handlers/readyz.go:62)
- [internal/api/handlers/readyz.go:71](/Users/matias/fenix/internal/api/handlers/readyz.go:71)

Interpretation:

- Some uses are legitimate bootstrap roots or explicit background-runtime roots:
  - server-owned background context in `internal/server/server.go`
  - default background runtime fallback in `internal/api/routes.go`
  - detached background pipeline trigger in `internal/domain/agent/orchestrator.go`
- Some uses are more debatable:
  - audit event consumption in `internal/domain/audit/service.go`
  - readiness probes in `internal/api/handlers/readyz.go`
  - workspace tool-definition bootstrap in `internal/api/routes.go`

This pattern is the only one that plausibly matches the intent of a semantic ratchet: it can hide cancellation/ownership mistakes while still compiling cleanly and passing complexity gates.

## Recommendation

Do **not** add a broad new Go semantic ratchet right now.

Recommended conclusion:

1. The current Go lint/quality stack is already strong and covers most obvious risky shortcuts.
2. There is **no clear high-signal equivalent** to DubBridge's Rust `.unwrap()` / `.expect()` ratchet that justifies immediate rollout.
3. If fenix wants one future semantic ratchet, the best candidate is a **narrow, scoped `context.Background()` rule** for selected runtime paths, with explicit exceptions for:
   - process/bootstrap roots,
   - server-owned background contexts,
   - CLI entry points,
   - clearly documented detached background work.

## Suggested future implementation surface

If this is pursued later, the best implementation surface is:

- **Preferred:** `ruleguard/rules-performance.go` with path-aware or package-aware conventions and documented allowlist cases.
- **Fallback:** a small diff-based addition to `scripts/check-maintainability.py` if the team wants the ratchet to be changed-lines-only rather than repo-wide.

## Verification

- Static inspection of [.golangci.yml](/Users/matias/fenix/.golangci.yml:1)
- Static inspection of [ruleguard/rules-performance.go](/Users/matias/fenix/ruleguard/rules-performance.go:1) and [ruleguard/rules-smells.go](/Users/matias/fenix/ruleguard/rules-smells.go:1)
- Repository scan of candidate risky patterns with `rg` across `internal/`, `pkg/`, `cmd/`, and `scripts/`
