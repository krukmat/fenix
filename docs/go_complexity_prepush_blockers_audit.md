---
doc_type: audit
id: GO-COMPLEXITY-AUDIT-2026-05-05
title: "Go pre-push complexity blockers audit"
status: active
phase: qa-hardening
week: 19
tags: [audit, go, qa, complexity, pre-push]
related_tasks: [GO-COMPLEXITY-PREPUSH-01]
created: 2026-05-05
updated: 2026-05-05
---

# Go pre-push complexity blockers audit

## Context

After the repository-wide `wrapcheck` remediation completed, the required Go
pre-push QA script still blocked commit/push at the `complexity` gate.

The failing gate is governed by `ADR-006`, which enforces a cyclomatic
complexity threshold of `7` for production Go code under `internal/`.

## Evidence

Command run:

```bash
bash scripts/qa-go-prepush.sh
```

Original observed blockers:

- `internal/domain/crm/deal.go`: `(*DealService).selectDealRowsByFilter` — `13`
- `internal/domain/crm/case.go`: `(*CaseService).selectCaseRowsByFilter` — `9`
- `internal/domain/agent/dsl_statement_trace.go`: `updateDSLRunStep` — `8`
- `internal/domain/agent/skill_runner.go`: `updateBridgeRunStep` — `8`

## Current gate status (2026-05-05)

| Function | Complexity before | Complexity after | Status |
|---|---|---|---|
| `updateDSLRunStep` | 8 | **4** | ✅ cleared |
| `updateBridgeRunStep` | 8 | **4** | ✅ cleared |
| `(*CaseService).selectCaseRowsByFilter` | 9 | — | ⏳ pending |
| `(*DealService).selectDealRowsByFilter` | 13 | — | ⏳ pending |

## Function analysis

### 1. Deal selector hotspot

File: `internal/domain/crm/deal.go`

Function: `(*DealService).selectDealRowsByFilter` — complexity **13**

Why it exceeds the gate:

- Serial `if` chain with 5 filter branches + count/list workspace fallback.
- Each branch has its own `if err != nil` check.
- gocyclo counts: 1 (entry) + 5×2 (branch + error check) + 2 (fallback x2 errors) = 13.

Behavioral constraints:

- Preserve the current priority order: `StageID`, `PipelineID`, `AccountID`,
  `OwnerID`, `Status`, then workspace-wide fallback.
- Preserve current wrapper wording and list semantics.
- Preserve workspace-wide fallback through `CountDealsByWorkspace` +
  `ListDealsByWorkspace` (no hardcoded limit).

Refactor seam:

- Extract 6 helpers: `listDealsByStage`, `listDealsByPipeline`,
  `listDealsByAccount`, `listDealsByOwner`, `listDealsByStatus`,
  `listDealsByWorkspaceAll`.
- Rewrite `selectDealRowsByFilter` as a `switch` dispatcher (5 case + default).

Target complexity: **7** (exactly at threshold — future filter branches require
sub-dispatching).

### 2. Case selector hotspot

File: `internal/domain/crm/case.go`

Function: `(*CaseService).selectCaseRowsByFilter` — complexity **9**

Why it exceeds the gate:

- Same structural pattern as the deal selector: serial filter dispatch plus a
  count-and-list fallback path.
- gocyclo counts: 1 + 3×2 (branch + error check) + 2 (fallback x2 errors) = 9.

Behavioral constraints:

- Preserve the current priority order: `AccountID`, `OwnerID`, `Status`, then
  workspace-wide fallback.
- Keep `Priority` filtering in `listFiltered`, not in the selector.
- Preserve current wrappers and result ordering behavior.
- Workspace fallback: `CountCasesByWorkspace` + `ListCasesByWorkspace` (no
  hardcoded limit).

Refactor seam:

- Extract 4 helpers: `listCasesByAccount`, `listCasesByOwner`,
  `listCasesByStatus`, `listCasesByWorkspaceAll`.
- Rewrite `selectCaseRowsByFilter` as a `switch` dispatcher (3 case + default).

Target complexity: **5** (comfortable headroom below threshold).

### 3. DSL step trace update hotspot ✅ CLEARED

File: `internal/domain/agent/dsl_statement_trace.go`

Function: `updateDSLRunStep` — was **8**, now **4**

Resolution (2026-05-05):

- Extracted `errTextFromError` and `commitRunStepUpdate` helpers to
  `internal/domain/agent/runtime_steps.go` (lines 721–754).
- `updateDSLRunStep` became: guard clause + `commitRunStepUpdate` call.
- Behavior preserved: nil/no-op semantics, transactional update, error text as
  `stepErr.Error()`, error wrapper wording identical.

### 4. Bridge step trace update hotspot ✅ CLEARED

File: `internal/domain/agent/skill_runner.go`

Function: `updateBridgeRunStep` — was **8**, now **4**

Resolution (2026-05-05):

- Reuses `commitRunStepUpdate` from `runtime_steps.go`.
- `updateBridgeRunStep` became: guard clause + `commitRunStepUpdate` call.
- Behavior preserved: nil/no-op semantics, transactional update, error text as
  `stepErr.Error()`, error wrapper wording identical.

## Execution plan

1. ✅ Refactor the duplicated agent runtime step-update logic (Step 1) — **done**
2. ⏳ Refactor `selectCaseRowsByFilter` (Step 2) — pending
3. ⏳ Refactor `selectDealRowsByFilter` (Step 3) — pending
4. Run `bash scripts/qa-go-prepush.sh` after Step 3 to confirm full pipeline passes.

## Verification plan

- `make complexity`
- `go test ./internal/domain/agent/... ./internal/domain/crm/...`
- `bash scripts/qa-go-prepush.sh`

## Risks

- Refactoring the selector functions can accidentally change filter precedence
  if `switch` case order does not match the original `if` chain order.
- The deal dispatcher lands exactly at complexity 7 — no future branches can be
  added without re-extraction.
- The complexity gate may reveal additional latent violations only after these
  four are reduced (monitor gate output after each step).

## Recommendation

Proceed with Steps 2 and 3 in order. Verify `make complexity` after each file
before running the full pipeline.
