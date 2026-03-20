# AGENT_SPEC Bridge Format

**Status**: Drafted in F3.1
**Purpose**: transitional declarative format before the final DSL

---

## Why it exists

This format exists to validate the execution model before introducing the full DSL parser.

It is intentionally smaller than the final language:

- one trigger
- sequential steps
- optional per-step condition
- supported verbs only from the future core set

It is not meant to become a second permanent language.

---

## Format

```json
{
  "name": "qualify_lead_bridge",
  "trigger": {
    "event": "lead.created"
  },
  "steps": [
    {
      "id": "step_1",
      "action": {
        "verb": "AGENT",
        "target": "evaluate_intent"
      }
    },
    {
      "id": "step_2",
      "condition": {
        "left": "lead.score",
        "operator": ">=",
        "right": 0.8
      },
      "action": {
        "verb": "SET",
        "target": "lead.status",
        "args": {
          "value": "qualified"
        }
      }
    }
  ]
}
```

---

## Supported concepts

- `trigger.event`
- sequential `steps`
- optional `condition`
- action verbs:
  - `SET`
  - `NOTIFY`
  - `AGENT`

---

## Explicitly unsupported in F3.1

- `WAIT`
- `DISPATCH`
- nested blocks
- loops
- full expression language
- final DSL syntax

---

## Design alignment

The bridge format is aligned with Fase 4 in these ways:

- verbs use the same conceptual vocabulary as the future DSL
- all mutations are expected to go through `ToolRegistry`
- policy and approvals remain external to the format and are enforced by the runtime
- step-level tracing maps directly to future `agent_run_step` semantics

---

## References

- `docs/agent-spec-design.md`
- `docs/agent-spec-phase3-analysis.md`
- `docs/tasks/task_agent_spec_f3_1.md`
