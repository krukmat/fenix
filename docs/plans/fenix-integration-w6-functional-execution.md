---
doc_type: plan
id: W6
title: Fenix Integration W6 Functional Execution
status: in_progress
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
PlannerExecutor
        |
        v
ToolRegistry
        |
        v
Fenix governance
        |
        +--> M2SF when Salesforce-flow semantics are required
        |       |
        |       v
        |   semantic result
        |
        +--> other governed capabilities
        |
        v
agent-safe business result

Separately, when evidence_planned=true:

Fenix governance / capability outcome
        |
        v
Evidence Runtime / durable outbox
        |
        v
VEL
        |
        v
compact proof / verification state
```

The Blackboard coordinates work; it does not become an external authority. ToolRegistry executes
business capabilities. VEL is not a ToolRegistry capability and is not directly callable by agents;
it is reached only through Fenix's evidence runtime. Agents never write raw reasoning to VEL.
Fenix remains the governance/execution authority.

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

## W6-A implementation status

**IMPLEMENTED / proof-ready — 2026-09-22**

The existing Blackboard `PlannerExecutor` already provides the required execution seam into
`ToolRegistry`; W6-A therefore required a functional composition proof rather than new runtime
plumbing. The proof lives at `internal/api/integration_runtime_w6_test.go` and covers evidence
OFF/ON variants with M2SF correlation, Blackboard projection and audit/evidence correlation.

## W6-B — Multi-agent coordination

After W6-A proves one vertical slice, exercise multiple agents sharing Blackboard state while one or
more governed capabilities execute. Validate ownership, handoff, stale-state handling and that each
external execution receives its own stable execution identity.

## W6-B implementation status

**CLOSED — 2026-09-22**

W6-B reuses the existing specialized-agent artifact model and adds one provider-neutral planner
extension: `PlanningConfig.ActionSteps`. When supplied, those concrete steps become the executable
selected-proposal sequence; when omitted, existing generic planner behavior is unchanged.

The W6-B proof demonstrates:

```text
Signal/Evidence agent artifacts
          +
ranked collaborative hypothesis
          ↓
Blackboard Planner
          ↓
bound governed action sequence
  1. salesforce.flow.export
  2. salesforce.flow.validate
          ↓
PlannerExecutor
          ↓
ToolRegistry governance
          ↓
M2SF
```

Identity behavior is explicit:

```text
export attempt 1  ─┐
export retry      ─┴─ same execution_id E1
validate             distinct execution_id E2
```

With optional evidence selected for export, Fenix emits only the minimized evidence envelope with
input/output digests; Blackboard-only collaboration content is not copied into it. A collaboration
missing required evidence remains `awaiting_evidence` and invokes no provider.

Executable proof:
`internal/api/integration_runtime_w6_test.go`.

## W6-C — Functional resilience

Exercise functional behavior when M2SF or VEL is unavailable:

- M2SF unavailable: semantic capability fails explicitly; agent does not invent a result.
- VEL unavailable with optional evidence: business result is preserved and evidence is indeterminate.
- verification failed: agent may report evidence failure but may not claim verified integrity.
- restart/reconciliation: only evidence lifecycle resumes; business action is not repeated.

## W6-C implementation status

**CLOSED — C1-C7 implemented, 2026-09-22**

The first resilience block is preserved in `internal/api/integration_runtime_w6_test.go`.

- **C1 — M2SF unavailable:** persistent 503 responses consume the bounded two-attempt retry
  contract, retain one stable `execution_id`, return no provider result, and project
  `tool_failure` instead of inventing success.
- **C2 — Retryable M2SF failure:** already proven by the W6-B functional proof: a transient 503
  retries once with the same `execution_id` and then succeeds. This behavior is reused rather
  than duplicated in another test.
- **C3 — VEL unavailable / optional evidence:** M2SF succeeds once; the business result remains
  present while the evidence path reports `evidence_indeterminate`. No business retry occurs.
- **C4 — verification failure:** M2SF succeeds once; the provider result remains present while
  Fenix exposes `evidence_verification_failed` and never presents the execution as verified.

- **C5 — restart/reconciliation:** a real durable outbox recorder persists an indeterminate append,
  then a fresh recorder instance reconciles the same SQLite row by execution identity, advances
  through checkpoint/verify, and reaches `verified`. M2SF remains at one business call.
- **C6 — agent-safe projection:** persisted Blackboard failure state retains provider output plus
  explicit evidence error status while excluding verification bundles, event hashes, Merkle roots,
  signature references and reasoning traces.
- **C7 — consolidated proof:** C1-C6 now form the resilience portion of the W6 demo/handoff.

No production runtime change was required for W6-C; the W1-W5 contracts already implement these
semantics. W6-C adds the functional vertical proofs.

No full GitHub QA run was started, per the repository-owner QA decision. The proof is therefore
recorded as implementation-complete rather than represented as a globally green QA run.

## W6-D — Product/demo handoff

Produce a concise end-to-end tour showing:

- agent interaction;
- Blackboard coordination;
- M2SF semantic work;
- optional VEL evidence;
- audit/trace;
- failure behavior.

This becomes the functional proof and entry point for whatever productization wave follows W6.

## W6-D implementation status

**IN PROGRESS — D1, D2 and D4 implemented, 2026-09-22**

### D1 — Happy-path tour

The canonical path is now documented in the repository README:

```text
agents
  -> Blackboard
  -> collaborative proposal
  -> PlannerExecutor
  -> ToolRegistry
  -> Fenix governance
  -> M2SF capability
  -> normalized business result
  -> Blackboard / agent-safe outcome
```

Evidence is a separate post-governance branch and does not make VEL a tool.

### D2 — Optionality matrix

| M2SF | VEL | Meaning |
|---|---|---|
| yes | no | Salesforce semantic capability without cryptographic evidence |
| yes | yes | Salesforce semantic capability plus evidence lifecycle |
| no | yes | another governed capability may produce VEL evidence |
| no | no | ordinary governed Fenix execution |

This preserves the W4/W5 invariant that provider execution and evidence participation are
independently governed.

### D4 — Authority / boundary model

```text
                FENIX
       orchestration + governance
          /                 \
         /                   \
Blackboard                ToolRegistry
coordination              capabilities
                              |
                              v
                         Mermaid2SF
                     semantic authority

governed outcome
      |
      v
Evidence Runtime / Outbox
      |
      v
     VEL
crypto evidence authority
```

VEL is explicitly outside ToolRegistry. Agents cannot invoke VEL directly.

Remaining W6-D work: D3 resilience tour, D5 executable proof index and D6 final README/operator
handoff + W6 closure.

## Exit criteria

W6 closes when at least one complete agent-driven vertical slice is executable and demonstrated,
the M2SF/VEL optionality invariant remains intact, Blackboard participation is explicit, and failure
behavior does not bypass Fenix governance or replay business actions.
