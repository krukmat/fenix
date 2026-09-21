# W2 Mermaid2SF Integration Contract v1

> Status: W2-T1 / W2-T2 / W2-T3 implemented  
> Contract owner: Fenix for governed invocation; Mermaid2SF for Salesforce Flow semantics and fidelity truth.  
> Transport: intentionally undefined.

## Evidence baseline

This contract was derived from Mermaid2SF `main` at commit `c91df096a5accb3dddf2efd63db43d8b5109e7e7`.

Provider evidence used:

- `SUPPORTED_FEATURES.md` — public fidelity contract.
- `PROJECT_PLAN.md` — canonical FlowIR v2 architecture and verification status.
- `src/cli/commands/compile.ts` — Mermaid → FlowIR → validation → Salesforce XML path.
- `src/cli/commands/decompile.ts` and `src/reverse/xml-parser.ts` — Salesforce XML → FlowIR → Mermaid path.
- `src/validator/salesforce-semantic-validator.ts` — stable semantic validation rules and `M2SF-SF-*` codes.
- `src/utils/flow-semantic.ts` — semantic snapshot/diff primitive.

## W2-T1 — capability catalog

Only operations backed by current production semantic paths are contracted now:

| Capability | Input | Output | Side effect |
|---|---|---|---|
| `salesforce.flow.import@1` | Salesforce Flow XML | FlowIR v2 + Mermaid | TRANSFORM |
| `salesforce.flow.export@1` | Mermaid or FlowIR v2 | Salesforce Flow XML + FlowIR v2 | TRANSFORM |
| `salesforce.flow.validate@1` | Mermaid or FlowIR v2 | semantic verdict | VERIFY |

Explicitly deferred:

- `flow.compare`: Mermaid2SF already has a semantic diff primitive, but W2-T4 owns the integration contract.
- `flow.roundtrip`: currently a correctness/evidence workflow, not a runtime capability.
- `flow.inspect`: no stable structured provider surface exists yet.

No CLI, HTTP, MCP, sidecar, or library transport is selected by this catalog.

## W2-T2 — request/result contract

Fenix owns only the boundary contract. It does not copy or reinterpret FlowIR.

```text
Request
├── contract_version = "1"
├── operation
└── input
    ├── format
    ├── name?
    └── content
```

Artifact formats:

- `salesforce_flow_xml`
- `mermaid`
- `flowir_v2`

FlowIR is opaque payload content from Fenix's perspective.

```text
Result
├── contract_version
├── operation
├── status
│   ├── succeeded
│   ├── rejected
│   └── unsupported
├── artifacts[]
├── fidelity
├── semantic_metadata
└── provider_reference?
```

Typed diagnostic normalization remains W2-T5.

## W2-T3 — fidelity contract

Mermaid2SF's published fidelity is feature-scoped. A Flow family being supported does **not** mean any arbitrary artifact in that family is guaranteed.

Runtime levels:

- `guaranteed`: the concrete artifact remains inside the documented guaranteed subset.
- `partial`: some semantics are represented while known metadata/features fall outside the guaranteed subset.
- `unsupported`: the requested semantic operation is outside the supported contract.

Guaranteed-family ceilings currently exist for:

- Autolaunched Flow.
- Screen Flow.
- Record-Triggered After Save.
- Record-Triggered Before Save.
- Record-Triggered Before Delete.
- Schedule-Triggered.
- Platform Event-Triggered.

Orchestrated Flow remains:

- import/reverse: at most `partial`;
- export/forward: `unsupported`;
- round-trip: `unsupported`.

Important invariant:

```text
family support ceiling ≠ runtime artifact verdict
```

Fenix must never convert a family-level `guaranteed subset` claim into `fidelity.level=guaranteed` without provider evidence for the concrete artifact.

A `guaranteed` runtime report cannot contain `unsupported_features`. A `partial` report must expose the semantic boundary instead of silently dropping unsupported metadata.

Verification scopes are separate from fidelity:

- `semantic_roundtrip`
- `canonical_fixture_org_dry_run`
- `none`

The authenticated Salesforce dry-runs documented by Mermaid2SF prove canonical fixtures/subsets, not universal target-org acceptance for every runtime artifact.

## Ownership

```text
Fenix
 ├── capability invocation
 ├── execution identity
 ├── policy / approval
 ├── retries / audit
 └── interpretation of fidelity outcome

Mermaid2SF
 ├── FlowIR v2
 ├── Mermaid semantics
 ├── Salesforce XML semantics
 ├── semantic validation
 └── fidelity truth

Salesforce
 └── actual org acceptance
```

## Next

W2-T4 defines semantic comparison as an integration capability without leaking FlowIR internals into Fenix.

W2-T5 then normalizes diagnostics.

W2-T6 defines which operations are safe for agent invocation.
