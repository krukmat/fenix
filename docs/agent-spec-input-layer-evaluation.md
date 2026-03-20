# AGENT_SPEC Input Layer Evaluation

## Summary

This note evaluates whether Fenix should add a future input layer that translates:

- BPMN -> DSL
- Natural language -> DSL

Current decision:

- **BPMN -> DSL** remains a **future roadmap item**
- **NL -> DSL** remains **out of the immediate runtime roadmap**
- neither layer should be introduced into the current runtime path now

The current runtime path remains:

`authoring -> Judge -> DSL -> Runtime`

## Why

The current AGENT_SPEC stack is now mature enough to run:

- verified workflows
- delayed execution
- internal dispatch
- A2A-first dispatch
- MCP-first tool integration
- surfaced signals

Adding a new input layer now would not increase runtime capability. It would add:

- ambiguity before verification
- a second source of failure before Judge
- more difficult auditability
- pressure to support partial translations as if they were stable workflow definitions

That is the wrong tradeoff at the current stage.

## BPMN -> DSL

### Fit

BPMN could become useful later for:

- organizations that already model flows in BPMN
- business users who need a visual process abstraction
- importing process definitions from external systems

### Risks

- BPMN parsing and normalization is a project of its own
- semantic mismatches between BPMN constructs and current DSL verbs are non-trivial
- the translation layer would need its own validation, diagnostics, and versioning
- a weak BPMN compiler would create misleading "valid" DSL output

### Decision

Keep BPMN -> DSL as a **future input adapter**, not a near-term implementation target.

If pursued later, it should be:

- optional
- fully compiled into DSL before runtime
- always re-verified by Judge after translation

It must never bypass Judge or feed the runtime directly.

## NL -> DSL

### Fit

Natural language is useful for:

- ideation
- scaffolding
- assisted drafting
- generating first-pass workflows for review

### Risks

- ambiguity is intrinsic
- determinism is weak
- repeatability is poor without strong constraints
- auditability becomes difficult if generated text is treated as authoritative workflow logic

### Decision

Do **not** treat NL -> DSL as a runtime or activation-time input layer.

At most, it can exist later as:

- an **authoring assistant**
- producing a **draft DSL proposal**
- always requiring explicit human review plus Judge verification before activation

That keeps the language model in the authoring loop, not in the execution trust boundary.

## Architecture Rule

Any future input layer must follow this shape:

```text
Input Layer -> DSL Draft -> Judge -> Workflow Lifecycle -> Runtime
```

Not this:

```text
Input Layer -> Runtime
```

This rule preserves:

- auditability
- determinism
- activation safety
- one executable source of truth

## Decision Record

### BPMN -> DSL

- status: keep in future roadmap
- priority: low
- runtime impact now: none
- implementation now: no

### NL -> DSL

- status: postpone as runtime feature
- priority: low
- allowed future role: authoring assistant only
- implementation now: no

## Recommended Next Position

The roadmap should continue prioritizing:

- execution reliability
- interoperable dispatch
- MCP/A2A maturity
- migration of high-value workflows

Input-layer translation should only return when:

- the DSL surface is more stable
- dispatch semantics are fully settled
- there is a concrete product need for BPMN import or NL-assisted authoring
