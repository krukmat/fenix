# W3 — Fenix ↔ Verifiable Event Ledger contract

> Status: W3-T1 / W3-T2 / W3-T3 implemented  
> Transport: intentionally undefined  
> VEL evidence baseline: `verifiable-event-ledger@0e942480` plus the external-authority seam introduced by this wave.

## W3-T1 — EvidenceEnvelope v1

Fenix owns the evidence envelope. It is constructed from an already governed Fenix execution and does not import VEL internal models.

```text
EvidenceEnvelope v1
├── schema_version = "1"
├── stream_id
├── workspace_id
├── trace_id
├── execution_id
├── run_id?
├── actor
│   ├── id
│   └── type
├── capability
│   ├── name
│   ├── version
│   ├── operation
│   └── side_effect_class
├── authorization
│   ├── policy_id
│   ├── policy_version
│   ├── decision
│   ├── reason
│   └── context_hash
├── approval?
│   ├── approval_id
│   └── decision
├── outcome
│   ├── status
│   └── error_code?
├── input_digest?
├── output_digest?
└── occurred_at
```

No full tool/provider payload is part of the envelope. Input/output evidence uses SHA-256 references when needed.

The policy context itself remains in Fenix; only its digest crosses the evidence boundary.

## W3-T2 — authority boundary

The authority model is explicit:

```text
Fenix = policy + approval authority
VEL   = cryptographic evidence + verification authority
```

VEL's existing standalone path remains:

```text
/v1/events
  → LocalPolicyEngine
  → signed authorization evidence
```

For the Fenix integration, VEL now exposes a domain seam:

```text
ExternalEventCreate
  → Ledger.append_external(...)
  → validate context hash
  → sign / chain / checkpoint
```

This path takes no local `PolicyEngine` and therefore cannot independently override or replay Fenix's policy decision.

VEL may reject structurally inconsistent or tampered evidence. That is evidence validation, not policy evaluation.

### Minimal VEL authorization context

The adapter should map the Fenix envelope to a minimized VEL context similar to:

```text
workspace_id
trace_id
execution_id
approval_id?
policy_context_hash
```

This allows VEL to verify the signed context hash without receiving the sensitive policy payload.

## W3-T3 — ProofReference v1

Fenix stores a compact locator, not the full VEL verification bundle.

```text
ProofReference v1
├── schema_version = "1"
├── provider
├── execution_id
├── stream_id
├── event_id
├── event_hash
├── key_id
├── signature_ref
├── sequence
├── checkpoint?
│   ├── checkpoint_id
│   ├── checkpoint_hash
│   ├── merkle_root
│   └── tree_size
├── verification_status
└── verification_issues[]
```

Verification status values:

- `recorded`
- `pending_checkpoint`
- `verified`
- `verification_failed`

The required `execution_id` is the join key back to Fenix operational audit.

### Mapping from current VEL material

```text
StoredEvent
  event_id      → event_id
  event_hash    → event_hash
  key_id        → key_id
  stream_id     → stream_id
  sequence      → sequence
  event record  → signature_ref

StoredCheckpoint
  checkpoint_id   → checkpoint.checkpoint_id
  checkpoint_hash → checkpoint.checkpoint_hash
  merkle_root     → checkpoint.merkle_root
  tree_size       → checkpoint.tree_size

VerificationResult
  valid=true  → verified
  valid=false → verification_failed
  issues[]    → verification_issues[]
```

Before a checkpoint exists, a recorded event may legitimately have `pending_checkpoint` status.

## Ownership

```text
Fenix operational audit
    │
    ├── trace_id
    ├── execution_id
    └── ProofReference
             │
             ▼
            VEL
       signed event
       hash chain
       checkpoint
       verification bundle
```

Fenix does not replicate the full signed event, Merkle bundle, or key registry into its operational audit.

## Deferred

W3-T4 through W3-T6 remain open:

- idempotency / replay contract;
- evidence failure and reconciliation semantics;
- agent/audit exposure.

Transport and deployment topology remain deferred.
