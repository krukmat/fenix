---
id: ADR-028
title: "Dual approval seed for snapshot runner approve/reject coverage"
date: 2026-04-26
status: accepted
deciders: [matias]
tags: [adr, testing, snapshots, bff, governance]
related_tasks: [bff-http-snapshots]
related_frs: [FR-070]
---

# ADR-028 — Dual approval seed for snapshot runner approve/reject coverage

## Status

`accepted`

## Context

The BFF HTTP snapshot runner (`npm run http-snapshots`) executes catalog entries
sequentially against a live backend. Two entries exercise the approval workflow:

- `approvals-approve` — POST `/bff/api/v1/approvals/:id/approve`
- `approvals-reject`  — POST `/bff/api/v1/approvals/:id/reject`

An `approval_request` record has a finite state machine with terminal states:

```
pending → approved  (terminal, no further transitions allowed)
pending → rejected  (terminal, no further transitions allowed)
```

When both entries share the same `approvalId`, the approve entry runs first and
transitions the record to `approved`. The reject entry then receives **409 Conflict**
("approval request is already decided") because the record is no longer `pending`.

This means it is impossible to test both the approve and reject happy paths in the
same sequential run using a single approval ID.

## Decision

The seeder (`scripts/e2e_seed_mobile_p2.go`) creates **two independent pending
approvals** per run and exposes both in `seedOutput.Inbox`:

```json
{
  "inbox": {
    "approvalId":       "<uuid — consumed by approvals-approve>",
    "rejectApprovalId": "<uuid — consumed by approvals-reject>",
    "signalId":         "<uuid>"
  }
}
```

The catalog binds each entry to its dedicated ID:
- `approvals-approve` → `seed.inbox.approvalId`
- `approvals-reject`  → `seed.inbox.rejectApprovalId`

Both entries expect `204 No Content` on success.

## Rationale

Each entry needs a `pending` approval at execution time. The only way to guarantee
that without adding runner complexity (re-seed between entries, parallel workspaces,
rollback) is to provision independent records at seed time. The seeder already
created two approvals in `seedInboxApprovals` — the change only exposes the second
ID in the output struct, which is a minimal diff with no new SQL.

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Single ID, `expectedStatus: 409` for reject | Loses coverage of the reject happy path entirely — the snapshot would document a conflict, not a successful rejection |
| Re-seed between entries in the runner | Adds stateful orchestration logic to the runner, violating the catalog-is-data principle; complicates parallelism if ever introduced |
| Reverse execution order (reject first, then approve) | Fixes reject but breaks approve for the same reason; order dependence is fragile and hard to reason about |
| Separate workspace per approval entry | Over-engineered; workspaces carry cost (seeder complexity, cleanup) for a problem solvable with a single extra INSERT |

## Consequences

**Positive:**
- Both approve and reject happy paths produce `204` artifacts and are marked ✅ in the report
- No runner logic change — the catalog is the only consumer of the new field
- The seeder already inserted two approvals; this just surfaces the second ID

**Negative / tradeoffs:**
- `SeederOutput` type grows by one field (`rejectApprovalId`) — any consumer of the seeder JSON output must handle the new field (currently only the BFF snapshot runner)
- The two approvals have different actions (`close_case` vs `send_external_email`), so the snapshots exercise slightly different approval payloads. This is acceptable for contract coverage but means the two entries are not fully symmetric

## Future evaluation trigger

Re-evaluate this decision if:
- The snapshot runner gains **parallel execution** — with parallelism, both entries
  could race on the same workspace state, and the two-ID approach may need to
  become N-IDs or workspace-isolated fixtures.
- The approval FSM gains a **reset or reopen** transition — if `pending` can be
  restored, a single ID with an intermediate reset step would suffice and this
  workaround could be removed.
- The seeder is replaced by a **declarative fixture system** — in that case the
  dual-seed pattern should be expressed as two fixture declarations rather than
  a hardcoded second INSERT.

## References

- `scripts/e2e_seed_mobile_p2.go` — `seedInboxApprovals`, `seedGovernanceAndApproval`, `buildSeedOutput`
- `bff/scripts/snapshots/catalog.ts` — `approvals-approve`, `approvals-reject` entries
- `bff/scripts/snapshots/types.ts` — `SeederOutput.inbox`
- `docs/plans/bff-http-snapshots-plan.md` — T9 catalog fix notes
