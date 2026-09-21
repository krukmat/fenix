# W4-A — Cross-Platform Governance Decision Contract

> Status: W4-T1 through W4-T4 implemented  
> Scope: governance decision only; runtime adapter binding remains deferred.  
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

## Closure

W4-A is complete when any capability can be assigned a valid governance profile and Fenix can deterministically resolve:

- provider execution permission;
- approval requirement;
- evidence requirement;
- actual evidence participation;
- policy reference.

Runtime wiring of this decision into the existing W1 execution pipeline is **W4-B** and remains intentionally outside this block.
