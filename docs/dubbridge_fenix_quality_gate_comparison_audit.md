---
doc_type: audit
title: "DubBridge vs Fenix quality gate comparison"
status: completed
created: 2026-06-28
updated: 2026-06-28
tags: [audit, qa, governance, dubbridge, fenix, maintainability, eslint]
---

# DubBridge vs Fenix quality gate comparison

## Objective

Determine whether the DubBridge quality gates that force decomposition, lower complexity, and smaller functions are already represented in fenix across Backend, Mobile, and BFF.

## Baseline observed in DubBridge

### Backend (Rust)

- Workspace Clippy lints deny `too_many_lines` and `cognitive_complexity`, so overly long or overly complex production Rust functions fail `qa-lint` directly. See `/Users/matias/dubbridge/Cargo.toml:53`.
- `clippy.toml` sets `cognitive-complexity-threshold = 15`, making the cognitive gate explicit. See `/Users/matias/dubbridge/clippy.toml:10`.
- `scripts/check-maintainability.py` adds a diff-based maintainability ratchet for backend/mobile changes: added-line budgets, duplicate/repeated block detection, declaration bursts, long lines, generated markers, and a production Rust `.unwrap()` / `.expect()` ratchet. See `/Users/matias/dubbridge/scripts/check-maintainability.py:1`.
- The maintainability gate is wired into both `make qa-maintainability`, pre-push, and CI. See `/Users/matias/dubbridge/Makefile:40`, `/Users/matias/dubbridge/scripts/hooks/pre-push:11`, and `/Users/matias/dubbridge/.github/workflows/ci.yml:76`.

### Mobile

- Mobile ESLint rejects `complexity > 10` and `max-lines-per-function > 60`, plus `no-explicit-any`, `ban-ts-comment`, `no-console`, and `no-debugger`. See `/Users/matias/dubbridge/mobile/eslint.config.js:34`.
- Mobile QA is enforced through `make qa-mobile` and CI. See `/Users/matias/dubbridge/Makefile:54` and `/Users/matias/dubbridge/.github/workflows/ci.yml:88`.

### BFF

- No separate BFF package or BFF-specific ESLint surface was found in `/Users/matias/dubbridge`.
- Comparison for BFF therefore uses DubBridge as "no direct counterpart".

## Fenix comparison

### Backend

Verdict: **Yes, covered, and in several respects stricter than DubBridge's intent.**

- `make complexity` blocks production Go functions above cyclomatic complexity 7 and explicitly instructs refactor-before-merge. See [Makefile](/Users/matias/fenix/Makefile:96).
- ADR-006 documents the intended decomposition patterns: extract validation, builders, and helpers to stay under the threshold. See [ADR-006-complexity-gate.md](/Users/matias/fenix/docs/decisions/ADR-006-complexity-gate.md:28).
- `.golangci.yml` adds cognitive complexity (`gocognit`), maintainability index (`maintidx`), function length (`funlen`), duplication (`dupl`), dependency-boundary (`depguard`), and interface-size (`interfacebloat`) gates. See [.golangci.yml](/Users/matias/fenix/.golangci.yml:4).
- Go pre-push runs complexity, lint, wrapcheck, tests, coverage, deadcode, traceability, vuln scan, race, and pattern-refactor gates. See [qa-go-prepush.sh](/Users/matias/fenix/scripts/qa-go-prepush.sh:1).
- CI runs dedicated complexity and lint/test jobs. See [ci.yml](/Users/matias/fenix/.github/workflows/ci.yml:91).

### Mobile

Verdict: **Yes, covered, with broader ESLint coverage but a looser function-line threshold than DubBridge.**

- Mobile ESLint enforces `complexity <= 10`, `sonarjs/cognitive-complexity <= 15`, `max-lines-per-function <= 80`, and `max-lines <= 300`, plus `no-explicit-any`, `import/no-cycle`, hook rules, `no-console`, and `no-debugger`. See [mobile/eslint.config.js](/Users/matias/fenix/mobile/eslint.config.js:19).
- Mobile pre-push runs no-inline-disable, typecheck, lint, architecture checks, and coverage. See [qa-mobile-prepush.sh](/Users/matias/fenix/scripts/qa-mobile-prepush.sh:1).
- CI runs mobile typecheck, lint, architecture, and coverage as a dedicated job. See [ci.yml](/Users/matias/fenix/.github/workflows/ci.yml:14).
- `quality:arch` adds app-architecture constraints that DubBridge did not expose in the inspected surface, such as BFF-only API calls, workspace-aware query keys, layer import restrictions, unstable keys, and hardcoded URL rejection. See [quality-check.mjs](/Users/matias/fenix/mobile/scripts/quality-check.mjs:37).

### BFF

Verdict: **Yes, covered in fenix, but this is a fenix-specific layer rather than a DubBridge port equivalent.**

- BFF ESLint enforces `complexity <= 8`, `sonarjs/cognitive-complexity <= 10`, `max-lines-per-function <= 60`, and `max-lines <= 200`, plus strict async/type hygiene. See [bff/eslint.config.js](/Users/matias/fenix/bff/eslint.config.js:26).
- BFF pre-push runs typecheck, lint, and coverage. See [qa-bff-prepush.sh](/Users/matias/fenix/scripts/qa-bff-prepush.sh:1).
- CI runs dedicated BFF typecheck, lint, and coverage. See [ci.yml](/Users/matias/fenix/.github/workflows/ci.yml:55).

## Gap analysis

### Gap 1 — Fenix maintainability gate is only partially wired

- The ported `scripts/check-maintainability.py` already supports Go, Mobile, and BFF path classification. See [check-maintainability.py](/Users/matias/fenix/scripts/check-maintainability.py:167).
- However, pre-push invokes it only for Go and Mobile, not for BFF changes. See [pre-push](/Users/matias/fenix/scripts/hooks/pre-push:45).
- CI does not run `qa-maintainability` at all, even though the Make target exists. See [Makefile](/Users/matias/fenix/Makefile:380) and [ci.yml](/Users/matias/fenix/.github/workflows/ci.yml:13).
- Compared with DubBridge, this means fenix has the diff-based maintainability checker present but not fully enforced as a repository-wide quality gate.

### Gap 2 — No direct Go equivalent of DubBridge's Rust runtime-risk ratchet

- DubBridge blocks newly added production `.unwrap()` / `.expect()` in diffs. See `/Users/matias/dubbridge/scripts/check-maintainability.py:69`.
- Fenix removed the Rust-specific runtime-risk ratchet during adaptation. See [check-maintainability.py](/Users/matias/fenix/scripts/check-maintainability.py:4).
- This is reasonable given the language change, but it means there is no one-to-one equivalent for that exact "unsafe convenience call" diff ratchet on the backend side.

### Gap 3 — Mobile threshold is not identical

- DubBridge mobile uses `max-lines-per-function = 60`. See `/Users/matias/dubbridge/mobile/eslint.config.js:42`.
- Fenix mobile uses `max-lines-per-function = 80` and adds `max-lines = 300`. See [mobile/eslint.config.js](/Users/matias/fenix/mobile/eslint.config.js:20).
- Conclusion: the mobile intent is present, but the specific decomposition pressure is somewhat looser per function in fenix.

## Final conclusion

Fenix **does already consider these quality gates** across the three areas, but not uniformly in the exact same way as DubBridge:

- **Backend:** yes, clearly covered through `gocyclo`, `gocognit`, `maintidx`, `funlen`, `dupl`, `depguard`, `wrapcheck`, and pre-push/CI wiring.
- **Mobile:** yes, clearly covered through ESLint complexity/line gates, no-inline-disable enforcement, architecture checks, and CI/pre-push wiring.
- **BFF:** yes, clearly covered in fenix through ESLint complexity/line gates and CI/pre-push, even though DubBridge did not expose a separate BFF counterpart.
- **Main governance gap:** the diff-based `qa-maintainability` gate exists in fenix but is **not fully enforced** the way it is in DubBridge, especially for **CI** and **BFF-triggered changes**.
