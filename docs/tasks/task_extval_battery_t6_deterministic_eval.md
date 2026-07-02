---
doc_type: task
id: EXTVAL-BATTERY-T6-001
title: "Run battery T6 — Deterministic eval regression against current checkout"
status: done
phase: external-validation-battery
week: "2026-W27"
tags: [external-validation, battery, eval, regression, governance]
fr_refs: []
uc_refs: []
blocked_by: []
blocks: [EXTVAL-BATTERY-T7-001]
files_affected: []
created: 2026-07-02
completed: 2026-07-02
blocked_reason:
---

# Task EXTVAL-BATTERY-T6-001

**Plan**: [External Validation First Test Battery Plan](../plans/external_validation_first_test_battery_plan.md#t6-deterministic-eval-regression)

## Task Card

Task: EXTVAL-BATTERY-T6-001

Task file: docs/tasks/task_extval_battery_t6_deterministic_eval.md

Plan file: docs/plans/external_validation_first_test_battery_plan.md

Summary: Run `make eval` (deterministic governance/eval fixture suite plus policy-compliance threshold gate) against the current checkout, which includes the uncommitted `EXTVAL-BUG-RESOLVE-THRESHOLD-UNREACHABLE-001` fix (`internal/domain/knowledge/evidence.go`, `internal/domain/agent/agents/support.go`) validated by T4. Confirms the real-LLM findings surfaced in T3/T5 are genuine product gaps, not a symptom of a broken deterministic gate.

Code affected: None expected. Runs an existing test command only; no source files touched.

Effort/reasoning: Low - single documented command (`make eval`), which already runs the regression suite `make eval-regression` is a strict subset of. No ambiguity in scope.

Recommended model: claude-sonnet-4-6

Estimated tokens: ~4000

Task type: operational validation. No dev-task pseudocode required — no code is written.

## Operational Procedure

1. Confirm the working tree still contains the uncommitted T4 fix (`internal/domain/knowledge/evidence.go`, `internal/domain/agent/agents/support.go`) — same checkout provenance requirement as T4's Retest Preconditions.
2. Run `make eval` and capture full output and exit code.
3. `make eval-regression` is a strict subset of `make eval`'s test selection (`TestRegressionFixtureSuite` vs. `TestRegressionFixtureSuite` + `TestRegressionFixturePolicyComplianceThreshold`) — run `make eval` only; running both would be redundant per the plan's step 2 ("any scenario-specific regression command"), since `make eval` already covers it.
4. Record pass/fail per suite and the policy-compliance threshold value used (`FENIX_EVAL_POLICY_COMPLIANCE_MIN`, default `1.0`).

## Acceptance Criteria

1. `make eval` exits 0.
2. `TestRegressionFixtureSuite` passes.
3. `TestRegressionFixturePolicyComplianceThreshold` passes at `FENIX_EVAL_POLICY_COMPLIANCE_MIN=1.0` (or the documented default).
4. Command output and artifact paths (if any) are captured in this task file's closure report.
5. No source file is modified by this task.

## Closure Report

Precondition confirmed: working tree still contained the uncommitted T4 fix
(`internal/domain/knowledge/evidence.go`, `internal/domain/agent/agents/support.go`)
at run time, matching `git status --short` before and after execution.

RRI: `python3 scripts/rri.py --C 0 --T 0 --A 0 --X 0 --D 0 --K 0 --P 0` -> Final RRI 0,
band Low (0-25). Executed directly per HITL_AUTONOMY_POLICY, no approval packet required.

Command run:

```
make eval
```

Output:

```
Running deterministic eval gate (FENIX_EVAL_POLICY_COMPLIANCE_MIN=1.0)...
FENIX_EVAL_POLICY_COMPLIANCE_MIN=1.0 go test -count=1 -run '^(TestRegressionFixtureSuite|TestRegressionFixturePolicyComplianceThreshold)$' ./internal/domain/eval/...
ok  	github.com/matiasleandrokruk/fenix/internal/domain/eval	0.377s
```

Exit code: 0

Per-suite verification (verbose rerun, same env var):

```
--- PASS: TestRegressionFixtureSuite (0.00s)
--- PASS: TestRegressionFixturePolicyComplianceThreshold (0.00s)
PASS
ok  	github.com/matiasleandrokruk/fenix/internal/domain/eval	0.230s
```

Policy-compliance threshold used: `FENIX_EVAL_POLICY_COMPLIANCE_MIN=1.0` (default), passed.

No output/artifact files were produced by this test run (in-process fixture suite,
no external artifact paths). `git status --short` diff was identical before and after
the run — no source file was modified by this task.

Go/no-go (per plan): Policy compliance threshold passes. **GO** — deterministic
governance/eval fixtures remain green with the T4 fix in place, confirming the T3/T5
real-LLM findings are genuine product gaps, not artifacts of a broken deterministic gate.
