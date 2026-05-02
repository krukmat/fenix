# eval — Deterministic Evaluation Domain

This package implements two complementary evaluation mechanisms for governed agent behavior in FenixCRM.

---

## Two Evaluation Mechanisms

### 1. Keyword-Based Eval Suites (`suite.go`, `runner.go`)

Used for continuous eval runs against prompt versions. Each `TestCase` has an input, expected keywords, and an abstention flag. Scoring is keyword hit-rate — no LLM required.

**Use when**: tracking prompt quality over time, running eval suites via the API (`POST /eval/suites/{id}/runs`).

### 2. Golden Scenario Registry (`scenario.go`, `testdata/scenarios/`)

Used for deterministic regression testing of governed agent behavior. Each `GoldenScenario` defines a closed expected behavioral contract: what tools the agent must call, what tools are forbidden, what policy decisions are expected, what audit events must appear, and what the final CRM state should be.

**Use when**: verifying that a specific agent run conforms to the expected governed behavior — no LLM judgment involved.

---

## Golden Scenario Contract

A golden scenario is a YAML file that defines a complete behavioral expectation for one agent execution:

```yaml
id: sc-support-001             # unique, stable identifier
title: "..."
description: "..."
domain: support                # support | sales | general
tags: [uc-c1, happy-path]

input_event:
  type: case.created           # event that triggers the agent
  payload: { ... }

initial_state:                 # CRM/workflow state before the run
  case.id: "..."
  case.status: "New"

expected:
  final_outcome: success       # success | abstained | awaiting_approval | escalated | blocked
  required_evidence: [...]     # evidence IDs that must appear in the run trace
  forbidden_evidence: [...]    # evidence IDs that must NOT appear
  expected_policy_decisions:   # policy outcomes the agent must produce
    - action: "tool:create_task"
      expected_outcome: allow  # allow | deny | require_approval
  expected_tool_calls:         # tools the agent must call
    - tool_name: create_task
      required: true
      params: { ... }          # partial match — only listed keys are checked
  forbidden_tool_calls:        # tools the agent must NOT call
    - tool_name: send_email
      reason: "..."
  approval_behavior:           # optional — only for approval scenarios
    required: true
    expected_outcome: pending  # pending | approved | rejected
  expected_audit_events:       # audit event types that must appear
    - agent.run.started
    - tool.executed
  expected_final_state:        # CRM/workflow state after the run
    case.status: "In Progress"
  should_abstain: false
  abstain_reason: ""

thresholds:
  min_score: 90                # minimum composite score (0-100)
  max_latency_ms: 5000         # maximum run duration
  max_tool_calls: 5            # maximum tool calls allowed
  max_retries: 2               # maximum retry attempts allowed
```

---

## Scenario Categories

The registry covers the seven required behavioral categories:

| File | ID | Category |
|---|---|---|
| `sc_support_happy_path.yaml` | sc-support-001 | 1 — Support case with sufficient evidence |
| `sc_support_weak_evidence_abstention.yaml` | sc-support-003 | 2 — Support case requiring abstention |
| `sc_support_sensitive_mutation_approval.yaml` | sc-support-004 | 3 — Sensitive mutation requiring approval |
| `sc_support_policy_denial.yaml` | sc-support-002 | 4 — Forbidden action denied by policy |
| `sc_support_tool_failure_handoff.yaml` | sc-support-005 | 5 — Tool failure leading to safe handoff |
| `sc_sales_brief_incomplete_context.yaml` | sc-sales-001 | 6 — Sales brief with incomplete context |
| `sc_workflow_activation_blocked.yaml` | sc-workflow-001 | 7 — Workflow activation blocked by conformance |

---

## Loading and Validating Scenarios

```go
sc, err := eval.LoadGoldenScenario("testdata/scenarios/sc_support_happy_path.yaml")
if err != nil {
    // file not found, YAML malformed, or contract invalid
}
```

`LoadGoldenScenario` reads the file, unmarshals, and calls `Validate()`. A scenario is invalid if:
- `id` is empty
- `domain` is not one of `support`, `sales`, `general`
- `input_event.type` is empty
- any `expected_policy_decisions[].expected_outcome` is not `allow`, `deny`, or `require_approval`

---

## Adding New Scenarios

1. Copy `testdata/scenarios/sc_support_happy_path.yaml` as a starting point.
2. Set a unique `id` (format: `sc-<domain>-NNN`).
3. Fill all `expected.*` fields. Do not leave `final_outcome` empty.
4. Add a test in `scenario_test.go` asserting the domain-specific invariant.
5. Run `go test ./internal/domain/eval/...` — all tests must pass before committing.

---

## Relationship to BDD Scenarios

Golden scenarios are **not** a replacement for BDD scenarios in `features/*.feature`. They are complementary:

| | BDD (`features/*.feature`) | Golden Scenarios (`testdata/scenarios/`) |
|---|---|---|
| Purpose | Live execution against real agent | Deterministic trace comparison |
| LLM required | Yes | No |
| Makefile target | `make test-bdd-go` | `make eval-regression` |
| Assertion style | Gherkin steps | Struct comparison |

---

## Regression Suite

Wave F7 adds a deterministic regression runner for golden scenarios:

- `RegressionRunner.Run([]RegressionCase)` evaluates multiple scenario/trace fixtures
- `RegressionReport` aggregates pass/fail counts, score range, verdict distribution, hard-gate counts, and actionable failed dimensions per scenario
- `RegressionReport.ToBaselineSnapshot()` plus `LoadRegressionBaseline` / `SaveRegressionBaseline` support baseline storage and regression comparison
- `CompareToBaseline` marks new failures, score regressions, verdict regressions, and hard-gate regressions deterministically

Local execution:

```bash
make eval-regression
```

CI integration:

- `make ci` now includes `make eval-regression`
- `make test-bdd-go` remains independent and is not replaced by the regression suite
