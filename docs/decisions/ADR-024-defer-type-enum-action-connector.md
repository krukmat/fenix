---
id: ADR-024
title: "Defer TYPE, ENUM, ACTION, CONNECTOR grammar — no implementation until runtime contracts exist"
date: 2026-04-23
status: accepted
deciders: [matias]
tags: [adr, dsl, grammar, language-design, wave5]
related_tasks: [CLSF-55]
related_frs: [FR-212, FR-241]
---

# ADR-024 — Defer TYPE, ENUM, ACTION, CONNECTOR grammar

## Status

`accepted`

## Context

Wave 5 introduced `CALL` and `APPROVE` as the first DSL v1 narrow extensions (CLSF-50 to
CLSF-54). These two statements were chosen because they have clear runtime targets:

- `CALL` → Tool Registry execution pipeline (`internal/domain/tool`)
- `APPROVE` → ApprovalRequest creation + role resolution (ADR-023)

Other grammar constructs referenced in early ULL and DSL v1 discussions are not in that
position:

| Keyword | Intended purpose | Runtime target | Status |
|---|---|---|---|
| `TYPE` | Declare a named schema or DTO | Unknown — schema registry? CRM entity alias? | No runtime |
| `ENUM` | Declare a named set of allowed values | Unknown — policy validator? schema validator? | No runtime |
| `ACTION` | Declare a reusable named action block | Unknown — skill? tool wrapper? macro? | No runtime |
| `CONNECTOR` | Declare an external system integration | Unknown — connector registry? plugin SDK? | No runtime |

None of these have a stable runtime contract. Their semantics, scoping rules, and
execution model are undefined. Implementing lexer/parser/AST support before the contracts
exist would produce dead code that is likely to require breaking changes when the runtime
is eventually designed.

## Decision

**Do not implement `TYPE`, `ENUM`, `ACTION`, or `CONNECTOR` in Wave 5 or any earlier wave.**

These keywords are not added to `dslKeywords` or `dslReservedKeywords`. They are not
tokenized, parsed, or projected into the semantic graph. Any DSL source containing them
will produce a lexer `ILLEGAL` token or a parser error — this is the correct behavior
until the runtime contracts are defined.

## What must be true before implementation

| Prerequisite | Description |
|---|---|
| Runtime contract for each keyword | What does executing `TYPE x { ... }` actually do at runtime? |
| Scoping rules | Are `TYPE` declarations file-local, workspace-global, or workflow-scoped? |
| Interaction with conformance | Do `TYPE`/`ENUM` declarations change the conformance profile of a workflow? |
| Interaction with the semantic graph | Do they produce semantic nodes? Are they referenced by other nodes? |
| Interaction with the policy engine | Can `CONNECTOR` declarations be restricted by RBAC? |

## Why not add them as reserved keywords (like CALL/APPROVE)?

`CALL` and `APPROVE` were added to `dslReservedKeywords` (CLSF-50) because:
1. Their runtime model was already designed (tool execution + approval chain).
2. Reserving them prevented accidental use as identifiers in v0 workflows.

`TYPE`, `ENUM`, `ACTION`, and `CONNECTOR` do not meet criterion 1. Reserving them would
imply a commitment to their eventual syntax that does not yet exist. If the runtime design
changes their meaning, the reservation becomes misleading.

They should be added to `dslReservedKeywords` only when their runtime contracts are
documented and approved — not before.

## Consequences

- Any DSL source using `TYPE`, `ENUM`, `ACTION`, or `CONNECTOR` will fail at lex or parse
  time. This is intentional and expected.
- Future agents must not implement these keywords without a companion ADR that defines
  their runtime contract, scoping rules, conformance profile impact, and semantic graph
  projection.
- This decision is revisited when FR-241 (Agent Studio authoring) or FR-240 (Skills
  builder) advance to implementation — those FRs are the most likely drivers of `ACTION`
  and `CONNECTOR` runtime contracts.
