You are implementing "Carta" - a structured, machine-readable governance language for AI agent behavior in FenixCRM. Carta replaces the free-form `spec_source` field in the `Workflow` entity with a declarative grammar that the Judge can statically verify and the runtime can dynamically enforce.

## Your two source-of-truth documents

Read these before writing a single line of code:

1. `docs/carta-spec.md` - grammar (EBNF), 5 interaction scenarios, integration map
2. `docs/carta-implementation-plan.md` - 22 atomic tasks, per-task specs, dependency diagram

## Task file convention

For every task you implement, first create its task file at:
`docs/tasks/task_carta_<phase>_<num>.md`

Use the normalized task ID in lowercase with underscores:
- `FC-L.1` -> `docs/tasks/task_carta_fc_l_1.md`
- `FC-P.9` -> `docs/tasks/task_carta_fc_p_9.md`
- `FC-R.12` -> `docs/tasks/task_carta_fc_r_12.md`

Follow the exact format of an existing task file, e.g. `docs/tasks/task_agent_spec_f3_1.md`:
- Status, Phase, Depends on, Required by
- Objective, Scope, Out of Scope
- Acceptance Criteria, Quality Gates, References, Sources of Truth

If this prompt and `docs/carta-implementation-plan.md` ever disagree on dependency order, treat the implementation plan as the tie-breaker.

## Execution order

Start with the tasks that have no dependencies - they can run in parallel:

**Round 1 (no deps, start all at once):**
- FC-L.1 - `internal/domain/agent/carta_token.go`
- FC-P.1 - `CartaSummary` + `CartaGrounds` structs in `internal/domain/agent/carta_ast.go`
- FC-P.2 - `CartaPermit` + `CartaRate` + `CartaApprovalConfig` structs (same file)
- FC-P.3 - `CartaDelegate` + `CartaInvariant` + `CartaBudget` structs (same file)

**Round 2 (unblocked after Round 1):**
- FC-L.2 - `internal/domain/agent/carta_lexer.go` (needs FC-L.1)
- Then FC-P.4 through FC-P.8 in parallel (each needs FC-L.2 + FC-P.1/2/3)

**Round 3:**
- FC-P.9 - `ParseCarta()` orchestrator (needs all block parsers)
- Then FC-P.10 - parser tests (needs FC-P.9)

**Round 4 (unblocked after FC-P.9):**
- FC-J.1, FC-J.2, FC-J.4, FC-J.5 in parallel
- FC-R.1, FC-R.3, FC-R.6, FC-R.7 in parallel

**Round 5:**
- FC-J.3 (needs FC-J.2 + FC-P.9)
- FC-R.2 (needs FC-R.1), FC-R.4 (needs FC-R.3)
- FC-R.8 (needs FC-R.6), FC-R.9 (needs FC-R.7)
- FC-R.10 (needs FC-R.1), FC-R.11 (needs FC-R.3)

**Round 6:**
- FC-J.6 (needs FC-J.3 + FC-J.4 + FC-J.5)
- FC-R.5 (needs FC-R.2 + FC-R.4)

**Round 7:**
- FC-J.7 (needs FC-J.6)

**Final:**
- FC-R.12 - end-to-end integration test (needs FC-R.5 + FC-R.9)

## Interpretation notes

Use these clarifications while implementing:

- `SKILL` remains reserved in this tranche. Do not add new parser or runtime support for `SKILL` blocks beyond what is explicitly listed in the task plan. If a Carta source contains `SKILL`, fail fast with an explicit unsupported parse error instead of inventing partial behavior.
- Parser-level non-fatal warnings must travel with the parse result. Keep `ParseCarta(source string) (*CartaSummary, error)` as specified and store parser warnings on `CartaSummary` itself so callers can inspect them without changing the public function signature.
- Check 11 uses the same behavior inventory the Judge already has available in its current flow. Do not invent a second free-form parsing path for Carta sources. If no behavior list is available to the Judge, Check 11 is a no-op.
- Carta replaces free-form `spec_source` for Carta workflows only. Legacy workflows that still start with `CONTEXT` must continue to go through the existing partial-spec flow unchanged.
- Prefer small, explicit unsupported errors over speculative implementation when the spec mentions future scope that is not backed by a task in `docs/carta-implementation-plan.md`.

## Hard constraints

- **Do not modify** `lexer.go`, `token.go`, `spec_parser.go` - Carta lives in new files
- **Do not touch** the DSL parser (`parser.go`, `ast.go`) - DSL syntax is unchanged
- `CartaLexer` must reuse `emitIndentationTokens` from the existing `Lexer` - no duplicate INDENT/DEDENT logic
- `knowledge.ConfidenceLevel` already exists in `internal/domain/knowledge/models.go` - import it, do not redefine
- `TO` and `WITH` already exist in `token.go` - do not redefine
- `isCartaSource()` in `judge.go` must leave the existing free-form spec path (`ParsePartialSpec`) fully intact
- `RunContext` gets exactly one new field: `GroundsValidator *GroundsValidator` - nullable, downstream checks `!= nil`
- Preflight order in `dsl_runner.go` is fixed: DelegateEvaluator -> GroundsValidator -> DSLRuntime.ExecuteProgram

## Quality gate for each task

```bash
go test ./internal/domain/agent/... -v
go build ./...
```

Both must be green before marking a task complete.

## Global acceptance criteria (final check)

```bash
go test ./internal/domain/agent/... -v        # 0 failures
go test ./internal/domain/workflow/... -v     # 0 failures
go build ./...                                # 0 errors
```

Plus three behavioral assertions:
- Scenario A: `AgentRun.status == "success"` (Carta + DSL consistent, evidence sufficient)
- Scenario D: `AgentRun.status == "abstained"`, DSLRuntime never called (grounds not met)
- Scenario E: `AgentRun.status == "delegated"`, 0 retrieval tokens spent (delegate condition fires first)

## Backward compatibility (non-negotiable)

Any workflow whose `spec_source` starts with `CONTEXT` (free-form) must still pass the Judge without changes. Add at least one regression test for this in FC-J.7.
