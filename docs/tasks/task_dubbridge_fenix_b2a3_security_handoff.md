---
doc_type: task
id: DB-FENIX-B2.A3.T4
title: "B2.A3.T4 independent security verification and B2.A4 handoff"
status: planned_blocked_by_t3
phase: B2.A3
week: 2026-W46
tags: [dubbridge, fenix, s2s, security, audit, handoff]
fr_refs: []
uc_refs: []
blocked_by: [DB-FENIX-B2.A3.T3]
blocks: [DB-FENIX-B2.A4]
files_affected:
  - docs/audit/dubbridge-fenix-b2a3-security-verification-2026-11-09.md
  - docs/plans/dubbridge_fenix_s2s_security_plan.md
  - docs/tasks/task_dubbridge_fenix_b2a3_security_handoff.md
criticality: critical
criticality_basis: "Security sign-off and handoff must not overstate authorization coverage, production readiness, or the absence of a real persisted B2.C operation resolver."
created: 2026-11-09
completed: null
---

# B2.A3.T4 independent security verification and B2.A4 handoff

**Plan**: [DubBridge to Fenix S2S security foundation](../plans/dubbridge_fenix_s2s_security_plan.md#current-direction).
**Dependencies**: [T2 source guard](task_dubbridge_fenix_b2a3_mandatory_scope_guard.md), [T3 real SQLite HTTP tests](task_dubbridge_fenix_b2a3_sqlite_http_integration.md), [QA debt #4](https://github.com/krukmat/fenix/issues/4).
**Current status**: PLANNED. Do not begin security acceptance or close B2.A3 while T3 tests and independent evidence are pending.
**RRI**: N/A (cloud, DB-FENIX-INT). Independent security reviewer and owner HITL for any actual security changes remain mandatory.
**Owner of task**: ChatGPT Cloud (audit, evidence reconciliation, risk ledger, documented handoff). Local agent only executes verification or reviewer commands that require the Mac environment. Owner retains production and security acceptance authority.

## Scope

1. Review T1 readiness, T2 committed implementation and exact-diff Claude code-review PASS, and T3 actual Fenix SQLite integration test matrix and corresponding independent peer review. Inspect *real* branch HEAD and affected files, not issue comments alone.
2. Reconcile security contracts: strict opaque service token, live revoke/expiry/rotation, Fenix-owned identity and org/project scope, human JWT isolation, no handler on denial, bounded transport, sanitized errors, zero operation/agent-run side effects on rejected calls, default-deny capabilities and unmounted production route.
3. Publish an evidence matrix with for each control: claim, test name, exact command, artifact SHA, source commit, reviewer/verdict, gap or deferred owner, and validation confidence. Never turn a T2 stub GET/status test into a claim of persisted B2.C operation lookup.
4. Clearly separate (a) feature-branch isolated security verification, (b) remaining full-repository QA debt #4, (c) B2.C persisted scoped operation store, (d) A4 operator registration, audit, trusted-proxy and principal rate-limit questions, and (e) B5 rollout/security acceptance.
5. Produce a concise A4 handoff: operator-only service principal register/rotate/revoke, audited credential distribution without plaintext persistence, monitor/runbook and error projection, principal-bound limits, migration/restore evidence, no production route activation.
6. Obtain genuine independent security assessment for any material security conclusion; record findings by severity, disposition and links. Document owner action required. The local independent reviewer provides findings; ChatGPT Cloud owns reconciliation and audit report.

## Required output

- `docs/audit/dubbridge-fenix-b2a3-security-verification-2026-11-09.md`, `doc_type: audit`, source-backed. If the report is written on a later day, correct the filename and `date` to the actual review date.
- Updated parent plan status and issue links, without changing ADR `PROPOSED` into `ACCEPTED` or claiming production readiness.
- A4 prerequisites/dependency table with owners and exact pending gates.
- Security review: PASS/FAIL/BLOCKED and real artifact reference; no self-sign-off. If T3 is blocked, mark T4 BLOCKED and provide a handoff rather than a fictitious PASS.

## Out of scope

No production code, credential issuance, table migration, public S2S route, real capability onboarding, CI/workflows, operational deployment or main merge. Correcting global QA failures (#4) is its own authorized task. B2.C resolver implementation is independent.

## Readiness and closure

Before beginning T4 execution, obtain independent `task-readiness` PASS on this record and the S2S plan, with reviewer concurrence with `criticality: critical`; T3 must already have actual QA/peer evidence. Complete the closure audit only after its own review and acceptance. Do not reuse a prior task's readiness verdict.

## Task Card (requires peer readiness PASS before presentation)

Task: DB-FENIX-B2.A3.T4
Task file: docs/tasks/task_dubbridge_fenix_b2a3_security_handoff.md
Plan file: docs/plans/dubbridge_fenix_s2s_security_plan.md
Summary: Audit B2.A3 security evidence after real SQLite integration tests and independent reviewer; document remaining B2.C/A4/B5 and QA prerequisites.
Code affected: none; docs/audit, docs/plans and docs/tasks only.
Criticality: critical
Criticality basis: A security-completion statement may misrepresent tenant isolation and rollout readiness.
Effort/reasoning: Medium - Evidence reconciliation and independent security review.
Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6
Estimated tokens: ~5000
Peer readiness review approval: reviewer=PENDING; artifact=PENDING; status=BLOCKED_UNTIL_ACTUAL_REVIEW
