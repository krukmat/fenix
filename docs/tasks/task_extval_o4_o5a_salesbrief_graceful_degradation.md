---
doc_type: task
id: EXTVAL-O4-O5A-SALESBRIEF-DEGRADATION-001
title: "Sales Brief must degrade gracefully on suggested-actions parse failure and always record usage"
status: complete
phase: external-validation-rerun
week: "2026-W27"
tags: [external-validation, bugfix, copilot, sales-brief, usage, o4, o5]
fr_refs: [FR-200, FR-202]
uc_refs: [UC-C1]
blocked_by: []
blocks: []
files_affected:
  - internal/domain/copilot/suggest_actions.go
  - internal/domain/copilot/suggest_actions_test.go
criticality: standard
criticality_basis: "No auth/security/data-model impact; internal/domain/copilot has no anchor-rubric floor above baseline. Behavior fix only, no public API/schema change."
created: 2026-07-05
completed: 2026-07-05
---

# Task EXTVAL-O4-O5A-SALESBRIEF-DEGRADATION-001

**Plan**: [External Validation Open Points Rerun Plan](../plans/external_validation_open_points_rerun_plan.md#o4-sales-brief-hard-fails-on-malformed-suggested-actions-output)

## Task Card

Task: EXTVAL-O4-O5A-SALESBRIEF-DEGRADATION-001

Task file: docs/tasks/task_extval_o4_o5a_salesbrief_graceful_degradation.md

Plan file: docs/plans/external_validation_open_points_rerun_plan.md (sections O4, O5)

Summary: `ActionService.SalesBrief` (`internal/domain/copilot/suggest_actions.go:294-344`) generates the brief text (`generateSalesBrief`) and the suggested actions (`generateSuggestedActions`) as two independent LLM calls, but only the first one degrades gracefully on parse failure. If `generateSuggestedActions` fails to parse the model's JSON (or produces zero valid actions after filtering), its error propagates all the way to the HTTP handler as a `500`, discarding an otherwise-successful brief (summary + risks) that already cost a real 20-26s LLM call. Additionally, none of `SalesBrief`'s early-return error paths call `recordSalesBriefUsage`, so a failed generation — which already consumed real tokens/latency — never emits a `usage_event`, undercounting cost/latency evidence.

Root cause evidence (confirmed via code read):
- `internal/domain/copilot/suggest_actions.go:322-329` — `generateSalesBrief` error and `generateSuggestedActions` error are both handled the same way (`return nil, err`), but only `generateSalesBrief` (lines 346-387) has an internal fallback: `parseSalesBriefPayload` failing at line 374-377 falls back to `fallbackSalesBriefPayload(...)` and continues without error. `generateSuggestedActions` (lines 169-201) has no equivalent — `parseSuggestedActions` failing (line 188-191) or producing zero actions (line 195-196, `errSuggestedActionsParseFail`) returns the error directly with no fallback.
- `internal/domain/copilot/suggest_actions.go:294-344` (`SalesBrief`) — `recordSalesBriefUsage` is only called at line 318 (abstention path) and line 342 (success path). The error returns at lines 296, 302, 324, and 328 never call it, so the real LLM call already made before line 326 (`generateSalesBrief`, which succeeded) has its usage silently dropped when `generateSuggestedActions` subsequently fails.

Code affected: `internal/domain/copilot/suggest_actions.go` (fix), `internal/domain/copilot/suggest_actions_test.go` (new tests).

Criticality: standard

Criticality basis: no anchor-rubric floor applies to `internal/domain/copilot/**`; this is a behavior/resilience fix with no security, auth, or persisted-schema impact.

Effort/reasoning: Low — the fix mirrors an existing pattern already present in the same file (`generateSalesBrief`'s fallback), applied symmetrically to `generateSuggestedActions`, plus moving the usage-recording call to cover all exit paths.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Estimated tokens: ~5000

## RRI

| Variable | Score | Evidence | Confidence |
|---|---|---|---|
| C cyclomatic | 0 | raw CC ~3 for the touched functions -> score 0 | High |
| F files | 0 | 1 file touched (`internal/domain/copilot/suggest_actions.go`) | High |
| D domain | 2 | no anchor-rubric floor for `internal/domain/copilot/**`; normal business logic | High |
| T coverage | 1 | `suggest_actions_test.go` has direct `SalesBrief` tests (`TestSalesBrief_CompletesForDealContext`, `_AbstainsWhenEvidenceIsInsufficient`, `_FallsBackWhenBriefResponseIsEmpty`) — reasonable existing coverage | High |
| A ambiguity | 0 | root cause fully diagnosed with file:line evidence; fix mirrors an existing in-file pattern | High |
| K coupling | 2 | no anchor-rubric floor; internal module logic only, no DB/external-service shape change | High |
| P impact | 2 | changes internal behavior (response shape when actions fail to parse), no public API/schema change | High |
| X context | 2 | must read `generateSalesBrief`'s existing fallback pattern plus `SalesBrief`'s call sites to apply symmetrically | Medium |

**Base value:** 100 x (weighted / 5) = 20
**Penalties applied:** none
**Final RRI:** 20 -> band Low (0-25) -> Effort S / claude-sonnet-4-6 / thinking Off
**Gates for this band:** Execute directly. No full approval packet required.
**Criticality suggested:** no — P < 4; no critical-task signal

## High-Level Pseudocode

```
# internal/domain/copilot/suggest_actions.go

# 1. generateSuggestedActions: add a fallback mirroring generateSalesBrief's
#    parseSalesBriefPayload -> fallbackSalesBriefPayload pattern.

func generateSuggestedActions(ctx, entityType, entityID, pack, sources):
    resp, err = llm.ChatCompletion(...)
    if err != nil:
        return nil, metrics{}, err   # unchanged: a hard LLM-call failure still fails
                                     # (SalesBrief's own generateSalesBrief call would
                                     # have already failed first in that case anyway)

    actions, parseErr = parseSuggestedActions(resp.Content)
    if parseErr != nil OR len(normalizeActions(actions)) == 0:
        # NEW: fall back to an empty (or minimally derived) action list instead
        # of propagating the error — mirrors fallbackSalesBriefPayload's role.
        actions = fallbackSuggestedActions(entityType, entityID)  # returns []SuggestedAction{}
        metrics = suggestActionsMetrics{generated: 0, returned: 0, discardReasons: {"parse_failure": 1}}
        return actions, metrics, nil   # no error — brief continues with empty actions

    actions = normalizeActions(actions, 3)
    actions = normalizeActionsWithContext(actions, entityType, entityID)
    filtered, metrics = scoreAndFilterSuggestedActions(actions, entityType, entityID, pack)
    return filtered, metrics, nil

# 2. SalesBrief: ensure recordSalesBriefUsage is called on every exit path that
#    followed a real LLM call, not just the success/abstention paths.

func SalesBrief(ctx, in):
    if invalid input: return nil, err          # no LLM call yet — no usage to record
    startedAt = now()
    prepared, err = prepareSuggestActionsContext(...)
    if err != nil: return nil, err             # no LLM call yet — no usage to record

    if abstention reason:
        record usage (existing path, unchanged)
        return abstained result, nil

    brief, record, err = generateSalesBrief(...)   # real LLM call happened here
    if err != nil:
        # NEW: record usage even on failure — the call already cost tokens/latency
        recordSalesBriefUsage(ctx, in, record, time.Since(startedAt))
        return nil, err

    actions, metrics, err = generateSuggestedActions(...)
    # NOTE: after fix #1 above, this branch only fires on a genuine transport/LLM
    # error (parse failures no longer propagate here), but keep it defensive:
    if err != nil:
        recordSalesBriefUsage(ctx, in, record, time.Since(startedAt))  # from generateSalesBrief
        return nil, err

    build result{Outcome: "completed", ...}
    log audit, record usage (existing path, unchanged)
    return result, nil
```

## Acceptance Criteria

1. When the LLM's suggested-actions JSON fails to parse (or normalizes to zero actions), `SalesBrief` returns a `completed` result with `Summary`/`Risks` populated and `NextBestActions` as an empty (or fallback-derived) slice — not a `500`/error.
2. The existing `generateSalesBrief` fallback behavior (`fallbackSalesBriefPayload`) is unchanged.
3. Every `SalesBrief` code path that made a real LLM call (whether it ultimately succeeds, abstains, or hits a genuine transport error after a partial success) results in exactly one `recordSalesBriefUsage` call; paths that never called the LLM (input validation, evidence-pack build failure) do not call it.
4. `go test ./internal/domain/copilot/...` passes, including new tests covering: (a) suggested-actions parse failure no longer fails the whole brief, (b) usage is recorded when `generateSuggestedActions` fails after `generateSalesBrief` succeeded.
5. No change to the `generateSalesBrief`/brief-summary behavior, `Summarize`, or any other `ActionService` method.

## Result (2026-07-05)

1. **PASS** — `generateSuggestedActions` (`internal/domain/copilot/suggest_actions.go:169-204`) now returns an empty `[]SuggestedAction{}` with `discardReasons: {"parse_failure": 1}` instead of propagating `errSuggestedActionsParseFail`, both when `parseSuggestedActions` fails and when normalization yields zero actions. This benefits both `SalesBrief` and the standalone `SuggestActions` endpoint, which share this method.
2. **PASS** — `generateSalesBrief`'s existing fallback (`fallbackSalesBriefPayload`) untouched.
3. **PASS** — `SalesBrief` (`suggest_actions.go:298-347`) now records usage on the defensive post-`generateSalesBrief` error path too (comment added at the call site); the pre-LLM-call paths (validation, evidence-pack build) correctly still don't record usage since no LLM call was made.
4. **PASS** — `go test ./internal/domain/copilot/...` passes. Added `TestSalesBrief_DegradesToEmptyActionsWhenActionsParseFails` (brief succeeds, actions parse fails → `completed` outcome, empty `NextBestActions`, exactly 1 usage event) and updated `TestSuggestActions_InvalidOutput_ReturnsError` → `TestSuggestActions_InvalidOutput_DegradesToEmptyList` to assert the new no-error/empty-list contract.
5. **PASS** — `go build ./...` and `go vet ./internal/domain/copilot/...` clean; no other `ActionService` method touched.
