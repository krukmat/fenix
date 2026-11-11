---
doc_type: task
id: DB-FENIX-B2.E4
title: "negative SQLite admission and idempotency tests"
status: cloud_source_candidate_qa_unverified
phase: B2.E
week: 2026-W46
tags: [dubbridge, fenix, s2s, admission, cloud]
fr_refs: []
uc_refs: []
traceability_scope: integration_infrastructure_no_product_fr_uc
blocked_by: [DB-FENIX-B2.E1, DB-FENIX-B2.E2, DB-FENIX-B2.E3]
blocks: [DB-FENIX-B2.E5]
files_affected:
  - internal/domain/integration/s2s_admission_test.go
criticality: critical
criticality_basis: "An incorrect request or policy boundary may disclose tenant authority, admit a forged scope, duplicate an operation or misrepresent execution; independent review is mandatory for formal closure."
created: 2026-11-11
completed: null
---

# DB-FENIX-B2.E4 — negative SQLite admission and idempotency tests

**Direction:** DubBridge consumes Fenix. **Owner:** ChatGPT Cloud. **RRI:** N/A (DB-FENIX-INT cloud). No local agent execution or tokens are requested. **Task status is a source candidate; not a gate PASS.**

## Task card
Task: DB-FENIX-B2.E4
Task file: docs/tasks/task_dubbridge_fenix_b2e4_inert_tests.md
Plan file: docs/plans/dubbridge-consumes-fenix-integration.md
Summary: Test actual persisted S2S credential, strict envelope rejection, forged org/project, revoked principal, disabled capability, absent input validator, idempotent replay, conflict, safe failure projection and absence of agent runs.
Code affected: internal/domain/integration/s2s_admission_test.go
Criticality: critical
Criticality basis: Cross-tenant isolation, bounded idempotent admission and no accidental provider execution.
Effort/reasoning: High - Review typed authority and rejection invariants; implement only the inert cloud scope.
Recommended model: Cloud author (reviewer delegated only when an approved reviewer is available)
Estimated tokens: ~6500
Pseudocode: create isolated migrated DB and test principal+credential → submit positive/replay → vary one invalid condition per case → verify denial and zero new operations/runs
Peer readiness review approval: reviewer=PENDING; artifact=NOT_RUN_NO_LOCAL_TOKENS; status=PENDING
Peer code review approval: reviewer=PENDING; artifact=NOT_RUN_NO_LOCAL_TOKENS; status=PENDING

## System context

```text
DubBridge caller (UNTRUSTED request/bearer)
    |
    v
Fenix S2S credential verifier (persisted principal)
    |
    v
B2.E preflight: bounded v1 envelope + exact Fenix scope
    |
    v
B2.B default-deny binding + REQUIRED input validator
    |
    v
B2.C2 atomic SQLite reservation  ---> safe receipt only (no status)
    |
    +-- denial/invalid/conflict --> static sanitized projection
    |
    X  NO HTTP route / NO agent dispatch / NO credentials generated
```

## Implementation and acceptance

Test actual persisted S2S credential, strict envelope rejection, forged org/project, revoked principal, disabled capability, absent input validator, idempotent replay, conflict, safe failure projection and absence of agent runs.

- All fields and negative cases remain capability-neutral. The input validator is mandatory, not registered automatically.
- Trusted principal and tenant are derived from Fenix authentication; body `source` is only checked against that identity, never used to grant scope.
- Receipt and error projections exclude internal records, identifiers, hashes, SQL text, runtime state and secrets.
- No route, C4 status GET, live worker, CI change, production migration, credential activation, deployment or merge.
- The Go 1.25.13 focused/full/race/vet/prepush and independent peer gates remain **NOT_RUN** in the current Cloud-only session; no waiver or PASS is inferred.
- QA/source review is the E5 follow-on when an approved execution channel is available; this task's candidate source can be further inspected by ChatGPT Cloud.

## Implementation sketch

create isolated migrated DB and test principal+credential → submit positive/replay → vary one invalid condition per case → verify denial and zero new operations/runs

