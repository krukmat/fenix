# Insights Pilot Workflow

**Status**: Modeling baseline for F7.2
**Phase**: AGENT_SPEC - Fase 7 Migracion progresiva
**Pilot**: `insights`

---

## Goal

Model the current Go `insights` behavior as a declarative workflow candidate for
shadow mode.

This document does not activate rollout.

It defines the workflow shape, parity assumptions, and intentional gaps that
must be tracked in F7.3 to F7.5.

---

## Current Go Baseline

The current Go `insights` agent does this:

1. validates `workspace_id` and `query`
2. normalizes language
3. applies daily usage limits
4. infers query intent
5. queries metrics through `query_metrics`
6. queries knowledge through `search_knowledge`
7. abstains when both metrics and evidence are weak
8. otherwise returns an answer payload with:
   - `action=answer`
   - `metrics`
   - `confidence`
   - `evidence_ids`
9. records `tool_calls`

Important baseline behaviors to preserve:

- backlog intent must win over generic case volume intent
- no approval path exists in the current baseline
- empty data must lead to `action=abstain` and `confidence=low`

---

## Declarative Modeling Strategy

The safest migration strategy is not to reproduce every internal helper of the
Go agent directly in one step.

Instead, the pilot workflow should make explicit these stages:

1. normalized trigger context
2. intent-based branch
3. metrics query
4. knowledge query
5. abstain decision
6. answer shaping

This keeps the migration auditable and makes parity comparison easier.

---

## Candidate Workflow DSL

```text
WORKFLOW insights_pilot
ON insights.query_received

IF query.intent == "case_backlog":
  AGENT insights_case_backlog WITH {"workspace_id":"ws_ref","query":"case backlog"}

IF query.intent == "case_volume":
  AGENT insights_case_volume WITH {"workspace_id":"ws_ref","query":"case volume"}

IF query.intent == "sales_funnel":
  AGENT insights_sales_funnel WITH {"workspace_id":"ws_ref","query":"sales funnel"}

IF query.intent == "deal_aging":
  AGENT insights_deal_aging WITH {"workspace_id":"ws_ref","query":"deal aging"}

IF query.intent == "mttr":
  AGENT insights_mttr WITH {"workspace_id":"ws_ref","query":"mttr"}

AGENT search_knowledge WITH {"workspace_id":"ws_ref","query":"raw query"}

IF evidence.top_score < 0.4:
  AGENT insights_abstain WITH {"reason":"insufficient_data","confidence":"low"}

AGENT insights_answer WITH {"metric":"query.intent","query":"raw query"}
```

---

## Modeling Notes

This DSL is a migration model, not yet the final polished workflow for
production.

Intentional decisions:

- `query.intent` is assumed to be available in trigger context.
  Reason:
  query intent parsing in Go is currently embedded in helper logic, and the DSL
  runtime does not yet provide a first-class string classification layer.

- metric-specific branches are explicit.
  Reason:
  this makes comparison against Go easier and preserves backlog-priority logic.

- answer shaping is represented as an `AGENT` stage.
  Reason:
  current DSL verbs do not yet express rich response composition directly.

- abstention is explicit and isolated.
  Reason:
  it is one of the most important parity conditions for the pilot.

---

## Parity Targets

The pilot should preserve these outcomes:

| Concern | Go baseline | Declarative target |
|---|---|---|
| query required | hard error | same |
| backlog priority | `case_backlog` wins | same |
| metrics query | `query_metrics` call | same observable tool call |
| knowledge query | `search_knowledge` call | same observable tool call |
| abstention | `action=abstain`, `confidence=low` | same |
| non-abstain | answer + metrics + confidence + evidence | same shape or equivalent |
| approvals | none | none |
| rollback | Go path resumes immediately | same |

---

## Known Gaps Before Shadow Mode

These gaps are accepted in F7.2 and must be resolved or accounted for in
F7.3-F7.5:

1. the current DSL does not yet provide a native intent-classification verb
2. the current DSL does not yet provide a first-class answer formatter for the
   exact Go output wording
3. some branch data is easier to compare as normalized structured output than as
   exact human-readable text
4. the candidate DSL shown here is a migration model and still needs concrete
   runner wiring for the chosen metric-specific branches

---

## Recommended Shadow-Mode Comparison Focus

For the pilot, parity should be evaluated in this order:

1. status and abstention behavior
2. metric selected from the same query
3. presence and order of `tool_calls`
4. evidence IDs and confidence level
5. final answer payload shape
6. final human-readable answer wording

This order reduces noise and keeps the migration decision based on stable
behavior first.
