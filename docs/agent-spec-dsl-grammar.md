# AGENT_SPEC DSL Grammar v0

**Status**: Initial grammar baseline
**Phase**: AGENT_SPEC - Fase 4 DSL Foundation
**Purpose**: define the first stable DSL shape before implementing lexer, parser, and runtime

---

## Design Intent

This grammar is intentionally small.

It exists to formalize the execution model already validated in Phase 3:

- trigger-driven execution
- conditional branching
- mapped mutations through tools
- sub-agent invocation
- notification actions

`WAIT` and `DISPATCH` are reserved but excluded from this grammar version.

---

## Grammar Shape

The initial DSL is line-oriented and indentation-sensitive.

Top-level structure:

```text
WORKFLOW <workflow_name>
ON <event_name>
<statement>*
```

Statements supported in v0:

- `IF <expression>:`
- `SET <target> = <value>`
- `NOTIFY <target> WITH <value>`
- `AGENT <agent_name> [WITH <object_or_identifier>]`

Blocks:

- only `IF` opens a nested block in v0
- nested statements are defined by indentation

---

## Reserved Keywords

Keywords recognized in v0:

- `WORKFLOW`
- `ON`
- `IF`
- `SET`
- `NOTIFY`
- `WITH`
- `AGENT`

Reserved for later phases, but not executable in v0:

- `WAIT`
- `DISPATCH`
- `SURFACE`

---

## Identifiers

Allowed identifier classes:

- workflow names: `resolve_support_case`
- events: `case.created`
- field references: `case.status`, `lead.score`
- agent names: `evaluate_intent`, `search_knowledge`

Identifier rule:

- start with letter or `_`
- continue with letters, digits, `_`, `.`

---

## Literals

Supported literals in v0:

- string: `"resolved"`
- number: `0.8`, `48`
- boolean: `true`, `false`
- array: `["high", "urgent"]`
- object: `{"value": "resolved"}`
- null: `null`

---

## Expressions

Supported comparison operators:

- `==`
- `!=`
- `>`
- `<`
- `>=`
- `<=`
- `IN`

Examples:

```text
IF case.priority IN ["high", "urgent"]:
IF lead.score >= 0.8:
IF evidence.top_score == null:
```

Expression rules in v0:

- left side must be a field or identifier reference
- right side must be a literal or identifier reference
- no boolean chaining yet (`AND`, `OR`, `NOT` out of scope)

---

## Statements

### 1. WORKFLOW

Declares the workflow name.

Example:

```text
WORKFLOW resolve_support_case
```

Rules:

- required exactly once
- must be the first non-empty statement

### 2. ON

Declares the trigger event.

Example:

```text
ON case.created
```

Rules:

- required exactly once
- must appear after `WORKFLOW`

### 3. IF

Conditional block.

Example:

```text
IF case.priority IN ["high", "urgent"]:
  NOTIFY salesperson WITH "review this case"
```

Rules:

- requires trailing `:`
- requires at least one indented child statement

### 4. SET

Mapped mutation statement.

Example:

```text
SET case.status = "resolved"
SET case.priority = "high"
```

Rules:

- target must be a dotted field reference
- value must be a literal or identifier reference
- actual mutation still resolves through `VerbMapper -> ToolRegistry`

### 5. NOTIFY

Mapped notification statement.

Example:

```text
NOTIFY contact WITH "We applied a solution and resolved your case."
NOTIFY salesperson WITH "review this resolved case"
```

Rules:

- target is required
- `WITH` payload is required
- actual side effect still resolves through `VerbMapper -> ToolRegistry`

### 6. AGENT

Sub-agent invocation.

Examples:

```text
AGENT evaluate_intent
AGENT search_knowledge WITH case
```

Rules:

- agent name is required
- `WITH` payload is optional in v0
- execution still resolves through runtime and runner/orchestrator integration

---

## Valid Example

```text
WORKFLOW resolve_support_case
ON case.created

IF case.priority IN ["high", "urgent"]:
  NOTIFY salesperson WITH "review this case"

SET case.status = "resolved"
NOTIFY contact WITH "We applied a solution and resolved your case."
AGENT search_knowledge WITH case
```

---

## Invalid Examples

Missing workflow header:

```text
ON case.created
SET case.status = "resolved"
```

Invalid because:

- `WORKFLOW` is required first

Invalid unsupported verb:

```text
WORKFLOW resolve_case
ON case.created
EMAIL contact WITH "hello"
```

Invalid because:

- `EMAIL` is not a supported v0 verb

Invalid `IF` without block:

```text
WORKFLOW resolve_case
ON case.created
IF case.priority == "high":
SET case.status = "resolved"
```

Invalid because:

- `IF` requires an indented child block

Invalid reserved-but-disabled statement:

```text
WORKFLOW follow_up_case
ON case.created
WAIT 48 hours
```

Invalid in v0 because:

- `WAIT` is reserved for a later phase

---

## Alignment with Phase 3 Bridge

Bridge format to DSL alignment:

- bridge `trigger.event` -> DSL `ON <event>`
- bridge `SET` -> DSL `SET`
- bridge `NOTIFY` -> DSL `NOTIFY`
- bridge `AGENT` -> DSL `AGENT`
- bridge `condition` -> DSL `IF`

What changes from bridge to DSL:

- bridge is JSON-shaped
- DSL is author-facing and line-oriented
- the runtime semantics stay aligned on purpose

---

## v0 Constraints

Explicitly excluded from this grammar version:

- `WAIT`
- `DISPATCH`
- `SURFACE`
- `ELSE`
- chained boolean conditions
- loops
- user-defined functions
- imports or includes

---

## Audit Checklist

Use this checklist to validate future parser work against this document:

- `WORKFLOW` and `ON` are required and ordered
- `IF` is indentation-sensitive
- only `SET`, `NOTIFY`, `AGENT`, `IF` are executable in v0
- `WAIT`, `DISPATCH`, `SURFACE` are reserved but rejected
- dotted field paths are accepted as identifiers
- expression operators match the bridge comparison set
