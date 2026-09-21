# W3 — Fenix ↔ Verifiable Event Ledger contract

> Status: W3-T1 through W3-T6 implemented — contract wave complete  
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

## W3-T4 — idempotency and replay

The Fenix → VEL append idempotency key is the stable `execution_id`.

```text
same stream_id + same execution_id + same evidence
    → return the original VEL event

same stream_id + same execution_id + different evidence
    → IDEMPOTENCY CONFLICT
    → do not create another event
```

VEL now validates replay consistency across actor, action/resource, payload digest, and authorization evidence.

This closes a critical ambiguity: a duplicate key is not accepted merely because the key matches.

VEL also exposes domain lookup by `(stream_id, idempotency_key)` so an uncertain caller can reconcile before re-appending.

## W3-T5 — failure and reconciliation semantics

Evidence delivery is independent from the governed business execution:

```text
pending_record
recorded
pending_checkpoint
verified
verification_failed
indeterminate
```

The safe recovery policy is:

```text
pending_record        → retry evidence append with same execution_id
recorded              → await checkpoint
pending_checkpoint    → await checkpoint
verified              → no action
verification_failed   → escalate
indeterminate         → lookup existing event by idempotency key first
```

**Invariant:** evidence reconciliation never repeats the governed business capability.

For an indeterminate append:

```text
timeout / lost response
        ↓
lookup(stream_id, execution_id)
        ├── found     → recover ProofReference
        └── not found → retry evidence append with same key
```

A VEL evidence failure therefore does not convert a successful Salesforce/agent operation into a request to execute that operation again.

## W3-T6 — agent and operational-audit exposure

Fenix operational audit stores only a compact `AuditProjection`:

```text
provider
execution_id
stream_id
event_id
event_hash
key_id
signature_ref
sequence
checkpoint_id?
checkpoint_hash?
merkle_root?
tree_size?
verification_status
issue_count
```

It does not copy:

- the signed event payload;
- the complete verification bundle;
- private/public key registry data;
- raw signature bytes;
- full verification issue messages.

Agent consumption is conservative:

```text
verified + valid verified ProofReference → USE
pending record/checkpoint                → PENDING
verification_failed                      → ABSTAIN
indeterminate                            → ABSTAIN
state/proof contradiction                → ABSTAIN
```

Agents may report that evidence is pending, but they may not describe evidence as cryptographically verified before the verified proof state exists.

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

## W3 closure

W3 contract work is complete.

The Fenix/VEL boundary now defines:

- evidence envelope;
- authority separation;
- proof reference;
- replay/idempotency;
- reconciliation semantics;
- compact audit projection;
- agent evidence-consumption rules.

Still deliberately deferred:

- transport selection;
- deployment topology;
- stream-partition strategy;
- checkpoint scheduling policy;
- asynchronous delivery implementation.

Those are integration-execution concerns and do not change the W3 semantic contract.
