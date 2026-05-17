# ADR-102 — Deterministic Agent Evaluation Framework

Status: Proposed
Date: 2026-05-16

## Context

Most AI CRM products cannot prove:
- consistency
- reliability
- policy compliance
- reproducibility

FenixCRM already has:
- audit
- governance
- execution traces

## Decision

FenixCRM shall implement a deterministic evaluation framework.

## Capabilities

The platform shall support:
- workflow replay
- scenario benchmarking
- synthetic datasets
- execution scoring
- policy compliance scoring
- regression evaluation

## Architectural Consequences

New modules:
- eval_runner
- replay_engine
- benchmark_suite
- synthetic_org_simulator

## Benefits

- enterprise trust
- measurable quality
- reproducible validation
- AI evaluator positioning

## Risks

- benchmark overfitting
- synthetic scenario bias

## Implementation Direction

Initial scope:
- replay existing traces
- deterministic workflow scoring
- benchmark registry
