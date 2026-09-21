# W5-A — Readiness Gap Inventory

> Date: 2026-09-21  
> Status: CLOSED  
> Scope: readiness inventory only; no transport/topology selection in W5-A.  
> CI policy: automatic GitHub Actions CI remains disabled during active W5 implementation; manual `workflow_dispatch` only for explicit checkpoints.

## Goal

W5-A converts the contracts and runtime governance completed in W1-W4 into an explicit inventory of what is still missing before Fenix can operate Mermaid2SF (M2SF) and Verifiable Event Ledger (VEL) as real external capabilities.

W5-A deliberately does **not** choose HTTP vs CLI vs sidecar vs embedded-library execution. It identifies the concrete gaps that W5-B must resolve.

The reference interaction remains:

```text
Agents
   ↓
Agentic Blackboard
   ↓
Fenix Governance
   ├── provider execution → M2SF when requested
   └── evidence planning  → VEL when evidence_planned=true
                                ↓
                          ProofReference
                                ↓
                         Fenix audit/context
                                ↓
                    agents may consume status
```

Agents and the Blackboard do not call VEL as an independent policy/evidence authority. Fenix remains the mediation and governance boundary.

---

## W5-A-T1 — Runtime integration inventory

### Already implemented in Fenix

The integration spine is present and transport-neutral:

- `ToolRegistry.RegisterCapability(...)` provides the external capability seam.
- `CapabilityGovernancePlanner` resolves one governance decision per logical execution.
- `CapabilityEvidenceRecorder` is independent from provider execution.
- M2SF capability catalog exists for:
  - `salesforce.flow.import@1`
  - `salesforce.flow.export@1`
  - `salesforce.flow.validate@1`
  - `salesforce.flow.compare@1`
- M2SF request/result, fidelity, semantic diff, diagnostics and agent-safe consumption contracts exist.
- `evidence.RuntimeRecorder` adapts W4 execution outcomes into the W3 `EvidenceEnvelope`.
- `evidence.RuntimeSink` defines:
  - `RecordEvidence(...)`
  - `LookupEvidence(stream_id, execution_id)`
- W4 audit includes governance, provider outcome, evidence lifecycle and compact proof metadata.
- Provider retries keep one `execution_id` and do not recompute governance.
- Evidence reconciliation is explicitly forbidden from replaying the business action.

### Missing concrete runtime bindings

The router currently installs the W4 governance planner but does **not** install:

- a concrete M2SF `ToolExecutor` for any of the four Salesforce Flow capabilities;
- a concrete VEL `RuntimeSink`;
- a production evidence recorder backed by a VEL adapter.

This means W1-W4 define and test the integration behavior, but no real M2SF or VEL provider call is yet made from the Fenix application runtime.

### T1 conclusion

```text
Contract/runtime policy spine     READY
Concrete M2SF execution adapter   MISSING
Concrete VEL evidence adapter     MISSING
Router provider wiring            MISSING
```

---

## W5-A-T2 — Mermaid2SF execution gap analysis

### What Mermaid2SF already provides

M2SF has the semantic core required by the Fenix W2 contract:

```text
Salesforce XML ⇄ FlowIR v2 ⇄ Mermaid
```

Verified provider primitives include:

- Mermaid parse → FlowIR v2;
- FlowIR validation;
- Salesforce-specific semantic validator and stable `M2SF-SF-*` diagnostics;
- FlowIR → Salesforce Flow XML;
- Salesforce XML → FlowIR v2;
- FlowIR → Mermaid;
- canonical semantic snapshot and semantic equality comparison;
- documented fidelity/guaranteed Flow-family subset.

M2SF also has two executable surfaces today:

1. CLI:
   - compile
   - decompile
   - lint
   - explain
2. Minimal Node HTTP facade:
   - `GET /health`
   - `POST /api/compile`
   - `POST /api/decompile`

### Gap M2SF-1 — no Fenix executor

Fenix does not register a `ToolExecutor` backed by M2SF.

Blocking effect:

```text
Fenix GovernanceDecision
        ↓
allowed=true
        ↓
NO concrete provider executor
```

Owner: Fenix integration layer.

### Gap M2SF-2 — HTTP facade does not cover the complete Fenix capability catalog

Current M2SF HTTP surface roughly covers:

```text
/api/compile   ≈ export
/api/decompile ≈ import
```

But Fenix also defines:

```text
salesforce.flow.validate
salesforce.flow.compare
```

Those semantics exist internally in M2SF, but they are not exposed as equivalent HTTP provider operations.

Owner: M2SF provider surface or Fenix adapter, depending on W5-B transport choice.

### Gap M2SF-3 — provider response shape != Fenix semantic Result

The Fenix result contract expects:

```text
status
artifacts[]
fidelity
semantic_metadata
diff?
diagnostics[]
provider_reference?
```

The current HTTP facade returns a simpler provider-native shape such as:

```text
dsl
xml
errors[]
warnings[]
```

The adapter must normalize provider-native output into the Fenix contract. Fenix must not absorb M2SF FlowIR types.

Owner: Fenix M2SF adapter.

### Gap M2SF-4 — semantic compare shape requires adaptation

M2SF currently exposes an internal semantic equality primitive based on canonical snapshots. Fenix owns a transport-neutral structured semantic diff contract.

The adapter/provider surface must bridge those representations without moving FlowIR ownership into Fenix.

Owner: M2SF provider surface + Fenix adapter contract mapping.

### Gap M2SF-5 — operational controls are not defined for Fenix calls

Still undefined:

- provider timeout;
- retry classification for transport failure;
- maximum artifact/request size;
- cancellation propagation;
- provider concurrency limit;
- provider reference/request id;
- health/readiness handling;
- service authentication if M2SF runs out-of-process.

The existing W1 retry rules remain authoritative: retries may not change the logical `execution_id`.

### Gap M2SF-6 — deployment packaging is provider-specific and not standardized

M2SF has a deployable Node process and documented PM2/Apache deployment, but no integration-specific deployment contract is selected for Fenix.

W5-A does not choose whether this becomes:

- a separate HTTP service;
- a sidecar/local Node process;
- CLI subprocess;
- another adapter form.

### M2SF readiness summary

```text
Semantic core               READY
FlowIR v2 ownership         READY
Fidelity contract           READY
CLI provider surface        AVAILABLE
HTTP compile/decompile      AVAILABLE / PARTIAL
HTTP validate/compare       MISSING
Fenix executor              MISSING
Response normalization      MISSING
S2S authentication          MISSING
Timeout/retry operations    MISSING
Deployment choice           DEFERRED TO W5-B
```

---

## W5-A-T3 — VEL execution gap analysis

### What VEL already provides

VEL already implements the required cryptographic/domain primitives:

- RFC 8785 canonical evidence;
- SHA-256 + Ed25519 signatures;
- per-stream hash chain;
- Merkle checkpoints;
- key lifecycle;
- verification bundles;
- independent verification;
- `Ledger.append_external(...)`;
- externally supplied Fenix authorization evidence validation;
- required external `idempotency_key`;
- same-key/same-evidence idempotent replay;
- conflicting replay rejection;
- `Ledger.get_by_idempotency(...)`;
- health/readiness/metrics primitives.

The Fenix/VEL authority boundary is therefore already valid at domain level.

### Gap VEL-1 — external-authority append is not exposed by the current FastAPI surface

The current HTTP endpoint:

```text
POST /v1/events
```

uses:

```text
EventCreate
  → Ledger.append(...)
  → LocalPolicyEngine
```

The Fenix integration path requires:

```text
ExternalEventCreate
  → Ledger.append_external(...)
  → NO LocalPolicyEngine
```

That domain path exists but has no corresponding FastAPI endpoint today.

This is a blocking gap for an HTTP-based Fenix → VEL adapter.

Owner: VEL provider surface.

### Gap VEL-2 — idempotency lookup is domain-only

VEL implements:

```text
Ledger.get_by_idempotency(stream_id, execution_id)
```

Fenix `RuntimeSink` requires an equivalent:

```text
LookupEvidence(stream_id, execution_id)
```

No direct HTTP endpoint currently exposes that lookup.

Owner: VEL provider surface if HTTP is selected.

### Gap VEL-3 — Fenix has no concrete RuntimeSink

Fenix has the interface and recorder:

```text
RuntimeRecorder
   ↓
RuntimeSink
├── RecordEvidence
└── LookupEvidence
```

but there is no implementation bound to VEL and no `SetCapabilityEvidenceRecorder(...)` wiring in the application router.

Owner: Fenix integration layer.

### Gap VEL-4 — ProofReference assembly is not yet a concrete adapter

VEL currently exposes raw provider material across several surfaces:

- stored event;
- checkpoint;
- verification bundle;
- verification result;
- public key lookup.

Fenix expects one compact `ProofReference v1`.

A concrete adapter must assemble and validate the Fenix proof projection without copying the complete bundle into Fenix audit.

Owner: Fenix VEL adapter.

### Gap VEL-5 — checkpoint policy is undefined

VEL can create checkpoints explicitly:

```text
POST /v1/streams/{stream_id}/checkpoints
```

but neither Fenix nor VEL defines the production policy for when they are created.

Still open:

- event-count threshold;
- time-based interval;
- checkpoint-on-sensitive-action;
- explicit/manual only;
- combination policy.

Owner: cross-component operational policy; decision belongs to W5-B/W5-C.

### Gap VEL-6 — stream partition strategy is undefined

Fenix currently constructs:

```text
stream_id = "workspace/" + workspace_id
```

This is a usable default mapping but has not been accepted as the production partition strategy.

Open alternatives include partitioning by:

- workspace;
- workspace + capability family;
- workspace + time window;
- another bounded operational partition.

The decision affects contention, checkpoint size, verification cost and retention.

Owner: Fenix + VEL architecture.

### Gap VEL-7 — service identity/authentication is missing

VEL explicitly defers authentication of the Fenix caller.

If VEL runs as a separate service, the external-authority append path must not be publicly callable without proving that Fenix is an authorized external authority.

Owner: deployment/security adapter.

### VEL readiness summary

```text
Cryptographic core                 READY
External-authority domain seam     READY
External idempotency semantics     READY
HTTP external append               MISSING
HTTP idempotency lookup            MISSING
Fenix RuntimeSink                  MISSING
ProofReference adapter             MISSING
Checkpoint policy                  MISSING
Stream policy                      MISSING
S2S authentication                 MISSING
Deployment choice                  DEFERRED TO W5-B
```

---

## W5-A-T4 — Operational topology gaps

W5-A does not select topology, but the following decisions are now explicitly open.

### M2SF

Need a single supported runtime form for Fenix:

```text
Fenix
  ├── HTTP service?
  ├── sidecar/local Node service?
  ├── CLI subprocess?
  └── embedded/other adapter?
```

Operational requirements to resolve after selection:

- process lifecycle;
- startup/readiness;
- artifact size bounds;
- request cancellation;
- timeout policy;
- concurrency;
- deployment version pinning;
- authentication where applicable.

### VEL

Current provider dependencies are:

```text
VEL application
   ↓
PostgreSQL
   ↓
signing key material
```

The repository provides PostgreSQL compose support, but the application deployment topology is not yet bound to Fenix.

Open questions:

- separate service vs same-host sidecar/service;
- database ownership/backup;
- signing key mount/storage;
- runtime vs owner PostgreSQL identities;
- network isolation;
- health/readiness dependency;
- trust-anchor placement;
- recovery order.

### Topology invariant

Regardless of deployment:

```text
Fenix = governance authority
M2SF  = Flow semantic authority
VEL   = cryptographic evidence authority
```

Topology must not collapse these authority boundaries.

---

## W5-A-T5 — Observability and reconciliation gaps

### Already available

Fenix audit already records:

- `trace_id`;
- `execution_id`;
- capability execution status;
- attempt count;
- governance decision fields;
- evidence lifecycle state;
- compact proof projection when present.

VEL already exposes:

- `/health`;
- `/ready`;
- lightweight append/checkpoint/verify metrics.

### Gap OBS-1 — provider-call metrics are not standardized

Fenix does not yet expose a dedicated cross-platform metric set for:

- M2SF latency;
- M2SF failures by operation/code;
- VEL append latency;
- VEL lookup/reconciliation latency;
- evidence lifecycle counts;
- verification failures;
- checkpoint age/lag.

Owner: Fenix integration observability + provider adapters.

### Gap OBS-2 — provider request correlation is undefined

`trace_id` and `execution_id` exist in Fenix, but no concrete transport defines how they are propagated to M2SF/VEL request metadata or logs.

Owner: adapter contract.

### Gap OBS-3 — reconciliation is modeled but not durable

Fenix has:

```text
DecideReconciliation(...)
RuntimeRecorder.reconcileIndeterminateRecord(...)
```

The current recorder performs an immediate idempotency lookup after an indeterminate append.

What does not yet exist is an operationally durable reconciliation mechanism for unresolved states across process restart/outage:

- persistent evidence-delivery record/outbox;
- scheduled retry/lookup worker;
- backoff policy;
- checkpoint-await polling/scheduling;
- durable escalation state.

This is a blocking production-readiness gap for asynchronous or outage-safe evidence delivery.

Owner: Fenix operational runtime.

### Gap OBS-4 — checkpoint lifecycle is not closed

Fenix understands:

```text
recorded
pending_checkpoint
verified
verification_failed
```

but there is no runtime worker/policy that advances a recorded proof through checkpoint creation and verification.

Owner: Fenix + VEL adapter/runtime.

### Gap OBS-5 — agent/Blackboard exposure must remain mediated

Agents may consume a Fenix-projected verification state, but should not query raw VEL bundles or signing material as their normal path.

Target interaction:

```text
VEL
 ↓
ProofReference / verification state
 ↓
Fenix audit/context
 ↓
Agentic Blackboard / Agent context
 ↓
agents
```

Required W5 implementation rule:

- never persist chain-of-thought or complete Blackboard content into VEL;
- VEL evidence contains only minimized governed execution evidence/digests;
- agents may claim cryptographic verification only when Fenix exposes a verified proof state.

---

## W5-A-T6 — Consolidated readiness matrix

Legend:

- **READY** — implemented and usable at the stated boundary.
- **PARTIAL** — substantial implementation exists but cannot yet satisfy the full Fenix integration contract.
- **MISSING** — required runtime/operational implementation does not exist.
- **DEFERRED** — W5-A intentionally identifies but does not decide it.

| Concern | Contract | Fenix runtime | Provider surface | Ops | Security | Observability | Blocking gap | Owner |
|---|---|---|---|---|---|---|---|---|
| M2SF import | READY | MISSING | PARTIAL | DEFERRED | PARTIAL | PARTIAL | concrete executor + normalization | Fenix/M2SF |
| M2SF export | READY | MISSING | PARTIAL | DEFERRED | PARTIAL | PARTIAL | concrete executor + normalization | Fenix/M2SF |
| M2SF validate | READY | MISSING | MISSING over HTTP | DEFERRED | PARTIAL | PARTIAL | provider operation + executor | Fenix/M2SF |
| M2SF compare | READY | MISSING | MISSING over HTTP | DEFERRED | PARTIAL | PARTIAL | provider operation + diff mapping | Fenix/M2SF |
| M2SF fidelity/diagnostics | READY | READY contractually | READY internally | DEFERRED | N/A | PARTIAL | runtime response mapping | Fenix adapter |
| VEL external append | READY | PARTIAL | MISSING over HTTP | DEFERRED | MISSING S2S auth | PARTIAL | external-authority provider endpoint + sink | Fenix/VEL |
| VEL idempotency lookup | READY | PARTIAL | MISSING over HTTP | DEFERRED | MISSING S2S auth | PARTIAL | provider lookup endpoint + sink mapping | Fenix/VEL |
| VEL ProofReference | READY | READY contractually | PARTIAL raw material | DEFERRED | PARTIAL | READY audit projection | adapter assembly | Fenix |
| VEL checkpoint | READY | PARTIAL state model | READY explicit API | MISSING policy | PARTIAL | PARTIAL | schedule/trigger policy | Fenix/VEL |
| VEL verification | READY | PARTIAL state model | READY bundle+verify | DEFERRED | PARTIAL | PARTIAL | orchestration into ProofReference lifecycle | Fenix/VEL |
| Evidence reconciliation | READY | PARTIAL immediate lookup | READY domain primitive | MISSING durable worker | PARTIAL | PARTIAL | durable outbox/reconciler | Fenix |
| Stream partitioning | READY field | default exists | READY streams | MISSING policy | N/A | PARTIAL | accepted partition strategy | Fenix/VEL |
| Provider correlation | READY ids | READY locally | MISSING transport propagation | DEFERRED | N/A | MISSING standardized metrics | adapter layer |
| Agent/Blackboard proof exposure | READY policy | PARTIAL via audit/context | N/A | DEFERRED | READY boundary rule | PARTIAL | explicit context projection when needed | Fenix |

---

## Blocking gap register

The blockers that must be resolved before claiming real integrated execution are:

| ID | Blocker | Dependency | Primary owner |
|---|---|---|---|
| W5-G01 | No concrete M2SF executor registered in Fenix | transport decision | Fenix |
| W5-G02 | M2SF provider surface does not expose all four Fenix operations in one stable runtime contract | transport decision/provider adapter | M2SF + Fenix |
| W5-G03 | No concrete VEL `RuntimeSink` wired in Fenix | transport decision | Fenix |
| W5-G04 | VEL external-authority append is not exposed through current HTTP API | only blocking if HTTP selected | VEL |
| W5-G05 | VEL idempotency lookup is not exposed through current HTTP API | only blocking if HTTP selected | VEL |
| W5-G06 | Service-to-service authentication for external providers is undefined | topology decision | Fenix + providers |
| W5-G07 | Evidence reconciliation has no durable persistence/worker | sink + storage design | Fenix |
| W5-G08 | Checkpoint scheduling/verification progression is undefined | VEL topology/operations | Fenix + VEL |
| W5-G09 | Production stream partition strategy is undefined | workload/topology assumptions | Fenix + VEL |
| W5-G10 | Provider metrics/correlation propagation is not standardized | adapter design | Fenix |

---

## W5-A closure result

W5-A is complete because every known integration concern now has:

- current implementation state;
- concrete missing piece;
- dependency;
- owner;
- blocker status.

The architecture is **not** blocked by missing semantic contracts. The remaining work is primarily adapter, provider-surface and operational productization work.

The highest-level readiness picture is:

```text
                    CONTRACTS       REAL RUNTIME
                    ---------       ------------
Fenix governance       READY            READY
M2SF semantics         READY            PARTIAL
VEL evidence model     READY            PARTIAL
M2SF adapter            —               MISSING
VEL adapter             —               MISSING
durable reconciliation  —               MISSING
operational topology    —               DEFERRED
```

## Entry condition for W5-B

W5-B may now select the concrete integration mechanism using the gap register above.

It must preserve these non-negotiable invariants:

1. Fenix remains the only governance/approval authority.
2. M2SF owns FlowIR and Salesforce Flow semantics.
3. VEL owns cryptographic evidence and verification.
4. Agents/Blackboard do not bypass Fenix to make independent VEL policy decisions.
5. `execution_id` remains stable across provider/evidence retries.
6. Evidence recovery never replays the business capability.
7. Raw Blackboard reasoning/chain-of-thought is not written to VEL.
