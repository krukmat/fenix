---
doc_type: task
id: DB-FENIX-B2.A3.T1
title: "B2.A3.T1 isolated S2S middleware readiness"
status: readiness_passed
phase: B2.A3
week: 2026-W45
tags: [dubbridge, fenix, s2s, security, middleware, readiness]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: [DB-FENIX-B2.A3.T2]
files_affected:
  - docs/tasks/task_dubbridge_fenix_b2a3_middleware_readiness.md
  - docs/plans/dubbridge_fenix_s2s_security_plan.md
criticality: critical
criticality_basis: "The proposed guard governs service authentication and organization/project authorization; omitting it could cross tenant boundaries. RRI: N/A for DB-FENIX-INT/cloud."
created: 2026-11-08
completed: 2026-11-08
---

# B2.A3.T1 isolated S2S middleware readiness

**Plan**: [DubBridge to Fenix S2S security foundation](../plans/dubbridge_fenix_s2s_security_plan.md#current-direction).
**Issue**: [DB-FENIX-B2.A3 #5](https://github.com/krukmat/fenix/issues/5).
**Baseline**: `87fd10d3e10658179d6f4ab626d33e2cf555539c` on `feature/int-v2-dubbridge-consumes-fenix`.
**RRI**: N/A — DB-FENIX-INT/cloud. No RRI calculator or Ollama delegation.

## Scope and boundary

This is a documentation and independent task-readiness gate. Record a reviewable contract for later B2.A3 implementation and testing. Do not modify functional code, mount or activate an endpoint, issue a credential, change production SQLite, run a workflow, merge, or dispatch CI. The user's 2026-11-08 follow-up explicitly authorizes committing and pushing this T1 documentation after local QA. A readiness PASS approves the plan's clarity; it is not permission to implement an auth boundary.

At the baseline, `DubBridgeServiceAuth` in `internal/api/middleware/dubbridge_service_auth.go` validates the dedicated bearer and inserts a private service identity, then calls `next`. It does **not** call `AuthorizeDubBridgeSource`; that helper is currently optional to the downstream handler. The existing isolated test calls the helper inside its handler, which does not prove that a future handler cannot omit it. No S2S route is mounted in `internal/api/routes.go`. Existing B2.B capabilities are default-deny and no real capability is registered.

The core readiness decision is that **every S2S handler must cross a mandatory source authorization guard before handler invocation, capability lookup, operation reservation, or runtime activity**. Construction/registration of an unguarded handler must be impossible through the proposed S2S route API. Human `/api/v1` auth remains separate; service identity must not populate human `ctxkeys`.

## Dependencies and authorization gates

| Dependency or gate | Evidence / required disposition |
|---|---|
| B2.A1 token format and B2.A2 store | Use the Fenix-owned opaque token and live SQLite principal/credential state; owner-accepted B2.A2 checkpoint and Go 1.25.13 evidence are linked from issue #3. Fresh security acceptance and global QA remain separately pending. |
| QA debt | Issue #4 records unrelated repository gate failures. Do not relabel a global failure as PASS. |
| Architecture and proposed ADR | `docs/architecture.md`, the strategic repositioning spec/plan, and `docs/decisions/ADR-DB-FENIX-S2S-PROPOSED.md` require workspace isolation, a distinct S2S principal, centrally provisioned source mapping, and default-deny capability execution. The S2S ADR is still proposed. |
| Independent readiness | Run `scripts/peer-workflow-review.py task-readiness` as `--caller-kind codex` with this task, the parent plan, and a concrete task-card preview. The reviewer must explicitly concur with or dispute `criticality: critical`. Only its real PASS can unblock task-card presentation. |
| Later implementation | B2.A3.T2 requires explicit owner HITL for auth/authorization changes, a separate task record and readiness gate, local Go QA, and independent code review. T1 does not supply that approval. |

## System context for the proposed guard

```text
 [DubBridge request: bearer + untrusted source / operation ID]
                            |
                            v
         +---------------------------------------------+
         | Isolated S2S ingress (currently not mounted) |
         | authenticate Fenix credential against SQLite|
         | resolve Fenix-owned workspace/org/project    |
         | authorize exact source or scoped operation   |
         | deny before dispatch on any missing/mismatch |
         +----------------------+----------------------+
                                | authorized, private service context only
                                v
           [S2S handler] -> [default-deny capability registry]
                                    | explicitly registered only
                                    v
                            [operation/runtime, later]

 [Human JWT /api/v1] ------> [human routes and ctxkeys, separate]

 Invariants: no handler sees an unscoped service request; caller-supplied
 workspace/org/project is never authority; GET/status resolves a stored
 operation using principal-bound workspace and source predicates; denied
 requests create no operation or agent run. The route owner composes the
 guard; the credential store owns its SQL reads; a future operation store
 owns scoped lookup. Errors return sanitized responses. The guard retains
 no goroutine and does not own DB lifecycle.
```

## High-level pseudocode for later implementation

```text
make_guarded_s2s_handler(verifier, source_resolver, scoped_operation_store, handler):
  reject missing dependencies; construct one compulsory wrapper for handler
  on request: reject unsupported method, malformed/duplicate bearer, invalid body
  authenticate token from current Fenix SQLite state; deny any error or inactive identity
  derive trusted workspace/org/project exclusively from authenticated principal
  if POST: parse one bounded, strict source envelope; require exact product/org/project binding
  if GET/status: look up operation by ID with principal, workspace, org and project predicates;
                 deny missing or mismatched operation before handler dispatch
  reject absent source, unknown scope, mismatch, or unavailable store with safe response
  attach private authorized service context; invoke handler only after all checks
  handler checks default-deny capability registry before any operation or agent run
```

The later design must specify how an org-only principal (`project_id` absent) is restricted to org-scoped requests, and how a project-bound principal matches only its exact project. No handler may call `AuthorizeDubBridgeSource` as an optional convention in place of the compulsory wrapper. A GET/status path cannot treat a client-provided operation ID as proof of source scope. The separate B2.B/B2.C capability and operation gates remain mandatory.

## Authorization and negative-test matrix for later tasks

| Case | Required decision before handler | Evidence required in later isolated tests |
|---|---|---|
| Valid active service credential, exact workspace/org/project | Allow only into the scoped handler; still deny an absent capability | Real migrated Fenix SQLite, `httptest`, no production route |
| Missing, duplicate, Basic, malformed or noncanonical Bearer; human JWT | 401, verifier and handler not called for malformed syntax | Distinct bearer cases and no secret in response |
| Expired, future-issued, revoked principal/credential, old token after rotation | Deny, no handler or durable write | Live store state; failed rotation preserves prior token |
| Fenix workspace mismatch or missing workspace | Deny, no tenant enumeration | Stored Fenix scope is authoritative; no client workspace override |
| Wrong/missing DubBridge org, wrong/missing project, org-only versus project-bound mismatch | Deny before handler and registry | Exact product/org/project comparison including empty project semantics |
| Forged request context, JSON source or header; direct handler entry attempt | Deny or make unguarded construction impossible | A handler spy stays at zero invocations for every denial |
| GET/status operation ID from another principal, workspace, org or project | Deny before handler | Principal-bound, tenant-scoped SQLite lookup; no ID-only lookup |
| SQLite unavailable; nil or typed-nil verifier; missing handler | Safe 503 or denial without panic | Error envelope is bounded and redacted; no downstream call |
| POST at 64 KiB and 64 KiB+1; unsupported method/content type or invalid JSON/depth | Accept only within explicit transport contract; otherwise deny before handler | Preserve 64 KiB cap; specify method and parser policy before code |
| Unknown/disabled capability | Default-deny before runtime, no 202, operation or `agent_run` | Separate B2.B/B2.C gate evidence; no auto-registration |
| Service bearer sent to human `/api/v1`; human JWT sent to S2S | Deny in both directions | No human `UserID`/`WorkspaceID` context from S2S |

## Acceptance criteria for B2.A3.T1

1. This task record has valid task frontmatter, plan/issue/baseline links, the precise planning-only scope, dependencies, criticality and its RRI N/A basis, the context diagram, pseudocode, and authorization matrix.
2. The contract identifies the current optional-helper bypass risk and requires a compulsory guard for **all** future S2S handlers, including operation read/status paths. It specifies default-deny, Fenix-owned authority, and no downstream side effects on denial.
3. The independent Claude Code reviewer runs through the repository script and explicitly concurs with or disputes criticality. A non-PASS verdict is reported as FAIL/BLOCKED; findings are corrected and the gate rerun where possible. No self-review is counted as independent.
4. Issue #5 receives the actual verdict, findings, exact baseline SHA, artifact path and review evidence, plus the complete task contract or an equivalent cloud-recoverable copy. The task file is explicitly added to Git despite `docs/tasks/.gitignore` under the user's subsequent publication authorization.
5. `git status` and `git diff` confirm that only T1 documentation changed and no functional code, route, credential, CI or deployment state was changed.

## Out-of-scope follow-up

B2.A3.T2 may implement the compulsory guard and bounded middleware refactor after its own HITL and peer gate. B2.A3.T3 may run real SQLite/`httptest` authorization tests. B2.A3.T4 may perform code/security review and QA handoff. Rate limiting by authenticated principal, response-size enforcement, strict depth/schema policy, operator runbooks and trusted-proxy decisions must be assigned to a separately authorized implementation layer before any activation. No follow-up is started by T1.
