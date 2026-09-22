---
doc_type: plan
id: W6
title: Fenix Integration W6 Functional Execution
status: planned
phase: integration
tags: [fenix, integration, agents, blackboard, mermaid2sf, vel]
created: 2026-09-22
---

# W6 — Functional Execution

## Goal

Move from "the integration infrastructure works" to "Fenix agents use it in complete functional
cases".

W6 does not introduce another transport, policy engine, evidence engine or semantic IR. It exercises
the architecture already closed in W1-W5 through realistic agent workflows.

## Runtime model

```text
User / workflow trigger
        |
        v
Fenix agent orchestration
        |
        +--> Agentic Blackboard
        |      shared task state / artifacts / coordination
        |
        v
ToolRegistry governed capability
        |
        +--> M2SF when Salesforce-flow semantics are required
        |
        +--> other Fenix capabilities
        |
        +--> VEL evidence lifecycle when evidence_planned=true
        |
        v
agent-safe result + audit + optional verified proof
```

The Blackboard coordinates work; it does not become an external authority. Agents never write raw
reasoning to VEL. Fenix remains the governance/execution authority.

## W6-A — First functional slice

Prove one complete agent-driven Salesforce Flow case end to end through the normal Fenix runtime.

Recommended first slice:

```text
agent receives Flow task
   -> Blackboard records shared task/artifact state
   -> Fenix resolves governance once
   -> agent calls salesforce.flow.validate or compare
   -> M2SF executes through FlowIR v2
   -> normalized fidelity/diagnostics return to Fenix
   -> optional evidence policy decides whether VEL participates
   -> agent consumes only agent-safe result
   -> audit/correlation remains traceable by execution_id
```

### W6-A tasks

- **A1** Define the functional scenario and expected agent-visible outcome.
- **A2** Bind the scenario to the existing agent/Blackboard execution path.
- **A3** Invoke the existing M2SF capability through ToolRegistry; no direct provider call.
- **A4** Exercise evidence OFF and evidence ON variants without coupling M2SF to VEL.
- **A5** Assert Blackboard state contains task/result references but no raw VEL bundle or hidden reasoning.
- **A6** Assert audit/trace/execution correlation across agent, capability and optional proof.
- **A7** Capture one executable/demo fixture as the W6 functional proof.

## W6-B — Multi-agent coordination

After W6-A proves one vertical slice, exercise multiple agents sharing Blackboard state while one or
more governed capabilities execute. Validate ownership, handoff, stale-state handling and that each
external execution receives its own stable execution identity.

## W6-C — Functional resilience

Exercise functional behavior when M2SF or VEL is unavailable:

- M2SF unavailable: semantic capability fails explicitly; agent does not invent a result.
- VEL unavailable with optional evidence: business result is preserved and evidence is indeterminate.
- verification failed: agent may report evidence failure but may not claim verified integrity.
- restart/reconciliation: only evidence lifecycle resumes; business action is not repeated.

## W6-D — Product/demo handoff

Produce a concise end-to-end tour showing:

- agent interaction;
- Blackboard coordination;
- M2SF semantic work;
- optional VEL evidence;
- audit/trace;
- failure behavior.

This becomes the functional proof and entry point for whatever productization wave follows W6.

## Exit criteria

W6 closes when at least one complete agent-driven vertical slice is executable and demonstrated,
the M2SF/VEL optionality invariant remains intact, Blackboard participation is explicit, and failure
behavior does not bypass Fenix governance or replay business actions.
