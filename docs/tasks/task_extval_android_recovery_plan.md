---
doc_type: task
id: EXTVAL-ANDROID-RECOVERY-001
title: "Analyze the broken Android AVD and document the T7 real-mode recovery plan"
status: complete
phase: external-validation-rerun
week: "2026-W27"
tags: [external-validation, android, avd, emulator, t7, planning]
fr_refs: [FR-230, FR-232]
uc_refs: [UC-C1]
blocked_by: []
blocks: []
files_affected:
  - docs/plans/external_validation_android_recovery_plan.md
criticality: standard
criticality_basis: "Docs-only analysis and planning task. No product-code, permission, or data mutation."
created: 2026-07-04
completed: 2026-07-04
---

# Task EXTVAL-ANDROID-RECOVERY-001

**Plan**: [External Validation Android Recovery and T7 Revalidation Plan](../plans/external_validation_android_recovery_plan.md)

## Task Card

Task: EXTVAL-ANDROID-RECOVERY-001

Task file: docs/tasks/task_extval_android_recovery_plan.md

Plan file: docs/plans/external_validation_android_recovery_plan.md

Summary: Analyze why the local Android rerun is currently blocked after the mobile approvals crash fix, separate environment failure from product failure, and document the shortest recovery path to resume real-mode T7 validation.

Code affected: docs/plans/external_validation_android_recovery_plan.md, docs/tasks/task_extval_android_recovery_plan.md

Criticality: standard

Criticality basis: Docs-only analysis and planning task. No product-code, permission, or data mutation.

Effort/reasoning: Medium - requires reconciling the prior T7 execution, the crash-fix task outcome, current SDK/AVD state, and the rerun dependencies into one actionable recovery document.

Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6

Estimated tokens: ~6000

Peer readiness review approval: reviewer=local-gemma; artifact=logs/peer-workflow-review/task-readiness_codex_by_local-gemma_20260704T201427Z.json; status=PASS

Task type: docs-only planning task. No dev-task pseudocode required.

## Acceptance Criteria

1. The new document clearly distinguishes product-fix status from environment-blocker status.
2. The new document identifies the current AVD/system-image failure mode with concrete evidence.
3. The new document defines a phased recovery plan for emulator repair and T7 rerun.
4. The new document defines explicit go/no-go branches based on emulator and rerun outcomes.
