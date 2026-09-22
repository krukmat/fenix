---
doc_type: plan
id: W5-FINAL
title: Fenix Integration W5 Final Validation
status: closed
phase: integration
tags: [fenix, integration, w5, validation, mermaid2sf, vel, governance]
created: 2026-09-22
---

# W5-FINAL — Integration Validation

## Goal

Close W5 by validating the integrated Fenix → Mermaid2SF / VEL runtime as one governed system,
without adding another architecture layer or reopening W5-C for its accepted 0.1 pp coverage waiver.

## Preconditions

- W1-W4 contracts/runtime governance: CLOSED.
- W5-A readiness inventory: CLOSED.
- W5-B provider/adapters/runtime wiring: IMPLEMENTED.
- W5-C operational readiness: CLOSED under documented QA waiver.
- M2SF and VEL remain independently optional according to the Fenix governance decision.

## Validation slices

### F1 — Governed Mermaid2SF execution

Validate a representative agent/governed execution through Fenix into Mermaid2SF and back:

```text
Fenix capability
  -> governance decision
  -> M2SF adapter
  -> FlowIR / provider result
  -> normalized Fenix result
  -> audit / correlation
```

Confirm trace/execution identity, fidelity/diagnostics and no bypass of Fenix governance.

### F2 — Governed VEL evidence lifecycle

Validate a representative governed execution with evidence enabled:

```text
business capability
  -> durable EvidenceEnvelope
  -> VEL append
  -> checkpoint
  -> independent verify
  -> compact ProofReference
  -> Fenix audit / agent-safe status
```

Confirm no business replay during evidence reconciliation and no full VEL bundle persisted in Fenix.

### F3 — Independent participation matrix

Exercise the intended combinations:

| M2SF | VEL | Expected |
|---|---|---|
| yes | no | capability executes without evidence lifecycle |
| yes | yes | capability executes and evidence lifecycle participates |
| no | yes | non-M2SF governed capability may still produce VEL evidence |
| no | no | ordinary Fenix governed execution remains valid |

This verifies that M2SF and VEL are capabilities/providers, not mandatory coupled services.

### F4 — Failure and recovery

Validate at least:

- provider deterministic failure;
- indeterminate VEL append + idempotency lookup/retry;
- restart-safe outbox recovery;
- checkpoint/verification failure exposure;
- no duplicate governed business action.

### F5 — Cross-repo compatibility

Confirm the Fenix contracts still match the provider surfaces currently present in:

- Mermaid2SF;
- verifiable-event-ledger.

No semantic authority is copied into Fenix.

### F6 — Final documentation and handoff

Update the integration baseline with:

- final W5 state;
- runtime interaction diagram;
- accepted W5-C QA waiver;
- known residual risks/debt;
- entry criteria for W6.

## Exit criteria

W5-FINAL closes when the integrated behavior above is demonstrated with existing tests or focused
cross-repo validation and there is no material contract/runtime defect blocking functional W6 work.

The W5-C 82.9% vs 83.0% coverage shortfall is explicitly out of scope for reopening unless a later
change creates a functional regression.

## Validation evidence — 2026-09-22

### F1 — Governed Mermaid2SF execution: PASS

Fenix already exercises the concrete M2SF adapter with correlation headers and normalized contract
results in `internal/infra/integration/m2sf/executor_test.go`. The current Mermaid2SF main
(`34d94c9d`) exposes authenticated `POST /api/fenix/flow` and supports:

- `salesforce.flow.import`
- `salesforce.flow.export`
- `salesforce.flow.validate`
- `salesforce.flow.compare`

FlowIR v2 and Salesforce/Mermaid semantics remain provider-owned.

### F2 — Governed VEL evidence lifecycle: PASS

Fenix tests cover external-authority append, execution-id idempotency, lookup, checkpoint/verify
progression and durable outbox recovery. The current VEL main (`69205324`) exposes:

- `POST /v1/external/events`
- `GET /v1/external/events/by-idempotency`
- `POST /v1/streams/{stream_id:path}/checkpoints`
- `GET /v1/streams/{stream_id:path}/bundle`
- `POST /v1/verify`

Fenix persists compact proof material only.

### F3 — Independent participation matrix: PASS

Existing ToolRegistry runtime tests prove that provider execution and evidence recording are
independently governed: provider-only, provider+evidence, evidence of denial without provider
invocation, and ordinary no-evidence execution are all valid paths.

### F4 — Failure and recovery: PASS

Existing runtime/outbox tests cover retryable provider failure, indeterminate evidence delivery,
lookup-before-retry, restart-safe reconciliation, checkpoint failure, verification failure and
the invariant that evidence recovery does not replay the business capability.

### F5 — Cross-repo compatibility: PASS by source-contract inspection

Fenix adapter paths and payload assumptions match the current provider surfaces on Mermaid2SF
`34d94c9d` and VEL `69205324`. No semantic or cryptographic authority is duplicated into Fenix.

### F6 — Documentation and handoff: PASS

W5-C waiver, W5 final state and W6 entry point are documented in the integration baseline and W6 plan.

## Residual risks

- No final live three-process smoke (Fenix + Mermaid2SF + VEL) was executed in this closure pass.
  Current confidence is based on focused adapter/runtime tests plus direct provider-contract inspection.
- W5-C retains the explicit 82.9% vs 83.0% coverage waiver.
- Deployment/network/secret-manager hardening remains an operational concern outside this integration wave.

None of these residuals blocks W6 functional execution under the current proof-of-concept scope.

## Closure

**W5-FINAL: CLOSED — 2026-09-22.**

W5 is complete. The next integration phase is W6 functional execution: exercising complete agent
use cases over the governed capability/runtime infrastructure rather than adding another integration layer.
