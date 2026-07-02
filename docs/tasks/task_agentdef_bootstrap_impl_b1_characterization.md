---
doc_type: task
id: AGENTDEF-BOOTSTRAP-IMPL-B1-001
title: "Characterization tests for triggerSupportRun's current AgentID lookup behavior"
status: completed
phase: remediation
week: "2026-W27"
tags: [agent, bootstrap, testing, adr-032-followup]
fr_refs: [FR-230]
uc_refs: [UC-C1]
blocked_by: [AGENTDEF-BOOTSTRAP-DESIGN-001]
blocks: [AGENTDEF-BOOTSTRAP-IMPL-B2-001]
files_affected:
  - internal/domain/agent/agents/support_test.go
created: 2026-07-02
completed: 2026-07-02
---

# Task AGENTDEF-BOOTSTRAP-IMPL-B1-001

**Plan**: [Support-agent definition bootstrap remediation](../plans/agent-definition-bootstrap-remediation-plan.md)

## Task Card

Task: AGENTDEF-BOOTSTRAP-IMPL-B1-001

Task file: docs/tasks/task_agentdef_bootstrap_impl_b1_characterization.md

Plan file: docs/plans/agent-definition-bootstrap-remediation-plan.md

Summary: Write characterization tests that pin down `triggerSupportRun`'s current behavior (`internal/domain/agent/agents/support.go:499-518`) before it is modified in `AGENTDEF-BOOTSTRAP-IMPL-B2-001`. This path currently has no dedicated test coverage of the hardcoded-`AgentID` lookup itself — only the fixture helper `insertSupportAgentDefinition` that manually inserts a row matching the literal id, which masks the collision bug rather than testing for it. Required by `docs/policies/RRI_POLICY.md` decomposition trigger "T≥4 and P≥4: first subtask must be characterization tests."

Code affected: `internal/domain/agent/agents/support_test.go` only. No production code changes.

Effort/reasoning: Medio - test-only change on the only P0 end-to-end agent path (UC-C1), so no test currently exists that would catch a regression in the trigger/lookup step. RRI=32 (Moderate band).

Recommended model: claude-sonnet-4-6

Estimated tokens: ~4000

## High-Level Pseudocode

```
// internal/domain/agent/agents/support_test.go

test "triggerSupportRun succeeds when agent_definition row exists with literal id":
    setup db, insertSupportAgentDefinition(db, workspaceID)  // existing helper
    call triggerSupportRun
    assert run created, no error
    // pins down: today's only working path

test "triggerSupportRun fails when agent_definition row is missing":
    setup db, do NOT insert any agent_definition row
    call triggerSupportRun
    assert error (whatever the current error actually is — record it exactly,
                   do not guess; this is what B2 must preserve or deliberately change)

test "triggerSupportRun fails when agent_definition exists for a DIFFERENT workspace
      with the same literal id 'support-agent'":
    setup db, insertSupportAgentDefinition(db, workspaceA)
    call triggerSupportRun with workspaceB
    assert error (pins down that cross-workspace lookup already fails today —
                   this is the collision symptom B2 is fixing, captured as a
                   test BEFORE the fix so the fix's test diff is reviewable)

test "two workspaces cannot both bootstrap an agent_definition row with the
      same literal id 'support-agent' due to the global PRIMARY KEY":
    setup db, insertSupportAgentDefinition(db, workspaceA, id="support-agent")
    attempt insertSupportAgentDefinition(db, workspaceB, id="support-agent")
    assert PRIMARY KEY constraint violation
    // pins down the exact collision bug described in the design task's Risks
    // section, as an executable regression test rather than only prose
```

## Acceptance Criteria

1. A test exists that pins down `triggerSupportRun`'s current successful path (literal id `"support-agent"`, single workspace, row present).
2. A test exists that pins down the current failure mode when no `agent_definition` row exists for the workspace, and records the exact error type/message produced today (not an assumed one).
3. A test exists that demonstrates the cross-workspace collision symptom: a second workspace cannot successfully trigger the support agent today because the lookup is scoped by a literal id shared across all workspaces.
4. A test exists that demonstrates the `PRIMARY KEY` constraint violation when two workspaces attempt to insert an `agent_definition` row with the same literal `id`.
5. All four tests pass against the codebase as it exists today, unmodified — this task adds tests only, it does not change `support.go`.
6. No production code changes. No schema migration.

## Scope

- In: `internal/domain/agent/agents/support_test.go` additions only.
- Out: any change to `internal/domain/agent/agents/support.go` (that is `AGENTDEF-BOOTSTRAP-IMPL-B2-001`, which depends on these tests existing first so the behavior-change diff in B2 is reviewable against a known-good baseline).
- Out: commits, pushes.

## Risks

- If these characterization tests are written loosely (e.g. asserting only "an error occurs" instead of the exact error type), they won't actually catch a regression in B2's replacement logic — be specific about what error type/value each test asserts.
- This task intentionally does not fix anything; if read out of context, "adds tests for a known bug without fixing it" could look like an incomplete task. It is not — it is the first of two required subtasks per the RRI decomposition rule for `T≥4 ∧ P≥4`, and `AGENTDEF-BOOTSTRAP-IMPL-B2-001` is the fix that depends on it.

## Verification Strategy

- `go test ./internal/domain/agent/agents/... -run TestSupportAgent -v` (or equivalent scoped run covering the new tests)
- `go test ./internal/domain/agent/...` full package run to confirm no regressions.
