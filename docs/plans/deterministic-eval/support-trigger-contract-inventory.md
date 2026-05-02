# Support Trigger Contract Inventory

## Scope

This note documents the current support-trigger path across `mobile`, `bff`, and backend as implemented today. It is an inventory only. No runtime behavior is changed here.

## End-to-End Path

| Layer | Source | Route | Outgoing / expected body | Notes |
|---|---|---|---|---|
| Mobile screen | `mobile/app/(tabs)/support/[id].tsx` | UI button `Run Support Agent` | Calls `triggerAgent.mutate({ entityType: 'case', entityId: caseData.id })` | The support case detail screen only knows `entityType` and `entityId` at trigger time. |
| Mobile hook | `mobile/src/hooks/useWedge.ts` | `useTriggerSupportAgent()` | Converts to `agentApi.triggerSupportRun({ entity_type, entity_id })` | Hook normalizes camelCase input to snake_case API fields, but still sends generic entity fields. |
| Mobile API service | `mobile/src/services/api.agents.ts` | `POST /bff/api/v1/agents/support/trigger` | JSON body `{ "entity_type": "...", "entity_id": "..." }` | This is the dedicated support-trigger client currently used by the support detail screen. |
| BFF | `bff/src/routes/proxy.ts` | `/bff/api/v1/* -> /api/v1/*` | No field translation | The proxy rewrites only the path and restreams the parsed JSON body. |
| Backend HTTP handler | `internal/api/handlers/agent.go` | `POST /api/v1/agents/support/trigger` | Expects `{ "case_id": string, "customer_query": string, "language"?: string, "priority"?: string }` | `buildSupportConfig()` rejects missing `case_id` and missing `customer_query`. |
| Backend domain config | `internal/domain/agent/agents/support.go` | `SupportAgent.Run()` | Uses `SupportAgentConfig{ WorkspaceID, CaseID, CustomerQuery, Language, Priority, ... }` | The downstream support-agent path is case-specific, not generic entity-based. |
| Backend orchestrator trigger context | `internal/domain/agent/agents/support.go` | `supportRunPayloads()` | Writes trigger context with `case_id`, `customer_query`, `language`, `priority`, `agent_type` | The run persisted by the support agent also uses case-native fields. |

## Effective Contract By Layer

### Mobile emits today

Support case detail sends:

```json
{
  "entity_type": "case",
  "entity_id": "<case-id>"
}
```

Observed path:

1. `SupportCaseDetailScreen` invokes `useTriggerSupportAgent()`.
2. `useTriggerSupportAgent()` calls `agentApi.triggerSupportRun({ entity_type, entity_id })`.
3. `agentApi.triggerSupportRun()` posts that body to `/bff/api/v1/agents/support/trigger`.

### BFF does today

The BFF does not adapt the support payload. The proxy:

1. receives `/bff/api/v1/agents/support/trigger`;
2. rewrites the path to `/api/v1/agents/support/trigger`;
3. restreams the same JSON body to Go.

There is no support-specific mapping from `entity_type/entity_id` to `case_id/customer_query`.

### Backend expects today

`SupportAgentHandler.TriggerSupportAgent()` decodes this request type:

```json
{
  "case_id": "<case-id>",
  "customer_query": "<operator prompt or customer issue>",
  "language": "es",
  "priority": "low|medium|high"
}
```

Validation in `buildSupportConfig()` makes these fields effectively required:

- `case_id`
- `customer_query`

Optional fields:

- `language`
- `priority`

The handler then builds `agents.SupportAgentConfig`, and the support agent persists trigger context using `case_id` and `customer_query`, not `entity_type` and `entity_id`.

## Exact Mismatch

| Concern | Mobile sends | Backend expects | Result |
|---|---|---|---|
| Case identifier field | `entity_id` | `case_id` | Identifier name does not match. |
| Entity discriminator | `entity_type = "case"` | no `entity_type` field in handler request | Extra field is ignored by the backend handler. |
| Customer prompt / issue text | not sent | `customer_query` required | Backend validation fails even if `case_id` were mapped. |
| Priority | not sent | optional `priority` | Not blocking by itself, but unavailable to the backend. |
| Language | not sent | optional `language` | Not blocking by itself; backend can default. |

The mismatch is not just naming. It is structural:

- mobile uses a generic entity trigger shape;
- backend uses a support-specific case workflow shape;
- BFF does not bridge the two.

Given the current code, `POST /api/v1/agents/support/trigger` will not receive the minimum required payload when invoked from the support case detail screen.

## Expected Failure Mode

Because the current mobile request body omits `case_id` and `customer_query`, the backend validation path in `buildSupportConfig()` rejects the request before the support agent runs.

The first validation failure is `case_id is required`.

If `case_id` were added without `customer_query`, the next failure would be `customer_query is required`.

## Other Support Trigger Paths In Repo

There is one other trigger surface related to support, but it is not the same contract:

- `mobile/src/components/agents/TriggerAgentButton.tsx` uses the generic `agentApi.triggerRun()` path and posts to `POST /bff/api/v1/agents/trigger`.
- That generic button lets a user select the support agent definition from a modal, but the current implementation confirms with an empty context object `{}`.
- This is a separate generic trigger route, not an alternate implementation of `/agents/support/trigger`.

Implication:

- the repo contains both a generic agent trigger endpoint and a dedicated support trigger endpoint;
- the support case detail screen uses the dedicated endpoint;
- the generic trigger flow does not currently provide the case-native support payload either.

## Decision Inputs For F9.A2

The next contract decision should explicitly choose one of these models:

1. Canonicalize the support trigger around generic entity semantics and teach backend/BFF to resolve `entity_type=case` + `entity_id` into support-agent inputs.
2. Canonicalize the support trigger around support-native semantics and teach mobile to send `case_id` plus a real `customer_query` source, with optional `priority` and `language`.

What is already true in code today:

- the support backend domain model is case-native;
- the support HTTP handler is case-native;
- the mobile support screen trigger is generic entity-native;
- the BFF is transparent and does not currently reconcile the difference.

---

## Post-F9.A4 Status

**Decision applied**: model 2 — canonical contract is support-native.

Reference: [support-trigger-contract-decision.md](./support-trigger-contract-decision.md)

### Server-side path — verified 2026-05-02

| Layer | Status | Notes |
|---|---|---|
| Backend handler `POST /api/v1/agents/support/trigger` | ✅ Aligned | `buildSupportConfig` validates `case_id` + `customer_query`. No production code change needed. |
| BFF proxy | ✅ Aligned | Transparent pass-through (`proxy.ts`). No translation introduced. |
| Go test — validation errors (401, 400, 404, 500) | ✅ Covered | 6 existing tests in `agent_test.go`. |
| Go test — happy-path 201 with real case | ✅ Covered | `TestSupportAgentHandler_TriggerSupportAgent_201` added in F9.A4. |
| BFF test — canonical payload forwarded | ✅ Covered | 2 new tests in `proxy.test.ts` describe `Support trigger pass-through`. |

### Remaining gap

The mobile support trigger still sends `{ entity_type, entity_id }`.
Migration to `{ case_id, customer_query, language?, priority? }` is F9.A5 scope.
