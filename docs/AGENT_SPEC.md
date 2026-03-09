# Agent Specification Framework
> Born from a conversation about what a CRM looks like when AI is the core, not a layer on top.

**Principle:** Describe *what* must happen, never *how* to implement it.  
**Audience:** Coding agents, LLM judges, and humans verifying consistency.  
**Core insight:** The DSL is the program. The DSL is also the message. Generated code is disposable.

---

## Document Status

This document remains the original conceptual source for AGENT_SPEC.

Current canonical implementation-oriented documents:
- `docs/agent-spec-overview.md`
- `docs/agent-spec-use-cases.md`
- `docs/agent-spec-design.md`
- `docs/agent-spec-integration-analysis.md`
- `docs/agent-spec-development-plan.md`

Rule:
- use this file for conceptual intent
- use the canonical set for naming, analysis, design, and phased implementation

---

## The Problem This Solves

Traditional CRMs model the world as it was in the 1990s:
Contact → Account → Opportunity → Stage.
AI is bolted on top of that model as a copilot.

This framework assumes the opposite:
**The unit of truth is not the contact. It is the signal.**
The DSL defines business intent. The Runtime executes it.
CRM state is a consequence of execution — not manual data entry.

This is closer to SQL than to pseudocode.
Nobody thinks of SQL as "pseudocode for disk operations."
SQL *is* the language. The engine handles the rest.
This DSL *is* the program. The CRM is the engine.

---

## The Pipeline

```
┌─────────────────────────────────────────┐
│              INPUT                       │
│  Human / MCP / A2A                      │
└─────────────────┬───────────────────────┘
                  ↓
         BPMN translation
         (resolves business concept ambiguity)
                  ↓
                 DSL
         (executable source of truth)
                  ↓
               Judge
         (verifies consistency)
                  ↓
              Runtime
                  ↓
┌─────────────────────────────────────────┐
│              OUTPUT                      │
│  CRM operations / MCP / A2A             │
│  ACCEPTED / REJECTED / DELEGATED        │
└─────────────────────────────────────────┘
```

Each layer has one owner:
- The human or agent owns the input
- BPMN owns the business semantics — open standard, not proprietary
- The DSL is the source of truth
- The Judge owns consistency verification
- The Runtime owns execution
- Nobody manually owns the CRM state — it emerges

### MMD diagrams

MMD is an optional inspection tool, not a pipeline layer.
If a human wants to visualize a DSL workflow, MMD is generated on demand.
It does not block or gate execution.

---

## Why BPMN

Natural language is ambiguous in two ways: words and business concepts.

"Qualify a lead" can mean five different things in five different companies.
BPMN resolves the conceptual ambiguity before the DSL is written.

BPMN is the right choice because:
- It is an open standard (OMG — Object Management Group)
- It models exactly what this system needs: events, decisions, flows, actors
- It is already understood by business analysts and agents alike
- It separates *what the business does* from *how technology implements it*

When input comes from a human, BPMN translates natural language to DSL.
When input comes from MCP or A2A, the agent already speaks DSL — BPMN translation is skipped.

---

## BPMN Concepts Used

Only the subset relevant to business workflows. No technical BPMN extensions.

```
START EVENT    → something triggers the workflow
END EVENT      → the workflow reaches a defined outcome
TASK           → a unit of work performed by an Actor or Agent
GATEWAY        → a decision point with explicit conditions
POOL           → the boundary of a participant (Salesperson, Contact, System)
MESSAGE FLOW   → communication between pools
```

---

## Protocol Modes

The DSL operates in three modes:

```
EXECUTE    → Runtime interprets and runs the workflow locally
DISPATCH   → DSL is sent to another agent via MCP or A2A for execution
VERIFY     → Judge validates consistency without executing
```

An agent receiving a DSL workflow must respond with one of:

```
ACCEPTED   → will execute this workflow
REJECTED   → divergence found + reason
DELEGATED  → forwarding to another agent + reason
```

---

## Spec Anatomy

Every spec has four blocks. All four are required.

```
CONTEXT      → why this workflow exists
ACTORS       → who participates and what they own
BEHAVIOR     → what must happen, as observable events
CONSTRAINTS  → what must never happen
```

---

## CONTEXT

One paragraph. The business problem and the intent.
No technology. No solution. No implementation verbs.

```
CONTEXT
  Salespeople lost deals because intent signals from contacts
  were buried in unstructured conversation history.
  This workflow defines how the system identifies, surfaces,
  and acts on high-intent signals without requiring manual data entry.
```

---

## ACTORS

Named roles with a single responsibility each.
Each Actor maps to a BPMN Pool.

```
ACTORS
  Salesperson   → owns the relationship and approves agent actions
  Contact       → external person whose intent is being assessed
  Workflow      → visual process defined by the Salesperson
  Agent         → executes the Workflow on behalf of the Salesperson
  Judge         → verifies that the DSL is consistent with the spec
```

**Rules:**
- One responsibility per actor (the → clause)
- No technical attributes (no "API", "database", "model")
- If an actor needs a sub-role, define it separately
- Each Actor corresponds to exactly one BPMN Pool

---

## BEHAVIOR

What must happen, expressed as observable events.
Each BEHAVIOR maps to a BPMN flow: Start Event → Tasks → Gateways → End Event.

```
BEHAVIOR <n>
  GIVEN   <state of the world before>     ← BPMN Start Event
  WHEN    <something happens>             ← BPMN Message or Timer
  THEN    <observable outcome>            ← BPMN Task or End Event
  AND     <additional outcome>            ← BPMN subsequent Task (optional, repeatable)
```

### Example: Workflow Definition

```
BEHAVIOR define_workflow
  GIVEN   a Salesperson wants to automate a business process
  WHEN    the Salesperson describes the process in natural language
  THEN    the system produces a BPMN representation
  AND     the BPMN is compiled to DSL
  AND     the Judge verifies the DSL is consistent with the spec
```

### Example: Intent Detection

```
BEHAVIOR detect_intent
  GIVEN   a Contact has at least one recorded interaction
  WHEN    the Agent evaluates the interaction history
  THEN    the Contact receives an intent signal
  AND     the signal is visible to the Salesperson
  AND     the reason for the signal is shown alongside it
```

### Example: Agent-to-Agent Dispatch

```
BEHAVIOR dispatch_workflow
  GIVEN   a workflow is ready for execution
  WHEN    the Runtime determines execution requires an external agent
  THEN    the DSL is dispatched via MCP or A2A
  AND     the receiving agent responds with ACCEPTED, REJECTED, or DELEGATED
  AND     a REJECTED response includes the divergence reason
```

### Example: Human Override

```
BEHAVIOR override_agent_action
  GIVEN   an Agent has proposed an action based on a Workflow
  WHEN    the Salesperson disagrees with the proposed action
  THEN    the Salesperson can reject or modify the action
  AND     the Agent records the override as feedback
  AND     the override improves the Agent's future proposals
```

---

## DSL BLOCK

This is not pseudocode. This is the program.
The Runtime reads this directly and generates CRM operations.
Technology-agnostic. Diffable. Versionable in git.
Generated code in any other language is a disposable artifact of the Runtime.

```
WORKFLOW detect_and_surface_intent
  ON      contact.interaction_recorded
  IF      contact.interactions.count >= 1
    AGENT evaluate_intent(contact.interaction_history)
    SET   contact.intent_signal = agent.result
    IF    contact.intent_signal == HIGH
      SURFACE contact TO salesperson.view WITH reason
      WAIT  48 hours
      IF    salesperson.has_not_acted
        NOTIFY salesperson WITH contact + reason
```

**The DSL is the source of truth. Not the generated code.**
If they diverge, the DSL wins.

**Language rules:**

| Allowed | Not Allowed |
|---|---|
| ON, IF, SET, WAIT, NOTIFY, SURFACE | HTTP, POST, SELECT, INSERT |
| contact.intent_signal | contact.crm_field_47 |
| AGENT evaluate_intent(...) | llm.complete(prompt=...) |
| WAIT 48 hours | setTimeout(172800000) |
| salesperson.has_not_acted | user.last_click == null |
| DISPATCH TO agent WITH workflow | POST /api/agent |

---

## CONSTRAINTS

What must never happen, regardless of circumstances.

```
CONSTRAINTS
  A Contact must never be acted upon without an active Workflow defined by a Salesperson
  An Agent must never execute a Workflow that has not passed Judge verification
  A Workflow must never be deleted — only deactivated with a recorded reason
  The Agent must never infer intent without prior interaction data
  An override by a Salesperson must never be silently discarded
  A REJECTED response must never be silent — it must always include a reason
```

---

## VERIFICATION BLOCK

Appended after the spec. Used by the Judge or a human reviewer.

```
VERIFY
  BEHAVIOR define_workflow covers: BPMN translation, DSL compilation, judge verification
  BEHAVIOR detect_intent covers: signal generation, visibility, reasoning
  BEHAVIOR dispatch_workflow covers: MCP/A2A dispatch, response handling
  BEHAVIOR override_agent_action covers: rejection, recording, feedback loop
  CONSTRAINTS cover: workflow gate, judge gate, data integrity, override integrity
  OPEN QUESTIONS: none
```

### Judge Prompt

```
Given this SPEC and this VERIFY block:
1. Is every THEN clause observable without knowing the implementation?
2. Does any BEHAVIOR contradict a CONSTRAINT?
3. Are there ACTORS in BEHAVIOR not defined in ACTORS?
4. Are there GIVEN states never produced by another BEHAVIOR?
5. Does the DSL BLOCK match what the BEHAVIOR blocks describe?
6. Does the DSL BLOCK use only BPMN-grounded concepts?
7. Are all protocol responses (ACCEPTED/REJECTED/DELEGATED) covered?
8. List any term that could be interpreted in more than one way.
```

---

## What This Actually Is

Not a CRM spec. Not pseudocode documentation. Not a workflow tool.

This is an **open protocol** where:
- Any agent or human can be input
- BPMN resolves business concept ambiguity
- The DSL is the executable source of truth and the unit of communication
- The Judge ensures consistency before execution
- The Runtime generates CRM operations as a side effect
- Any agent or system can be output via MCP or A2A
- MMD is an optional visualization, never a gate

The CRM state emerges from execution.
Nobody manually maintains it.

> "Describe the problem clearly, and half of it is already solved."
