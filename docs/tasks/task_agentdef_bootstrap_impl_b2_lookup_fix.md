---
doc_type: task
id: AGENTDEF-BOOTSTRAP-IMPL-B2-001
title: "Fix triggerSupportRun to resolve AgentID by workspace instead of a hardcoded literal"
status: completed
phase: remediation
week: "2026-W27"
tags: [agent, bootstrap, auth, onboarding, adr-032-followup]
fr_refs: [FR-230]
uc_refs: [UC-C1]
blocked_by: [AGENTDEF-BOOTSTRAP-IMPL-A-001, AGENTDEF-BOOTSTRAP-IMPL-B1-001]
blocks: [AGENTDEF-BOOTSTRAP-DOCS-001]
files_affected:
  - internal/domain/agent/agents/support.go
  - internal/domain/agent/agents/support_test.go
  - internal/domain/agent/orchestrator.go
  - internal/api/handlers/agent.go
  - internal/api/handlers/agent_test.go
created: 2026-07-02
completed: 2026-07-02
---

# Task AGENTDEF-BOOTSTRAP-IMPL-B2-001

**Plan**: [Support-agent definition bootstrap remediation](../plans/agent-definition-bootstrap-remediation-plan.md)

## Task Card

Task: AGENTDEF-BOOTSTRAP-IMPL-B2-001

Task file: docs/tasks/task_agentdef_bootstrap_impl_b2_lookup_fix.md

Plan file: docs/plans/agent-definition-bootstrap-remediation-plan.md

Summary: Replace the hardcoded literal `AgentID: "support-agent"` in `triggerSupportRun` (`internal/domain/agent/agents/support.go:508`) with a lookup of the workspace's own `support-agent` definition via `ListAgentDefinitionsByType(workspace_id, "support")`, using the resolved row's `id`. This closes the id-collision bug documented in `AGENTDEF-BOOTSTRAP-DESIGN-001` and makes the bootstrap row created in `AGENTDEF-BOOTSTRAP-IMPL-A-001` actually usable per-workspace. Depends on `AGENTDEF-BOOTSTRAP-IMPL-B1-001`'s characterization tests as the baseline this change is reviewed against.

Code affected: `internal/domain/agent/agents/support.go`, `internal/domain/agent/agents/support_test.go`.

Effort/reasoning: Medio-alto - cambia el comportamiento del único camino de trigger del agente P0 (UC-C1), pero ahora apoyado en tests de caracterización previos (B1) que acotan el riesgo de regresión. RRI=45 (Med-high band) tras la decomposición.

Recommended model: claude-opus-4-8

Estimated tokens: ~6000

## High-Level Pseudocode

```
// internal/domain/agent/agents/support.go

type SupportAgent struct {
    orchestrator *agent.Orchestrator
    // NEW: needs a way to resolve agent_definition rows by workspace+type.
    // Simplest: reuse whatever DB handle/queries the orchestrator already
    // holds, or accept a narrow interface, e.g.:
    definitionLookup SupportAgentDefinitionLookup   // new small interface
    ...existing fields...
}

interface SupportAgentDefinitionLookup:
    ListByType(ctx, workspaceID, agentType) ([]AgentDefinition, error)

func (a *SupportAgent) triggerSupportRun(ctx, config) (*agent.Run, error):
    defs, err := a.definitionLookup.ListByType(ctx, config.WorkspaceID, "support")
    if err != nil:
        return nil, fmt.Errorf("resolve support agent definition: %w", err)
    if len(defs) == 0:
        return nil, ErrSupportAgentNotProvisioned   // new sentinel, distinct
                                                       // from orchestrator's
                                                       // ErrAgentNotFound so
                                                       // callers can tell
                                                       // "not bootstrapped"
                                                       // apart from "bad id"
    resolvedID := defs[0].ID

    triggerContext, inputs := supportRunPayloads(config, a.AllowedTools())
    triggeredBy := supportUserID(ctx)
    ...unchanged...
    run, err := a.orchestrator.TriggerAgent(ctx, agent.TriggerAgentInput{
        AgentID:        resolvedID,   // CHANGED: was literal "support-agent"
        WorkspaceID:    config.WorkspaceID,
        ...unchanged fields...
    })
    return run, err
```

## Acceptance Criteria

1. `triggerSupportRun` no longer contains the literal string `"support-agent"` as an `AgentID` value; it resolves the id via a workspace-scoped lookup.
2. All four characterization tests from `AGENTDEF-BOOTSTRAP-IMPL-B1-001` are updated to reflect the new (fixed) behavior where applicable — specifically, the cross-workspace collision test must now show the second workspace's own row is found and used, not an error.
3. A fresh workspace bootstrapped by `AGENTDEF-BOOTSTRAP-IMPL-A-001` can call `POST /agents/support/trigger` and reach `orchestrator.TriggerAgent` successfully, with no manual `agent_definition` insert — this is the end-to-end proof that A + B1 + B2 together close the original gap.
4. A workspace with no `agent_definition` row (e.g. one created before this fix shipped, matching the legacy-workspace exclusion already established by ADR-032) fails with a clear, distinguishable error (`ErrSupportAgentNotProvisioned` or equivalent) rather than the current generic `500`.
5. `internal/domain/agent/agents/support_test.go`'s `insertSupportAgentDefinition` helper is either removed (if no longer needed once real per-workspace rows are used in tests) or explicitly repurposed with a comment/test name making clear it now simulates "a workspace that predates this fix" — not left as accidental dead code.
6. `prospecting.go`, `kb.go`, `insights.go`, `deal_risk.go` remain untouched — same hardcoded pattern, explicitly out of scope per the design decision record.
7. No schema migration, no new REST route, no legacy-workspace backfill.

## Scope

- In: the lookup fix in `support.go` and its direct test updates.
- Out: `prospecting.go`, `kb.go`, `insights.go`, `deal_risk.go` (identical pattern, flagged for a future P1 task, not fixed here).
- Out: any Agent Studio CRUD surface.
- Out: commits, pushes.

## Risks

- This is the highest-risk subtask in the decomposition: it changes behavior on the only P0 agent's trigger path. Mitigated by requiring B1's characterization tests as a prerequisite baseline (blocked_by), so the diff is reviewable against known-good current behavior rather than reasoned about from scratch.
- If `ListAgentDefinitionsByType` ever returns more than one `support`-typed definition for a workspace (not possible today given `UNIQUE(workspace_id, name)` and bootstrap only creating one, but not structurally prevented at the `agent_type` level), silently picking `defs[0]` could mask a data problem — worth an explicit comment or guard if the implementer judges it warranted, not mandatory for this P0 fix.

## Verification Strategy

- `go test ./internal/domain/agent/agents/...`
- `go test ./internal/domain/agent/...`
- `go test ./internal/domain/auth/...` (confirm no interaction with A's bootstrap change)
- Full local QA gate before any push: `scripts/qa-go-prepush.sh`.
