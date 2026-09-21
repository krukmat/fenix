# W4-A — Cross-Platform Governance Decision Contract

> Status: W4-A and W4-B implemented  
> Scope: governance decision + runtime binding; external transport adapters remain deferred.  
> Authority: Fenix.

## Goal

W4-A turns the W1 capability boundary, W2 Mermaid2SF semantics, and W3 VEL evidence contract into one Fenix-owned governance decision.

The decision answers, before provider execution:

```text
Can the capability execute?
Does it require approval?
Is immutable evidence required / optional / absent?
Will VEL participate in this execution?
Which policy decision authorized the result?
```

It does not select HTTP, CLI, MCP, library embedding, sidecar deployment, or any other transport.

## W4-T1 — governance classification

Capability operational risk remains the W1 side-effect classification:

```text
READ
TRANSFORM
VERIFY
MUTATE
IRREVERSIBLE
```

W4 adds an independent evidence dimension:

```text
NONE
OPTIONAL
REQUIRED
```

Approval and evidence are intentionally separate axes:

```text
GovernanceProfile
├── CapabilityDescriptor
│   ├── side_effect_class
│   └── approval_required
├── evidence_requirement
└── evidence_reason
```

An irreversible operation still requires approval through W1 semantics. That does not imply that every irreversible capability must use the same evidence policy; evidence remains an explicit Fenix governance decision.

## W4-T2 — evidence policy

Evidence participation is resolved before provider invocation:

```text
NONE
  → VEL is not planned

OPTIONAL
  → Fenix policy/context decides whether evidence is planned

REQUIRED
  → VEL evidence is planned
```

A non-NONE profile must state an evidence reason.

VEL never decides whether it should participate. It receives evidence only after Fenix has already planned evidence for the execution.

### Current Mermaid2SF profiles

```text
salesforce.flow.validate  → VERIFY     + evidence NONE
salesforce.flow.compare   → VERIFY     + evidence NONE
salesforce.flow.import    → TRANSFORM  + evidence OPTIONAL
salesforce.flow.export    → TRANSFORM  + evidence OPTIONAL
```

Therefore Mermaid2SF and VEL are not a fixed chain.

## W4-T3 — GovernanceDecision

The resolver consumes Fenix-owned facts:

```text
Governance Input
├── governance profile
├── policy_allowed
├── policy_reference
├── approval_state
└── optional_evidence selection
          ↓
GovernanceDecision
├── allowed
├── denial_reason?
├── approval_required
├── evidence_requirement
├── evidence_planned
├── evidence_reason?
├── side_effect_class
├── capability_name
├── capability_version
└── policy_reference
```

Two fields drive component participation independently:

```text
allowed = true
    → provider capability may execute

evidence_planned = true
    → VEL evidence lifecycle participates
```

Examples:

```text
validate Flow
allowed=true
evidence_planned=false
    → Fenix → M2SF → Fenix

export Flow without evidence
allowed=true
evidence_planned=false
    → Fenix → M2SF → Fenix

export Flow with optional evidence selected
allowed=true
evidence_planned=true
    → Fenix → M2SF → Fenix → VEL

policy-denied operation whose profile requires evidence
allowed=false
evidence_planned=true
    → provider is NOT invoked
    → Fenix may evidence the governance denial in VEL
```

The last case demonstrates why provider execution and evidence participation must not be represented by one boolean.

## W4-T4 — cross-component invariants

1. **Fenix is governance authority.** Provider output cannot override the Fenix decision.
2. **M2SF owns Flow semantics, not governance.** Fidelity/diagnostics can influence later agent behavior but do not decide approval/evidence policy.
3. **VEL owns cryptographic evidence, not authorization.** VEL cannot decide whether the capability should execute.
4. **Evidence planning happens before provider execution.**
5. **Approval and evidence are independent dimensions.**
6. **A capability with evidence NONE cannot request VEL opportunistically.**
7. **OPTIONAL evidence is selected only by Fenix-owned policy/context.**
8. **REQUIRED evidence is always planned, including when the governed result itself is a denial worth evidencing.**
9. **VEL failure never causes the business capability to execute again.** W3 reconciliation applies.
10. **execution_id remains the join identity across governance, capability execution, audit, and evidence.**

## Current component model

```text
                         FENIX
                           │
                    GovernanceDecision
                           │
             ┌─────────────┴─────────────┐
             │                           │
        allowed=true              evidence_planned=true
             │                           │
             ▼                           ▼
      capability provider               VEL
       e.g. Mermaid2SF           cryptographic evidence
             │                           │
             ▼                           ▼
 fidelity / diagnostics            ProofReference
             │                           │
             └─────────────┬─────────────┘
                           ▼
                         FENIX
```

The two branches are independent. Either branch can exist without the other when the governance decision requires it.

## W4-B — runtime integration

The W4 decision is now bound into the W1 `ToolRegistry` execution path through two transport-neutral runtime ports:

```text
CapabilityGovernancePlanner
CapabilityEvidenceRecorder
```

The runtime order is:

```text
tool prechecks
    ↓
execution context
    ↓
W1 governor / approval facts
    ↓
GovernanceDecision            ← once per execution_id
    ↓
allowed?
├── no  → optional/required evidence of denial → audit → stop
└── yes → provider execution / retry
              ↓
          terminal outcome
              ↓
       evidence_planned?
       ├── no  → unified audit
       └── yes → EvidenceEnvelope
                     ↓
                 evidence sink
                     ↓
               ProofReference
                     ↓
                unified audit
```

Provider retries do **not** recompute governance. The same decision and `execution_id` remain stable across retries.

### Evidence failure behavior

Provider/business execution and evidence recording remain independent:

```text
provider succeeded
VEL append indeterminate
        ↓
provider is NOT executed again
        ↓
evidence reconciliation uses execution_id
```

An indeterminate evidence append may be reconciled through the transport-neutral sink lookup:

```text
RecordEvidence(...)
   ↓ indeterminate
LookupEvidence(stream_id, execution_id)
   ├── found     → recover ProofReference
   └── not found → evidence remains indeterminate
```

A deterministic evidence error is not converted into a replay lookup.

`verification_failed` is surfaced separately from `evidence_indeterminate`; provider output is preserved in both cases so callers can distinguish business outcome from evidence outcome.

### Unified audit

The terminal audit event contains:

```text
execution_id
capability status
attempt count
governance
├── allowed
├── denial_reason
├── approval_required
├── evidence_requirement
├── evidence_planned
├── evidence_reason
└── policy_reference
evidence
├── lifecycle state
└── compact ProofReference projection when available
```

Raw capability payloads, signed bundles, and raw signatures are not copied into operational audit.

### Runtime wiring

The Fenix router installs `governance.RuntimePlanner` as the capability governance planner.

Current Mermaid2SF governance profiles are registered during router construction. No VEL transport is wired in the router yet because selecting a concrete sink would choose deployment/transport topology, which remains deferred.

### W4-B tests

Runtime tests cover:

- provider execution without evidence;
- provider + planned evidence;
- policy/governance denial with evidence;
- missing approval/governor cannot be overridden by a planner;
- provider success + evidence indeterminate without provider replay;
- explicit verification failure;
- governance resolution exactly once across provider retries;
- EvidenceEnvelope payload minimization;
- indeterminate append reconciliation by `execution_id`;
- deterministic evidence failure without reconciliation lookup.

## Closure

W4 contract/runtime integration is complete.

Still deferred:

- concrete M2SF runtime adapter;
- concrete VEL transport/sink;
- deployment topology;
- stream/checkpoint operational policy.

Those belong to later readiness/execution waves and must preserve the W4 governance runtime contract.
