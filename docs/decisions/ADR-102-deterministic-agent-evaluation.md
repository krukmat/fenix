---
id: ADR-102
title: "Deterministic Agent Evaluation Framework"
date: 2026-05-16
status: accepted
deciders: [matias]
tags: [adr, architecture, eval, testing, determinism, agentic-upgrade]
related_tasks: []
related_frs: []
---

# ADR-102 — Deterministic Agent Evaluation Framework

## Status

`accepted`

## Context

Most AI CRM products cannot prove consistency, reliability, policy compliance, or reproducibility. FenixCRM already has audit, governance, and execution traces. The next step is a formal evaluation framework that makes quality measurable and gatable.

## Decision

FenixCRM shall implement a deterministic evaluation framework with:
- workflow replay over recorded traces
- golden scenario benchmarking
- synthetic dataset support
- execution scoring (15 metrics across 8 dimensions)
- policy compliance scoring (`PolicyComplianceScore`, threshold enforced via `FENIX_EVAL_POLICY_COMPLIANCE_MIN`)
- regression evaluation gated in CI via `make eval`

## Rationale

Deterministic evaluation is the only way to prove that a prompt or policy change does not regress quality. Eval-gated releases prevent silent degradation. The framework positions FenixCRM as an enterprise-grade AI layer where quality is provable, not anecdotal.

## Alternatives considered

| Option | Why rejected |
|--------|-------------|
| Manual QA only | Not scalable; misses regressions on prompt or policy changes |
| External eval service (e.g., Braintrust) | Vendor lock-in; network dependency; cannot run offline |
| LLM-as-judge | Non-deterministic; cannot be used as a hard CI gate |

## Consequences

**Positive:**
- Enterprise trust via measurable, reproducible quality
- CI gate prevents regressions on every push
- Eval framework positions FenixCRM as an AI evaluator platform
- Policy compliance enforced as a hard gate (zero tolerance)

**Negative / tradeoffs:**
- Benchmark overfitting risk (mitigated by golden scenarios based on real CRM workflows)
- Synthetic scenario bias (mitigated by using realistic fixture data)

## Score Architecture

The framework produces **15 metrics** aggregated into **8 scorecard dimensions**. The three top-level scores referenced in this ADR map to scorecard dimensions in `internal/domain/eval/metrics.go`.

`PolicyComplianceScore` threshold is externalized to env var `FENIX_EVAL_POLICY_COMPLIANCE_MIN` (default `1.0`), set explicitly in the Makefile `eval` target. See R.15.

## New Modules

- `eval_runner` — `RegressionRunner` executes scenarios deterministically
- `replay_engine` — replays recorded traces against golden scenarios
- `benchmark_suite` — `BenchmarkRegistry` + `BenchmarkCase` fixtures
- `synthetic_org_simulator` — deterministic fixture generation

## References

- Remediation plan: `docs/plans/fenixcrm_agentic_upgrade_remediation_plan.md`
- `internal/domain/eval/` — implementation root
- R.15: Externalize PolicyCompliance threshold
- CI gate: `make eval` → `TestRegressionFixtureSuite` + `TestRegressionFixturePolicyComplianceThreshold`

## Changelog

- 2026-05-16: Created as `Proposed` in `new reqs/fenixcrm_agentic_upgrade_pack/`
- 2026-05-18: Promoted to `docs/decisions/` with status `Accepted`
