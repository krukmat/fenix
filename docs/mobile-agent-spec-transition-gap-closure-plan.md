# Mobile Agent Spec Transition Gap Closure Plan

## Context

This document is the source of truth for closing the mobile gaps of
`agent-spec-transition`.

Deployment-topology audit and the BFF minimization/remediation rationale are
tracked separately in `docs/mobile-bff-api-audit-remediation-plan.md` so this
document can stay focused on mobile scope closure and contract consumption.

It is reconstructed from four confirmed sources:

1. Completed Mobile P1 task documents in `docs/tasks/task_mobile_p1_*`
2. Current mobile implementation under `mobile/`
3. Current backend and BFF/API contracts in the repository
4. The partial Claude draft at `C:\Users\octoedro\.claude\plans\piped-imagining-allen.md`

Mobile P1 is complete. There is no reliable in-repo Mobile P2, P3, or P4
documentation today. The next mobile closure phase is therefore a reconstructed
Mobile P2 for `agent-spec-transition`, not a generic future roadmap.

This reconstructed phase is limited to four capability groups:

- workflow authoring and versioning
- agent run status alignment
- CRM entity agent activity
- signal visibility on CRM entities

## Current Status

The reconstructed Mobile P2 scope is now substantially implemented in the
repository.

Implemented in code:

- workflow create, edit, version history, new version, and rollback
- agent run status alignment including `accepted`, `rejected`, and `delegated`
- `rejection_reason` rendering in mobile list/detail flows
- `AgentActivitySection` on account, deal, and case detail screens
- contextual `active_signal_count` badges on CRM lists and aggregated payloads
- Detox smoke coverage for workflows, rejected runs, and account Agent Activity

This document remains useful as a closure map, but several sections below
describe gaps that have already been closed.

## Phase And Task Dependency Model

The reconstructed Mobile P2 phase is layered on top of the completed Mobile P1
baseline. Dependencies are not only internal to P2; each P2 task is anchored to
specific P1 work that already exists in the repository.

Phase-level dependency:

- Mobile P2 depends on Mobile P1.1 through Mobile P1.8
- Mobile P2 also depends on Mobile P1 QA as the validated baseline

Task-level dependency model:

- P2.1 depends on the full Mobile P1 baseline and unlocks all later P2 tasks
- P2.2 depends on P2.1 and extends the API client introduced in P1.1
- P2.3 depends on P2.2 and extends the hook layer introduced in P1.2
- P2.4 depends on P2.3 and builds on workflow screens introduced in P1.4
- P2.5 depends on P2.3 and extends workflow detail flows introduced in P1.4
- P2.6 depends on P2.3 and extends the agent run surfaces already used in P1.8
- P2.7 depends on P2.3 and extends entity-detail and navigation work from P1.5
  and P1.6
- P2.QA depends on P2.1 through P2.7 and uses Mobile P1 QA as the parity
  baseline

## What Phase 1 Already Delivered

Mobile P1 already delivered the signal-driven base experience:

- API access for signals, workflows, approvals, agents, handoff, and copilot
- TanStack Query hooks for signals, workflows, approvals, and handoff
- workflow list and workflow detail screens with activate, verify, and execute
- agent runs list and agent run detail with handoff support for escalated runs
- entity-level signal rendering through `EntitySignalsSection`
- drawer and navigation integration
- P1 QA coverage documented as completed

The current mobile code confirms this baseline:

- `mobile/src/services/api.ts` already exposes `signalApi`, `workflowApi`,
  `approvalApi`, `agentApi`, and `copilotApi`
- `mobile/src/hooks/useAgentSpec.ts` already exposes signal, workflow,
  approval, and handoff hooks
- workflow list/detail screens already exist
- agent runs list/detail screens already exist
- account, deal, and case detail screens already show signals

The reconstructed closure phase must preserve all of this behavior.

## Confirmed Gaps In Current Mobile

The following gap list reflects the original reconstructed plan. It is preserved
for traceability, but most items are now resolved in the repo.

The current repository still has these confirmed gaps:

### 1. Workflow authoring and versioning are incomplete in mobile

Original gap:

- create workflow
- edit workflow DSL
- list workflow versions
- create a new workflow version
- rollback an archived version

Status now:

- resolved in `mobile/src/services/api.ts`
- resolved in `mobile/src/hooks/useAgentSpec.ts`
- resolved in `mobile/app/(tabs)/workflows/new.tsx`
- resolved in `mobile/app/(tabs)/workflows/edit/[id].tsx`
- resolved in `mobile/app/(tabs)/workflows/[id].tsx`

### 2. Agent run status modeling is behind AGENT_SPEC

AGENT_SPEC introduced these additional statuses:

- `accepted`
- `rejected`
- `delegated`

Original gap:

- `running`
- `success`
- `failed`
- `abstained`
- `partial`
- `escalated`

Status now:

- mobile list/detail support `accepted`, `rejected`, and `delegated`
- `rejection_reason` is exposed by API and rendered in mobile
- yesterday's commits aligned the rejected-run surface and smoke coverage

### 3. CRM entity detail screens do not surface agent activity

Original gap:

Status now:

- resolved through `AgentActivitySection` on `Account`, `Deal`, and `Case`
  detail screens

### 4. CRM lists do not expose contextual signal counts

Original gap:

The app can render signals for a single entity detail, but CRM lists do not
show a contextual signal badge per entity. This is a product gap and also a
data-shape gap: mobile should not issue one signal query per row.

Status now:

- resolved via `active_signal_count` in BFF aggregated payloads
- resolved via mobile signal badges in account, deal, and case lists

### 5. Part of the missing mobile scope is blocked by missing backend contracts

Original gap:

The workflow domain service already supports versioning and rollback, but the
required API surface is not fully exposed in the current routes. The mobile
closure phase therefore includes backend/BFF prerequisites and is not only a UI
task.

Status now:

- resolved in backend routes and handlers
- resolved in BFF/mobile consumption paths

## Required Backend And BFF Prerequisites

The following contracts were required for the mobile closure phase and are now
present in the repository.

### Workflow contracts

The backend must expose:

- `GET /api/v1/workflows/{id}/versions`
- `POST /api/v1/workflows/{id}/new-version`
- `PUT /api/v1/workflows/{id}/rollback`

Existing workflow endpoints remain part of the contract:

- `POST /api/v1/workflows`
- `GET /api/v1/workflows`
- `GET /api/v1/workflows/{id}`
- `PUT /api/v1/workflows/{id}`
- `PUT /api/v1/workflows/{id}/activate`
- `POST /api/v1/workflows/{id}/verify`
- `POST /api/v1/workflows/{id}/execute`

### Agent run contracts

The backend must expose `GET /api/v1/agents/runs` with support for filtering by:

- `entity_type`
- `entity_id`
- `workflow_id`
- `status`

The response must include enough structured context for mobile to render status
and navigation without ad hoc payload inspection. At minimum:

- `status`
- `workflow_id`
- `entity_type`
- `entity_id`
- `rejection_reason`

If handoff-specific metadata is already exposed, it should remain stable and
separate from delegated-status semantics.

### CRM/BFF contracts

CRM list payloads exposed to mobile through the BFF must include:

- `active_signal_count`

This is required to avoid mobile-side N+1 signal fetching per list row.

### Contract rule

Mobile must consume explicit endpoint-path contracts. It should not infer
workflow versioning, entity linkage, or rejection semantics from loosely shaped
JSON blobs where avoidable.

## Mobile Data Layer Changes

The mobile data layer must be extended in these areas.

### API layer

Add the following workflow methods to `mobile/src/services/api.ts`:

- `workflowApi.create`
- `workflowApi.update`
- `workflowApi.getVersions`
- `workflowApi.newVersion`
- `workflowApi.rollback`

Define or extend these types:

- `CreateWorkflowInput`
- `UpdateWorkflowInput`
- `WorkflowVersion`
- `AgentRun`

`AgentRun` must explicitly support:

- `accepted`
- `rejected`
- `delegated`
- `rejection_reason`
- `workflow_id`
- `entity_type`
- `entity_id`

### Query hooks

Extend `mobile/src/hooks/useAgentSpec.ts` with:

- `useCreateWorkflow`
- `useUpdateWorkflow`
- `useWorkflowVersions`
- `useNewVersion`
- `useRollback`
- `useAgentRunsByEntity`
- `useAgentRunsByWorkflow`

Query keys must remain workspace-isolated and invalidate correctly after:

- workflow creation
- workflow update
- new version creation
- rollback
- run-triggering flows that change workflow or entity activity views

### Data-flow defaults

The default implementation choice for CRM agent activity is:

- use filtered `agents/runs` queries
- do not embed full agent activity collections inside aggregated CRM detail
  payloads unless later evidence proves it is necessary

The default implementation choice for signal badges is:

- consume `active_signal_count` from CRM list/detail payloads
- do not query the signal API once per visible list row

## Mobile UI Changes

### 1. Workflow create and edit

Add:

- `mobile/app/(tabs)/workflows/new.tsx`
- `mobile/app/(tabs)/workflows/edit/[id].tsx`

Behavior:

- create a draft workflow with name, description, and DSL source
- edit only workflows in `draft`
- navigate back to workflow detail on success
- keep MVP editing as a multiline text editor; no DSL diff or advanced editor in
  this phase

### 2. Workflow version history and rollback

Extend workflow detail to show:

- version history
- current version state
- action to create a new version from an active workflow
- action to rollback an archived version

This should be implemented as a dedicated version-history section or component,
not as ad hoc inline buttons only.

### 3. Agent run list and detail alignment

Update agent list and detail screens to:

- render `accepted`, `rejected`, and `delegated`
- display `rejection_reason` when present
- distinguish `delegated` from `escalated`

Rule:

- `escalated` continues to represent handoff-related behavior
- `delegated` represents AGENT_SPEC delegation and must not be treated as the
  same state

### 4. CRM entity Agent Activity section

Add an `Agent Activity` section to:

- account detail
- deal detail
- case detail

The section should show recent or relevant runs for that entity and allow
navigation to run detail.

### 5. Contextual signal badges

Add contextual signal badges to CRM entities:

- on CRM list rows/cards, based on `active_signal_count`
- on CRM detail headers or visible entity summary areas when useful

The badge should guide users toward existing signal-aware surfaces. It should
not introduce a second parallel signal UX.

## Validation Strategy

Validation for this closure phase should confirm parity and prevent regression.

### Backend validation

- handler tests for workflow versioning endpoints
- handler tests for filtered `GET /api/v1/agents/runs`
- contract tests for enriched run response fields used by mobile
- BFF tests for `active_signal_count` exposure on CRM payloads

### Mobile validation

- `cd mobile && npm run typecheck`
- `cd mobile && npm run lint`
- `cd mobile && npm test`
- `cd mobile && npm run test:ui`

### Mobile smoke E2E

Targeted Detox smoke scenarios now present:

- creating a draft workflow
- editing a draft workflow
- viewing version history and performing version actions
- rendering a rejected run with rejection reason
- rendering Agent Activity on a CRM entity detail

### Parity requirement

The closure phase must preserve all working Mobile P1 behavior:

- signals
- approvals
- workflows list and current detail actions
- handoff surfaces
- drawer navigation

No regression is acceptable in those areas.

## Follow-up Task Breakdown

This master document maps to a reconstructed task series in `docs/tasks`.

Original required task set:

- `task_mobile_p2_1.md` — backend and BFF prerequisites for workflow versioning,
  filtered agent runs, and signal counts
- `task_mobile_p2_2.md` — mobile API layer and TypeScript contract alignment
- `task_mobile_p2_3.md` — TanStack Query hooks for workflow versioning and
  filtered agent runs
- `task_mobile_p2_4.md` — workflow create/edit screens
- `task_mobile_p2_5.md` — workflow version history and rollback UI
- `task_mobile_p2_6.md` — agent run status alignment in list/detail
- `task_mobile_p2_7.md` — CRM Agent Activity and contextual signal badges
- `task_mobile_p2_qa.md` — QA, parity coverage, and E2E smoke coverage

Original execution order:

1. backend and BFF prerequisites
2. mobile API and hooks
3. workflow authoring/versioning UI
4. agent run status alignment
5. CRM activity and signal visibility
6. QA and parity validation

That ordering is no longer the current repository state. The remaining work
should focus on parity verification, documentation cleanup, and any missing QA
coverage discovered during validation.
