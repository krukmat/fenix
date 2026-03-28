# The Accountability Layer

*Before this article, there is a [Part 1](https://iotforce.medium.com/crm-is-becoming-an-operating-system-not-a-database-bcf673429ffd). It covers why CRM needs to move from a system that records work to one that participates in it — signals, evidence, governed actions. This article goes one level deeper: how FenixCRM makes that participation auditable.*

---

Deploying an AI agent in a production system surfaces a problem that does not exist with traditional software: the system can be wrong in ways that are hard to detect, hard to reproduce, and hard to explain after the fact.

Hallucinations are the obvious case. But the harder problem is subtler — as agent implementations grow in complexity, the number of steps between a trigger and an action multiplies. Retrieval, reasoning, tool selection, approval, execution. Each step introduces a decision point. And as those chains grow longer, keeping a human meaningfully in the loop gets harder, not easier.

The instinct is to add more monitoring. The real fix is to design for auditability from the start — so that every step in the chain is recorded, every decision is traceable, and the human reviewing it has enough context to actually understand what happened.

That is what this article is about. Four mechanisms in FenixCRM that make agentic execution auditable by design: AgentRun tracing, policy enforcement, cost control, and eval-gating.

---

## AgentRun: the complete record of a decision

![AgentRun lifecycle: from trigger to audit](article-assets/diagram-8-agentrun-lifecycle.png)

Every agent execution in FenixCRM produces a single record called an `AgentRun`. Not a log entry. Not a status flag. The full history of what happened at every step.

```
AgentRun {
  id, agent_id, triggered_by, trigger_type,
  inputs,
  retrieval_queries, retrieved_evidence,
  reasoning_trace,
  tool_calls,
  output, status,
  cost_tokens, cost_euros, latency_ms,
  audit_events: [who executed, perms checked, approvals],
  created_at, updated_at
}
```

When an agent runs, the record captures: what triggered it, what it searched for, what evidence it retrieved, what it reasoned, which tools it called with what parameters, whether any approvals were required, the final output, the cost in tokens and euros, and the full audit trail.

This matters specifically for the hallucination and complexity problem. When an agent produces a wrong output — whether from a retrieval failure, a reasoning error, or a tool call with bad parameters — the AgentRun tells you exactly where in the chain the failure occurred. Not "the agent got it wrong" but which retrieval query returned stale data, or which reasoning step led to the wrong tool call, or which approval was bypassed.

Without this, debugging an agent failure means reconstructing a chain of events from incomplete logs. With it, the full execution is queryable after the fact.

---

## Policy enforcement at four points

![Policy enforcement: 4 points in the execution path](article-assets/diagram-7-policy-enforcement.png)

Keeping humans in the loop requires knowing where in the execution path a human needs to intervene — and making that intervention reliable, not optional.

In FenixCRM, every agent action passes through four enforcement points before anything executes.

**1. Retrieval** — Before any evidence reaches the prompt, the system checks what the requesting user is allowed to see. RBAC/ABAC filters apply here. If a no-cloud policy is active, PII fields are redacted before leaving the retrieval layer — regardless of which LLM is configured downstream. A hallucination that cites a document the user was never supposed to see is a governance failure, not just a model failure.

**2. Prompt building** — Sensitivity labels travel with evidence through the pipeline. What gets included in the prompt, what gets redacted, and what gets withheld are decisions made by policy, not by the model. The model cannot override what the policy layer removes.

**3. Tool execution** — The model proposes tool calls. It cannot execute them directly. Each tool has a registered schema, a permission set, and a rate limit. Invalid parameters, missing permissions, or exceeded rate limits result in a denied call — logged, not silently dropped. If the action requires human approval, it pauses here until a decision is made.

**4. Output formatting** — Before a response reaches the user, the output layer applies visibility rules. The same agent, the same question, a manager and a sales rep — they may see different outputs. By design.

The approval gate at point 3 is where the human-in-the-loop mechanism lives. It is not a dashboard you check after the fact. It is a hard stop in the execution path — the agent cannot proceed until a human with the right permissions decides.

---

## Cost control as a legibility problem

As agent chains grow more complex, their cost becomes harder to attribute. A single agent interaction may involve multiple retrieval passes, several LLM calls, and one or more tool executions. Multiply that across users and tenants, and the spend becomes invisible until it is a problem.

Every agent and role in FenixCRM has configurable quotas: tokens per day, cost per day in euros, executions per day. When a quota is reached, the system responds in a structured way:

- **Circuit breaker**: new executions pause until the window resets
- **Graceful degradation**: switch to a cheaper model, reduce context, raise the abstention threshold
- **Alert**: notify the agent owner and the workspace admin

And because every `AgentRun` records `cost_euros`, the spend is attributable at the execution level — not just as a monthly total. You can answer: what did this specific agent action cost? What is the average cost per resolved ticket? Where is the spend going?

That turns AI cost from an opaque bill into an operational metric you can actually manage.

---

## Eval-gating: making prompt changes safe

Prompt changes are not configuration changes. They change system behavior in ways that are often non-obvious and can interact badly with the retrieval and tool layers in unexpected ways.

In FenixCRM, a prompt or policy update cannot reach production until it passes a set of quality gates:

| Gate | What it measures | Threshold |
|------|-----------------|-----------|
| Groundedness | % outputs with sufficient evidence | >95% |
| Accuracy | % correct vs. CRM ground truth | configurable |
| Abstention correctness | % false positives in self-denial | >98% |
| Policy adherence | policy violations | 0 |

If any gate fails, the update does not ship. Every prompt version is stored. Rollback to any previous version is a single operation. The system knows which version is active per agent per environment.

This is directly related to the auditability problem. If a prompt change degrades groundedness — meaning the agent starts producing outputs less anchored to retrieved evidence — that is a hallucination risk increase. The eval gate catches it before it reaches production. If it reaches production and degrades after the fact, the rollback path is clear.

---

## Self-hosted and model-agnostic

The tracing and governance described above only work if you control the infrastructure they run on. An audit trail stored in a vendor's system is an audit trail you cannot fully trust or fully own.

FenixCRM runs entirely in your own infrastructure: Go backend, SQLite, BFF, and LLM inference. A single `docker-compose up`. No mandatory external dependency.

The LLM adapter is model-agnostic. The same agent runs against a local Ollama instance, a self-hosted vLLM deployment, or a cloud API. Switching providers does not touch the agent logic or the governance layer.

The no-cloud policy enforces this at the system level. When active, PII is redacted before any data leaves the retrieval layer — regardless of which model is configured. The policy is in the code path, not in a documentation page.

---

## What this adds up to

The four mechanisms above — AgentRun tracing, policy enforcement at four points, cost quotas, and eval-gating — are not independent features. They are the answer to the same problem: as agent implementations grow in complexity, how do you keep a human meaningfully in the loop?

The answer is not more dashboards. It is a system where every execution step is recorded, every sensitive action requires explicit approval, every prompt change is gated on quality, and the full audit trail is yours to query.

FenixCRM is not a proposal. All of this is implemented and tested. Single `docker-compose up`. Open source: [REPO_URL]

---

*Part 1: [CRM Is Becoming an Operating System, Not a Database](https://iotforce.medium.com/crm-is-becoming-an-operating-system-not-a-database-bcf673429ffd)*
