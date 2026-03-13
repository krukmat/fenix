# AGENT_SPEC Traceability

> Date: 2026-03-09
> Purpose: Keep AGENT_SPEC changes traceable for human and synthetic agents
> Naming source of truth: `docs/agent-spec-overview.md`

---

## Scope

This document defines:
- the canonical identifier chain
- the mapping between use cases, behaviors, components, and phases
- the minimum update protocol for documentation changes

---

## Canonical Identifier Chain

Use this hierarchy in every AGENT_SPEC change:

1. `UC-*`
- stable top-level capability identifier

2. `BEHAVIOR`
- detailed scenario identifier in `snake_case`

3. component
- runtime or service responsible for implementation

4. phase/task
- planned implementation unit

Rule:
- never introduce a new behavior without mapping it to one `UC`
- never introduce a new task without linking it to at least one `UC` or component

---

## Traceability Matrix

| UC | Behavior family | Main components | Main phases | Canonical docs |
|---|---|---|---|---|
| `UC-A2` | `define_workflow*` | `WorkflowService`, `WorkflowRepository` | F2 | overview, use cases, design, plan |
| `UC-A3` | `verify_workflow*` | `Judge`, `SpecParser`, activation flow | F5 | overview, use cases, design, analysis, plan |
| `UC-A4` | `execute_workflow*` | `AgentRunner`, `RunnerRegistry`, `DSLRunner`, `DSLRuntime` | F1, F3, F4 | overview, use cases, design, analysis, plan |
| `UC-A5` | `detect_signal*` | `SignalService`, `EventBus` | F2 | overview, use cases, design, analysis, plan |
| `UC-A6` | `defer_action*` | `Scheduler`, resume handler | F6 | overview, use cases, design, plan |
| `UC-A7` | `human_override*` | `ApprovalService`, `agent_run`, audit | F1, F5 | overview, use cases, design, plan |
| `UC-A8` | `version_workflow*` | `WorkflowService`, versioning lifecycle | F2, F5 | overview, use cases, design, plan |
| `UC-A9` | `delegate_workflow*` | `ProtocolHandler`, `a2aproject/a2a-go` adapter, `RunnerRegistry` | F8 | overview, use cases, design, analysis, plan |

---

## Canonical Documents

These documents are authoritative for active AGENT_SPEC work:

- `docs/agent-spec-overview.md`
- `docs/agent-spec-use-cases.md`
- `docs/agent-spec-design.md`
- `docs/agent-spec-integration-analysis.md`
- `docs/agent-spec-development-plan.md`
- `docs/agent-spec-traceability.md`

Reference-only:

- `docs/agent-spec-transition-plan.md`
- `docs/AGENT_SPEC.md`

Rule:
- if there is a conflict, the canonical set wins

---

## Update Protocol

### When adding a new top-level capability

Update all of:
- `docs/agent-spec-overview.md`
- `docs/agent-spec-use-cases.md`
- `docs/agent-spec-design.md`
- `docs/agent-spec-development-plan.md`
- `docs/agent-spec-traceability.md`

### When adding or renaming a behavior

Update all of:
- `docs/agent-spec-use-cases.md`
- `docs/agent-spec-design.md` if the responsible component changes
- `docs/agent-spec-traceability.md`

### When changing implementation ownership

Update all of:
- `docs/agent-spec-design.md`
- `docs/agent-spec-development-plan.md`
- `docs/agent-spec-traceability.md`

### When changing interoperability direction

Update all of:
- `docs/agent-spec-overview.md`
- `docs/agent-spec-integration-analysis.md`
- `docs/agent-spec-design.md`
- `README.md`

---

## Consistency Checks

Before closing a documentation change, verify:

- every `UC-A*` mentioned outside the overview exists in the overview
- every new `BEHAVIOR` belongs to one existing `UC-A*`
- every new phase task can be traced to a `UC-A*` or a core component
- no reference-only document introduces new naming decisions
- README links point to canonical docs first

---

## Minimal Working Rule for Agents

If an agent needs to make a change and is unsure where to write:

1. check `docs/agent-spec-overview.md`
2. update the canonical document for the affected layer
3. update `docs/agent-spec-traceability.md`
4. only then update any reference-only document if still useful

This is the minimum process to keep future work consistent.
