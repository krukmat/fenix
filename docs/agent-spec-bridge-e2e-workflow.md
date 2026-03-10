# Bridge E2E Workflow Pilot

This document captures the minimal bridge workflow used to validate `F3.8`.

## Workflow

```json
{
  "name": "resolve_support_case_bridge",
  "trigger": { "event": "case.created" },
  "steps": [
    {
      "id": "step_set_status",
      "condition": {
        "left": "case.priority",
        "operator": "IN",
        "right": ["high", "urgent"]
      },
      "action": {
        "verb": "SET",
        "target": "case.status",
        "args": { "value": "resolved" }
      }
    },
    {
      "id": "step_notify_owner",
      "action": {
        "verb": "NOTIFY",
        "target": "salesperson",
        "args": { "message": "review resolved case" }
      }
    }
  ]
}
```

## Expected Runtime Path

```mermaid
sequenceDiagram
    participant E as case.created
    participant R as SkillRunner
    participant P as PolicyEngine
    participant T as ToolRegistry
    participant S as agent_run_step

    E->>R: bridge workflow trigger
    R->>S: insert bridge_step start
    R->>P: check update_case permission
    P-->>R: allow
    R->>T: update_case(case.status=resolved)
    R->>S: mark success
    R->>S: insert next bridge_step
    R->>P: check create_task permission
    P-->>R: allow
    R->>T: create_task(review resolved case)
    R->>S: mark success
```

## Observable Outcome

- `agent_run.status = success`
- `tool_calls` include `update_case` and `create_task`
- `agent_run_step` includes one `bridge_step` row per workflow step
- both bridge steps finish in `success`
