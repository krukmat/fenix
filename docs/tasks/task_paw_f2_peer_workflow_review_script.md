---
doc_type: task
id: PAW-F2
title: "Implement peer workflow review script, adapters, and tests"
status: done
phase: F
week: ""
tags: [paw, devex, workflow, peer-review, python, testing]
fr_refs: []
uc_refs: []
blocked_by: [PAW-F1]
blocks: [PAW-F4]
files_affected:
  - scripts/peer-workflow-review.py
  - scripts/peer_workflow_review_test.py
  - docs/tasks/task_paw_f2_peer_workflow_review_script.md
created: 2026-07-01
completed: "2026-07-01"
rri: 42
rri_band: Med-high
hp: "Claude Code caller with valid task and plan resolves to Codex, receives a pass verdict, writes a pass artifact, and exits 0"
ec: "Reviewer CLI unavailable writes blocked artifact and exits nonzero; invalid JSON verdict writes blocked artifact; non-pass verdict blocks presentation or closure"
coverage_cert: ""
---

# Task PAW-F2

**Plan**: [Portable Agent Workflow Port Plan](../plans/portable_agent_workflow_port_plan.md#8-task-decomposition)

## Task Card

Task: PAW-F2
Task file: docs/tasks/task_paw_f2_peer_workflow_review_script.md
Plan file: docs/plans/portable_agent_workflow_port_plan.md
Summary: Implement the provider-aware peer workflow review script with Claude and Codex adapters plus mocked tests. The script must support task-readiness and post-code-review modes, write local review artifacts, and fail closed on unavailable peers or non-pass verdicts.
Code affected: scripts/peer-workflow-review.py, scripts/peer_workflow_review_test.py, and this task file.
Effort/reasoning: High - Adds a new workflow gate with external CLI adapters, strict verdict parsing, and failure-mode handling.
Recommended model: claude-opus-4-8
Estimated tokens: ~9000
Pseudocode: Parse command and caller metadata; resolve reviewer from caller kind; build redacted review packet from task, plan, optional task-card preview, diff, and verification log; invoke Claude or Codex adapter in read-only/non-mutating mode; parse strict JSON verdict; write artifact under logs/peer-workflow-review; exit 0 only when status is pass.

## Summary

Build the script that enforces Phase F peer review. The script is the single local interface used by future workflow steps to request either a pre-task readiness review or a post-code task review from an independent provider.

## Acceptance Criteria

1. `scripts/peer-workflow-review.py task-readiness` accepts caller metadata, task path, plan path, and task-card preview path.
2. `scripts/peer-workflow-review.py post-code-review` accepts caller metadata, task path, plan path, base ref, and verification log path.
3. Reviewer resolution follows the Phase F contract: `claude-code -> codex`, `codex -> claude`, `local-provider -> claude`, `remote-provider -> claude`, `unknown -> claude`.
4. Claude adapter invokes Claude Code in non-mutating review mode and requests a strict JSON verdict.
5. Codex readiness adapter invokes `codex exec` with read-only sandboxing and a strict output schema.
6. Codex post-code adapter invokes `codex review` against the requested base ref with custom instructions.
7. The script writes redacted artifacts under `logs/peer-workflow-review/`.
8. The script exits 0 only for `status: pass`; `needs_changes`, `out_of_scope`, `blocked`, invalid JSON, unavailable CLI, and timeouts exit nonzero.
9. Tests mock all external CLI calls and cover pass, non-pass, invalid JSON, unavailable CLI, artifact writing, reviewer resolution, and secret redaction.
10. No hooks, Makefile targets, CI jobs, or product code are changed in this task.

## Scope

- **In**: Python script, mocked unit tests, strict verdict parsing, local artifact writing, secret redaction, and adapter command construction.
- **Out**: No Makefile wiring, no hook enforcement, no CI job, no edits to product code, no live Claude/Codex calls in tests.

## Risks

- CLI output formats can drift. Keep adapters behind small functions and make tests assert command construction and parser behavior separately.
- The script must never mutate repo state. Use read-only/sandboxed adapter flags where available and avoid accepting reviewer-generated patches.
- Secrets can leak into review packets. Redact token, key, password, secret, and credential-like values before writing artifacts or sending prompts.

## High-Level Pseudocode

```
main(argv):
  args = parse_args(argv)
  caller = normalize_caller(args.caller_kind, args.caller_provider)
  reviewer = resolve_reviewer(caller.kind)
  packet = build_packet(args, caller, reviewer)
  packet = redact_secrets(packet)

  try:
    raw = invoke_reviewer(reviewer, args.mode, packet)
    verdict = parse_verdict(raw)
  except ReviewerUnavailable as err:
    verdict = blocked_verdict("reviewer_unavailable", err)
  except InvalidVerdict as err:
    verdict = blocked_verdict("invalid_verdict", err)

  artifact = write_artifact(args.mode, caller, reviewer, packet, verdict)
  print_summary(artifact, verdict)
  return 0 if verdict.status == "pass" else 1

resolve_reviewer(kind):
  if kind == "claude-code": return "codex"
  if kind == "codex": return "claude"
  return "claude"

invoke_reviewer(reviewer, mode, packet):
  if reviewer == "claude":
    return run_claude_print(packet)
  if reviewer == "codex" and mode == "task-readiness":
    return run_codex_exec_readonly(packet)
  if reviewer == "codex" and mode == "post-code-review":
    return run_codex_review(packet)
```
