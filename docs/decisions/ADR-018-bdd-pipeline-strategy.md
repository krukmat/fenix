---
id: ADR-018
title: "BDD pipeline: Go stack implemented (godog), BFF/Mobile as placeholders; UC→FR→TST traceability via cmd/frtrace"
date: 2026-03-01
status: accepted
deciders: [matias]
tags: [adr, bdd, testing, traceability, doorstop]
related_tasks: [task_bdd_use_cases_conversion_master, task_bdd_p7_ci_and_runner_entrypoints]
related_frs: [FR-070]
---

# ADR-018 — BDD pipeline: Go stack implemented (godog), BFF/Mobile as placeholders; UC→FR→TST traceability via cmd/frtrace

## Status

`accepted`

## Context

FenixCRM has three test stacks (Go backend, BFF Express, Mobile Detox). Each has a
corresponding BDD runner entry point. Additionally, the project uses Doorstop for
requirements traceability (UC → FR → TST chain).

Key decisions were required:

1. **Which BDD stack to implement first?** All 3 at once vs. prioritized.
2. **How to enforce traceability?** Manual vs. automated gate.
3. **How to tag BDD scenarios?** Free-form vs. structured metadata contract.

## Decision

### BDD stack implementation priority

| Stack | Status | Runner | Notes |
|-------|--------|--------|-------|
| Go (godog) | ✅ Implemented | `make test-bdd-go` | 33 scenarios across 16 UCs |
| BFF (Jest) | ⏳ Placeholder | `make test-bdd-bff` | `bff/tests/bdd/README.md` only |
| Mobile (Detox) | ⏳ Blocked | `make test-bdd-mobile` | Blocked by Android SDK env in CI |

Go stack is implemented first because:
- All business logic and agent runtime lives in Go
- Go BDD scenarios directly exercise domain services and tool execution
- BFF is a thin proxy (ADR-009) — BDD at the BFF level adds minimal value over Go tests
- Mobile E2E Detox requires a running Android emulator — not available in CI

### Feature file structure

18 `.feature` files in `features/`, one per UC:
- Business UCs: `uc-s1` through `uc-g1`, `uc-a1`
- AGENT_SPEC UCs: `uc-a2` through `uc-a9`
- System UCs: `uc-b1` (Safe Tool Routing)

### BDD metadata contract (TST items)

Every BDD scenario maps to a TST item in Doorstop (`reqs/TST/`). Required fields:

```yaml
bdd:
  feature: "features/uc-s1-sales-copilot.feature"
  scenario: "Launch Sales Copilot from account detail with grounded context"
  stack: go   # go | bff | mobile
  behavior: "execute_workflow"  # optional, AGENT_SPEC only
```

### Traceability chain

```
UC (reqs/UC/*.yml)
  → links to: FR (reqs/FR/*.yml)
    → links to: TST (reqs/TST/*.yml)
      → contains: bdd.feature + bdd.scenario
```

Validated by `cmd/frtrace` — enforced in CI via `make bdd-trace-check`.

### Scenario tags

Every scenario must include:
- `@UC-*` — use case ID
- `@FR-*` — one or more functional requirement IDs
- `@TST-*` — TST item ID
- `@stack-go` / `@stack-bff` / `@stack-mobile`
- `@behavior-*` — AGENT_SPEC scenarios only (e.g., `@behavior-execute_workflow`)

## Rationale

- Doorstop UC layer makes traceability machine-verifiable — `frtrace` catches broken links in CI
- Go-first BDD is pragmatic: covers the highest-value layer (business logic) first
- BFF/Mobile placeholders reserve the structure without blocking current CI
- TST metadata contract enables future automation (e.g., generate BDD reports from TST items)

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Implement all 3 stacks simultaneously | Mobile Detox blocked by Android SDK; BFF adds minimal value over Go |
| No traceability gate (free-form tags) | Tags drift; UC coverage becomes unverifiable |
| Use Cucumber for all stacks | Different tooling per stack (godog for Go, Jest for BFF, Detox for Mobile) — Cucumber would require shims |
| TST items without BDD metadata | Cannot automate coverage reporting; manual cross-reference only |

## Consequences

**Positive:**
- 33 BDD scenarios cover all 16 UCs in Go — full business logic coverage
- Traceability chain UC→FR→TST is CI-enforced — no silent coverage drift
- Placeholder structure means BFF/Mobile BDD can be added without restructuring

**Negative / tradeoffs:**
- Mobile BDD blocked until Android SDK is available in CI (tracked in `task_uc_gap_closure.md`)
- BFF BDD step definitions not yet written — 4 mobile UC gaps remain (P0/P1)

## References

- `features/` — 18 `.feature` files
- `tests/bdd/go/` — Go godog runner (`bdd_test.go`, `state.go`, `workflow_runtime_bdd.go`)
- `bff/tests/bdd/` — BFF placeholder
- `mobile/e2e/bdd/` — Mobile placeholder + `uc-s1-sales-copilot.e2e.ts`
- `reqs/UC/`, `reqs/FR/`, `reqs/TST/` — Doorstop requirement layers
- `cmd/frtrace/main.go` — traceability validator
- `docs/tasks/task_uc_gap_closure.md` — 8 pending UC gap closure tasks
