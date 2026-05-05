---
doc_type: plan
id: WRAPCHECK-GLOBAL-PLAN
title: "Global wrapcheck remediation plan"
status: complete
phase: qa-hardening
week: 19
tags: [plan, go, lint, wrapcheck, qa, observability, debt]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: []
files_affected:
  - Makefile
  - scripts/qa-go-prepush.sh
  - cmd/frtrace/main.go
  - internal/api/handlers
  - internal/domain/agent
  - internal/domain/agent/agents
  - internal/domain/audit
  - internal/domain/auth
  - internal/domain/copilot
  - internal/domain/crm
  - internal/domain/eval
  - internal/domain/knowledge
  - internal/domain/policy
  - internal/domain/scheduler
  - internal/domain/signal
  - internal/domain/tool
  - internal/domain/usage
  - internal/domain/workflow
  - internal/infra/llm
  - internal/infra/sqlite
  - internal/lsp
  - internal/server
  - scripts/e2e_seed_mobile_p2.go
  - tests/bdd/go
created: 2026-05-05
completed: 2026-05-05
---

# Global wrapcheck remediation plan

## Objective

Bring the repository to a state where `make wrapcheck-gate` passes with
`WRAPCHECK_SCOPE=./...` and the Go pre-push hook can publish changes without
lint suppression or scope reduction.

## Baseline

- Current gate command: `make wrapcheck-gate`
- Current findings: 385
- Current files impacted: 93
- Current package buckets impacted: 33
- Current failure classes:
  - 291 raw external-package errors
  - 84 raw interface-method errors

## Decision

Keep `wrapcheck` fail-hard and global.

Do not revert to scoped enforcement.
Do not add broad path exclusions.
Do not disable `wrapcheck` in `.golangci.yml`.

The cleanup strategy is progressive code remediation, not policy rollback.

## Remediation principles

- Wrap every external or interface error at the boundary where local operation
  context becomes known.
- Use short, operation-specific messages such as `load policy versions`,
  `commit knowledge ingest transaction`, or `publish diagnostics payload`.
- Preserve error identity with `%w`.
- Avoid generic wrappers like `operation failed`.
- Keep behavior unchanged except for enriched errors.
- Re-run `make wrapcheck-gate` after each package wave.

## Error pattern taxonomy

### Database boundary returns

Common signatures:

- `BeginTx`
- `Commit`
- `ExecContext`
- `QueryContext`
- `Scan`
- `Rows.Err`
- `Rows.Close`
- `RowsAffected`

Preferred remediation pattern:

```go
if err != nil {
	return nil, fmt.Errorf("list policy versions: %w", err)
}
```

### Interface boundary returns

Common signatures:

- domain service interfaces
- policy enforcers
- workflow resolvers
- tool authorizers
- schedulers
- MCP resource providers

Preferred remediation pattern:

```go
if err != nil {
	return nil, fmt.Errorf("build evidence pack: %w", err)
}
```

### Serialization and transport boundaries

Common signatures:

- `json.Marshal`
- `json.Unmarshal`
- `decoder.Decode`
- `http.NewRequestWithContext`
- `client.Do`
- `io.ReadAll`
- `io.ReadFull`
- `bufio.Reader.ReadString`
- `os.Open`
- `os.Stat`

Preferred remediation pattern:

```go
if err != nil {
	return nil, fmt.Errorf("marshal insights shadow payload: %w", err)
}
```

## Remediation waves

### Wave 0 - Baseline and governance

Status: complete

- Capture current failure inventory
- Lock the gate to fail-hard global scope
- Document counts, hotspots, and remediation rules

Exit criteria:

- Task record exists
- Plan exists
- Push failure mode is confirmed by the real pre-push hook

### Wave 1 - Fast low-fanout packages

Status: complete

Observed reduction: 46 findings removed from the global gate snapshot
(`385 -> 339`) with zero remaining findings in the Wave 1 package set.

Target packages:

- `internal/server`
- `internal/domain/auth`
- `internal/lsp`
- `internal/lsp/handlers`
- `cmd/frtrace`
- `internal/domain/signal`
- `internal/infra/sqlite`
- `internal/domain/scheduler`
- `internal/domain/eval`

Reasoning:

These packages have small to medium finding counts and relatively direct error
flows. They should reduce the global count quickly and validate the remediation
style before attacking the heavy orchestration packages.

Exit criteria:

- Zero `wrapcheck` findings in all Wave 1 packages
- No behavior regressions in existing tests for those packages

### Wave 2 - HTTP and policy boundary packages

Status: complete

Observed reduction: 54 findings removed from the global gate snapshot
(`339 -> 285`) with zero remaining findings in the Wave 2 package set.

Target packages:

- `internal/api/handlers`
- `internal/domain/usage`
- `internal/domain/policy`
- `internal/domain/audit`

Reasoning:

These files sit on boundary layers where operator-visible error context matters.
They also contain repeated DB and interface call patterns that are easy to
normalize once one pattern is chosen.

Exit criteria:

- Zero `wrapcheck` findings in all Wave 2 packages
- Handler and policy tests remain green

### Wave 3 - Tooling and copilot orchestration

Status: complete

Observed reduction: 38 findings removed from the global gate snapshot
(`285 -> 247`) with zero remaining findings in the Wave 3 package set.

Target packages:

- `internal/domain/tool`
- `internal/domain/copilot`
- `internal/infra/llm`

Reasoning:

These packages bridge policy, evidence, LLM, MCP, and persistence boundaries.
They need careful message wording so failures remain diagnosable without adding
noise.

Exit criteria:

- Zero `wrapcheck` findings in all Wave 3 packages
- Tool and copilot tests remain green

### Wave 4 - Knowledge and CRM service layers

Status: complete

Observed reduction: 49 findings removed from the global gate snapshot
(`247 -> 198`) with zero remaining findings in the Wave 4 package set.

Target packages:

- `internal/domain/knowledge`
- `internal/domain/crm`
- `internal/domain/workflow`

Reasoning:

These packages are database-heavy and contain many repeated raw returns from
service and query methods. They are high-value because the same remediation
pattern can eliminate large chunks of debt.

Exit criteria:

- Zero `wrapcheck` findings in all Wave 4 packages
- Service and integration tests remain green

### Wave 5 - Agent runtime and agent definitions

Status: complete

Observed reduction: 120 findings removed from the global gate snapshot
(`198 -> 78`) with zero remaining findings in the Wave 5 package set.

Target packages:

- `internal/domain/agent`
- `internal/domain/agent/agents`

Reasoning:

This is the largest hotspot. The package combines orchestration, runtime,
protocol, DSL, and adapter boundaries. It should be attacked after lower-risk
patterns have been standardized elsewhere.

Exit criteria:

- Zero `wrapcheck` findings in both agent package groups
- Agent runtime and orchestration tests remain green

### Wave 6 - Operational support code

Status: complete

Observed reduction: 78 findings removed from the global gate snapshot
(`78 -> 0`) with zero remaining findings in the Wave 6 package set and a
clean global `make wrapcheck-gate` result.

Target packages:

- `scripts/e2e_seed_mobile_p2.go`
- `tests/bdd/go`

Reasoning:

These are still in gate scope because the gate runs on `./...`. They are lower
product risk than production packages but must be cleaned if the repository is
to pass the global gate honestly.

Exit criteria:

- Zero `wrapcheck` findings in scripts and BDD support code
- Seed and BDD tests still execute successfully

## Order of execution

1. Use the Wave 1 wrapper wording conventions as the baseline for later waves
2. Apply the established wrapper wording conventions through Wave 2 and Wave 3
3. Attack the database-heavy Wave 4 packages
4. Dedicate a focused pass to the agent runtime hotspot in Wave 5
5. Clean operational support code in Wave 6
6. Re-run `make wrapcheck-gate`
7. Re-run full `bash scripts/qa-go-prepush.sh`
8. Retry `git push`

## Suggested wrapper wording conventions

- `load ...`
- `list ...`
- `get ...`
- `create ...`
- `update ...`
- `delete ...`
- `parse ...`
- `marshal ...`
- `unmarshal ...`
- `start ...`
- `execute ...`
- `commit ...`
- `schedule ...`
- `publish ...`

Example:

- `fmt.Errorf("commit approval transaction: %w", err)`
- `fmt.Errorf("decode prompt experiment request: %w", err)`
- `fmt.Errorf("call MCP tool %q: %w", name, err)`

## Definition of done

- `make wrapcheck-gate` passes with global scope unchanged
- `bash scripts/qa-go-prepush.sh` passes
- `git push origin main` is no longer blocked by `wrapcheck`
- No broad exclusions or policy rollbacks were introduced to get there
