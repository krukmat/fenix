---
doc_type: task
id: FENIX-GO-PREPUSH-UNBLOCK
title: "Unblock Go pre-push QA for the DubBridge integration branch"
status: completed
phase: B2.A3
week: 2026-W46
tags: [fenix, dubbridge, go, qa, pre-push]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: [DB-FENIX-B2.A3.T3]
files_affected:
  - docs/plans/fenix_go_prepush_unblock_plan.md
  - docs/tasks/task_fenix_go_prepush_unblock.md
  - internal/api/integration_runtime.go
  - internal/api/integration_runtime_test.go
  - internal/api/middleware/auth.go
  - internal/api/middleware/dubbridge_service_auth.go
  - internal/api/middleware/dubbridge_source_guard.go
  - internal/api/middleware/ratelimit.go
  - internal/domain/
  - internal/infra/integration/
criticality: critical
criticality_basis: "The current full gate includes auth-boundary complexity and lint findings; any remediation there can affect tenant isolation. The task also includes repository-wide formatting and error propagation."
created: 2026-11-09
completed: 2026-11-09
---

# Unblock Go pre-push QA for the DubBridge integration branch

**Plan**: [Go pre-push unblock](../plans/fenix_go_prepush_unblock_plan.md#current-direction).
**Issues**: [QA debt #4](https://github.com/krukmat/fenix/issues/4), [T3 #7](https://github.com/krukmat/fenix/issues/7).
**Branch**: `feature/int-v2-dubbridge-consumes-fenix`.
**Initial local commit**: `367a70c3f297a3d2ca5af6fd18e77e50dae98918`.
**Mechanical formatting commit**: `43d935c` (26 files; each matches `gofmt` of its parent revision byte for byte).
**RRI**: N/A for the DB-FENIX-INT cloud-agent workstream; criticality is developer judgment.

## Scope and acceptance

Clear the actual local Go pre-push gates needed to publish the T3 formatting commit. Preserve the unrelated local `.gitignore` modification. Formatting is limited to the exact files reported by `make fmt-check`. Subsequent code fixes must be minimal, behavior preserving, and separately committed. Record baseline and final evidence for complexity, lint, wrapcheck, tests, coverage, race, traceability, vulnerability and pattern gates as they apply. Do not weaken QA rules, alter CI, mount S2S routes, provision credentials, migrate production, deploy, or merge to main.

Security/auth code changes require a concrete reviewable proposal and task-scoped owner approval. A failing independent review or required local QA gate blocks push. The operator's integration-branch pre-push hook pause is not itself proof of QA success.

## Outcome and local verification

- `43d935c`: formatted exactly the 26 files reported by `fmt-check`, with byte-for-byte comparison against `gofmt` of each parent revision.
- Scoped behavior-preserving changes: reduced capability-registry complexity, reused HTTP header/media-type constants, and wrapped propagated errors while retaining `errors.Is`/`errors.As` chains.
- `GOTOOLCHAIN=go1.25.13 bash scripts/qa-go-prepush.sh`: PASS. Application coverage 83.0%; TDD coverage 84.7%; deadcode findings 0; race stability and pattern gates PASS.
- `GOTOOLCHAIN=go1.25.13 go vet ./...`: PASS.
- Traceability skipped by the pre-push script because `.venv/bin/doorstop` is absent; govulncheck skipped because `go.mod` and `go.sum` are unchanged.
- Verification log: `/tmp/b2a3-t3-qa-ef69918/prepush-final.log`.

## High-Level Pseudocode

```text
collect gate failures and identify files attributable to each gate
for each formatting failure: apply the formatter to that file only
verify formatted content equals formatter output over its original content
commit mechanical formatting separately
for each remaining failing gate: diagnose cause and make the smallest scoped fix
if a fix changes the auth boundary: prepare a concrete patch and request owner approval
run focused tests and full required local QA; obtain independent code review
if every required gate passes: push the branch and verify its remote SHA
publish exact commands, logs, hashes, reviewer artifacts and outcome
```

## Task Card

Task: FENIX-GO-PREPUSH-UNBLOCK
Task file: docs/tasks/task_fenix_go_prepush_unblock.md
Plan file: docs/plans/fenix_go_prepush_unblock_plan.md
Summary: Clear actual Go pre-push gate failures so the T3 formatting commit can be published without weakening gates.
Code affected: Exact fmt-check files, followed by the smallest source or test changes implicated by remaining gates.
Criticality: critical
Criticality basis: Auth-boundary and error-propagation fixes may affect tenant isolation; mechanical formatting and test coverage are also in scope.
Effort/reasoning: High - Multiple ordered QA gates, independent review, and possible security approval.
Recommended model: OpenAI: gpt-5.4 | Anthropic: claude-sonnet-4-6
Estimated tokens: ~30000
Pseudocode: Collect gate failures; format named files; verify and commit; repair scoped failures; run QA and review; push and verify remote SHA.
