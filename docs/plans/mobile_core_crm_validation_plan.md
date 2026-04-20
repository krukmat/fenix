---
doc_type: task
id: MOBILE-CORE-CRM-VALIDATION
title: "Mobile Core CRM Validation Plan"
status: planning
phase: crm-validation
week: ~
tags: [mobile, crm, bff, validation, no-ai]
fr_refs: [FR-010, FR-020, FR-030, FR-040, FR-050]
uc_refs: [UC-C1]
blocked_by: [PHASE-CRM-VALIDATION]
blocks: []
files_affected:
  - mobile/src/services/api.ts
  - mobile/src/services/api.crm.ts
  - mobile/src/services/api.crm.endpoints.ts
  - mobile/src/services/api.crm.types.ts
  - mobile/src/services/api.crm.normalizers.ts
  - mobile/src/hooks/useCRM.ts
  - mobile/src/hooks/useCRM.keys.ts
  - mobile/src/hooks/useCRM.full.ts
  - mobile/src/hooks/useCRM.entityMutations.ts
  - mobile/app/(tabs)/crm
  - mobile/src/components/crm/CoreCRMReadOnly.tsx
  - mobile/src/components/crm/CoreCRMListViews.tsx
  - mobile/src/components/crm/CoreCRMDetailViews.tsx
  - mobile/__tests__/app/(tabs)/crm/read-only.test.tsx
  - mobile/__tests__/services/api.crm.endpoints.test.ts
  - mobile/__tests__/services/api.crm.normalizers.test.ts
  - mobile/__tests__/hooks/useCRM.test.ts
created: 2026-04-19
completed: null
parent_plan: docs/plans/phase_core_crm_validation.md
---

# Mobile Core CRM Validation Plan

## Summary

The backend CRM validation phase is documented in `docs/plans/phase_core_crm_validation.md` and explicitly kept Mobile/BFF out of scope. This plan defines the mobile follow-up: validate the same no-AI CRM surface through the existing BFF proxy and mobile app, without adding Copilot, agent runtime, approvals, usage metering, or other AI workflows.

The mobile phase should prove that core CRM entities can be listed, opened, created, updated, and deleted from mobile where the backend supports those operations. It should also preserve the current wedge routes while making `/crm/*` a real validation surface instead of redirects to Sales/Support-only screens.

## Current Baseline

- `mobile/src/services/api.crm.ts` supports CRM list/detail reads, `createAccount`, and Deal/Case create/update only.
- `mobile/src/hooks/useCRM.ts` exposes list/detail hooks plus Deal/Case mutations only.
- `/crm/*` routes mostly re-export legacy `/accounts`, `/deals`, `/cases`, and those legacy routes redirect to Sales/Support wedge screens.
- Existing tests cover API wrapper calls, query hooks, Sales/Support wedge lists, and Deal/Case form validation.
- Full mobile CRM CRUD parity is not yet covered.
- Backend timeline support is not uniform for every entity; mobile must render timeline when present and tolerate empty or missing timeline payloads.

## Frozen Mobile CRM Contract

**Task 1 status**: completed on 2026-04-19.

Source of truth:
- Backend parent plan: `docs/plans/phase_core_crm_validation.md`.
- Backend route registry: `internal/api/routes.go`.
- Backend integration coverage: `internal/api/integration_crm_test.go`.
- BFF pass-through: `bff/src/routes/proxy.ts` forwards all regular `/bff/api/v1/*` requests to Go `/api/v1/*`.

Mobile must call the BFF prefix, not the Go API directly. The mobile path is always `/bff/api/v1/...`; the equivalent backend path below is shown only to document the underlying contract.

| Entity | Mobile BFF routes | Required mobile operations | Backend validation coverage | Notes |
|--------|-------------------|----------------------------|-----------------------------|-------|
| Account | `/bff/api/v1/accounts`, `/bff/api/v1/accounts/{id}` | list, detail, create, update, delete | create/get, update, list, soft-delete | Root dependency for Contacts and Deals. |
| Account contacts | `/bff/api/v1/accounts/{account_id}/contacts` | list by Account | list by Account | Relationship view only; creation remains through `/contacts`. |
| Contact | `/bff/api/v1/contacts`, `/bff/api/v1/contacts/{id}` | list, detail, create, update, delete | create/get, update, list by Account, invalid Account, soft-delete | Requires valid `accountId` for linked contact validation. |
| Lead | `/bff/api/v1/leads`, `/bff/api/v1/leads/{id}` | list, detail, create, update, delete | create/get, status transitions, soft-delete | Standalone CRM entity. |
| Pipeline | `/bff/api/v1/pipelines`, `/bff/api/v1/pipelines/{id}` | list, detail, create/update/delete only if needed for validation fixtures | create/get, list | Mobile forms primarily need read/select behavior. |
| Pipeline Stage | `/bff/api/v1/pipelines/{id}/stages`, `/bff/api/v1/pipelines/stages/{stage_id}` | list by Pipeline; create/update/delete only if fixtures cannot pre-seed stages | create/list, update, delete | Deal creation requires a valid stage. |
| Deal | `/bff/api/v1/deals`, `/bff/api/v1/deals/{id}` | list, detail, create, update, delete | create/get, update stage/status, nullable contact, soft-delete | Requires Account, Pipeline, and Stage. Contact is optional. |
| Case | `/bff/api/v1/cases`, `/bff/api/v1/cases/{id}` | list, detail, create, update, delete | create/get, status transitions, standalone no Account/Contact, soft-delete | Account and Contact are optional. |
| Activity | `/bff/api/v1/activities`, `/bff/api/v1/activities/{id}` | list, detail, create, update, delete | create on Account, create on Case, status update, soft-delete | Polymorphic via entity type/id. |
| Note | `/bff/api/v1/notes`, `/bff/api/v1/notes/{id}` | list, detail, create, update, delete | create on Account, internal flag on Case, soft-delete | Polymorphic via entity type/id. |
| Attachment | `/bff/api/v1/attachments`, `/bff/api/v1/attachments/{id}` | list, detail, create metadata, delete | create/get, list by entity, delete | Metadata-only validation; no binary upload scope in this phase. |
| Timeline | `/bff/api/v1/timeline`, `/bff/api/v1/timeline/{entity_type}/{entity_id}` | list workspace events, list entity events, render when present | Case create/update/delete timeline events | Read-only from mobile. Timeline events are not guaranteed for every entity. |

Contract exclusions:
- AI/Copilot chat and Sales Brief.
- Agent runtime, agent triggers, handoff flows, and approvals.
- Governance, audit drilldowns, quota/usage metering, and policy administration.
- Workflow authoring/execution.
- Binary attachment upload and download.
- Backend timeline wiring fixes for entities that currently do not emit timeline events.

Contract rules:
- Mobile must preserve existing `/sales` and `/support` wedge surfaces; this plan only restores `/crm/*` as a broad no-AI validation surface.
- Mobile list calls should support the pagination shape already used by the backend, including `limit`/`offset` or existing page wrappers where current mobile code already depends on them.
- Mobile detail screens must tolerate missing related resources and optional timeline arrays.
- Form validation should enforce required fields before submit, but backend validation remains authoritative.
- Delete operations are expected to behave as soft-delete from the user's perspective: deleted records should disappear from list/detail retrieval.

## Implementation Tasks In Dependency Order

1. **Freeze the mobile CRM contract**
   - Use backend integration tests and route registration as the source of truth.
   - Keep scope to Account, Contact, Lead, Pipeline/Stages, Deal, Case, Activity, Note, Attachment, and Timeline.
   - Keep AI, Copilot, approvals, governance, and usage out of this validation phase.

2. **Add shared mobile CRM types and normalizers**
   - Status: completed on 2026-04-19.
   - Define typed contracts for Account, Contact, Lead, Pipeline, PipelineStage, Deal, Case, Activity, Note, Attachment, TimelineEvent, and paginated CRM responses.
   - Normalize backend snake_case and existing mobile camelCase variants where needed.
   - Normalize list responses so hooks consume a stable `{ data, total/meta }` shape.
   - Treat timeline as optional.

3. **Complete `crmApi` endpoint coverage**
   - Status: completed on 2026-04-19.
   - Add Account CRUD: get, list, create, update, delete.
   - Add Contact CRUD plus account-scoped contact listing.
   - Add Lead CRUD.
   - Complete Deal and Case delete methods.
   - Add Pipeline and Stage list/read methods needed for Deal forms.
   - Add Activity, Note, and Attachment list/create/update/delete methods where backend supports them.
   - Add Timeline list-by-entity method.
   - Keep all mobile calls under the existing `/bff/api/v1/...` proxy prefix.

4. **Complete `useCRM` query and mutation hooks**
   - Status: completed on 2026-04-19.
   - Add query keys for pipelines, stages, activities, notes, attachments, and timeline.
   - Add create/update/delete hooks for Account, Contact, Lead, Deal, and Case.
   - Add Activity/Note/Attachment mutation hooks for entity detail screens.
   - Invalidate affected list, detail, relationship, and timeline queries after successful mutations.
   - Preserve workspace isolation in every query key.

5. **Replace `/crm/*` shims with real core CRM screens**
   - Status: completed on 2026-04-19.
   - Keep Sales and Support wedge routes intact.
   - Make `/crm` the broad core CRM validation surface.
   - Implement real read-only CRM lists for Accounts, Contacts, Leads, Deals, and Cases.
   - Add loading, empty, error, retry, refresh, and pagination states.
   - Add read-only detail screens that show primary fields and available related records.
   - Do not add create/edit/delete UI in this task; those flows belong to Task 6.

6. **Implement mobile CRM forms in dependency order**
   - Keep this task split into independent form waves so most implementation passes can run with `reasoning_effort: medium`.
   - Wave 6A: Account create/edit forms (`reasoning_effort: medium`) completed on 2026-04-19, because Account is the root standalone CRM entity.
   - Wave 6B: Lead create/edit forms (`reasoning_effort: medium`) completed on 2026-04-20, because Lead is standalone and does not need relationship selectors.
   - Wave 6C: Contact create/edit forms (`reasoning_effort: medium`) completed on 2026-04-20, because Contact only needs Account selection.
   - Wave 6D: Case create/edit forms (`reasoning_effort: medium`) completed on 2026-04-20, because Account and Contact are optional and can be validated independently.
   - Wave 6E1: Deal selector foundation (`reasoning_effort: medium`) completed on 2026-04-20: Account, optional Contact, Pipeline, and Stage data loading/selection helpers only.
   - Wave 6E2: Deal create form (`reasoning_effort: medium`) completed on 2026-04-20: create route, required-field validation, submit payload, and create tests using 6E1 helpers.
   - Wave 6E3: Deal edit form (`reasoning_effort: medium`) completed on 2026-04-20: edit route, existing Deal hydration, update payload, and edit tests using 6E1 helpers.
   - Wave 6F: Activity, Note, and Attachment create metadata forms (`reasoning_effort: medium`) completed on 2026-04-20: embedded entity-detail forms, backend-aligned Activity/Attachment payload keys, and metadata-only attachment validation.
   - Pipeline/Stage management remains read/select unless Deal validation proves fixture creation is required from mobile.
   - Do not combine 6E1-6E3 in one implementation pass; combining Deal selector foundation, create, and edit raises the Deal work back to `high`.
   - Do not combine 6A-6F in one implementation pass; combining multiple selector-heavy forms can raise the task back to `high` or `xhigh`.

7. **Add mobile functional tests**
   - Completed on 2026-04-20 with focused mobile functional coverage.
   - API wrapper tests cover every new `crmApi` method and exact BFF path/payload.
   - Hook tests cover query keys, enabled behavior, pagination, and mutation invalidation.
   - Screen tests cover CRM hub navigation, list states, detail rendering, and embedded child forms.
   - Form tests cover required fields, submit success, submit failure for child mutations, and dependency selectors.
   - Regression tests prove `/crm/accounts`, `/crm/deals`, and `/crm/cases` no longer redirect to wedge-only routes.

8. **Add mobile E2E/UAT validation path**
   - Keep this task split into independent waves so each implementation pass can run with `reasoning_effort: medium`.
   - Wave 8A: Audit and reuse existing seed/E2E entry points (`reasoning_effort: medium`) completed on 2026-04-20.
     - Reuse `scripts/e2e_seed_mobile_p2.go` as the primary deterministic runner seed because `mobile/maestro/seed-and-run.sh` already consumes its auth/session output and exports seeded CRM IDs to Maestro.
     - Reuse `mobile/e2e/helpers/seed.helper.ts` for Detox context if Detox is revived; current `mobile/e2e/accounts.e2e.ts`, `deals.e2e.ts`, and `cases.e2e.ts` are skipped and still target older wedge/legacy testIDs.
     - Keep `scripts/seed_uat_mobile.mjs` as a manual BFF/API UAT helper, not the first automated path, because it uses HTTP endpoints and does not currently feed the Maestro bootstrap/export contract.
     - Existing Maestro authenticated audit already seeds and exports Account, Contact, Lead, Deal, Stale Deal, Case, and Resolved Case IDs, but it does not yet exercise `/crm/*` routes.
     - Gap for 8B: expose Pipeline/Stage IDs from the primary Go seed if mutation UAT needs Deal creation; otherwise read-only CRM smoke can proceed with the existing exported IDs.
   - Wave 8B: Add or extend deterministic CRM seed output (`reasoning_effort: medium`) completed on 2026-04-20: `scripts/e2e_seed_mobile_p2.go` now exposes Account, Contact, Pipeline, Stage, Deal, Stale Deal, standalone Case, Resolved Case, and Lead IDs; `mobile/maestro/seed-and-run.sh` exports `SEED_PIPELINE_ID` and `SEED_STAGE_ID`; no UI flow changes in this wave.
   - Wave 8C: Add one core CRM smoke path (`reasoning_effort: medium`) completed on 2026-04-20: Maestro authenticated audit now validates `/crm` hub -> Accounts list -> seeded Account detail with stable testIDs and authenticated bootstrap; create/update remains out of this wave.
   - Wave 8D: Add one mutation UAT path (`reasoning_effort: medium`) completed on 2026-04-20: `mobile/maestro/crm-mutation-case.yaml` creates a Case ("UAT Case Wave 8D"), waits for router to return to `/crm/cases`, and asserts the subject text appears in the refreshed list; flow is called as a Maestro sub-flow at the end of `authenticated-audit.yaml` (section 11) so it can be isolated without touching the 8C smoke path.
   - Keep screenshot-only validation separate unless the Maestro screenshot runner already guarantees stable seeded CRM state.
   - Do not combine 8A-8D in one implementation pass; combining seed changes, runner orchestration, navigation smoke, and mutation validation raises the task back to `high`.

9. **Run required mobile QA gates**
   - `bash scripts/check-no-inline-eslint-disable.sh`
   - `cd mobile && npm run typecheck`
   - `cd mobile && npm run lint`
   - `cd mobile && npm run quality:arch`
   - `cd mobile && npm run test:coverage`
   - Preferred shortcut: `bash scripts/qa-mobile-prepush.sh`
   - If Go/BFF files are touched while fixing contract mismatches, also run the relevant local Go/BFF gates before push.

## Task Sizing And Reasoning Effort

| Task | Complexity | Recommended `reasoning_effort` | Depends on | Notes |
|------|------------|--------------------------------|------------|-------|
| 1. Freeze the mobile CRM contract | Medium | `medium` | Backend validation plan | Requires reading backend tests/routes and locking scope, but no code design yet. |
| 2. Add shared mobile CRM types and normalizers | High | `high` | Task 1 | Cross-cutting contract work; mistakes propagate into API, hooks, screens, and tests. |
| 3. Complete `crmApi` endpoint coverage | High | `high` | Task 2 | Broad API surface with payload-shape risk and BFF path consistency requirements. |
| 4. Complete `useCRM` query and mutation hooks | High | `high` | Tasks 2-3 | Query-key isolation and invalidation must be correct across related entities. |
| 5. Replace `/crm/*` shims with real core CRM screens | High | `high` | Tasks 2-4 | Scope reduced to read-only CRM lists/details; forms and destructive UI stay in Task 6. |
| 6. Implement mobile CRM forms in dependency order | High | `medium` per wave | Tasks 2-5 | Split into waves 6A-6F, with Deal further split into 6E1-6E3 so no single pass owns all dependent selectors and mutations. |
| 7. Add mobile functional tests | High | `medium` | Tasks 3-6 | Completed on 2026-04-20. Downgraded from `high` because prior waves already added most coverage and this pass only audited gaps plus child mutation failure coverage. |
| 8. Add mobile E2E/UAT validation path | High | `medium` per wave | Tasks 5-7 | Split into waves 8A-8D: seed audit, deterministic CRM seed output, read-only smoke path, and one mutation UAT path. Escalate to `high` only if multiple waves are combined or the runner/emulator fails in a non-local way. |
| 9. Run required mobile QA gates | Medium | `medium` | Tasks 1-8 | Execution-focused; escalate only if gates expose cross-layer failures. |

No remaining task requires `high` or `xhigh` if the implementation follows the wave boundaries above. Task 6 becomes `high` or `xhigh` only if multiple form waves are combined into one monolithic pass.

## Test Scenarios

- Account: create, list, detail, edit, delete, and confirm deleted records disappear from lists.
- Contact: create under Account, list all contacts, list contacts for Account, edit, and delete.
- Pipeline/Stage: list pipelines and stages; stage selection works for Deal forms.
- Deal: create with Account/Pipeline/Stage, edit status/stage, resolve linked Account/Contact when present, and delete.
- Case: create standalone, create linked to Account/Contact, edit status/priority, filter/search list, and delete.
- Lead: create standalone, edit status, list, detail, and delete.
- Activity/Note/Attachment: create linked records and verify entity details refresh.
- Timeline: render events when backend returns them and render a stable empty state when no events exist.
- Navigation: `/crm/*` routes stay inside core CRM while `/sales` and `/support` wedge behavior remains unchanged.

## Assumptions

- Mobile uses the existing BFF proxy URL pattern and does not call backend `/api/v1` directly.
- This phase validates product CRM functionality, not AI or wedge workflows.
- Backend timeline gaps for Account/Contact are not fixed in this mobile phase; mobile handles absent timeline gracefully.
- Pipeline/Stage management is read/select in mobile unless Deal creation cannot be validated without mobile-side stage creation.
- This plan is separate from the backend validation plan so the completed backend work and new mobile work remain clearly scoped.

## Validation For This Document

- The document has YAML frontmatter with allowed `doc_type: task`.
- The document references `docs/plans/phase_core_crm_validation.md` as its parent plan.
- Because this change is documentation-only and does not touch `mobile/`, mobile QA gates are not required for this commit.
