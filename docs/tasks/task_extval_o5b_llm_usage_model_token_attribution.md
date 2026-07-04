---
doc_type: task
id: EXTVAL-O5B-LLM-USAGE-MODEL-ATTRIBUTION-001
title: "Fix chat usage_event model attribution and zero-token accounting for Ollama"
status: complete
phase: external-validation-rerun
week: "2026-W27"
tags: [external-validation, bugfix, llm, usage, ollama, o5]
fr_refs: [FR-040, FR-041]
uc_refs: [UC-C1]
blocked_by: []
blocks: []
files_affected:
  - internal/infra/llm/types.go
  - internal/infra/llm/ollama.go
  - internal/infra/llm/openai_compat.go
  - internal/domain/copilot/chat.go
  - internal/domain/copilot/suggest_actions.go
  - internal/infra/llm/ollama_test.go
  - internal/infra/llm/openai_compat_test.go
  - internal/domain/copilot/chat_test.go
criticality: standard
criticality_basis: "internal/infra/** carries an ADR-010 anchor floor (D/K/P=2), but this is cost/usage-accounting accuracy, not auth/security/data-loss risk. No public API contract changes — ChatResponse is an internal package type."
created: 2026-07-05
completed: 2026-07-05
---

# Task EXTVAL-O5B-LLM-USAGE-MODEL-ATTRIBUTION-001

**Plan**: [External Validation Open Points Rerun Plan](../plans/external_validation_open_points_rerun_plan.md#o5-llm-usage-accounting-is-incomplete-or-incorrect)

## Task Card

Task: EXTVAL-O5B-LLM-USAGE-MODEL-ATTRIBUTION-001

Task file: docs/tasks/task_extval_o5b_llm_usage_model_token_attribution.md

Plan file: docs/plans/external_validation_open_points_rerun_plan.md (section O5)

Summary: Two of the three O5 usage-accounting defects share one root cause: `ChatResponse` (the return value of `LLMProvider.ChatCompletion`) does not carry which model was actually used, and Ollama's response mapping does not carry a token count either. Both `chat.go` and `suggest_actions.go` recover the "model used" after the fact via `s.llm.ModelInfo().ID` — but `OllamaProvider.ModelInfo()` (`internal/infra/llm/ollama.go:177-184`) always returns `p.model` (the *embedding* model), never `p.chatModel` (the model actually used for the chat call that just happened), because `ModelInfo()` takes no arguments and cannot know which operation the caller just performed. Separately, `ollamaChatResponse` (`ollama.go:70-74`) never maps Ollama's token-count fields into `ChatResponse.Tokens`, so every Ollama-backed chat call reports `0` usage units regardless of the model mismatch.

Root cause evidence (confirmed via code read):
- `internal/infra/llm/ollama.go:26-30` — `OllamaProvider` holds two distinct models: `model` (embedding) and `chatModel` (chat), added specifically because `routes.go` previously passed the embed model for both, causing 404s on `/api/chat` (see the existing "UAT fix" comments at lines 29, 35, 122).
- `internal/infra/llm/ollama.go:177-184` (`ModelInfo`) — hardcoded to return `p.model` unconditionally; has no way to reflect "the model used in the last ChatCompletion call" because the interface method takes no context of the prior call.
- `internal/domain/copilot/chat.go:154` and `internal/domain/copilot/suggest_actions.go:364` — both call `s.llm.ModelInfo().ID` immediately after a successful `ChatCompletion`, assuming it reflects the chat model. For Ollama, it silently returns the embedding model name instead.
- `internal/infra/llm/ollama.go:70-74` (`ollamaChatResponse`) has no token-count field; `ChatCompletion` (`ollama.go:155-158`) builds `&ChatResponse{Content: ..., StopReason: ...}` with `Tokens` left at its zero value. Contrast with `internal/infra/llm/openai_compat.go:65-67,133` (`openaiUsage.TotalTokens` mapped to `ChatResponse.Tokens`), which works correctly for that provider.
- Ollama's `/api/chat` response includes `prompt_eval_count` and `eval_count` fields (documented Ollama API — total input/output token counts) when `done: true`, which are currently not decoded at all.

Code affected: `internal/infra/llm/types.go` (interface/struct change), `internal/infra/llm/ollama.go` (fix), `internal/infra/llm/openai_compat.go` (symmetry — populate the same new field), `internal/domain/copilot/chat.go` and `internal/domain/copilot/suggest_actions.go` (consume the new field instead of `ModelInfo().ID`).

Criticality: standard

Criticality basis: `internal/infra/**` carries an ADR-010 anchor floor (D/K/P=2), but the change is accounting-accuracy only — it does not touch auth, persisted schema, or any externally-visible API contract (`ChatResponse` is an internal Go package type, not a wire/HTTP contract).

Effort/reasoning: Medium — touches 5 files across two packages (`llm` and `copilot`), and requires reasoning about keeping the `LLMProvider` interface's existing contract intact for any other implementers, but each individual change is small and mechanical.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Estimated tokens: ~6500

## RRI

| Variable | Score | Evidence | Confidence |
|---|---|---|---|
| C cyclomatic | 0 | raw CC ~2 for touched functions -> score 0 | High |
| F files | 2 | `--touches` -> 5 files | High |
| D domain | 2 | anchor rubric: `internal/infra/**` (ADR-010) -> floor 2 | High |
| T coverage | 1 | `ollama_test.go`, `openai_compat_test.go`, `chat_test.go`, `suggest_actions_test.go` all exist and cover the touched functions directly | High |
| A ambiguity | 0 | root cause fully diagnosed with file:line evidence for all three symptoms; fix approach (carry the used model + map tokens on the response, not via a stateless `ModelInfo()` lookup) is unambiguous | High |
| K coupling | 2 | anchor rubric: `internal/infra/**` (ADR-010) -> floor 2 | High |
| P impact | 2 | anchor rubric: `internal/infra/**` (ADR-010) -> floor 2 — changes an internal struct field, not persisted data or a public API | High |
| X context | 3 | must reason about the shared `LLMProvider` interface contract across two provider implementations plus two call sites in a different package | Medium |

**Base value:** 100 x (weighted / 5) = 26
**Penalties applied:** none
**Final RRI:** 26 -> band Moderate (26-40) -> Effort M / claude-sonnet-4-6 / thinking Off
**Gates for this band:** Present task card and wait for explicit approval. Confirm tests exist in the affected area (confirmed: yes, for all touched files).
**Criticality suggested:** no — P < 4; no critical-task signal

## System Context

```
ChatService.buildChatResult (chat.go)              ActionService.generateSalesBrief /
       |                                            generateSuggestedActions (suggest_actions.go)
       v                                                          |
  llm.ChatCompletion(ctx, ChatRequest)  <──────────────────────────
       |
       v
  ┌─────────────────┐         ┌──────────────────────┐
  │ OllamaProvider   │         │ OpenAICompatProvider │
  │ .ChatCompletion  │         │ .ChatCompletion      │
  │  uses p.chatModel│         │  uses p.model        │
  └────────┬─────────┘         └──────────┬───────────┘
           |                              |
           v                              v
     ChatResponse{                  ChatResponse{
       Content, StopReason,           Content, StopReason,
       Tokens: 0 (BUG),                Tokens: <from usage.total_tokens>,
       Model: "" (BUG, missing)        Model: "" (BUG, missing)
     }                               }
           |                              |
           └──────────────┬───────────────┘
                           v
       caller reads resp.Tokens (0 for Ollama) AND
       separately calls s.llm.ModelInfo().ID (BUG: always
       returns the embed model for Ollama, regardless of
       which model ChatCompletion actually used)
                           |
                           v
              usage.RecordEventInput{ModelName, InputUnits, OutputUnits}
              -> wrong model name + zero units for every Ollama chat call
```

Upstream trigger: any real `ChatCompletion` call from `ChatService` or `ActionService` when the active provider is Ollama.
Downstream consumers: `usage.RecordEvent` (cost/latency/model provenance evidence — NFR-040/041 direction), any governance/cost dashboard reading `usage_event` rows.

Key invariants for whoever picks this up:
- `LLMProvider` (`provider.go:13-25`) is implemented by at least `OllamaProvider` and `OpenAICompatProvider`. Any field added to `ChatResponse` must be populated by both, or the fix only half-solves O5b (openai-compat already reports the correct model via `p.model`, since it only has one model — populate the new field there too for symmetry, not just Ollama).
- Do NOT change `ModelInfo()`'s existing behavior or signature — it is used elsewhere for static provider/model metadata unrelated to a specific chat call (grep before assuming it's safe to repurpose).
- The correct fix is for `ChatCompletion` itself to report which model it used on the `ChatResponse` it returns (it already knows — see `ollama.go:124-127`, `model := req.Model; if model == "" { model = p.chatModel }`), not to make callers infer it after the fact from a stateless method.
- Ollama's real `/api/chat` JSON includes `prompt_eval_count` (input tokens) and `eval_count` (output tokens) as separate fields — do not conflate them into a single total the way the current (missing) mapping would; `ChatResponse.Tokens` is documented as "Total tokens consumed (prompt + completion)" (`types.go:24`), so sum both when mapping.
- `chat.go:156-157` and `suggest_actions.go:365-368` currently set `inputUnits = outputUnits = resp.Tokens` (both equal to the same total) — preserve that shape unless you also fix the input/output split, which is out of scope here (not part of the O5 report).

## High-Level Pseudocode

```
# internal/infra/llm/types.go

type ChatResponse struct:
    Content    string
    StopReason string
    Tokens     int
    Model      string   # NEW — the model actually used for this call

# internal/infra/llm/ollama.go

type ollamaChatResponse struct:
    Message           ollamaChatMessage
    DoneReason        string
    Done              bool
    PromptEvalCount   int  `json:"prompt_eval_count"`  # NEW
    EvalCount         int  `json:"eval_count"`         # NEW

func (p *OllamaProvider) ChatCompletion(ctx, req) (*ChatResponse, error):
    model = req.Model or p.chatModel   # unchanged — already correct
    ...
    ollamaResp = decode(...)
    return &ChatResponse{
        Content:    ollamaResp.Message.Content,
        StopReason: ollamaResp.DoneReason,
        Tokens:     ollamaResp.PromptEvalCount + ollamaResp.EvalCount,  # NEW
        Model:      model,                                             # NEW — the model this call actually used
    }, nil

# internal/infra/llm/openai_compat.go

func decodeChatResponse(respBody) (*ChatResponse, error):
    ... unchanged parsing ...
    return &ChatResponse{
        Content:    choice.Message.Content,
        StopReason: choice.FinishReason,
        Tokens:     oaiResp.Usage.TotalTokens,
        Model:      <the model this call used>,  # NEW — thread through from
                                                   # buildOpenAIChatRequest's
                                                   # resolved model, or re-derive
                                                   # via coalesceModel(req.Model, p.model)
    }, nil

# internal/domain/copilot/chat.go (buildChatResult) and
# internal/domain/copilot/suggest_actions.go (generateSalesBrief, generateSuggestedActions)

# BEFORE:
modelName := strings.TrimSpace(s.llm.ModelInfo().ID)

# AFTER:
modelName := strings.TrimSpace(resp.Model)
# (resp is the *ChatResponse already returned by the ChatCompletion call above)
```

## Acceptance Criteria

1. `ChatResponse` has a new `Model` field populated by every `LLMProvider` implementation with the model that specific call actually used (not a static provider default unrelated to the call).
2. `OllamaProvider.ChatCompletion` returns `Model: <the resolved chatModel/override>`, matching what was actually sent to `/api/chat`.
3. `OllamaProvider.ChatCompletion` returns `Tokens: prompt_eval_count + eval_count` from Ollama's real response, no longer hardcoded to 0.
4. `OpenAICompatProvider.ChatCompletion` also populates `Model` (for symmetry and correctness, even though its existing `Tokens` mapping was already correct).
5. `ChatService.buildChatResult` and `ActionService.generateSalesBrief`/`generateSuggestedActions` read the model name from the `ChatResponse` returned by their `ChatCompletion` call, not from `s.llm.ModelInfo().ID`.
6. A copilot chat or sales-brief run against Ollama now records a `usage_event` with the correct chat-model name and non-zero token units (verified via a unit test with a fake/stub provider asserting the recorded `ModelName`/`InputUnits`/`OutputUnits`, and ideally a live check against a running Ollama instance if available).
7. `go test ./internal/infra/llm/... ./internal/domain/copilot/...` passes, including new/updated tests for the `Model` field and Ollama token mapping.
8. No behavior change for the O5c symptom (Support Agent not using the chat model at all) — that remains explicitly out of scope for this task (tracked under O2/O5c, pending the Support Agent architecture decision).

## Result (2026-07-05)

1. **PASS** — `ChatResponse` (`internal/infra/llm/types.go`) gained a `Model string` field; both `OllamaProvider.ChatCompletion` and `OpenAICompatProvider.ChatCompletion` populate it with the model actually resolved/used for that call (`req.Model` override or the provider's chat-model default), not a static lookup.
2. **PASS** — `OllamaProvider.ChatCompletion` (`ollama.go:123-163`) now returns `Model: model` (the resolved `chatModel`/override), verified with a new test `TestOllamaProvider_ChatCompletion_ReportsChatModelNotEmbedModel` using distinct embed/chat models on the same provider instance.
3. **PASS** — `ollamaChatResponse` (`ollama.go:70-76`) now decodes `prompt_eval_count`/`eval_count`; `ChatCompletion` maps `Tokens: PromptEvalCount + EvalCount`. Verified both via updated unit test (`TestOllamaProvider_ChatCompletion_Success`, tokens 10+15=25) and a live call to the running Ollama instance (`POST /api/chat` against `gemma4:26b-a4b-it-qat`), which confirmed the real API does return `prompt_eval_count`/`eval_count` as assumed.
4. **PASS** — `OpenAICompatProvider.ChatCompletion`/`decodeChatResponse` now thread the resolved model through to `ChatResponse.Model`; verified by extending `TestOpenAICompatProvider_ChatCompletion_ModelOverride` to assert `resp.Model == "override-model"`.
5. **PASS** — `chat.go:154` and `suggest_actions.go:375` (`generateSalesBrief`) now read `resp.Model` instead of `s.llm.ModelInfo().ID`.
6. **PASS** — `TestChat_RecordsUsageForGroundedQuery` (updated stub to set `Model: "stub"` on the returned `ChatResponse`) confirms the recorded `usage_event.ModelName` now comes from the actual chat-completion response, not a stateless provider lookup — which is exactly the shape of the O5b defect (Ollama's `ModelInfo()` always returning the embed model regardless of the call made).
7. **PASS** — `go test ./internal/infra/llm/... ./internal/domain/copilot/...` and `go test ./...` (full repo) both pass; `go build ./...` and `go vet` clean.
8. **PASS** — No changes to `internal/domain/agent/agents/support.go` or any Support Agent code path; O5c remains explicitly open pending the O2 architecture decision.
