# Support Trigger Contract Decision

## Status

Accepted for implementation planning.

## Context

The support trigger inventory shows a live contract mismatch:

- mobile support detail currently sends `{ entity_type, entity_id }`;
- BFF transparently proxies that payload without translation;
- backend `POST /api/v1/agents/support/trigger` expects `{ case_id, customer_query, language?, priority? }`;
- the support agent domain model is already case-native and depends on `case_id` plus a real operator/customer query.

Reference: [support-trigger-contract-inventory.md](./support-trigger-contract-inventory.md)

## Decision

The canonical trigger contract for live support runs will be the support-native request body:

```json
{
  "case_id": "<case-id>",
  "customer_query": "<operator prompt or customer issue>",
  "language": "es",
  "priority": "low|medium|high"
}
```

Required fields:

- `case_id`
- `customer_query`

Optional fields:

- `language`
- `priority`

## Why This Is The Canonical Contract

### 1. It matches the existing backend domain model

The support agent is not modeled as a generic entity dispatcher. Its config, validation, run payloads, and downstream behavior are all centered on:

- a support case identifier;
- a human/customer issue statement;
- optional language/priority context.

Choosing a different canonical contract would mean forcing translation into the backend's real shape anyway.

### 2. `customer_query` is not derivable from `entity_type/entity_id`

The generic mobile shape identifies a record, but it does not express the operator intent that actually triggers the run.

For a live support workflow, the support agent needs more than "run on this case":

- what the customer asked;
- what the operator wants analyzed or drafted;
- or what issue statement should anchor retrieval and response generation.

Without `customer_query`, the backend cannot reliably reconstruct intent from the current support detail screen contract.

### 3. It keeps the dedicated support endpoint meaningfully dedicated

There are already two trigger concepts in the repo:

- generic `POST /api/v1/agents/trigger`;
- dedicated `POST /api/v1/agents/support/trigger`.

If the dedicated support endpoint were canonicalized around generic entity fields, the distinction would become weaker while still requiring hidden support-specific enrichment later. The dedicated endpoint should carry the support-specific inputs explicitly.

### 4. It minimizes hidden coupling in BFF

Teaching BFF to translate `{ entity_type: "case", entity_id }` into `{ case_id, customer_query, ... }` would require BFF to invent or source `customer_query` from somewhere else. That creates opaque coupling and makes the support trigger behavior harder to reason about, test, and document.

The cleaner boundary is:

- mobile gathers support-specific operator input;
- BFF forwards it transparently;
- backend validates and runs it.

## Rejected Alternative

Rejected canonical contract:

```json
{
  "entity_type": "case",
  "entity_id": "<case-id>"
}
```

Why it is rejected:

- it does not carry `customer_query`;
- it reflects UI navigation identity, not support-run intent;
- it conflicts with the current backend support handler and support-agent config;
- it would push support-specific enrichment into BFF or backend-side inference, which is less explicit and less testable.

## Implementation Consequence

### Mobile

`mobile` should become the side that adapts:

- the support case detail flow must source a real `customer_query`;
- the trigger call must post `case_id` instead of `entity_id`;
- optional `priority` can be passed from current case data when useful;
- optional `language` can be defaulted or exposed explicitly later.

This likely means the current "Run Support Agent" action cannot remain a blind one-tap call from case detail unless the UI defines where `customer_query` comes from.

### BFF

`bff` should remain a transparent proxy for this route.

No contract translation should be introduced in BFF for the canonical live path unless a future requirement explicitly calls for an anti-corruption layer. The current route boundary is cleaner if BFF simply forwards the support-native payload unchanged.

### Backend

The backend support handler and support-agent config are already aligned with the chosen canonical contract.

Backend work, if any, should be limited to:

- preserving or tightening contract validation;
- documenting the request schema more clearly;
- adding tests around the final agreed mobile payload shape.

No contract inversion is needed on the backend side.

## Migration Note For The Non-Canonical Side

The non-canonical side is the current mobile support trigger implementation.

Migration direction:

1. Replace the current `{ entity_type, entity_id }` support trigger payload with `{ case_id, customer_query, language?, priority? }`.
2. Define the UI/UX source of `customer_query` before wiring the trigger.
3. Keep generic `agentApi.triggerRun()` for generic agent-launch surfaces only; do not treat it as the live support contract.

## Follow-Up Constraint

Before implementation, the next design/engineering step must answer one concrete UX question:

`customer_query` will come from where?

Valid options include:

- a modal prompt from support case detail;
- a prefilled editable text area in the support surface;
- a Copilot-originated prompt/action that then triggers the run.

That UX choice is required before changing the mobile trigger behavior.
