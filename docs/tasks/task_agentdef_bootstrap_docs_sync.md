---
doc_type: task
id: AGENTDEF-BOOTSTRAP-DOCS-001
title: "Sync validation and governance artifacts after agent-definition bootstrap fix"
status: completed
phase: remediation
week: "2026-W27"
tags: [agent, bootstrap, docs, adr-032-followup]
fr_refs: [FR-230]
uc_refs: [UC-C1]
blocked_by: [AGENTDEF-BOOTSTRAP-IMPL-B2-001]
blocks: []
files_affected:
  - docs/tasks/task_extval_battery_t3_support_agent.md
  - docs/plans/adr-032-workspace-bootstrap-remediation-plan.md
created: 2026-07-02
completed: 2026-07-02
---

# Task AGENTDEF-BOOTSTRAP-DOCS-001

**Plan**: [Support-agent definition bootstrap remediation](../plans/agent-definition-bootstrap-remediation-plan.md#w4-sync-validation-and-governance-artifacts)

## Task Card

Task: AGENTDEF-BOOTSTRAP-DOCS-001

Task file: docs/tasks/task_agentdef_bootstrap_docs_sync.md

Plan file: docs/plans/agent-definition-bootstrap-remediation-plan.md

Summary: Implement plan section W4. Update `docs/tasks/task_extval_battery_t3_support_agent.md` Finding 1 to record that `agent_definition` provisioning is now resolved for newly registered workspaces (mirroring how `ADR032-BOOTSTRAP-DOCS-001` updated Finding 2), and update `docs/plans/adr-032-workspace-bootstrap-remediation-plan.md:180` so its cross-reference note no longer describes the agent-definition bootstrap gap as unresolved.

Code affected: `docs/tasks/task_extval_battery_t3_support_agent.md`, `docs/plans/adr-032-workspace-bootstrap-remediation-plan.md`. Docs-only, no source files.

Effort/reasoning: Bajo - docs-only sync task, no code changes, no test surface. RRI=14 (Low band, computed via `scripts/rri.py --C 0 --T 1 --A 1 --X 1 --D 1 --K 1 --P 1`).

Recommended model: claude-sonnet-4-6

Estimated tokens: ~2500

## High-Level Pseudocode

```
// docs/tasks/task_extval_battery_t3_support_agent.md

Finding 1 section:
    append a "Current status after AGENTDEF-BOOTSTRAP-IMPL-B2-001" note,
    parallel in structure to the existing "Current status after ADR-032" note
    already present in Finding 1:
        - record that AGENTDEF-BOOTSTRAP-IMPL-A-001 seeds a per-workspace
          agent_definition row at Register time
        - record that AGENTDEF-BOOTSTRAP-IMPL-B2-001 fixed the literal-id
          lookup bug this finding called out as a blocker
          ("cannot be safely bootstrapped per-workspace without also fixing
          that lookup")
        - state plainly: resolved for newly registered workspaces; existing
          workspaces are not backfilled (same caveat pattern as Finding 2)

// docs/plans/adr-032-workspace-bootstrap-remediation-plan.md:180

replace the "add a short follow-up note if agent-definition bootstrap is
still unresolved" bullet (which is itself the follow-up-note instruction,
not the note) with a resolved-status line pointing at
agent-definition-bootstrap-remediation-plan.md, consistent with how W4's
own Status line already reads "completed by ADR032-BOOTSTRAP-DOCS-001 on
2026-07-02"
```

## Acceptance Criteria

1. `task_extval_battery_t3_support_agent.md` Finding 1 clearly states the provisioning gap is resolved for newly registered workspaces, with a pointer to `AGENTDEF-BOOTSTRAP-IMPL-A-001` and `AGENTDEF-BOOTSTRAP-IMPL-B2-001`.
2. The update explicitly notes that existing/legacy workspaces are not backfilled, so the finding does not overstate resolution scope.
3. `adr-032-workspace-bootstrap-remediation-plan.md`'s W4 cross-reference no longer reads as if the agent-definition gap is still open.
4. No source code files touched.
5. No new doc_type introduced — both target files already have their own frontmatter/structure; this task only edits body content.

## Scope

- In: the two files listed in `files_affected`.
- Out: `docs/decisions/` ADR creation (plan W4 marks this optional and not required for this decision).
- Out: any code change, any test.
- Out: commits, pushes.

## Risks

- Low risk: docs-only edit with no runtime surface. Main failure mode is prose drift/inaccuracy — mitigated by citing exact task IDs and file:line locations already established in the B1/B2 closing reports rather than re-describing the fix from memory.

## Verification Strategy

- Manual review: re-read both edited sections for consistency with the actual B2 implementation (`internal/domain/agent/agents/support.go`, `internal/domain/agent/orchestrator.go`) before closing.
- No automated tests apply (docs-only).
