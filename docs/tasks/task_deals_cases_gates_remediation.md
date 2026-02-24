# Task: Deal/Case Gate Remediation

> **Date**: 2026-02-24  
> **Status**: Completed  
> **Related**: `docs/tasks/task_deals_cases_gap.md` (WS-1, WS-2)

---

## 1) Executive Report (Current State)

Gate run executed locally after WS-1/WS-2.

### PASS
- `make fmt`
- `make pattern-opportunities-gate PATTERN_GATE_MODE=warn PATTERN_GATE_TS_DUP_THRESHOLD=2`
- `make doorstop-check`
- `make trace-check`
- `make race-stability`
- `COVERAGE_MIN=79 make coverage-gate`
- `COVERAGE_APP_MIN=79 make coverage-app-gate`
- `TDD_COVERAGE_MIN=79 make coverage-tdd`
- `make build`
- `cd mobile && npm run typecheck`
- `cd mobile && npm run lint`
- `cd mobile && npm run quality:arch`
- `cd mobile && npm run test:coverage`

### FAIL
- `make complexity`
- `make lint`
- `make test`
- `make contract-test`
- `make contract-test-strict`

### Root Cause Summary
1. **Code issues (must fix in repo):**
   - Cyclomatic/cognitive complexity exceeded after adding list filter/sort logic in deal/case services.
   - Lint findings (`gocognit`, `goconst`) in new/updated files.
2. **Environment/sandbox issues (cannot fix in repo):**
   - `make test` fails in `internal/infra/llm` because `httptest.NewServer` cannot bind ports.
   - `contract-test` and `contract-test-strict` cannot start server on `0.0.0.0:8081`.

---

## 2) Scope of This Task

Fix all **code-owned** failures so the only remaining failures (if any) are environment/network bind restrictions.

---

## 3) Work Plan

## A. Complexity Refactor (Deal/Case services)

**Goal:** reduce complexity below configured thresholds.

### A.1 Deal service
- File: `internal/domain/crm/deal.go`
- Refactor:
  - extract query selection into helper (`selectDealRowsByFilter`)
  - extract sorting into helper (`sortDealsByCreatedAt`)
  - keep `List` and `listFiltered` orchestration thin

### A.2 Case service
- File: `internal/domain/crm/case.go`
- Refactor:
  - extract query selection into helper (`selectCaseRowsByFilter`)
  - extract priority filtering into helper (`filterCasesByPriority`)
  - extract sorting into helper (`sortCasesByCreatedAt`)
  - reduce branching in `List` and `listFiltered`

**Acceptance criteria:**
- `make complexity` passes.

---

## B. Lint Remediation

**Goal:** clear `make lint` for code findings introduced in WS-1.

### B.1 goconst cleanup
- Replace repeated literals with constants:
  - `-created_at`
  - `owner_id`
  - `account_id`
  - use existing `paramStageID` in deal handler

### B.2 gocognit cleanup
- Ensure extracted helpers bring `gocognit` below threshold in case/deal paths.

**Acceptance criteria:**
- `make lint` passes (excluding only sandbox cache warnings that do not fail exit code).

---

## C. Re-validation Sequence

Run in this order:

1. `make fmt`
2. `make complexity`
3. `make lint`
4. `GOCACHE=$(pwd)/.tmp/go-build go test ./internal/domain/crm ./internal/api/handlers`
5. `make race-stability`
6. coverage gates (`coverage-gate`, `coverage-app-gate`, `coverage-tdd`)
7. `make build`
8. mobile gates (`typecheck`, `lint`, `quality:arch`, `test:coverage`)

Then run full gates and classify residual failures:
- `make test`
- `make contract-test`
- `make contract-test-strict`

**Expected residual (environment):**
- port bind failures in sandbox for full test/contract gates.

---

## 4) Deliverables

1. Refactored deal/case service code with lower complexity.
2. Handler constants cleanup for lint compliance.
3. Updated task execution log in `task_deals_cases_gap.md` with:
   - commands run
   - PASS/FAIL matrix
   - explicit distinction between code failures and environment failures.

---

## 5) Exit Criteria

This remediation task is considered complete when:

1. `make complexity` passes.
2. `make lint` passes.
3. Targeted backend tests for WS-1 pass.
4. Mobile gates continue passing.
5. Any remaining gate failures are only sandbox/network bind restrictions and are reported explicitly.

---

## 6) Closure Evidence

Remediation closure validated in CI after pushing commit:
- Commit: `d6063077d98613e32fd43149651cf78a2217d191`
- Run: `https://github.com/krukmat/fenix/actions/runs/22357831459`
- Final result: `success`

Verified passing jobs:
- `Complexity Gate (max 7)`
- `Traceability Gate`
- `Mobile Quality Gates`
- `Lint and Test` (incluye race/coverage/build)
- `API Contract Tests` (strict)

Notes:
- Local sandbox still showed occasional friction for long-running race checks and port binding.
- Authoritative merge gate status is the GitHub Actions run above, which passed end-to-end.
