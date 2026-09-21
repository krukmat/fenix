# W5-B — Adapter & Provider Integration

> Date: 2026-09-21  
> Implementation status: COMPLETE  
> Provider validation: Mermaid2SF GREEN; VEL GREEN  
> Fenix full CI: DEFERRED to W5 closure by repository CI policy  
> Branch policy: direct work on `main`

## Goal

Turn the W1-W4 semantic/governance contracts and W5-A gap inventory into real,
versioned cross-repository runtime boundaries.

W5-B does not yet solve durable asynchronous reconciliation, checkpoint scheduling,
or final production topology. Those remain W5 operational-readiness work.

## W5-B1 — Transport decision

Selected transport for both providers:

```text
Fenix (Go)
   ├── HTTP/JSON v1 ──► Mermaid2SF (Node/TypeScript)
   └── HTTP/JSON v1 ──► VEL (FastAPI/Python)
```

Reasons:

- preserves language/process isolation;
- preserves one authority per concern;
- uses provider processes that already exist;
- makes timeout/auth/versioning explicit;
- avoids importing FlowIR into Fenix;
- avoids embedding the Python cryptographic runtime into Fenix.

The integration is opt-in. Fenix still starts without either provider configured.

### Runtime configuration

Fenix:

```text
FENIX_M2SF_URL
FENIX_M2SF_TOKEN
FENIX_VEL_URL
FENIX_VEL_TOKEN
FENIX_INTEGRATION_TIMEOUT_MS       # default 15000
FENIX_M2SF_EVIDENCE_ENABLED       # default false
```

Providers:

```text
Mermaid2SF: M2SF_FENIX_TOKEN
VEL:        VEL_FENIX_TOKEN
```

URL and token must be configured together. Partial provider configuration fails
closed during router construction.

No token value is stored in the repositories.

---

## W5-B2 — Mermaid2SF provider surface

Mermaid2SF now exposes a dedicated authenticated Fenix boundary:

```text
GET  /api/fenix/ready
POST /api/fenix/flow
```

Authentication:

```text
Authorization: Bearer <M2SF_FENIX_TOKEN>
```

The provider contract supports exactly the four Fenix W2 capabilities:

```text
salesforce.flow.import
salesforce.flow.export
salesforce.flow.validate
salesforce.flow.compare
```

Provider implementation:

```text
web/server/fenix-contract.js
       ↓
existing M2SF semantic core
       ├── MermaidParser
       ├── FlowIR v2 builder/parser
       ├── FlowValidator
       ├── Salesforce XML generator/parser
       ├── Mermaid generator
       └── semantic snapshot
       ↓
Fenix Result v1
```

The response is normalized into:

- status;
- artifacts;
- fidelity;
- semantic metadata;
- typed diagnostics;
- semantic diff for compare;
- provider reference.

FlowIR remains owned by M2SF.

The legacy `/api/compile` and `/api/decompile` surfaces remain available.

Operational guards:

- 2 MiB request-body limit on the Fenix surface;
- fail-closed auth;
- stable provider request/result contract;
- Fenix trace/execution headers accepted by the HTTP boundary.

Provider validation:

```text
Mermaid2SF CI
├── compiler-core                   PASS
├── Fenix integration contract smoke PASS
├── Salesforce org gate             PASS
└── legacy style debt                non-blocking existing backlog
workflow conclusion                  PASS
```

---

## W5-B3 — VEL provider surface

VEL now exposes the existing W3 external-authority domain path over authenticated HTTP:

```text
POST /v1/external/events
     ↓
ExternalEventCreate
     ↓
Ledger.append_external(...)
     ↓
NO LocalPolicyEngine

GET /v1/external/events/by-idempotency
    ?stream_id=<stream>
    &idempotency_key=<execution_id>
     ↓
Ledger.get_by_idempotency(...)
```

Authentication:

```text
Authorization: Bearer <VEL_FENIX_TOKEN>
```

Security behavior:

- missing VEL integration token → 503;
- missing/invalid bearer → 401;
- idempotency conflict → 409;
- invalid external evidence → 400;
- standalone `POST /v1/events` remains unchanged and still uses local policy.

The lookup uses query parameters instead of a stream path segment because the current
Fenix default stream shape is:

```text
workspace/<workspace_id>
```

VEL provider metrics now distinguish external append and lookup operations.

Provider validation: current VEL CI is GREEN, including lint, functional/reliability
tests and the existing local/upgrade smoke paths.

---

## W5-B4 — Fenix adapters

### Mermaid2SF ToolExecutor

Implemented in:

`internal/infra/integration/m2sf/executor.go`

Responsibilities:

```text
flowinterop.Request
      ↓ validate locally
HTTP POST /api/fenix/flow
      ↓
M2SF provider
      ↓
flowinterop.Result
      ↓ validate provider contract
ToolRegistry runtime
```

It propagates:

- Bearer authentication;
- `X-Fenix-Trace-ID`;
- `X-Fenix-Execution-ID`.

Transport classification:

- network / response-read / 5xx → retryable provider failure;
- deterministic 4xx → non-retryable provider failure;
- semantic `rejected|unsupported` → valid provider result, not transport failure.

Response size is bounded to 4 MiB.

### VEL RuntimeSink

Implemented in:

`internal/infra/integration/vel/sink.go`

It implements:

```text
evidence.RuntimeSink
├── RecordEvidence(...)
└── LookupEvidence(...)
```

The adapter maps `EvidenceEnvelope` to minimized `ExternalEventCreate` evidence.

Invariant:

```text
VEL idempotency_key = Fenix execution_id
```

An uncertain append is surfaced as `IndeterminateRecordError`, allowing the existing
Fenix recorder to lookup by execution identity instead of replaying the business action.

A stored VEL event becomes a compact Fenix `ProofReference` with status `recorded`.
Checkpoint/verified progression remains later operational work.

### Authorization hash compatibility

W5-B identified and fixed a real cross-language incompatibility.

Both sides now use the same contract:

```text
SHA-256(
  "VEL:authorization-context:v1\0"
  + RFC8785-equivalent canonical authorization context
)
```

For the current flat string-only Fenix authorization context, Go emits lexicographically
ordered JSON with HTML escaping disabled.

Shared contract vector:

```text
context:
{
  "workspace_id": "ws-1",
  "trace_id": "trace-1",
  "execution_id": "exec-1",
  "policy_reference": "fenix:w1-policy-gate"
}

hash:
45f5c2054741a39cfff345c3a5054553d5c80dbb663445718d4ff498024ea85e
```

The same vector is pinned by tests in Fenix and VEL.

---

## W5-B5 — Fenix runtime wiring

The router now configures the adapters through the existing governed ToolRegistry.

```text
Router
  ↓
cross-platform settings
  ↓
Governance RuntimePlanner
  ↓
ToolRegistry
  ├── M2SF ToolExecutors
  │     ├── import
  │     ├── export
  │     ├── validate
  │     └── compare
  │
  └── Evidence RuntimeRecorder
          ↓
        VEL Sink
```

External capabilities receive persisted tool definitions instead of bypassing normal
Fenix validation/permission surfaces.

### Optional evidence remains Fenix-owned

M2SF and VEL are **not automatically coupled**.

Default:

```text
FENIX_M2SF_EVIDENCE_ENABLED=false

validate / compare → evidence NONE
import / export    → evidence OPTIONAL but not selected
```

When Fenix is explicitly configured with:

```text
FENIX_M2SF_EVIDENCE_ENABLED=true
```

and VEL is also configured:

```text
import / export
   ↓
Fenix GovernanceDecision
   ↓ evidence_planned=true
M2SF execution
   ↓
VEL EvidenceEnvelope
```

The flag cannot be enabled without a configured VEL provider.

This preserves the W4 rule that the provider never decides whether evidence participates.

---

## W5-B6 — Cross-repository integration tests

Implemented test layers:

### Fenix

- M2SF adapter:
  - bearer auth;
  - trace/execution propagation;
  - response contract validation;
  - 5xx retry classification.
- VEL adapter:
  - execution id → idempotency key;
  - minimized authorization context;
  - stream IDs containing `/`;
  - 404 lookup;
  - uncertain append classification.
- runtime configuration:
  - providers disabled by default;
  - partial configuration fails closed;
  - four M2SF capabilities are registered;
  - optional evidence requires VEL;
  - evidence `NONE` cannot be selected.
- shared Fenix/VEL authorization hash vector.

### Mermaid2SF

CI executes the provider contract smoke after the TypeScript build:

```text
validate → succeeded
compare identical FlowIR → equal=true
invalid export request → rejected provider diagnostic
```

### VEL

Tests cover:

- integration disabled without runtime token;
- missing/invalid bearer;
- valid bearer;
- external API route exposure;
- shared Fenix/VEL context-hash vector.

### Validation policy note

Fenix automatic CI remains disabled by explicit W5 repository policy.

Therefore:

```text
provider CI                GREEN
Fenix adapter tests         IMPLEMENTED
Fenix full CI execution     DEFERRED TO FINAL W5 CHECKPOINT
```

No claim is made that the final Fenix pipeline has run for W5-B yet.

---

## W5-B result

The W5-A blockers resolved by this block are:

| Gap | Result |
|---|---|
| W5-G01 concrete M2SF executor | RESOLVED |
| W5-G02 stable four-operation M2SF provider surface | RESOLVED |
| W5-G03 concrete VEL RuntimeSink | RESOLVED |
| W5-G04 external-authority VEL HTTP append | RESOLVED |
| W5-G05 VEL HTTP idempotency lookup | RESOLVED |
| W5-G06 S2S authentication | RESOLVED for PoC runtime contract |
| provider correlation transport | RESOLVED for trace/execution propagation |
| Fenix-controlled optional M2SF evidence | RESOLVED |

Still open after W5-B:

```text
W5-G07 durable evidence reconciliation/outbox
W5-G08 checkpoint scheduling + verification progression
W5-G09 final stream partition strategy
W5-G10 integration observability/metrics completion
production deployment/topology hardening
production secret-management implementation
```

Those are operational-readiness concerns, not provider-contract gaps.

## Status

W5-B implementation is COMPLETE.

Provider-side validation is GREEN.

Final Fenix CI validation remains intentionally deferred until all W5 work is finished,
in accordance with the repository wave policy.
