---
doc_type: plan
id: GO-COMPLEXITY-PREPUSH-PLAN
title: "Go pre-push complexity unblock plan"
status: completed
phase: qa-hardening
week: 19
tags: [plan, go, qa, complexity, pre-push]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - internal/domain/crm/deal.go
  - internal/domain/crm/case.go
  - internal/domain/agent/dsl_statement_trace.go
  - internal/domain/agent/skill_runner.go
  - internal/domain/agent/runtime_steps.go
  - docs/go_complexity_prepush_blockers_audit.md
created: 2026-05-05
completed:
---

# Go pre-push complexity unblock plan

## Objective

Reduce the four current `gocyclo` violations to `<= 7` without changing
runtime behavior, then rerun the full Go pre-push QA pipeline successfully.

## Trigger

`bash scripts/qa-go-prepush.sh` is currently blocked at `make complexity`
despite a clean global `wrapcheck` result.

## Scope

Target functions:

- `internal/domain/crm/deal.go`: `(*DealService).selectDealRowsByFilter`
- `internal/domain/crm/case.go`: `(*CaseService).selectCaseRowsByFilter`
- `internal/domain/agent/dsl_statement_trace.go`: `updateDSLRunStep`
- `internal/domain/agent/skill_runner.go`: `updateBridgeRunStep`

Out of scope:

- Changing the complexity threshold
- Suppressing or bypassing the gate
- Broad refactors outside the four blocking functions unless required by tests

---

## Step 1 — Remove duplicated agent runtime step-update complexity ✅ DONE

**Status:** completed 2026-05-05

**What was done:**
- Extracted `errTextFromError(err error) *string` helper to `runtime_steps.go`
- Extracted `commitRunStepUpdate(ctx, db, workspaceID, stepID, status, output, stepErr, beginPrefix, commitPrefix)` helper to `runtime_steps.go`
- Rewrote `updateDSLRunStep` as a 6-line thin wrapper — complexity 8 → **4**
- Rewrote `updateBridgeRunStep` as a 6-line thin wrapper — complexity 8 → **4**

**Files changed:**
- `internal/domain/agent/runtime_steps.go` — added helpers at lines 721–754
- `internal/domain/agent/dsl_statement_trace.go` — thin wrapper
- `internal/domain/agent/skill_runner.go` — thin wrapper

**Verification:**
- `make complexity` — agent blockers cleared
- `go test ./internal/domain/agent/...` — ok (4.0s + 1.6s)

---

## Step 2 — Refactor the case selector dispatch ✅ DONE

**Status:** completed 2026-05-05

**Strategy:**
Extract 4 branch-specific helpers as package-private methods on `*CaseService`.
Keep `selectCaseRowsByFilter` as a `switch`-based dispatcher.

**Helpers to extract:**
- `listCasesByAccount(ctx, workspaceID, accountID) ([]sqlcgen.CaseTicket, error)`
- `listCasesByOwner(ctx, workspaceID, ownerID) ([]sqlcgen.CaseTicket, error)`
- `listCasesByStatus(ctx, workspaceID, status) ([]sqlcgen.CaseTicket, error)`
- `listCasesByWorkspaceAll(ctx, workspaceID) ([]sqlcgen.CaseTicket, error)`

**Dispatcher target shape:**
```go
// complexity: 1 + 3 case + 1 default = 5
switch {
case input.AccountID != "": return s.listCasesByAccount(...)
case input.OwnerID != "":   return s.listCasesByOwner(...)
case input.Status != "":    return s.listCasesByStatus(...)
default:                    return s.listCasesByWorkspaceAll(...)
}
```

**Behavioral constraints:**
- Preserve filter precedence: AccountID → OwnerID → Status → workspace fallback
- `Priority` filtering stays in `listFiltered`, NOT in the selector
- Preserve workspace-wide fallback via `CountCasesByWorkspace` + `ListCasesByWorkspace`
- Preserve all `fmt.Errorf` wrapper strings exactly

**Expected complexity after:** `selectCaseRowsByFilter` → **5**

**Files affected:**
- `internal/domain/crm/case.go`

**Verification:**
- `make complexity`
- `go test ./internal/domain/crm/...`

---

## Step 3 — Refactor the deal selector dispatch ✅ DONE

**Status:** completed 2026-05-05

**Strategy:**
Extract 6 branch-specific helpers as package-private methods on `*DealService`.
Keep `selectDealRowsByFilter` as a `switch`-based dispatcher.

**Helpers to extract:**
- `listDealsByStage(ctx, workspaceID, stageID) ([]sqlcgen.Deal, error)`
- `listDealsByPipeline(ctx, workspaceID, pipelineID) ([]sqlcgen.Deal, error)`
- `listDealsByAccount(ctx, workspaceID, accountID) ([]sqlcgen.Deal, error)`
- `listDealsByOwner(ctx, workspaceID, ownerID) ([]sqlcgen.Deal, error)`
- `listDealsByStatus(ctx, workspaceID, status) ([]sqlcgen.Deal, error)`
- `listDealsByWorkspaceAll(ctx, workspaceID) ([]sqlcgen.Deal, error)`

**Dispatcher target shape:**
```go
// complexity: 1 + 5 case + 1 default = 7  (exactly at threshold)
switch {
case input.StageID != "":    return s.listDealsByStage(...)
case input.PipelineID != "": return s.listDealsByPipeline(...)
case input.AccountID != "":  return s.listDealsByAccount(...)
case input.OwnerID != "":    return s.listDealsByOwner(...)
case input.Status != "":     return s.listDealsByStatus(...)
default:                     return s.listDealsByWorkspaceAll(...)
}
```

**Behavioral constraints:**
- Preserve filter precedence: StageID → PipelineID → AccountID → OwnerID → Status → workspace fallback
- Preserve workspace-wide fallback via `CountDealsByWorkspace` + `ListDealsByWorkspace`
- Preserve all `fmt.Errorf` wrapper strings exactly

**Risk note:** The dispatcher sits exactly at complexity 7 after refactor. Any future
filter branch requires sub-dispatching or a new extraction to stay within the gate.

**Expected complexity after:** `selectDealRowsByFilter` → **7**

**Files affected:**
- `internal/domain/crm/deal.go`

**Verification:**
- `make complexity`
- `go test ./internal/domain/crm/...`
- `bash scripts/qa-go-prepush.sh` (final full pipeline run)

---

## Verification sequence

1. `make complexity`
2. `go test ./internal/domain/agent/... ./internal/domain/crm/...`
3. `bash scripts/qa-go-prepush.sh`

## Exit criteria

- All four functions are at or below complexity `7`
- No behavior changes are introduced
- `bash scripts/qa-go-prepush.sh` passes end to end

## Supporting analysis

See `docs/go_complexity_prepush_blockers_audit.md` and
`docs/tasks/task_go_complexity_prepush_unblock.md`.
