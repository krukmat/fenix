# Fenix Integration Baseline — W0 + W1

> Date: 2026-09-21  
> Status: active prerequisite baseline  
> Branch policy: work directly on `main` unless explicitly requested otherwise.

## Wave status

| Wave / Task | Status |
|---|---|
| W0-T1 Fenix ↔ Mermaid2SF scope | CLOSED |
| W0-T2 Fenix ↔ VEL scope | CLOSED |
| W0-T3 Non-goals | CLOSED |
| W0-T4 Cross-platform invariants | CLOSED |
| W0-T5 Ownership matrix | CLOSED |
| W1-T1 IntegrationExecutionContext v1 | CLOSED |
| W1-T2 Correlation contract | CLOSED |
| W1-T3 Capability boundary | CLOSED |
| W1-T4 Governance path | CLOSED |
| W1-T5 Audit / trace propagation | CLOSED |
| W1-T6 Failure / retry / idempotency | CLOSED |
| W2-T1 Mermaid2SF capability catalog | CLOSED |
| W2-T2 Mermaid2SF request/result contract | CLOSED |
| W2-T3 Mermaid2SF fidelity contract | CLOSED |
| W2-T4 Semantic diff contract | CLOSED |
| W2-T5 Diagnostics contract | CLOSED |
| W2-T6 Agent-safe surface | CLOSED |

W1 and the W2 Mermaid2SF semantic contract are closed. W2 remains transport-neutral: no CLI, HTTP, MCP, sidecar, or library adapter has been selected. W3 remains independent and may proceed without changing W2 contract decisions.

## W0 — ownership and scope

Fenix owns orchestration, workspace/actor identity, policy, approvals, operational audit, and governed capability execution.

Mermaid2SF owns Salesforce Flow semantics, FlowIR, semantic validation, fidelity classification, and Mermaid ⇄ Salesforce XML transformation.

VEL owns cryptographic event canonicalization, signatures, key lifecycle, hash-chain / Merkle evidence, checkpoints, and independent verification.

Salesforce remains authoritative for target-org acceptance and metadata existence.

### Non-goals

- no Salesforce deployment/mutation in the prerequisite phase;
- no CLI vs HTTP vs MCP vs sidecar choice yet;
- no FlowIR merged into Fenix Carta/Workflow IR;
- no VEL replacement of Fenix operational audit;
- no VEL policy authority for Fenix executions;
- no readiness ranking before W5.

### Invariants

1. External capabilities use one governed Fenix execution path.
2. One logical execution keeps a stable execution identity across retries.
3. Traceability is end-to-end.
4. One authority exists per concern.
5. Retry semantics are idempotent.
6. Integration contracts are versioned.
7. Fidelity/capability support is explicit.
8. External failures remain observable.
9. Sensitive payloads are minimized.
10. Internal IR ownership remains local to each domain.

## W1-T1 — IntegrationExecutionContext v1

```text
IntegrationExecutionContext v1
├── schema_version       required = "1"
├── workspace_id         required
├── trace_id             required
├── execution_id         required
├── run_id               optional
├── workflow_id          optional
├── actor
│   ├── id               required
│   └── type             required
├── approval_id          optional
├── capability
│   ├── name             required
│   └── version          required
└── started_at           required UTC timestamp
```

Rules:

- Fenix creates `trace_id` and `execution_id`.
- Retry of the same logical operation retains `execution_id`.
- A distinct logical operation gets a new `execution_id`.
- Provider-native IDs never replace Fenix IDs.
- Full provider payloads do not belong in this context.

## W1-T2 — correlation

```text
trace_id
   ├── run_id?
   ├── workflow_id?
   └── execution_id
         ├── provider request/result
         ├── operational audit
         └── eventual proof reference
```

## W1-T3 — capability boundary

Do not introduce a second external-capability runtime.

The existing `ToolRegistry` is the integration seam because it already provides:

- persisted definitions;
- schema validation;
- active/inactive lifecycle;
- permissions;
- executor registration;
- usage;
- audit hooks.

External capability metadata:

```text
Capability
├── name
├── version
├── operation
├── side_effect_class
├── input_schema
├── required_permissions
└── executor
```

Side-effect classes:

```text
READ
TRANSFORM
VERIFY
MUTATE
IRREVERSIBLE
```

## W1-T4 — governance

Reuse Fenix `PolicyEngine` and `ApprovalService`; do not introduce another policy engine.

```text
READ / TRANSFORM / VERIFY
  → permission / policy
  → execute

MUTATE
  → permission / policy
  → approval when required
  → execute

IRREVERSIBLE
  → permission / policy
  → mandatory approval
  → execute
```

Mutating and irreversible capabilities are fail-closed when their required governor is absent or unresolved.

## W1-T5 — audit and correlation

Current verified gaps:

- `ctxkeys` has WorkspaceID/UserID/RunID but no TraceID/ExecutionID.
- `AuditService.LogWithDetails()` does not populate native `AuditEvent.TraceID`.
- tool audit currently skips tools outside the built-in whitelist.
- `execution_id` has no audit representation.

Implementation target:

```text
ctxkeys
├── WorkspaceID
├── UserID
├── RunID
├── TraceID
└── ExecutionID
```

- propagate `trace_id` into native audit storage;
- store `execution_id` in audit metadata initially;
- audit all registered governed tools, not only built-ins;
- keep only parameter keys in audit, not full payloads;
- inject run/trace/execution identity from runtime paths before closing T5.

## Mermaid2SF integration — W2 current state

Existing semantic contract:

```text
Salesforce Flow XML ⇄ FlowIR v2 ⇄ Mermaid
```

Implemented Fenix-facing capabilities:

```text
salesforce.flow.import@1
salesforce.flow.export@1
salesforce.flow.validate@1
salesforce.flow.compare@1
```

The request/result contract is transport-neutral and FlowIR v2 remains opaque to Fenix. Fidelity is artifact-scoped: family support is only a ceiling and cannot automatically become a runtime `guaranteed` verdict.

W2 also defines structured semantic diff, typed diagnostics, and an agent-safe consumption policy: guaranteed results may be used automatically, partial results require review, and unsupported/rejected results require abstention.

See `docs/plans/fenix-integration-w2-mermaid2sf-contract.md`.

W2 contract status: CLOSED. Runtime adapter/transport selection remains a later integration concern.

## VEL prerequisite

Current standalone VEL `POST /v1/events` evaluates through a local policy engine.

W3 must define the Fenix integration path where Fenix remains the policy authority and VEL records/verifies that authorization evidence without re-deciding it.

## Dependency spine

```text
W0 CLOSED
   ↓
W1-T1/T2
   ↓
W1-T3
   ├── W1-T4
   └── W1-T5
        ↓
      W1-T6
        ↓
   ┌────┴────┐
   W2        W3
   └────┬────┘
        W4
        ↓
        W5
        ↓
        W6
```
