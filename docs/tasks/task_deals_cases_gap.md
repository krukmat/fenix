# Deal/Case Gap Implementation Plan (P0)

> **Date**: 2026-02-24  
> **Status**: Completed  
> **Scope**: Close remaining gaps for Deal/Case listing, creation, and modification across Backend, BFF, Mobile, and tests.
> **Gate remediation task**: `docs/tasks/task_deals_cases_gates_remediation.md`

---

## 1. Objective

Deliver full, test-backed, traceable support for:

- `deals`: list + create + update
- `cases`: list + create + update

for API and mobile flows, aligned with requirements, architecture, and implementation plan.

---

## 0. Execution Log

### 2026-02-24 — WS-1 Started

Status: ✅ Implemented and locally validated (targeted test scope).

Completed:
- Deal list contract in backend:
  - Filters: `status`, `owner_id`, `account_id`, `pipeline_id`, `stage_id`
  - Sort: `created_at` / `-created_at`
  - Validation: reject ambiguous multi-filter combinations
- Case list contract in backend:
  - Filters: `status`, `priority`, `owner_id`, `account_id`
  - Sort: `created_at` / `-created_at`
  - Validation: reject ambiguous multi-filter combinations
- Handler tests added for:
  - valid filtering path
  - invalid multi-filter path (400)
  - update not found path (404)
- Domain service tests added for filtered list behavior.
- OpenAPI updated with query params for `GET /api/v1/deals` and `GET /api/v1/cases`.

Validation executed:
- `GOCACHE=$(pwd)/.tmp/go-build go test ./internal/domain/crm ./internal/api/handlers` ✅

### 2026-02-24 — WS-2 Started

Status: ✅ Implemented and locally validated (targeted mobile scope).

Completed:
- `mobile/src/services/api.ts`:
  - Added `createDeal`, `updateDeal`, `createCase`, `updateCase`.
- `mobile/src/hooks/useCRM.ts`:
  - Added `useCreateDeal`, `useUpdateDeal`, `useCreateCase`, `useUpdateCase`.
  - Added cache invalidation for list/detail query keys on successful mutation.
- Tests:
  - `mobile/__tests__/services/api.test.ts` updated for deal/case mutation endpoints.
  - `mobile/__tests__/hooks/useCRM.test.ts` updated for mutation hooks + invalidation behavior.

Validation executed:
- `cd mobile && npm test -- --runTestsByPath __tests__/services/api.test.ts __tests__/hooks/useCRM.test.ts` ✅
- `cd mobile && npm run lint` ✅

### 2026-02-24 — WS-3 Started

Status: ✅ Implemented and locally validated (mobile UI scope).

Completed:
- Added create screens:
  - `mobile/app/(tabs)/deals/new.tsx`
  - `mobile/app/(tabs)/cases/new.tsx`
- Added edit screens:
  - `mobile/app/(tabs)/deals/edit/[id].tsx`
  - `mobile/app/(tabs)/cases/edit/[id].tsx`
- Added list entry points (FAB):
  - deals list -> `/deals/new`
  - cases list -> `/cases/new`
- Added detail entry points (edit CTA):
  - deal detail -> `/deals/edit/[id]`
  - case detail -> `/cases/edit/[id]`
- Added WS-3 form validation tests:
  - `mobile/__tests__/components/dealsCasesForms.test.ts`

Validation executed:
- `cd mobile && npm run typecheck` ✅
- `cd mobile && npm run lint` ✅
- `cd mobile && npm test -- --runTestsByPath __tests__/components/dealsCasesForms.test.ts __tests__/components/dealsListScreen.test.tsx __tests__/components/caseDetailScreen.test.tsx` ✅

### 2026-02-24 — WS-4 Completed

Status: ✅ Completed with CI validation in GitHub.

Completed:
- Added and linked Doorstop TST artifacts:
  - `reqs/TST/TST047.yml`
  - `reqs/TST/TST048.yml`
  - `reqs/TST/TST049.yml`
  - `reqs/TST/TST050.yml`
- Added `Traces: FR-001` annotation in mobile tests introduced for WS-3.
- Completed quality gates in CI on pushed commit `d6063077d98613e32fd43149651cf78a2217d191`.

Validation executed:
- Local:
  - `make doorstop-check` ✅
  - `make trace-check` ✅
  - `cd mobile && npm run quality` ✅
- GitHub Actions (authoritative):
  - Run: `https://github.com/krukmat/fenix/actions/runs/22357831459` ✅
  - Jobs:
    - `Complexity Gate (max 7)` ✅
    - `Traceability Gate` ✅
    - `Mobile Quality Gates` ✅
    - `Lint and Test` (incluye race/coverage/build) ✅
    - `API Contract Tests` (strict) ✅

---

## 2. Traceability (Requirements → Deliverables)

| Requirement | Coverage in this plan |
|---|---|
| FR-001 (Core CRM Entities) | CRUD behavior with explicit L-C-U acceptance for Deal/Case |
| FR-002 (Pipelines and Stages) | Deal/Case update flows preserve stage/pipeline consistency |
| FR-051 (Timeline) | Timeline events verified on create/update |
| FR-070 (Audit Trail) | Audit entries verified for create/update API mutations |
| FR-300 (Mobile App) | Deal/Case create/edit screens + list/detail refresh behavior |
| FR-301 (BFF Gateway) | Proxy behavior for Deal/Case L-C-U paths and query params |

---

## 3. Current Gap (As-Is)

1. Backend list endpoints for deals/cases are paginated, but lack explicit filter/sort contract handling in handler/service.
2. Mobile app has list/detail screens for deals/cases but no create/edit screens or mutation hooks.
3. API client (`mobile/src/services/api.ts`) exposes list/detail methods, but no deal/case create/update methods.
4. Tests do not yet cover end-to-end L-C-U flows for deals/cases at contract and mobile levels.

---

## 4. Implementation Workstreams

### WS-1 Backend API Contract Hardening (Deal/Case List + Create + Update)

**Goal**: Ensure list endpoints support explicit filters/sort and creation/update behavior is contract-tested.

**Files (planned):**
- `internal/api/handlers/deal.go`
- `internal/api/handlers/case.go`
- `internal/domain/crm/deal.go`
- `internal/domain/crm/case.go`
- `internal/infra/sqlite/queries/deal.sql`
- `internal/infra/sqlite/queries/case.sql`
- `docs/openapi.yaml`
- `internal/api/handlers/deal_test.go`
- `internal/api/handlers/case_test.go`
- `internal/domain/crm/deal_test.go`
- `internal/domain/crm/case_test.go`

**Tasks:**
1. Define allowed query params for list:
   - Deals: `status`, `owner_id`, `account_id`, `pipeline_id`, `stage_id`, `sort`
   - Cases: `status`, `priority`, `owner_id`, `account_id`, `sort`
2. Add deterministic sort options (at least `created_at desc` default, and one alternate key).
3. Validate/normalize list query params in handlers.
4. Ensure service/query path applies workspace isolation + filters.
5. Add contract tests for:
   - list with pagination/filter/sort
   - create happy/validation paths
   - update happy/not-found paths
6. Confirm timeline + audit side effects in tests for create/update.

**Done when:**
- `GET /api/v1/deals` and `GET /api/v1/cases` pass filter/sort contract tests.
- `POST/PUT` for deals/cases pass validation and mutation tests.
- OpenAPI documents list params and update/create request behavior.

---

### WS-2 Mobile Data Layer (Mutations + Cache Invalidation)

**Goal**: Expose create/update operations and keep list/detail cache coherent.

**Files (planned):**
- `mobile/src/services/api.ts`
- `mobile/src/hooks/useCRM.ts`
- `mobile/__tests__/services/api.test.ts`
- `mobile/__tests__/hooks/useCRM.test.ts`

**Tasks:**
1. Add API methods:
   - `createDeal`, `updateDeal`
   - `createCase`, `updateCase`
2. Add mutation hooks in `useCRM`:
   - `useCreateDeal`, `useUpdateDeal`
   - `useCreateCase`, `useUpdateCase`
3. Invalidate/refetch related keys after mutation:
   - list key (`deals`/`cases`)
   - detail key (`deal`/`case`) when applicable
4. Add unit tests for payload mapping and cache invalidation behavior.

**Done when:**
- Mobile can call create/update endpoints through BFF proxy.
- Query cache refreshes correctly after create/update.

---

### WS-3 Mobile UI (Create/Edit Screens + Navigation)

**Goal**: Add complete user flows for create/edit in deals/cases.

**Files (planned):**
- `mobile/app/(tabs)/deals/new.tsx`
- `mobile/app/(tabs)/deals/[id]/edit.tsx` or `mobile/app/(tabs)/deals/edit/[id].tsx`
- `mobile/app/(tabs)/cases/new.tsx`
- `mobile/app/(tabs)/cases/[id]/edit.tsx` or `mobile/app/(tabs)/cases/edit/[id].tsx`
- `mobile/app/(tabs)/deals/index.tsx`
- `mobile/app/(tabs)/deals/[id].tsx`
- `mobile/app/(tabs)/cases/index.tsx`
- `mobile/app/(tabs)/cases/[id].tsx`

**Tasks:**
1. Create Deal form:
   - required fields: `accountId`, `pipelineId`, `stageId`, `ownerId`, `title`
2. Edit Deal form:
   - business fields: status/stage/owner/amount/expected close/metadata
3. Create Case form:
   - required fields: `ownerId`, `subject`
4. Edit Case form:
   - business fields: status/priority/stage/owner/description/metadata
5. Add CTA entry points:
   - from list screen to `new`
   - from detail screen to `edit`
6. UX handling:
   - inline validation
   - loading/disabled submit state
   - error surface and success feedback
   - navigation back with refresh

**Done when:**
- User can execute `list -> create -> edit -> verify` for deals and cases in app.

---

### WS-4 Verification & Traceability Closure

**Goal**: Close traceability with tests and requirement links.

**Files (planned):**
- `reqs/TST/TST047.yml` (Deal list/create/update contract)
- `reqs/TST/TST048.yml` (Case list/create/update contract)
- `reqs/TST/TST049.yml` (Deal mobile create/edit flow)
- `reqs/TST/TST050.yml` (Case mobile create/edit flow)
- `docs/openapi.yaml` (`x-fr-traces` validation)

**Tasks:**
1. Add new TST artifacts linked to FR-001/002/051/070/300/301 where applicable.
2. Ensure backend and mobile tests include `Traces: FR-XXX`.
3. Run quality gateways required by current repo CI:
   - Go/backend gates:
     - `make fmt`
     - `make complexity`
     - `make pattern-opportunities-gate PATTERN_GATE_MODE=warn PATTERN_GATE_TS_DUP_THRESHOLD=2`
     - `make doorstop-check`
     - `make trace-check`
     - `make lint`
     - `make test`
     - `make race-stability`
     - `COVERAGE_MIN=79 make coverage-gate`
     - `COVERAGE_APP_MIN=79 make coverage-app-gate`
     - `TDD_COVERAGE_MIN=79 make coverage-tdd`
     - `make build`
     - `make contract-test` (and `make contract-test-strict` before release cut)
   - Mobile/BFF gates:
     - `cd mobile && npm run typecheck`
     - `cd mobile && npm run lint`
     - `cd mobile && npm run quality:arch`
     - `cd mobile && npm run test:coverage`
4. Resolve suspect/unreviewed requirement links caused by updated FR text.

**Done when:**
- No traceability regression for Deal/Case L-C-U flows.
- Requirement links and tests are consistent.

---

## 5. Execution Order

1. WS-1 Backend contract hardening  
2. WS-2 Mobile data layer  
3. WS-3 Mobile UI  
4. WS-4 Verification and traceability closure

Rationale: avoids UI work on unstable API semantics and minimizes rework.

---

## 6. Exit Criteria

1. Backend:
   - Deal/Case list supports documented pagination/filter/sort contract.
   - Deal/Case create/update endpoints validated and tested.
2. Mobile:
   - Dedicated create/edit flows for deals and cases.
   - List/detail cache coherence after mutations.
3. Quality:
   - Contract + unit/integration + mobile tests for both entities pass.
   - Traceability checks pass for requirements tied to these flows.

---

## 7. Risks and Mitigations

1. Risk: Filter/sort mismatch between OpenAPI and implementation.
   - Mitigation: update OpenAPI and handler tests in same change set.
2. Risk: Cache invalidation bugs after mutation.
   - Mitigation: mutation hook tests and explicit query key invalidation.
3. Risk: Form payload drift (`camelCase` vs backend schema).
   - Mitigation: typed request mappers + API service tests.

---

## 8. Quality Gateway Alignment

This plan is explicitly aligned to current CI gateways defined in:

- `Makefile` (`ci`, `doorstop-check`, `trace-check`, coverage gates, contract tests)
- `.github/workflows/ci.yml` (`mobile-quality`, `complexity`, `traceability`, `test`, `contract`)
- `scripts/pattern-refactor-gate.sh` (pattern/refactor gate in warn/strict modes)

No task in this plan is considered done unless the corresponding gateway set passes.
