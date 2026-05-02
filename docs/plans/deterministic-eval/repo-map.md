---
doc_type: summary
title: FenixCRM Deterministic Eval — Repository Map
status: complete
created: 2026-05-02
---

# FenixCRM Deterministic Eval — Repository Map

Produced during Wave F0. All entries are confirmed facts from direct file inspection unless marked **[assumption]**.

---

## 1. Extension Point — `internal/domain/eval/`

The existing eval domain is the primary extension point for Waves F1–F8.

| File | Contents | Nature |
|---|---|---|
| `suite.go` | `Suite`, `TestCase{Input, ExpectedKeywords, ShouldAbstain}`, `Thresholds`, `SuiteService` CRUD | MVP operational — keyword-based, persisted in SQLite |
| `runner.go` | `Run`, `Scores{Groundedness, Exactitude, Abstention, PolicyAdherence}`, `RunnerService` | MVP stub — scoring echoes input, abstention always 1.0 |
| `suite_test.go` | 9 tests: CRUD, workspace isolation, paginated runs | Real tests using SQLite in-memory + full migrations |

**Wave guidance:**
- New files to add alongside (not replacing): `scenario.go`, `comparator.go`, `hardgate.go`, `textcontract.go`, `metrics.go`, `packet.go`
- `Scores` struct must be extended with additive fields in F4, not replaced
- Test helper `mustOpenDB` in `suite_test.go` is reusable for all new test files
- No `testdata/` directory exists yet — F1 must create `internal/domain/eval/testdata/scenarios/`

---

## 2. Agent Run Trace Sources — `internal/domain/agent/`

All data needed for `ActualRunTrace` (Wave F2) already exists. No schema changes required.

### Primary trace source

| Field | Location | Description |
|---|---|---|
| `Run.ID` | `orchestrator.go:83` | Run identifier |
| `Run.TraceID` | `orchestrator.go:100` | Auto-generated UUID v7, join key for audit events |
| `Run.ToolCalls` | `orchestrator.go:94` | JSON array of `ToolCall{ToolName, Params, Result, Error}` |
| `Run.RetrievedEvidenceIDs` | `orchestrator.go:92` | JSON array of evidence IDs used |
| `Run.ReasoningTrace` | `orchestrator.go:93` | JSON reasoning steps |
| `Run.Output` | `orchestrator.go:95` | Final agent output text |
| `Run.TotalCost` | `orchestrator.go:98` | Cost in euros |
| `Run.LatencyMs` | `orchestrator.go:99` | Execution latency |
| `Run.Status` | `orchestrator.go:89` | `running/success/partial/abstained/failed/escalated` |

### Step-level trace

| File | Key types | Description |
|---|---|---|
| `runtime_steps.go:37` | `RunStep{StepType, Status, Attempt, Input, Output}` | Per-step execution record |
| `runtime_steps.go:18` | Step types: `retrieve_evidence`, `reason`, `tool_call`, `finalize`, `bridge_step` | Execution phase classification |
| `dsl_statement_trace.go:11` | `StepTypeDSLStatement`, `tracedDSLExecutor` | DSL Carta statement-level tracing |

### Runtime dependency injection

| File | Key type | Description |
|---|---|---|
| `runner.go:20` | `RunContext{ToolRegistry, PolicyEngine, ApprovalService, AuditService, GroundsValidator}` | Single struct carrying all runtime dependencies |

### Judge and conformance (relevant for F5, F10, F12)

| File | Key type | Wave |
|---|---|---|
| `judge_result.go:14` | `JudgeResult{Passed, Violations[]Violation, Warnings[]Warning}` | F5 — reuse `Violation` as base type for hard gate evidence |
| `judge.go:10` | `WorkflowJudge.Verify() → JudgeResult` | F5 — existing protocol checks |
| `judge_protocol.go:15` | `RunProtocolJudgeChecks()` | F5 — dispatch/surface validation |
| `conformance.go:24` | `ConformanceResult{Profile: safe/extended/invalid}` | F10 — blocks workflow activation |
| `visual_projection.go:11` | `WorkflowVisualProjection{Nodes, Edges}`, `ProjectWorkflowSemanticGraph()` | F12 — graph already exists, only expose |
| `visual_authoring.go:6` | `VisualAuthoringGraph`, `ValidateVisualAuthoringGraph()` | F12 — bidirectional authoring implemented |

---

## 3. Audit Join — `internal/infra/sqlite/migrations/`

| Fact | Evidence |
|---|---|
| `agent_run.trace_id TEXT` column exists | `018_agents.up.sql:74` |
| `audit_event.trace_id TEXT` column exists | `010_audit_base.up.sql:18` |
| `idx_audit_trace ON audit_event(trace_id)` index exists | `010_audit_base.up.sql:46` |
| Join `agent_run.trace_id = audit_event.trace_id` is performant | Index confirmed |

Wave F2 `ActualRunTrace` DTO can be built as a read-side enrichment using this join. No schema modification required.

---

## 4. Policy Domain — `internal/domain/policy/`

| Type | Values | File |
|---|---|---|
| `PolicyDecision` | `{Allow bool, Trace *PolicyDecisionTrace}` | `evaluator.go:46` |
| `PolicyDecisionTrace` | `{MatchedEffect, RuleTrace []string, Resource, Action}` | `evaluator.go:32` |
| `ApprovalStatus` FSM | `pending → approved / rejected / expired / cancelled` | `approval.go:16` |
| `ApprovalRequest` | `{RequestedBy, ApproverID, Action, ResourceType, ResourceID, Status}` | `approval.go:49` |

**Important:** `denied` is a legacy alias for `rejected` in `ApprovalStatus`. Wave F2 must use `rejected`, not `denied`.

---

## 5. Audit Domain — `internal/domain/audit/`

| Type | Values | File |
|---|---|---|
| `ActorType` | `user`, `agent`, `system` | `types.go:11` |
| `Outcome` | `success`, `denied`, `error` | `types.go:20` |
| `AuditEvent` | struct with `ActorType`, `Outcome`, actor/resource/change fields | `types.go:30` |

Action strings (e.g. `tool.executed`, `policy.evaluated`) are free-form strings in `AuditLogEvent.Action` — not typed constants.

---

## 6. Workflow Domain — `internal/domain/workflow/`

| Type | Values | File |
|---|---|---|
| `Status` FSM | `draft → testing → active → archived` | `repository.go:15` |
| `Service.Activate()` | Validates conformance before promoting to active | `service.go:262` |

**Important distinction for Wave F10:** `invalid` is NOT a workflow FSM state. It is a `ConformanceProfile` value from `agent/conformance.go`. The workflow FSM has 4 states only. Conformance result is a separate concept that gates activation.

---

## 7. Current Test Commands

| Command | Purpose | Baseline result |
|---|---|---|
| `make test` | All Go unit + integration tests | ✅ 30 packages ok, 0 FAIL |
| `make test-bdd-go` | BDD scenarios (godog) | ✅ ok |
| `make complexity` | gocyclo ≤ 7 production code | ✅ PASSED, avg 2.95 |
| `make lint` | gocognit ≤ 10, maintidx ≥ 20 | Available |
| `make ci` | Full pipeline gate | Available |

### Coverage baseline (2026-05-02)

| Package | Coverage |
|---|---|
| `internal/domain/eval` | 81.7% |
| `internal/domain/agent` | 83.8% |
| `internal/domain/policy` | 86.6% |
| `internal/domain/workflow` | 82.6% |
| Global | 77.4% |

**New target for F7:** `make eval-regression` — complements `make test-bdd-go`, does not replace it.

---

## 8. BDD Scenarios Relevant to Deterministic Eval

Located in `features/` and `tests/bdd/go/`.

| Feature file | UC | Relevance |
|---|---|---|
| `uc-c1-support-agent.feature` | UC-C1 | F9 demo — happy path, abstention, approval, handoff |
| `uc-b1-safe-tool-routing.feature` | UC-B1 | F11 — tool denial as product event |
| `uc-a7-approval-workflow.feature` | UC-A7 | F5 — approval required hard gate |
| `uc-g1-governance.feature` | UC-G1 | F12 — audit inspection, usage, quota |
| `uc-a3-workflow-verification-and-activation.feature` | UC-A3 | F10 — workflow activation and conformance |

---

## 9. Gaps — What Does Not Exist Yet

| Component | Target wave |
|---|---|
| Golden scenario YAML fixtures | F1 |
| `GoldenScenario` Go struct + YAML validator | F1 |
| `ActualRunTrace` DTO (read-side enrichment) | F2 |
| `Comparator` (expected vs actual) | F3 |
| Formula-based metrics module | F4 |
| Hard gate validator | F5 |
| Text output contract validator | F6 |
| `make eval-regression` Makefile target | F7 |
| Review Packet generator (Markdown + JSON) | F8 |
| `docs/plans/deterministic-eval/` directory | ✅ Created in F0 |
| `internal/domain/eval/testdata/scenarios/` directory | F1 |

---

## 10. Files That Must NOT Be Modified

The following files are stable foundations. Waves must extend or read them, never replace them.

| File | Reason |
|---|---|
| `internal/domain/eval/suite.go` | Existing keyword-based eval — backward compat |
| `internal/domain/eval/runner.go` | Existing `Scores` struct — extend only |
| `internal/domain/agent/orchestrator.go` | Agent run schema — read only for trace |
| `internal/domain/agent/runner.go` | `RunContext` contract — do not change |
| `internal/domain/policy/evaluator.go` | `PolicyDecision` type — read only |
| `internal/domain/audit/types.go` | `AuditEvent` type — read only |
