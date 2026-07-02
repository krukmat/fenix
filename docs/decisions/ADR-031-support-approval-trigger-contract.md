---
id: ADR-031
title: "Support approval trigger contract: approval gates sensitive case mutation, handoff remains a human-routing fallback"
date: 2026-07-01
status: accepted
deciders: [matias]
tags: [adr, support, approval, handoff, governance, runtime]
related_tasks: [EXTVAL-DESIGN-APPROVAL-CONTRACT-001, EXTVAL-BUG-APPROVAL-UNREACHABLE-001]
related_frs: [FR-070, FR-071, FR-232]
---

# ADR-031 — Support approval trigger contract: approval gates sensitive case mutation, handoff remains a human-routing fallback

## Status

`accepted`

## Context

External validation found a P0 bug: no API-reachable support flow currently produces a
real `ApprovalRequest`. The support path computes approval eligibility from
`action.Metadata`, but the producer writes a plain string while the consumer expects a JSON
object, making the approval branch unreachable in practice.

At the same time, the real escalation path, `InitiateHandoff`, packages evidence and marks
the case escalated without creating an approval request.

The product docs already establish two adjacent but distinct governance promises:

- approvals gate risky or sensitive actions (`FR-070`, `FR-071`, CLAUDE.md tool flow),
- human handoff preserves context when confidence is low or when approval is required
  (`FR-232`, strategic repositioning spec).

Before product-code changes begin, the runtime needs a canonical contract for which support
outcomes create approvals and which outcomes are direct human-routing fallbacks.

## Decision

**1. Approval is attached to sensitive support case mutation, not to handoff itself.**

The canonical approval gate for the current support flow is the proposed support case
mutation (`support.case.update`). If the runtime determines that the proposed case update is
sensitive, it must create an `ApprovalRequest` before the mutation executes.

`InitiateHandoff` is not itself an approval-triggering action in this contract. Handoff is
the fallback packaging step that transfers evidence and rationale to a human when the agent
abstains, lacks confidence, or cannot proceed after governance checks.

**2. Approval eligibility must be driven by explicit structured action metadata.**

The producer and consumer of the approval signal must share a structured contract. Free-form
strings are not sufficient.

For the current support path, the minimum canonical metadata shape is:

```json
{
  "sensitivity": "high",
  "approval_reason": "high_sensitivity",
  "source": "support-agent"
}
```

Equivalent future representations are allowed only if both sides are migrated together and
the contract remains explicit and testable. Malformed or missing approval metadata must fail
closed: no mutation executes, and the runtime must take the non-approval path defined by the
flow rather than silently treating malformed input as approved.

**3. Approver resolution is a separate concern from trigger reachability.**

The identity of the approver is governed separately from whether an approval is required.
The current self-approval defect (`ApproverID == RequestedBy`) must be fixed, but not in the
same decision as the trigger contract. A follow-up implementation task will define and test
approver resolution explicitly.

**4. Handoff remains the governed fallback when confidence is low or execution cannot proceed.**

If the agent lacks confidence, cannot satisfy the approval contract, or must defer after a
governance check, the runtime may hand off to a human with evidence and rationale. This
preserves the FR-232 promise without redefining every handoff as an approval workflow.

**5. Any future requirement to approve handoff itself needs a separate decision.**

If the product later decides that some class of escalations must themselves be approved, that
is a new governance expansion and must be planned as a separate task/ADR. It is not part of
the reachability fix for the current bug.

## Consequences

- The next implementation task may focus on restoring one real API-reachable approval path
  for sensitive support case mutation without redesigning handoff.
- `actionRequiresApproval` and its producer must share an explicit, testable metadata
  contract.
- `InitiateHandoff` may remain approval-free in the current fix as long as it continues to
  preserve evidence, rationale, and auditability.
- Approver resolution remains blocked on a separate follow-up task; no implementation may
  preserve self-approval as an acceptable steady state.

## Alternatives considered

**A. Make every handoff create an approval request (rejected)**

This would conflate two different product promises: approval of privileged mutation and
human-routing fallback. It would also expand scope beyond the current reachability defect and
force new governance/UI behavior before the existing support approval path even works.

**B. Keep approval eligibility on opaque string metadata (rejected)**

The current bug shows that free-form strings are too weak for a governance boundary. The
contract must be structured and testable.

**C. Combine trigger reachability and approver resolution into one decision (rejected)**

This would enlarge scope and make it harder to keep the first implementation task inside the
target RRI band.
