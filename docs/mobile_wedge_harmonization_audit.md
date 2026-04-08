---
doc_type: audit
id: audit_mobile_wedge_harmonization_2026_04_08
title: Mobile Wedge Harmonization Audit
status: completed
date: 2026-04-08
created: 2026-04-08
updated: 2026-04-08
tags: [audit, mobile, wedge, obsidian]
---

# Mobile Wedge Harmonization Audit

## Scope

Audit of the implementation claimed by `docs/plans/mobile_wedge_harmonization_plan.md` against the current `mobile/`, `bff/`, and supporting API-facing code on 2026-04-08.

## Verdict

The harmonization is **partially implemented**, not fully complete.

The repo contains meaningful wedge-aligned work:

- five visible wedge tabs exist
- support and sales route families exist
- BFF inbox aggregation exists
- governance summary API and BFF proxy exist
- Maestro wedge-first seeding and screenshot flow exist

However, the audited runtime surface still misses or violates multiple exit criteria from the plan, so the implementation should not be documented as complete.

## Confirmed Gaps

### 1. Inbox surface is not implemented as the required unified wedge feed

The plan requires Inbox to be the default wedge landing surface with filter chips, deterministic ordering, approval actions, handoff routing, and signal detail entry.

Current implementation:

- `mobile/app/(tabs)/inbox/index.tsx` is only a placeholder title/subtitle screen
- no inbox hook is consumed there
- no approval, handoff, or signal cards are rendered
- no screen-level inbox tests were found under `mobile/__tests__/app/(tabs)/inbox`

Evidence:

- `mobile/app/(tabs)/inbox/index.tsx:1-18`
- plan requirement: `docs/plans/mobile_wedge_harmonization_plan.md` section 4.1 and exit criteria

### 2. Handoff accept flow still routes into legacy non-wedge destinations

The plan requires handed-off runs to be actionable from Inbox and Activity, with entity routing to wedge destinations and fallback to activity detail when no entity context exists.

Current implementation:

- `HandoffBanner` pushes entity handoffs to `/(tabs)/crm/...`
- no-entity fallback pushes to `/(tabs)/copilot`
- the test suite explicitly asserts those legacy routes

Evidence:

- `mobile/src/components/agents/HandoffBanner.tsx:28-33`
- `mobile/__tests__/components/agents/HandoffBanner.test.tsx:94-110`

Result:

- this contradicts the wedge routing model and the Wave 5/Wave 6 completion claims

### 3. Sales Brief screen only renders a reduced payload, not the frozen contract

The plan requires the mobile sales brief surface to render the dedicated contract directly, including completed and abstained outcomes with:

- `summary`
- `risks`
- `nextBestActions`
- `confidence`
- `abstentionReason`
- `evidencePack`

Current implementation:

- the screen only models `summary` and `recommendations`
- it does not render `outcome`, `risks`, `nextBestActions`, `confidence`, `abstentionReason`, or `evidencePack`
- the tests only assert the reduced shape and therefore do not protect the real wedge contract

Evidence:

- `mobile/app/(tabs)/sales/[id]/brief.tsx:21-64`
- `mobile/__tests__/app/(tabs)/sales/brief.test.tsx:32-68`
- canonical type still defines the richer contract in `mobile/src/services/api.types.ts`

### 4. Approval UX still exposes legacy deny semantics

The plan fixes the public approval contract to `approve` and `reject` only, and explicitly says mobile must not send or render `deny` or `denied`.

Current implementation still uses deny semantics in UI and tests:

- `ApprovalCard` props, dialog copy, button labels, and test IDs use `deny`
- legacy service tests still validate `decision: 'deny'`

Evidence:

- `mobile/src/components/approvals/ApprovalCard.tsx:1-112`
- `mobile/__tests__/services/api.test.ts`

Result:

- the API aliases were normalized, but the visible UX contract is still drifting from the plan

### 5. Support active-run badge is wired against the wrong query shape

`useAgentRuns` is implemented as a `useQuery` returning a normal response payload.

Current support detail logic reads `data.pages` as if it were an infinite query, and the unit test mocks that same incorrect shape.

Evidence:

- `mobile/src/hooks/useWedge.ts:114-127`
- `mobile/app/(tabs)/support/[id].tsx:131-135`
- `mobile/__tests__/app/(tabs)/support/agent-trigger.test.tsx:125-129`

Result:

- the active run badge can pass tests while still failing in real runtime conditions

## What Is Actually Implemented

- Wedge tab shell exists in `mobile/app/(tabs)/_layout.tsx`
- Support list/detail/coplanar copilot routes exist
- Sales segmented browsing and detail routes exist
- Governance summary surface exists
- Activity detail has meaningful wedge-oriented rendering
- BFF inbox aggregation and approval alias routes exist
- Go governance summary endpoint exists
- Maestro wedge seed/audit flow exists

## Validation Run

The following targeted suites were executed during this audit and passed:

- `cd mobile && npm run test:ui -- --runTestsByPath '__tests__/app/(tabs)/activity/index.test.tsx' '__tests__/app/(tabs)/sales/brief.test.tsx' '__tests__/app/(tabs)/governance/index.test.tsx' '__tests__/app/(tabs)/support/agent-trigger.test.tsx' '__tests__/components/agents/HandoffBanner.test.tsx'`
- `cd bff && npm test -- --runTestsByPath tests/inbox.test.ts`

Interpretation:

- passing tests confirm the current implementation shape
- they do **not** prove wedge completion, because some tests codify the same legacy or reduced behavior identified above

## Documentation Decision

The canonical vault state should be:

- plan remains open/active from an execution perspective
- implementation status is **partial**
- completion claims in Wave 3 through Wave 6 should be read as claimed delivery, not audited parity

## Recommended Follow-up

1. Implement the real Inbox screen against `useInbox`, including chips, ordering, approval actions, handoff routing, and signal entry.
2. Rewrite `HandoffBanner` routing to wedge destinations plus activity-detail fallback.
3. Rework Sales Brief UI and tests around the real `SalesBrief` contract.
4. Remove `deny` wording and identifiers from approval UI/tests in favor of `reject`.
5. Fix support active-run consumption to match the `useAgentRuns` response shape.
